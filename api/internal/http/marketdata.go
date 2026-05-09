package httpapi

import (
	"context"
	"strings"

	"trade-buddy/api/internal/marketdata"
)

func (s *Server) defaultSource() string {
	if strings.TrimSpace(s.cfg.DefaultSource) == "" {
		return "finnhub"
	}
	return strings.ToLower(strings.TrimSpace(s.cfg.DefaultSource))
}

func (s *Server) marketDataSource(source string) (marketdata.MarketDataSource, error) {
	return marketdata.NewSourceFromName(source, s.cfg.FinnhubAPIKey)
}

func (s *Server) loadMarketCandles(
	ctx context.Context,
	symbol string,
	timeframe string,
	source string,
	limit int,
) ([]marketdata.Candle, string, error) {
	src, err := s.marketDataSource(source)
	if err != nil {
		return nil, source, err
	}
	candles, err := src.Load(ctx, symbol, timeframe, limit)
	if err == nil {
		return candles, source, nil
	}
	if source != "finnhub" {
		return nil, source, err
	}

	fallback := "yahoo"
	fallbackSrc, fallbackErr := s.marketDataSource(fallback)
	if fallbackErr != nil {
		return nil, source, err
	}
	fallbackCandles, fallbackErr := fallbackSrc.Load(ctx, symbol, timeframe, limit)
	if fallbackErr != nil {
		return nil, source, err
	}
	return fallbackCandles, fallback, nil
}
