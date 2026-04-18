package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net/http"
	"time"

	"relay/internal/model"
	"relay/internal/store"
)

const (
	idleWait     = 50 * time.Millisecond
	errBackoff   = time.Second
	maxTries     = 10
	httpTimeout  = 10 * time.Second
	backoffBase  = time.Second
	backoffMax   = 60 * time.Second
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
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
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

// exponentialPart is min(max, base * 2^(failedAttempt-1)) for failedAttempt >= 1.
func exponentialPart(failedAttempt int, base, max time.Duration) time.Duration {
	if failedAttempt < 1 {
		failedAttempt = 1
	}
	shift := failedAttempt - 1
	if shift >= 63 {
		return max
	}
	exp := int64(1) << uint(shift)
	p := time.Duration(exp) * base
	if p > max {
		return max
	}
	return p
}

func backoffAfterFailedAttempt(failedAttempt int, base, max time.Duration, rnd *rand.Rand) time.Duration {
	capped := exponentialPart(failedAttempt, base, max)
	jitter := time.Duration(rnd.Int64N(int64(base) + 1))
	return capped + jitter
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
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

	rng := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().UnixNano()>>32)))
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
			if i < maxTries {
				wait := backoffAfterFailedAttempt(i, backoffBase, backoffMax, rng)
				if err := sleepCtx(ctx, wait); err != nil {
					return err
				}
			}
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
		if i < maxTries {
			wait := backoffAfterFailedAttempt(i, backoffBase, backoffMax, rng)
			if err := sleepCtx(ctx, wait); err != nil {
				return err
			}
		}
	}
	return lastErr
}
