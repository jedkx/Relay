package store

import (
	"context"

	"relay/internal/model"
)

type Store interface {
	InsertPending(ctx context.Context, ev *model.Event) error
	// ClaimNext returns (nil, nil) when there's nothing to do.
	ClaimNext(ctx context.Context) (*model.Event, error)
	MarkDelivered(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id string) error
	RecordAttempt(ctx context.Context, eventID string, attemptNo int, httpStatus *int, errText *string) error
}
