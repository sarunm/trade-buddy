package analysis

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"trade-buddy/api/internal/marketdata"
)

const DetectorVersion = "1.0"

// TFContext holds analysis results for a single timeframe.
type TFContext struct {
	Timeframe   string
	Trend       Direction
	SwingHighs  []SwingPoint
	SwingLows   []SwingPoint
	Supports    []Level
	Resistances []Level
}

// TopDownContext holds multi-timeframe analysis for a symbol.
type TopDownContext struct {
	Symbol          string
	CapturedAt      time.Time
	DetectorVersion string
	Monthly         TFContext
	Weekly          TFContext
	Daily           TFContext
	H4              TFContext
	H1              TFContext
	M15             TFContext
}

// BuildTopDownContext fetches candles for 5 timeframes concurrently, derives
// H4 from H1, and computes trend/swings/levels for each timeframe.
func BuildTopDownContext(ctx context.Context, symbol string, src marketdata.MarketDataSource) (TopDownContext, error) {
	type result struct {
		tf      string
		candles []marketdata.Candle
	}

	tfs := []struct {
		tf    string
		limit int
	}{
		{"1mo", 24},
		{"1wk", 52},
		{"1d", 200},
		{"1h", 200},
		{"15m", 200},
	}

	results := make([]result, len(tfs))
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(ctx)

	for i, t := range tfs {
		i, t := i, t
		g.Go(func() error {
			candles, err := src.Load(gctx, symbol, t.tf, t.limit)
			if err != nil {
				return err
			}
			mu.Lock()
			results[i] = result{tf: t.tf, candles: candles}
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return TopDownContext{}, err
	}

	// map tf → candles
	byTF := make(map[string][]marketdata.Candle, len(results))
	for _, r := range results {
		byTF[r.tf] = r.candles
	}

	tdCtx := TopDownContext{
		Symbol:          symbol,
		CapturedAt:      time.Now().UTC(),
		DetectorVersion: DetectorVersion,
		Monthly:         buildTFContext("1mo", byTF["1mo"]),
		Weekly:          buildTFContext("1wk", byTF["1wk"]),
		Daily:           buildTFContext("1d", byTF["1d"]),
		H4:              buildTFContext("4h", derive4H(byTF["1h"])),
		H1:              buildTFContext("1h", byTF["1h"]),
		M15:             buildTFContext("15m", byTF["15m"]),
	}
	return tdCtx, nil
}

func buildTFContext(tf string, candles []marketdata.Candle) TFContext {
	if len(candles) == 0 {
		return TFContext{Timeframe: tf, Trend: DirectionNeutral}
	}
	highs, lows := DetectSwings(candles, 3)
	close := candles[len(candles)-1].Close
	supports, resistances := WeeklyLevels(candles, close)
	return TFContext{
		Timeframe:   tf,
		Trend:       DetectTrend(candles),
		SwingHighs:  highs,
		SwingLows:   lows,
		Supports:    supports,
		Resistances: resistances,
	}
}

// derive4H groups consecutive 1H candles into 4-bar OHLCV buckets.
func derive4H(h1 []marketdata.Candle) []marketdata.Candle {
	if len(h1) == 0 {
		return nil
	}
	out := make([]marketdata.Candle, 0, len(h1)/4)
	for i := 0; i+3 < len(h1); i += 4 {
		group := h1[i : i+4]
		bar := marketdata.Candle{
			Time:  group[0].Time,
			Open:  group[0].Open,
			High:  group[0].High,
			Low:   group[0].Low,
			Close: group[3].Close,
		}
		for _, c := range group {
			if c.High > bar.High {
				bar.High = c.High
			}
			if c.Low < bar.Low {
				bar.Low = c.Low
			}
			bar.Volume += c.Volume
		}
		out = append(out, bar)
	}
	return out
}
