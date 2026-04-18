package webhook_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"relay/internal/store"
	"relay/internal/webhook"
)

func TestHandler_Create_ValidRequest_Returns202AndEnqueuesEvent(t *testing.T) {
	t.Parallel()

	st := store.NewMemory()
	h := webhook.NewHandler(st)

	body := map[string]any{
		"target_url": "https://example.com/hooks/1",
		"event_type": "payment.completed",
		"payload":    map[string]any{"amount": 42},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusAccepted, rec.Body.String())
	}

	var got struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v body=%q", err, rec.Body.String())
	}
	if got.ID == "" {
		t.Fatal("expected non-empty id")
	}
	if got.Status != "accepted" {
		t.Fatalf("status field = %q, want accepted", got.Status)
	}

	ev, status, ok := st.Get(got.ID)
	if !ok {
		t.Fatal("expected event in store")
	}
	if status != "pending" {
		t.Fatalf("status = %q, want pending", status)
	}
	if ev.ID != got.ID {
		t.Fatalf("event id = %q, want %q", ev.ID, got.ID)
	}
	if ev.TargetURL != "https://example.com/hooks/1" {
		t.Fatalf("target_url = %q", ev.TargetURL)
	}
	if ev.EventType != "payment.completed" {
		t.Fatalf("event_type = %q", ev.EventType)
	}
}

func TestHandler_Create_InvalidJSON_Returns400(t *testing.T) {
	t.Parallel()

	st := store.NewMemory()
	h := webhook.NewHandler(st)

	req := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewReader([]byte(`{`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestHandler_Create_MissingTargetURL_Returns400(t *testing.T) {
	t.Parallel()

	st := store.NewMemory()
	h := webhook.NewHandler(st)

	body := map[string]any{
		"event_type": "x",
		"payload":    map[string]any{},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestHandler_Create_InvalidTargetURL_Returns400(t *testing.T) {
	t.Parallel()

	st := store.NewMemory()
	h := webhook.NewHandler(st)

	body := map[string]any{
		"target_url": "not-a-valid-url",
		"event_type": "x",
		"payload":    map[string]any{},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
