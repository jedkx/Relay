CREATE TABLE IF NOT EXISTS events (
    id              TEXT PRIMARY KEY,
    target_url      TEXT NOT NULL,
    event_type      TEXT NOT NULL,
    payload         JSONB NOT NULL,
    status          TEXT NOT NULL CHECK (status IN ('pending', 'processing', 'delivered', 'failed')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS events_pending_created_idx
    ON events (status, created_at)
    WHERE status = 'pending';

CREATE TABLE IF NOT EXISTS delivery_attempts (
    id              BIGSERIAL PRIMARY KEY,
    event_id        TEXT NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    attempt_no      INT NOT NULL,
    http_status     INT,
    error_text      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS delivery_attempts_event_idx ON delivery_attempts (event_id);
