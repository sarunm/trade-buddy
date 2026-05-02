package marketdata_test

import (
	"context"
	"testing"
	"time"

	"trade-buddy/api/internal/marketdata"
)

// TestYahooLoad_XAUUSD_1h tests a live fetch against Yahoo Finance.
// Requires internet access. Skipped in short mode.
func TestYahooLoad_XAUUSD_1h(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live network test in short mode")
	}

	src := marketdata.NewYahooSource()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	candles, err := src.Load(ctx, "XAUUSD", "1h", 10)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(candles) == 0 {
		t.Fatal("expected at least 1 candle, got 0")
	}

	c := candles[0]
	if c.Close == 0 {
		t.Errorf("candle[0].Close is 0 — likely a nil/filtered candle leaked through")
	}
	if c.Time.IsZero() {
		t.Errorf("candle[0].Time is zero")
	}
	if c.High < c.Low {
		t.Errorf("candle[0].High (%v) < Low (%v)", c.High, c.Low)
	}
}

// TestYahooLoad_SymbolAlias verifies XAUUSD is mapped to GC=F (not rejected).
func TestYahooLoad_SymbolAlias(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live network test in short mode")
	}

	src := marketdata.NewYahooSource()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	candles, err := src.Load(ctx, "XAUUSD", "1d", 5)
	if err != nil {
		t.Fatalf("XAUUSD symbol alias failed: %v", err)
	}
	if len(candles) == 0 {
		t.Fatal("expected candles for XAUUSD (GC=F alias), got 0")
	}
}

// TestYahooLoad_NilCandlesFiltered verifies no zero-Close candles appear in output.
func TestYahooLoad_NilCandlesFiltered(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live network test in short mode")
	}

	src := marketdata.NewYahooSource()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	candles, err := src.Load(ctx, "XAUUSD", "1h", 50)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	for i, c := range candles {
		if c.Close == 0 && c.Open == 0 && c.High == 0 && c.Low == 0 {
			t.Errorf("candle[%d] is all-zero (nil candle not filtered)", i)
		}
	}
}

// TestYahooLoad_TimeframeMap verifies multiple timeframes are accepted.
func TestYahooLoad_TimeframeMap(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live network test in short mode")
	}

	src := marketdata.NewYahooSource()

	timeframes := []string{"15m", "1h", "1d"}
	for _, tf := range timeframes {
		t.Run(tf, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			candles, err := src.Load(ctx, "XAUUSD", tf, 5)
			if err != nil {
				t.Fatalf("timeframe %q failed: %v", tf, err)
			}
			if len(candles) == 0 {
				t.Fatalf("timeframe %q returned 0 candles", tf)
			}
		})
	}
}
