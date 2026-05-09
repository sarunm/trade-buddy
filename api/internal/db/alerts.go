package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Alert struct {
	ID         uuid.UUID
	Symbol     string
	Timeframe  string
	Direction  string
	Entry      *float64
	StopLoss   *float64
	TakeProfit *float64
	RiskReward *float64
	Confidence *float64
	Reason     json.RawMessage
	Context    json.RawMessage
	Status     string
	CreatedAt  time.Time
	ResolvedAt *time.Time
}

type AlertOutcome struct {
	ID            int64
	AlertID       uuid.UUID
	Outcome       string
	ResolvedPrice *float64
	BarsElapsed   *int
	MFE           *float64
	MAE           *float64
	Details       json.RawMessage
	CreatedAt     time.Time
}

func CreateAlert(ctx context.Context, db *gorm.DB, alert Alert) error {
	if alert.ID == uuid.Nil {
		alert.ID = uuid.New()
	}
	if len(alert.Reason) == 0 {
		alert.Reason = json.RawMessage("[]")
	}
	if len(alert.Context) == 0 {
		alert.Context = json.RawMessage("{}")
	}
	if alert.Status == "" {
		alert.Status = "open"
	}

	return db.WithContext(ctx).Exec(
		`INSERT INTO alerts (id, symbol, timeframe, direction, entry, stop_loss, take_profit, risk_reward, confidence, reason, context, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?::jsonb, ?::jsonb, ?)`,
		alert.ID, alert.Symbol, alert.Timeframe, alert.Direction, alert.Entry, alert.StopLoss, alert.TakeProfit,
		alert.RiskReward, alert.Confidence, string(alert.Reason), string(alert.Context), alert.Status,
	).Error
}

func ListAlerts(ctx context.Context, db *gorm.DB, symbol string) ([]Alert, error) {
	var rows []alertRow
	query := db.WithContext(ctx).Raw(
		`SELECT id, symbol, timeframe, direction, entry, stop_loss, take_profit, risk_reward, confidence,
		        reason, context, status, created_at, resolved_at
		 FROM alerts
		 WHERE symbol = ?
		 ORDER BY created_at DESC`,
		symbol,
	)
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}
	return alertsFromRows(rows), nil
}

func GetAlert(ctx context.Context, db *gorm.DB, id uuid.UUID) (Alert, error) {
	var row alertRow
	err := db.WithContext(ctx).Raw(
		`SELECT id, symbol, timeframe, direction, entry, stop_loss, take_profit, risk_reward, confidence,
		        reason, context, status, created_at, resolved_at
		 FROM alerts
		 WHERE id = ?`,
		id,
	).Scan(&row).Error
	if err != nil {
		return Alert{}, err
	}
	if row.ID == uuid.Nil {
		return Alert{}, gorm.ErrRecordNotFound
	}
	return alertFromRow(row), nil
}

func ResolveAlert(ctx context.Context, db *gorm.DB, id uuid.UUID, outcome AlertOutcome) error {
	if len(outcome.Details) == 0 {
		outcome.Details = json.RawMessage("{}")
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(
			`UPDATE alerts SET status = 'resolved', resolved_at = now() WHERE id = ?`,
			id,
		).Error; err != nil {
			return err
		}
		return tx.Exec(
			`INSERT INTO alert_outcomes (alert_id, outcome, resolved_price, bars_elapsed, mfe, mae, details)
			 VALUES (?, ?, ?, ?, ?, ?, ?::jsonb)`,
			id, outcome.Outcome, outcome.ResolvedPrice, outcome.BarsElapsed, outcome.MFE, outcome.MAE, string(outcome.Details),
		).Error
	})
}

type alertRow struct {
	ID         uuid.UUID       `gorm:"column:id"`
	Symbol     string          `gorm:"column:symbol"`
	Timeframe  string          `gorm:"column:timeframe"`
	Direction  string          `gorm:"column:direction"`
	Entry      sql.NullFloat64 `gorm:"column:entry"`
	StopLoss   sql.NullFloat64 `gorm:"column:stop_loss"`
	TakeProfit sql.NullFloat64 `gorm:"column:take_profit"`
	RiskReward sql.NullFloat64 `gorm:"column:risk_reward"`
	Confidence sql.NullFloat64 `gorm:"column:confidence"`
	Reason     []byte          `gorm:"column:reason"`
	Context    []byte          `gorm:"column:context"`
	Status     string          `gorm:"column:status"`
	CreatedAt  time.Time       `gorm:"column:created_at"`
	ResolvedAt *time.Time      `gorm:"column:resolved_at"`
}

func alertsFromRows(rows []alertRow) []Alert {
	alerts := make([]Alert, len(rows))
	for i, row := range rows {
		alerts[i] = alertFromRow(row)
	}
	return alerts
}

func alertFromRow(row alertRow) Alert {
	return Alert{
		ID:         row.ID,
		Symbol:     row.Symbol,
		Timeframe:  row.Timeframe,
		Direction:  row.Direction,
		Entry:      nullFloatPtr(row.Entry),
		StopLoss:   nullFloatPtr(row.StopLoss),
		TakeProfit: nullFloatPtr(row.TakeProfit),
		RiskReward: nullFloatPtr(row.RiskReward),
		Confidence: nullFloatPtr(row.Confidence),
		Reason:     json.RawMessage(row.Reason),
		Context:    json.RawMessage(row.Context),
		Status:     row.Status,
		CreatedAt:  row.CreatedAt,
		ResolvedAt: row.ResolvedAt,
	}
}

func nullFloatPtr(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	return &value.Float64
}
