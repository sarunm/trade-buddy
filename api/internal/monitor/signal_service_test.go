package monitor

import (
	"testing"

	"trade-buddy/api/internal/patterns"
)

func TestAdjustConfidenceColdStart(t *testing.T) {
	svc := NewSignalService(nil, 0)
	svc.cache["pattern:engulfing_bullish:15m:london"] = CalibrationEntry{
		Delta:      0.10,
		SampleSize: 3,
	}
	sig := patterns.PatternSignal{Type: "engulfing_bullish", Confidence: 0.60}

	got := svc.AdjustConfidence(sig, "15m", "london")
	if got != sig.Confidence {
		t.Fatalf("expected confidence unchanged, got %.2f", got)
	}
}

func TestAdjustConfidenceBoosts(t *testing.T) {
	svc := NewSignalService(nil, 0)
	svc.cache["pattern:engulfing_bullish:15m:london"] = CalibrationEntry{
		Delta:      0.10,
		SampleSize: 25,
	}
	sig := patterns.PatternSignal{Type: "engulfing_bullish", Confidence: 0.60}

	got := svc.AdjustConfidence(sig, "15m", "london")
	if got != 0.70 {
		t.Fatalf("expected boosted confidence 0.70, got %.2f", got)
	}
}

func TestAdjustConfidenceCeiling(t *testing.T) {
	svc := NewSignalService(nil, 0)
	svc.cache["pattern:engulfing_bullish:15m:london"] = CalibrationEntry{
		Delta:      0.10,
		SampleSize: 25,
	}
	sig := patterns.PatternSignal{Type: "engulfing_bullish", Confidence: 0.88}

	got := svc.AdjustConfidence(sig, "15m", "london")
	if got != 0.95 {
		t.Fatalf("expected ceiling confidence 0.95, got %.2f", got)
	}
}

func TestAdjustConfidenceFloor(t *testing.T) {
	svc := NewSignalService(nil, 0)
	svc.cache["pattern:engulfing_bullish:15m:london"] = CalibrationEntry{
		Delta:      -0.10,
		SampleSize: 25,
	}
	sig := patterns.PatternSignal{Type: "engulfing_bullish", Confidence: 0.10}

	got := svc.AdjustConfidence(sig, "15m", "london")
	if got != 0.15 {
		t.Fatalf("expected floor confidence 0.15, got %.2f", got)
	}
}

func TestAdjustConfidenceKeyMismatch(t *testing.T) {
	svc := NewSignalService(nil, 0)
	svc.cache["pattern:engulfing_bullish:15m:london"] = CalibrationEntry{
		Delta:      0.10,
		SampleSize: 25,
	}
	sig := patterns.PatternSignal{Type: "hammer", Confidence: 0.60}

	got := svc.AdjustConfidence(sig, "15m", "london")
	if got != sig.Confidence {
		t.Fatalf("expected confidence unchanged, got %.2f", got)
	}
}
