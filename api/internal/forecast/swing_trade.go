package forecast

import (
	"strconv"

	"trade-buddy/api/internal/analysis"
)

type SwingTrade struct {
	Available   bool               `json:"available"`
	Direction   analysis.Direction `json:"direction"`
	TradeType   string             `json:"trade_type"`
	Summary     string             `json:"summary"`
	Action      string             `json:"action"`
	EntryLow    *float64           `json:"entry_low,omitempty"`
	EntryHigh   *float64           `json:"entry_high,omitempty"`
	StopLoss    *float64           `json:"stop_loss,omitempty"`
	TP1         *float64           `json:"tp1,omitempty"`
	TP2         *float64           `json:"tp2,omitempty"`
	HoldingNote string             `json:"holding_note,omitempty"`
	InvalidIf   string             `json:"invalid_if,omitempty"`
	Why         []string           `json:"why,omitempty"`
}

type TrendContext struct {
	Timeframe  string
	Trend      analysis.Direction
	Support    float64
	Resistance float64
	ATR        float64
}

func SwingTradeBias(bias BiasResult, daily, execution TrendContext, supports, resistances []analysis.Level, atr float64) SwingTrade {
	direction := bias.Direction
	if direction == analysis.DirectionNeutral {
		return SwingTrade{
			Available: true,
			Direction: direction,
			TradeType: "รอภาพชัด",
			Summary:   "ภาพรวม 1M/1W ยังไม่เลือกทางชัดเจน ยังไม่ควรรีบวาง swing trade",
			Action:    "รอให้ D1 หรือ H1 ปิดเลือกทางก่อน",
		}
	}

	support := firstNonZero(execution.Support, daily.Support, levelPrice(supports, 0))
	resistance := firstNonZero(execution.Resistance, daily.Resistance, levelPrice(resistances, 0))
	referenceATR := firstNonZero(execution.ATR, daily.ATR, atr, 1.0)

	dailyTrend := daily.Trend
	if dailyTrend == "" {
		dailyTrend = analysis.DirectionNeutral
	}
	executionTrend := execution.Trend
	if executionTrend == "" {
		executionTrend = analysis.DirectionNeutral
	}

	tradeType, holdingNote := swingTradeType(direction, dailyTrend, executionTrend)

	trade := SwingTrade{
		Available:   true,
		Direction:   direction,
		TradeType:   tradeType,
		Summary:     "ภาพรวม 1M/1W ให้น้ำหนัก" + directionTH(direction) + " ส่วน D1 ตอนนี้" + alignmentText(dailyTrend, direction),
		HoldingNote: holdingNote,
		Why: []string{
			"1M/1W forecast = " + directionTH(direction),
			"D1 = " + trendTH(dailyTrend),
			defaultString(execution.Timeframe, "H1/H4") + " = " + trendTH(executionTrend),
		},
	}

	if direction == analysis.DirectionShort {
		trade.Action = "รอเด้งขึ้นมาโซนต้าน แล้วค่อยหา short"
		if resistance != 0 {
			trade.EntryLow = floatPtr(round2(resistance - referenceATR*0.35))
			trade.EntryHigh = floatPtr(round2(resistance + referenceATR*0.15))
			trade.StopLoss = floatPtr(round2(resistance + referenceATR*0.75))
			trade.InvalidIf = "แผน short เสียทรงถ้า H1/H4 ปิดเหนือ " + formatPrice(*trade.StopLoss)
		} else {
			trade.InvalidIf = "แผน short เสียถ้าราคายืนเหนือแนวต้านสำคัญ"
		}
		if support != 0 {
			trade.TP1 = floatPtr(round2(support))
			trade.TP2 = floatPtr(round2(firstNonZero(levelPrice(supports, 1), support-referenceATR*1.4)))
		}
		return trade
	}

	trade.Action = "รอย่อลงมาโซนรับ แล้วค่อยหา long"
	if support != 0 {
		trade.EntryLow = floatPtr(round2(support - referenceATR*0.15))
		trade.EntryHigh = floatPtr(round2(support + referenceATR*0.35))
		trade.StopLoss = floatPtr(round2(support - referenceATR*0.75))
		trade.InvalidIf = "แผน long เสียทรงถ้า H1/H4 ปิดต่ำกว่า " + formatPrice(*trade.StopLoss)
	} else {
		trade.InvalidIf = "แผน long เสียถ้าราคาหลุดแนวรับสำคัญ"
	}
	if resistance != 0 {
		trade.TP1 = floatPtr(round2(resistance))
		trade.TP2 = floatPtr(round2(firstNonZero(levelPrice(resistances, 1), resistance+referenceATR*1.4)))
	}
	return trade
}

func swingTradeType(direction, dailyTrend, executionTrend analysis.Direction) (string, string) {
	alignedDaily := dailyTrend == direction
	alignedExecution := executionTrend == direction
	if alignedDaily && alignedExecution {
		return "ถือสวิงได้", "D1 และ H1 ไปทางเดียวกับภาพใหญ่ ถือเป้าถัดไปได้มากขึ้น"
	}
	if alignedDaily {
		return "สวิงสั้น", "D1 ไปตามภาพใหญ่ แต่ H1 ยังไม่พร้อม ให้รอจังหวะเข้า"
	}
	return "เล่นสั้นพอ", "D1 ยังสวนภาพใหญ่ ถ้าเข้าควรเก็บสั้นและลดความโลภ"
}

func firstNonZero(values ...float64) float64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func floatPtr(value float64) *float64 {
	return &value
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func alignmentText(actual, expected analysis.Direction) string {
	if actual == expected {
		return "กำลังเดินตามแผน"
	}
	if actual == analysis.DirectionNeutral || actual == "" {
		return "ยังไม่ยืนยัน ต้องรอจังหวะ"
	}
	return "ยังสวนแผนอยู่ ควรเล่นสั้นหรือรอ pullback"
}

func trendTH(direction analysis.Direction) string {
	switch direction {
	case analysis.DirectionLong:
		return "เทรนขาขึ้น"
	case analysis.DirectionShort:
		return "เทรนขาลง"
	default:
		return "ภาวะยังไม่ชัด"
	}
}

func directionTH(direction analysis.Direction) string {
	switch direction {
	case analysis.DirectionLong:
		return "ขาขึ้น"
	case analysis.DirectionShort:
		return "ขาลง"
	default:
		return "รอดู"
	}
}

func formatPrice(value float64) string {
	return trimTrailingZeros(value)
}

func trimTrailingZeros(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
