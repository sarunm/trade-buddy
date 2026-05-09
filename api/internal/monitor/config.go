package monitor

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"gorm.io/gorm"
)

type Settings struct {
	Interval               time.Duration
	TickTimeout            time.Duration
	MinConfidence          float64
	NotifyEnabled          bool
	Symbols                []string
	ExecutionTimeframe     string
	DedupBars              int
	SessionFilterEnabled   bool
	CalibrationEveryNTicks int
}

func DefaultSettings() Settings {
	return Settings{
		Interval:               60 * time.Second,
		TickTimeout:            30 * time.Second,
		MinConfidence:          0.65,
		NotifyEnabled:          false,
		Symbols:                []string{"XAUUSD"},
		ExecutionTimeframe:     "15m",
		DedupBars:              4,
		SessionFilterEnabled:   false,
		CalibrationEveryNTicks: 100,
	}
}

type ConfigService struct {
	DB       *gorm.DB
	TTL      time.Duration
	mu       sync.RWMutex
	cached   Settings
	loadedAt time.Time
}

var monitorConfigKeys = []string{
	"monitor_interval_seconds",
	"monitor_tick_timeout_seconds",
	"min_confidence_threshold",
	"notify_enabled",
	"symbols_to_monitor",
	"execution_timeframe",
	"dedup_bars",
	"session_filter_enabled",
	"calibration_every_n_ticks",
}

type appConfigRow struct {
	Key   string          `gorm:"column:key"`
	Value json.RawMessage `gorm:"column:value"`
}

func NewConfigService(db *gorm.DB, ttl time.Duration) *ConfigService {
	return &ConfigService{
		DB:  db,
		TTL: ttl,
	}
}

func (s *ConfigService) Load(ctx context.Context) (Settings, error) {
	s.mu.RLock()
	if !s.loadedAt.IsZero() && time.Since(s.loadedAt) < s.TTL {
		cfg := cloneSettings(s.cached)
		s.mu.RUnlock()
		return cfg, nil
	}
	s.mu.RUnlock()

	var rows []appConfigRow
	if err := s.DB.WithContext(ctx).
		Table("app_config").
		Select("key, value").
		Where("key IN ?", monitorConfigKeys).
		Find(&rows).Error; err != nil {
		return DefaultSettings(), nil
	}
	if len(rows) == 0 {
		return DefaultSettings(), nil
	}

	cfg := DefaultSettings()
	for _, row := range rows {
		applyConfigValue(&cfg, row)
	}
	cfg.Symbols = append([]string{}, cfg.Symbols...)

	s.mu.Lock()
	s.cached = cfg
	s.loadedAt = time.Now()
	result := cloneSettings(s.cached)
	s.mu.Unlock()

	return result, nil
}

func applyConfigValue(cfg *Settings, row appConfigRow) {
	switch row.Key {
	case "monitor_interval_seconds":
		if value, ok := jsonInt(row.Value); ok {
			cfg.Interval = time.Duration(value) * time.Second
		}
	case "monitor_tick_timeout_seconds":
		if value, ok := jsonInt(row.Value); ok {
			cfg.TickTimeout = time.Duration(value) * time.Second
		}
	case "min_confidence_threshold":
		var value float64
		if json.Unmarshal(row.Value, &value) == nil {
			cfg.MinConfidence = value
		}
	case "notify_enabled":
		var value bool
		if json.Unmarshal(row.Value, &value) == nil {
			cfg.NotifyEnabled = value
		}
	case "symbols_to_monitor":
		var value []string
		if json.Unmarshal(row.Value, &value) == nil {
			cfg.Symbols = append([]string{}, value...)
		}
	case "execution_timeframe":
		var value string
		if json.Unmarshal(row.Value, &value) == nil {
			cfg.ExecutionTimeframe = value
		}
	case "dedup_bars":
		if value, ok := jsonInt(row.Value); ok {
			cfg.DedupBars = value
		}
	case "session_filter_enabled":
		var value bool
		if json.Unmarshal(row.Value, &value) == nil {
			cfg.SessionFilterEnabled = value
		}
	case "calibration_every_n_ticks":
		if value, ok := jsonInt(row.Value); ok {
			cfg.CalibrationEveryNTicks = value
		}
	}
}

func jsonInt(raw json.RawMessage) (int, bool) {
	var value int
	if json.Unmarshal(raw, &value) != nil {
		return 0, false
	}
	return value, true
}

func cloneSettings(cfg Settings) Settings {
	cfg.Symbols = append([]string{}, cfg.Symbols...)
	return cfg
}
