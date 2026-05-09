package analysis

import "math"

// AccountConfig holds broker/account parameters for XAUUSD risk calculations.
type AccountConfig struct {
	Equity       float64 // account equity in USD
	LotSize      float64 // lot size (e.g. 0.01, 0.1, 1.0)
	ContractSize float64 // units per lot (XAUUSD standard = 100 oz)
	PointValue   float64 // USD value per 1-point move per lot (XAUUSD = 1.0 per 0.01 move)
	Leverage     float64 // e.g. 100
	Spread       float64 // spread in points (e.g. 30 = $0.30)
	Slippage     float64 // slippage assumption in points
}

// DefaultXAUUSDConfig returns a sensible default for a micro-lot XAUUSD account.
func DefaultXAUUSDConfig() AccountConfig {
	return AccountConfig{
		Equity:       10000,
		LotSize:      0.01,
		ContractSize: 100,
		PointValue:   0.01, // $0.01 per point per 0.01 lot
		Leverage:     100,
		Spread:       30,
		Slippage:     10,
	}
}

// RiskResult summarises the risk exposure for a given trade setup.
type RiskResult struct {
	Points      float64 // distance entry→SL in points (price * 100 for XAUUSD)
	DollarRisk  float64 // total USD at risk
	MarginReq   float64 // approximate margin required
	RiskPercent float64 // DollarRisk / Equity * 100
	Warning     string  // non-empty if risk is elevated
	Error       string  // non-empty if risk is dangerously high
}

// RiskCheck evaluates the risk for an entry/SL pair given account config.
func RiskCheck(entry, sl float64, cfg AccountConfig) RiskResult {
	points := math.Abs(entry-sl) * 100 // XAUUSD: 1 point = $0.01
	dollarPerPoint := cfg.LotSize * cfg.ContractSize * cfg.PointValue
	dollarRisk := points * dollarPerPoint
	marginReq := (cfg.LotSize * cfg.ContractSize * entry) / cfg.Leverage
	riskPct := 0.0
	if cfg.Equity > 0 {
		riskPct = dollarRisk / cfg.Equity * 100
	}

	res := RiskResult{
		Points:      points,
		DollarRisk:  dollarRisk,
		MarginReq:   marginReq,
		RiskPercent: riskPct,
	}
	if points > 10000 {
		res.Error = "SL exceeds 10,000 points — position will likely be wiped"
	} else if points > 5000 {
		res.Warning = "SL exceeds 5,000 points — high risk, review position size"
	}
	return res
}

// LotFromRiskPercent calculates the lot size needed to risk exactly riskPct% of equity.
func LotFromRiskPercent(equity, riskPct, entry, sl float64, cfg AccountConfig) float64 {
	if equity <= 0 || riskPct <= 0 || entry == sl {
		return 0
	}
	dollarRisk := equity * riskPct / 100
	points := math.Abs(entry-sl) * 100
	dollarPerPointPerLot := cfg.ContractSize * cfg.PointValue
	if dollarPerPointPerLot == 0 || points == 0 {
		return 0
	}
	return dollarRisk / (points * dollarPerPointPerLot)
}
