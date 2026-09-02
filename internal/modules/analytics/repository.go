package analytics

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

var ErrExamNotFound = errors.New("exam not found")

type Repository struct{ db *gorm.DB }

type adminDashboardCounts struct {
	ActiveUsers    int64
	ActiveCourses  int64
	PublishedExams int64
	ActiveAttempts int64
}

type teacherDashboardCounts struct {
	Courses            int64
	Questions          int64
	Exams              int64
	UnpublishedResults int64
}

type studentDashboardCounts struct {
	Courses            int64
	AvailableExams     int64
	PublishedResults   int64
	CompletedMaterials int64
}

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) CourseID(ctx context.Context, examID string) (string, error) {
	var row struct{ CourseID string }
	if err := r.db.WithContext(ctx).Table("exams").Select("course_id").Where("id = ?", examID).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrExamNotFound
		}
		return "", err
	}
	return row.CourseID, nil
}

func (r *Repository) AdminDashboardCounts(ctx context.Context, now time.Time) (adminDashboardCounts, error) {
	counts := adminDashboardCounts{}
	queries := []struct {
		table string
		where string
		args  []any
		value *int64
	}{
		{"users", "status = 'active'", nil, &counts.ActiveUsers},
		{"courses", "status = 'active'", nil, &counts.ActiveCourses},
		{"exams", "status = 'published'", nil, &counts.PublishedExams},
		{"attempts", "status = 'in_progress' AND deadline_at > ?", []any{now}, &counts.ActiveAttempts},
	}
	for _, query := range queries {
		if err := r.db.WithContext(ctx).Table(query.table).Where(query.where, query.args...).Count(query.value).Error; err != nil {
			return adminDashboardCounts{}, err
		}
	}
	return counts, nil
}

func (r *Repository) TeacherDashboardCounts(ctx context.Context, teacherID string) (teacherDashboardCounts, error) {
	counts := teacherDashboardCounts{}
	queries := []struct {
		query *gorm.DB
		value *int64
	}{
		{r.db.WithContext(ctx).Table("course_teachers").Where("teacher_id = ?", teacherID), &counts.Courses},
		{r.db.WithContext(ctx).Table("questions q").Joins("JOIN course_teachers ct ON ct.course_id = q.course_id").Where("ct.teacher_id = ? AND q.status = 'active'", teacherID).Distinct("q.id"), &counts.Questions},
		{r.db.WithContext(ctx).Table("exams e").Joins("JOIN course_teachers ct ON ct.course_id = e.course_id").Where("ct.teacher_id = ?", teacherID).Distinct("e.id"), &counts.Exams},
		{r.db.WithContext(ctx).Table("results r").Joins("JOIN exams e ON e.id = r.exam_id").Joins("JOIN course_teachers ct ON ct.course_id = e.course_id").Where("ct.teacher_id = ? AND r.status <> 'published'", teacherID).Distinct("r.id"), &counts.UnpublishedResults},
	}
	for _, query := range queries {
		if err := query.query.Count(query.value).Error; err != nil {
			return teacherDashboardCounts{}, err
		}
	}
	return counts, nil
}

func (r *Repository) StudentDashboardCounts(ctx context.Context, studentID string, now time.Time) (studentDashboardCounts, error) {
	counts := studentDashboardCounts{}
	queries := []struct {
		query *gorm.DB
		value *int64
	}{
		{r.db.WithContext(ctx).Table("course_students").Where("student_id = ?", studentID), &counts.Courses},
		{r.db.WithContext(ctx).Table("exam_participants ep").Joins("JOIN exams e ON e.id = ep.exam_id").Where("ep.student_id = ? AND e.status = 'published' AND e.starts_at <= ? AND e.ends_at > ?", studentID, now, now).Distinct("e.id"), &counts.AvailableExams},
		{r.db.WithContext(ctx).Table("results").Where("student_id = ? AND status = 'published'", studentID), &counts.PublishedResults},
		{r.db.WithContext(ctx).Table("material_progress").Where("student_id = ?", studentID), &counts.CompletedMaterials},
	}
	for _, query := range queries {
		if err := query.query.Count(query.value).Error; err != nil {
			return studentDashboardCounts{}, err
		}
	}
	return counts, nil
}

func (r *Repository) ExamSummary(ctx context.Context, examID string) (ExamSummary, error) {
	var summary ExamSummary
	summary.ExamID = examID
	if err := r.db.WithContext(ctx).Table("exam_participants").Where("exam_id = ?", examID).Count(&summary.ParticipantCount).Error; err != nil {
		return ExamSummary{}, err
	}
	if err := r.db.WithContext(ctx).Table("attempts").Where("exam_id = ?", examID).Count(&summary.StartedCount).Error; err != nil {
		return ExamSummary{}, err
	}
	if err := r.db.WithContext(ctx).Table("attempts").Where("exam_id = ? AND status = 'submitted'", examID).Count(&summary.SubmittedCount).Error; err != nil {
		return ExamSummary{}, err
	}
	if err := r.db.WithContext(ctx).Table("attempts").Where("exam_id = ? AND status = 'expired'", examID).Count(&summary.ExpiredCount).Error; err != nil {
		return ExamSummary{}, err
	}
	var scores struct {
		Average        float64
		Highest        float64
		Lowest         float64
		AveragePercent float64
	}
	if err := r.db.WithContext(ctx).Table("results").Select("COALESCE(AVG(score),0) AS average, COALESCE(MAX(score),0) AS highest, COALESCE(MIN(score),0) AS lowest, COALESCE(AVG(CASE WHEN max_score > 0 THEN score / max_score * 100 ELSE 0 END),0) AS average_percent").Where("exam_id = ?", examID).Scan(&scores).Error; err != nil {
		return ExamSummary{}, err
	}
	summary.AverageScore, summary.HighestScore, summary.LowestScore, summary.AveragePercent = scores.Average, scores.Highest, scores.Lowest, scores.AveragePercent
	return summary, nil
}

func (r *Repository) ItemAnalysis(ctx context.Context, examID string) ([]ItemAnalysis, error) {
	var rows []ItemAnalysis
	err := r.db.WithContext(ctx).Table("exam_questions q").
		Select(`q.id AS question_id, q.stem, COUNT(DISTINCT a.id) AS answered_count,
			SUM(CASE WHEN o.is_correct = TRUE THEN 1 ELSE 0 END) AS correct_count`).
		Joins("LEFT JOIN attempts at ON at.exam_id = q.exam_id AND at.status IN ('submitted', 'expired')").
		Joins("LEFT JOIN attempt_answers a ON a.exam_question_id = q.id AND a.attempt_id = at.id").
		Joins("LEFT JOIN exam_question_options o ON o.id = a.selected_option_id").
		Where("q.exam_id = ?", examID).Group("q.id, q.stem, q.position").Order("q.position ASC").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for index := range rows {
		if rows[index].AnsweredCount > 0 {
			rows[index].Accuracy = float64(rows[index].CorrectCount) / float64(rows[index].AnsweredCount) * 100
		}
	}
	return rows, nil
}
