package governance

import (
	"context"
	"gorm.io/gorm"
	"time"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

const examSelect = `e.id, e.title, e.course_id, c.name AS course_name, cg.name AS class_name,
 e.status, e.starts_at, e.ends_at, e.duration_minutes,
 (SELECT COUNT(*) FROM exam_questions q WHERE q.exam_id = e.id) AS question_count,
 (SELECT COUNT(*) FROM exam_participants p WHERE p.exam_id = e.id) AS participant_count,
 (SELECT COUNT(*) FROM exam_participants p JOIN users u ON u.id=p.student_id WHERE p.exam_id=e.id AND u.status<>'active') AS inactive_participants,
 (SELECT COUNT(*) FROM course_teachers ct JOIN users u ON u.id=ct.teacher_id WHERE ct.course_id=e.course_id AND u.status='active') AS teacher_count,
 (SELECT COUNT(*) FROM attempts a WHERE a.exam_id=e.id) AS attempt_count,
 (SELECT COUNT(*) FROM exams other JOIN courses oc ON oc.id=other.course_id WHERE other.id<>e.id AND other.status='published' AND oc.class_group_id=c.class_group_id AND other.starts_at<e.ends_at AND other.ends_at>e.starts_at) AS conflict_count`

func (r *Repository) Exams(ctx context.Context, page, size int, search, role, userID string) ([]ExamReadiness, int64, error) {
	q := r.db.WithContext(ctx).Table("exams e").Joins("JOIN courses c ON c.id=e.course_id").Joins("JOIN class_groups cg ON cg.id=c.class_group_id")
	if role == "teacher" {
		q = q.Where("EXISTS (SELECT 1 FROM course_teachers ct WHERE ct.course_id=e.course_id AND ct.teacher_id=?)", userID)
	}
	if search != "" {
		q = q.Where("e.title LIKE ? OR c.name LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := []ExamReadiness{}
	err := q.Select(examSelect).Order("e.starts_at DESC, e.id").Offset((page - 1) * size).Limit(size).Scan(&rows).Error
	return rows, total, err
}
func (r *Repository) Exam(ctx context.Context, id string) (ExamReadiness, error) {
	var row ExamReadiness
	err := r.db.WithContext(ctx).Table("exams e").Joins("JOIN courses c ON c.id=e.course_id").Joins("JOIN class_groups cg ON cg.id=c.class_group_id").Select(examSelect).Where("e.id=?", id).Take(&row).Error
	return row, err
}

type StateCount struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

func (r *Repository) UpcomingExams(ctx context.Context, now time.Time) ([]ExamReadiness, error) {
	rows := []ExamReadiness{}
	err := r.db.WithContext(ctx).Table("exams e").Joins("JOIN courses c ON c.id=e.course_id").Joins("JOIN class_groups cg ON cg.id=c.class_group_id").Select(examSelect).Where("e.status='published' AND e.ends_at>?", now).Order("e.starts_at,e.id").Limit(6).Scan(&rows).Error
	return rows, err
}

func (r *Repository) Overview(ctx context.Context, now time.Time) (Overview, []StateCount, error) {
	result := Overview{GeneratedAt: now, Roles: []RoleCount{}}
	err := r.db.WithContext(ctx).Raw(`SELECT (SELECT COUNT(*) FROM courses WHERE status='active') AS courses,
 (SELECT COUNT(*) FROM class_groups) AS classes,
 (SELECT COUNT(*) FROM attempts WHERE status='in_progress') AS active_attempts,
 (SELECT COUNT(*) FROM governance_requests WHERE status='pending') AS pending_requests,
 (SELECT COUNT(*) FROM results WHERE status<>'published') AS draft_results`).Scan(&result).Error
	if err != nil {
		return result, nil, err
	}
	if err = r.db.WithContext(ctx).Table("users").Select("role, SUM(status='active') AS active, SUM(status='inactive') AS inactive").Group("role").Scan(&result.Roles).Error; err != nil {
		return result, nil, err
	}
	states := []StateCount{}
	err = r.db.WithContext(ctx).Table("exams").Select("CASE WHEN status='draft' THEN 'draft' WHEN status IN ('closed','archived') OR ends_at<=? THEN 'ended' WHEN starts_at>? THEN 'scheduled' ELSE 'running' END AS `key`, COUNT(*) AS count", now, now).Group("`key`").Scan(&states).Error
	return result, states, err
}

func (r *Repository) Quality(ctx context.Context) ([]QualityGroup, error) {
	definitions := []struct{ key, title, description, from, where, label string }{
		{"students_without_courses", "Siswa belum memiliki course", "Hubungkan siswa aktif ke course melalui pengelolaan akademik.", "users u", "u.role='student' AND u.status='active' AND NOT EXISTS (SELECT 1 FROM course_students cs WHERE cs.student_id=u.id)", "u.full_name"},
		{"teachers_without_courses", "Guru belum ditugaskan", "Tinjau guru aktif yang belum memiliki penugasan mengajar.", "users u", "u.role='teacher' AND u.status='active' AND NOT EXISTS (SELECT 1 FROM course_teachers ct WHERE ct.teacher_id=u.id)", "u.full_name"},
		{"courses_without_teachers", "Course tanpa guru aktif", "Tetapkan minimal satu guru aktif sebelum menyiapkan ujian.", "courses u", "u.status='active' AND NOT EXISTS (SELECT 1 FROM course_teachers ct JOIN users t ON t.id=ct.teacher_id WHERE ct.course_id=u.id AND t.status='active')", "u.name"},
		{"courses_without_students", "Course belum memiliki siswa", "Tinjau kelas dan daftar peserta course.", "courses u", "u.status='active' AND NOT EXISTS (SELECT 1 FROM course_students cs WHERE cs.course_id=u.id)", "u.name"},
		{"exams_without_questions", "Ujian tanpa soal", "Lengkapi bank soal dan konfigurasi ujian bersama guru.", "exams u", "u.status='draft' AND NOT EXISTS (SELECT 1 FROM exam_questions q WHERE q.exam_id=u.id)", "u.title"},
		{"exams_without_participants", "Ujian belum memiliki peserta", "Tetapkan peserta sebelum publikasi ujian.", "exams u", "u.status='draft' AND NOT EXISTS (SELECT 1 FROM exam_participants p WHERE p.exam_id=u.id)", "u.title"},
	}
	groups := make([]QualityGroup, 0, len(definitions))
	for _, d := range definitions {
		g := QualityGroup{Key: d.key, Title: d.title, Description: d.description, Samples: []Finding{}}
		q := r.db.WithContext(ctx).Table(d.from).Where(d.where)
		if err := q.Count(&g.Count).Error; err != nil {
			return nil, err
		}
		if err := q.Select("u.id, " + d.label + " AS label").Order(d.label).Limit(8).Scan(&g.Samples).Error; err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, nil
}

func (r *Repository) Requests(ctx context.Context, role, userID, status string, page, size int) ([]RequestRow, int64, error) {
	q := r.db.WithContext(ctx).Table("governance_requests g").Joins("JOIN exams e ON e.id=g.exam_id").Joins("JOIN courses c ON c.id=e.course_id").Joins("JOIN users u ON u.id=g.requester_id").Joins("LEFT JOIN users v ON v.id=g.reviewer_id")
	if role != "superadmin" {
		q = q.Where("g.requester_id=?", userID)
	}
	if status != "" {
		q = q.Where("g.status=?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := []RequestRow{}
	err := q.Select("g.id,g.exam_id,g.requester_id,g.reason,g.proposed_starts_at,g.proposed_ends_at,g.original_starts_at,g.original_ends_at,g.status,g.reviewer_id,g.decision_note,g.decided_at,g.created_at,e.title AS exam_title,c.name AS course_name,u.full_name AS requester_name,COALESCE(v.full_name,'') AS reviewer_name").Order("g.created_at DESC, g.id").Offset((page - 1) * size).Limit(size).Scan(&rows).Error
	return rows, total, err
}
func (r *Repository) Events(ctx context.Context, from, to time.Time) ([]Event, error) {
	rows := []Event{}
	err := r.db.WithContext(ctx).Where("status='active' AND starts_at < ? AND ends_at > ?", to, from).Order("starts_at, id").Limit(500).Find(&rows).Error
	return rows, err
}
func (r *Repository) ExamAgenda(ctx context.Context, from, to time.Time) ([]Event, error) {
	rows := []Event{}
	err := r.db.WithContext(ctx).Table("exams").Select("id,title,description,'exam' AS category,starts_at,ends_at,status,0 AS version").Where("status='published' AND starts_at < ? AND ends_at > ?", to, from).Order("starts_at,id").Limit(500).Scan(&rows).Error
	return rows, err
}
func (r *Repository) Settings(ctx context.Context) (Settings, error) {
	var row Settings
	err := r.db.WithContext(ctx).First(&row, 1).Error
	return row, err
}

type ChangeRow struct {
	SettingChange
	ActorName string `json:"actor_name"`
}

func (r *Repository) History(ctx context.Context) ([]ChangeRow, error) {
	rows := []ChangeRow{}
	err := r.db.WithContext(ctx).Table("school_setting_changes s").Select("s.*,u.full_name AS actor_name").Joins("JOIN users u ON u.id=s.actor_id").Order("s.created_at DESC").Limit(50).Scan(&rows).Error
	return rows, err
}
func (r *Repository) Reports(ctx context.Context, year, classID string, page, size int) ([]CourseReport, int64, error) {
	q := r.db.WithContext(ctx).Table("courses c").Joins("JOIN class_groups g ON g.id=c.class_group_id").Joins("JOIN academic_years y ON y.id=c.academic_year_id")
	if year != "" {
		q = q.Where("c.academic_year_id=?", year)
	}
	if classID != "" {
		q = q.Where("c.class_group_id=?", classID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := []CourseReport{}
	err := q.Select(`c.id,c.name,g.name AS class_name,y.name AS year_name,
 (SELECT COUNT(*) FROM course_students s WHERE s.course_id=c.id) AS students,
 (SELECT COUNT(*) FROM course_teachers t WHERE t.course_id=c.id) AS teachers,
 (SELECT COUNT(*) FROM exams e WHERE e.course_id=c.id) AS exams,
 (SELECT COUNT(*) FROM attempts a JOIN exams e ON e.id=a.exam_id WHERE e.course_id=c.id AND a.status<>'in_progress') AS submitted_attempts,
 (SELECT COUNT(*) FROM results r JOIN exams e ON e.id=r.exam_id WHERE e.course_id=c.id AND r.status='published') AS published_results,
 (SELECT AVG(r.score/NULLIF(r.max_score,0)*100) FROM results r JOIN exams e ON e.id=r.exam_id WHERE e.course_id=c.id AND r.status='published') AS average_score,
 (SELECT COUNT(*) FROM results r JOIN exams e ON e.id=r.exam_id WHERE e.course_id=c.id AND r.status='published' AND r.max_score>0 AND r.score/r.max_score*100<(SELECT passing_reference FROM school_settings WHERE id=1)) AS below_reference,
 (SELECT passing_reference FROM school_settings WHERE id=1) AS passing_reference,
 (SELECT COUNT(*) FROM assignments a WHERE a.course_id=c.id) AS assignments,
 (SELECT COUNT(*) FROM assignment_submissions s JOIN assignments a ON a.id=s.assignment_id WHERE a.course_id=c.id AND s.status='submitted') AS pending_grading`).Order("y.starts_on DESC,g.name,c.name,c.id").Offset((page - 1) * size).Limit(size).Scan(&rows).Error
	return rows, total, err
}
