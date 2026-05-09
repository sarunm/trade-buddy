package httpapi

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type playbookResponse struct {
	Symbol   string         `json:"symbol"`
	MinCount int64          `json:"min_count"`
	Rules    []playbookRule `json:"rules"`
}

type playbookRule struct {
	Key    string        `json:"key"`
	Action string        `json:"action"`
	Delta  float64       `json:"delta"`
	Stats  playbookStats `json:"stats"`
}

type playbookStats struct {
	WinRate   float64  `json:"win_rate"`
	AvgR      *float64 `json:"avg_r"`
	Count     int64    `json:"count"`
	Pattern   string   `json:"pattern"`
	Timeframe string   `json:"timeframe"`
	Session   string   `json:"session"`
}

type playbookRow struct {
	Pattern   sql.NullString  `gorm:"column:pattern"`
	Timeframe sql.NullString  `gorm:"column:timeframe"`
	Session   sql.NullString  `gorm:"column:session"`
	Total     int64           `gorm:"column:total"`
	Wins      int64           `gorm:"column:wins"`
	AvgR      sql.NullFloat64 `gorm:"column:avg_r"`
}

func (s *Server) playbookHandler(c *gin.Context) {
	if s.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
		return
	}

	symbol := strings.ToUpper(strings.TrimSpace(defaultString(c.Query("symbol"), "XAUUSD")))
	minCount, err := parseMinCount(c.Query("min_count"), 5)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "min_count must be an integer"})
		return
	}

	var rows []playbookRow
	if err := s.db.WithContext(c.Request.Context()).Raw(playbookSQL, symbol, minCount).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "load playbook: " + err.Error()})
		return
	}

	rules := make([]playbookRule, len(rows))
	for i, row := range rows {
		pattern := nullStringValue(row.Pattern)
		timeframe := nullStringValue(row.Timeframe)
		session := nullStringValue(row.Session)
		laplaceWinRate := (float64(row.Wins) + 1.0) / (float64(row.Total) + 2.0)
		delta := ((laplaceWinRate - 0.50) * 0.5) + sessionWeight(session)

		rules[i] = playbookRule{
			Key:    fmt.Sprintf("pattern:%s:%s:%s", pattern, timeframe, session),
			Action: playbookAction(laplaceWinRate),
			Delta:  delta,
			Stats: playbookStats{
				WinRate:   laplaceWinRate,
				AvgR:      nullFloat64Ptr(row.AvgR),
				Count:     row.Total,
				Pattern:   pattern,
				Timeframe: timeframe,
				Session:   session,
			},
		}
	}

	c.JSON(http.StatusOK, playbookResponse{
		Symbol:   symbol,
		MinCount: int64(minCount),
		Rules:    rules,
	})
}

const playbookSQL = `
	SELECT
		a.context->>'pattern'   AS pattern,
		a.context->>'timeframe' AS timeframe,
		a.context->>'session'   AS session,
		COUNT(*)                                             AS total,
		SUM(CASE WHEN ao.outcome = 'win' THEN 1 ELSE 0 END) AS wins,
		AVG((ao.details->>'r_multiple')::DOUBLE PRECISION)  AS avg_r
	FROM alerts a
	JOIN alert_outcomes ao ON ao.alert_id = a.id
	WHERE a.symbol = ?
	  AND ao.outcome IN ('win', 'loss')
	  AND a.context->>'pattern' IS NOT NULL
	GROUP BY a.context->>'pattern', a.context->>'timeframe', a.context->>'session'
	HAVING COUNT(*) >= ?
	ORDER BY AVG((ao.details->>'r_multiple')::DOUBLE PRECISION) DESC NULLS LAST
`

func parseMinCount(value string, fallback int) (int, error) {
	if strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	minCount, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	if minCount < 1 {
		return 1, nil
	}
	return minCount, nil
}

func nullStringValue(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func sessionWeight(session string) float64 {
	switch session {
	case "london_ny_overlap":
		return 0.05
	case "asia":
		return -0.10
	case "newyork":
		return 0.02
	case "dead":
		return -0.05
	default:
		return 0
	}
}

func playbookAction(winRate float64) string {
	if winRate >= 0.55 {
		return "boost"
	}
	if winRate <= 0.40 {
		return "avoid"
	}
	return "neutral"
}
