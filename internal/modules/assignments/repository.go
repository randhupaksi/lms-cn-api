package assignments

import (
	"context"
	"errors"
	"strings"
	"time"

	"lms-cn-api/pkg/pagination"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrNotFound = errors.New("assignment not found")
	ErrLocked   = errors.New("assignment locked")
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, assignment *Assignment) error {
	return r.db.WithContext(ctx).Create(assignment).Error
}

func (r *Repository) Find(ctx context.Context, id string) (Assignment, error) {
	var assignment Assignment
	if err := r.db.WithContext(ctx).First(&assignment, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Assignment{}, ErrNotFound
		}
		return Assignment{}, err
	}
	return assignment, nil
}

func (r *Repository) List(ctx context.Context, courseID, studentID string, publishedOnly bool) ([]assignmentRow, error) {
	query := r.db.WithContext(ctx).Table("assignments a").Select("a.*").Where("a.course_id = ?", courseID)
	if studentID != "" {
		query = query.Select(`a.*, s.id AS submission_id, s.student_id AS submission_student_id,
			s.content AS submission_content, s.attachment_url AS submission_attachment_url,
			s.status AS submission_status, s.score AS submission_score, s.feedback AS submission_feedback,
			s.submitted_at AS submission_submitted_at, s.graded_at AS submission_graded_at`).
			Joins("LEFT JOIN assignment_submissions s ON s.assignment_id = a.id AND s.student_id = ?", studentID)
	}
	if publishedOnly {
		query = query.Where("a.status IN ?", []string{StatusPublished, StatusClosed})
	}
	var rows []assignmentRow
	err := query.Order("a.due_at ASC, a.created_at DESC").Scan(&rows).Error
	return rows, err
}

func (r *Repository) ListPaged(ctx context.Context, courseID, role, userID string, page pagination.Request, filter ListFilter) ([]assignmentRow, int64, error) {
	query := r.db.WithContext(ctx).Table("assignments a").
		Select("a.*, c.name AS course_name").
		Joins("JOIN courses c ON c.id = a.course_id")
	switch role {
	case "teacher":
		query = query.Joins("JOIN course_teachers ct ON ct.course_id = a.course_id AND ct.teacher_id = ?", userID)
	case "student":
		query = query.Joins("JOIN course_students cs ON cs.course_id = a.course_id AND cs.student_id = ?", userID).
			Select(`a.*, c.name AS course_name, s.id AS submission_id, s.student_id AS submission_student_id,
				s.content AS submission_content, s.attachment_url AS submission_attachment_url,
				s.status AS submission_status, s.score AS submission_score, s.feedback AS submission_feedback,
				s.submitted_at AS submission_submitted_at, s.graded_at AS submission_graded_at`).
			Joins("LEFT JOIN assignment_submissions s ON s.assignment_id = a.id AND s.student_id = ?", userID).
			Where("a.status IN ?", []string{StatusPublished, StatusClosed})
	case "admin":
	default:
		return nil, 0, errors.New("unsupported assignment list role")
	}
	if courseID != "" {
		query = query.Where("a.course_id = ?", courseID)
	}
	if filter.Search != "" {
		pattern := "%" + strings.TrimSpace(filter.Search) + "%"
		query = query.Where("a.title LIKE ? OR a.instructions LIKE ?", pattern, pattern)
	}
	if filter.Status != "" {
		query = query.Where("a.status = ?", filter.Status)
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Distinct("a.id").Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []assignmentRow
	err := query.Order("c.name ASC, a.due_at ASC, a.created_at DESC").
		Offset(page.Offset()).Limit(page.PerPage).Scan(&rows).Error
	return rows, total, err
}

func (r *Repository) UpdateDraft(ctx context.Context, id string, apply func(*Assignment)) (Assignment, error) {
	var updated Assignment
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var assignment Assignment
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&assignment, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if assignment.Status != StatusDraft {
			return ErrLocked
		}
		apply(&assignment)
		if err := tx.Save(&assignment).Error; err != nil {
			return err
		}
		updated = assignment
		return nil
	})
	return updated, err
}

func (r *Repository) Publish(ctx context.Context, id string, now time.Time) error {
	result := r.db.WithContext(ctx).Model(&Assignment{}).Where("id = ? AND status = ?", id, StatusDraft).Updates(map[string]any{"status": StatusPublished, "published_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrLocked
	}
	return nil
}

func (r *Repository) Submit(ctx context.Context, submission *Submission) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "assignment_id"}, {Name: "student_id"}}, DoUpdates: clause.AssignmentColumns([]string{"content", "attachment_url", "status", "score", "feedback", "submitted_at", "graded_at", "graded_by", "updated_at"})}).Create(submission).Error
}

func (r *Repository) ListSubmissions(ctx context.Context, assignmentID string) ([]submissionRow, error) {
	var rows []submissionRow
	err := r.db.WithContext(ctx).Table("assignment_submissions s").Select("s.*, u.full_name AS student_name, u.identifier").Joins("JOIN users u ON u.id = s.student_id").Where("s.assignment_id = ?", assignmentID).Order("u.full_name ASC").Scan(&rows).Error
	return rows, err
}

func (r *Repository) ListSubmissionsPage(ctx context.Context, assignmentID, search string, page pagination.Request) ([]submissionRow, int64, error) {
	query := r.db.WithContext(ctx).Table("assignment_submissions s").
		Joins("JOIN users u ON u.id = s.student_id").
		Where("s.assignment_id = ?", assignmentID)
	if strings.TrimSpace(search) != "" {
		pattern := "%" + strings.TrimSpace(search) + "%"
		query = query.Where("u.full_name LIKE ? OR u.identifier LIKE ?", pattern, pattern)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []submissionRow
	err := query.Select("s.*, u.full_name AS student_name, u.identifier").
		Order("u.full_name ASC").Offset(page.Offset()).Limit(page.PerPage).Scan(&rows).Error
	return rows, total, err
}

func (r *Repository) Grade(ctx context.Context, submissionID, graderID string, score float64, feedback string, now time.Time) error {
	result := r.db.WithContext(ctx).Model(&Submission{}).Where("id = ?", submissionID).Updates(map[string]any{"status": "graded", "score": score, "feedback": feedback, "graded_at": now, "graded_by": graderID})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) SubmissionAssignment(ctx context.Context, submissionID string) (Assignment, error) {
	var assignment Assignment
	err := r.db.WithContext(ctx).Table("assignments a").Select("a.*").Joins("JOIN assignment_submissions s ON s.assignment_id = a.id").Where("s.id = ?", submissionID).Take(&assignment).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Assignment{}, ErrNotFound
	}
	return assignment, err
}
