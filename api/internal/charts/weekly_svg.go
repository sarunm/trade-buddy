package charts

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"

	"trade-buddy/api/internal/analysis"
	"trade-buddy/api/internal/forecast"
	"trade-buddy/api/internal/marketdata"
)

type WeeklyPlanSVGInput struct {
	Symbol    string
	Source    string
	Timeframe string
	Candles   []marketdata.Candle
	Levels    []analysis.Level
	Paths     []forecast.ForecastPath
	Bias      forecast.BiasResult
}

type CachedSVG struct {
	Path     string
	URL      string
	Cached   bool
	Filename string
}

const weeklySVGRendererVersion = 2

func CachedWeeklyPlanSVG(dataDir string, input WeeklyPlanSVGInput, reset bool) (CachedSVG, error) {
	cacheDir := filepath.Join(dataDir, "weekly-plan-maps")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return CachedSVG{}, fmt.Errorf("create weekly plan cache dir: %w", err)
	}

	filename := fmt.Sprintf("%s-%s-%s-%s.svg", slug(input.Symbol), slug(input.Source), slug(input.Timeframe), WeeklySVGDigest(input))
	path := filepath.Join(cacheDir, filename)
	cached := !reset && fileExists(path)
	if !cached {
		if err := os.WriteFile(path, []byte(RenderWeeklyPlanSVG(input, 1280, 620)), 0o644); err != nil {
			return CachedSVG{}, fmt.Errorf("write weekly plan svg: %w", err)
		}
	}

	info, err := os.Stat(path)
	if err != nil {
		return CachedSVG{}, fmt.Errorf("stat weekly plan svg: %w", err)
	}

	return CachedSVG{
		Path:     path,
		URL:      fmt.Sprintf("/weekly-plan-maps/%s?v=%d", filename, info.ModTime().Unix()),
		Cached:   cached,
		Filename: filename,
	}, nil
}

func WeeklySVGDigest(input WeeklyPlanSVGInput) string {
	candles := lastCandles(input.Candles, 80)
	payload := struct {
		Renderer  int                     `json:"renderer"`
		Symbol    string                  `json:"symbol"`
		Source    string                  `json:"source"`
		Timeframe string                  `json:"timeframe"`
		Candles   []digestCandle          `json:"candles"`
		Levels    []analysis.Level        `json:"levels"`
		Paths     []forecast.ForecastPath `json:"paths"`
		Bias      forecast.BiasResult     `json:"bias"`
	}{
		Renderer:  weeklySVGRendererVersion,
		Symbol:    input.Symbol,
		Source:    input.Source,
		Timeframe: input.Timeframe,
		Levels:    input.Levels,
		Paths:     input.Paths,
		Bias:      input.Bias,
	}
	for _, candle := range candles {
		payload.Candles = append(payload.Candles, digestCandle{
			Time:  candle.Time.UTC().Format(time.RFC3339),
			Open:  round4(candle.Open),
			High:  round4(candle.High),
			Low:   round4(candle.Low),
			Close: round4(candle.Close),
		})
	}

	data, _ := json.Marshal(payload)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:16]
}

func RenderWeeklyPlanSVG(input WeeklyPlanSVGInput, width, height int) string {
	candles := lastCandles(input.Candles, 48)
	if len(candles) == 0 {
		return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1280 620"><text x="40" y="60">No weekly candles</text></svg>`
	}

	low, high := priceRange(candles, input.Levels, input.Paths)
	latest := candles[len(candles)-1]
	pad := math.Max((high-low)*0.12, math.Max(math.Abs(latest.Close)*0.004, 10))
	low -= pad
	high += pad
	span := math.Max(high-low, 0.0001)

	plotLeft, plotTop := 58.0, 58.0
	plotRight, plotBottom := float64(width-430), float64(height-70)
	labelX := float64(width - 188)
	forecastRight := labelX - 22
	plotW, plotH := plotRight-plotLeft, plotBottom-plotTop
	step := plotW / math.Max(float64(len(candles)), 1)
	bodyW := math.Max(3, math.Min(11, step*0.55))

	xFor := func(index int) float64 {
		return plotLeft + float64(index)*step + step/2
	}
	yFor := func(price float64) float64 {
		return plotTop + (high-price)/span*plotH
	}

	labels := layoutPriceLabels(priceLabels(input.Levels, latest.Close, yFor), plotTop, plotBottom, 31)

	parts := []string{
		fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" role="img" aria-label="weekly candlestick plan">`, width, height),
		`<rect width="100%" height="100%" fill="#dbeafe"/>`,
		fmt.Sprintf(`<text x="22" y="30" fill="#0f172a" font-size="22" font-weight="700">%s · Weekly Plan · latest 1W close %.2f</text>`, html.EscapeString(input.Symbol), latest.Close),
		`<text x="22" y="54" fill="#334155" font-size="14">Forecast จาก 1M/1W · เส้นหนา = route หลัก · แผนภาพถูก cache เป็นไฟล์ SVG</text>`,
		fmt.Sprintf(`<clipPath id="weekly-plot-clip"><rect x="%.2f" y="%.2f" width="%.2f" height="%.2f"/></clipPath>`, plotLeft, plotTop, plotW, plotH),
		fmt.Sprintf(`<line x1="%.2f" x2="%.2f" y1="%.2f" y2="%.2f" stroke="#475569" stroke-width="1.4" stroke-dasharray="7 6" opacity=".55"/>`, plotRight, plotRight, plotTop, plotBottom),
	}

	for _, label := range labels {
		if label.Kind == "latest" {
			continue
		}
		parts = append(parts, levelSVG(label, plotLeft, labelX))
	}

	parts = append(parts, `<g clip-path="url(#weekly-plot-clip)">`)
	for i, candle := range candles {
		parts = append(parts, candleSVG(candle, xFor(i), yFor, bodyW))
	}
	parts = append(parts, `</g>`)

	latestY := yFor(latest.Close)
	latestLabelY := latestY
	for _, label := range labels {
		if label.Kind == "latest" {
			latestLabelY = label.LabelY
			break
		}
	}
	parts = append(parts,
		fmt.Sprintf(`<line x1="%.2f" x2="%.2f" y1="%.2f" y2="%.2f" stroke="#0f766e" stroke-width="1.5" stroke-dasharray="4 4"/>`, plotLeft, labelX-8, latestY, latestY),
		connectorSVG(labelX-8, latestY, labelX, latestLabelY, "#0f766e"),
		fmt.Sprintf(`<rect x="%.2f" y="%.2f" width="214" height="28" rx="5" fill="#0f766e" opacity=".92"/>`, labelX, latestLabelY-14),
		fmt.Sprintf(`<text x="%.2f" y="%.2f" fill="#fff" font-size="14" font-weight="700">1W ล่าสุด %.2f</text>`, labelX+9, latestLabelY+4, latest.Close),
	)

	startIndex := maxInt(len(candles)-2, 0)
	for i, path := range input.Paths {
		parts = append(parts, pathSVG(path, i, xFor(startIndex), yFor(candles[startIndex].Close), forecastRight, yFor))
	}

	parts = append(parts, `</svg>`)
	return join(parts)
}

type digestCandle struct {
	Time  string  `json:"time"`
	Open  float64 `json:"open"`
	High  float64 `json:"high"`
	Low   float64 `json:"low"`
	Close float64 `json:"close"`
}

func priceRange(candles []marketdata.Candle, levels []analysis.Level, paths []forecast.ForecastPath) (float64, float64) {
	low, high := candles[0].Low, candles[0].High
	for _, candle := range candles {
		low = math.Min(low, candle.Low)
		high = math.Max(high, candle.High)
	}
	for _, level := range levels {
		low = math.Min(low, level.Price)
		high = math.Max(high, level.Price)
	}
	for _, path := range paths {
		for _, price := range []float64{path.From, path.Via, path.To} {
			if price != 0 {
				low = math.Min(low, price)
				high = math.Max(high, price)
			}
		}
	}
	return low, high
}

type priceLabel struct {
	Label  string
	Price  float64
	Kind   string
	LineY  float64
	LabelY float64
}

func priceLabels(levels []analysis.Level, latestClose float64, yFor func(float64) float64) []priceLabel {
	labels := make([]priceLabel, 0, len(levels)+1)
	for _, level := range levels {
		labels = append(labels, priceLabel{
			Label:  level.Label,
			Price:  level.Price,
			Kind:   level.Kind,
			LineY:  yFor(level.Price),
			LabelY: yFor(level.Price),
		})
	}
	labels = append(labels, priceLabel{
		Label:  "1W ล่าสุด",
		Price:  latestClose,
		Kind:   "latest",
		LineY:  yFor(latestClose),
		LabelY: yFor(latestClose),
	})
	return labels
}

func layoutPriceLabels(labels []priceLabel, top, bottom, minGap float64) []priceLabel {
	if len(labels) == 0 {
		return labels
	}
	out := append([]priceLabel(nil), labels...)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].LineY < out[j].LineY
	})

	minY := top + 14
	maxY := bottom - 14
	for i := range out {
		out[i].LabelY = clamp(out[i].LineY, minY, maxY)
		if i > 0 && out[i].LabelY < out[i-1].LabelY+minGap {
			out[i].LabelY = out[i-1].LabelY + minGap
		}
	}
	overflow := out[len(out)-1].LabelY - maxY
	if overflow > 0 {
		for i := range out {
			out[i].LabelY -= overflow
		}
	}
	for i := len(out) - 2; i >= 0; i-- {
		if out[i].LabelY > out[i+1].LabelY-minGap {
			out[i].LabelY = out[i+1].LabelY - minGap
		}
	}
	for i := range out {
		out[i].LabelY = clamp(out[i].LabelY, minY, maxY)
	}
	return out
}

func levelSVG(label priceLabel, plotLeft, labelX float64) string {
	color := "#dc2626"
	if len(label.Kind) >= 7 && label.Kind[:7] == "support" {
		color = "#2563eb"
	}
	return fmt.Sprintf(`<line x1="%.2f" x2="%.2f" y1="%.2f" y2="%.2f" stroke="%s" stroke-width="2" stroke-dasharray="7 5" opacity=".75"/>%s<rect x="%.2f" y="%.2f" width="164" height="26" rx="4" fill="%s" opacity=".75"/><text x="%.2f" y="%.2f" fill="#fff" font-size="14" font-weight="700">%s %.2f</text>`, plotLeft, labelX-8, label.LineY, label.LineY, color, connectorSVG(labelX-8, label.LineY, labelX, label.LabelY, color), labelX, label.LabelY-14, color, labelX+8, label.LabelY+4, html.EscapeString(label.Label), label.Price)
}

func connectorSVG(x1, y1, x2, y2 float64, color string) string {
	if math.Abs(y1-y2) < 1 {
		return ""
	}
	return fmt.Sprintf(`<path d="M%.2f %.2f L%.2f %.2f" fill="none" stroke="%s" stroke-width="1.2" opacity=".65"/>`, x1, y1, x2, y2, color)
}

func candleSVG(candle marketdata.Candle, x float64, yFor func(float64) float64, bodyW float64) string {
	color := "#0f9f8f"
	if candle.Close < candle.Open {
		color = "#ef4444"
	}
	highY, lowY := yFor(candle.High), yFor(candle.Low)
	openY, closeY := yFor(candle.Open), yFor(candle.Close)
	top := math.Min(openY, closeY)
	bodyH := math.Max(math.Abs(closeY-openY), 1.2)
	return fmt.Sprintf(`<line x1="%.2f" x2="%.2f" y1="%.2f" y2="%.2f" stroke="%s" stroke-width="1.4"/><rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s" rx="1"/>`, x, x, highY, lowY, color, x-bodyW/2, top, bodyW, bodyH, color)
}

func pathSVG(path forecast.ForecastPath, index int, startX, startY, forecastRight float64, yFor func(float64) float64) string {
	points := path.Points
	if len(points) == 0 {
		points = []float64{path.Via, path.To}
	}
	stepX := math.Max(72, (forecastRight-startX)/math.Max(float64(len(points)), 1))
	x1, y1 := startX, startY
	out := ""
	for i, price := range points {
		x2 := math.Min(startX+stepX*float64(i+1), forecastRight)
		if i == len(points)-1 {
			x2 = forecastRight
		}
		y2 := yFor(price)
		color := "#0f9f8f"
		if y2 > y1 {
			color = "#ef4444"
		}
		width := 2.1
		opacity := 0.62
		if path.Priority == "primary" || index == 0 {
			width = 3.4
			opacity = 0.98
		}
		markerID := fmt.Sprintf("arrow-%d-%d", index, i)
		out += fmt.Sprintf(`<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="%s" stroke-width="%.1f" marker-end="url(#%s)" opacity="%.2f"/><marker id="%s" markerWidth="10" markerHeight="10" refX="8" refY="4" orient="auto"><path d="M0,0 L0,8 L9,4 z" fill="%s"/></marker>`, x1, y1, x2, y2, color, width, markerID, opacity, markerID, color)
		x1, y1 = x2, y2
	}
	return out
}

func lastCandles(candles []marketdata.Candle, limit int) []marketdata.Candle {
	if len(candles) <= limit {
		return candles
	}
	return candles[len(candles)-limit:]
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func slug(value string) string {
	out := ""
	for _, ch := range value {
		if ch >= 'A' && ch <= 'Z' {
			ch += 'a' - 'A'
		}
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' {
			out += string(ch)
		}
	}
	if out == "" {
		return "weekly"
	}
	return out
}

func round4(value float64) float64 {
	return math.Round(value*10000) / 10000
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clamp(value, minValue, maxValue float64) float64 {
	return math.Max(minValue, math.Min(maxValue, value))
}

func join(parts []string) string {
	out := ""
	for _, part := range parts {
		out += part
	}
	return out
}
