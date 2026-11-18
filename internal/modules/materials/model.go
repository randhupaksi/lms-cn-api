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

type Attachment struct {
	ID             string    `gorm:"type:char(36);primaryKey"`
	MaterialID     string    `gorm:"type:char(36);not null;index:idx_material_attachments_material,priority:1;uniqueIndex:uq_material_attachments_checksum,priority:1"`
	FileName       string    `gorm:"size:255;not null"`
	ContentType    string    `gorm:"size:100;not null"`
	SizeBytes      uint64    `gorm:"not null"`
	ChecksumSHA256 string    `gorm:"column:checksum_sha256;size:64;not null;uniqueIndex:uq_material_attachments_checksum,priority:2"`
	FileData       []byte    `gorm:"column:file_data;type:longblob;not null"`
	CreatedAt      time.Time `gorm:"index:idx_material_attachments_material,priority:2"`
}

func (Attachment) TableName() string { return "course_material_attachments" }

type Progress struct {
	MaterialID  string `gorm:"type:char(36);primaryKey"`
	StudentID   string `gorm:"type:char(36);primaryKey"`
	CompletedAt time.Time
}

func (Progress) TableName() string { return "material_progress" }
