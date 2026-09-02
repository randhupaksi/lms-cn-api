package monitoring

import (
	"context"
	"errors"

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
