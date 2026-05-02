package db

import (
	"context"
	"time"

	"gorm.io/gorm"

	"trade-buddy/api/internal/marketdata"
)

// UpsertBars inserts candles into market_bars, ignoring duplicates.
func UpsertBars(ctx context.Context, db *gorm.DB, symbol, timeframe, source string, candles []marketdata.Candle) error {
	for _, c := range candles {
		result := db.WithContext(ctx).Exec(
			`INSERT INTO market_bars (symbol, timeframe, source, ts, open, high, low, close, volume)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			 ON CONFLICT (symbol, timeframe, source, ts) DO NOTHING`,
			symbol, timeframe, source, c.Time, c.Open, c.High, c.Low, c.Close, c.Volume,
		)
		if result.Error != nil {
			return result.Error
		}
	}
	return nil
}

// FetchBars retrieves candles for the given symbol/timeframe/source in ascending time order.
func FetchBars(ctx context.Context, db *gorm.DB, symbol, timeframe, source string, limit int) ([]marketdata.Candle, error) {
	type row struct {
		Ts     time.Time `gorm:"column:ts"`
		Open   float64   `gorm:"column:open"`
		High   float64   `gorm:"column:high"`
		Low    float64   `gorm:"column:low"`
		Close  float64   `gorm:"column:close"`
		Volume float64   `gorm:"column:volume"`
	}

	var rows []row
	result := db.WithContext(ctx).Raw(
		`SELECT ts, open, high, low, close, volume
		 FROM market_bars
		 WHERE symbol = ? AND timeframe = ? AND source = ?
		 ORDER BY ts ASC
		 LIMIT ?`,
		symbol, timeframe, source, limit,
	).Scan(&rows)
	if result.Error != nil {
		return nil, result.Error
	}

	candles := make([]marketdata.Candle, len(rows))
	for i, r := range rows {
		candles[i] = marketdata.Candle{
			Time:   r.Ts,
			Open:   r.Open,
			High:   r.High,
			Low:    r.Low,
			Close:  r.Close,
			Volume: r.Volume,
		}
	}
	return candles, nil
}

// LatestBarTime returns the most recent timestamp stored for the given symbol/timeframe/source.
// Returns a zero time.Time if no rows exist.
func LatestBarTime(ctx context.Context, db *gorm.DB, symbol, timeframe, source string) (time.Time, error) {
	var result struct {
		MaxTs *time.Time `gorm:"column:max_ts"`
	}
	res := db.WithContext(ctx).Raw(
		`SELECT MAX(ts) AS max_ts FROM market_bars WHERE symbol = ? AND timeframe = ? AND source = ?`,
		symbol, timeframe, source,
	).Scan(&result)
	if res.Error != nil {
		return time.Time{}, res.Error
	}
	if result.MaxTs == nil {
		return time.Time{}, nil
	}
	return *result.MaxTs, nil
}
