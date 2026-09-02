package attempts

import (
	"context"
	"errors"
	"time"

	"lms-cn-api/internal/modules/exams"
	"lms-cn-api/internal/modules/grading"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrNotFound            = errors.New("attempt not found")
	ErrNotEligible         = errors.New("student not eligible")
	ErrExamUnavailable     = errors.New("exam unavailable")
	ErrAttemptFinal        = errors.New("attempt is final")
	ErrAttemptExpired      = errors.New("attempt expired")
	ErrInvalidAnswer       = errors.New("invalid answer")
	ErrIdempotencyConflict = errors.New("idempotency key conflict")
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

type availableRow struct {
	ID              string
	CourseID        string
	Title           string
	Description     string
	StartsAt        time.Time
	EndsAt          time.Time
	DurationMinutes uint
	AttemptStatus   *string
}

func (r *Repository) ListAvailable(ctx context.Context, studentID string, now time.Time) ([]availableRow, error) {
	var rows []availableRow
	err := r.db.WithContext(ctx).Table("exams e").
		Select("e.id, e.course_id, e.title, e.description, e.starts_at, e.ends_at, e.duration_minutes, a.status AS attempt_status").
		Joins("JOIN exam_participants ep ON ep.exam_id = e.id AND ep.student_id = ?", studentID).
		Joins("LEFT JOIN attempts a ON a.exam_id = e.id AND a.student_id = ?", studentID).
		Where("e.status = ? AND e.ends_at > ?", exams.StatusPublished, now).
		Order("e.starts_at ASC").Scan(&rows).Error
	return rows, err
}

func (r *Repository) Start(ctx context.Context, examID, studentID, idempotencyKey string, now time.Time) (Attempt, exams.Exam, error) {
	var result Attempt
	var exam exams.Exam
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).First(&exam, "id = ?", examID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrExamUnavailable
			}
			return err
		}
		if exam.Status != exams.StatusPublished || now.Before(exam.StartsAt) || !now.Before(exam.EndsAt) {
			return ErrExamUnavailable
		}
		var eligible int64
		if err := tx.Model(&exams.Participant{}).Where("exam_id = ? AND student_id = ?", examID, studentID).Count(&eligible).Error; err != nil {
			return err
		}
		if eligible == 0 {
			return ErrNotEligible
		}
		var keyed Attempt
		err := tx.Where("student_id = ? AND start_idempotency_key = ?", studentID, idempotencyKey).First(&keyed).Error
		if err == nil {
			if keyed.ExamID != examID {
				return ErrIdempotencyConflict
			}
			result = keyed
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var existing Attempt
		err = tx.Where("exam_id = ? AND student_id = ?", examID, studentID).First(&existing).Error
		if err == nil {
			result = existing
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		deadline := now.Add(time.Duration(exam.DurationMinutes) * time.Minute)
		if deadline.After(exam.EndsAt) {
			deadline = exam.EndsAt
		}
		result = Attempt{ID: uuid.NewString(), ExamID: examID, StudentID: studentID, Status: StatusInProgress, StartIdempotencyKey: idempotencyKey, StartedAt: now, DeadlineAt: deadline}
		if err := tx.Create(&result).Error; err != nil {
			return err
		}
		return tx.Create(&Event{ID: uuid.NewString(), AttemptID: result.ID, ActorID: studentID, EventType: "attempt.started", Metadata: []byte("{}")}).Error
	})
	if err != nil {
		return Attempt{}, exams.Exam{}, err
	}
	hydratedExam, err := r.loadExam(ctx, examID)
	return result, hydratedExam, err
}

func (r *Repository) FindForStudent(ctx context.Context, attemptID, studentID string) (Attempt, exams.Exam, error) {
	var attempt Attempt
	err := r.db.WithContext(ctx).Preload("Answers").First(&attempt, "id = ? AND student_id = ?", attemptID, studentID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Attempt{}, exams.Exam{}, ErrNotFound
	}
	if err != nil {
		return Attempt{}, exams.Exam{}, err
	}
	exam, err := r.loadExam(ctx, attempt.ExamID)
	return attempt, exam, err
}

func (r *Repository) SaveAnswer(ctx context.Context, attemptID, studentID, key string, request SaveAnswerRequest, now time.Time) (Answer, error) {
	var result Answer
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var attempt Attempt
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&attempt, "id = ? AND student_id = ?", attemptID, studentID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if attempt.Status != StatusInProgress {
			return ErrAttemptFinal
		}
		if !now.Before(attempt.DeadlineAt) {
			return ErrAttemptExpired
		}

		var valid int64
		err = tx.Table("exam_question_options eqo").Joins("JOIN exam_questions eq ON eq.id = eqo.exam_question_id").
			Where("eq.exam_id = ? AND eq.id = ? AND eqo.id = ?", attempt.ExamID, request.ExamQuestionID, request.SelectedOptionID).Count(&valid).Error
		if err != nil {
			return err
		}
		if valid == 0 {
			return ErrInvalidAnswer
		}

		var savedRequest AnswerSaveRequest
		err = tx.Where("attempt_id = ? AND idempotency_key = ?", attemptID, key).First(&savedRequest).Error
		if err == nil {
			if savedRequest.ExamQuestionID != request.ExamQuestionID || savedRequest.SelectedOptionID != request.SelectedOptionID {
				return ErrIdempotencyConflict
			}
			result = Answer{ID: savedRequest.AnswerID, AttemptID: attemptID, ExamQuestionID: savedRequest.ExamQuestionID, SelectedOptionID: savedRequest.SelectedOptionID, Revision: savedRequest.Revision, SavedAt: savedRequest.SavedAt}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var answer Answer
		err = tx.Where("attempt_id = ? AND exam_question_id = ?", attemptID, request.ExamQuestionID).First(&answer).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			answer = Answer{ID: uuid.NewString(), AttemptID: attemptID, ExamQuestionID: request.ExamQuestionID, SelectedOptionID: request.SelectedOptionID, Revision: 1, SavedAt: now}
			if err := tx.Create(&answer).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			answer.SelectedOptionID = request.SelectedOptionID
			answer.Revision++
			answer.SavedAt = now
			if err := tx.Save(&answer).Error; err != nil {
				return err
			}
		}
		if err := tx.Create(&AnswerSaveRequest{AttemptID: attemptID, IdempotencyKey: key, ExamQuestionID: request.ExamQuestionID, SelectedOptionID: request.SelectedOptionID, AnswerID: answer.ID, Revision: answer.Revision, SavedAt: answer.SavedAt, CreatedAt: now}).Error; err != nil {
			return err
		}
		result = answer
		return nil
	})
	return result, err
}

func (r *Repository) Finalize(ctx context.Context, attemptID, studentID, mode string, now time.Time, calculator grading.Calculator) (Attempt, error) {
	var result Attempt
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", attemptID)
		if studentID != "" {
			query = query.Where("student_id = ?", studentID)
		}
		var attempt Attempt
		err := query.First(&attempt).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if attempt.Status != StatusInProgress {
			result = attempt
			return nil
		}

		status := StatusSubmitted
		if mode == StatusExpired || !now.Before(attempt.DeadlineAt) {
			status = StatusExpired
		}
		score, err := r.calculateScore(tx, attempt, calculator)
		if err != nil {
			return err
		}
		receipt := uuid.NewString()
		attempt.Status = status
		attempt.SubmittedAt = &now
		attempt.SubmissionReceipt = &receipt
		if err := tx.Save(&attempt).Error; err != nil {
			return err
		}
		grade := gradeRecord{ID: uuid.NewString(), AttemptID: attempt.ID, ExamID: attempt.ExamID, StudentID: attempt.StudentID, Status: "draft", Score: score.Earned, MaxScore: score.Maximum, GradedAt: now}
		if err := tx.Table("results").Create(&grade).Error; err != nil {
			return err
		}
		eventType := "attempt.submitted"
		if status == StatusExpired {
			eventType = "attempt.expired"
		}
		if err := tx.Create(&Event{ID: uuid.NewString(), AttemptID: attempt.ID, ActorID: attempt.StudentID, EventType: eventType, Metadata: []byte("{}")}).Error; err != nil {
			return err
		}
		result = attempt
		return nil
	})
	return result, err
}

func (r *Repository) ExpiredAttemptIDs(ctx context.Context, now time.Time, limit int) ([]string, error) {
	var ids []string
	err := r.db.WithContext(ctx).Model(&Attempt{}).Where("status = ? AND deadline_at <= ?", StatusInProgress, now).Order("deadline_at ASC").Limit(limit).Pluck("id", &ids).Error
	return ids, err
}

func (r *Repository) loadExam(ctx context.Context, examID string) (exams.Exam, error) {
	var exam exams.Exam
	err := r.db.WithContext(ctx).Preload("Questions", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC") }).Preload("Questions.Options", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC") }).First(&exam, "id = ?", examID).Error
	return exam, err
}

func (r *Repository) calculateScore(tx *gorm.DB, attempt Attempt, calculator grading.Calculator) (grading.Score, error) {
	var questions []exams.ExamQuestion
	if err := tx.Preload("Options").Where("exam_id = ?", attempt.ExamID).Find(&questions).Error; err != nil {
		return grading.Score{}, err
	}
	var answers []Answer
	if err := tx.Where("attempt_id = ?", attempt.ID).Find(&answers).Error; err != nil {
		return grading.Score{}, err
	}
	selected := make(map[string]string, len(answers))
	for _, answer := range answers {
		selected[answer.ExamQuestionID] = answer.SelectedOptionID
	}
	input := grading.Input{Questions: make([]grading.Question, len(questions))}
	for i, question := range questions {
		correct := ""
		for _, option := range question.Options {
			if option.IsCorrect {
				correct = option.ID
				break
			}
		}
		input.Questions[i] = grading.Question{ID: question.ID, Type: question.Type, Points: question.Points, CorrectOptionID: correct, SelectedOptionID: selected[question.ID]}
	}
	return calculator.Calculate(input), nil
}

type gradeRecord struct {
	ID        string
	AttemptID string
	ExamID    string
	StudentID string
	Status    string
	Score     float64
	MaxScore  float64
	GradedAt  time.Time
}
