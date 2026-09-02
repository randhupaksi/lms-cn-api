package assignments

import "time"

type WriteRequest struct {
	CourseID     string    `json:"course_id" binding:"required"`
	Title        string    `json:"title" binding:"required,max=180"`
	Instructions string    `json:"instructions" binding:"required,max=20000"`
	DueAt        time.Time `json:"due_at" binding:"required"`
	MaxScore     float64   `json:"max_score" binding:"required,gt=0,lte=10000"`
}

type SubmitRequest struct {
	Content       string `json:"content" binding:"required,max=30000"`
	AttachmentURL string `json:"attachment_url" binding:"omitempty,url,max=1000"`
}

type GradeRequest struct {
	Score    float64 `json:"score" binding:"gte=0"`
	Feedback string  `json:"feedback" binding:"max=10000"`
}

type Response struct {
	ID           string              `json:"id"`
	CourseID     string              `json:"course_id"`
	AuthorID     string              `json:"author_id"`
	Title        string              `json:"title"`
	Instructions string              `json:"instructions"`
	DueAt        time.Time           `json:"due_at"`
	MaxScore     float64             `json:"max_score"`
	Status       string              `json:"status"`
	PublishedAt  *time.Time          `json:"published_at"`
	Submission   *SubmissionResponse `json:"submission,omitempty"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
}

type SubmissionResponse struct {
	ID            string     `json:"id"`
	AssignmentID  string     `json:"assignment_id"`
	StudentID     string     `json:"student_id"`
	StudentName   string     `json:"student_name,omitempty"`
	Identifier    string     `json:"identifier,omitempty"`
	Content       string     `json:"content"`
	AttachmentURL string     `json:"attachment_url"`
	Status        string     `json:"status"`
	Score         *float64   `json:"score"`
	Feedback      string     `json:"feedback"`
	SubmittedAt   time.Time  `json:"submitted_at"`
	GradedAt      *time.Time `json:"graded_at"`
}

type assignmentRow struct {
	Assignment
	SubmissionID            *string
	SubmissionStudentID     *string
	SubmissionContent       *string
	SubmissionAttachmentURL *string
	SubmissionStatus        *string
	SubmissionScore         *float64
	SubmissionFeedback      *string
	SubmissionSubmittedAt   *time.Time
	SubmissionGradedAt      *time.Time
}

type submissionRow struct {
	Submission
	StudentName string
	Identifier  string
}

func toResponse(row assignmentRow) Response {
	result := Response{ID: row.ID, CourseID: row.CourseID, AuthorID: row.AuthorID, Title: row.Title, Instructions: row.Instructions, DueAt: row.DueAt, MaxScore: row.MaxScore, Status: row.Status, PublishedAt: row.PublishedAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
	if row.SubmissionID != nil && row.SubmissionSubmittedAt != nil {
		result.Submission = &SubmissionResponse{ID: *row.SubmissionID, AssignmentID: row.ID, StudentID: value(row.SubmissionStudentID), Content: value(row.SubmissionContent), AttachmentURL: value(row.SubmissionAttachmentURL), Status: value(row.SubmissionStatus), Score: row.SubmissionScore, Feedback: value(row.SubmissionFeedback), SubmittedAt: *row.SubmissionSubmittedAt, GradedAt: row.SubmissionGradedAt}
	}
	return result
}

func toSubmissionResponse(row submissionRow) SubmissionResponse {
	return SubmissionResponse{ID: row.ID, AssignmentID: row.AssignmentID, StudentID: row.StudentID, StudentName: row.StudentName, Identifier: row.Identifier, Content: row.Content, AttachmentURL: row.AttachmentURL, Status: row.Status, Score: row.Score, Feedback: row.Feedback, SubmittedAt: row.SubmittedAt, GradedAt: row.GradedAt}
}

func value(pointer *string) string {
	if pointer == nil {
		return ""
	}
	return *pointer
}
