package patterns

import (
	"trade-buddy/api/internal/analysis"
	"trade-buddy/api/internal/marketdata"
)

const defaultTolerance = 0.002 // 0.2% price proximity for double/triple patterns

// PatternSignal describes a detected pattern with context for trading decisions.
type PatternSignal struct {
	Type         string             // e.g. "engulfing_bullish", "hammer", "double_top"
	Bias         analysis.Direction // long, short, neutral
	Confidence   float64            // 0.0–1.0
	Invalidation float64            // price level that would negate this signal
	CandleRange  [2]int             // [startIndex, endIndex] in the input candles slice
}

// DetectPatterns runs all pattern detectors on the given candles and swing points.
// Returns confirmed signals only (no noise).
func DetectPatterns(candles []marketdata.Candle, highs, lows []analysis.SwingPoint) []PatternSignal {
	var signals []PatternSignal

	// Candle patterns — scan last 50 candles to keep it bounded
	window := candles
	offset := 0
	if len(candles) > 50 {
		offset = len(candles) - 50
		window = candles[offset:]
	}

	for i := 1; i < len(window); i++ {
		prev := window[i-1]
		curr := window[i]
		absIdx := offset + i

		if IsEngulfingBullish(prev, curr) {
			signals = append(signals, PatternSignal{
				Type:         "engulfing_bullish",
				Bias:         analysis.DirectionLong,
				Confidence:   0.65,
				Invalidation: curr.Low,
				CandleRange:  [2]int{absIdx - 1, absIdx},
			})
		}
		if IsEngulfingBearish(prev, curr) {
			signals = append(signals, PatternSignal{
				Type:         "engulfing_bearish",
				Bias:         analysis.DirectionShort,
				Confidence:   0.65,
				Invalidation: curr.High,
				CandleRange:  [2]int{absIdx - 1, absIdx},
			})
		}
		if IsHammer(curr) {
			signals = append(signals, PatternSignal{
				Type:         "hammer",
				Bias:         analysis.DirectionLong,
				Confidence:   0.55,
				Invalidation: curr.Low,
				CandleRange:  [2]int{absIdx, absIdx},
			})
		}
		if IsShootingStar(curr) {
			signals = append(signals, PatternSignal{
				Type:         "shooting_star",
				Bias:         analysis.DirectionShort,
				Confidence:   0.55,
				Invalidation: curr.High,
				CandleRange:  [2]int{absIdx, absIdx},
			})
		}
	}

	// Swing structure patterns
	if len(candles) > 0 {
		lastClose := candles[len(candles)-1].Close

		if IsDoubleTop(highs, defaultTolerance) {
			h := highs[len(highs)-1]
			signals = append(signals, PatternSignal{
				Type:         "double_top",
				Bias:         analysis.DirectionShort,
				Confidence:   0.70,
				Invalidation: h.Price * 1.002,
				CandleRange:  [2]int{highs[len(highs)-2].Index, h.Index},
			})
		}
		if IsDoubleBottom(lows, defaultTolerance) {
			l := lows[len(lows)-1]
			signals = append(signals, PatternSignal{
				Type:         "double_bottom",
				Bias:         analysis.DirectionLong,
				Confidence:   0.70,
				Invalidation: l.Price * 0.998,
				CandleRange:  [2]int{lows[len(lows)-2].Index, l.Index},
			})
		}
		if IsTripleTop(highs, defaultTolerance) {
			h := highs[len(highs)-1]
			signals = append(signals, PatternSignal{
				Type:         "triple_top",
				Bias:         analysis.DirectionShort,
				Confidence:   0.80,
				Invalidation: h.Price * 1.002,
				CandleRange:  [2]int{highs[len(highs)-3].Index, h.Index},
			})
		}
		if IsTripleBottom(lows, defaultTolerance) {
			l := lows[len(lows)-1]
			signals = append(signals, PatternSignal{
				Type:         "triple_bottom",
				Bias:         analysis.DirectionLong,
				Confidence:   0.80,
				Invalidation: l.Price * 0.998,
				CandleRange:  [2]int{lows[len(lows)-3].Index, l.Index},
			})
		}

		_ = lastClose // available for future context-aware filtering
	}

	return signals
}
