package delivery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"relay/internal/queue"
	"relay/internal/webhook"
)

// Start runs a background worker that dequeues events and POSTs JSON to each event's TargetURL.
// Failed deliveries: up to 3 attempts, 100ms between attempts, 10s per HTTP round trip.
func Start(q *queue.MemoryQueue) {
	go run(q)
}

func run(q *queue.MemoryQueue) {
	for {
		item := q.Dequeue()
		ev, ok := item.(*webhook.Event)
		if !ok {
			log.Printf("relay: unexpected queue item type %T", item)
			continue
		}
		if err := deliver(ev); err != nil {
			log.Printf("relay: delivery failed id=%s url=%s: %v", ev.ID, ev.TargetURL, err)
		}
	}
}

func deliver(ev *webhook.Event) error {
	out := map[string]any{
		"relay_event_id": ev.ID,
		"event_type":     ev.EventType,
		"payload":        ev.Payload,
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		req, err := http.NewRequest(http.MethodPost, ev.TargetURL, bytes.NewReader(raw))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(100 * time.Millisecond)
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("http %d", resp.StatusCode)
		time.Sleep(100 * time.Millisecond)
	}
	return lastErr
}
