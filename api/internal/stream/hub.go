package stream

import "sync"

// CandleEvent is broadcast to SSE subscribers on every tick.
type CandleEvent struct {
	Symbol    string  `json:"symbol"`
	Timeframe string  `json:"timeframe"`
	Time      int64   `json:"time"` // unix seconds
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Volume    float64 `json:"volume"`
	Closed    bool    `json:"closed"` // true = candle period ended
}

type subscriber struct {
	ch     chan CandleEvent
	symbol string
	tf     string
}

// Hub manages SSE subscribers and broadcasts candle events.
type Hub struct {
	mu   sync.RWMutex
	subs map[string]*subscriber
}

func NewHub() *Hub {
	return &Hub{subs: make(map[string]*subscriber)}
}

// Subscribe registers a new SSE client and returns its event channel.
func (h *Hub) Subscribe(id, symbol, tf string) <-chan CandleEvent {
	ch := make(chan CandleEvent, 16)
	h.mu.Lock()
	h.subs[id] = &subscriber{ch: ch, symbol: symbol, tf: tf}
	h.mu.Unlock()
	return ch
}

// Unsubscribe removes the client and closes its channel.
func (h *Hub) Unsubscribe(id string) {
	h.mu.Lock()
	if s, ok := h.subs[id]; ok {
		close(s.ch)
		delete(h.subs, id)
	}
	h.mu.Unlock()
}

// Broadcast sends an event to all matching subscribers (non-blocking).
func (h *Hub) Broadcast(event CandleEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, s := range h.subs {
		if s.symbol == event.Symbol && s.tf == event.Timeframe {
			select {
			case s.ch <- event:
			default: // drop if subscriber is slow
			}
		}
	}
}
