package analysis

import (
	"testing"

	"trade-buddy/api/internal/marketdata"
)

func TestWeeklyLevels(t *testing.T) {
	close := 100.0
	candles := []marketdata.Candle{
		{High: 118, Low: 82},
		{High: 116, Low: 84},
		{High: 121, Low: 79},
		{High: 114, Low: 86},
		{High: 112, Low: 88},
		{High: 130, Low: 70},
		{High: 111, Low: 89},
		{High: 113, Low: 87},
		{High: 124, Low: 76},
		{High: 115, Low: 85},
	}

	supports, resistances := WeeklyLevels(candles, close)

	if len(supports) == 0 {
		t.Fatal("expected at least one support")
	}
	if len(resistances) == 0 {
		t.Fatal("expected at least one resistance")
	}
	if supports[0].Price >= close {
		t.Fatalf("S1 = %v, want below close %v", supports[0].Price, close)
	}
	if resistances[0].Price <= close {
		t.Fatalf("R1 = %v, want above close %v", resistances[0].Price, close)
	}
	if supports[0].Label != "S1" || supports[0].Kind != "support1" {
		t.Fatalf("support label/kind = %s/%s", supports[0].Label, supports[0].Kind)
	}
	if resistances[0].Label != "R1" || resistances[0].Kind != "resistance1" {
		t.Fatalf("resistance label/kind = %s/%s", resistances[0].Label, resistances[0].Kind)
	}
}

func TestWeeklyLevelsEmpty(t *testing.T) {
	supports, resistances := WeeklyLevels(nil, 100)
	if supports != nil || resistances != nil {
		t.Fatalf("WeeklyLevels(nil) = %v, %v; want nil, nil", supports, resistances)
	}
}

func TestSelectSpacedLevels(t *testing.T) {
	got := selectSpacedLevels([]float64{99, 98.99, 80, 60}, 10, 3)
	want := []float64{99, 80, 60}

	assertFloatSeries(t, got, want)
}

func TestWeeklyLevelsPreferSwingsBeforeNearestWicks(t *testing.T) {
	close := 100.0
	candles := []marketdata.Candle{
		{High: 110, Low: 90, Close: 96},
		{High: 108, Low: 88, Close: 94},
		{High: 124, Low: 80, Close: 118},
		{High: 118, Low: 86, Close: 104},
		{High: 106, Low: 92, Close: 99},
		{High: 101, Low: 99, Close: 100},
	}

	supports, resistances := WeeklyLevels(candles, close)

	if len(supports) == 0 || supports[0].Price != 80 {
		t.Fatalf("S1 = %+v, want swing low 80 before nearer wick", supports)
	}
	if len(resistances) == 0 || resistances[0].Price != 124 {
		t.Fatalf("R1 = %+v, want swing high 124 before nearer wick", resistances)
	}
}
