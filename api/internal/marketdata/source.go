package marketdata

import (
	"fmt"
	"strings"
)

// NewSourceFromName returns a MarketDataSource for the given source name.
// Supported: "yahoo", "finnhub" (requires apiKey).
// Falls back to Yahoo if name is empty.
func NewSourceFromName(name string, finnhubAPIKey string) (MarketDataSource, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "":
		return NewYahooSource(), nil
	case "yahoo":
		return NewYahooSource(), nil
	case "finnhub":
		if strings.TrimSpace(finnhubAPIKey) == "" {
			return nil, fmt.Errorf("FINNHUB_API_KEY is required")
		}
		return NewFinnhubSource(finnhubAPIKey), nil
	default:
		return nil, fmt.Errorf("unsupported source: %s", name)
	}
}
