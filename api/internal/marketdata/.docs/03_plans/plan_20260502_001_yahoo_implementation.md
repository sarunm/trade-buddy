# Plan - Yahoo Implementation

- Task ID: 001
- Created: 2026-05-02 14:07
- Status: Planning
- Priority: High

## User Story / Objective
As a developer, I want a Yahoo Finance adapter in Go so that I can fetch market data for the trading buddy system.

## Acceptance Criteria
- [ ] `YahooSource` struct and `NewYahooSource` constructor.
- [ ] `Load` method implementing `MarketDataSource`.
- [ ] Symbol mapping: `XAUUSD`, `GOLD` -> `GC=F`.
- [ ] Timeframe mapping: `1m, 2m, 5m, 15m, 30m, 60m, 1h, 1d, 1w, 1mo`.
- [ ] Range logic based on limit and interval.
- [ ] User-Agent: `trade-buddy/0.1`.
- [ ] Filtering of candles where OHLC is missing/zero.
- [ ] Last `limit` candles returned.
- [ ] No external dependencies.

## Technical Design

### Mappings
- `mapSymbol(symbol string) string`
- `mapTimeframe(tf string) (interval string, err error)`
- `calculateRange(interval string, limit int) string`

### JSON Structure
```go
type yahooResponse struct {
	Chart struct {
		Result []struct {
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Open   []float64 `json:"open"`
					High   []float64 `json:"high"`
					Low    []float64 `json:"low"`
					Close  []float64 `json:"close"`
					Volume []float64 `json:"volume"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"chart"`
}
```

## Implementation Steps
1. Define `YahooSource` and constants.
2. Implement mapping and range functions.
3. Implement `Load` method:
    - Map inputs.
    - Construct URL.
    - Execute HTTP request with context and headers.
    - Decode JSON.
    - Iterate through results, validate, and convert to `Candle` slice.
    - Apply limit.
4. Run tests to verify.
