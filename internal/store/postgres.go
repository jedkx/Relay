package store

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"relay/internal/model"
)

//go:embed migrations/001_initial.sql
var schemaSQL string

func OpenPostgres(ctx context.Context, dsn string) (*Postgres, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if _, err := pool.Exec(ctx, schemaSQL); err != nil {
		pool.Close()
		return nil, fmt.Errorf("schema: %w", err)
	}
	return &Postgres{pool: pool}, nil
}

type Postgres struct {
	pool *pgxpool.Pool
}

func (p *Postgres) Close() { p.pool.Close() }

func (p *Postgres) InsertPending(ctx context.Context, ev *model.Event) error {
	b, err := json.Marshal(ev.Payload)
	if err != nil {
		return err
	}
	_, err = p.pool.Exec(ctx, `
		INSERT INTO events (id, target_url, event_type, payload, status)
		VALUES ($1, $2, $3, $4::jsonb, 'pending')
	`, ev.ID, ev.TargetURL, ev.EventType, b)
	return err
}

func (p *Postgres) ClaimNext(ctx context.Context) (*model.Event, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var (
		id, url, typ string
		raw           []byte
	)
	err = tx.QueryRow(ctx, `
		SELECT id, target_url, event_type, payload
		FROM events
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`).Scan(&id, &url, &typ, &raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `UPDATE events SET status = 'processing', updated_at = now() WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return &model.Event{
		ID:        id,
		TargetURL: url,
		EventType: typ,
		Payload:   payload,
	}, nil
}

func (p *Postgres) MarkDelivered(ctx context.Context, id string) error {
	_, err := p.pool.Exec(ctx, `UPDATE events SET status = 'delivered', updated_at = now() WHERE id = $1`, id)
	return err
}

func (p *Postgres) MarkFailed(ctx context.Context, id string) error {
	_, err := p.pool.Exec(ctx, `UPDATE events SET status = 'failed', updated_at = now() WHERE id = $1`, id)
	return err
}

func (p *Postgres) GetEvent(ctx context.Context, id string) (*model.EventDetail, error) {
	var ev model.EventDetail
	err := p.pool.QueryRow(ctx, `
		SELECT id, target_url, event_type, status, created_at
		FROM events WHERE id = $1
	`, id).Scan(&ev.ID, &ev.TargetURL, &ev.EventType, &ev.Status, &ev.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := p.pool.Query(ctx, `
		SELECT attempt_no, http_status, error_text, created_at
		FROM delivery_attempts WHERE event_id = $1
		ORDER BY attempt_no ASC
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ev.Attempts = []model.Attempt{}
	for rows.Next() {
		var a model.Attempt
		if err := rows.Scan(&a.AttemptNo, &a.HTTPStatus, &a.Error, &a.AttemptedAt); err != nil {
			return nil, err
		}
		ev.Attempts = append(ev.Attempts, a)
	}
	return &ev, rows.Err()
}

func (p *Postgres) ReclaimStuck(ctx context.Context, stuckFor time.Duration) (int64, error) {
	tag, err := p.pool.Exec(ctx, `
		UPDATE events
		SET status = 'pending', updated_at = now()
		WHERE status = 'processing'
		  AND updated_at < now() - make_interval(secs => $1)
	`, stuckFor.Seconds())
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (p *Postgres) RecordAttempt(ctx context.Context, eventID string, n int, status *int, msg *string) error {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO delivery_attempts (event_id, attempt_no, http_status, error_text)
		VALUES ($1, $2, $3, $4)
	`, eventID, n, status, msg)
	return err
}
