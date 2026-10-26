package database

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"lms-cn-api/internal/modules/audit"
)

func seedDemoAuditTrail(tx *gorm.DB, dataset demoDataset, now time.Time) error {
	entries := make([]audit.Event, 0, len(dataset.courses)*8)
	for _, course := range dataset.courses {
		teacher := course.teachers[0]
		student := course.students[course.index%len(course.students)].user
		entities := []struct {
			action     string
			entityType string
			entityID   string
			actorID    string
		}{
			{"course.created", "course", course.course.ID, dataset.admin.ID},
			{"question.created", "question", course.questions[0].ID, teacher.ID},
			{"exam.published", "exam", course.exams[demoExamActive].exam.ID, teacher.ID},
			{"material.published", "material", course.materials[0].ID, teacher.ID},
			{"assignment.published", "assignment", course.assignments[0].ID, teacher.ID},
			{"attempt.submitted", "attempt", seedID(demoHistoricalAttemptIDSuffix(course.index, course.index%len(course.students))), student.ID},
			{"assignment.graded", "assignment_submission", seedID(demoSubmissionIDSuffix(course.index, 1, 0)), teacher.ID},
			{"result.published", "result", seedID(demoHistoricalResultIDSuffix(course.index, course.index%len(course.students))), teacher.ID},
		}
		for kind, entity := range entities {
			sequence := course.index*len(entities) + kind
			actorID := entity.actorID
			entityID := entity.entityID
			createdAt := now.AddDate(0, 0, -(1 + (sequence*3)%13)).Add(time.Duration((sequence*7)%11) * time.Hour)
			entries = append(entries, audit.Event{
				ID:         seedID(1200 + sequence),
				ActorID:    stringPointer(actorID),
				Action:     entity.action,
				EntityType: entity.entityType,
				EntityID:   stringPointer(entityID),
				Metadata:   metadata(map[string]any{"source": "development_seed", "group": "sample_activity"}),
				CreatedAt:  createdAt,
			})
		}
	}

	for index := range entries {
		if err := seedRecord(tx, &entries[index]); err != nil {
			return fmt.Errorf("audit event %s: %w", entries[index].Action, err)
		}
	}
	return nil
}
