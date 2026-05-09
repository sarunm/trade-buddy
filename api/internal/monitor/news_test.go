package monitor

import (
	"testing"
	"time"
)

func TestNewsNFPFridayBlocked(t *testing.T) {
	at := time.Date(2026, time.May, 1, 13, 25, 0, 0, time.UTC)

	if !IsNewsBlocked(at) {
		t.Fatal("IsNewsBlocked = false, want true")
	}
	if got := BlockReason(at); got != "NFP" {
		t.Fatalf("BlockReason = %q, want %q", got, "NFP")
	}
}

func TestNewsNFPFridayAfterWindow(t *testing.T) {
	at := time.Date(2026, time.May, 1, 13, 50, 0, 0, time.UTC)

	if IsNewsBlocked(at) {
		t.Fatal("IsNewsBlocked = true, want false")
	}
	if got := BlockReason(at); got != "" {
		t.Fatalf("BlockReason = %q, want empty", got)
	}
}

func TestNewsNormalTuesdayNotBlocked(t *testing.T) {
	at := time.Date(2026, time.May, 5, 14, 0, 0, 0, time.UTC)

	if IsNewsBlocked(at) {
		t.Fatal("IsNewsBlocked = true, want false")
	}
	if got := BlockReason(at); got != "" {
		t.Fatalf("BlockReason = %q, want empty", got)
	}
}

func TestNewsCPIWednesdayBlocked(t *testing.T) {
	at := time.Date(2026, time.May, 13, 13, 28, 0, 0, time.UTC)

	if !IsNewsBlocked(at) {
		t.Fatal("IsNewsBlocked = false, want true")
	}
	if got := BlockReason(at); got != "CPI" {
		t.Fatalf("BlockReason = %q, want %q", got, "CPI")
	}
}

func TestNewsFOMCMonthFilter(t *testing.T) {
	at := time.Date(2026, time.February, 18, 19, 0, 0, 0, time.UTC)

	if IsNewsBlocked(at) {
		t.Fatal("IsNewsBlocked = true, want false")
	}
	if got := BlockReason(at); got != "" {
		t.Fatalf("BlockReason = %q, want empty", got)
	}
}
