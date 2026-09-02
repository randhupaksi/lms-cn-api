package auth

import (
	"time"

	"lms-cn-api/internal/modules/users"
)

type Session struct {
	ID               string `gorm:"type:char(36);primaryKey"`
	UserID           string `gorm:"type:char(36);not null"`
	RefreshTokenHash string `gorm:"type:char(64);uniqueIndex;not null"`
	ExpiresAt        time.Time
	LastUsedAt       time.Time
	RevokedAt        *time.Time
	CreatedAt        time.Time
	User             users.User `gorm:"foreignKey:UserID"`
}

func (Session) TableName() string { return "auth_sessions" }

func (s Session) IsValid(now time.Time) bool {
	return s.RevokedAt == nil && s.ExpiresAt.After(now) && s.User.IsActive()
}
