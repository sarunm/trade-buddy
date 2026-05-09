package monitor

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"trade-buddy/api/internal/analysis"
	"trade-buddy/api/internal/marketdata"
	"trade-buddy/api/internal/patterns"
)

// DispatchResult is returned per signal that was processed.
type DispatchResult struct {
	AlertID       uuid.UUID
	EventID       uuid.UUID
	Created       bool
	NotifyMessage string
}

// ConfidenceAdjuster adjusts raw pattern confidence before threshold filtering.
type ConfidenceAdjuster interface {
	AdjustConfidence(sig patterns.PatternSignal, tf string, session string) float64
}

type Monitor struct {
	DB         *gorm.DB
	Source     marketdata.MarketDataSource
	Config     *ConfigService
	SvcSignal  ConfidenceAdjuster
	Logger     *slog.Logger
	Notifier   Notifier
	lastTickAt atomic.Value // stores time.Time
	tickCount  atomic.Int64
}

func New(db *gorm.DB, source marketdata.MarketDataSource, config *ConfigService, adjuster ConfidenceAdjuster, logger *slog.Logger, notifier Notifier) *Monitor {
	if logger == nil {
		logger = slog.Default()
	}
	if notifier == nil {
		notifier = NoopNotifier{}
	}
	return &Monitor{
		DB:        db,
		Source:    source,
		Config:    config,
		SvcSignal: adjuster,
		Logger:    logger,
		Notifier:  notifier,
	}
}

type MonitorStats struct {
	LastTickAt time.Time
	TickCount  int64
}

func (m *Monitor) Stats() MonitorStats {
	ts, _ := m.lastTickAt.Load().(time.Time)
	return MonitorStats{
		LastTickAt: ts,
		TickCount:  m.tickCount.Load(),
	}
}

func (m *Monitor) Run(ctx context.Context) error {
	cfg := m.loadSettings(ctx)
	tickCtx, cancel := context.WithTimeout(ctx, cfg.TickTimeout)
	_ = m.tick(tickCtx)
	cancel()

	for {
		cfg = m.loadSettings(ctx)
		timer := time.NewTimer(cfg.Interval)
		select {
		case <-timer.C:
			tickCtx, cancel := context.WithTimeout(ctx, cfg.TickTimeout)
			_ = m.tick(tickCtx)
			cancel()
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return ctx.Err()
		}
	}
}

func (m *Monitor) tick(ctx context.Context) error {
	m.lastTickAt.Store(time.Now().UTC())
	m.tickCount.Add(1)

	cfg := m.loadSettings(ctx)
	for _, symbol := range cfg.Symbols {
		results, err := MonitorTick(ctx, m.DB, m.Source, m.SvcSignal, cfg, symbol)
		if err != nil {
			m.logger().Error("monitor symbol tick failed", "symbol", symbol, "error", err)
			continue
		}

		if m.Notifier == nil {
			continue
		}
		for _, result := range results {
			if !result.Created || result.NotifyMessage == "" {
				continue
			}
			go func(msg string) {
				notifyCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
				defer cancel()
				if err := m.Notifier.Notify(notifyCtx, msg); err != nil {
					m.logger().Error("monitor notification failed", "error", err)
				}
			}(result.NotifyMessage)
		}
	}
	if err := RunResolution(ctx, m.DB, m.Source); err != nil {
		m.logger().Error("monitor alert resolution failed", "error", err)
	}

	count := m.tickCount.Load()
	if cfg.CalibrationEveryNTicks > 0 && count%int64(cfg.CalibrationEveryNTicks) == 0 {
		m.logger().Info(fmt.Sprintf("calibration tick %d", count))
	}

	return nil
}

func MonitorTick(
	ctx context.Context,
	db *gorm.DB,
	src marketdata.MarketDataSource,
	adjuster ConfidenceAdjuster,
	settings Settings,
	symbol string,
) ([]DispatchResult, error) {
	tdCtx, err := analysis.BuildTopDownContext(ctx, symbol, src)
	if err != nil {
		return nil, err
	}

	candles, err := src.Load(ctx, symbol, settings.ExecutionTimeframe, 200)
	if err != nil {
		return nil, err
	}

	tfCtx, ok := topDownTFContext(tdCtx, settings.ExecutionTimeframe)
	var highs, lows []analysis.SwingPoint
	if ok {
		highs = tfCtx.SwingHighs
		lows = tfCtx.SwingLows
	}

	signals := patterns.DetectPatterns(candles, highs, lows)
	results := make([]DispatchResult, 0, len(signals))
	if reason := BlockReason(time.Now().UTC()); reason != "" {
		slog.Default().Info("monitor tick blocked by news", "reason", reason)
		return results, nil
	}

	for _, sig := range signals {
		if sig.CandleRange[1] < 0 || sig.CandleRange[1] >= len(candles) {
			continue
		}

		session := analysis.DeriveSession(candles[sig.CandleRange[1]].Time)
		if tdCtx.Daily.Trend != analysis.DirectionNeutral && tdCtx.Daily.Trend == sig.Bias {
			sig.Confidence += 0.05
			slog.Default().Info("confluence boost applied", "pattern", sig.Type, "bias", sig.Bias)
		}

		adjusted := sig.Confidence
		if adjuster != nil {
			adjusted = adjuster.AdjustConfidence(sig, settings.ExecutionTimeframe, session)
		}
		if adjusted < settings.MinConfidence {
			continue
		}

		alert, event, err := BuildAlertFromSignal(symbol, settings.ExecutionTimeframe, candles, tdCtx, sig, session, adjusted)
		if err != nil {
			return results, err
		}
		slog.Default().Info("monitor signal detected", "dedup_key", event.DedupKey, "confidence", adjusted)
		result, err := CreateSignalAlertTx(ctx, db, alert, event)
		if err != nil {
			return results, err
		}
		if result.Created {
			result.NotifyMessage = signalNotifyMessage(symbol, settings.ExecutionTimeframe, sig.Type, adjusted, alert.Entry, alert.StopLoss, alert.TakeProfit, session)
		}
		results = append(results, result)
	}

	return results, nil
}

func signalNotifyMessage(symbol string, tf string, signalType string, adjusted float64, entry *float64, stopLoss *float64, takeProfit *float64, session string) string {
	return fmt.Sprintf(
		"[SIGNAL] %s %s %s | Conf:%.2f | Entry:%s | SL:%s | TP:%s | Session:%s",
		symbol,
		tf,
		signalType,
		adjusted,
		formatFloatPtr(entry),
		formatFloatPtr(stopLoss),
		formatFloatPtr(takeProfit),
		session,
	)
}

func formatFloatPtr(value *float64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatFloat(*value, 'f', -1, 64)
}

func (m *Monitor) loadSettings(ctx context.Context) Settings {
	if m.Config == nil {
		return DefaultSettings()
	}
	cfg, err := m.Config.Load(ctx)
	if err != nil {
		m.logger().Error("monitor config load failed", "error", err)
		return DefaultSettings()
	}
	return cfg
}

func (m *Monitor) logger() *slog.Logger {
	if m == nil || m.Logger == nil {
		return slog.Default()
	}
	return m.Logger
}

func topDownTFContext(tdCtx analysis.TopDownContext, tf string) (analysis.TFContext, bool) {
	switch tf {
	case "1mo":
		return tdCtx.Monthly, true
	case "1wk", "1w":
		return tdCtx.Weekly, true
	case "1d":
		return tdCtx.Daily, true
	case "4h":
		return tdCtx.H4, true
	case "1h":
		return tdCtx.H1, true
	case "15m":
		return tdCtx.M15, true
	default:
		return analysis.TFContext{}, false
	}
}
