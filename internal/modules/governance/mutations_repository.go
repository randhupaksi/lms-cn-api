package governance

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"lms-cn-api/internal/modules/audit"
	"lms-cn-api/internal/modules/exams"
	"lms-cn-api/pkg/apperror"
	"time"
)

func (r *Repository) CreateRequest(ctx context.Context, value ScheduleRequest, validate func(ExamReadiness) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var exam exams.Exam
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&exam, "id=?", value.ExamID).Error; err != nil {
			return err
		}
		row, err := NewRepository(tx).Exam(ctx, value.ExamID)
		if err != nil {
			return err
		}
		if err = validate(row); err != nil {
			return err
		}
		value.OriginalStartsAt = exam.StartsAt
		value.OriginalEndsAt = exam.EndsAt
		if err = tx.Create(&value).Error; err != nil {
			return err
		}
		return audit.NewService(tx).Record(ctx, audit.Record{ActorID: value.RequesterID, Action: "governance.request_created", EntityType: "governance_request", EntityID: value.ID})
	})
}

func (r *Repository) Decide(ctx context.Context, id, actorID, status, note string, now time.Time, validate func(ScheduleRequest, ExamReadiness) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var value ScheduleRequest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&value, "id=?", id).Error; err != nil {
			return err
		}
		var exam exams.Exam
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&exam, "id=?", value.ExamID).Error; err != nil {
			return err
		}
		row, err := NewRepository(tx).Exam(ctx, value.ExamID)
		if err != nil {
			return err
		}
		if err = validate(value, row); err != nil {
			return err
		}
		if status == "approved" {
			if err = tx.Model(&exam).Updates(map[string]any{"starts_at": value.ProposedStartsAt, "ends_at": value.ProposedEndsAt, "updated_at": now}).Error; err != nil {
				return err
			}
		}
		updates := map[string]any{"status": status, "decision_note": note, "decided_at": now}
		if status != "cancelled" {
			updates["reviewer_id"] = actorID
		}
		if err = tx.Model(&value).Updates(updates).Error; err != nil {
			return err
		}
		return audit.NewService(tx).Record(ctx, audit.Record{ActorID: actorID, Action: "governance.request_" + status, EntityType: "governance_request", EntityID: id, Metadata: map[string]any{"exam_id": value.ExamID, "note": note}})
	})
}

func (r *Repository) SaveSettings(ctx context.Context, value Settings, actorID, reason string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current Settings
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, 1).Error; err != nil {
			return err
		}
		if current.Version != value.Version {
			return apperror.New(409, "STALE_SETTINGS", "Pengaturan telah berubah. Muat ulang sebelum menyimpan.")
		}
		value.Version++
		value.ID = 1
		if err := tx.Save(&value).Error; err != nil {
			return err
		}
		snapshot, err := json.Marshal(value)
		if err != nil {
			return err
		}
		if err = tx.Create(&SettingChange{ID: uuid.NewString(), ActorID: actorID, Version: value.Version, Reason: reason, Snapshot: snapshot, CreatedAt: value.UpdatedAt}).Error; err != nil {
			return err
		}
		return audit.NewService(tx).Record(ctx, audit.Record{ActorID: actorID, Action: "school.settings_updated", EntityType: "school_settings", Metadata: map[string]any{"version": value.Version, "reason": reason}})
	})
}
