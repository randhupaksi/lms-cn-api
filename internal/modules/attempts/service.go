package attempts

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"lms-cn-api/internal/authz"
	"lms-cn-api/internal/modules/exams"
	"lms-cn-api/internal/modules/grading"
	"lms-cn-api/internal/modules/users"
	"lms-cn-api/pkg/apperror"
)

type Service struct {
	repository *Repository
	calculator grading.Calculator
	now        func() time.Time
}

func NewService(repository *Repository, calculator grading.Calculator) *Service {
	return &Service{repository: repository, calculator: calculator, now: time.Now}
}

func (s *Service) ListAvailable(ctx context.Context, actor authz.Principal) ([]AvailableExamResponse, error) {
	if err := requireStudent(actor); err != nil {
		return nil, err
	}
	now := s.now().UTC()
	rows, err := s.repository.ListAvailable(ctx, actor.UserID, now)
	if err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, "AVAILABLE_EXAMS_FAILED", "Gagal memuat ujian siswa", err)
	}
	result := make([]AvailableExamResponse, len(rows))
	for i, row := range rows {
		result[i] = AvailableExamResponse{ID: row.ID, CourseID: row.CourseID, Title: row.Title, Description: row.Description, StartsAt: row.StartsAt, EndsAt: row.EndsAt, DurationMinutes: row.DurationMinutes, AttemptStatus: row.AttemptStatus, ServerTime: now}
	}
	return result, nil
}

func (s *Service) Start(ctx context.Context, actor authz.Principal, examID, idempotencyKey string) (StartResponse, error) {
	if err := requireStudent(actor); err != nil {
		return StartResponse{}, err
	}
	if err := validateIdempotencyKey(idempotencyKey); err != nil {
		return StartResponse{}, err
	}
	now := s.now().UTC()
	attempt, exam, err := s.repository.Start(ctx, examID, actor.UserID, idempotencyKey, now)
	if err != nil {
		return StartResponse{}, mapAttemptError(err)
	}
	if hydrated, hydratedExam, hydrateErr := s.repository.FindForStudent(ctx, attempt.ID, actor.UserID); hydrateErr == nil {
		attempt, exam = hydrated, hydratedExam
	}
	if attempt.Status != StatusInProgress {
		return StartResponse{}, apperror.New(http.StatusConflict, "ATTEMPT_FINAL", "Attempt sudah selesai")
	}
	return buildStartResponse(attempt, exam, now), nil
}

func (s *Service) Resume(ctx context.Context, actor authz.Principal, attemptID string) (StartResponse, error) {
	if err := requireStudent(actor); err != nil {
		return StartResponse{}, err
	}
	now := s.now().UTC()
	attempt, exam, err := s.repository.FindForStudent(ctx, attemptID, actor.UserID)
	if err != nil {
		return StartResponse{}, mapAttemptError(err)
	}
	if attempt.Status == StatusInProgress && !now.Before(attempt.DeadlineAt) {
		attempt, err = s.repository.Finalize(ctx, attempt.ID, actor.UserID, StatusExpired, now, s.calculator)
		if err != nil {
			return StartResponse{}, mapAttemptError(err)
		}
	}
	return buildStartResponse(attempt, exam, now), nil
}

func (s *Service) SaveAnswer(ctx context.Context, actor authz.Principal, attemptID, idempotencyKey string, request SaveAnswerRequest) (AnswerResponse, error) {
	if err := requireStudent(actor); err != nil {
		return AnswerResponse{}, err
	}
	if err := validateIdempotencyKey(idempotencyKey); err != nil {
		return AnswerResponse{}, err
	}
	now := s.now().UTC()
	answer, err := s.repository.SaveAnswer(ctx, attemptID, actor.UserID, idempotencyKey, request, now)
	if errors.Is(err, ErrAttemptExpired) {
		_, _ = s.repository.Finalize(ctx, attemptID, actor.UserID, StatusExpired, now, s.calculator)
	}
	if err != nil {
		return AnswerResponse{}, mapAttemptError(err)
	}
	return toAnswerResponse(answer), nil
}

func (s *Service) Submit(ctx context.Context, actor authz.Principal, attemptID, idempotencyKey string) (ReceiptResponse, error) {
	if err := requireStudent(actor); err != nil {
		return ReceiptResponse{}, err
	}
	if err := validateIdempotencyKey(idempotencyKey); err != nil {
		return ReceiptResponse{}, err
	}
	attempt, err := s.repository.Finalize(ctx, attemptID, actor.UserID, StatusSubmitted, s.now().UTC(), s.calculator)
	if err != nil {
		return ReceiptResponse{}, mapAttemptError(err)
	}
	return ReceiptResponse{AttemptID: attempt.ID, Status: attempt.Status, Receipt: attempt.SubmissionReceipt, SubmittedAt: attempt.SubmittedAt}, nil
}

func (s *Service) FinalizeExpired(ctx context.Context, limit int) error {
	now := s.now().UTC()
	ids, err := s.repository.ExpiredAttemptIDs(ctx, now, limit)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if _, err := s.repository.Finalize(ctx, id, "", StatusExpired, now, s.calculator); err != nil && !errors.Is(err, ErrAttemptFinal) {
			slog.ErrorContext(ctx, "failed to finalize expired attempt", "attempt_id", id, "error", err)
		}
	}
	return nil
}

func buildStartResponse(attempt Attempt, exam exams.Exam, now time.Time) StartResponse {
	questions := make([]exams.QuestionResponse, len(exam.Questions))
	for i, question := range exam.Questions {
		options := make([]exams.OptionResponse, len(question.Options))
		for j, option := range question.Options {
			options[j] = exams.OptionResponse{ID: option.ID, Content: option.Content, Position: option.Position}
		}
		questions[i] = exams.QuestionResponse{ID: question.ID, SourceQuestionID: question.SourceQuestionID, Type: question.Type, Stem: question.Stem, Position: question.Position, Points: question.Points, Options: options}
	}
	answers := make([]AnswerResponse, len(attempt.Answers))
	for i, answer := range attempt.Answers {
		answers[i] = toAnswerResponse(answer)
	}
	return StartResponse{AttemptID: attempt.ID, Status: attempt.Status, StartedAt: attempt.StartedAt, DeadlineAt: attempt.DeadlineAt, ServerTime: now, AllowBackNavigation: exam.AllowBackNavigation, Questions: questions, Answers: answers, SubmissionReceipt: attempt.SubmissionReceipt, SubmittedAt: attempt.SubmittedAt}
}

func toAnswerResponse(answer Answer) AnswerResponse {
	return AnswerResponse{ExamQuestionID: answer.ExamQuestionID, SelectedOptionID: answer.SelectedOptionID, Revision: answer.Revision, SavedAt: answer.SavedAt}
}

func requireStudent(actor authz.Principal) error {
	if actor.Role != string(users.RoleStudent) {
		return apperror.New(http.StatusForbidden, "STUDENT_ONLY", "Fitur ini hanya tersedia untuk siswa")
	}
	return nil
}

func validateIdempotencyKey(value string) error {
	value = strings.TrimSpace(value)
	if len(value) < 8 || len(value) > 120 {
		return apperror.New(http.StatusBadRequest, "IDEMPOTENCY_KEY_REQUIRED", "Idempotency-Key yang valid wajib dikirim")
	}
	return nil
}

func mapAttemptError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return apperror.New(http.StatusNotFound, "ATTEMPT_NOT_FOUND", "Attempt tidak ditemukan")
	case errors.Is(err, ErrNotEligible):
		return apperror.New(http.StatusForbidden, "EXAM_NOT_ELIGIBLE", "Kamu tidak terdaftar sebagai peserta ujian")
	case errors.Is(err, ErrExamUnavailable):
		return apperror.New(http.StatusConflict, "EXAM_UNAVAILABLE", "Ujian belum tersedia atau sudah berakhir")
	case errors.Is(err, ErrAttemptFinal):
		return apperror.New(http.StatusConflict, "ATTEMPT_FINAL", "Attempt sudah selesai dan tidak dapat diubah")
	case errors.Is(err, ErrAttemptExpired):
		return apperror.New(http.StatusConflict, "ATTEMPT_EXPIRED", "Waktu ujian telah berakhir")
	case errors.Is(err, ErrInvalidAnswer):
		return apperror.New(http.StatusUnprocessableEntity, "ANSWER_INVALID", "Jawaban tidak sesuai dengan soal attempt")
	case errors.Is(err, ErrIdempotencyConflict):
		return apperror.New(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "Idempotency-Key sudah digunakan untuk payload lain")
	default:
		return apperror.Wrap(http.StatusInternalServerError, "ATTEMPT_FAILED", "Gagal memproses attempt", err)
	}
}

type ExpiryWorker struct {
	service  *Service
	interval time.Duration
}

func NewExpiryWorker(service *Service, interval time.Duration) *ExpiryWorker {
	return &ExpiryWorker{service: service, interval: interval}
}

func (w *ExpiryWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.service.FinalizeExpired(ctx, 100); err != nil {
				slog.ErrorContext(ctx, "failed to scan expired attempts", "error", err)
			}
		}
	}
}
