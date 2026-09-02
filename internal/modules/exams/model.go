package exams

import "time"

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusClosed    = "closed"
)

type Exam struct {
	ID                  string `gorm:"type:char(36);primaryKey"`
	CourseID            string `gorm:"type:char(36);not null"`
	AuthorID            string `gorm:"type:char(36);not null"`
	Title               string `gorm:"size:180;not null"`
	Description         string `gorm:"type:text"`
	Status              string `gorm:"size:20;not null"`
	StartsAt            time.Time
	EndsAt              time.Time
	DurationMinutes     uint
	MaxAttempts         uint
	AllowBackNavigation bool
	RandomizeQuestions  bool
	RandomizeOptions    bool
	ResultPolicy        string `gorm:"size:24;not null"`
	PublishedAt         *time.Time
	Questions           []ExamQuestion `gorm:"foreignKey:ExamID"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type ExamQuestion struct {
	ID               string `gorm:"type:char(36);primaryKey"`
	ExamID           string `gorm:"type:char(36);not null"`
	SourceQuestionID string `gorm:"type:char(36);not null"`
	SourceVersion    uint
	Type             string `gorm:"size:32;not null"`
	Stem             string `gorm:"type:text;not null"`
	Position         uint
	Points           float64
	Options          []ExamQuestionOption `gorm:"foreignKey:ExamQuestionID"`
	CreatedAt        time.Time
}

func (ExamQuestion) TableName() string { return "exam_questions" }

type ExamQuestionOption struct {
	ID             string `gorm:"type:char(36);primaryKey"`
	ExamQuestionID string `gorm:"type:char(36);not null"`
	SourceOptionID string `gorm:"type:char(36);not null"`
	Content        string `gorm:"type:text;not null"`
	IsCorrect      bool
	Position       uint
}

func (ExamQuestionOption) TableName() string { return "exam_question_options" }

type Participant struct {
	ExamID     string `gorm:"type:char(36);primaryKey"`
	StudentID  string `gorm:"type:char(36);primaryKey"`
	AssignedAt time.Time
}

func (Participant) TableName() string { return "exam_participants" }
