package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	dbstore "trade-buddy/api/internal/db"
	"trade-buddy/api/internal/marketdata"
)

type chartResponse struct {
	Symbol    string         `json:"symbol"`
	Timeframe string         `json:"timeframe"`
	Source    string         `json:"source"`
	UpdatedAt time.Time      `json:"updated_at"`
	Candles   []chartCandle  `json:"candles"`
	Levels    []chartLevel   `json:"levels"`
	Markers   []chartMarker  `json:"markers"`
	Overlays  []chartOverlay `json:"overlays"`
}

type chartCandle struct {
	Time   int64   `json:"time"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
}

type chartLevel struct {
	Label string  `json:"label"`
	Price float64 `json:"price"`
	Kind  string  `json:"kind"`
}

type chartMarker struct {
	Time     int64  `json:"time"`
	Position string `json:"position"`
	Color    string `json:"color"`
	Shape    string `json:"shape"`
	Text     string `json:"text"`
}

type chartOverlay struct {
	ID        string         `json:"id"`
	Kind      string         `json:"kind"`
	Name      string         `json:"name"`
	Direction string         `json:"direction"`
	Geometry  map[string]any `json:"geometry"`
}

func (s *Server) chart(c *gin.Context) {
	if s.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
		return
	}

	symbol := strings.ToUpper(strings.TrimSpace(defaultString(c.Query("symbol"), "XAUUSD")))
	timeframe := strings.ToLower(strings.TrimSpace(defaultString(c.Query("tf"), "1h")))
	source := strings.ToLower(strings.TrimSpace(defaultString(c.Query("source"), s.defaultSource())))
	limit, err := parseLimit(c.Query("limit"), 500)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	candles, err := dbstore.FetchBars(ctx, s.db, symbol, timeframe, source, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "fetch bars: " + err.Error()})
		return
	}

	if len(candles) < limit {
		fetched, effectiveSource, err := s.loadMarketCandles(ctx, symbol, timeframe, source, limit)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "fetch " + source + " candles: " + err.Error()})
			return
		}
		source = effectiveSource
		if err := dbstore.UpsertBars(ctx, s.db, symbol, timeframe, source, fetched); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "upsert bars: " + err.Error()})
			return
		}
		candles, err = dbstore.FetchBars(ctx, s.db, symbol, timeframe, source, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "fetch bars: " + err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, chartResponse{
		Symbol:    symbol,
		Timeframe: timeframe,
		Source:    source,
		UpdatedAt: time.Now().UTC(),
		Candles:   chartCandles(candles),
		Levels:    []chartLevel{},
		Markers:   []chartMarker{},
		Overlays:  []chartOverlay{},
	})
}

func chartCandles(candles []marketdata.Candle) []chartCandle {
	out := make([]chartCandle, len(candles))
	for i, candle := range candles {
		out[i] = chartCandle{
			Time:   candle.Time.Unix(),
			Open:   candle.Open,
			High:   candle.High,
			Low:    candle.Low,
			Close:  candle.Close,
			Volume: candle.Volume,
		}
	}
	return out
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func parseLimit(value string, fallback int) (int, error) {
	if strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	if limit < 1 {
		return 0, strconv.ErrSyntax
	}
	if limit > 5000 {
		return 5000, nil
	}
	return limit, nil
}
