package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"trade-buddy/api/internal/analysis"
	dbstore "trade-buddy/api/internal/db"
	"trade-buddy/api/internal/marketdata"
	"trade-buddy/api/internal/patterns"
)

var errDuplicateSignalEvent = errors.New("duplicate signal event")

type SignalEvent struct {
	ID         uuid.UUID
	Symbol     string
	Timeframe  string
	SignalType string
	Direction  string
	Confidence float64
	DedupKey   string
	Ts         time.Time
}

func BuildAlertFromSignal(
	symbol string,
	tf string,
	candles []marketdata.Candle,
	td analysis.TopDownContext,
	sig patterns.PatternSignal,
	session string,
	adjustedConfidence float64,
) (dbstore.Alert, SignalEvent, error) {
	if sig.CandleRange[1] >= len(candles) || sig.CandleRange[1] < 0 {
		return dbstore.Alert{}, SignalEvent{}, errors.New("signal CandleRange out of bounds")
	}

	candleCloseTs := candles[sig.CandleRange[1]].Time
	direction := directionString(sig.Bias)
	dedupKey := fmt.Sprintf("%s:%s:%s:%s:%d", symbol, tf, sig.Type, direction, candleCloseTs.Unix())
	confidence := adjustedConfidence
	entry := candles[sig.CandleRange[1]].Close
	stopLoss := sig.Invalidation

	alertContext := map[string]any{
		"pattern":          sig.Type,
		"session":          session,
		"timeframe":        tf,
		"detector_version": "v1",
		"confidence":       adjustedConfidence,
		"invalidation":     sig.Invalidation,
	}
	if td.DetectorVersion != "" {
		alertContext["topdown_detector_version"] = td.DetectorVersion
	}
	contextJSON, err := json.Marshal(alertContext)
	if err != nil {
		return dbstore.Alert{}, SignalEvent{}, fmt.Errorf("marshal alert context: %w", err)
	}

	reasonJSON, err := json.Marshal([]string{sig.Type})
	if err != nil {
		return dbstore.Alert{}, SignalEvent{}, fmt.Errorf("marshal alert reason: %w", err)
	}

	alert := dbstore.Alert{
		ID:         uuid.New(),
		Symbol:     symbol,
		Timeframe:  tf,
		Direction:  direction,
		Entry:      &entry,
		StopLoss:   &stopLoss,
		Confidence: &confidence,
		Reason:     reasonJSON,
		Context:    contextJSON,
		Status:     "open",
	}
	event := SignalEvent{
		ID:         uuid.New(),
		Symbol:     symbol,
		Timeframe:  tf,
		SignalType: sig.Type,
		Direction:  direction,
		Confidence: adjustedConfidence,
		DedupKey:   dedupKey,
		Ts:         candleCloseTs,
	}
	return alert, event, nil
}

func CreateSignalAlertTx(
	ctx context.Context,
	db *gorm.DB,
	alert dbstore.Alert,
	event SignalEvent,
) (DispatchResult, error) {
	if db == nil {
		return DispatchResult{}, errors.New("db is nil")
	}
	if alert.ID == uuid.Nil {
		alert.ID = uuid.New()
	}
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		insertEvent := tx.Exec(
			`INSERT INTO signal_events (id, symbol, timeframe, signal_type, name, direction, confidence, dedup_key, ts)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			 ON CONFLICT (dedup_key) DO NOTHING`,
			event.ID, event.Symbol, event.Timeframe, event.SignalType, event.SignalType, event.Direction,
			event.Confidence, event.DedupKey, event.Ts,
		)
		if insertEvent.Error != nil {
			return insertEvent.Error
		}
		if insertEvent.RowsAffected == 0 {
			return errDuplicateSignalEvent
		}

		if err := dbstore.CreateAlert(ctx, tx, alert); err != nil {
			return err
		}
		return nil
	})
	if errors.Is(err, errDuplicateSignalEvent) {
		return DispatchResult{Created: false}, nil
	}
	if err != nil {
		return DispatchResult{}, err
	}

	// LINE send must happen after the transaction commits.
	return DispatchResult{AlertID: alert.ID, EventID: event.ID, Created: true}, nil
}

func directionString(direction analysis.Direction) string {
	switch direction {
	case analysis.DirectionLong:
		return "long"
	case analysis.DirectionShort:
		return "short"
	default:
		return "neutral"
	}
}
