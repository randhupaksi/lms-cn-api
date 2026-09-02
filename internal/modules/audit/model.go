package audit

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"lms-cn-api/internal/authz"
	"lms-cn-api/pkg/apperror"
	"lms-cn-api/pkg/pagination"

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

func (s *Service) List(ctx context.Context, actor authz.Principal, page pagination.Request, filter Filter) ([]Response, int64, error) {
	if actor.Role != "admin" {
		return nil, 0, apperror.New(http.StatusForbidden, "AUDIT_ACCESS_DENIED", "Audit aktivitas hanya tersedia untuk admin")
	}
	query := s.db.WithContext(ctx).Table("audit_logs a")
	if value := strings.TrimSpace(filter.Action); value != "" {
		query = query.Where("a.action = ?", value)
	}
	if value := strings.TrimSpace(filter.EntityType); value != "" {
		query = query.Where("a.entity_type = ?", value)
	}
	if value := strings.TrimSpace(filter.ActorID); value != "" {
		query = query.Where("a.actor_id = ?", value)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, apperror.Wrap(http.StatusInternalServerError, "AUDIT_READ_FAILED", "Gagal memuat audit aktivitas", err)
	}
	var rows []eventRow
	if err := query.Select("a.*, COALESCE(u.full_name, '') AS actor_name").Joins("LEFT JOIN users u ON u.id = a.actor_id").Order("a.created_at DESC").Offset(page.Offset()).Limit(page.PerPage).Scan(&rows).Error; err != nil {
		return nil, 0, apperror.Wrap(http.StatusInternalServerError, "AUDIT_READ_FAILED", "Gagal memuat audit aktivitas", err)
	}
	result := make([]Response, len(rows))
	for index, row := range rows {
		result[index] = Response{ID: row.ID, ActorID: row.ActorID, ActorName: row.ActorName, Action: row.Action, EntityType: row.EntityType, EntityID: row.EntityID, Metadata: row.Metadata, CreatedAt: row.CreatedAt}
	}
	return result, total, nil
}
