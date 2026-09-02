package questions

import "time"

const TypeSingleChoice = "single_choice"

type Question struct {
	ID            string `gorm:"type:char(36);primaryKey"`
	CourseID      string `gorm:"type:char(36);not null"`
	AuthorID      string `gorm:"type:char(36);not null"`
	Type          string `gorm:"size:32;not null"`
	Stem          string `gorm:"type:text;not null"`
	DefaultPoints float64
	Category      string   `gorm:"size:80"`
	Tags          []string `gorm:"type:json;serializer:json"`
	Status        string   `gorm:"size:20;not null"`
	Version       uint
	Options       []Option `gorm:"foreignKey:QuestionID"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Option struct {
	ID         string `gorm:"type:char(36);primaryKey"`
	QuestionID string `gorm:"type:char(36);not null"`
	Content    string `gorm:"type:text;not null"`
	IsCorrect  bool
	Position   uint
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (Option) TableName() string { return "question_options" }
