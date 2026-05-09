package monitor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const linePushEndpoint = "https://api.line.me/v2/bot/message/push"

var linePushURL = linePushEndpoint
var lineHTTPClient = &http.Client{Timeout: 10 * time.Second}

type Notifier interface {
	Notify(ctx context.Context, msg string) error
}

// LineNotifier sends LINE Push API messages.
type LineNotifier struct {
	token string
	toID  string
}

// NoopNotifier silently discards all notifications.
type NoopNotifier struct{}

func NewNotifier(token, toID string) Notifier {
	token = strings.TrimSpace(token)
	toID = strings.TrimSpace(toID)
	if token == "" || toID == "" {
		return NoopNotifier{}
	}
	return &LineNotifier{token: token, toID: toID}
}

func (n *LineNotifier) Notify(ctx context.Context, msg string) error {
	if n == nil {
		return nil
	}
	text := msg
	if len(text) > 5000 {
		text = text[:5000]
	}
	body := linePushRequest{
		To: n.toID,
		Messages: []linePushMessage{
			{Type: "text", Text: text},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal line push body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, linePushURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create line push request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+n.token)
	req.Header.Set("Content-Type", "application/json")

	client := lineHTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send line push: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("line push status: %s", resp.Status)
	}
	return nil
}

func (n NoopNotifier) Notify(ctx context.Context, msg string) error {
	return nil
}

type linePushRequest struct {
	To       string            `json:"to"`
	Messages []linePushMessage `json:"messages"`
}

type linePushMessage struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
