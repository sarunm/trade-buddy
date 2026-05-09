package analysis

import "time"

func DeriveSession(t time.Time) string {
	hour := t.UTC().Hour()

	switch {
	case hour <= 6:
		return "asia"
	case hour <= 11:
		return "london"
	case hour <= 15:
		return "london_ny_overlap"
	case hour <= 19:
		return "newyork"
	default:
		return "dead"
	}
}
