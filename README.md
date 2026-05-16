# relay

## Why this exists

When your service calls a webhook, the target URL might be down. The naive fix is retry logic inside the caller — but now every caller reimplements it differently, and none of them survive a process restart.

Relay moves that responsibility out. The caller fires a single HTTP request and gets a 202 back immediately. Relay writes the event to Postgres and delivers it in the background, retrying up to 10 times with exponential backoff. If the process crashes mid-flight, stuck events are reclaimed on the next start.

The caller doesn't need to know whether delivery succeeded. It can check later.

## What it does

```
Your service
      │
      │  POST /webhooks
      │  {"target_url": "...", "event_type": "...", "payload": {...}}
      ▼
  Handler  →  Postgres (status: pending)  →  202 Accepted + id
                    │
                    ▼
             Background worker
             ├── claim row (FOR UPDATE SKIP LOCKED)
             ├── POST payload to target_url
             ├── record attempt (http_status or error)
             └── mark delivered / retry with backoff / mark failed
```

On startup, any event stuck in `processing` for more than 5 minutes (e.g. from a previous crash) is moved back to `pending` automatically.

## This is a core

No auth, no idempotency, no circuit breaker, no metrics. Those are your responsibility:

- **Auth on ingest** — `POST /webhooks` is open. Add your own middleware or reverse proxy in front of it.
- **Deduplication** — if your caller retries the POST, two events are created. Handle dedup at the caller or add it here.
- **Single node** — one worker goroutine, one process. No distributed locking across replicas.

The extension points are the `Store` interface (`internal/store/store.go`) and the `delivery` package — both are straightforward to replace or wrap.

## Run

```bash
docker compose up --build
```

- Relay: `http://127.0.0.1:8080`
- Postgres: `127.0.0.1:5433`, db `relay_dev`, user/password `relay` / `relay`

Schema is applied when relay connects, not when Postgres starts.

## Configuration

| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | yes | Postgres DSN |

Everything else is hard-coded in `internal/delivery/worker.go`:

| Constant | Value | Description |
|----------|-------|-------------|
| `maxTries` | 10 | Max delivery attempts per event |
| `backoffBase` | 1s | Base for exponential backoff |
| `backoffMax` | 60s | Backoff cap |
| `httpTimeout` | 10s | Per-attempt HTTP timeout |
| `stuckFor` | 5m | Threshold to reclaim stuck events on startup |

## API

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Returns `200 ok` |
| `POST` | `/webhooks` | Enqueue a webhook for delivery |
| `GET` | `/events/{id}` | Get event status + delivery attempts |

### POST /webhooks

```bash
curl -s -X POST http://127.0.0.1:8080/webhooks \
  -H "Content-Type: application/json" \
  -d '{"target_url":"https://example.com/hook","event_type":"order.created","payload":{"id":42}}'
```

Response `202 Accepted`:

```json
{ "id": "a3f8c1d2...", "status": "accepted" }
```

### GET /events/{id}

```bash
curl http://127.0.0.1:8080/events/a3f8c1d2...
```

Response `200 OK`:

```json
{
  "id": "a3f8c1d2...",
  "target_url": "https://example.com/hook",
  "event_type": "order.created",
  "status": "failed",
  "created_at": "2026-05-16T10:00:00Z",
  "attempts": [
    { "attempt_no": 1, "http_status": 500, "error": "http 500", "attempted_at": "..." },
    { "attempt_no": 2, "http_status": null, "error": "connection refused", "attempted_at": "..." }
  ]
}
```

`status` is one of: `pending`, `processing`, `delivered`, `failed`.

## Tests

```bash
docker compose --profile test run --rm test
```

## License

MIT — `LICENSE`.
