package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"relay/internal/model"
	"relay/internal/store"
)

const (
	idleWait   = 50 * time.Millisecond
	errBackoff = time.Second
	maxTries   = 3
	httpTimeout = 10 * time.Second
	retryPause  = 100 * time.Millisecond
)

func Start(ctx context.Context, s store.Store) {
	go run(ctx, s)
}

func run(ctx context.Context, s store.Store) {
	for {
		if ctx.Err() != nil {
			return
		}

		ev, err := s.ClaimNext(ctx)
		if err != nil {
			if ctx.Err() == nil {
				log.Printf("claim: %v", err)
				time.Sleep(errBackoff)
			}
			continue
		}
		if ev == nil {
			time.Sleep(idleWait)
			continue
		}

		err = deliver(ctx, s, ev)
		if err != nil {
			if e := s.MarkFailed(ctx, ev.ID); e != nil {
				log.Printf("mark failed %s: %v", ev.ID, e)
			}
			log.Printf("deliver id=%s url=%s: %v", ev.ID, ev.TargetURL, err)
			continue
		}
		if e := s.MarkDelivered(ctx, ev.ID); e != nil {
			log.Printf("mark delivered %s: %v", ev.ID, e)
		}
	}
}

func deliver(ctx context.Context, s store.Store, ev *model.Event) error {
	body, err := json.Marshal(map[string]any{
		"relay_event_id": ev.ID,
		"event_type":     ev.EventType,
		"payload":        ev.Payload,
	})
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: httpTimeout}
	var lastErr error

	for i := 1; i <= maxTries; i++ {
		var status *int
		var msg *string

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, ev.TargetURL, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			m := err.Error()
			msg = &m
			_ = s.RecordAttempt(ctx, ev.ID, i, status, msg)
			time.Sleep(retryPause)
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		code := resp.StatusCode
		status = &code
		if code < 200 || code >= 300 {
			t := fmt.Sprintf("http %d", code)
			msg = &t
			lastErr = fmt.Errorf("http %d", code)
		}
		_ = s.RecordAttempt(ctx, ev.ID, i, status, msg)
		if code >= 200 && code < 300 {
			return nil
		}
		time.Sleep(retryPause)
	}
	return lastErr
}
