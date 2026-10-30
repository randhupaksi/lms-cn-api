package governance

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"lms-cn-api/internal/modules/audit"
)

var (
	ErrLeadershipActionNotFound = errors.New("leadership action not found")
	ErrLeadershipAssignee       = errors.New("leadership action assignee is unavailable")
	ErrLeadershipVersion        = errors.New("leadership action version is stale")
)

func (r *Repository) ListActionAssignees(ctx context.Context) ([]LeadershipActionAssignee, error) {
	rows := []LeadershipActionAssignee{}
	err := r.db.WithContext(ctx).Table("users").
		Select("id, full_name, role").
		Where("status = ? AND role IN ?", "active", []string{"admin", "teacher", "curriculum"}).
		Order("role, full_name, id").Scan(&rows).Error
	return rows, err
}

func (r *Repository) ListLeadershipActions(ctx context.Context, role, userID, status, category string, page, size int) ([]leadershipActionListRow, int64, error) {
	query := r.db.WithContext(ctx).Table("leadership_actions a")
	if role != "superadmin" && role != "principal" {
		query = query.Where("a.assignee_id = ?", userID)
	}
	if status != "" {
		query = query.Where("a.status = ?", status)
	}
	if category != "" {
		query = query.Where("a.category = ?", category)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := []leadershipActionListRow{}
	err := query.Select(`a.*,
 COALESCE(assignee.full_name, '') AS assignee_name,
 creator.full_name AS created_by_name,
 COALESCE((SELECT u.note FROM leadership_action_updates u WHERE u.action_id=a.id ORDER BY u.created_at DESC,u.id DESC LIMIT 1), '') AS latest_note`).
		Joins("LEFT JOIN users assignee ON assignee.id = a.assignee_id").
		Joins("JOIN users creator ON creator.id = a.created_by").
		Order("CASE a.priority WHEN 'high' THEN 0 WHEN 'normal' THEN 1 ELSE 2 END, a.due_at IS NULL, a.due_at, a.created_at DESC, a.id").
		Offset((page - 1) * size).Limit(size).Scan(&rows).Error
	return rows, total, err
}

func (r *Repository) CreateLeadershipAction(ctx context.Context, value LeadershipAction, actorID string, note string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if value.AssigneeID != nil {
			var count int64
			if err := tx.Table("users").Where("id = ? AND status = ? AND role IN ?", *value.AssigneeID, "active", []string{"admin", "teacher", "curriculum"}).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return ErrLeadershipAssignee
			}
		}
		if err := tx.Create(&value).Error; err != nil {
			return err
		}
		if err := tx.Create(&LeadershipActionUpdate{ID: uuid.NewString(), ActionID: value.ID, ActorID: actorID, ToStatus: value.Status, Note: note, CreatedAt: value.CreatedAt}).Error; err != nil {
			return err
		}
		return audit.NewService(tx).Record(ctx, audit.Record{ActorID: actorID, Action: "leadership.action_created", EntityType: "leadership_action", EntityID: value.ID, Metadata: map[string]any{"category": value.Category, "priority": value.Priority}})
	})
}

func (r *Repository) FindLeadershipAction(ctx context.Context, id string) (LeadershipAction, error) {
	var value LeadershipAction
	err := r.db.WithContext(ctx).First(&value, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return LeadershipAction{}, ErrLeadershipActionNotFound
	}
	return value, err
}

func (r *Repository) UpdateLeadershipActionStatus(ctx context.Context, id, actorID, nextStatus, note string, expectedVersion int, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var value LeadershipAction
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&value, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrLeadershipActionNotFound
			}
			return err
		}
		if value.Version != expectedVersion {
			return ErrLeadershipVersion
		}
		previous := value.Status
		value.Status = nextStatus
		value.Version++
		value.UpdatedAt = now
		if nextStatus == "completed" {
			value.CompletedAt = &now
		} else {
			value.CompletedAt = nil
		}
		if err := tx.Save(&value).Error; err != nil {
			return err
		}
		if err := tx.Create(&LeadershipActionUpdate{ID: uuid.NewString(), ActionID: id, ActorID: actorID, FromStatus: &previous, ToStatus: nextStatus, Note: note, CreatedAt: now}).Error; err != nil {
			return err
		}
		return audit.NewService(tx).Record(ctx, audit.Record{ActorID: actorID, Action: "leadership.action_updated", EntityType: "leadership_action", EntityID: id, Metadata: map[string]any{"from": previous, "to": nextStatus, "version": value.Version}})
	})
}

func (r *Repository) LeadershipActionUpdates(ctx context.Context, id string) ([]leadershipActionUpdateRow, error) {
	rows := []leadershipActionUpdateRow{}
	err := r.db.WithContext(ctx).Table("leadership_action_updates u").
		Select("u.*, actor.full_name AS actor_name").
		Joins("JOIN users actor ON actor.id = u.actor_id").
		Where("u.action_id = ?", id).
		Order("u.created_at ASC, u.id ASC").Find(&rows).Error
	return rows, err
}

func (r *Repository) LeadershipActionCount(ctx context.Context, role, userID string, statuses ...string) (int64, error) {
	query := r.db.WithContext(ctx).Model(&LeadershipAction{})
	if role != "superadmin" && role != "principal" {
		query = query.Where("assignee_id = ?", userID)
	}
	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}
	var count int64
	err := query.Count(&count).Error
	return count, err
}

func (r *Repository) OverdueLeadershipActionCount(ctx context.Context, role, userID string, now time.Time) (int64, error) {
	query := r.db.WithContext(ctx).Model(&LeadershipAction{}).
		Where("due_at < ? AND status NOT IN ?", now, []string{"completed", "cancelled"})
	if role != "superadmin" && role != "principal" {
		query = query.Where("assignee_id = ?", userID)
	}
	var count int64
	err := query.Count(&count).Error
	return count, err
}
