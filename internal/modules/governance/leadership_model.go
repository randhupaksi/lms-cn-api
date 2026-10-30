package governance

import "time"

type LeadershipAction struct {
	ID          string     `gorm:"type:char(36);primaryKey"`
	Title       string     `gorm:"size:180;not null"`
	Description string     `gorm:"size:2000;not null"`
	Category    string     `gorm:"size:24;not null"`
	Priority    string     `gorm:"size:12;not null"`
	Status      string     `gorm:"size:20;not null"`
	AssigneeID  *string    `gorm:"type:char(36)"`
	DueAt       *time.Time `gorm:"type:datetime(6)"`
	CreatedBy   string     `gorm:"type:char(36);not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time `gorm:"type:datetime(6)"`
	Version     int        `gorm:"not null"`
}

func (LeadershipAction) TableName() string { return "leadership_actions" }

type LeadershipActionUpdate struct {
	ID         string    `gorm:"type:char(36);primaryKey"`
	ActionID   string    `gorm:"type:char(36);not null"`
	ActorID    string    `gorm:"type:char(36);not null"`
	FromStatus *string   `gorm:"size:20"`
	ToStatus   string    `gorm:"size:20;not null"`
	Note       string    `gorm:"size:1000;not null"`
	CreatedAt  time.Time `gorm:"not null"`
}

func (LeadershipActionUpdate) TableName() string { return "leadership_action_updates" }
