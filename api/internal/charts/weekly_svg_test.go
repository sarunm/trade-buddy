package charts

import (
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"trade-buddy/api/internal/analysis"
	"trade-buddy/api/internal/forecast"
	"trade-buddy/api/internal/marketdata"
)

func TestRenderWeeklyPlanSVG(t *testing.T) {
	input := testWeeklyInput()

	svg := RenderWeeklyPlanSVG(input, 1280, 620)

	for _, want := range []string{"<svg", "Weekly Plan", "1W ล่าสุด", "marker-end", "S1", "R1"} {
		if !strings.Contains(svg, want) {
			t.Fatalf("svg missing %q: %s", want, svg)
		}
	}
}

func TestLayoutPriceLabelsSeparatesCloseLevels(t *testing.T) {
	labels := []priceLabel{
		{Label: "R1", Price: 100.00, Kind: "resistance1", LineY: 200},
		{Label: "S1", Price: 99.95, Kind: "support1", LineY: 203},
		{Label: "latest", Price: 100.02, Kind: "latest", LineY: 201},
	}

	got := layoutPriceLabels(labels, 58, 550, 31)

	for i := 1; i < len(got); i++ {
		if gap := math.Abs(got[i].LabelY - got[i-1].LabelY); gap < 31 {
			t.Fatalf("label gap = %.2f, want at least 31: %+v", gap, got)
		}
	}
}

func TestCachedWeeklyPlanSVG(t *testing.T) {
	dir := t.TempDir()
	input := testWeeklyInput()

	first, err := CachedWeeklyPlanSVG(dir, input, false)
	if err != nil {
		t.Fatalf("CachedWeeklyPlanSVG first: %v", err)
	}
	if first.Cached {
		t.Fatal("first render should not be cached")
	}
	if _, err := os.Stat(first.Path); err != nil {
		t.Fatalf("expected svg file: %v", err)
	}

	second, err := CachedWeeklyPlanSVG(dir, input, false)
	if err != nil {
		t.Fatalf("CachedWeeklyPlanSVG second: %v", err)
	}
	if !second.Cached {
		t.Fatal("second render should be cached")
	}
	if first.Filename != second.Filename {
		t.Fatalf("filename changed: %s != %s", first.Filename, second.Filename)
	}
}

func testWeeklyInput() WeeklyPlanSVGInput {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := make([]marketdata.Candle, 12)
	for i := range candles {
		base := 100.0 + float64(i)
		candles[i] = marketdata.Candle{
			Time:  start.AddDate(0, 0, i*7),
			Open:  base,
			High:  base + 8,
			Low:   base - 8,
			Close: base + 2,
		}
	}

	return WeeklyPlanSVGInput{
		Symbol:    "XAUUSD",
		Source:    "yahoo",
		Timeframe: "1w",
		Candles:   candles,
		Levels: []analysis.Level{
			{Label: "S1", Price: 95, Kind: "support1"},
			{Label: "R1", Price: 120, Kind: "resistance1"},
		},
		Paths: []forecast.ForecastPath{
			{Priority: "primary", Points: []float64{120, 130}, From: 112, Via: 120, To: 130},
		},
		Bias: forecast.BiasResult{Direction: analysis.DirectionLong},
	}
}
