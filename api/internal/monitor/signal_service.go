package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
	"trade-buddy/api/internal/patterns"
)

const (
	minAdjustedConfidence = 0.15
	maxAdjustedConfidence = 0.95
	minCalibrationSamples = 5
)

type CalibrationEntry struct {
	Delta           float64
	SampleSize      int
	SmoothedWinRate float64
}

type SignalService struct {
	DB       *gorm.DB
	TTL      time.Duration
	mu       sync.RWMutex
	cache    map[string]CalibrationEntry
	loadedAt time.Time
}

var _ ConfidenceAdjuster = (*SignalService)(nil)

func NewSignalService(db *gorm.DB, ttl time.Duration) *SignalService {
	return &SignalService{
		DB:    db,
		TTL:   ttl,
		cache: make(map[string]CalibrationEntry),
	}
}

func (s *SignalService) LoadCalibrations(ctx context.Context) error {
	if s == nil {
		return errors.New("signal service is nil")
	}

	s.mu.RLock()
	valid := s.cache != nil && !s.loadedAt.IsZero() && s.TTL > 0 && time.Since(s.loadedAt) < s.TTL
	s.mu.RUnlock()
	if valid {
		return nil
	}

	if s.DB == nil {
		return errors.New("signal service database is nil")
	}

	rows, err := s.DB.WithContext(ctx).Raw("SELECT rule_name, params FROM rule_calibrations").Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	next := make(map[string]CalibrationEntry)
	for rows.Next() {
		var ruleName string
		var params []byte
		if err := rows.Scan(&ruleName, &params); err != nil {
			return err
		}

		var raw struct {
			FinalDelta      float64 `json:"final_delta"`
			SampleSize      int     `json:"sample_size"`
			SmoothedWinRate float64 `json:"smoothed_winrate"`
		}
		if len(params) > 0 {
			if err := json.Unmarshal(params, &raw); err != nil {
				return fmt.Errorf("parse calibration params for %s: %w", ruleName, err)
			}
		}

		next[ruleName] = CalibrationEntry{
			Delta:           raw.FinalDelta,
			SampleSize:      raw.SampleSize,
			SmoothedWinRate: raw.SmoothedWinRate,
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	s.cache = next
	s.loadedAt = time.Now()
	s.mu.Unlock()

	return nil
}

func (s *SignalService) AdjustConfidence(sig patterns.PatternSignal, tf string, session string) float64 {
	if s == nil {
		return sig.Confidence
	}
	_ = s.LoadCalibrations(context.Background())

	key := fmt.Sprintf("pattern:%s:%s:%s", sig.Type, tf, session)
	s.mu.RLock()
	entry, ok := s.cache[key]
	s.mu.RUnlock()
	if !ok || entry.SampleSize < minCalibrationSamples {
		return sig.Confidence
	}

	adjusted := sig.Confidence + entry.Delta
	if adjusted < minAdjustedConfidence {
		return minAdjustedConfidence
	}
	if adjusted > maxAdjustedConfidence {
		return maxAdjustedConfidence
	}
	return adjusted
}
