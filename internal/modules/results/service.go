package results

import (
	"context"
	"errors"
	"net/http"
	"time"

	"lms-cn-api/internal/authz"
	"lms-cn-api/internal/modules/academics"
	"lms-cn-api/internal/modules/audit"
	"lms-cn-api/internal/modules/users"
	"lms-cn-api/pkg/apperror"
	"lms-cn-api/pkg/pagination"
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

func (s *Service) ListByExam(ctx context.Context, actor authz.Principal, examID string, page pagination.Request) ([]Response, int64, error) {
	courseID, err := s.repository.ExamCourseID(ctx, examID)
	if err != nil {
		return nil, 0, mapResultError(err)
	}
	if err := s.academics.RequireCourseManager(ctx, actor, courseID); err != nil {
		return nil, 0, err
	}
	rows, total, err := s.repository.ListByExam(ctx, examID, page)
	if err != nil {
		return nil, 0, mapResultError(err)
	}
	result := make([]Response, len(rows))
	for i, row := range rows {
		result[i] = toResponse(row, true)
	}
	return result, total, nil
}

func (s *Service) PublishByExam(ctx context.Context, actor authz.Principal, examID string) (int64, error) {
	if actor.Role != string(users.RoleTeacher) {
		return 0, apperror.New(http.StatusForbidden, "RESULT_PUBLISH_DENIED", "Hanya guru yang ditugaskan dapat mempublikasikan hasil")
	}
	courseID, err := s.repository.ExamCourseID(ctx, examID)
	if err != nil {
		return 0, mapResultError(err)
	}
	if err := s.academics.RequireCourseManager(ctx, actor, courseID); err != nil {
		return 0, err
	}
	count, err := s.repository.PublishByExam(ctx, examID, s.now().UTC())
	if err != nil {
		return 0, mapResultError(err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "results.published", EntityType: "exam", EntityID: examID, Metadata: map[string]any{"count": count}})
	return count, nil
}

func (s *Service) ListStudent(ctx context.Context, actor authz.Principal, page pagination.Request) ([]Response, int64, error) {
	if actor.Role != string(users.RoleStudent) {
		return nil, 0, apperror.New(http.StatusForbidden, "STUDENT_ONLY", "Hasil ini hanya tersedia untuk siswa")
	}
	rows, total, err := s.repository.ListPublishedForStudent(ctx, actor.UserID, page)
	if err != nil {
		return nil, 0, mapResultError(err)
	}
	result := make([]Response, len(rows))
	for i, row := range rows {
		result[i] = toResponse(row, false)
	}
	return result, total, nil
}

func (s *Service) FindStudent(ctx context.Context, actor authz.Principal, id string) (Response, error) {
	if actor.Role != string(users.RoleStudent) {
		return Response{}, apperror.New(http.StatusForbidden, "STUDENT_ONLY", "Hasil ini hanya tersedia untuk siswa")
	}
	row, err := s.repository.FindPublishedForStudent(ctx, id, actor.UserID)
	if err != nil {
		return Response{}, mapResultError(err)
	}
	return toResponse(row, false), nil
}

func mapResultError(err error) error {
	if errors.Is(err, ErrNotFound) {
		return apperror.New(http.StatusNotFound, "RESULT_NOT_FOUND", "Hasil tidak ditemukan atau belum dipublikasikan")
	}
	return apperror.Wrap(http.StatusInternalServerError, "RESULT_READ_FAILED", "Gagal memproses hasil ujian", err)
}
