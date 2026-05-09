package marketdata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	yahooBaseURL   = "https://query1.finance.yahoo.com/v8/finance/chart"
	yahooUserAgent = "trade-buddy/0.1"
)

// YahooSource implements MarketDataSource for Yahoo Finance.
type YahooSource struct {
	client *http.Client
}

// NewYahooSource creates a new Yahoo Finance data source.
func NewYahooSource() *YahooSource {
	return &YahooSource{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Load fetches market data from Yahoo Finance.
func (s *YahooSource) Load(ctx context.Context, symbol string, timeframe string, limit int) ([]Candle, error) {
	yahooSymbol := mapYahooSymbol(symbol)
	interval, err := mapYahooInterval(timeframe)
	if err != nil {
		return nil, err
	}
	rangeStr := calculateYahooRange(interval, limit)

	url := fmt.Sprintf("%s/%s?interval=%s&range=%s", yahooBaseURL, yahooSymbol, interval, rangeStr)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", yahooUserAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo finance returned status: %s", resp.Status)
	}

	var data yahooResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(data.Chart.Result) == 0 {
		return nil, fmt.Errorf("no results from yahoo finance")
	}

	result := data.Chart.Result[0]
	if len(result.Timestamp) == 0 {
		return nil, nil // No data
	}

	quote := result.Indicators.Quote[0]
	candles := make([]Candle, 0, len(result.Timestamp))

	for i, ts := range result.Timestamp {
		if i >= len(quote.Open) || i >= len(quote.High) || i >= len(quote.Low) || i >= len(quote.Close) || i >= len(quote.Volume) {
			break
		}

		open, high, low, close, volume, ok := yahooQuoteValues(quote.Open[i], quote.High[i], quote.Low[i], quote.Close[i], quote.Volume[i])
		if !ok {
			continue
		}

		candles = append(candles, Candle{
			Time:   time.Unix(ts, 0),
			Open:   open,
			High:   high,
			Low:    low,
			Close:  close,
			Volume: volume,
		})
	}

	// Return last 'limit' candles
	if len(candles) > limit {
		candles = candles[len(candles)-limit:]
	}

	return candles, nil
}

func mapYahooSymbol(symbol string) string {
	s := strings.ToUpper(symbol)
	switch s {
	case "XAUUSD", "GOLD":
		return "GC=F"
	default:
		return s
	}
}

func mapYahooInterval(tf string) (string, error) {
	switch strings.ToLower(tf) {
	case "1m":
		return "1m", nil
	case "2m":
		return "2m", nil
	case "5m":
		return "5m", nil
	case "15m":
		return "15m", nil
	case "30m":
		return "30m", nil
	case "60m":
		return "60m", nil
	case "1h":
		return "1h", nil
	case "1d":
		return "1d", nil
	case "1w", "1wk", "1week":
		return "1wk", nil
	case "1mo", "1moth": // note: "1moth" was likely a typo in my thought but I'll stick to prompt "1month"
		return "1mo", nil
	case "1month":
		return "1mo", nil
	default:
		return "", fmt.Errorf("unsupported timeframe: %s", tf)
	}
}

func calculateYahooRange(interval string, limit int) string {
	switch interval {
	case "1m", "2m", "5m", "15m", "30m", "60m", "1h":
		if limit <= 390 {
			return "5d"
		}
		if limit <= 1500 {
			return "1mo"
		}
		return "3mo"
	case "1d":
		if limit <= 90 {
			return "3mo"
		}
		if limit <= 365 {
			return "1y"
		}
		return "2y"
	default:
		return "5y"
	}
}

func yahooQuoteValues(open, high, low, close, volume *float64) (float64, float64, float64, float64, float64, bool) {
	if open == nil || high == nil || low == nil || close == nil {
		return 0, 0, 0, 0, 0, false
	}
	if *open == 0 || *high == 0 || *low == 0 || *close == 0 {
		return 0, 0, 0, 0, 0, false
	}
	if volume == nil {
		return *open, *high, *low, *close, 0, true
	}
	return *open, *high, *low, *close, *volume, true
}

type yahooResponse struct {
	Chart struct {
		Result []struct {
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Open   []*float64 `json:"open"`
					High   []*float64 `json:"high"`
					Low    []*float64 `json:"low"`
					Close  []*float64 `json:"close"`
					Volume []*float64 `json:"volume"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"chart"`
}
