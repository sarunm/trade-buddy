package monitor

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"gorm.io/gorm"
)

type calibrationRow struct {
	Pattern   string          `gorm:"column:pattern"`
	Timeframe string          `gorm:"column:timeframe"`
	Session   string          `gorm:"column:session"`
	Total     int64           `gorm:"column:total"`
	Wins      int64           `gorm:"column:wins"`
	AvgR      sql.NullFloat64 `gorm:"column:avg_r"`
}

type calibrationParams struct {
	SmoothedWinrate float64  `json:"smoothed_winrate"`
	BaseDelta       float64  `json:"base_delta"`
	SessionWeight   float64  `json:"session_weight"`
	FinalDelta      float64  `json:"final_delta"`
	SampleSize      int64    `json:"sample_size"`
	AvgR            *float64 `json:"avg_r"`
}

const calibrationSQL = `
	SELECT
		a.context->>'pattern'   AS pattern,
		a.context->>'timeframe' AS timeframe,
		a.context->>'session'   AS session,
		COUNT(*)                                                                        AS total,
		SUM(CASE WHEN ao.outcome = 'win' THEN 1 ELSE 0 END)                           AS wins,
		AVG((ao.details->>'r_multiple')::DOUBLE PRECISION)                             AS avg_r
	FROM alerts a
	JOIN alert_outcomes ao ON ao.alert_id = a.id
	WHERE ao.outcome IN ('win', 'loss')
	  AND a.context->>'pattern' IS NOT NULL
	  AND a.context->>'timeframe' IS NOT NULL
	  AND a.context->>'session' IS NOT NULL
	GROUP BY a.context->>'pattern', a.context->>'timeframe', a.context->>'session'
	HAVING COUNT(*) >= 10
`

const upsertCalibrationSQL = `
	INSERT INTO rule_calibrations (id, rule_name, version, params, updated_at)
	VALUES (gen_random_uuid(), ?, 1, ?, now())
	ON CONFLICT (rule_name) DO UPDATE SET
		version = rule_calibrations.version + 1,
		params = EXCLUDED.params,
		updated_at = now()
`

func RunCalibration(ctx context.Context, db *gorm.DB) error {
	var rows []calibrationRow
	if err := db.WithContext(ctx).Raw(calibrationSQL).Scan(&rows).Error; err != nil {
		return err
	}

	rulesUpserted := 0
	for _, row := range rows {
		ruleName, params, ok := buildCalibration(row)
		if !ok {
			continue
		}

		rawParams, err := json.Marshal(params)
		if err != nil {
			return err
		}
		if err := db.WithContext(ctx).Exec(upsertCalibrationSQL, ruleName, rawParams).Error; err != nil {
			return err
		}
		rulesUpserted++
	}

	slog.Info("calibration complete", "rules_upserted", rulesUpserted)
	return nil
}

func buildCalibration(row calibrationRow) (string, calibrationParams, bool) {
	if row.Total < 10 {
		return "", calibrationParams{}, false
	}

	smoothed := (float64(row.Wins) + 1.0) / (float64(row.Total) + 2.0)
	baseDelta := (smoothed - 0.50) * 0.5
	weight := calibrationSessionWeight(row.Session)
	finalDelta := baseDelta + weight

	return fmt.Sprintf("pattern:%s:%s:%s", row.Pattern, row.Timeframe, row.Session), calibrationParams{
		SmoothedWinrate: smoothed,
		BaseDelta:       baseDelta,
		SessionWeight:   weight,
		FinalDelta:      finalDelta,
		SampleSize:      row.Total,
		AvgR:            calibrationAvgR(row.AvgR),
	}, true
}

func calibrationSessionWeight(session string) float64 {
	switch session {
	case "london_ny_overlap":
		return 0.05
	case "asia":
		return -0.10
	case "newyork":
		return 0.02
	case "dead":
		return -0.05
	default:
		return 0
	}
}

func calibrationAvgR(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	return &value.Float64
}
