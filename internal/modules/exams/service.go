package exams

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"lms-cn-api/internal/authz"
	"lms-cn-api/internal/modules/academics"
	"lms-cn-api/internal/modules/audit"
	"lms-cn-api/internal/modules/users"
	"lms-cn-api/pkg/apperror"
	"lms-cn-api/pkg/pagination"

	"github.com/google/uuid"
)

type Service struct {
	repository *Repository
	academics  *academics.Service
	audit      audit.Recorder
	now        func() time.Time
}

func NewService(repository *Repository, academicsService *academics.Service, auditRecorder audit.Recorder) *Service {
	return &Service{repository: repository, academics: academicsService, audit: auditRecorder, now: time.Now}
}

func (s *Service) Create(ctx context.Context, actor authz.Principal, request WriteRequest) (Response, error) {
	if err := requireTeacher(actor); err != nil {
		return Response{}, err
	}
	if err := s.academics.RequireCourseManager(ctx, actor, request.CourseID); err != nil {
		return Response{}, err
	}
	if err := validateConfiguration(request); err != nil {
		return Response{}, err
	}
	exam := Exam{
		ID: uuid.NewString(), CourseID: request.CourseID, AuthorID: actor.UserID,
		Title: strings.TrimSpace(request.Title), Description: strings.TrimSpace(request.Description),
		Status: StatusDraft, StartsAt: request.StartsAt.UTC(), EndsAt: request.EndsAt.UTC(),
		DurationMinutes: request.DurationMinutes, MaxAttempts: 1,
		AllowBackNavigation: request.AllowBackNavigation, ResultPolicy: "after_publish",
	}
	if err := s.repository.Create(ctx, &exam); err != nil {
		return Response{}, apperror.Wrap(http.StatusInternalServerError, "EXAM_CREATE_FAILED", "Gagal membuat ujian", err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "exam.created", EntityType: "exam", EntityID: exam.ID})
	return toResponse(exam, 0, false), nil
}

func (s *Service) List(ctx context.Context, actor authz.Principal, courseID string, page pagination.Request) ([]Response, int64, error) {
	if err := s.academics.RequireCourseManager(ctx, actor, courseID); err != nil {
		return nil, 0, err
	}
	values, counts, total, err := s.repository.List(ctx, courseID, page)
	if err != nil {
		return nil, 0, apperror.Wrap(http.StatusInternalServerError, "EXAMS_READ_FAILED", "Gagal memuat ujian", err)
	}
	result := make([]Response, len(values))
	for i, value := range values {
		result[i] = toResponse(value, counts[value.ID], false)
	}
	return result, total, nil
}

func (s *Service) Find(ctx context.Context, actor authz.Principal, id string) (Response, error) {
	exam, participantCount, err := s.repository.Find(ctx, id)
	if err != nil {
		return Response{}, mapExamError(err)
	}
	if err := s.academics.RequireCourseManager(ctx, actor, exam.CourseID); err != nil {
		return Response{}, err
	}
	participantIDs, err := s.repository.ParticipantIDs(ctx, id)
	if err != nil {
		return Response{}, apperror.Wrap(http.StatusInternalServerError, "EXAM_PARTICIPANTS_READ_FAILED", "Gagal memuat peserta ujian", err)
	}
	result := toResponse(exam, participantCount, true)
	result.ParticipantIDs = participantIDs
	return result, nil
}

func (s *Service) Update(ctx context.Context, actor authz.Principal, id string, request WriteRequest) (Response, error) {
	if err := requireTeacher(actor); err != nil {
		return Response{}, err
	}
	current, participantCount, err := s.repository.Find(ctx, id)
	if err != nil {
		return Response{}, mapExamError(err)
	}
	if current.CourseID != request.CourseID {
		return Response{}, apperror.New(http.StatusConflict, "EXAM_COURSE_IMMUTABLE", "Course ujian tidak dapat dipindahkan")
	}
	if err := s.academics.RequireCourseManager(ctx, actor, current.CourseID); err != nil {
		return Response{}, err
	}
	if err := validateConfiguration(request); err != nil {
		return Response{}, err
	}
	updated, err := s.repository.UpdateDraft(ctx, id, func(exam *Exam) {
		exam.Title = strings.TrimSpace(request.Title)
		exam.Description = strings.TrimSpace(request.Description)
		exam.StartsAt = request.StartsAt.UTC()
		exam.EndsAt = request.EndsAt.UTC()
		exam.DurationMinutes = request.DurationMinutes
		exam.AllowBackNavigation = request.AllowBackNavigation
	})
	if err != nil {
		return Response{}, mapExamError(err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "exam.updated", EntityType: "exam", EntityID: id})
	return toResponse(updated, participantCount, false), nil
}

func (s *Service) SetQuestions(ctx context.Context, actor authz.Principal, id string, request SetQuestionsRequest) error {
	exam, _, err := s.mutableExam(ctx, actor, id)
	if err != nil {
		return err
	}
	if err := s.repository.ReplaceQuestions(ctx, exam, request.Questions); err != nil {
		return mapExamError(err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "exam.questions_replaced", EntityType: "exam", EntityID: id, Metadata: map[string]any{"count": len(request.Questions)}})
	return nil
}

func (s *Service) SetParticipants(ctx context.Context, actor authz.Principal, id string, request SetParticipantsRequest) error {
	exam, _, err := s.mutableExam(ctx, actor, id)
	if err != nil {
		return err
	}
	if err := s.repository.ReplaceParticipants(ctx, exam, request.StudentIDs); err != nil {
		return mapExamError(err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "exam.participants_replaced", EntityType: "exam", EntityID: id, Metadata: map[string]any{"count": len(request.StudentIDs)}})
	return nil
}

func (s *Service) Publish(ctx context.Context, actor authz.Principal, id string) error {
	exam, _, err := s.mutableExam(ctx, actor, id)
	if err != nil {
		return err
	}
	if !exam.EndsAt.After(s.now().UTC()) {
		return apperror.New(http.StatusConflict, "EXAM_SCHEDULE_ENDED", "Ujian yang jadwalnya telah berakhir tidak dapat dipublikasikan")
	}
	if err := s.repository.Publish(ctx, id, s.now().UTC()); err != nil {
		return mapExamError(err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "exam.published", EntityType: "exam", EntityID: id})
	return nil
}

func (s *Service) Unpublish(ctx context.Context, actor authz.Principal, id string) error {
	if err := requireTeacher(actor); err != nil {
		return err
	}
	exam, _, err := s.repository.Find(ctx, id)
	if err != nil {
		return mapExamError(err)
	}
	if err := s.academics.RequireCourseManager(ctx, actor, exam.CourseID); err != nil {
		return err
	}
	if err := s.repository.Unpublish(ctx, id); err != nil {
		return mapExamError(err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "exam.unpublished", EntityType: "exam", EntityID: id})
	return nil
}

func (s *Service) mutableExam(ctx context.Context, actor authz.Principal, id string) (Exam, int64, error) {
	if err := requireTeacher(actor); err != nil {
		return Exam{}, 0, err
	}
	exam, count, err := s.repository.Find(ctx, id)
	if err != nil {
		return Exam{}, 0, mapExamError(err)
	}
	if err := s.academics.RequireCourseManager(ctx, actor, exam.CourseID); err != nil {
		return Exam{}, 0, err
	}
	if exam.Status != StatusDraft {
		return Exam{}, 0, apperror.New(http.StatusConflict, "EXAM_LOCKED", "Konfigurasi ujian terkunci karena sudah dipublikasikan")
	}
	return exam, count, nil
}

func requireTeacher(actor authz.Principal) error {
	if actor.Role != string(users.RoleTeacher) {
		return apperror.New(http.StatusForbidden, "EXAM_WRITE_DENIED", "Hanya guru yang ditugaskan dapat mengelola ujian")
	}
	return nil
}

func validateConfiguration(request WriteRequest) error {
	if !request.EndsAt.After(request.StartsAt) {
		return apperror.New(http.StatusUnprocessableEntity, "EXAM_SCHEDULE_INVALID", "Waktu selesai harus setelah waktu mulai")
	}
	if float64(request.DurationMinutes) > request.EndsAt.Sub(request.StartsAt).Minutes() {
		return apperror.New(http.StatusUnprocessableEntity, "EXAM_DURATION_INVALID", "Durasi ujian tidak boleh melebihi window jadwal")
	}
	return nil
}

func mapExamError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return apperror.New(http.StatusNotFound, "EXAM_NOT_FOUND", "Ujian tidak ditemukan")
	case errors.Is(err, ErrExamLocked):
		return apperror.New(http.StatusConflict, "EXAM_LOCKED", "Ujian tidak dapat diubah pada status saat ini")
	case errors.Is(err, ErrInvalidSelection):
		return apperror.New(http.StatusUnprocessableEntity, "EXAM_CONFIGURATION_INVALID", "Soal atau peserta ujian belum valid")
	default:
		return apperror.Wrap(http.StatusInternalServerError, "EXAM_WRITE_FAILED", "Gagal memproses ujian", err)
	}
}
