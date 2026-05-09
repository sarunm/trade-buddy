package db

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"trade-buddy/api/internal/simulation"
)

// SimulationRow maps to the order_simulations table.
type SimulationRow struct {
	ID          string    `gorm:"column:id"`
	Symbol      string    `gorm:"column:symbol"`
	Timeframe   string    `gorm:"column:timeframe"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	Direction   string    `gorm:"column:direction"`
	OrderType   string    `gorm:"column:order_type"`
	Entry       float64   `gorm:"column:entry"`
	SL          float64   `gorm:"column:sl"`
	TP          float64   `gorm:"column:tp"`
	ExpiryBars  *int      `gorm:"column:expiry_bars"`
}

// OutcomeRow maps to the simulation_outcomes table.
type OutcomeRow struct {
	ID            string     `gorm:"column:id"`
	SimulationID  string     `gorm:"column:simulation_id"`
	TriggeredAt   *time.Time `gorm:"column:triggered_at"`
	Outcome       string     `gorm:"column:outcome"`
	MAE           *float64   `gorm:"column:mae"`
	MFE           *float64   `gorm:"column:mfe"`
	RMultiple     *float64   `gorm:"column:r_multiple"`
	DurationBars  *int       `gorm:"column:duration_bars"`
	BarsToOutcome *int       `gorm:"column:bars_to_outcome"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
}

// CreateSimulation inserts a new order_simulations row and returns the generated UUID.
func CreateSimulation(ctx context.Context, db *gorm.DB, order simulation.SimOrder) (string, error) {
	var id string
	var expiryBars *int
	if order.ExpiryBars > 0 {
		eb := order.ExpiryBars
		expiryBars = &eb
	}
	result := db.WithContext(ctx).Raw(
		`INSERT INTO order_simulations (symbol, timeframe, direction, order_type, entry, sl, tp, expiry_bars)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 RETURNING id`,
		order.Symbol, order.Timeframe, order.Direction, order.OrderType,
		order.Entry, order.SL, order.TP, expiryBars,
	).Scan(&id)
	if result.Error != nil {
		return "", fmt.Errorf("create simulation: %w", result.Error)
	}
	return id, nil
}

// SaveOutcome inserts a simulation_outcomes row for the given simulation ID.
func SaveOutcome(ctx context.Context, db *gorm.DB, simID string, out simulation.SimOutcome) error {
	var triggeredAt *time.Time
	if !out.TriggeredAt.IsZero() {
		t := out.TriggeredAt
		triggeredAt = &t
	}
	nullableFloat := func(v float64) *float64 {
		if v == 0 {
			return nil
		}
		return &v
	}
	nullableInt := func(v int) *int {
		if v == 0 {
			return nil
		}
		return &v
	}
	result := db.WithContext(ctx).Exec(
		`INSERT INTO simulation_outcomes
		 (simulation_id, triggered_at, outcome, mae, mfe, r_multiple, duration_bars, bars_to_outcome)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		simID, triggeredAt, out.Outcome,
		nullableFloat(out.MAE), nullableFloat(out.MFE), nullableFloat(out.RMultiple),
		nullableInt(out.DurationBars), nullableInt(out.BarsToOutcome),
	)
	return result.Error
}

// ListSimulations returns all simulations for a symbol, newest first.
func ListSimulations(ctx context.Context, db *gorm.DB, symbol string) ([]SimulationRow, error) {
	var rows []SimulationRow
	result := db.WithContext(ctx).Raw(
		`SELECT id, symbol, timeframe, created_at, direction, order_type, entry, sl, tp, expiry_bars
		 FROM order_simulations WHERE symbol = ? ORDER BY created_at DESC`,
		symbol,
	).Scan(&rows)
	if result.Error != nil {
		return nil, result.Error
	}
	return rows, nil
}

// GetSimulation returns one simulation and its outcome (if any).
func GetSimulation(ctx context.Context, db *gorm.DB, id string) (*SimulationRow, *OutcomeRow, error) {
	var sim SimulationRow
	res := db.WithContext(ctx).Raw(
		`SELECT id, symbol, timeframe, created_at, direction, order_type, entry, sl, tp, expiry_bars
		 FROM order_simulations WHERE id = ?`, id,
	).Scan(&sim)
	if res.Error != nil {
		return nil, nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, nil, nil
	}

	var outcome OutcomeRow
	res2 := db.WithContext(ctx).Raw(
		`SELECT id, simulation_id, triggered_at, outcome, mae, mfe, r_multiple, duration_bars, bars_to_outcome, created_at
		 FROM simulation_outcomes WHERE simulation_id = ? LIMIT 1`, id,
	).Scan(&outcome)
	if res2.Error != nil {
		return &sim, nil, res2.Error
	}
	if res2.RowsAffected == 0 {
		return &sim, nil, nil
	}
	return &sim, &outcome, nil
}
