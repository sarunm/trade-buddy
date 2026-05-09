package analysis

import (
	"testing"

	"trade-buddy/api/internal/marketdata"
)

func TestDetectTrendLong(t *testing.T) {
	candles := closeSeries([]float64{
		10, 10, 10, 10, 10, 10, 10,
		11, 12, 13, 14, 15, 16, 17,
		18, 19, 20, 21, 22, 23, 24,
		25,
	})

	if got := DetectTrend(candles); got != DirectionLong {
		t.Fatalf("DetectTrend = %q, want %q", got, DirectionLong)
	}
}

func TestDetectTrendShort(t *testing.T) {
	candles := closeSeries([]float64{
		30, 30, 30, 30, 30, 30, 30,
		29, 28, 27, 26, 25, 24, 23,
		22, 21, 20, 19, 18, 17, 16,
		15,
	})

	if got := DetectTrend(candles); got != DirectionShort {
		t.Fatalf("DetectTrend = %q, want %q", got, DirectionShort)
	}
}

func TestDetectTrendNeutralWithInsufficientCandles(t *testing.T) {
	if got := DetectTrend(closeSeries([]float64{1, 2, 3})); got != DirectionNeutral {
		t.Fatalf("DetectTrend = %q, want %q", got, DirectionNeutral)
	}
}

func TestDetectSwings(t *testing.T) {
	candles := []marketdata.Candle{
		{High: 10, Low: 5},
		{High: 12, Low: 4},
		{High: 15, Low: 6},
		{High: 11, Low: 3},
		{High: 9, Low: 5},
		{High: 8, Low: 7},
	}

	highs, lows := DetectSwings(candles, 2)

	if len(highs) != 1 || highs[0].Index != 2 || highs[0].Price != 15 {
		t.Fatalf("highs = %+v, want index 2 price 15", highs)
	}
	if len(lows) != 1 || lows[0].Index != 3 || lows[0].Price != 3 {
		t.Fatalf("lows = %+v, want index 3 price 3", lows)
	}
}

func closeSeries(values []float64) []marketdata.Candle {
	candles := make([]marketdata.Candle, len(values))
	for i, value := range values {
		candles[i] = marketdata.Candle{
			Open:  value,
			High:  value,
			Low:   value,
			Close: value,
		}
	}
	return candles
}
