# Roadmap

Where relay is headed. Not a release calendar—more a checklist so I don’t forget what “honest” delivery still needs.

### Shipped (current `main`)

- **Postgres persistence** — `events` + `delivery_attempts`, schema applied when `relay` starts (`internal/store/migrations`).
- **Claim path** — `FOR UPDATE SKIP LOCKED`, `pending` → `processing` → `delivered` / `failed`.
- **Attempt rows** — each HTTP try recorded (status and/or error text).
- **Outbound retries** — exponential backoff with jitter between attempts (10 tries max, 1s base, 60s cap); shutdown cancels waits without marking the event failed.
- **Graceful stop** — worker context cancelled on shutdown (SIGTERM etc.); not a full drain story yet.

The old **`internal/queue`** package is only for its own tests; production path is the store.

### Next if I open the editor

Rough priority:

1. **Idempotency on ingest** — optional client key, dedupe / same response on replay.
2. **Stuck `processing`** — reclaim rows if the process dies after claim (lease / timeout job).
3. **Tighter updates** — e.g. `MarkDelivered` only from `processing` so bugs don’t flip wrong rows.
4. **Auth on ingest** — API key (or similar) + basic SSRF guardrails on `target_url`.
5. **Read API** — `GET /events/:id` (+ attempts) before any UI fantasy.

Finer control (Retry-After header, per-destination presets) stays in Phase 2.

---

## Phase 1 — don’t embarrass yourself

Goal: persistence, retries you can explain, enough trail that “what happened to event X?” isn’t printf archaeology.

| Area | Status |
|------|--------|
| Durability | Postgres in place; restart keeps queue. Gaps: no versioned migrations, no stale-`processing` reclaim. |
| Idempotency | Not started. |
| Failure handling | Terminal `failed` + attempts table; no separate DLQ queue yet. |
| Retries | Exponential backoff + jitter between HTTP tries (see worker). Still no Retry-After, no named presets. |
| Audit trail | Attempt rows exist; could add request bytes, duration, correlation IDs. |
| Lifecycle | Worker stops with process; in-flight HTTP may still run briefly—document / tighten if needed. |
| Security | No ingest auth yet; URL validation is minimal. |
| Observability | `log.Printf`; structured logs + metrics later. |
| Operator surface | DB or future `GET /events/:id`. |

Phase 1 “feels done” when I’d paste the README in a PR and not hedge with “well, in prod you’d…”

---

## Phase 2 — worth bragging about

- **Retry-After** on 429 (and similar): honor header within caps; document malformed headers.
- **Breaker per destination** (not one global switch).
- **Named retry presets** in config (`standard`, `relaxed`, …) with versioned definitions.
- **Failure taxonomy** — e.g. rate limit vs auth vs permanent vs timeout; one field an operator can act on.

Each of these deserves a short `docs/` note with examples and sharp edges, not only Go structs.

---

## Phase 3 — only if it becomes a product

Replay (single event, then time-window), thin UI over the same read APIs, tenants/billing—**after** Phase 1–2 aren’t hollow. UI should not become a second source of truth.

---

## Maintenance

When something ships, update this file or it lies. If two goals conflict (e.g. zero migration pain vs strict versioning), pick the tradeoff here in one sentence.
