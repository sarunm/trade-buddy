package analysis

import (
	"testing"

	"trade-buddy/api/internal/marketdata"
)

func TestSMA(t *testing.T) {
	candles := []marketdata.Candle{
		{Close: 10},
		{Close: 20},
		{Close: 30},
		{Close: 40},
		{Close: 50},
	}

	got := SMA(candles, 3)
	want := []float64{0, 0, 20, 30, 40}

	assertFloatSeries(t, got, want)
}

func TestSMAInvalidPeriod(t *testing.T) {
	got := SMA([]marketdata.Candle{{Close: 10}, {Close: 20}}, 0)
	assertFloatSeries(t, got, []float64{0, 0})
}

func TestATR(t *testing.T) {
	candles := []marketdata.Candle{
		{High: 10, Low: 8, Close: 9},
		{High: 12, Low: 9, Close: 11},
		{High: 13, Low: 10, Close: 12},
		{High: 15, Low: 11, Close: 14},
	}

	got := ATR(candles, 2)
	want := []float64{0, 0, 3, 3.5}

	assertFloatSeries(t, got, want)
}

func TestATRInsufficientCandles(t *testing.T) {
	got := ATR([]marketdata.Candle{{High: 10, Low: 8, Close: 9}}, 14)
	assertFloatSeries(t, got, []float64{0})
}

func assertFloatSeries(t *testing.T, got, want []float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %v, want %v (series: %v)", i, got[i], want[i], got)
		}
	}
}
