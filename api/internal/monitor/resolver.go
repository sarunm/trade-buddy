package monitor

import (
	"context"
	"encoding/json"
	"log/slog"
	"sort"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	dbstore "trade-buddy/api/internal/db"
	"trade-buddy/api/internal/marketdata"
	"trade-buddy/api/internal/simulation"
)

const defaultAlertExpiryBars = 20

var (
	listOpenAlerts = queryOpenAlerts
	resolveAlert   = dbstore.ResolveAlert
)

// RunResolution replays open alerts against newer candles and persists resolved outcomes.
func RunResolution(
	ctx context.Context,
	gormDB *gorm.DB,
	src marketdata.MarketDataSource,
) error {
	alerts, err := listOpenAlerts(ctx, gormDB)
	if err != nil {
		return err
	}
	if len(alerts) == 0 {
		return nil
	}

	resolved := 0
	for _, alert := range alerts {
		ok, err := resolveOneAlert(ctx, gormDB, src, alert)
		if err != nil {
			slog.Default().Error("alert resolution failed", "alert_id", alert.ID, "error", err)
			continue
		}
		if ok {
			resolved++
		}
	}

	if resolved > 0 {
		slog.Default().Info("alert resolution completed", "resolved", resolved, "checked", len(alerts))
	}
	return nil
}

func queryOpenAlerts(ctx context.Context, gormDB *gorm.DB) ([]dbstore.Alert, error) {
	var rows []openAlertRow
	if err := gormDB.WithContext(ctx).Raw(
		`SELECT id, symbol, timeframe, direction, entry, stop_loss, take_profit, context, created_at
		 FROM alerts
		 WHERE status = 'open'
		 ORDER BY created_at ASC`,
	).Scan(&rows).Error; err != nil {
		return nil, err
	}

	alerts := make([]dbstore.Alert, 0, len(rows))
	for _, row := range rows {
		alerts = append(alerts, dbstore.Alert{
			ID:         row.ID,
			Symbol:     row.Symbol,
			Timeframe:  row.Timeframe,
			Direction:  row.Direction,
			Entry:      row.Entry,
			StopLoss:   row.StopLoss,
			TakeProfit: row.TakeProfit,
			Context:    row.Context,
			Status:     "open",
			CreatedAt:  row.CreatedAt,
		})
	}
	return alerts, nil
}

type openAlertRow struct {
	ID         uuid.UUID
	Symbol     string
	Timeframe  string
	Direction  string
	Entry      *float64
	StopLoss   *float64
	TakeProfit *float64
	Context    json.RawMessage
	CreatedAt  time.Time
}

func resolveOneAlert(ctx context.Context, gormDB *gorm.DB, src marketdata.MarketDataSource, alert dbstore.Alert) (bool, error) {
	if alert.Entry == nil || alert.StopLoss == nil || alert.TakeProfit == nil {
		return false, nil
	}

	candles, err := src.Load(ctx, alert.Symbol, alert.Timeframe, 200)
	if err != nil {
		return false, err
	}
	candlesAfter := filterCandlesAfter(candles, alert.CreatedAt)

	order := simulation.SimOrder{
		Symbol:     alert.Symbol,
		Timeframe:  alert.Timeframe,
		Direction:  alert.Direction,
		OrderType:  "market",
		Entry:      *alert.Entry,
		SL:         *alert.StopLoss,
		TP:         *alert.TakeProfit,
		ExpiryBars: alertExpiryBars(alert.Context),
	}
	outcome := simulation.ReplayOrder(order, candlesAfter)
	if outcome.Outcome == "open" {
		return false, nil
	}

	mae := outcome.MAE
	mfe := outcome.MFE
	barsElapsed := outcome.BarsToOutcome
	details, err := json.Marshal(map[string]any{
		"replay_outcome": outcome.Outcome,
		"r_multiple":     outcome.RMultiple,
		"triggered_at":   outcome.TriggeredAt,
		"duration_bars":  outcome.DurationBars,
	})
	if err != nil {
		return false, err
	}

	alertOutcome := dbstore.AlertOutcome{
		Outcome:     alertOutcomeName(outcome.Outcome),
		BarsElapsed: &barsElapsed,
		MFE:         &mfe,
		MAE:         &mae,
		Details:     details,
	}
	if err := resolveAlert(ctx, gormDB, alert.ID, alertOutcome); err != nil {
		return false, err
	}
	return true, nil
}

func filterCandlesAfter(candles []marketdata.Candle, after time.Time) []marketdata.Candle {
	filtered := make([]marketdata.Candle, 0, len(candles))
	for _, candle := range candles {
		if candle.Time.After(after) {
			filtered = append(filtered, candle)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Time.Before(filtered[j].Time)
	})
	return filtered
}

func alertExpiryBars(raw json.RawMessage) int {
	if len(raw) == 0 {
		return defaultAlertExpiryBars
	}

	var context map[string]any
	if err := json.Unmarshal(raw, &context); err != nil {
		return defaultAlertExpiryBars
	}
	value, ok := context["expiry_bars"]
	if !ok {
		return defaultAlertExpiryBars
	}
	switch typed := value.(type) {
	case float64:
		if typed > 0 {
			return int(typed)
		}
	case int:
		if typed > 0 {
			return typed
		}
	}
	return defaultAlertExpiryBars
}

func alertOutcomeName(replayOutcome string) string {
	switch replayOutcome {
	case "tp":
		return "win"
	case "sl":
		return "loss"
	default:
		return replayOutcome
	}
}
