package marketdata

import (
	"context"
	"time"
)

// Candle represents a single OHLCV bar.
type Candle struct {
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}

// MarketDataSource is the interface all data sources must implement.
type MarketDataSource interface {
	Load(ctx context.Context, symbol string, timeframe string, limit int) ([]Candle, error)
}
