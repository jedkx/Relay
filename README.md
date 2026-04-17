# relay

Small HTTP service: take webhook-style payloads, queue them, POST them to whatever URL the client asked for. Written in Go with the stdlib only.

## Why this exists

If your integration calls someone else’s URL synchronously, their slowness becomes your latency, and their outage becomes your 500. I wanted a place where the caller can get a quick **202** after validation, and the actual outbound HTTP happens in the background with retries. That’s all this is—not a message bus, just a clear split between “we accepted the work” and “we tried to deliver it.”

## Why delivery isn’t in the ingest handler

Waiting for the downstream round trip inside `POST /webhooks` would mean every client waits on that network hop. Instead the handler validates, enqueues a struct, returns `id` / `status`, and **`internal/delivery`** does the HTTP work in a loop.

## Why the queue isn’t persistent

A real system would stick pending work in Postgres or a real queue so restarts don’t lose anything. Here it’s a **buffered channel** in memory so the whole pipeline fits in one process and stays easy to follow. **Restart clears the queue.** If you fork this for production, that’s the first thing I’d replace.

## Outbound JSON

The worker POSTs to `target_url` with:

```json
{
  "relay_event_id": "<id>",
  "event_type": "<string>",
  "payload": { }
}
```

Up to 3 tries, 100ms sleep between failures, 10s timeout per request. 2xx = success. After that it logs and moves on—no dead-letter queue in this code. See **`internal/delivery/worker.go`**.

## Flow

1. `POST /webhooks` with `target_url`, `event_type`, `payload`.
2. Validation (URL must be `http`/`https` with a host, etc.) in **`internal/webhook`**.
3. Worker picks up events and POSTs as above. Routes are **`internal/httpserver`** (Go 1.22 `ServeMux` patterns).

## Run with Docker (default)

You don’t need Go on the host—images build the binaries for you. From the repo root:

```bash
docker compose up --build
```

- **relay** → [http://127.0.0.1:8080](http://127.0.0.1:8080) (`GET /health`, `POST /webhooks`)
- **relay-mock** → [http://127.0.0.1:8081](http://127.0.0.1:8081) (`POST /receive` logs the body)

Smoke check (host shell or Postman). **Important:** the ingest API runs in a container, but the **delivery worker** also runs there—so `target_url` must be reachable **from inside the relay container**. With Compose, use the mock service name, not `127.0.0.1`:

```bash
curl -s http://127.0.0.1:8080/health
curl -s -X POST http://127.0.0.1:8080/webhooks \
  -H "Content-Type: application/json" \
  -d '{"target_url":"http://relay-mock:8081/receive","event_type":"demo","payload":{"k":1}}'
```

Expect **202** and JSON with `id` / `accepted`; **`docker compose logs -f relay-mock`** (or the mock container logs) should show the forwarded payload.

**Postman:** import **`postman/relay.postman_collection.json`**. Collection variable **`deliveryTarget`** defaults to `http://relay-mock:8081/receive` for Docker. If you run **`go run`** for both binaries on the host instead, set **`deliveryTarget`** to `http://127.0.0.1:8081/receive`.

On Windows, **Docker Desktop** (Linux engine) is required; bind mounts for the test profile need it.

## Tests (Docker, no local Go)

```bash
docker compose --profile test run --rm test
```

That spins up `golang:1.22-alpine`, mounts the repo, and runs `go test ./...`. Use this when you don’t have Go installed locally.

## Run without Docker

If you have **Go 1.22+** on the machine:

```bash
go run ./cmd/relay-mock
go run ./cmd/relay
```

Same ports and curl as above.

```bash
go test ./...
```

## Stack

Go 1.22, `net/http`, no frameworks. Runtime image is Alpine + static binaries (`Dockerfile` multi-stage build).

## Notes

No env-based config yet—ports and retry numbers are hard-coded in **`cmd/relay`** and **`internal/delivery`**. Docs and errors in this repo are English.

There’s no UI, so no screenshots.

## License

MIT — see **`LICENSE`**.
