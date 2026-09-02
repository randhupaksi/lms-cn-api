package materials

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"lms-cn-api/internal/authz"
	"lms-cn-api/internal/modules/academics"
	"lms-cn-api/internal/modules/audit"
	"lms-cn-api/pkg/apperror"

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

func (s *Service) List(ctx context.Context, actor authz.Principal, courseID string) ([]Response, error) {
	studentID := ""
	publishedOnly := false
	if actor.Role == "student" {
		if err := s.academics.RequireCourseStudent(ctx, actor, courseID); err != nil {
			return nil, err
		}
		studentID, publishedOnly = actor.UserID, true
	} else if err := s.academics.RequireCourseManager(ctx, actor, courseID); err != nil {
		return nil, err
	}
	rows, err := s.repository.List(ctx, courseID, studentID, publishedOnly)
	if err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, "MATERIALS_READ_FAILED", "Gagal memuat materi", err)
	}
	result := make([]Response, len(rows))
	for index, row := range rows {
		result[index] = toResponse(row)
	}
	return result, nil
}

func (s *Service) Create(ctx context.Context, actor authz.Principal, request WriteRequest) (Response, error) {
	if actor.Role != "teacher" {
		return Response{}, apperror.New(http.StatusForbidden, "MATERIAL_WRITE_DENIED", "Hanya guru yang ditugaskan dapat mengelola materi")
	}
	if err := s.academics.RequireCourseManager(ctx, actor, request.CourseID); err != nil {
		return Response{}, err
	}
	material := Material{ID: uuid.NewString(), CourseID: request.CourseID, AuthorID: actor.UserID, Title: strings.TrimSpace(request.Title), Description: strings.TrimSpace(request.Description), Content: strings.TrimSpace(request.Content), Position: request.Position, Status: StatusDraft}
	if err := s.repository.Create(ctx, &material); err != nil {
		return Response{}, apperror.Wrap(http.StatusInternalServerError, "MATERIAL_CREATE_FAILED", "Gagal membuat materi", err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "material.created", EntityType: "material", EntityID: material.ID})
	return toResponse(materialRow{Material: material}), nil
}

func (s *Service) Update(ctx context.Context, actor authz.Principal, id string, request WriteRequest) (Response, error) {
	material, err := s.requireTeacherOwner(ctx, actor, id, request.CourseID)
	if err != nil {
		return Response{}, err
	}
	updated, err := s.repository.UpdateDraft(ctx, material.ID, func(value *Material) {
		value.Title = strings.TrimSpace(request.Title)
		value.Description = strings.TrimSpace(request.Description)
		value.Content = strings.TrimSpace(request.Content)
		value.Position = request.Position
	})
	if err != nil {
		return Response{}, mapError(err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "material.updated", EntityType: "material", EntityID: id})
	return toResponse(materialRow{Material: updated}), nil
}

func (s *Service) Publish(ctx context.Context, actor authz.Principal, id string) error {
	material, err := s.requireTeacherOwner(ctx, actor, id, "")
	if err != nil {
		return err
	}
	if err := s.repository.Publish(ctx, material.ID, s.now().UTC()); err != nil {
		return mapError(err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "material.published", EntityType: "material", EntityID: id})
	return nil
}

func (s *Service) Complete(ctx context.Context, actor authz.Principal, id string) error {
	if actor.Role != "student" {
		return apperror.New(http.StatusForbidden, "MATERIAL_PROGRESS_DENIED", "Progress materi hanya tersedia untuk siswa")
	}
	material, err := s.repository.Find(ctx, id)
	if err != nil {
		return mapError(err)
	}
	if material.Status != StatusPublished {
		return apperror.New(http.StatusNotFound, "MATERIAL_NOT_FOUND", "Materi tidak ditemukan")
	}
	if err := s.academics.RequireCourseStudent(ctx, actor, material.CourseID); err != nil {
		return err
	}
	if err := s.repository.Complete(ctx, id, actor.UserID, s.now().UTC()); err != nil {
		return apperror.Wrap(http.StatusInternalServerError, "MATERIAL_PROGRESS_FAILED", "Gagal menyimpan progress materi", err)
	}
	return nil
}

func (s *Service) requireTeacherOwner(ctx context.Context, actor authz.Principal, id, requestedCourseID string) (Material, error) {
	if actor.Role != "teacher" {
		return Material{}, apperror.New(http.StatusForbidden, "MATERIAL_WRITE_DENIED", "Hanya guru yang ditugaskan dapat mengelola materi")
	}
	material, err := s.repository.Find(ctx, id)
	if err != nil {
		return Material{}, mapError(err)
	}
	if requestedCourseID != "" && requestedCourseID != material.CourseID {
		return Material{}, apperror.New(http.StatusConflict, "MATERIAL_COURSE_IMMUTABLE", "Course materi tidak dapat dipindahkan")
	}
	if err := s.academics.RequireCourseManager(ctx, actor, material.CourseID); err != nil {
		return Material{}, err
	}
	return material, nil
}

func mapError(err error) error {
	if errors.Is(err, ErrNotFound) {
		return apperror.New(http.StatusNotFound, "MATERIAL_NOT_FOUND", "Materi tidak ditemukan")
	}
	if errors.Is(err, ErrLocked) {
		return apperror.New(http.StatusConflict, "MATERIAL_LOCKED", "Materi yang telah dipublikasikan tidak dapat diubah")
	}
	return apperror.Wrap(http.StatusInternalServerError, "MATERIAL_WRITE_FAILED", "Gagal memproses materi", err)
}
