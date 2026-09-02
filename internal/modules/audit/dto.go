package audit

import (
	"encoding/json"
	"time"
)

type Filter struct {
	Action     string
	EntityType string
	ActorID    string
}

type Response struct {
	ID         string          `json:"id"`
	ActorID    *string         `json:"actor_id"`
	ActorName  string          `json:"actor_name,omitempty"`
	Action     string          `json:"action"`
	EntityType string          `json:"entity_type"`
	EntityID   *string         `json:"entity_id"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

type eventRow struct {
	Event
	ActorName string
}
