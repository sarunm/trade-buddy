package analysis

import (
	"testing"
	"time"
)

func TestDeriveSession(t *testing.T) {
	tests := []struct {
		name string
		at   time.Time
		want string
	}{
		{
			name: "00:00 UTC is asia",
			at:   time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC),
			want: "asia",
		},
		{
			name: "06:59 UTC is asia",
			at:   time.Date(2026, 5, 9, 6, 59, 0, 0, time.UTC),
			want: "asia",
		},
		{
			name: "07:00 UTC is london",
			at:   time.Date(2026, 5, 9, 7, 0, 0, 0, time.UTC),
			want: "london",
		},
		{
			name: "11:59 UTC is london",
			at:   time.Date(2026, 5, 9, 11, 59, 0, 0, time.UTC),
			want: "london",
		},
		{
			name: "12:00 UTC is london_ny_overlap",
			at:   time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC),
			want: "london_ny_overlap",
		},
		{
			name: "15:59 UTC is london_ny_overlap",
			at:   time.Date(2026, 5, 9, 15, 59, 0, 0, time.UTC),
			want: "london_ny_overlap",
		},
		{
			name: "16:00 UTC is newyork",
			at:   time.Date(2026, 5, 9, 16, 0, 0, 0, time.UTC),
			want: "newyork",
		},
		{
			name: "19:59 UTC is newyork",
			at:   time.Date(2026, 5, 9, 19, 59, 0, 0, time.UTC),
			want: "newyork",
		},
		{
			name: "20:00 UTC is dead",
			at:   time.Date(2026, 5, 9, 20, 0, 0, 0, time.UTC),
			want: "dead",
		},
		{
			name: "23:59 UTC is dead",
			at:   time.Date(2026, 5, 9, 23, 59, 0, 0, time.UTC),
			want: "dead",
		},
		{
			name: "non-UTC timezone derives from UTC hour",
			at:   time.Date(2026, 5, 9, 14, 30, 0, 0, time.FixedZone("UTC+7", 7*60*60)),
			want: "london",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeriveSession(tt.at)
			if got != tt.want {
				t.Fatalf("DeriveSession(%s) = %q, want %q", tt.at.Format(time.RFC3339), got, tt.want)
			}
		})
	}
}
