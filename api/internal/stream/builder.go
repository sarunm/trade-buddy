package stream

import (
	"sync"
	"time"
)

// Tick is a single price event from a data source.
type Tick struct {
	Symbol string
	Price  float64
	Volume float64
	Time   time.Time
}

type candleState struct {
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
	Period time.Time
}

// Builder aggregates ticks into candles for all supported timeframes.
type Builder struct {
	mu     sync.Mutex
	states map[string]*candleState // key = "symbol:tf"
	hub    *Hub
}

var supportedTFs = []string{"1m", "5m", "15m", "30m", "1h"}

func NewBuilder(hub *Hub) *Builder {
	return &Builder{
		states: make(map[string]*candleState),
		hub:    hub,
	}
}

// ProcessTick updates candles for all supported timeframes and broadcasts.
func (b *Builder) ProcessTick(tick Tick) {
	for _, tf := range supportedTFs {
		b.processTF(tick, tf)
	}
}

func (b *Builder) processTF(tick Tick, tf string) {
	period := periodStart(tick.Time, tf)
	key := tick.Symbol + ":" + tf

	b.mu.Lock()
	defer b.mu.Unlock()

	state, exists := b.states[key]

	if !exists || !state.Period.Equal(period) {
		// Emit closed candle before starting a new one
		if exists {
			b.hub.Broadcast(CandleEvent{
				Symbol:    tick.Symbol,
				Timeframe: tf,
				Time:      state.Period.Unix(),
				Open:      state.Open,
				High:      state.High,
				Low:       state.Low,
				Close:     state.Close,
				Volume:    state.Volume,
				Closed:    true,
			})
		}
		b.states[key] = &candleState{
			Open:   tick.Price,
			High:   tick.Price,
			Low:    tick.Price,
			Close:  tick.Price,
			Volume: tick.Volume,
			Period: period,
		}
		state = b.states[key]
	} else {
		if tick.Price > state.High {
			state.High = tick.Price
		}
		if tick.Price < state.Low {
			state.Low = tick.Price
		}
		state.Close = tick.Price
		state.Volume += tick.Volume
	}

	// Broadcast live (in-progress) candle update
	b.hub.Broadcast(CandleEvent{
		Symbol:    tick.Symbol,
		Timeframe: tf,
		Time:      state.Period.Unix(),
		Open:      state.Open,
		High:      state.High,
		Low:       state.Low,
		Close:     state.Close,
		Volume:    state.Volume,
		Closed:    false,
	})
}

func periodStart(t time.Time, tf string) time.Time {
	t = t.UTC()
	switch tf {
	case "1m":
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, time.UTC)
	case "5m":
		m := (t.Minute() / 5) * 5
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), m, 0, 0, time.UTC)
	case "15m":
		m := (t.Minute() / 15) * 15
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), m, 0, 0, time.UTC)
	case "30m":
		m := (t.Minute() / 30) * 30
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), m, 0, 0, time.UTC)
	case "1h":
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, time.UTC)
	default:
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, time.UTC)
	}
}
