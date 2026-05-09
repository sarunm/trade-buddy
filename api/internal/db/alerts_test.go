package db_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"trade-buddy/api/internal/db"
)

func TestAlertsRoundTrip(t *testing.T) {
	conn := openTestDB(t)
	ctx := context.Background()

	id := uuid.New()
	entry := 4640.0
	stop := 4590.0
	tp := 4740.0
	alert := db.Alert{
		ID:         id,
		Symbol:     "XAUUSD",
		Timeframe:  "1h",
		Direction:  "long",
		Entry:      &entry,
		StopLoss:   &stop,
		TakeProfit: &tp,
		Reason:     json.RawMessage(`["test setup"]`),
		Context:    json.RawMessage(`{"source":"test"}`),
	}

	if err := db.CreateAlert(ctx, conn, alert); err != nil {
		t.Fatalf("CreateAlert failed: %v", err)
	}

	listed, err := db.ListAlerts(ctx, conn, "XAUUSD")
	if err != nil {
		t.Fatalf("ListAlerts failed: %v", err)
	}
	if !containsAlert(listed, id) {
		t.Fatalf("created alert %s not found in list: %+v", id, listed)
	}

	got, err := db.GetAlert(ctx, conn, id)
	if err != nil {
		t.Fatalf("GetAlert failed: %v", err)
	}
	if got.Status != "open" || got.Direction != "long" {
		t.Fatalf("unexpected alert: %+v", got)
	}

	resolvedPrice := 4740.0
	barsElapsed := 12
	if err := db.ResolveAlert(ctx, conn, id, db.AlertOutcome{
		Outcome:       "win",
		ResolvedPrice: &resolvedPrice,
		BarsElapsed:   &barsElapsed,
		Details:       json.RawMessage(`{"note":"hit tp"}`),
	}); err != nil {
		t.Fatalf("ResolveAlert failed: %v", err)
	}

	resolved, err := db.GetAlert(ctx, conn, id)
	if err != nil {
		t.Fatalf("GetAlert resolved failed: %v", err)
	}
	if resolved.Status != "resolved" {
		t.Fatalf("resolved status = %q, want resolved", resolved.Status)
	}
	if resolved.ResolvedAt == nil {
		t.Fatal("ResolvedAt is nil")
	}
}

func containsAlert(alerts []db.Alert, id uuid.UUID) bool {
	for _, alert := range alerts {
		if alert.ID == id {
			return true
		}
	}
	return false
}
