package marketdata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	finnhubBaseURL   = "https://finnhub.io/api/v1/forex/candle"
	finnhubUserAgent = "trade-buddy/0.1"
)

// FinnhubSource implements MarketDataSource for Finnhub Forex candles.
type FinnhubSource struct {
	apiKey string
	client *http.Client
}

func NewFinnhubSource(apiKey string) *FinnhubSource {
	return &FinnhubSource{
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *FinnhubSource) Load(ctx context.Context, symbol string, timeframe string, limit int) ([]Candle, error) {
	if strings.TrimSpace(s.apiKey) == "" {
		return nil, fmt.Errorf("FINNHUB_API_KEY is required")
	}

	finnhubSymbol := mapFinnhubSymbol(symbol)
	resolution, err := mapFinnhubResolution(timeframe)
	if err != nil {
		return nil, err
	}
	from, to := finnhubRange(resolution, limit)

	values := url.Values{}
	values.Set("symbol", finnhubSymbol)
	values.Set("resolution", resolution)
	values.Set("from", fmt.Sprintf("%d", from))
	values.Set("to", fmt.Sprintf("%d", to))
	values.Set("token", s.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, finnhubBaseURL+"?"+values.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", finnhubUserAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		if len(body) > 0 {
			return nil, fmt.Errorf("finnhub returned status: %s: %s", resp.Status, strings.TrimSpace(string(body)))
		}
		return nil, fmt.Errorf("finnhub returned status: %s", resp.Status)
	}

	var data finnhubCandleResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if data.Error != "" {
		return nil, fmt.Errorf("finnhub error: %s", data.Error)
	}
	if data.Status == "no_data" {
		return nil, nil
	}
	if data.Status != "" && data.Status != "ok" {
		return nil, fmt.Errorf("finnhub status: %s", data.Status)
	}

	n := min(len(data.Time), len(data.Open), len(data.High), len(data.Low), len(data.Close))
	candles := make([]Candle, 0, n)
	for i := 0; i < n; i++ {
		if data.Open[i] == 0 || data.High[i] == 0 || data.Low[i] == 0 || data.Close[i] == 0 {
			continue
		}
		volume := 0.0
		if i < len(data.Volume) {
			volume = data.Volume[i]
		}
		candles = append(candles, Candle{
			Time:   time.Unix(data.Time[i], 0).UTC(),
			Open:   data.Open[i],
			High:   data.High[i],
			Low:    data.Low[i],
			Close:  data.Close[i],
			Volume: volume,
		})
	}
	if len(candles) > limit {
		candles = candles[len(candles)-limit:]
	}
	return candles, nil
}

func mapFinnhubSymbol(symbol string) string {
	switch strings.ToUpper(strings.TrimSpace(symbol)) {
	case "XAUUSD", "GOLD":
		return "OANDA:XAU_USD"
	default:
		return strings.ToUpper(strings.TrimSpace(symbol))
	}
}

func mapFinnhubResolution(tf string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(tf)) {
	case "1m":
		return "1", nil
	case "5m":
		return "5", nil
	case "15m":
		return "15", nil
	case "30m":
		return "30", nil
	case "60m", "1h":
		return "60", nil
	case "1d":
		return "D", nil
	case "1w", "1wk", "1week":
		return "W", nil
	case "1mo", "1month":
		return "M", nil
	default:
		return "", fmt.Errorf("unsupported timeframe: %s", tf)
	}
}

func finnhubRange(resolution string, limit int) (int64, int64) {
	if limit < 1 {
		limit = 500
	}
	to := time.Now().UTC()
	var step time.Duration
	switch resolution {
	case "1":
		step = time.Minute
	case "5":
		step = 5 * time.Minute
	case "15":
		step = 15 * time.Minute
	case "30":
		step = 30 * time.Minute
	case "60":
		step = time.Hour
	case "D":
		step = 24 * time.Hour
	case "W":
		step = 7 * 24 * time.Hour
	case "M":
		step = 31 * 24 * time.Hour
	default:
		step = time.Hour
	}
	// Ask for extra bars because forex sessions can have gaps.
	from := to.Add(-time.Duration(limit+50) * step)
	return from.Unix(), to.Unix()
}

type finnhubCandleResponse struct {
	Close  []float64 `json:"c"`
	High   []float64 `json:"h"`
	Low    []float64 `json:"l"`
	Open   []float64 `json:"o"`
	Status string    `json:"s"`
	Time   []int64   `json:"t"`
	Volume []float64 `json:"v"`
	Error  string    `json:"error"`
}
