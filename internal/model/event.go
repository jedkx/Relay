package model

import "time"

type Event struct {
	ID        string         `json:"id"`
	TargetURL string         `json:"target_url"`
	EventType string         `json:"event_type"`
	Payload   map[string]any `json:"payload"`
}

type Attempt struct {
	AttemptNo   int       `json:"attempt_no"`
	HTTPStatus  *int      `json:"http_status"`
	Error       *string   `json:"error"`
	AttemptedAt time.Time `json:"attempted_at"`
}

type EventDetail struct {
	ID        string    `json:"id"`
	TargetURL string    `json:"target_url"`
	EventType string    `json:"event_type"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	Attempts  []Attempt `json:"attempts"`
}
