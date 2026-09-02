package materials

import "time"

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusArchived  = "archived"
)

type Material struct {
	ID          string `gorm:"type:char(36);primaryKey"`
	CourseID    string `gorm:"type:char(36);not null"`
	AuthorID    string `gorm:"type:char(36);not null"`
	Title       string `gorm:"size:180;not null"`
	Description string `gorm:"type:text"`
	Content     string `gorm:"type:text;not null"`
	Position    uint
	Status      string `gorm:"size:20;not null"`
	PublishedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Material) TableName() string { return "course_materials" }

type Progress struct {
	MaterialID  string `gorm:"type:char(36);primaryKey"`
	StudentID   string `gorm:"type:char(36);primaryKey"`
	CompletedAt time.Time
}

func (Progress) TableName() string { return "material_progress" }
