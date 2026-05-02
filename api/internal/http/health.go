package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type healthResponse struct {
	OK      bool   `json:"ok"`
	Service string `json:"service"`
	DB      string `json:"db,omitempty"`
	Error   string `json:"error,omitempty"`
}

func (s *Server) health(c *gin.Context) {
	if s.db == nil {
		c.JSON(http.StatusOK, healthResponse{OK: true, Service: "trade-buddy-api"})
		return
	}

	sqlDB, err := s.db.DB()
	if err == nil {
		err = sqlDB.Ping()
	}

	if err != nil {
		c.JSON(http.StatusServiceUnavailable, healthResponse{
			OK:      false,
			Service: "trade-buddy-api",
			DB:      "error",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, healthResponse{
		OK:      true,
		Service: "trade-buddy-api",
		DB:      "ok",
	})
}
