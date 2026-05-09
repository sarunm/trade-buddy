package httpapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"trade-buddy/api/internal/analysis"
	dbstore "trade-buddy/api/internal/db"
)

type tfContextJSON struct {
	Timeframe   string                `json:"timeframe"`
	Trend       string                `json:"trend"`
	SwingHighs  []analysis.SwingPoint `json:"swing_highs"`
	SwingLows   []analysis.SwingPoint `json:"swing_lows"`
	Supports    []analysis.Level      `json:"supports"`
	Resistances []analysis.Level      `json:"resistances"`
}

type topDownResponse struct {
	Symbol          string        `json:"symbol"`
	CapturedAt      string        `json:"captured_at"`
	DetectorVersion string        `json:"detector_version"`
	Monthly         tfContextJSON `json:"monthly"`
	Weekly          tfContextJSON `json:"weekly"`
	Daily           tfContextJSON `json:"daily"`
	H4              tfContextJSON `json:"h4"`
	H1              tfContextJSON `json:"h1"`
	M15             tfContextJSON `json:"m15"`
}

func (s *Server) topDownContext(c *gin.Context) {
	if s.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
		return
	}

	symbol := strings.ToUpper(strings.TrimSpace(defaultString(c.Query("symbol"), "XAUUSD")))
	source := strings.ToLower(strings.TrimSpace(defaultString(c.Query("source"), s.defaultSource())))
	ctx := c.Request.Context()

	src, err := s.marketDataSource(source)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tdCtx, err := analysis.BuildTopDownContext(ctx, symbol, src)
	if err != nil && source == "finnhub" {
		fallback, fallbackErr := s.marketDataSource("yahoo")
		if fallbackErr == nil {
			tdCtx, err = analysis.BuildTopDownContext(ctx, symbol, fallback)
		}
	}
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "build context: " + err.Error()})
		return
	}

	if err := dbstore.SaveTopDownContext(ctx, s.db, tdCtx); err != nil {
		// log but don't fail — return context anyway
		_ = err
	}

	c.JSON(http.StatusOK, toTopDownResponse(tdCtx))
}

func (s *Server) topDownContextHistory(c *gin.Context) {
	if s.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
		return
	}

	symbol := strings.ToUpper(strings.TrimSpace(defaultString(c.Query("symbol"), "XAUUSD")))
	ctx := c.Request.Context()

	row, err := dbstore.LatestTopDownContext(ctx, s.db, symbol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if row == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no context found for symbol"})
		return
	}
	c.JSON(http.StatusOK, row)
}

func toTopDownResponse(c analysis.TopDownContext) topDownResponse {
	return topDownResponse{
		Symbol:          c.Symbol,
		CapturedAt:      c.CapturedAt.UTC().Format("2006-01-02T15:04:05Z"),
		DetectorVersion: c.DetectorVersion,
		Monthly:         toTFContextJSON(c.Monthly),
		Weekly:          toTFContextJSON(c.Weekly),
		Daily:           toTFContextJSON(c.Daily),
		H4:              toTFContextJSON(c.H4),
		H1:              toTFContextJSON(c.H1),
		M15:             toTFContextJSON(c.M15),
	}
}

func toTFContextJSON(t analysis.TFContext) tfContextJSON {
	highs := t.SwingHighs
	if highs == nil {
		highs = []analysis.SwingPoint{}
	}
	lows := t.SwingLows
	if lows == nil {
		lows = []analysis.SwingPoint{}
	}
	supports := t.Supports
	if supports == nil {
		supports = []analysis.Level{}
	}
	resistances := t.Resistances
	if resistances == nil {
		resistances = []analysis.Level{}
	}
	return tfContextJSON{
		Timeframe:   t.Timeframe,
		Trend:       string(t.Trend),
		SwingHighs:  highs,
		SwingLows:   lows,
		Supports:    supports,
		Resistances: resistances,
	}
}
