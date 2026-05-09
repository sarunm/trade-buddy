package analysis

import "trade-buddy/api/internal/marketdata"

type Direction string

const (
	DirectionLong    Direction = "long"
	DirectionShort   Direction = "short"
	DirectionNeutral Direction = "neutral"
)

type SwingPoint struct {
	Index int     `json:"index"`
	Kind  string  `json:"kind"`
	Price float64 `json:"price"`
}

func DetectTrend(candles []marketdata.Candle) Direction {
	if len(candles) == 0 {
		return DirectionNeutral
	}

	fast := SMA(candles, 8)
	slow := SMA(candles, 21)
	lastIndex := len(candles) - 1
	if fast[lastIndex] == 0 || slow[lastIndex] == 0 {
		return DirectionNeutral
	}

	lastClose := candles[lastIndex].Close
	if fast[lastIndex] > slow[lastIndex] && lastClose > fast[lastIndex] {
		return DirectionLong
	}
	if fast[lastIndex] < slow[lastIndex] && lastClose < fast[lastIndex] {
		return DirectionShort
	}
	return DirectionNeutral
}

func DetectSwings(candles []marketdata.Candle, radius int) (highs, lows []SwingPoint) {
	if radius <= 0 || len(candles) < radius*2+1 {
		return nil, nil
	}

	for index := radius; index < len(candles)-radius; index++ {
		current := candles[index]
		if isSwingHigh(candles, index, radius) {
			highs = append(highs, SwingPoint{Index: index, Kind: "high", Price: current.High})
		}
		if isSwingLow(candles, index, radius) {
			lows = append(lows, SwingPoint{Index: index, Kind: "low", Price: current.Low})
		}
	}
	return highs, lows
}

func isSwingHigh(candles []marketdata.Candle, index int, radius int) bool {
	price := candles[index].High
	for i := index - radius; i <= index+radius; i++ {
		if candles[i].High > price {
			return false
		}
	}
	return true
}

func isSwingLow(candles []marketdata.Candle, index int, radius int) bool {
	price := candles[index].Low
	for i := index - radius; i <= index+radius; i++ {
		if candles[i].Low < price {
			return false
		}
	}
	return true
}
