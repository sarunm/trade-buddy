package patterns

import (
	"math"

	"trade-buddy/api/internal/analysis"
)

// IsDoubleTop returns true when the two most recent swing highs are within
// tolerance of each other (price proximity as a fraction, e.g. 0.002 = 0.2%).
func IsDoubleTop(highs []analysis.SwingPoint, tolerance float64) bool {
	if len(highs) < 2 {
		return false
	}
	a := highs[len(highs)-2].Price
	b := highs[len(highs)-1].Price
	return math.Abs(a-b)/((a+b)/2) <= tolerance
}

// IsDoubleBottom returns true when the two most recent swing lows are within
// tolerance of each other.
func IsDoubleBottom(lows []analysis.SwingPoint, tolerance float64) bool {
	if len(lows) < 2 {
		return false
	}
	a := lows[len(lows)-2].Price
	b := lows[len(lows)-1].Price
	return math.Abs(a-b)/((a+b)/2) <= tolerance
}

// IsTripleTop returns true when the three most recent swing highs are all
// within tolerance of each other.
func IsTripleTop(highs []analysis.SwingPoint, tolerance float64) bool {
	if len(highs) < 3 {
		return false
	}
	n := len(highs)
	a := highs[n-3].Price
	b := highs[n-2].Price
	c := highs[n-1].Price
	avg := (a + b + c) / 3
	return math.Abs(a-avg)/avg <= tolerance &&
		math.Abs(b-avg)/avg <= tolerance &&
		math.Abs(c-avg)/avg <= tolerance
}

// IsTripleBottom returns true when the three most recent swing lows are all
// within tolerance of each other.
func IsTripleBottom(lows []analysis.SwingPoint, tolerance float64) bool {
	if len(lows) < 3 {
		return false
	}
	n := len(lows)
	a := lows[n-3].Price
	b := lows[n-2].Price
	c := lows[n-1].Price
	avg := (a + b + c) / 3
	return math.Abs(a-avg)/avg <= tolerance &&
		math.Abs(b-avg)/avg <= tolerance &&
		math.Abs(c-avg)/avg <= tolerance
}
