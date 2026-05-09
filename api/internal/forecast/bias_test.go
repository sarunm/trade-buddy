package forecast

import (
	"testing"

	"trade-buddy/api/internal/analysis"
)

func TestForecastBiasMonthlyWeighted(t *testing.T) {
	got := ForecastBias(analysis.DirectionLong, analysis.DirectionShort)

	if got.Direction != analysis.DirectionLong {
		t.Fatalf("Direction = %q, want long", got.Direction)
	}
	if got.Confidence != 0.33 {
		t.Fatalf("Confidence = %v, want 0.33", got.Confidence)
	}
	if len(got.Reasons) != 2 {
		t.Fatalf("Reasons = %v, want monthly and weekly reasons", got.Reasons)
	}
}

func TestForecastBiasNeutral(t *testing.T) {
	got := ForecastBias(analysis.DirectionLong, analysis.DirectionShort)
	if got.Direction != analysis.DirectionLong {
		t.Fatalf("precondition direction = %q", got.Direction)
	}

	got = ForecastBias(analysis.DirectionNeutral, analysis.DirectionNeutral)
	if got.Direction != analysis.DirectionNeutral {
		t.Fatalf("Direction = %q, want neutral", got.Direction)
	}
	if got.Confidence != 0 {
		t.Fatalf("Confidence = %v, want 0", got.Confidence)
	}
}
