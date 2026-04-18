package delivery

import (
	"context"
	"io"
	"math/rand/v2"
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

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if hits.Load() >= 1 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("timeout waiting for delivery")
}

func TestExponentialPart(t *testing.T) {
	t.Parallel()
	base := time.Second
	max := 60 * time.Second
	cases := []struct {
		failed int
		want   time.Duration
	}{
		{1, time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
		{6, 32 * time.Second},
		{7, 60 * time.Second},
		{8, 60 * time.Second},
	}
	for _, tc := range cases {
		got := exponentialPart(tc.failed, base, max)
		if got != tc.want {
			t.Fatalf("failedAttempt=%d: got %v, want %v", tc.failed, got, tc.want)
		}
	}
}

func TestBackoffAfterFailedAttempt_JitterInRange(t *testing.T) {
	t.Parallel()
	base := time.Second
	max := 60 * time.Second
	r := rand.New(rand.NewPCG(42, 7))
	for attempt := 1; attempt <= 9; attempt++ {
		exp := exponentialPart(attempt, base, max)
		got := backoffAfterFailedAttempt(attempt, base, max, r)
		if got < exp || got > exp+base {
			t.Fatalf("attempt %d: got %v, want in [%v, %v]", attempt, got, exp, exp+base)
		}
	}
}
