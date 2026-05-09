package monitor

import (
	"database/sql"
	"math"
	"testing"
)

func TestRunCalibrationMath(t *testing.T) {
	ruleName, params, ok := buildCalibration(calibrationRow{
		Pattern:   "engulfing_bullish",
		Timeframe: "15m",
		Session:   "london_ny_overlap",
		Total:     20,
		Wins:      12,
		AvgR:      sql.NullFloat64{Float64: 1.4, Valid: true},
	})
	if !ok {
		t.Fatal("buildCalibration ok = false, want true")
	}
	if ruleName != "pattern:engulfing_bullish:15m:london_ny_overlap" {
		t.Fatalf("ruleName = %q", ruleName)
	}
	assertApprox(t, params.SmoothedWinrate, 0.5909, 0.0001)
	assertApprox(t, params.BaseDelta, 0.0455, 0.0001)
	assertApprox(t, params.SessionWeight, 0.05, 0.0001)
	assertApprox(t, params.FinalDelta, 0.0955, 0.0001)
	if params.SampleSize != 20 {
		t.Fatalf("SampleSize = %d, want 20", params.SampleSize)
	}
	if params.AvgR == nil || *params.AvgR != 1.4 {
		t.Fatalf("AvgR = %v, want 1.4", params.AvgR)
	}
}

func TestRunCalibrationSkipsLowCount(t *testing.T) {
	_, _, ok := buildCalibration(calibrationRow{
		Pattern:   "pinbar",
		Timeframe: "15m",
		Session:   "asia",
		Total:     9,
		Wins:      7,
	})
	if ok {
		t.Fatal("buildCalibration ok = true, want false for total < 10")
	}
}

func assertApprox(t *testing.T, got, want, tolerance float64) {
	t.Helper()
	if math.Abs(got-want) > tolerance {
		t.Fatalf("got %.6f, want %.6f +/- %.6f", got, want, tolerance)
	}
}
