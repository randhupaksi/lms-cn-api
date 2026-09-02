package results

import "time"

type Response struct {
	ID          string     `json:"id"`
	AttemptID   string     `json:"attempt_id"`
	ExamID      string     `json:"exam_id"`
	ExamTitle   string     `json:"exam_title"`
	StudentID   string     `json:"student_id"`
	StudentName string     `json:"student_name,omitempty"`
	Identifier  string     `json:"identifier,omitempty"`
	Status      string     `json:"status"`
	Score       float64    `json:"score"`
	MaxScore    float64    `json:"max_score"`
	Percentage  float64    `json:"percentage"`
	GradedAt    time.Time  `json:"graded_at"`
	PublishedAt *time.Time `json:"published_at"`
}

type resultRow struct {
	Result
	ExamTitle   string
	StudentName string
	Identifier  string
}

func toResponse(row resultRow, includeStudentIdentity bool) Response {
	percentage := float64(0)
	if row.MaxScore > 0 {
		percentage = row.Score / row.MaxScore * 100
	}
	result := Response{ID: row.ID, AttemptID: row.AttemptID, ExamID: row.ExamID, ExamTitle: row.ExamTitle, StudentID: row.StudentID, Status: row.Status, Score: row.Score, MaxScore: row.MaxScore, Percentage: percentage, GradedAt: row.GradedAt, PublishedAt: row.PublishedAt}
	if includeStudentIdentity {
		result.StudentName = row.StudentName
		result.Identifier = row.Identifier
	}
	return result
}
