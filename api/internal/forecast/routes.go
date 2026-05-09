package forecast

import "trade-buddy/api/internal/analysis"

type ForecastPath struct {
	Label     string             `json:"label"`
	Direction analysis.Direction `json:"direction"`
	Priority  string             `json:"priority"`
	From      float64            `json:"from"`
	Via       float64            `json:"via"`
	To        float64            `json:"to"`
	Points    []float64          `json:"points"`
	Text      string             `json:"text"`
}

func WeeklyRoutes(bias analysis.Direction, close float64, supports, resistances []analysis.Level, atr float64) []ForecastPath {
	if close == 0 {
		return nil
	}

	var bullish []ForecastPath
	var bearish []ForecastPath

	support, support2 := levelPrice(supports, 0), levelPrice(supports, 1)
	resistance, resistance2 := levelPrice(resistances, 0), levelPrice(resistances, 1)

	if resistance != 0 {
		continuation := resistance2
		if continuation == 0 {
			continuation = resistance + atr*0.6
		}
		pullback := support
		if pullback == 0 {
			pullback = resistance - atr*0.6
		}
		bullish = append(bullish, ForecastPath{
			Label:     "ไป R1 แล้วต่อ R2",
			Direction: analysis.DirectionLong,
			Priority:  priorityFor(bias, analysis.DirectionLong),
			From:      round2(close),
			Via:       round2(resistance),
			To:        round2(continuation),
			Points:    []float64{round2(resistance), round2(continuation)},
			Text:      "ถ้าวิ่งขึ้นถึง R1 แล้วผ่านได้ มีโอกาสไปต่อ R2",
		})
		bearish = append(bearish, ForecastPath{
			Label:     "ไป R1 แล้วกลับ S1",
			Direction: analysis.DirectionShort,
			Priority:  priorityFor(bias, analysis.DirectionShort),
			From:      round2(close),
			Via:       round2(resistance),
			To:        round2(pullback),
			Points:    []float64{round2(resistance), round2(pullback)},
			Text:      "ถ้าวิ่งขึ้นถึง R1 แล้วโดนขาย มีโอกาสกลับลงมา S1",
		})
	}

	if support != 0 {
		continuation := support2
		if continuation == 0 {
			continuation = support - atr*0.6
		}
		rebound := support + atr*0.6
		bearish = append(bearish, ForecastPath{
			Label:     "ไป S1 แล้วต่อ S2",
			Direction: analysis.DirectionShort,
			Priority:  priorityFor(bias, analysis.DirectionShort),
			From:      round2(close),
			Via:       round2(support),
			To:        round2(continuation),
			Points:    []float64{round2(support), round2(continuation)},
			Text:      "ถ้าลงถึง S1 แล้วหลุด มีโอกาสไปต่อ S2",
		})
		bullish = append(bullish, ForecastPath{
			Label:     "ไป S1 แล้วเด้ง",
			Direction: analysis.DirectionLong,
			Priority:  priorityFor(bias, analysis.DirectionLong),
			From:      round2(close),
			Via:       round2(support),
			To:        round2(rebound),
			Points:    []float64{round2(support), round2(rebound)},
			Text:      "ถ้าลงถึง S1 แล้วยืนได้ มีโอกาสเด้งกลับไป",
		})
	}

	switch bias {
	case analysis.DirectionLong:
		return firstN(append(bullish, bearish...), 2)
	case analysis.DirectionShort:
		return firstN(append(bearish, bullish...), 2)
	default:
		return firstN(append(firstN(bullish, 1), firstN(bearish, 1)...), 2)
	}
}

func levelPrice(levels []analysis.Level, index int) float64 {
	if index >= len(levels) {
		return 0
	}
	return levels[index].Price
}

func priorityFor(bias, direction analysis.Direction) string {
	if bias == direction {
		return "primary"
	}
	return "alternative"
}

func firstN(paths []ForecastPath, limit int) []ForecastPath {
	if len(paths) <= limit {
		return paths
	}
	return paths[:limit]
}
