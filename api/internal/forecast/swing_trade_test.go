package forecast

import (
	"testing"

	"trade-buddy/api/internal/analysis"
)

func TestSwingTradeBiasLong(t *testing.T) {
	bias := BiasResult{Direction: analysis.DirectionLong}
	daily := TrendContext{Trend: analysis.DirectionLong, Support: 95, Resistance: 110, ATR: 10}
	execution := TrendContext{Timeframe: "1h", Trend: analysis.DirectionLong, Support: 96, Resistance: 109, ATR: 8}
	resistances := []analysis.Level{{Label: "R1", Price: 110}, {Label: "R2", Price: 120}}

	got := SwingTradeBias(bias, daily, execution, nil, resistances, 10)

	if got.Direction != analysis.DirectionLong {
		t.Fatalf("Direction = %q, want long", got.Direction)
	}
	if got.TradeType != "ถือสวิงได้" {
		t.Fatalf("TradeType = %q, want ถือสวิงได้", got.TradeType)
	}
	if got.EntryLow == nil || *got.EntryLow != 94.8 {
		t.Fatalf("EntryLow = %v, want 94.8", got.EntryLow)
	}
	if got.TP2 == nil || *got.TP2 != 120 {
		t.Fatalf("TP2 = %v, want 120", got.TP2)
	}
}

func TestSwingTradeBiasNeutral(t *testing.T) {
	got := SwingTradeBias(BiasResult{Direction: analysis.DirectionNeutral}, TrendContext{}, TrendContext{}, nil, nil, 0)

	if got.Direction != analysis.DirectionNeutral {
		t.Fatalf("Direction = %q, want neutral", got.Direction)
	}
	if got.EntryLow != nil || got.StopLoss != nil {
		t.Fatalf("neutral trade should not include entry/stop: %+v", got)
	}
}
