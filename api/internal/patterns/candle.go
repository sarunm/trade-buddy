package patterns

import "trade-buddy/api/internal/marketdata"

// IsEngulfingBullish returns true when curr's body fully engulfs prev's body
// and curr closes higher (bullish reversal signal).
func IsEngulfingBullish(prev, curr marketdata.Candle) bool {
	prevBody := prev.Close - prev.Open
	currBody := curr.Close - curr.Open
	if prevBody >= 0 || currBody <= 0 {
		return false // prev must be bearish, curr must be bullish
	}
	return curr.Open <= prev.Close && curr.Close >= prev.Open
}

// IsEngulfingBearish returns true when curr's body fully engulfs prev's body
// and curr closes lower (bearish reversal signal).
func IsEngulfingBearish(prev, curr marketdata.Candle) bool {
	prevBody := prev.Close - prev.Open
	currBody := curr.Close - curr.Open
	if prevBody <= 0 || currBody >= 0 {
		return false // prev must be bullish, curr must be bearish
	}
	return curr.Open >= prev.Close && curr.Close <= prev.Open
}

// IsHammer returns true when the candle has a small body at the top and a long
// lower wick (≥2× body), with little or no upper wick.
func IsHammer(c marketdata.Candle) bool {
	body := abs(c.Close - c.Open)
	lowerWick := min2(c.Open, c.Close) - c.Low
	upperWick := c.High - max2(c.Open, c.Close)
	totalRange := c.High - c.Low
	if totalRange == 0 || body == 0 {
		return false
	}
	return lowerWick >= 2*body && upperWick <= body && body/totalRange <= 0.35
}

// IsShootingStar returns true when the candle has a small body at the bottom
// and a long upper wick (≥2× body), with little or no lower wick.
func IsShootingStar(c marketdata.Candle) bool {
	body := abs(c.Close - c.Open)
	upperWick := c.High - max2(c.Open, c.Close)
	lowerWick := min2(c.Open, c.Close) - c.Low
	totalRange := c.High - c.Low
	if totalRange == 0 || body == 0 {
		return false
	}
	return upperWick >= 2*body && lowerWick <= body && body/totalRange <= 0.35
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func min2(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max2(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
