package db_test

import (
	"context"
	"os"
	"testing"
	"time"

	"trade-buddy/api/internal/db"
	"trade-buddy/api/internal/marketdata"

	"gorm.io/gorm"
)

// openTestDB opens a real DB connection using DATABASE_URL env var.
// Skips the test if DATABASE_URL is not set.
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set — skipping DB integration test")
	}
	conn, err := db.Connect(url)
	if err != nil {
		t.Fatalf("db.Connect failed: %v", err)
	}
	return conn
}

// TestUpsertBars_RoundTrip upserts candles and fetches them back.
func TestUpsertBars_RoundTrip(t *testing.T) {
	conn := openTestDB(t)
	ctx := context.Background()

	symbol := "XAUUSD"
	timeframe := "1h"
	source := "test-roundtrip"

	now := time.Now().UTC().Truncate(time.Hour)
	candles := []marketdata.Candle{
		{Time: now.Add(-2 * time.Hour), Open: 3300.0, High: 3310.0, Low: 3295.0, Close: 3305.0, Volume: 100},
		{Time: now.Add(-1 * time.Hour), Open: 3305.0, High: 3320.0, Low: 3300.0, Close: 3315.0, Volume: 150},
		{Time: now, Open: 3315.0, High: 3325.0, Low: 3310.0, Close: 3320.0, Volume: 120},
	}

	if err := db.UpsertBars(ctx, conn, symbol, timeframe, source, candles); err != nil {
		t.Fatalf("UpsertBars failed: %v", err)
	}

	fetched, err := db.FetchBars(ctx, conn, symbol, timeframe, source, 10)
	if err != nil {
		t.Fatalf("FetchBars failed: %v", err)
	}
	if len(fetched) < len(candles) {
		t.Fatalf("expected at least %d candles, got %d", len(candles), len(fetched))
	}

	// Verify last candle (FetchBars returns ascending order by ts)
	last := fetched[len(fetched)-1]
	if last.Close != 3320.0 {
		t.Errorf("last candle Close = %v, want 3320.0", last.Close)
	}
}

// TestUpsertBars_Idempotent verifies duplicate upserts don't create duplicate rows.
func TestUpsertBars_Idempotent(t *testing.T) {
	conn := openTestDB(t)
	ctx := context.Background()

	symbol := "XAUUSD"
	timeframe := "1h"
	source := "test-idem"

	ts := time.Now().UTC().Truncate(time.Hour).Add(-5 * time.Hour)
	candles := []marketdata.Candle{
		{Time: ts, Open: 3300.0, High: 3310.0, Low: 3295.0, Close: 3305.0, Volume: 100},
	}

	// Upsert twice — should not fail or duplicate
	if err := db.UpsertBars(ctx, conn, symbol, timeframe, source, candles); err != nil {
		t.Fatalf("first UpsertBars failed: %v", err)
	}
	if err := db.UpsertBars(ctx, conn, symbol, timeframe, source, candles); err != nil {
		t.Fatalf("second UpsertBars (idempotent) failed: %v", err)
	}

	fetched, err := db.FetchBars(ctx, conn, symbol, timeframe, source, 100)
	if err != nil {
		t.Fatalf("FetchBars failed: %v", err)
	}

	count := 0
	for _, c := range fetched {
		if c.Time.Equal(ts) {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 row for ts %v, got %d (idempotency failed)", ts, count)
	}
}

// TestLatestBarTime returns the most recent timestamp for a symbol/tf/source.
func TestLatestBarTime(t *testing.T) {
	conn := openTestDB(t)
	ctx := context.Background()

	symbol := "XAUUSD"
	timeframe := "1h"
	source := "test-latest"

	now := time.Now().UTC().Truncate(time.Hour)
	candles := []marketdata.Candle{
		{Time: now.Add(-2 * time.Hour), Open: 3300.0, High: 3310.0, Low: 3295.0, Close: 3305.0},
		{Time: now.Add(-1 * time.Hour), Open: 3305.0, High: 3320.0, Low: 3300.0, Close: 3315.0},
	}
	if err := db.UpsertBars(ctx, conn, symbol, timeframe, source, candles); err != nil {
		t.Fatalf("UpsertBars failed: %v", err)
	}

	latest, err := db.LatestBarTime(ctx, conn, symbol, timeframe, source)
	if err != nil {
		t.Fatalf("LatestBarTime failed: %v", err)
	}
	expected := now.Add(-1 * time.Hour)
	if !latest.Equal(expected) {
		t.Errorf("LatestBarTime = %v, want %v", latest, expected)
	}
}
