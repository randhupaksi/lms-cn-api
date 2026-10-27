package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"lms-cn-api/internal/modules/academics"
	"lms-cn-api/internal/modules/assignments"
	"lms-cn-api/internal/modules/exams"
	"lms-cn-api/internal/modules/materials"
	"lms-cn-api/internal/modules/questions"
	"lms-cn-api/internal/modules/users"

	"gorm.io/gorm"
)

const demoPassword = "12121212"

type demoStudent struct {
	user        users.User
	classIndex  int
	classNumber int
}

type demoExam struct {
	exam            exams.Exam
	questionIDStart int
}

type demoCourse struct {
	index       int
	class       academics.ClassGroup
	subject     academics.Subject
	course      academics.Course
	teachers    []users.User
	students    []demoStudent
	questions   []questions.Question
	exams       []demoExam
	materials   []materials.Material
	assignments []assignments.Assignment
}

type demoDataset struct {
	admin      users.User
	superadmin users.User
	teachers   []users.User
	students   []demoStudent
	year       academics.AcademicYear
	classes    []academics.ClassGroup
	subjects   []academics.Subject
	courses    []demoCourse
}

// SeedDemoData upserts a connected, fictional LMS dataset for local
// development. Seed-owned records have stable IDs, so each startup refreshes
// relative dates and demo state without duplicating rows. Configuration keeps
// this path disabled outside development.
func SeedDemoData(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return errors.New("cannot seed demo data with a nil database")
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		dataset, err := seedDemoUsers(tx)
		if err != nil {
			return fmt.Errorf("seed demo users: %w", err)
		}
		if err := seedDemoAcademics(tx, &dataset, now); err != nil {
			return fmt.Errorf("seed demo academics: %w", err)
		}
		if err := seedDemoQuestionBanks(tx, &dataset); err != nil {
			return fmt.Errorf("seed demo question banks: %w", err)
		}
		if err := seedDemoExamsAndAttempts(tx, &dataset, now); err != nil {
			return fmt.Errorf("seed demo exams and attempts: %w", err)
		}
		if err := seedDemoLearningContent(tx, &dataset, now); err != nil {
			return fmt.Errorf("seed demo learning content: %w", err)
		}
		if err := seedDemoAuditTrail(tx, dataset, now); err != nil {
			return fmt.Errorf("seed demo audit trail: %w", err)
		}
		if err := seedDemoGovernance(tx, dataset, now); err != nil {
			return fmt.Errorf("seed school governance: %w", err)
		}
		return nil
	})
}

func seedRecord(tx *gorm.DB, value any) error {
	return tx.Save(value).Error
}

func seedID(suffix int) string {
	return fmt.Sprintf("00000000-0000-4000-8000-%012d", suffix)
}

func pointer(value float64) *float64 { return &value }

func stringPointer(value string) *string { return &value }

func metadata(value map[string]any) json.RawMessage {
	encoded, _ := json.Marshal(value)
	return encoded
}

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
