package webhook

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"relay/internal/queue"
)

type Event struct {
	ID        string         `json:"id"`
	TargetURL string         `json:"target_url"`
	EventType string         `json:"event_type"`
	Payload   map[string]any `json:"payload"`
}

type Handler struct {
	q *queue.MemoryQueue
}

func NewHandler(q *queue.MemoryQueue) *Handler {
	return &Handler{q: q}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		TargetURL string         `json:"target_url"`
		EventType string         `json:"event_type"`
		Payload   map[string]any `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if in.TargetURL == "" {
		http.Error(w, "target_url is required", http.StatusBadRequest)
		return
	}
	if in.EventType == "" {
		http.Error(w, "event_type is required", http.StatusBadRequest)
		return
	}
	if err := validateTargetURL(in.TargetURL); err != nil {
		http.Error(w, "invalid target_url", http.StatusBadRequest)
		return
	}

	ev := &Event{
		ID:        newID(),
		TargetURL: in.TargetURL,
		EventType: in.EventType,
		Payload:   in.Payload,
	}

	h.q.Enqueue(ev)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"id":     ev.ID,
		"status": "accepted",
	})
}

func validateTargetURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("invalid scheme")
	}
	if u.Host == "" {
		return errors.New("invalid host")
	}
	return nil
}

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}
