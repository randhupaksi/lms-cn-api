package users

import "time"

type Role string

const (
	RoleAdmin   Role = "admin"
	RoleTeacher Role = "teacher"
	RoleStudent Role = "student"
)

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)

type User struct {
	ID                 string `gorm:"type:char(36);primaryKey"`
	Identifier         string `gorm:"size:64;uniqueIndex;not null"`
	FullName           string `gorm:"size:160;not null"`
	Role               Role   `gorm:"size:20;not null"`
	Status             Status `gorm:"size:20;not null"`
	PasswordHash       string `gorm:"size:255;not null"`
	MustChangePassword bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (User) TableName() string { return "users" }

func (u User) IsActive() bool { return u.Status == StatusActive }
