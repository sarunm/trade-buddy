package httpapi

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"trade-buddy/api/internal/analysis"
	"trade-buddy/api/internal/charts"
	dbstore "trade-buddy/api/internal/db"
	"trade-buddy/api/internal/forecast"
	"trade-buddy/api/internal/marketdata"
)

type weeklyPlanResponse struct {
	Symbol        string                  `json:"symbol"`
	Source        string                  `json:"source"`
	UpdatedAt     time.Time               `json:"updated_at"`
	ForecastBias  forecast.BiasResult     `json:"forecast_bias"`
	Close         float64                 `json:"close"`
	Levels        []weeklyLevelResponse   `json:"levels"`
	Paths         []forecast.ForecastPath `json:"paths"`
	SwingTrade    forecast.SwingTrade     `json:"swing_trade"`
	ImageURL      string                  `json:"image_url"`
	ImageCached   bool                    `json:"image_cached"`
	WeeklyCandles int                     `json:"weekly_candles"`
	MonthlyTrend  analysis.Direction      `json:"monthly_trend"`
	WeeklyTrend   analysis.Direction      `json:"weekly_trend"`
}

type weeklyLevelResponse struct {
	Label string  `json:"label"`
	Price float64 `json:"price"`
	Kind  string  `json:"kind"`
}

func (s *Server) weeklyPlan(c *gin.Context) {
	symbol := strings.ToUpper(strings.TrimSpace(defaultString(c.Query("symbol"), "XAUUSD")))
	source := strings.ToLower(strings.TrimSpace(defaultString(c.Query("source"), s.defaultSource())))
	if _, err := s.marketDataSource(source); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	weekly, monthly, err := s.loadWeeklyPlanCandles(c, symbol, source)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	if len(weekly) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "weekly candles unavailable"})
		return
	}

	close := weekly[len(weekly)-1].Close
	weeklyTrend := analysis.DetectTrend(weekly)
	monthlyTrend := analysis.DetectTrend(monthly)
	bias := forecast.ForecastBias(monthlyTrend, weeklyTrend)
	atrSeries := analysis.ATR(weekly, 14)
	atr := lastNonZero(atrSeries)
	supports, resistances := analysis.WeeklyLevels(weekly, close)
	levels := append([]analysis.Level{}, resistances...)
	levels = append(levels, supports...)
	paths := forecast.WeeklyRoutes(bias.Direction, close, supports, resistances, atr)
	swing := forecast.SwingTradeBias(bias, forecast.TrendContext{}, forecast.TrendContext{}, supports, resistances, atr)

	image, err := charts.CachedWeeklyPlanSVG(s.cfg.DataDir, charts.WeeklyPlanSVGInput{
		Symbol:    symbol,
		Source:    source,
		Timeframe: "1w",
		Candles:   weekly,
		Levels:    levels,
		Paths:     paths,
		Bias:      bias,
	}, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, weeklyPlanResponse{
		Symbol:        symbol,
		Source:        source,
		UpdatedAt:     time.Now().UTC(),
		ForecastBias:  bias,
		Close:         roundFloat(close),
		Levels:        weeklyLevels(levels),
		Paths:         paths,
		SwingTrade:    swing,
		ImageURL:      image.URL,
		ImageCached:   image.Cached,
		WeeklyCandles: len(weekly),
		MonthlyTrend:  monthlyTrend,
		WeeklyTrend:   weeklyTrend,
	})
}

func (s *Server) resetWeeklyPlan(c *gin.Context) {
	symbol := strings.ToLower(strings.TrimSpace(defaultString(c.Query("symbol"), "XAUUSD")))
	pattern := filepath.Join(s.cfg.DataDir, "weekly-plan-maps", slugForGlob(symbol)+"-*.svg")
	files, err := filepath.Glob(pattern)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	deleted := 0
	for _, file := range files {
		if err := os.Remove(file); err == nil {
			deleted++
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "deleted": deleted})
}

func (s *Server) weeklyPlanMap(c *gin.Context) {
	filename := filepath.Base(c.Param("filename"))
	c.File(filepath.Join(s.cfg.DataDir, "weekly-plan-maps", filename))
}

func (s *Server) loadWeeklyPlanCandles(c *gin.Context, symbol, source string) ([]marketdata.Candle, []marketdata.Candle, error) {
	var weekly, monthly []marketdata.Candle
	var weeklyErr, monthlyErr error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		weekly, weeklyErr = s.loadCandles(c, symbol, "1w", source, 200)
	}()
	go func() {
		defer wg.Done()
		monthly, monthlyErr = s.loadCandles(c, symbol, "1mo", source, 80)
	}()
	wg.Wait()
	if weeklyErr != nil {
		return nil, nil, weeklyErr
	}
	if monthlyErr != nil {
		return nil, nil, monthlyErr
	}
	return weekly, monthly, nil
}

func (s *Server) loadCandles(c *gin.Context, symbol, timeframe, source string, limit int) ([]marketdata.Candle, error) {
	ctx := c.Request.Context()
	candles, err := dbstore.FetchBars(ctx, s.db, symbol, timeframe, source, limit)
	if err != nil {
		return nil, err
	}
	if len(candles) >= limit {
		return candles, nil
	}
	fetched, effectiveSource, err := s.loadMarketCandles(ctx, symbol, timeframe, source, limit)
	if err != nil {
		return nil, err
	}
	if err := dbstore.UpsertBars(ctx, s.db, symbol, timeframe, effectiveSource, fetched); err != nil {
		return nil, err
	}
	return dbstore.FetchBars(ctx, s.db, symbol, timeframe, effectiveSource, limit)
}

func weeklyLevels(levels []analysis.Level) []weeklyLevelResponse {
	out := make([]weeklyLevelResponse, len(levels))
	for i, level := range levels {
		out[i] = weeklyLevelResponse{Label: level.Label, Price: level.Price, Kind: level.Kind}
	}
	return out
}

func lastNonZero(values []float64) float64 {
	for i := len(values) - 1; i >= 0; i-- {
		if values[i] != 0 {
			return values[i]
		}
	}
	return 1
}

func roundFloat(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}

func slugForGlob(value string) string {
	out := ""
	for _, ch := range value {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' {
			out += string(ch)
		}
	}
	return out
}
