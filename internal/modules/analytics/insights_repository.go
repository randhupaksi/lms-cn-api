package analytics

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// courseColumn is a repository-owned SQL identifier, never request input.
// EXISTS avoids multiplying aggregates when a course has multiple managers.
func insightCourseScope(query *gorm.DB, role, userID, courseColumn string) *gorm.DB {
	switch role {
	case "admin", "superadmin":
		return query
	case "teacher":
		return query.Where("EXISTS (SELECT 1 FROM course_teachers scope_member WHERE scope_member.course_id = "+courseColumn+" AND scope_member.teacher_id = ?)", userID)
	case "student":
		return query.Where("EXISTS (SELECT 1 FROM course_students scope_member WHERE scope_member.course_id = "+courseColumn+" AND scope_member.student_id = ?)", userID)
	default:
		return query.Where("1 = 0")
	}
}

func (r *Repository) InsightMetrics(ctx context.Context, role, userID string) ([]Metric, error) {
	var metrics []Metric
	queries := []*gorm.DB{}
	if role == "admin" || role == "superadmin" {
		metrics = []Metric{{Key: "teachers", Label: "Guru aktif"}, {Key: "students", Label: "Siswa aktif"}, {Key: "pending_grading", Label: "Pengumpulan perlu dinilai"}}
		queries = append(queries, r.db.WithContext(ctx).Table("users").Where("role = ? AND status = ?", "teacher", "active"), r.db.WithContext(ctx).Table("users").Where("role = ? AND status = ?", "student", "active"))
	} else {
		metrics = []Metric{{Key: "published_materials", Label: "Materi dipublikasikan"}, {Key: "published_assignments", Label: "Tugas dipublikasikan"}, {Key: "pending_grading", Label: "Pengumpulan perlu dinilai"}}
		queries = append(queries, insightCourseScope(r.db.WithContext(ctx).Table("course_materials m").Where("m.status = ?", "published"), role, userID, "m.course_id"), insightCourseScope(r.db.WithContext(ctx).Table("assignments a").Where("a.status = ?", "published"), role, userID, "a.course_id"))
	}
	submissions := r.db.WithContext(ctx).Table("assignment_submissions s").Joins("JOIN assignments a ON a.id = s.assignment_id")
	if role == "student" {
		metrics[2] = Metric{Key: "submitted_assignments", Label: "Tugas sudah dikumpulkan"}
		submissions = insightCourseScope(submissions.Where("s.student_id = ? AND a.status = ?", userID, "published"), role, userID, "a.course_id")
	} else {
		submissions = insightCourseScope(submissions.Where("s.status = ?", "submitted"), role, userID, "a.course_id")
	}
	queries = append(queries, submissions)
	for i, query := range queries {
		var total int64
		if err := query.Count(&total).Error; err != nil {
			return nil, err
		}
		metrics[i].Value = float64(total)
	}
	return metrics, nil
}

func (r *Repository) InsightActivity(ctx context.Context, role, userID string, start, end time.Time) ([]activityCount, []activityCount, error) {
	exams := r.db.WithContext(ctx).Table("attempts at").Joins("JOIN exams e ON e.id = at.exam_id").Where("at.status IN ? AND at.submitted_at >= ? AND at.submitted_at <= ?", []string{"submitted", "expired"}, start, end)
	assignments := r.db.WithContext(ctx).Table("assignment_submissions s").Joins("JOIN assignments a ON a.id = s.assignment_id").Where("s.submitted_at >= ? AND s.submitted_at <= ?", start, end)
	if role == "student" {
		exams = exams.Where("at.student_id = ?", userID)
		assignments = assignments.Where("s.student_id = ?", userID)
	} else {
		exams = insightCourseScope(exams, role, userID, "e.course_id")
		assignments = insightCourseScope(assignments, role, userID, "a.course_id")
	}
	// UTC storage + fixed WIB offset avoids depending on MySQL timezone tables.
	examRows, assignmentRows := []activityCount{}, []activityCount{}
	if err := exams.Select("DATE_FORMAT(DATE_ADD(at.submitted_at, INTERVAL 7 HOUR), '%Y-%m-%d') AS day, COUNT(*) AS total").Group("day").Order("day").Scan(&examRows).Error; err != nil {
		return nil, nil, err
	}
	if err := assignments.Select("DATE_FORMAT(DATE_ADD(s.submitted_at, INTERVAL 7 HOUR), '%Y-%m-%d') AS day, COUNT(*) AS total").Group("day").Order("day").Scan(&assignmentRows).Error; err != nil {
		return nil, nil, err
	}
	return examRows, assignmentRows, nil
}

func (r *Repository) InsightDistribution(ctx context.Context, role, userID string, now time.Time) (map[string]int64, error) {
	var query *gorm.DB
	if role == "student" {
		query = insightCourseScope(r.db.WithContext(ctx).Table("assignments a").Joins("LEFT JOIN assignment_submissions s ON s.assignment_id = a.id AND s.student_id = ?", userID).Where("a.status = ?", "published"), role, userID, "a.course_id").Select("CASE WHEN s.id IS NOT NULL THEN s.status WHEN a.due_at <= ? THEN 'overdue' ELSE 'pending' END AS status_key, COUNT(*) AS total", now)
	} else {
		query = insightCourseScope(r.db.WithContext(ctx).Table("exams e").Where("e.status IN ?", []string{"draft", "published"}), role, userID, "e.course_id").Select("CASE WHEN e.status = 'draft' THEN 'draft' WHEN e.ends_at <= ? THEN 'ended' WHEN e.starts_at > ? THEN 'scheduled' ELSE 'ongoing' END AS status_key, COUNT(*) AS total", now, now)
	}
	var rows []struct {
		StatusKey string
		Total     int64
	}
	if err := query.Group("status_key").Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]int64, len(rows))
	for _, row := range rows {
		result[row.StatusKey] = row.Total
	}
	return result, nil
}

func (r *Repository) InsightLearningProgress(ctx context.Context, studentID string) (LearningProgress, error) {
	var progress LearningProgress
	err := insightCourseScope(r.db.WithContext(ctx).Table("course_materials m").Joins("LEFT JOIN material_progress p ON p.material_id = m.id AND p.student_id = ?", studentID).Where("m.status = ?", "published"), "student", studentID, "m.course_id").Select("COUNT(*) AS total, COUNT(p.material_id) AS completed").Scan(&progress).Error
	return progress, err
}

func (r *Repository) InsightRecentResults(ctx context.Context, studentID string) ([]RecentResult, error) {
	rows := []RecentResult{}
	err := r.db.WithContext(ctx).Table("results r").Joins("JOIN exams e ON e.id = r.exam_id").Joins("JOIN courses c ON c.id = e.course_id").Where("r.student_id = ? AND r.status = ? AND r.max_score > 0", studentID, "published").Select("r.id, e.title, c.name AS course_name, r.score / r.max_score * 100 AS percentage, r.published_at").Order("r.published_at DESC, r.id DESC").Limit(5).Scan(&rows).Error
	return rows, err
}
