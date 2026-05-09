package monitor

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNoopNotifierReturnsNil(t *testing.T) {
	if err := (NoopNotifier{}).Notify(context.Background(), "ignored"); err != nil {
		t.Fatalf("NoopNotifier.Notify returned error: %v", err)
	}
}

func TestLineNotifierSendsCorrectPayload(t *testing.T) {
	const msg = "new XAUUSD signal"
	requests := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Fatalf("Authorization = %q, want Bearer tok", got)
		}
		if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
			t.Fatalf("Content-Type = %q, want application/json", got)
		}

		var body linePushRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.To != "uid" {
			t.Fatalf("to = %q, want uid", body.To)
		}
		if len(body.Messages) != 1 {
			t.Fatalf("messages len = %d, want 1", len(body.Messages))
		}
		if body.Messages[0].Type != "text" {
			t.Fatalf("message type = %q, want text", body.Messages[0].Type)
		}
		if !strings.Contains(body.Messages[0].Text, msg) {
			t.Fatalf("message text = %q, want it to contain %q", body.Messages[0].Text, msg)
		}
		w.WriteHeader(http.StatusOK)
	})

	oldURL := linePushURL
	oldClient := lineHTTPClient
	url, client, cleanup := lineTestEndpoint(handler)
	linePushURL = url
	lineHTTPClient = client
	defer func() { linePushURL = oldURL }()
	defer func() { lineHTTPClient = oldClient }()
	defer cleanup()

	notifier := &LineNotifier{token: "tok", toID: "uid"}
	if err := notifier.Notify(context.Background(), msg); err != nil {
		t.Fatalf("LineNotifier.Notify returned error: %v", err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestNewNotifierReturnsNoop(t *testing.T) {
	if _, ok := NewNotifier("", "").(NoopNotifier); !ok {
		t.Fatal("NewNotifier with empty values should return NoopNotifier")
	}
}

func TestNewNotifierReturnsLine(t *testing.T) {
	if _, ok := NewNotifier("tok", "uid").(*LineNotifier); !ok {
		t.Fatal("NewNotifier with token and toID should return *LineNotifier")
	}
}

func lineTestEndpoint(handler http.Handler) (string, *http.Client, func()) {
	server, ok := tryLineTestServer(handler)
	if ok {
		return server.URL, server.Client(), server.Close
	}
	return "http://line.test/push", &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, r)
		return recorder.Result(), nil
	})}, func() {}
}

func tryLineTestServer(handler http.Handler) (server *httptest.Server, ok bool) {
	defer func() {
		if recover() != nil {
			server = nil
			ok = false
		}
	}()
	server = httptest.NewServer(handler)
	return server, true
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := f(r)
	if resp != nil && resp.Body == nil {
		resp.Body = io.NopCloser(strings.NewReader(""))
	}
	return resp, err
}
