package forecast

import (
	"testing"

	"trade-buddy/api/internal/analysis"
)

func TestWeeklyRoutesLongPriority(t *testing.T) {
	supports := []analysis.Level{{Label: "S1", Price: 95}, {Label: "S2", Price: 90}}
	resistances := []analysis.Level{{Label: "R1", Price: 110}, {Label: "R2", Price: 120}}

	got := WeeklyRoutes(analysis.DirectionLong, 100, supports, resistances, 10)

	if len(got) != 2 {
		t.Fatalf("len(routes) = %d, want 2", len(got))
	}
	if got[0].Direction != analysis.DirectionLong || got[0].Priority != "primary" {
		t.Fatalf("first route = %+v, want primary long", got[0])
	}
	if got[0].Via != 110 || got[0].To != 120 {
		t.Fatalf("first route via/to = %v/%v, want 110/120", got[0].Via, got[0].To)
	}
}

func TestWeeklyRoutesShortPriority(t *testing.T) {
	supports := []analysis.Level{{Label: "S1", Price: 95}, {Label: "S2", Price: 90}}
	resistances := []analysis.Level{{Label: "R1", Price: 110}}

	got := WeeklyRoutes(analysis.DirectionShort, 100, supports, resistances, 10)

	if len(got) != 2 {
		t.Fatalf("len(routes) = %d, want 2", len(got))
	}
	if got[0].Direction != analysis.DirectionShort || got[0].Priority != "primary" {
		t.Fatalf("first route = %+v, want primary short", got[0])
	}
}
