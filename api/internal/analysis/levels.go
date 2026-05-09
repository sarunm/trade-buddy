package analysis

import (
	"math"
	"sort"

	"trade-buddy/api/internal/marketdata"
)

type Level struct {
	Label string  `json:"label"`
	Price float64 `json:"price"`
	Kind  string  `json:"kind"`
}

func WeeklyLevels(candles []marketdata.Candle, close float64) (supports, resistances []Level) {
	window := lastCandles(candles, 52)
	if len(window) == 0 {
		return nil, nil
	}

	minGap := weeklyLevelMinGap(window, close)
	highs, lows := DetectSwings(window, 2)

	swingSupports := make([]float64, 0, len(lows))
	for _, point := range lows {
		if point.Price < close {
			swingSupports = append(swingSupports, point.Price)
		}
	}
	candleSupports := make([]float64, 0, len(window))
	for _, candle := range window {
		if candle.Low < close {
			candleSupports = append(candleSupports, candle.Low)
		}
	}

	swingResistances := make([]float64, 0, len(highs))
	for _, point := range highs {
		if point.Price > close {
			swingResistances = append(swingResistances, point.Price)
		}
	}
	candleResistances := make([]float64, 0, len(window))
	for _, candle := range window {
		if candle.High > close {
			candleResistances = append(candleResistances, candle.High)
		}
	}

	sortByDistance(swingSupports, close)
	sortByDistance(candleSupports, close)
	sortByDistance(swingResistances, close)
	sortByDistance(candleResistances, close)

	supportPrices := selectSpacedLevels(append(swingSupports, candleSupports...), minGap, 3)
	resistancePrices := selectSpacedLevels(append(swingResistances, candleResistances...), minGap, 3)

	for i, price := range supportPrices {
		supports = append(supports, Level{Label: levelLabel("S", i), Price: price, Kind: levelKind("support", i)})
	}
	for i, price := range resistancePrices {
		resistances = append(resistances, Level{Label: levelLabel("R", i), Price: price, Kind: levelKind("resistance", i)})
	}
	return supports, resistances
}

func lastCandles(candles []marketdata.Candle, limit int) []marketdata.Candle {
	if len(candles) <= limit {
		return candles
	}
	return candles[len(candles)-limit:]
}

func weeklyLevelMinGap(candles []marketdata.Candle, close float64) float64 {
	recent := lastCandles(candles, 26)
	ranges := make([]float64, 0, len(recent))
	for _, candle := range recent {
		ranges = append(ranges, candle.High-candle.Low)
	}
	sort.Float64s(ranges)

	var medianRange float64
	if len(ranges) > 0 {
		medianRange = ranges[len(ranges)/2]
	}
	return maxFloat(medianRange*0.65, math.Abs(close)*0.009, 30.0)
}

func sortByDistance(values []float64, close float64) {
	sort.SliceStable(values, func(i, j int) bool {
		return math.Abs(values[i]-close) < math.Abs(values[j]-close)
	})
}

func selectSpacedLevels(candidates []float64, minGap float64, limit int) []float64 {
	selected := make([]float64, 0, limit)
	seen := map[float64]bool{}
	for _, candidate := range candidates {
		price := round2(candidate)
		if seen[price] {
			continue
		}
		seen[price] = true

		spaced := true
		for _, existing := range selected {
			if math.Abs(price-existing) < minGap {
				spaced = false
				break
			}
		}
		if spaced {
			selected = append(selected, price)
			if len(selected) == limit {
				break
			}
		}
	}
	return selected
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func levelLabel(prefix string, index int) string {
	return prefix + string(rune('1'+index))
}

func levelKind(prefix string, index int) string {
	return prefix + string(rune('1'+index))
}
