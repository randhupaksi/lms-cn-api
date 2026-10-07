package monitoring

import (
	"context"
	"errors"
	"strings"
	"time"

	"lms-cn-api/pkg/pagination"

	"gorm.io/gorm"
)

var ErrExamNotFound = errors.New("exam not found")

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) ExamCourseID(ctx context.Context, examID string) (string, error) {
	var row struct{ CourseID string }
	if err := r.db.WithContext(ctx).Table("exams").Select("course_id").Where("id = ?", examID).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrExamNotFound
		}
		return "", err
	}
	return row.CourseID, nil
}

func (r *Repository) Participants(ctx context.Context, examID string) ([]ParticipantStatus, error) {
	var rows []ParticipantStatus
	err := r.db.WithContext(ctx).Table("exam_participants ep").
		Select(`ep.student_id, u.full_name AS student_name, u.identifier,
			COALESCE(a.status, 'not_started') AS status, a.started_at, a.deadline_at, a.submitted_at,
			MAX(aa.saved_at) AS last_activity_at, COUNT(DISTINCT aa.exam_question_id) AS answered_count`).
		Joins("JOIN users u ON u.id = ep.student_id").
		Joins("LEFT JOIN attempts a ON a.exam_id = ep.exam_id AND a.student_id = ep.student_id").
		Joins("LEFT JOIN attempt_answers aa ON aa.attempt_id = a.id").
		Where("ep.exam_id = ?", examID).
		Group("ep.student_id, u.full_name, u.identifier, a.status, a.started_at, a.deadline_at, a.submitted_at").
		Order("u.full_name ASC").Scan(&rows).Error
	return rows, err
}

type participantCounts struct {
	Total      int
	NotStarted int
	InProgress int
	Submitted  int
	Expired    int
}

func (r *Repository) ParticipantCounts(ctx context.Context, examID string, now time.Time) (participantCounts, error) {
	var result participantCounts
	err := r.db.WithContext(ctx).Table("exam_participants ep").
		Select(`COUNT(*) AS total,
			COALESCE(SUM(CASE WHEN a.id IS NULL THEN 1 ELSE 0 END), 0) AS not_started,
			COALESCE(SUM(CASE WHEN a.status = 'in_progress' AND (a.deadline_at IS NULL OR a.deadline_at > ?) THEN 1 ELSE 0 END), 0) AS in_progress,
			COALESCE(SUM(CASE WHEN a.status = 'submitted' THEN 1 ELSE 0 END), 0) AS submitted,
			COALESCE(SUM(CASE WHEN a.status = 'expired' OR (a.status = 'in_progress' AND a.deadline_at <= ?) THEN 1 ELSE 0 END), 0) AS expired`, now, now).
		Joins("LEFT JOIN attempts a ON a.exam_id = ep.exam_id AND a.student_id = ep.student_id").
		Where("ep.exam_id = ?", examID).Scan(&result).Error
	return result, err
}

func (r *Repository) ParticipantsPage(ctx context.Context, examID, search string, page pagination.Request) ([]ParticipantStatus, int64, error) {
	query := r.db.WithContext(ctx).Table("exam_participants ep").
		Select(`ep.student_id, u.full_name AS student_name, u.identifier,
			COALESCE(a.status, 'not_started') AS status, a.started_at, a.deadline_at, a.submitted_at,
			MAX(aa.saved_at) AS last_activity_at, COUNT(DISTINCT aa.exam_question_id) AS answered_count`).
		Joins("JOIN users u ON u.id = ep.student_id").
		Joins("LEFT JOIN attempts a ON a.exam_id = ep.exam_id AND a.student_id = ep.student_id").
		Joins("LEFT JOIN attempt_answers aa ON aa.attempt_id = a.id").
		Where("ep.exam_id = ?", examID)
	if strings.TrimSpace(search) != "" {
		pattern := "%" + strings.TrimSpace(search) + "%"
		query = query.Where("u.full_name LIKE ? OR u.identifier LIKE ?", pattern, pattern)
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Distinct("ep.student_id").Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []ParticipantStatus
	err := query.Group("ep.student_id, u.full_name, u.identifier, a.status, a.started_at, a.deadline_at, a.submitted_at").
		Order("u.full_name ASC").Offset(page.Offset()).Limit(page.PerPage).Scan(&rows).Error
	return rows, total, err
}
