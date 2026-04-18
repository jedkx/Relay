package model

type Event struct {
	ID        string         `json:"id"`
	TargetURL string         `json:"target_url"`
	EventType string         `json:"event_type"`
	Payload   map[string]any `json:"payload"`
}
