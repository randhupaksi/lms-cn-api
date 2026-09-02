package academics

import "time"

type AcademicYear struct {
	ID        string `gorm:"type:char(36);primaryKey"`
	Name      string `gorm:"size:80;uniqueIndex;not null"`
	StartsOn  time.Time
	EndsOn    time.Time
	Status    string `gorm:"size:20;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ClassGroup struct {
	ID             string `gorm:"type:char(36);primaryKey"`
	AcademicYearID string `gorm:"type:char(36);not null"`
	Name           string `gorm:"size:80;not null"`
	GradeLevel     int
	AcademicYear   AcademicYear
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Subject struct {
	ID        string `gorm:"type:char(36);primaryKey"`
	Code      string `gorm:"size:32;uniqueIndex;not null"`
	Name      string `gorm:"size:120;uniqueIndex;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Course struct {
	ID             string `gorm:"type:char(36);primaryKey"`
	AcademicYearID string `gorm:"type:char(36);not null"`
	ClassGroupID   string `gorm:"type:char(36);not null"`
	SubjectID      string `gorm:"type:char(36);not null"`
	Name           string `gorm:"size:160;not null"`
	Status         string `gorm:"size:20;not null"`
	AcademicYear   AcademicYear
	ClassGroup     ClassGroup
	Subject        Subject
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CourseTeacher struct {
	CourseID   string `gorm:"type:char(36);primaryKey"`
	TeacherID  string `gorm:"type:char(36);primaryKey"`
	AssignedAt time.Time
}

type CourseStudent struct {
	CourseID   string `gorm:"type:char(36);primaryKey"`
	StudentID  string `gorm:"type:char(36);primaryKey"`
	EnrolledAt time.Time
}
