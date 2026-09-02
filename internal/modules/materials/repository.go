package materials

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrNotFound = errors.New("material not found")
	ErrLocked   = errors.New("material locked")
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, material *Material) error {
	return r.db.WithContext(ctx).Create(material).Error
}

func (r *Repository) Find(ctx context.Context, id string) (Material, error) {
	var material Material
	if err := r.db.WithContext(ctx).First(&material, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Material{}, ErrNotFound
		}
		return Material{}, err
	}
	return material, nil
}

func (r *Repository) List(ctx context.Context, courseID, studentID string, publishedOnly bool) ([]materialRow, error) {
	query := r.db.WithContext(ctx).Table("course_materials m").Select("m.*")
	if studentID != "" {
		query = query.Select("m.*, mp.completed_at").Joins("LEFT JOIN material_progress mp ON mp.material_id = m.id AND mp.student_id = ?", studentID)
	}
	query = query.Where("m.course_id = ?", courseID)
	if publishedOnly {
		query = query.Where("m.status = ?", StatusPublished)
	}
	var rows []materialRow
	err := query.Order("m.position ASC, m.created_at ASC").Scan(&rows).Error
	return rows, err
}

func (r *Repository) UpdateDraft(ctx context.Context, id string, apply func(*Material)) (Material, error) {
	var result Material
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var material Material
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&material, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if material.Status != StatusDraft {
			return ErrLocked
		}
		apply(&material)
		if err := tx.Save(&material).Error; err != nil {
			return err
		}
		result = material
		return nil
	})
	return result, err
}

func (r *Repository) Publish(ctx context.Context, id string, now time.Time) error {
	result := r.db.WithContext(ctx).Model(&Material{}).Where("id = ? AND status = ?", id, StatusDraft).Updates(map[string]any{"status": StatusPublished, "published_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrLocked
	}
	return nil
}

func (r *Repository) Complete(ctx context.Context, materialID, studentID string, now time.Time) error {
	progress := Progress{MaterialID: materialID, StudentID: studentID, CompletedAt: now}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "material_id"}, {Name: "student_id"}}, DoUpdates: clause.AssignmentColumns([]string{"completed_at"})}).Create(&progress).Error
}
