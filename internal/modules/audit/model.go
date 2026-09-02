package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Event struct {
	ID         string          `gorm:"type:char(36);primaryKey"`
	ActorID    *string         `gorm:"type:char(36)"`
	Action     string          `gorm:"size:80;not null"`
	EntityType string          `gorm:"size:64;not null"`
	EntityID   *string         `gorm:"type:char(36)"`
	Metadata   json.RawMessage `gorm:"type:json"`
	CreatedAt  time.Time
}

func (Event) TableName() string { return "audit_logs" }

type Record struct {
	ActorID    string
	Action     string
	EntityType string
	EntityID   string
	Metadata   map[string]any
}

type Recorder interface {
	Record(ctx context.Context, record Record) error
}

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

func (s *Service) Record(ctx context.Context, record Record) error {
	metadata, err := json.Marshal(record.Metadata)
	if err != nil {
		return err
	}
	var actorID, entityID *string
	if record.ActorID != "" {
		actorID = &record.ActorID
	}
	if record.EntityID != "" {
		entityID = &record.EntityID
	}
	event := Event{
		ID:         uuid.NewString(),
		ActorID:    actorID,
		Action:     record.Action,
		EntityType: record.EntityType,
		EntityID:   entityID,
		Metadata:   metadata,
	}
	return s.db.WithContext(ctx).Create(&event).Error
}
