package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"trade-buddy/api/internal/stream"
)

func (s *Server) streamCandle(c *gin.Context) {
	if s.hub == nil || s.cfg.FinnhubAPIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "realtime streaming unavailable — set FINNHUB_API_KEY",
		})
		return
	}

	symbol := strings.ToUpper(strings.TrimSpace(defaultString(c.Query("symbol"), "XAUUSD")))
	tf := strings.ToLower(strings.TrimSpace(defaultString(c.Query("tf"), "1m")))

	// Clear write deadline so SSE connection stays open indefinitely
	rc := http.NewResponseController(c.Writer)
	_ = rc.SetWriteDeadline(time.Time{})

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	id := fmt.Sprintf("%s:%s:%d", symbol, tf, time.Now().UnixNano())
	ch := s.hub.Subscribe(id, symbol, tf)
	defer s.hub.Unsubscribe(id)

	// Send a connected event immediately
	fmt.Fprintf(c.Writer, "data: {\"connected\":true,\"symbol\":%q,\"tf\":%q}\n\n", symbol, tf)
	c.Writer.Flush()

	heartbeat := time.NewTicker(10 * time.Second)
	defer heartbeat.Stop()

	ctx := c.Request.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			fmt.Fprintf(c.Writer, ": ping\n\n")
			c.Writer.Flush()
		case event, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(stream.CandleEvent(event))
			if err != nil {
				continue
			}
			fmt.Fprintf(c.Writer, "data: %s\n\n", data)
			c.Writer.Flush()
		}
	}
}
