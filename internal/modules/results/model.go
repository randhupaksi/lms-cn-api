package results

import "time"

type Result struct {
	ID          string `gorm:"type:char(36);primaryKey"`
	AttemptID   string `gorm:"type:char(36);not null"`
	ExamID      string `gorm:"type:char(36);not null"`
	StudentID   string `gorm:"type:char(36);not null"`
	Status      string `gorm:"size:20;not null"`
	Score       float64
	MaxScore    float64
	GradedAt    time.Time
	PublishedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
