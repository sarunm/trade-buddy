package httpapi

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

type journalStatsResponse struct {
	TotalOutcomes int64                `json:"total_outcomes"`
	WinRate       *float64             `json:"win_rate"`
	AvgRR         *float64             `json:"avg_rr"`
	ByPattern     []journalStatsBucket `json:"by_pattern"`
	ByTimeframe   []journalStatsBucket `json:"by_timeframe"`
}

type journalStatsBucket struct {
	Key           string   `json:"key"`
	Pattern       string   `json:"pattern,omitempty"`
	Timeframe     string   `json:"timeframe,omitempty"`
	TotalOutcomes int64    `json:"total_outcomes"`
	WinRate       *float64 `json:"win_rate"`
	AvgRR         *float64 `json:"avg_rr"`
}

type journalStatsRow struct {
	Key           string          `gorm:"column:key"`
	TotalOutcomes int64           `gorm:"column:total_outcomes"`
	WinRate       sql.NullFloat64 `gorm:"column:win_rate"`
	AvgRR         sql.NullFloat64 `gorm:"column:avg_rr"`
}

func (s *Server) journalStats(c *gin.Context) {
	if s.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
		return
	}

	overall, err := s.loadJournalStats(c, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "load journal stats: " + err.Error()})
		return
	}
	byPattern, err := s.loadJournalStats(c, "pattern")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "load pattern stats: " + err.Error()})
		return
	}
	byTimeframe, err := s.loadJournalStats(c, "timeframe")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "load timeframe stats: " + err.Error()})
		return
	}

	var total journalStatsRow
	if len(overall) > 0 {
		total = overall[0]
	}

	c.JSON(http.StatusOK, journalStatsResponse{
		TotalOutcomes: total.TotalOutcomes,
		WinRate:       nullFloat64Ptr(total.WinRate),
		AvgRR:         nullFloat64Ptr(total.AvgRR),
		ByPattern:     statsBuckets(byPattern, "pattern"),
		ByTimeframe:   statsBuckets(byTimeframe, "timeframe"),
	})
}

func (s *Server) loadJournalStats(c *gin.Context, group string) ([]journalStatsRow, error) {
	query := journalStatsSQL("")
	if group == "pattern" {
		query = journalStatsSQL("COALESCE(NULLIF(a.context->>'pattern', ''), 'unknown')")
	}
	if group == "timeframe" {
		query = journalStatsSQL("a.timeframe")
	}

	var rows []journalStatsRow
	err := s.db.WithContext(c.Request.Context()).Raw(query).Scan(&rows).Error
	return rows, err
}

func journalStatsSQL(groupExpr string) string {
	selectKey := "'all' AS key,"
	groupClause := ""
	orderClause := ""
	if groupExpr != "" {
		selectKey = groupExpr + " AS key,"
		groupClause = "GROUP BY key"
		orderClause = "ORDER BY total_outcomes DESC, key ASC"
	}

	return `
		SELECT
			` + selectKey + `
			COUNT(*) AS total_outcomes,
			CASE WHEN COUNT(*) = 0 THEN NULL
				ELSE SUM(CASE WHEN o.outcome = 'win' THEN 1 ELSE 0 END)::double precision / COUNT(*)
			END AS win_rate,
			AVG(
				CASE
					WHEN o.outcome = 'win' THEN COALESCE(a.risk_reward, 0)
					WHEN o.outcome = 'loss' THEN -1
					WHEN o.outcome IN ('timeout', 'ambiguous') THEN 0
					ELSE NULL
				END
			) AS avg_rr
		FROM alert_outcomes o
		JOIN alerts a ON a.id = o.alert_id
		` + groupClause + `
		` + orderClause
}

func statsBuckets(rows []journalStatsRow, kind string) []journalStatsBucket {
	buckets := make([]journalStatsBucket, len(rows))
	for i, row := range rows {
		bucket := journalStatsBucket{
			Key:           row.Key,
			TotalOutcomes: row.TotalOutcomes,
			WinRate:       nullFloat64Ptr(row.WinRate),
			AvgRR:         nullFloat64Ptr(row.AvgRR),
		}
		if kind == "pattern" {
			bucket.Pattern = row.Key
		}
		if kind == "timeframe" {
			bucket.Timeframe = row.Key
		}
		buckets[i] = bucket
	}
	return buckets
}

func nullFloat64Ptr(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	return &value.Float64
}
