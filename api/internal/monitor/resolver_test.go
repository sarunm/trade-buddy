package monitor

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	dbstore "trade-buddy/api/internal/db"
	"trade-buddy/api/internal/marketdata"
)

type resolverSource struct {
	candles []marketdata.Candle
	loads   int
}

func (s *resolverSource) Load(ctx context.Context, symbol string, timeframe string, limit int) ([]marketdata.Candle, error) {
	s.loads++
	return append([]marketdata.Candle{}, s.candles...), nil
}

func TestRunResolutionNoOpenAlerts(t *testing.T) {
	restore := stubResolutionStore(
		func(ctx context.Context, db *gorm.DB) ([]dbstore.Alert, error) {
			return nil, nil
		},
		func(ctx context.Context, db *gorm.DB, id uuid.UUID, outcome dbstore.AlertOutcome) error {
			t.Fatal("ResolveAlert should not be called when no alerts are open")
			return nil
		},
	)
	defer restore()

	src := &resolverSource{}
	if err := RunResolution(context.Background(), nil, src); err != nil {
		t.Fatalf("RunResolution returned error: %v", err)
	}
	if src.loads != 0 {
		t.Fatalf("Load calls = %d, want 0", src.loads)
	}
}

func TestRunResolutionResolvesWin(t *testing.T) {
	alertID := uuid.New()
	createdAt := time.Date(2026, 5, 9, 8, 0, 0, 0, time.UTC)
	entry := 100.0
	stop := 95.0
	takeProfit := 110.0

	var gotOutcome dbstore.AlertOutcome
	var gotID uuid.UUID
	restore := stubResolutionStore(
		func(ctx context.Context, db *gorm.DB) ([]dbstore.Alert, error) {
			return []dbstore.Alert{
				{
					ID:         alertID,
					Symbol:     "XAUUSD",
					Timeframe:  "15m",
					Direction:  "long",
					Entry:      &entry,
					StopLoss:   &stop,
					TakeProfit: &takeProfit,
					Context:    json.RawMessage(`{"expiry_bars":20}`),
					Status:     "open",
					CreatedAt:  createdAt,
				},
			}, nil
		},
		func(ctx context.Context, db *gorm.DB, id uuid.UUID, outcome dbstore.AlertOutcome) error {
			gotID = id
			gotOutcome = outcome
			return nil
		},
	)
	defer restore()

	src := &resolverSource{
		candles: []marketdata.Candle{
			{Time: createdAt.Add(-15 * time.Minute), Open: 90, High: 91, Low: 89, Close: 90},
			{Time: createdAt.Add(15 * time.Minute), Open: 100, High: 111, Low: 99, Close: 110},
		},
	}
	if err := RunResolution(context.Background(), nil, src); err != nil {
		t.Fatalf("RunResolution returned error: %v", err)
	}
	if src.loads != 1 {
		t.Fatalf("Load calls = %d, want 1", src.loads)
	}
	if gotID != alertID {
		t.Fatalf("resolved alert id = %s, want %s", gotID, alertID)
	}
	if gotOutcome.Outcome != "win" {
		t.Fatalf("outcome = %q, want win", gotOutcome.Outcome)
	}
	if gotOutcome.MFE == nil || *gotOutcome.MFE != 11 {
		t.Fatalf("MFE = %v, want 11", gotOutcome.MFE)
	}
	if gotOutcome.MAE == nil || *gotOutcome.MAE != 1 {
		t.Fatalf("MAE = %v, want 1", gotOutcome.MAE)
	}
}

func stubResolutionStore(
	list func(context.Context, *gorm.DB) ([]dbstore.Alert, error),
	resolve func(context.Context, *gorm.DB, uuid.UUID, dbstore.AlertOutcome) error,
) func() {
	origList := listOpenAlerts
	origResolve := resolveAlert
	listOpenAlerts = list
	resolveAlert = resolve
	return func() {
		listOpenAlerts = origList
		resolveAlert = origResolve
	}
}
