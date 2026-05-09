package httpapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"trade-buddy/api/internal/config"
	"trade-buddy/api/internal/monitor"
	"trade-buddy/api/internal/stream"
)

type Dependencies struct {
	Config  config.Config
	DB      *gorm.DB
	Hub     *stream.Hub
	Monitor *monitor.Monitor
}

type Server struct {
	cfg     config.Config
	db      *gorm.DB
	hub     *stream.Hub
	monitor *monitor.Monitor
}

func NewRouter(deps Dependencies) *gin.Engine {
	server := &Server{cfg: deps.Config, db: deps.DB, hub: deps.Hub, monitor: deps.Monitor}

	r := gin.Default()
	r.Use(withCommonHeaders())
	r.GET("/health", server.health)
	r.GET("/api/monitor/health", server.monitorHealth)
	r.GET("/api/chart", server.chart)
	r.GET("/api/weekly-plan", server.weeklyPlan)
	r.POST("/api/weekly-plan/reset", server.resetWeeklyPlan)
	r.GET("/api/alerts", server.listAlerts)
	r.GET("/api/alerts/:id", server.getAlert)
	r.POST("/api/alerts", server.createAlert)
	r.POST("/api/alerts/resolve", server.resolveAlert)
	r.GET("/api/playbook", server.playbookHandler)
	r.GET("/api/journal/stats", server.journalStats)
	r.GET("/api/top-down/context", server.topDownContext)
	r.GET("/api/top-down/context/history", server.topDownContextHistory)
	r.POST("/api/simulations", server.createSimulation)
	r.GET("/api/simulations", server.listSimulations)
	r.GET("/api/simulations/:id", server.getSimulation)
	r.GET("/api/stream", server.streamCandle)
	r.GET("/weekly-plan-maps/:filename", server.weeklyPlanMap)

	return r
}

func withCommonHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if isAllowedOrigin(origin) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Accept,Authorization,Content-Type")
		}
		c.Header("X-Content-Type-Options", "nosniff")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func isAllowedOrigin(origin string) bool {
	if origin == "" {
		return false
	}
	return strings.HasPrefix(origin, "http://localhost:") ||
		strings.HasPrefix(origin, "http://127.0.0.1:")
}
