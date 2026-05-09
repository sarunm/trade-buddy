package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

const finnhubWSS = "wss://ws.finnhub.io"

// symbolMap maps trade-buddy symbols → Finnhub symbols.
var symbolMap = map[string]string{
	"XAUUSD": "OANDA:XAU_USD",
}

type finnhubMsg struct {
	Type string         `json:"type"`
	Data []finnhubTrade `json:"data"`
}

type finnhubTrade struct {
	Symbol string  `json:"s"`
	Price  float64 `json:"p"`
	Time   int64   `json:"t"` // milliseconds UTC
	Volume float64 `json:"v"`
}

// RunFinnhub connects to Finnhub WebSocket and feeds ticks to builder.
// Reconnects automatically until ctx is cancelled.
func RunFinnhub(ctx context.Context, apiKey string, builder *Builder) {
	for {
		if err := connectAndConsume(ctx, apiKey, builder); err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Warn("finnhub disconnected, reconnecting in 5s", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}

func connectAndConsume(ctx context.Context, apiKey string, builder *Builder) error {
	u, _ := url.Parse(finnhubWSS)
	q := u.Query()
	q.Set("token", apiKey)
	u.RawQuery = q.Encode()

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		return fmt.Errorf("dial finnhub: %w", err)
	}
	defer conn.Close()

	// Subscribe to all symbols
	for _, fsym := range symbolMap {
		msg, _ := json.Marshal(map[string]string{"type": "subscribe", "symbol": fsym})
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return fmt.Errorf("subscribe %s: %w", fsym, err)
		}
		slog.Info("finnhub: subscribed", "symbol", fsym)
	}

	// Reverse map: finnhub symbol → our symbol
	reverse := make(map[string]string, len(symbolMap))
	for ours, theirs := range symbolMap {
		reverse[theirs] = ours
	}

	for {
		if ctx.Err() != nil {
			return nil
		}

		_, raw, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}

		var msg finnhubMsg
		if err := json.Unmarshal(raw, &msg); err != nil || msg.Type != "trade" {
			continue
		}

		for _, trade := range msg.Data {
			ourSym, ok := reverse[trade.Symbol]
			if !ok {
				continue
			}
			builder.ProcessTick(Tick{
				Symbol: ourSym,
				Price:  trade.Price,
				Volume: trade.Volume,
				Time:   time.UnixMilli(trade.Time).UTC(),
			})
		}
	}
}
