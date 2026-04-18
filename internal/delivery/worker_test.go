package delivery

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"relay/internal/model"
	"relay/internal/store"
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

	ctx := context.Background()
	st := store.NewMemory()
	Start(ctx, st)

	ev := &model.Event{
		ID:        "evt-1",
		TargetURL: ts.URL,
		EventType: "test.event",
		Payload:   map[string]any{"n": 1},
	}
	if err := st.InsertPending(ctx, ev); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if hits.Load() >= 1 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("timeout waiting for delivery")
}
