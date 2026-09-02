package attempts

import "time"

const (
	StatusInProgress = "in_progress"
	StatusSubmitted  = "submitted"
	StatusExpired    = "expired"
)

type Attempt struct {
	ID                  string `gorm:"type:char(36);primaryKey"`
	ExamID              string `gorm:"type:char(36);not null"`
	StudentID           string `gorm:"type:char(36);not null"`
	Status              string `gorm:"size:20;not null"`
	StartIdempotencyKey string `gorm:"size:120;not null"`
	StartedAt           time.Time
	DeadlineAt          time.Time
	SubmittedAt         *time.Time
	SubmissionReceipt   *string  `gorm:"type:char(36)"`
	Answers             []Answer `gorm:"foreignKey:AttemptID"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type Answer struct {
	ID               string `gorm:"type:char(36);primaryKey"`
	AttemptID        string `gorm:"type:char(36);not null"`
	ExamQuestionID   string `gorm:"type:char(36);not null"`
	SelectedOptionID string `gorm:"type:char(36);not null"`
	Revision         uint
	SavedAt          time.Time
}

func (Answer) TableName() string { return "attempt_answers" }

type AnswerSaveRequest struct {
	AttemptID        string `gorm:"type:char(36);primaryKey"`
	IdempotencyKey   string `gorm:"size:120;primaryKey"`
	ExamQuestionID   string `gorm:"type:char(36);not null"`
	SelectedOptionID string `gorm:"type:char(36);not null"`
	AnswerID         string `gorm:"type:char(36);not null"`
	Revision         uint
	SavedAt          time.Time
	CreatedAt        time.Time
}

func (AnswerSaveRequest) TableName() string { return "attempt_answer_save_requests" }

type Event struct {
	ID        string `gorm:"type:char(36);primaryKey"`
	AttemptID string `gorm:"type:char(36);not null"`
	ActorID   string `gorm:"type:char(36);not null"`
	EventType string `gorm:"size:40;not null"`
	Metadata  []byte `gorm:"type:json"`
	CreatedAt time.Time
}

func (Event) TableName() string { return "attempt_events" }
