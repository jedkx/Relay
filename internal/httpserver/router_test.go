package httpserver_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"relay/internal/httpserver"
	"relay/internal/queue"
	"relay/internal/webhook"
)

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()

	q := queue.NewMemoryQueue(10)
	wh := webhook.NewHandler(q)
	return httpserver.NewRouter(wh)
}

func TestRouter_Health_Returns200(t *testing.T) {
	t.Parallel()

	h := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}
	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(body)); got != "ok" {
		t.Fatalf("body = %q, want ok", got)
	}
}

func TestRouter_WebhooksPOST_DelegatesToHandler(t *testing.T) {
	t.Parallel()

	h := newTestRouter(t)

	payload := map[string]any{
		"target_url": "https://example.com/hooks/1",
		"event_type": "test.event",
		"payload":    map[string]any{"k": 1},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusAccepted, rec.Body.String())
	}

	var got struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID == "" || got.Status != "accepted" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestRouter_WebhooksGET_Returns405(t *testing.T) {
	t.Parallel()

	h := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/webhooks", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusMethodNotAllowed, rec.Body.String())
	}
}

func TestRouter_UnknownPath_Returns404(t *testing.T) {
	t.Parallel()

	h := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}
