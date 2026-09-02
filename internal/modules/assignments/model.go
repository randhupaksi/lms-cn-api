package assignments

import "time"

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusClosed    = "closed"
)

type Assignment struct {
	ID           string `gorm:"type:char(36);primaryKey"`
	CourseID     string `gorm:"type:char(36);not null"`
	AuthorID     string `gorm:"type:char(36);not null"`
	Title        string `gorm:"size:180;not null"`
	Instructions string `gorm:"type:text;not null"`
	DueAt        time.Time
	MaxScore     float64
	Status       string `gorm:"size:20;not null"`
	PublishedAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Submission struct {
	ID            string `gorm:"type:char(36);primaryKey"`
	AssignmentID  string `gorm:"type:char(36);not null"`
	StudentID     string `gorm:"type:char(36);not null"`
	Content       string `gorm:"type:text;not null"`
	AttachmentURL string `gorm:"size:1000"`
	Status        string `gorm:"size:20;not null"`
	Score         *float64
	Feedback      string `gorm:"type:text"`
	SubmittedAt   time.Time
	GradedAt      *time.Time
	GradedBy      *string `gorm:"type:char(36)"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (Submission) TableName() string { return "assignment_submissions" }
