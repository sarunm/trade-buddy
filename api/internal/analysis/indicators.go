package analysis

import "trade-buddy/api/internal/marketdata"

// SMA returns a close-price simple moving average series.
// Entries before enough candles exist are 0.
func SMA(candles []marketdata.Candle, period int) []float64 {
	values := make([]float64, len(candles))
	if period <= 0 || len(candles) < period {
		return values
	}

	var sum float64
	for i, candle := range candles {
		sum += candle.Close
		if i >= period {
			sum -= candles[i-period].Close
		}
		if i >= period-1 {
			values[i] = sum / float64(period)
		}
	}
	return values
}

// ATR returns an average true range series using the Python reference formula.
// Entries before period+1 candles exist are 0.
func ATR(candles []marketdata.Candle, period int) []float64 {
	values := make([]float64, len(candles))
	if period <= 0 || len(candles) < period+1 {
		return values
	}

	trueRanges := make([]float64, len(candles))
	for i := 1; i < len(candles); i++ {
		current := candles[i]
		previous := candles[i-1]
		trueRanges[i] = maxFloat(
			current.High-current.Low,
			absFloat(current.High-previous.Close),
			absFloat(current.Low-previous.Close),
		)
	}

	var sum float64
	for i := 1; i < len(candles); i++ {
		sum += trueRanges[i]
		if i > period {
			sum -= trueRanges[i-period]
		}
		if i >= period {
			values[i] = sum / float64(period)
		}
	}
	return values
}

func maxFloat(values ...float64) float64 {
	if len(values) == 0 {
		return 0
	}
	maxValue := values[0]
	for _, value := range values[1:] {
		if value > maxValue {
			maxValue = value
		}
	}
	return maxValue
}

func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
