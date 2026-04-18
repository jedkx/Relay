package store

import (
	"context"
	"sync"

	"relay/internal/model"
)

// Memory is a Store backed by a map; fine for tests, not for production.
type Memory struct {
	mu    sync.Mutex
	byID  map[string]*memRow
	order []string
}

type memRow struct {
	ev     *model.Event
	status string
}

func NewMemory() *Memory {
	return &Memory{byID: make(map[string]*memRow)}
}

func copyEvent(e *model.Event) *model.Event {
	p := make(map[string]any, len(e.Payload))
	for k, v := range e.Payload {
		p[k] = v
	}
	return &model.Event{
		ID:        e.ID,
		TargetURL: e.TargetURL,
		EventType: e.EventType,
		Payload:   p,
	}
}

func (m *Memory) InsertPending(_ context.Context, ev *model.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.byID[ev.ID] = &memRow{ev: copyEvent(ev), status: "pending"}
	m.order = append(m.order, ev.ID)
	return nil
}

func (m *Memory) ClaimNext(_ context.Context) (*model.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range m.order {
		r := m.byID[id]
		if r != nil && r.status == "pending" {
			r.status = "processing"
			return copyEvent(r.ev), nil
		}
	}
	return nil, nil
}

func (m *Memory) MarkDelivered(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r := m.byID[id]; r != nil {
		r.status = "delivered"
	}
	return nil
}

func (m *Memory) MarkFailed(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r := m.byID[id]; r != nil {
		r.status = "failed"
	}
	return nil
}

func (m *Memory) RecordAttempt(_ context.Context, _ string, _ int, _ *int, _ *string) error {
	return nil
}

func (m *Memory) Get(id string) (*model.Event, string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.byID[id]
	if !ok {
		return nil, "", false
	}
	return copyEvent(r.ev), r.status, true
}
