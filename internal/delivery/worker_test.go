package delivery

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"relay/internal/queue"
	"relay/internal/webhook"
)

func TestWorker_DeliversPOSTToTarget(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		b, _ := io.ReadAll(r.Body)
		if len(b) == 0 {
			t.Error("empty body")
		}
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	q := queue.NewMemoryQueue(10)
	Start(q)

	ev := &webhook.Event{
		ID:        "evt-1",
		TargetURL: ts.URL,
		EventType: "test.event",
		Payload:   map[string]any{"n": 1},
	}
	q.Enqueue(ev)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if hits.Load() >= 1 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("timeout waiting for delivery")
}
