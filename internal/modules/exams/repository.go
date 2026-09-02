package exams

import (
	"context"
	"errors"
	"time"

	"lms-cn-api/internal/modules/questions"
	"lms-cn-api/internal/modules/users"
	"lms-cn-api/pkg/pagination"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrNotFound = errors.New("exam not found")

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, exam *Exam) error {
	return r.db.WithContext(ctx).Create(exam).Error
}

func (r *Repository) Find(ctx context.Context, id string) (Exam, int64, error) {
	var exam Exam
	err := r.db.WithContext(ctx).
		Preload("Questions", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC") }).
		Preload("Questions.Options", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC") }).
		First(&exam, "exams.id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Exam{}, 0, ErrNotFound
	}
	if err != nil {
		return Exam{}, 0, err
	}
	var participants int64
	err = r.db.WithContext(ctx).Model(&Participant{}).Where("exam_id = ?", id).Count(&participants).Error
	return exam, participants, err
}

func (r *Repository) ParticipantIDs(ctx context.Context, examID string) ([]string, error) {
	var ids []string
	err := r.db.WithContext(ctx).Model(&Participant{}).Where("exam_id = ?", examID).Order("assigned_at ASC").Pluck("student_id", &ids).Error
	return ids, err
}

func (r *Repository) List(ctx context.Context, courseID string, page pagination.Request) ([]Exam, map[string]int64, int64, error) {
	query := r.db.WithContext(ctx).Model(&Exam{}).Where("course_id = ?", courseID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, nil, 0, err
	}
	var exams []Exam
	if err := query.Preload("Questions").Order("starts_at DESC").Offset(page.Offset()).Limit(page.PerPage).Find(&exams).Error; err != nil {
		return nil, nil, 0, err
	}
	counts := make(map[string]int64, len(exams))
	if len(exams) == 0 {
		return exams, counts, total, nil
	}
	type countRow struct {
		ExamID string
		Total  int64
	}
	var rows []countRow
	ids := make([]string, len(exams))
	for i, exam := range exams {
		ids[i] = exam.ID
	}
	if err := r.db.WithContext(ctx).Model(&Participant{}).Select("exam_id, COUNT(*) AS total").Where("exam_id IN ?", ids).Group("exam_id").Scan(&rows).Error; err != nil {
		return nil, nil, 0, err
	}
	for _, row := range rows {
		counts[row.ExamID] = row.Total
	}
	return exams, counts, total, nil
}

func (r *Repository) UpdateDraft(ctx context.Context, id string, apply func(*Exam)) (Exam, error) {
	var exam Exam
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&exam, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if exam.Status != StatusDraft {
			return ErrExamLocked
		}
		apply(&exam)
		return tx.Save(&exam).Error
	})
	return exam, err
}

var ErrExamLocked = errors.New("exam is locked")
var ErrInvalidSelection = errors.New("invalid exam selection")

func (r *Repository) ReplaceQuestions(ctx context.Context, exam Exam, selections []QuestionSelection) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var locked Exam
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, "id = ?", exam.ID).Error; err != nil {
			return err
		}
		if locked.Status != StatusDraft {
			return ErrExamLocked
		}
		ids := make([]string, len(selections))
		for i, selection := range selections {
			ids[i] = selection.QuestionID
		}
		var source []questions.Question
		if err := tx.Preload("Options", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC") }).Where("id IN ? AND course_id = ? AND status = 'active'", ids, exam.CourseID).Find(&source).Error; err != nil {
			return err
		}
		byID := make(map[string]questions.Question, len(source))
		for _, question := range source {
			byID[question.ID] = question
		}
		if len(byID) != len(selections) {
			return ErrInvalidSelection
		}
		if err := tx.Where("exam_id = ?", exam.ID).Delete(&ExamQuestion{}).Error; err != nil {
			return err
		}
		for index, selection := range selections {
			question := byID[selection.QuestionID]
			snapshot := ExamQuestion{ID: uuid.NewString(), ExamID: exam.ID, SourceQuestionID: question.ID, SourceVersion: question.Version, Type: question.Type, Stem: question.Stem, Position: uint(index + 1), Points: selection.Points}
			if err := tx.Create(&snapshot).Error; err != nil {
				return err
			}
			options := make([]ExamQuestionOption, len(question.Options))
			for optionIndex, option := range question.Options {
				options[optionIndex] = ExamQuestionOption{ID: uuid.NewString(), ExamQuestionID: snapshot.ID, SourceOptionID: option.ID, Content: option.Content, IsCorrect: option.IsCorrect, Position: option.Position}
			}
			if err := tx.Create(&options).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) ReplaceParticipants(ctx context.Context, exam Exam, studentIDs []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var locked Exam
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, "id = ?", exam.ID).Error; err != nil {
			return err
		}
		if locked.Status != StatusDraft {
			return ErrExamLocked
		}
		var count int64
		err := tx.Table("course_students cs").Joins("JOIN users u ON u.id = cs.student_id").
			Where("cs.course_id = ? AND cs.student_id IN ? AND u.role = ? AND u.status = ?", exam.CourseID, studentIDs, users.RoleStudent, users.StatusActive).Count(&count).Error
		if err != nil {
			return err
		}
		if count != int64(len(studentIDs)) {
			return ErrInvalidSelection
		}
		if err := tx.Where("exam_id = ?", exam.ID).Delete(&Participant{}).Error; err != nil {
			return err
		}
		participants := make([]Participant, len(studentIDs))
		for i, studentID := range studentIDs {
			participants[i] = Participant{ExamID: exam.ID, StudentID: studentID}
		}
		return tx.Create(&participants).Error
	})
}

func (r *Repository) Publish(ctx context.Context, examID string, publishedAt time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var exam Exam
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&exam, "id = ?", examID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if exam.Status != StatusDraft {
			return ErrExamLocked
		}
		var questionCount, participantCount int64
		if err := tx.Model(&ExamQuestion{}).Where("exam_id = ?", examID).Count(&questionCount).Error; err != nil {
			return err
		}
		if err := tx.Model(&Participant{}).Where("exam_id = ?", examID).Count(&participantCount).Error; err != nil {
			return err
		}
		if questionCount == 0 || participantCount == 0 {
			return ErrInvalidSelection
		}
		return tx.Model(&exam).Updates(map[string]any{"status": StatusPublished, "published_at": publishedAt}).Error
	})
}

func (r *Repository) Unpublish(ctx context.Context, examID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var exam Exam
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&exam, "id = ?", examID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if exam.Status != StatusPublished {
			return ErrExamLocked
		}
		var attempts int64
		if err := tx.Table("attempts").Where("exam_id = ?", examID).Count(&attempts).Error; err != nil {
			return err
		}
		if attempts > 0 {
			return ErrExamLocked
		}
		return tx.Model(&exam).Updates(map[string]any{"status": StatusDraft, "published_at": nil}).Error
	})
}
