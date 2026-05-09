package simulation

import (
	"math"
	"time"

	"trade-buddy/api/internal/marketdata"
)

// SimOrder describes an order to simulate.
type SimOrder struct {
	Symbol    string
	Timeframe string
	Direction string  // "long" or "short"
	OrderType string  // "market", "buy_limit", "sell_limit"
	Entry     float64
	SL        float64
	TP        float64
	ExpiryBars int // 0 = no expiry
}

// SimOutcome holds the result of replaying an order against candles.
type SimOutcome struct {
	Outcome      string    // "tp", "sl", "expired", "ambiguous", "open"
	TriggeredAt  time.Time
	MAE          float64 // maximum adverse excursion (price distance against trade)
	MFE          float64 // maximum favorable excursion (price distance for trade)
	RMultiple    float64 // (MFE or exit price - entry) / (entry - SL)
	DurationBars int
	BarsToOutcome int
}

// ReplayOrder simulates an order bar-by-bar against the provided candles.
// candles must be in ascending time order.
func ReplayOrder(order SimOrder, candles []marketdata.Candle) SimOutcome {
	riskDist := math.Abs(order.Entry - order.SL)
	if riskDist == 0 {
		return SimOutcome{Outcome: "open"}
	}

	triggered := false
	triggerBar := -1

	// For market orders, entry is triggered immediately on first bar.
	// For limit orders, wait for price to reach entry.
	for i, c := range candles {
		if !triggered {
			if order.OrderType == "market" {
				triggered = true
				triggerBar = i
			} else if order.Direction == "long" && c.Low <= order.Entry {
				triggered = true
				triggerBar = i
			} else if order.Direction == "short" && c.High >= order.Entry {
				triggered = true
				triggerBar = i
			}
			if !triggered {
				if order.ExpiryBars > 0 && i >= order.ExpiryBars {
					return SimOutcome{Outcome: "expired", DurationBars: i}
				}
				continue
			}
			// fall through to TP/SL check on the same bar that triggered entry
		}

		// Order is live — check TP/SL
		barsSinceEntry := i - triggerBar
		if order.ExpiryBars > 0 && barsSinceEntry >= order.ExpiryBars {
			return SimOutcome{
				Outcome:       "expired",
				TriggeredAt:   candles[triggerBar].Time,
				DurationBars:  barsSinceEntry,
				BarsToOutcome: barsSinceEntry,
			}
		}

		tpHit := (order.Direction == "long" && c.High >= order.TP) ||
			(order.Direction == "short" && c.Low <= order.TP)
		slHit := (order.Direction == "long" && c.Low <= order.SL) ||
			(order.Direction == "short" && c.High >= order.SL)

		// Intrabar ambiguity: both TP and SL in same candle
		if tpHit && slHit {
			return SimOutcome{
				Outcome:       "ambiguous",
				TriggeredAt:   candles[triggerBar].Time,
				DurationBars:  barsSinceEntry,
				BarsToOutcome: barsSinceEntry,
			}
		}
		if tpHit {
			mae, mfe := calcMAEMFE(order, candles[triggerBar:i+1])
			return SimOutcome{
				Outcome:       "tp",
				TriggeredAt:   candles[triggerBar].Time,
				MAE:           mae,
				MFE:           mfe,
				RMultiple:     mfe / riskDist,
				DurationBars:  barsSinceEntry,
				BarsToOutcome: barsSinceEntry,
			}
		}
		if slHit {
			mae, mfe := calcMAEMFE(order, candles[triggerBar:i+1])
			return SimOutcome{
				Outcome:       "sl",
				TriggeredAt:   candles[triggerBar].Time,
				MAE:           mae,
				MFE:           mfe,
				RMultiple:     -1.0,
				DurationBars:  barsSinceEntry,
				BarsToOutcome: barsSinceEntry,
			}
		}
	}

	// Never resolved
	if triggered {
		mae, mfe := calcMAEMFE(order, candles[triggerBar:])
		return SimOutcome{
			Outcome:      "open",
			TriggeredAt:  candles[triggerBar].Time,
			MAE:          mae,
			MFE:          mfe,
			DurationBars: len(candles) - triggerBar,
		}
	}
	return SimOutcome{Outcome: "open"}
}

// calcMAEMFE computes maximum adverse and favorable excursion from entry.
func calcMAEMFE(order SimOrder, candles []marketdata.Candle) (mae, mfe float64) {
	for _, c := range candles {
		var favorable, adverse float64
		if order.Direction == "long" {
			favorable = c.High - order.Entry
			adverse = order.Entry - c.Low
		} else {
			favorable = order.Entry - c.Low
			adverse = c.High - order.Entry
		}
		if favorable > mfe {
			mfe = favorable
		}
		if adverse > mae {
			mae = adverse
		}
	}
	return mae, mfe
}
