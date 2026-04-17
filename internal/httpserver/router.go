package httpserver

import (
	"net/http"

	"relay/internal/webhook"
)

// NewRouter registers GET /health and POST /webhooks (Go 1.22+ ServeMux patterns).
func NewRouter(h *webhook.Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("POST /webhooks", h.Create)

	return mux
}
