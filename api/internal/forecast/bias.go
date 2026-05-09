package forecast

import (
	"math"

	"trade-buddy/api/internal/analysis"
)

type BiasResult struct {
	Direction  analysis.Direction `json:"direction"`
	Confidence float64            `json:"confidence"`
	Reasons    []string           `json:"reasons"`
}

func ForecastBias(monthlyTrend, weeklyTrend analysis.Direction) BiasResult {
	score := directionScore(weeklyTrend)
	reasons := []string{"1W trend=" + string(weeklyTrend)}

	if monthlyTrend != "" {
		score += directionScore(monthlyTrend) * 2
		reasons = append([]string{"1M trend=" + string(monthlyTrend)}, reasons...)
	}

	direction := analysis.DirectionNeutral
	if score > 0 {
		direction = analysis.DirectionLong
	} else if score < 0 {
		direction = analysis.DirectionShort
	}

	confidence := math.Min(math.Abs(float64(score))/3.0, 1.0)
	return BiasResult{
		Direction:  direction,
		Confidence: round2(confidence),
		Reasons:    reasons,
	}
}

func directionScore(direction analysis.Direction) int {
	switch direction {
	case analysis.DirectionLong:
		return 1
	case analysis.DirectionShort:
		return -1
	default:
		return 0
	}
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
