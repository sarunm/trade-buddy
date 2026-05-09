package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"trade-buddy/api/internal/analysis"
)

// SaveTopDownContext persists a TopDownContext snapshot to the DB.
func SaveTopDownContext(ctx context.Context, db *gorm.DB, c analysis.TopDownContext) error {
	swingData, err := json.Marshal(map[string]any{
		"monthly_highs": c.Monthly.SwingHighs,
		"monthly_lows":  c.Monthly.SwingLows,
		"weekly_highs":  c.Weekly.SwingHighs,
		"weekly_lows":   c.Weekly.SwingLows,
		"daily_highs":   c.Daily.SwingHighs,
		"daily_lows":    c.Daily.SwingLows,
		"h4_highs":      c.H4.SwingHighs,
		"h4_lows":       c.H4.SwingLows,
		"h1_highs":      c.H1.SwingHighs,
		"h1_lows":       c.H1.SwingLows,
		"m15_highs":     c.M15.SwingHighs,
		"m15_lows":      c.M15.SwingLows,
	})
	if err != nil {
		return fmt.Errorf("marshal swing data: %w", err)
	}

	levelsData, err := json.Marshal(map[string]any{
		"monthly_supports":     c.Monthly.Supports,
		"monthly_resistances":  c.Monthly.Resistances,
		"weekly_supports":      c.Weekly.Supports,
		"weekly_resistances":   c.Weekly.Resistances,
		"daily_supports":       c.Daily.Supports,
		"daily_resistances":    c.Daily.Resistances,
		"h4_supports":          c.H4.Supports,
		"h4_resistances":       c.H4.Resistances,
		"h1_supports":          c.H1.Supports,
		"h1_resistances":       c.H1.Resistances,
		"m15_supports":         c.M15.Supports,
		"m15_resistances":      c.M15.Resistances,
	})
	if err != nil {
		return fmt.Errorf("marshal levels data: %w", err)
	}

	result := db.WithContext(ctx).Exec(
		`INSERT INTO top_down_contexts
		 (symbol, captured_at, detector_version,
		  monthly_trend, weekly_trend, daily_trend,
		  h4_trend, h1_trend, m15_trend,
		  swing_data, levels_data)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.Symbol, c.CapturedAt, c.DetectorVersion,
		string(c.Monthly.Trend), string(c.Weekly.Trend), string(c.Daily.Trend),
		string(c.H4.Trend), string(c.H1.Trend), string(c.M15.Trend),
		string(swingData), string(levelsData),
	)
	return result.Error
}

// LatestTopDownContext returns the most recent snapshot for a symbol.
func LatestTopDownContext(ctx context.Context, db *gorm.DB, symbol string) (*TopDownRow, error) {
	var row TopDownRow
	result := db.WithContext(ctx).Raw(
		`SELECT id, symbol, captured_at, detector_version,
		        monthly_trend, weekly_trend, daily_trend,
		        h4_trend, h1_trend, m15_trend,
		        swing_data, levels_data
		 FROM top_down_contexts
		 WHERE symbol = ?
		 ORDER BY captured_at DESC
		 LIMIT 1`,
		symbol,
	).Scan(&row)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &row, nil
}

// TopDownRow is the DB representation of a top_down_contexts row.
type TopDownRow struct {
	ID              string    `gorm:"column:id"`
	Symbol          string    `gorm:"column:symbol"`
	CapturedAt      time.Time `gorm:"column:captured_at"`
	DetectorVersion string    `gorm:"column:detector_version"`
	MonthlyTrend    string    `gorm:"column:monthly_trend"`
	WeeklyTrend     string    `gorm:"column:weekly_trend"`
	DailyTrend      string    `gorm:"column:daily_trend"`
	H4Trend         string    `gorm:"column:h4_trend"`
	H1Trend         string    `gorm:"column:h1_trend"`
	M15Trend        string    `gorm:"column:m15_trend"`
	SwingData       string    `gorm:"column:swing_data"`
	LevelsData      string    `gorm:"column:levels_data"`
}
