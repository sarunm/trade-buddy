package httpapi

import (
	"github.com/gin-gonic/gin"
)

// monitorHealth returns monitor runtime status
// GET /api/monitor/health
func (s *Server) monitorHealth(c *gin.Context) {
	if s.monitor == nil {
		c.JSON(200, gin.H{"running": false, "last_tick_at": nil, "tick_count": 0, "open_alerts": 0})
		return
	}
	stats := s.monitor.Stats()

	var openAlerts int64
	s.db.Raw(`SELECT COUNT(*) FROM alerts WHERE id NOT IN (SELECT alert_id FROM alert_outcomes)`).Scan(&openAlerts)

	c.JSON(200, gin.H{
		"running":      true,
		"last_tick_at": stats.LastTickAt,
		"tick_count":   stats.TickCount,
		"open_alerts":  openAlerts,
	})
}
