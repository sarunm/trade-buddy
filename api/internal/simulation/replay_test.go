package simulation

import (
	"math"
	"testing"
	"time"

	"trade-buddy/api/internal/marketdata"
)

func bar(o, h, l, c float64) marketdata.Candle {
	return marketdata.Candle{Time: time.Now(), Open: o, High: h, Low: l, Close: c}
}

func TestReplay_LongTP(t *testing.T) {
	order := SimOrder{
		Direction: "long",
		OrderType: "market",
		Entry:     2300,
		SL:        2290,
		TP:        2320,
	}
	candles := []marketdata.Candle{
		bar(2300, 2305, 2298, 2303),
		bar(2303, 2310, 2301, 2308),
		bar(2308, 2325, 2307, 2322), // TP hit
	}
	out := ReplayOrder(order, candles)
	if out.Outcome != "tp" {
		t.Fatalf("expected tp, got %s", out.Outcome)
	}
	// R-multiple: MFE / risk = (2325-2300) / (2300-2290) = 25/10 = 2.5
	if math.Abs(out.RMultiple-2.5) > 0.01 {
		t.Fatalf("expected R=2.5, got %.2f", out.RMultiple)
	}
}

func TestReplay_LongSL(t *testing.T) {
	order := SimOrder{
		Direction: "long",
		OrderType: "market",
		Entry:     2300,
		SL:        2290,
		TP:        2320,
	}
	candles := []marketdata.Candle{
		bar(2300, 2302, 2295, 2298),
		bar(2298, 2299, 2288, 2289), // SL hit
	}
	out := ReplayOrder(order, candles)
	if out.Outcome != "sl" {
		t.Fatalf("expected sl, got %s", out.Outcome)
	}
	if out.RMultiple != -1.0 {
		t.Fatalf("expected R=-1.0, got %.2f", out.RMultiple)
	}
}

func TestReplay_Ambiguous(t *testing.T) {
	order := SimOrder{
		Direction: "long",
		OrderType: "market",
		Entry:     2300,
		SL:        2290,
		TP:        2320,
	}
	// One candle that hits both TP high and SL low
	candles := []marketdata.Candle{
		bar(2300, 2325, 2285, 2300),
	}
	out := ReplayOrder(order, candles)
	if out.Outcome != "ambiguous" {
		t.Fatalf("expected ambiguous, got %s", out.Outcome)
	}
}

func TestReplay_Expired(t *testing.T) {
	order := SimOrder{
		Direction:  "long",
		OrderType:  "market",
		Entry:      2300,
		SL:         2290,
		TP:         2500,
		ExpiryBars: 3,
	}
	candles := []marketdata.Candle{
		bar(2300, 2305, 2298, 2303),
		bar(2303, 2308, 2301, 2306),
		bar(2306, 2310, 2304, 2308),
		bar(2308, 2312, 2306, 2310), // expiry bar
	}
	out := ReplayOrder(order, candles)
	if out.Outcome != "expired" {
		t.Fatalf("expected expired, got %s", out.Outcome)
	}
}
