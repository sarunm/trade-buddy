package patterns

import (
	"testing"
	"time"

	"trade-buddy/api/internal/marketdata"
)

func candle(o, h, l, c float64) marketdata.Candle {
	return marketdata.Candle{Time: time.Now(), Open: o, High: h, Low: l, Close: c}
}

func TestEngulfingBullish(t *testing.T) {
	prev := candle(100, 102, 95, 96) // bearish
	curr := candle(95, 105, 94, 103) // bullish, engulfs prev
	if !IsEngulfingBullish(prev, curr) {
		t.Fatal("expected bullish engulfing")
	}
	// not engulfing: curr smaller than prev
	if IsEngulfingBullish(prev, candle(97, 99, 96, 98)) {
		t.Fatal("should not detect non-engulfing")
	}
}

func TestEngulfingBearish(t *testing.T) {
	prev := candle(95, 103, 94, 102) // bullish
	curr := candle(103, 104, 93, 94) // bearish, engulfs prev
	if !IsEngulfingBearish(prev, curr) {
		t.Fatal("expected bearish engulfing")
	}
}

func TestHammer(t *testing.T) {
	// body at top, long lower wick
	h := candle(100, 101, 88, 99) // open 100, close 99 → body=1, lower wick=11, upper=1
	if !IsHammer(h) {
		t.Fatal("expected hammer")
	}
	// shooting star — should NOT be hammer
	ss := candle(90, 102, 89, 91)
	if IsHammer(ss) {
		t.Fatal("shooting star should not be hammer")
	}
}

func TestShootingStar(t *testing.T) {
	// body at bottom, long upper wick
	ss := candle(90, 103, 89, 91) // body=1, upper wick=12, lower wick=1
	if !IsShootingStar(ss) {
		t.Fatal("expected shooting star")
	}
}
