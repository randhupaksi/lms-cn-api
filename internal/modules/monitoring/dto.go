package monitoring

import "time"

type ParticipantStatus struct {
	StudentID      string     `json:"student_id"`
	StudentName    string     `json:"student_name"`
	Identifier     string     `json:"identifier"`
	Status         string     `json:"status"`
	StartedAt      *time.Time `json:"started_at"`
	DeadlineAt     *time.Time `json:"deadline_at"`
	SubmittedAt    *time.Time `json:"submitted_at"`
	LastActivityAt *time.Time `json:"last_activity_at"`
	AnsweredCount  int64      `json:"answered_count"`
}

type Summary struct {
	ExamID       string              `json:"exam_id"`
	ServerTime   time.Time           `json:"server_time"`
	Total        int                 `json:"total"`
	NotStarted   int                 `json:"not_started"`
	InProgress   int                 `json:"in_progress"`
	Submitted    int                 `json:"submitted"`
	Expired      int                 `json:"expired"`
	Participants []ParticipantStatus `json:"participants"`
}
