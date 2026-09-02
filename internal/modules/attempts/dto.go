package attempts

import (
	"time"

	"lms-cn-api/internal/modules/exams"
)

type AvailableExamResponse struct {
	ID              string    `json:"id"`
	CourseID        string    `json:"course_id"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	StartsAt        time.Time `json:"starts_at"`
	EndsAt          time.Time `json:"ends_at"`
	DurationMinutes uint      `json:"duration_minutes"`
	AttemptStatus   *string   `json:"attempt_status"`
	ServerTime      time.Time `json:"server_time"`
}

type StartResponse struct {
	AttemptID           string                   `json:"attempt_id"`
	Status              string                   `json:"status"`
	StartedAt           time.Time                `json:"started_at"`
	DeadlineAt          time.Time                `json:"deadline_at"`
	ServerTime          time.Time                `json:"server_time"`
	AllowBackNavigation bool                     `json:"allow_back_navigation"`
	Questions           []exams.QuestionResponse `json:"questions"`
	Answers             []AnswerResponse         `json:"answers"`
	SubmissionReceipt   *string                  `json:"submission_receipt,omitempty"`
	SubmittedAt         *time.Time               `json:"submitted_at,omitempty"`
}

type SaveAnswerRequest struct {
	ExamQuestionID   string `json:"exam_question_id" binding:"required"`
	SelectedOptionID string `json:"selected_option_id" binding:"required"`
}

type AnswerResponse struct {
	ExamQuestionID   string    `json:"exam_question_id"`
	SelectedOptionID string    `json:"selected_option_id"`
	Revision         uint      `json:"revision"`
	SavedAt          time.Time `json:"saved_at"`
}

type ReceiptResponse struct {
	AttemptID   string     `json:"attempt_id"`
	Status      string     `json:"status"`
	Receipt     *string    `json:"receipt"`
	SubmittedAt *time.Time `json:"submitted_at"`
}
