package store

import (
	"context"
	"time"

	"relay/internal/model"
)

type Store interface {
	InsertPending(ctx context.Context, ev *model.Event) error
	// ClaimNext returns (nil, nil) when there's nothing to do.
	ClaimNext(ctx context.Context) (*model.Event, error)
	MarkDelivered(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id string) error
	RecordAttempt(ctx context.Context, eventID string, attemptNo int, httpStatus *int, errText *string) error
	// ReclaimStuck moves events stuck in "processing" for longer than stuckFor back to "pending".
	ReclaimStuck(ctx context.Context, stuckFor time.Duration) (int64, error)
	// GetEvent returns the event with its delivery attempts, or (nil, nil) if not found.
	GetEvent(ctx context.Context, id string) (*model.EventDetail, error)
}
