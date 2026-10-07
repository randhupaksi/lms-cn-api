package results

import (
	"context"
	"errors"
	"strings"
	"time"

	"lms-cn-api/internal/modules/exams"
	"lms-cn-api/pkg/pagination"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("result not found")

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) ExamCourseID(ctx context.Context, examID string) (string, error) {
	var exam exams.Exam
	err := r.db.WithContext(ctx).Select("course_id").First(&exam, "id = ?", examID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", ErrNotFound
	}
	return exam.CourseID, err
}

func (r *Repository) ListByExam(ctx context.Context, examID string, page pagination.Request) ([]resultRow, int64, error) {
	query := r.db.WithContext(ctx).Table("results r").Where("r.exam_id = ?", examID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []resultRow
	err := query.Select("r.*, e.title AS exam_title, u.full_name AS student_name, u.identifier").
		Joins("JOIN exams e ON e.id = r.exam_id").Joins("JOIN users u ON u.id = r.student_id").
		Order("u.full_name ASC").Offset(page.Offset()).Limit(page.PerPage).Scan(&rows).Error
	return rows, total, err
}

func (r *Repository) ListScoped(ctx context.Context, role, userID string, filter ListFilter, page pagination.Request) ([]resultRow, int64, error) {
	query := r.db.WithContext(ctx).Table("results r").
		Joins("JOIN exams e ON e.id = r.exam_id").
		Joins("JOIN courses c ON c.id = e.course_id").
		Joins("JOIN users u ON u.id = r.student_id")
	if role == "teacher" {
		query = query.Joins("JOIN course_teachers ct ON ct.course_id = e.course_id AND ct.teacher_id = ?", userID)
	}
	if filter.CourseID != "" {
		query = query.Where("e.course_id = ?", filter.CourseID)
	}
	if filter.ExamID != "" {
		query = query.Where("r.exam_id = ?", filter.ExamID)
	}
	if filter.Search != "" {
		pattern := "%" + strings.TrimSpace(filter.Search) + "%"
		query = query.Where("e.title LIKE ? OR u.full_name LIKE ? OR u.identifier LIKE ?", pattern, pattern, pattern)
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Distinct("r.id").Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []resultRow
	err := query.Select("r.*, e.title AS exam_title, e.course_id AS course_id, c.name AS course_name, u.full_name AS student_name, u.identifier").
		Order("c.name ASC, e.title ASC, u.full_name ASC").Offset(page.Offset()).Limit(page.PerPage).Scan(&rows).Error
	return rows, total, err
}

func (r *Repository) ExportByExam(ctx context.Context, examID string) ([]resultRow, error) {
	var rows []resultRow
	err := r.db.WithContext(ctx).Table("results r").
		Select("r.*, e.title AS exam_title, u.full_name AS student_name, u.identifier").
		Joins("JOIN exams e ON e.id = r.exam_id").Joins("JOIN users u ON u.id = r.student_id").
		Where("r.exam_id = ?", examID).Order("u.full_name ASC").Scan(&rows).Error
	return rows, err
}

func (r *Repository) PublishByExam(ctx context.Context, examID string, now time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Model(&Result{}).Where("exam_id = ? AND status IN ?", examID, []string{"draft", "reviewed"}).Updates(map[string]any{"status": "published", "published_at": now})
	return result.RowsAffected, result.Error
}

func (r *Repository) ListPublishedForStudent(ctx context.Context, studentID string, page pagination.Request, searches ...string) ([]resultRow, int64, error) {
	query := r.db.WithContext(ctx).Table("results r").
		Joins("JOIN exams e ON e.id = r.exam_id").
		Joins("JOIN courses c ON c.id = e.course_id").
		Where("r.student_id = ? AND r.status = 'published'", studentID)
	if len(searches) > 0 && strings.TrimSpace(searches[0]) != "" {
		pattern := "%" + strings.TrimSpace(searches[0]) + "%"
		query = query.Where("e.title LIKE ? OR c.name LIKE ?", pattern, pattern)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []resultRow
	err := query.Select("r.*, e.title AS exam_title, e.course_id AS course_id, c.name AS course_name").
		Order("r.published_at DESC").Offset(page.Offset()).Limit(page.PerPage).Scan(&rows).Error
	return rows, total, err
}

func (r *Repository) FindPublishedForStudent(ctx context.Context, id, studentID string) (resultRow, error) {
	var row resultRow
	err := r.db.WithContext(ctx).Table("results r").Select("r.*, e.title AS exam_title").Joins("JOIN exams e ON e.id = r.exam_id").
		Where("r.id = ? AND r.student_id = ? AND r.status = 'published'", id, studentID).Scan(&row).Error
	if err != nil {
		return resultRow{}, err
	}
	if row.ID == "" {
		return resultRow{}, ErrNotFound
	}
	return row, nil
}
