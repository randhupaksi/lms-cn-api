package questions

import (
	"context"
	"errors"

	"lms-cn-api/pkg/pagination"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrNotFound = errors.New("question not found")

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, question *Question) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("Options").Create(question).Error; err != nil {
			return err
		}
		return tx.Create(&question.Options).Error
	})
}

func (r *Repository) Find(ctx context.Context, id string) (Question, error) {
	var question Question
	err := r.db.WithContext(ctx).Preload("Options", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC") }).First(&question, "questions.id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Question{}, ErrNotFound
	}
	return question, err
}

func (r *Repository) List(ctx context.Context, courseID string, page pagination.Request, filter ListFilter) ([]Question, int64, error) {
	query := r.db.WithContext(ctx).Model(&Question{}).Where("course_id = ?", courseID)
	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}
	if filter.Tag != "" {
		query = query.Where("JSON_CONTAINS(tags, JSON_QUOTE(?))", filter.Tag)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Search != "" {
		query = query.Where("stem LIKE ?", "%"+filter.Search+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var result []Question
	err := query.Preload("Options", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC") }).
		Order("updated_at DESC").Offset(page.Offset()).Limit(page.PerPage).Find(&result).Error
	return result, total, err
}

func (r *Repository) Update(ctx context.Context, id string, apply func(*Question)) (Question, error) {
	var result Question
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var question Question
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&question, "id = ?", id).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		apply(&question)
		question.Version++
		if err := tx.Omit("Options").Save(&question).Error; err != nil {
			return err
		}
		if err := tx.Where("question_id = ?", id).Delete(&Option{}).Error; err != nil {
			return err
		}
		if err := tx.Create(&question.Options).Error; err != nil {
			return err
		}
		result = question
		return nil
	})
	return result, err
}

func (r *Repository) Archive(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Model(&Question{}).Where("id = ?", id).Update("status", "archived")
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
