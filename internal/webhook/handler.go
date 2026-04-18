package webhook

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"relay/internal/model"
	"relay/internal/store"
)

type Handler struct {
	s store.Store
}

func NewHandler(s store.Store) *Handler {
	return &Handler{s: s}
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
	if in.Payload == nil {
		in.Payload = map[string]any{}
	}

	ev := &model.Event{
		ID:        newID(),
		TargetURL: in.TargetURL,
		EventType: in.EventType,
		Payload:   in.Payload,
	}

	if err := h.s.InsertPending(r.Context(), ev); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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
