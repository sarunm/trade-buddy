package httpapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	dbstore "trade-buddy/api/internal/db"
	"trade-buddy/api/internal/simulation"
)

type createSimRequest struct {
	Symbol     string  `json:"symbol" binding:"required"`
	Timeframe  string  `json:"timeframe" binding:"required"`
	Direction  string  `json:"direction" binding:"required"`
	OrderType  string  `json:"order_type" binding:"required"`
	Entry      float64 `json:"entry" binding:"required"`
	SL         float64 `json:"sl" binding:"required"`
	TP         float64 `json:"tp" binding:"required"`
	ExpiryBars int     `json:"expiry_bars"`
}

func (s *Server) createSimulation(c *gin.Context) {
	if s.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
		return
	}

	var req createSimRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Symbol = strings.ToUpper(strings.TrimSpace(req.Symbol))

	order := simulation.SimOrder{
		Symbol:     req.Symbol,
		Timeframe:  req.Timeframe,
		Direction:  req.Direction,
		OrderType:  req.OrderType,
		Entry:      req.Entry,
		SL:         req.SL,
		TP:         req.TP,
		ExpiryBars: req.ExpiryBars,
	}

	ctx := c.Request.Context()
	source := s.defaultSource()
	simID, err := dbstore.CreateSimulation(ctx, s.db, order)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create simulation: " + err.Error()})
		return
	}

	// Replay immediately using cached candles
	candles, _ := dbstore.FetchBars(ctx, s.db, req.Symbol, req.Timeframe, source, 500)
	if len(candles) == 0 {
		fetched, effectiveSource, err := s.loadMarketCandles(ctx, req.Symbol, req.Timeframe, source, 500)
		if err == nil {
			_ = dbstore.UpsertBars(ctx, s.db, req.Symbol, req.Timeframe, effectiveSource, fetched)
			candles = fetched
		}
	}

	var outcome *simulation.SimOutcome
	if len(candles) > 0 {
		out := simulation.ReplayOrder(order, candles)
		outcome = &out
		_ = dbstore.SaveOutcome(ctx, s.db, simID, out)
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":      simID,
		"outcome": outcome,
	})
}

func (s *Server) listSimulations(c *gin.Context) {
	if s.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
		return
	}
	symbol := strings.ToUpper(strings.TrimSpace(defaultString(c.Query("symbol"), "XAUUSD")))
	rows, err := dbstore.ListSimulations(c.Request.Context(), s.db, symbol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if rows == nil {
		rows = []dbstore.SimulationRow{}
	}
	c.JSON(http.StatusOK, rows)
}

func (s *Server) getSimulation(c *gin.Context) {
	if s.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
		return
	}
	id := c.Param("id")
	sim, outcome, err := dbstore.GetSimulation(c.Request.Context(), s.db, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if sim == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "simulation not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"simulation": sim, "outcome": outcome})
}
