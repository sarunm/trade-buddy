package analysis

import (
	"math"
	"testing"
)

func TestRiskCheck(t *testing.T) {
	cfg := DefaultXAUUSDConfig()

	// entry 2300, SL 2280 = 20 points gap * 100 = 2000 points
	res := RiskCheck(2300, 2280, cfg)
	if res.Error != "" {
		t.Fatalf("unexpected error: %s", res.Error)
	}
	if res.Points != 2000 {
		t.Fatalf("expected 2000 points, got %.0f", res.Points)
	}
}

func TestRiskCheck_WarnThreshold(t *testing.T) {
	cfg := DefaultXAUUSDConfig()
	// entry 2300, SL 2240 = 60 * 100 = 6000 points → warn
	res := RiskCheck(2300, 2240, cfg)
	if res.Warning == "" {
		t.Fatal("expected warning for 6000-point SL")
	}
}

func TestRiskCheck_ErrorThreshold(t *testing.T) {
	cfg := DefaultXAUUSDConfig()
	// entry 2300, SL 2195 = 105 * 100 = 10500 points → error
	res := RiskCheck(2300, 2195, cfg)
	if res.Error == "" {
		t.Fatal("expected error for >10000-point SL")
	}
}

func TestLotFromRiskPercent(t *testing.T) {
	cfg := DefaultXAUUSDConfig()
	// equity=10000, risk=1%, entry=2300, sl=2290 → 10 points gap * 100 = 1000 points
	// dollar risk = 100, dollarPerPointPerLot = 100*0.01=1, lot = 100/1000 = 0.1
	lot := LotFromRiskPercent(10000, 1, 2300, 2290, cfg)
	if math.Abs(lot-0.1) > 0.0001 {
		t.Fatalf("expected lot ~0.1, got %.4f", lot)
	}
}
