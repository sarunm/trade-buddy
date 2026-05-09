package analysis

// FibLevels holds standard Fibonacci retracement levels for a swing leg.
type FibLevels struct {
	High   float64
	Low    float64
	Levels map[string]float64
}

// SwingLegFib computes standard Fibonacci retracement levels from a swing high to low.
// Levels are absolute prices (not percentages).
func SwingLegFib(high, low float64) FibLevels {
	diff := high - low
	return FibLevels{
		High: high,
		Low:  low,
		Levels: map[string]float64{
			"0.000": low,
			"0.236": low + diff*0.236,
			"0.382": low + diff*0.382,
			"0.500": low + diff*0.500,
			"0.618": low + diff*0.618,
			"0.786": low + diff*0.786,
			"1.000": high,
		},
	}
}
