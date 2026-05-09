package monitor

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestConfigLoadWithFixtureRowsReturnsSettings(t *testing.T) {
	store := newConfigTestStore(map[string]string{
		"monitor_interval_seconds":     "15",
		"monitor_tick_timeout_seconds": "7",
		"min_confidence_threshold":     "0.82",
		"notify_enabled":               "true",
		"symbols_to_monitor":           `["XAUUSD","EURUSD"]`,
		"execution_timeframe":          `"5m"`,
		"dedup_bars":                   "9",
		"session_filter_enabled":       "true",
		"calibration_every_n_ticks":    "25",
	})
	svc := NewConfigService(openConfigTestDB(t, store), time.Minute)

	got, err := svc.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.Interval != 15*time.Second {
		t.Fatalf("Interval = %v, want %v", got.Interval, 15*time.Second)
	}
	if got.TickTimeout != 7*time.Second {
		t.Fatalf("TickTimeout = %v, want %v", got.TickTimeout, 7*time.Second)
	}
	if got.MinConfidence != 0.82 {
		t.Fatalf("MinConfidence = %v, want 0.82", got.MinConfidence)
	}
	if !got.NotifyEnabled {
		t.Fatal("NotifyEnabled = false, want true")
	}
	if strings.Join(got.Symbols, ",") != "XAUUSD,EURUSD" {
		t.Fatalf("Symbols = %v, want [XAUUSD EURUSD]", got.Symbols)
	}
	if got.ExecutionTimeframe != "5m" {
		t.Fatalf("ExecutionTimeframe = %q, want %q", got.ExecutionTimeframe, "5m")
	}
	if got.DedupBars != 9 {
		t.Fatalf("DedupBars = %d, want 9", got.DedupBars)
	}
	if !got.SessionFilterEnabled {
		t.Fatal("SessionFilterEnabled = false, want true")
	}
	if got.CalibrationEveryNTicks != 25 {
		t.Fatalf("CalibrationEveryNTicks = %d, want 25", got.CalibrationEveryNTicks)
	}
}

func TestConfigLoadTwiceWithinTTLQueriesOnce(t *testing.T) {
	store := newConfigTestStore(map[string]string{
		"monitor_interval_seconds": "45",
	})
	svc := NewConfigService(openConfigTestDB(t, store), time.Minute)

	if _, err := svc.Load(context.Background()); err != nil {
		t.Fatalf("first Load() error = %v", err)
	}
	if _, err := svc.Load(context.Background()); err != nil {
		t.Fatalf("second Load() error = %v", err)
	}

	if got := store.queryCount(); got != 1 {
		t.Fatalf("app_config query count = %d, want 1", got)
	}
}

func TestConfigLoadClonesReturnedSymbols(t *testing.T) {
	store := newConfigTestStore(map[string]string{
		"symbols_to_monitor": `["XAUUSD","GBPUSD"]`,
	})
	svc := NewConfigService(openConfigTestDB(t, store), time.Minute)

	first, err := svc.Load(context.Background())
	if err != nil {
		t.Fatalf("first Load() error = %v", err)
	}
	first.Symbols[0] = "MUTATED"

	second, err := svc.Load(context.Background())
	if err != nil {
		t.Fatalf("second Load() error = %v", err)
	}
	if second.Symbols[0] != "XAUUSD" {
		t.Fatalf("cached Symbols mutated: got %v", second.Symbols)
	}
}

func TestConfigLoadEmptyDBReturnsDefaultSettings(t *testing.T) {
	svc := NewConfigService(openConfigTestDB(t, newConfigTestStore(nil)), time.Minute)

	got, err := svc.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	assertSettingsEqual(t, got, DefaultSettings())
}

func assertSettingsEqual(t *testing.T, got Settings, want Settings) {
	t.Helper()
	if got.Interval != want.Interval ||
		got.TickTimeout != want.TickTimeout ||
		got.MinConfidence != want.MinConfidence ||
		got.NotifyEnabled != want.NotifyEnabled ||
		got.ExecutionTimeframe != want.ExecutionTimeframe ||
		got.DedupBars != want.DedupBars ||
		got.SessionFilterEnabled != want.SessionFilterEnabled ||
		got.CalibrationEveryNTicks != want.CalibrationEveryNTicks ||
		strings.Join(got.Symbols, ",") != strings.Join(want.Symbols, ",") {
		t.Fatalf("Settings = %+v, want %+v", got, want)
	}
}

var configTestDriverRegistered atomic.Bool

func openConfigTestDB(t *testing.T, store *configTestStore) *gorm.DB {
	t.Helper()
	if configTestDriverRegistered.CompareAndSwap(false, true) {
		sql.Register("monitor_config_test", configTestDriver{})
	}

	name := fmt.Sprintf("monitor-config-%d", time.Now().UnixNano())
	configTestStores.Store(name, store)
	t.Cleanup(func() {
		configTestStores.Delete(name)
	})

	sqlDB, err := sql.Open("monitor_config_test", name)
	if err != nil {
		t.Fatalf("open fake sql db: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{
		DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatalf("open gorm db: %v", err)
	}
	return db
}

type configTestStore struct {
	rows    map[string]string
	queries atomic.Int64
}

func newConfigTestStore(rows map[string]string) *configTestStore {
	if rows == nil {
		rows = map[string]string{}
	}
	return &configTestStore{rows: rows}
}

func (s *configTestStore) queryCount() int64 {
	return s.queries.Load()
}

var configTestStores sync.Map

type configTestDriver struct{}

func (configTestDriver) Open(name string) (driver.Conn, error) {
	value, ok := configTestStores.Load(name)
	if !ok {
		return nil, fmt.Errorf("missing config test store %q", name)
	}
	return &configTestConn{store: value.(*configTestStore)}, nil
}

type configTestConn struct {
	store *configTestStore
}

func (c *configTestConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("Prepare is not implemented")
}

func (c *configTestConn) Close() error {
	return nil
}

func (c *configTestConn) Begin() (driver.Tx, error) {
	return nil, errors.New("Begin is not implemented")
}

func (c *configTestConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	if strings.Contains(strings.ToLower(query), "app_config") {
		c.store.queries.Add(1)
		rows := make([][]driver.Value, 0, len(c.store.rows))
		for key, value := range c.store.rows {
			rows = append(rows, []driver.Value{key, []byte(value)})
		}
		return &configTestRows{
			columns: []string{"key", "value"},
			rows:    rows,
		}, nil
	}

	return &configTestRows{
		columns: []string{"version"},
		rows:    [][]driver.Value{{"PostgreSQL 16.0"}},
	}, nil
}

var _ driver.QueryerContext = (*configTestConn)(nil)

type configTestRows struct {
	columns []string
	rows    [][]driver.Value
	index   int
}

func (r *configTestRows) Columns() []string {
	return r.columns
}

func (r *configTestRows) Close() error {
	return nil
}

func (r *configTestRows) Next(dest []driver.Value) error {
	if r.index >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.index])
	r.index++
	return nil
}
