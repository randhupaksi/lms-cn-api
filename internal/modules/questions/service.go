package questions

import (
	"context"
	"errors"
	"net/http"
	"strings"

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
}

func NewService(repository *Repository, academicsService *academics.Service, auditRecorder audit.Recorder) *Service {
	return &Service{repository: repository, academics: academicsService, audit: auditRecorder}
}

func (s *Service) Create(ctx context.Context, actor authz.Principal, request WriteRequest) (Response, error) {
	if actor.Role != string(users.RoleTeacher) {
		return Response{}, apperror.New(http.StatusForbidden, "QUESTION_WRITE_DENIED", "Hanya guru yang ditugaskan dapat membuat soal")
	}
	if err := s.academics.RequireCourseManager(ctx, actor, request.CourseID); err != nil {
		return Response{}, err
	}
	if err := validateOptions(request.Options); err != nil {
		return Response{}, err
	}
	question := Question{
		ID: uuid.NewString(), CourseID: request.CourseID, AuthorID: actor.UserID,
		Type: request.Type, Stem: strings.TrimSpace(request.Stem), DefaultPoints: request.DefaultPoints,
		Category: strings.TrimSpace(request.Category), Tags: normalizeTags(request.Tags),
		Status: "active", Version: 1,
	}
	question.Options = buildOptions(question.ID, request.Options)
	if err := s.repository.Create(ctx, &question); err != nil {
		return Response{}, apperror.Wrap(http.StatusInternalServerError, "QUESTION_CREATE_FAILED", "Gagal membuat soal", err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "question.created", EntityType: "question", EntityID: question.ID})
	return toResponse(question), nil
}

func (s *Service) List(ctx context.Context, actor authz.Principal, courseID string, page pagination.Request, filter ListFilter) ([]Response, int64, error) {
	if err := s.academics.RequireCourseManager(ctx, actor, courseID); err != nil {
		return nil, 0, err
	}
	filter.Category = strings.TrimSpace(filter.Category)
	filter.Tag = strings.ToLower(strings.TrimSpace(filter.Tag))
	filter.Search = strings.TrimSpace(filter.Search)
	if filter.Status != "" && filter.Status != "active" && filter.Status != "archived" {
		return nil, 0, apperror.New(http.StatusBadRequest, "QUESTION_STATUS_INVALID", "Status soal tidak valid")
	}
	values, total, err := s.repository.List(ctx, courseID, page, filter)
	if err != nil {
		return nil, 0, apperror.Wrap(http.StatusInternalServerError, "QUESTIONS_READ_FAILED", "Gagal memuat bank soal", err)
	}
	result := make([]Response, len(values))
	for i, value := range values {
		result[i] = toResponse(value)
	}
	return result, total, nil
}

func (s *Service) Update(ctx context.Context, actor authz.Principal, id string, request WriteRequest) (Response, error) {
	if actor.Role != string(users.RoleTeacher) {
		return Response{}, apperror.New(http.StatusForbidden, "QUESTION_WRITE_DENIED", "Hanya guru yang ditugaskan dapat mengubah soal")
	}
	current, err := s.repository.Find(ctx, id)
	if err != nil {
		return Response{}, mapQuestionError(err)
	}
	if current.CourseID != request.CourseID {
		return Response{}, apperror.New(http.StatusConflict, "QUESTION_COURSE_IMMUTABLE", "Course soal tidak dapat dipindahkan")
	}
	if err := s.academics.RequireCourseManager(ctx, actor, current.CourseID); err != nil {
		return Response{}, err
	}
	if err := validateOptions(request.Options); err != nil {
		return Response{}, err
	}
	updated, err := s.repository.Update(ctx, id, func(question *Question) {
		question.Type = request.Type
		question.Stem = strings.TrimSpace(request.Stem)
		question.DefaultPoints = request.DefaultPoints
		question.Category = strings.TrimSpace(request.Category)
		question.Tags = normalizeTags(request.Tags)
		question.Options = buildOptions(question.ID, request.Options)
	})
	if err != nil {
		return Response{}, mapQuestionError(err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "question.updated", EntityType: "question", EntityID: id, Metadata: map[string]any{"version": updated.Version}})
	return toResponse(updated), nil
}

func (s *Service) Archive(ctx context.Context, actor authz.Principal, id string) error {
	if actor.Role != string(users.RoleTeacher) {
		return apperror.New(http.StatusForbidden, "QUESTION_WRITE_DENIED", "Hanya guru yang ditugaskan dapat mengarsipkan soal")
	}
	question, err := s.repository.Find(ctx, id)
	if err != nil {
		return mapQuestionError(err)
	}
	if err := s.academics.RequireCourseManager(ctx, actor, question.CourseID); err != nil {
		return err
	}
	if err := s.repository.Archive(ctx, id); err != nil {
		return mapQuestionError(err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "question.archived", EntityType: "question", EntityID: id})
	return nil
}

func validateOptions(options []OptionRequest) error {
	correct := 0
	for _, option := range options {
		if strings.TrimSpace(option.Content) == "" {
			return apperror.New(http.StatusUnprocessableEntity, "QUESTION_OPTION_EMPTY", "Semua opsi jawaban wajib diisi")
		}
		if option.IsCorrect {
			correct++
		}
	}
	if correct != 1 {
		return apperror.New(http.StatusUnprocessableEntity, "QUESTION_CORRECT_OPTION_INVALID", "Pilihan ganda harus memiliki tepat satu jawaban benar")
	}
	return nil
}

func buildOptions(questionID string, requests []OptionRequest) []Option {
	result := make([]Option, len(requests))
	for index, option := range requests {
		result[index] = Option{ID: uuid.NewString(), QuestionID: questionID, Content: strings.TrimSpace(option.Content), IsCorrect: option.IsCorrect, Position: uint(index + 1)}
	}
	return result
}

func normalizeTags(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		tag := strings.ToLower(strings.TrimSpace(value))
		if tag == "" {
			continue
		}
		if _, exists := seen[tag]; exists {
			continue
		}
		seen[tag] = struct{}{}
		result = append(result, tag)
	}
	return result
}

func mapQuestionError(err error) error {
	if errors.Is(err, ErrNotFound) {
		return apperror.New(http.StatusNotFound, "QUESTION_NOT_FOUND", "Soal tidak ditemukan")
	}
	return apperror.Wrap(http.StatusInternalServerError, "QUESTION_WRITE_FAILED", "Gagal memproses soal", err)
}
