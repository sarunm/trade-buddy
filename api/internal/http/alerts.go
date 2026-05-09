package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dbstore "trade-buddy/api/internal/db"
)

type alertResponse struct {
	ID         string          `json:"id"`
	Symbol     string          `json:"symbol"`
	Timeframe  string          `json:"timeframe"`
	Direction  string          `json:"direction"`
	Entry      *float64        `json:"entry"`
	StopLoss   *float64        `json:"stop_loss"`
	TakeProfit *float64        `json:"take_profit"`
	RiskReward *float64        `json:"risk_reward"`
	Confidence *float64        `json:"confidence"`
	Reason     json.RawMessage `json:"reason"`
	Context    json.RawMessage `json:"context"`
	Status     string          `json:"status"`
	CreatedAt  time.Time       `json:"created_at"`
	ResolvedAt *time.Time      `json:"resolved_at"`
}

type createAlertRequest struct {
	Symbol     string          `json:"symbol"`
	Timeframe  string          `json:"timeframe"`
	Direction  string          `json:"direction"`
	Entry      *float64        `json:"entry"`
	StopLoss   *float64        `json:"stop_loss"`
	TakeProfit *float64        `json:"take_profit"`
	RiskReward *float64        `json:"risk_reward"`
	Confidence *float64        `json:"confidence"`
	Reason     json.RawMessage `json:"reason"`
	Context    json.RawMessage `json:"context"`
}

type resolveAlertRequest struct {
	ID            string          `json:"id"`
	Outcome       string          `json:"outcome"`
	ResolvedPrice *float64        `json:"resolved_price"`
	BarsElapsed   *int            `json:"bars_elapsed"`
	MFE           *float64        `json:"mfe"`
	MAE           *float64        `json:"mae"`
	Details       json.RawMessage `json:"details"`
}

func (s *Server) listAlerts(c *gin.Context) {
	if s.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
		return
	}

	symbol := strings.ToUpper(strings.TrimSpace(defaultString(c.Query("symbol"), "XAUUSD")))
	alerts, err := dbstore.ListAlerts(c.Request.Context(), s.db, symbol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list alerts: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol": symbol,
		"alerts": alertResponses(alerts),
	})
}

func (s *Server) getAlert(c *gin.Context) {
	if s.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
		return
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert id"})
		return
	}

	alert, err := dbstore.GetAlert(c.Request.Context(), s.db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get alert: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, alertToResponse(alert))
}

func (s *Server) createAlert(c *gin.Context) {
	if s.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
		return
	}

	var req createAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}

	alert, err := req.toAlert()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	alert.ID = uuid.New()

	if err := dbstore.CreateAlert(c.Request.Context(), s.db, alert); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create alert: " + err.Error()})
		return
	}

	created, err := dbstore.GetAlert(c.Request.Context(), s.db, alert.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "load created alert: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, alertToResponse(created))
}

func (s *Server) resolveAlert(c *gin.Context) {
	if s.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
		return
	}

	var req resolveAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}

	id, outcome, err := req.toOutcome()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if _, err := dbstore.GetAlert(c.Request.Context(), s.db, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get alert: " + err.Error()})
		return
	}

	if err := dbstore.ResolveAlert(c.Request.Context(), s.db, id, outcome); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "resolve alert: " + err.Error()})
		return
	}

	alert, err := dbstore.GetAlert(c.Request.Context(), s.db, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "load resolved alert: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, alertToResponse(alert))
}

func (req createAlertRequest) toAlert() (dbstore.Alert, error) {
	symbol := strings.ToUpper(strings.TrimSpace(req.Symbol))
	if symbol == "" {
		symbol = "XAUUSD"
	}
	timeframe := strings.ToLower(strings.TrimSpace(req.Timeframe))
	if timeframe == "" {
		return dbstore.Alert{}, errors.New("timeframe is required")
	}
	direction := strings.ToLower(strings.TrimSpace(req.Direction))
	if direction == "" {
		return dbstore.Alert{}, errors.New("direction is required")
	}
	if !validDirection(direction) {
		return dbstore.Alert{}, errors.New("direction must be long, short, or neutral")
	}
	if !validJSON(req.Reason) {
		return dbstore.Alert{}, errors.New("reason must be valid JSON")
	}
	if !validJSON(req.Context) {
		return dbstore.Alert{}, errors.New("context must be valid JSON")
	}

	return dbstore.Alert{
		Symbol:     symbol,
		Timeframe:  timeframe,
		Direction:  direction,
		Entry:      req.Entry,
		StopLoss:   req.StopLoss,
		TakeProfit: req.TakeProfit,
		RiskReward: req.RiskReward,
		Confidence: req.Confidence,
		Reason:     req.Reason,
		Context:    req.Context,
	}, nil
}

func (req resolveAlertRequest) toOutcome() (uuid.UUID, dbstore.AlertOutcome, error) {
	id, err := uuid.Parse(strings.TrimSpace(req.ID))
	if err != nil {
		return uuid.Nil, dbstore.AlertOutcome{}, errors.New("invalid alert id")
	}
	outcome := strings.ToLower(strings.TrimSpace(req.Outcome))
	if outcome == "" {
		return uuid.Nil, dbstore.AlertOutcome{}, errors.New("outcome is required")
	}
	if !validOutcome(outcome) {
		return uuid.Nil, dbstore.AlertOutcome{}, errors.New("outcome must be win, loss, timeout, or ambiguous")
	}
	if !validJSON(req.Details) {
		return uuid.Nil, dbstore.AlertOutcome{}, errors.New("details must be valid JSON")
	}

	return id, dbstore.AlertOutcome{
		AlertID:       id,
		Outcome:       outcome,
		ResolvedPrice: req.ResolvedPrice,
		BarsElapsed:   req.BarsElapsed,
		MFE:           req.MFE,
		MAE:           req.MAE,
		Details:       req.Details,
	}, nil
}

func alertResponses(alerts []dbstore.Alert) []alertResponse {
	out := make([]alertResponse, len(alerts))
	for i, alert := range alerts {
		out[i] = alertToResponse(alert)
	}
	return out
}

func alertToResponse(alert dbstore.Alert) alertResponse {
	return alertResponse{
		ID:         alert.ID.String(),
		Symbol:     alert.Symbol,
		Timeframe:  alert.Timeframe,
		Direction:  alert.Direction,
		Entry:      alert.Entry,
		StopLoss:   alert.StopLoss,
		TakeProfit: alert.TakeProfit,
		RiskReward: alert.RiskReward,
		Confidence: alert.Confidence,
		Reason:     defaultJSON(alert.Reason, json.RawMessage("[]")),
		Context:    defaultJSON(alert.Context, json.RawMessage("{}")),
		Status:     alert.Status,
		CreatedAt:  alert.CreatedAt,
		ResolvedAt: alert.ResolvedAt,
	}
}

func validDirection(direction string) bool {
	switch direction {
	case "long", "short", "neutral":
		return true
	default:
		return false
	}
}

func validOutcome(outcome string) bool {
	switch outcome {
	case "win", "loss", "timeout", "ambiguous":
		return true
	default:
		return false
	}
}

func validJSON(raw json.RawMessage) bool {
	return len(raw) == 0 || json.Valid(raw)
}

func defaultJSON(raw, fallback json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return fallback
	}
	return raw
}
