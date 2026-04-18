# relay

Small Go service: accept a webhook-shaped JSON, write it to **Postgres**, deliver it in the background with retries. Stack: Go 1.22, `net/http`, [pgx](https://github.com/jackc/pgx).

## What it does

Callers get **202** quickly. The handler validates, inserts **`pending`** in `events`, returns `id` / `accepted`. A worker **claims** rows (`FOR UPDATE SKIP LOCKED`), POSTs JSON to `target_url`, logs each try in **`delivery_attempts`**, then marks **`delivered`** or **`failed`**.

Outbound body shape:

```json
{
  "relay_event_id": "<id>",
  "event_type": "<string>",
  "payload": { }
}
```

Outbound retries: up to **10** attempts, **exponential backoff** (1s base, 60s cap) **plus jitter** in `[0, 1s]` between tries, **10s** HTTP client timeout per try. See `internal/delivery/worker.go`.

## Layout

| Path | Role |
|------|------|
| `cmd/relay` | HTTP API + worker (needs `DATABASE_URL`) |
| `cmd/relay-mock` | Tiny receiver for local smoke tests |
| `internal/webhook` | `POST /webhooks` |
| `internal/httpserver` | Routes (`GET /health`, …) |
| `internal/store` | Postgres + in-memory `Store` for tests |
| `internal/model` | Shared `Event` struct |
| `internal/delivery` | Claim loop + HTTP delivery |
| `docs/ROADMAP.md` | What’s done vs what’s next |

## Config

- **`DATABASE_URL`** — required for `cmd/relay` (see `docker-compose.yml` or `.env.example`).
- Ports and retry tuning are still hard-coded.

## Docker (easiest)

```bash
docker compose up --build
```

- Relay: `http://127.0.0.1:8080` — `GET /health`, `POST /webhooks`
- Mock: `http://127.0.0.1:8081` — `POST /receive` (logs body)
- Postgres: `127.0.0.1:5432`, db `relay_dev`, user/password `relay` / `relay` (dev only)

**Schema:** applied when **relay** connects (`internal/store/migrations`), not when Postgres starts alone.

**`target_url` from relay’s container:** use `http://relay-mock:8081/receive`, not `127.0.0.1`, so delivery can reach the mock. From the host (curl, Postman, etc.) you still call relay at `127.0.0.1:8080`.

Example:

```bash
curl -s http://127.0.0.1:8080/health
curl -s -X POST http://127.0.0.1:8080/webhooks \
  -H "Content-Type: application/json" \
  -d '{"target_url":"http://relay-mock:8081/receive","event_type":"demo","payload":{"k":1}}'
```

Wipe data: `docker compose down -v`.

## Without Docker

Postgres running locally, then:

```bash
go run ./cmd/relay-mock
export DATABASE_URL=postgres://relay:relay@127.0.0.1:5432/relay_dev?sslmode=disable   # Unix
go run ./cmd/relay
```

Windows (cmd): `set DATABASE_URL=postgres://relay:relay@127.0.0.1:5432/relay_dev?sslmode=disable`

Use `http://127.0.0.1:8081/receive` in `target_url` when mock runs on the host.

## Tests

```bash
go test ./... -count=1
```

Or no local Go: `docker compose --profile test run --rm test`

## CI

`.github/workflows/ci.yml` — `go vet`, `go test`, `go build` on push/PR to `main` / `master`.

## Roadmap / license

See **`docs/ROADMAP.md`**. Errors and docs in this repo are English. **MIT** — `LICENSE`.
