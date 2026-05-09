package monitor

import (
	"context"
	"sync"
	"testing"
	"time"

	"trade-buddy/api/internal/marketdata"
	"trade-buddy/api/internal/patterns"
)

type fakeSource struct {
	mu        sync.Mutex
	candles   map[string][]marketdata.Candle
	sequences map[string][][]marketdata.Candle
	calls     map[string]int
}

func (s *fakeSource) Load(ctx context.Context, symbol string, timeframe string, limit int) ([]marketdata.Candle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.calls == nil {
		s.calls = make(map[string]int)
	}
	if seq := s.sequences[timeframe]; len(seq) > 0 {
		index := s.calls[timeframe]
		s.calls[timeframe]++
		if index >= len(seq) {
			index = len(seq) - 1
		}
		return append([]marketdata.Candle{}, seq[index]...), nil
	}
	return append([]marketdata.Candle{}, s.candles[timeframe]...), nil
}

type multiplierAdjuster struct {
	multiplier float64
}

func (a multiplierAdjuster) AdjustConfidence(sig patterns.PatternSignal, tf string, session string) float64 {
	return sig.Confidence * a.multiplier
}

func TestMonitorTickUsesRawConfidenceWhenAdjusterNil(t *testing.T) {
	db := openMonitorTestDB(t)
	src := newFakeSource(engulfingCandles())
	settings := DefaultSettings()
	settings.ExecutionTimeframe = "15m"
	settings.MinConfidence = 0.65

	results, err := MonitorTick(context.Background(), db, src, nil, settings, "XAUUSD")
	if err != nil {
		t.Fatalf("MonitorTick returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Created {
		t.Fatal("expected dispatch result with Created=true")
	}
}

func TestMonitorTickAdjustsConfidenceBeforeFilter(t *testing.T) {
	src := newFakeSource(engulfingCandles())
	settings := DefaultSettings()
	settings.ExecutionTimeframe = "15m"
	settings.MinConfidence = 0.65

	results, err := MonitorTick(context.Background(), nil, src, multiplierAdjuster{multiplier: 0.5}, settings, "XAUUSD")
	if err != nil {
		t.Fatalf("MonitorTick returned error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected adjusted confidence to filter signal, got %d results", len(results))
	}
}

func TestMonitorTickSkipsOutOfBoundsSignalRange(t *testing.T) {
	src := &fakeSource{
		candles: map[string][]marketdata.Candle{
			"1mo": genericCandles(8),
			"1wk": genericCandles(8),
			"1d":  genericCandles(8),
			"1h":  genericCandles(8),
		},
		sequences: map[string][][]marketdata.Candle{
			"15m": {doubleTopContextCandles(), nonPatternCandles()},
		},
	}
	settings := DefaultSettings()
	settings.ExecutionTimeframe = "15m"
	settings.MinConfidence = 0.65

	results, err := MonitorTick(context.Background(), nil, src, nil, settings, "XAUUSD")
	if err != nil {
		t.Fatalf("MonitorTick returned error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected out-of-bounds signal to be skipped, got %d results", len(results))
	}
}

func newFakeSource(execCandles []marketdata.Candle) *fakeSource {
	return &fakeSource{
		candles: map[string][]marketdata.Candle{
			"1mo": genericCandles(8),
			"1wk": genericCandles(8),
			"1d":  genericCandles(8),
			"1h":  genericCandles(8),
			"15m": execCandles,
		},
	}
}

func engulfingCandles() []marketdata.Candle {
	base := time.Date(2026, 5, 9, 8, 0, 0, 0, time.UTC)
	return []marketdata.Candle{
		{Time: base, Open: 100, High: 101, Low: 97, Close: 98, Volume: 10},
		{Time: base.Add(15 * time.Minute), Open: 97, High: 102, Low: 96, Close: 101, Volume: 12},
	}
}

func nonPatternCandles() []marketdata.Candle {
	base := time.Date(2026, 5, 9, 8, 0, 0, 0, time.UTC)
	return []marketdata.Candle{
		{Time: base, Open: 100, High: 102, Low: 99, Close: 101, Volume: 10},
		{Time: base.Add(15 * time.Minute), Open: 101, High: 103, Low: 100, Close: 102, Volume: 12},
	}
}

func genericCandles(n int) []marketdata.Candle {
	base := time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC)
	candles := make([]marketdata.Candle, n)
	for i := range candles {
		price := 100 + float64(i)
		candles[i] = marketdata.Candle{
			Time:   base.Add(time.Duration(i) * time.Hour),
			Open:   price,
			High:   price + 1,
			Low:    price - 1,
			Close:  price + 0.25,
			Volume: 10,
		}
	}
	return candles
}

func doubleTopContextCandles() []marketdata.Candle {
	candles := genericCandles(13)
	highs := []float64{100, 101, 102, 110, 103, 102, 101, 102, 103, 110.1, 103, 102, 101}
	for i := range candles {
		candles[i].High = highs[i]
		candles[i].Low = 90 + float64(i%3)
		candles[i].Open = 95
		candles[i].Close = 96
	}
	return candles
}
