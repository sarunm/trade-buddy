package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"trade-buddy/api/internal/analysis"
	dbstore "trade-buddy/api/internal/db"
	"trade-buddy/api/internal/marketdata"
	"trade-buddy/api/internal/patterns"
)

func TestCreateSignalAlertTxDedup(t *testing.T) {
	conn := openMonitorTestDB(t)
	ctx := context.Background()

	suffix := uuid.NewString()
	symbol := "XAUUSD-" + suffix
	dedupKey := fmt.Sprintf("%s:15m:engulfing_bullish:long:1778328000", symbol)
	confidence := 0.72
	alert := dbstore.Alert{
		ID:         uuid.New(),
		Symbol:     symbol,
		Timeframe:  "15m",
		Direction:  "long",
		Confidence: &confidence,
		Reason:     json.RawMessage(`["engulfing_bullish"]`),
		Context:    json.RawMessage(`{"pattern":"engulfing_bullish"}`),
		Status:     "open",
	}
	event := SignalEvent{
		ID:         uuid.New(),
		Symbol:     symbol,
		Timeframe:  "15m",
		SignalType: "engulfing_bullish",
		Direction:  "long",
		Confidence: confidence,
		DedupKey:   dedupKey,
		Ts:         time.Date(2026, 5, 9, 8, 0, 0, 0, time.UTC),
	}

	first, err := CreateSignalAlertTx(ctx, conn, alert, event)
	if err != nil {
		t.Fatalf("first CreateSignalAlertTx failed: %v", err)
	}
	if !first.Created || first.AlertID != alert.ID || first.EventID != event.ID {
		t.Fatalf("unexpected first result: %+v", first)
	}

	alert.ID = uuid.New()
	event.ID = uuid.New()
	second, err := CreateSignalAlertTx(ctx, conn, alert, event)
	if err != nil {
		t.Fatalf("second CreateSignalAlertTx failed: %v", err)
	}
	if second.Created {
		t.Fatalf("expected duplicate dispatch not to create rows: %+v", second)
	}

	var alertCount int64
	if err := conn.WithContext(ctx).Raw(
		`SELECT count(*) FROM alerts WHERE symbol = ? AND context->>'pattern' = ?`,
		symbol, "engulfing_bullish",
	).Scan(&alertCount).Error; err != nil {
		t.Fatalf("count alerts failed: %v", err)
	}
	if alertCount != 1 {
		t.Fatalf("alerts count = %d, want 1", alertCount)
	}

	var eventCount int64
	if err := conn.WithContext(ctx).Raw(
		`SELECT count(*) FROM signal_events WHERE dedup_key = ?`,
		dedupKey,
	).Scan(&eventCount).Error; err != nil {
		t.Fatalf("count signal_events failed: %v", err)
	}
	if eventCount != 1 {
		t.Fatalf("signal_events count = %d, want 1", eventCount)
	}
}

func TestBuildAlertFromSignalContextKeys(t *testing.T) {
	candles := []marketdata.Candle{
		{Time: time.Date(2026, 5, 9, 8, 0, 0, 0, time.UTC), Open: 100, High: 101, Low: 99, Close: 100.5},
		{Time: time.Date(2026, 5, 9, 8, 15, 0, 0, time.UTC), Open: 100.5, High: 102, Low: 100, Close: 101.5},
	}
	sig := patterns.PatternSignal{
		Type:         "engulfing_bullish",
		Bias:         analysis.DirectionLong,
		Confidence:   0.65,
		Invalidation: 99,
		CandleRange:  [2]int{0, 1},
	}

	alert, event, err := BuildAlertFromSignal("XAUUSD", "15m", candles, analysis.TopDownContext{}, sig, "london", 0.77)
	if err != nil {
		t.Fatalf("BuildAlertFromSignal failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(alert.Context, &got); err != nil {
		t.Fatalf("unmarshal alert context failed: %v", err)
	}
	for _, key := range []string{"pattern", "session", "timeframe", "detector_version"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("context missing key %q: %v", key, got)
		}
	}
	if event.Confidence != 0.77 {
		t.Fatalf("event confidence = %v, want 0.77", event.Confidence)
	}
	if event.DedupKey != "XAUUSD:15m:engulfing_bullish:long:1778314500" {
		t.Fatalf("dedup key = %q", event.DedupKey)
	}
}

func TestBuildAlertFromSignalBoundsCheck(t *testing.T) {
	_, _, err := BuildAlertFromSignal(
		"XAUUSD",
		"15m",
		[]marketdata.Candle{{Time: time.Now().UTC()}},
		analysis.TopDownContext{},
		patterns.PatternSignal{Type: "hammer", Bias: analysis.DirectionLong, CandleRange: [2]int{0, 1}},
		"asia",
		0.7,
	)
	if err == nil {
		t.Fatal("expected bounds error")
	}
	if err.Error() != "signal CandleRange out of bounds" {
		t.Fatalf("error = %q", err.Error())
	}
}

func openMonitorTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL/DATABASE_URL not set; skipping Postgres integration test")
	}

	conn, err := dbstore.Connect(databaseURL)
	if err != nil {
		t.Fatalf("db.Connect failed: %v", err)
	}
	if err := dbstore.Migrate(conn); err != nil {
		t.Fatalf("db.Migrate failed: %v", err)
	}
	return conn
}
