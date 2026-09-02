package assignments

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
	studentID, publishedOnly := "", false
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
		return nil, apperror.Wrap(http.StatusInternalServerError, "ASSIGNMENTS_READ_FAILED", "Gagal memuat tugas", err)
	}
	result := make([]Response, len(rows))
	for index, row := range rows {
		result[index] = toResponse(row)
	}
	return result, nil
}

func (s *Service) Create(ctx context.Context, actor authz.Principal, request WriteRequest) (Response, error) {
	if actor.Role != "teacher" {
		return Response{}, apperror.New(http.StatusForbidden, "ASSIGNMENT_WRITE_DENIED", "Hanya guru yang ditugaskan dapat mengelola tugas")
	}
	if err := s.academics.RequireCourseManager(ctx, actor, request.CourseID); err != nil {
		return Response{}, err
	}
	if !request.DueAt.After(s.now().UTC()) {
		return Response{}, apperror.New(http.StatusUnprocessableEntity, "ASSIGNMENT_DUE_INVALID", "Batas pengumpulan harus berada di masa depan")
	}
	assignment := Assignment{ID: uuid.NewString(), CourseID: request.CourseID, AuthorID: actor.UserID, Title: strings.TrimSpace(request.Title), Instructions: strings.TrimSpace(request.Instructions), DueAt: request.DueAt.UTC(), MaxScore: request.MaxScore, Status: StatusDraft}
	if err := s.repository.Create(ctx, &assignment); err != nil {
		return Response{}, apperror.Wrap(http.StatusInternalServerError, "ASSIGNMENT_CREATE_FAILED", "Gagal membuat tugas", err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "assignment.created", EntityType: "assignment", EntityID: assignment.ID})
	return toResponse(assignmentRow{Assignment: assignment}), nil
}

func (s *Service) Update(ctx context.Context, actor authz.Principal, id string, request WriteRequest) (Response, error) {
	assignment, err := s.requireTeacherOwner(ctx, actor, id)
	if err != nil {
		return Response{}, err
	}
	if request.CourseID != assignment.CourseID {
		return Response{}, apperror.New(http.StatusConflict, "ASSIGNMENT_COURSE_IMMUTABLE", "Course tugas tidak dapat dipindahkan")
	}
	updated, err := s.repository.UpdateDraft(ctx, id, func(value *Assignment) {
		value.Title = strings.TrimSpace(request.Title)
		value.Instructions = strings.TrimSpace(request.Instructions)
		value.DueAt = request.DueAt.UTC()
		value.MaxScore = request.MaxScore
	})
	if err != nil {
		return Response{}, mapError(err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "assignment.updated", EntityType: "assignment", EntityID: id})
	return toResponse(assignmentRow{Assignment: updated}), nil
}

func (s *Service) Publish(ctx context.Context, actor authz.Principal, id string) error {
	assignment, err := s.requireTeacherOwner(ctx, actor, id)
	if err != nil {
		return err
	}
	if !assignment.DueAt.After(s.now().UTC()) {
		return apperror.New(http.StatusConflict, "ASSIGNMENT_DUE_PASSED", "Tugas dengan batas waktu lampau tidak dapat dipublikasikan")
	}
	if err := s.repository.Publish(ctx, id, s.now().UTC()); err != nil {
		return mapError(err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "assignment.published", EntityType: "assignment", EntityID: id})
	return nil
}

func (s *Service) Submit(ctx context.Context, actor authz.Principal, id string, request SubmitRequest) error {
	if actor.Role != "student" {
		return apperror.New(http.StatusForbidden, "ASSIGNMENT_SUBMIT_DENIED", "Pengumpulan tugas hanya tersedia untuk siswa")
	}
	assignment, err := s.repository.Find(ctx, id)
	if err != nil {
		return mapError(err)
	}
	if assignment.Status != StatusPublished || !s.now().UTC().Before(assignment.DueAt) {
		return apperror.New(http.StatusConflict, "ASSIGNMENT_CLOSED", "Tugas tidak tersedia untuk dikumpulkan")
	}
	if err := s.academics.RequireCourseStudent(ctx, actor, assignment.CourseID); err != nil {
		return err
	}
	now := s.now().UTC()
	submission := Submission{ID: uuid.NewString(), AssignmentID: id, StudentID: actor.UserID, Content: strings.TrimSpace(request.Content), AttachmentURL: strings.TrimSpace(request.AttachmentURL), Status: "submitted", SubmittedAt: now}
	if err := s.repository.Submit(ctx, &submission); err != nil {
		return apperror.Wrap(http.StatusInternalServerError, "ASSIGNMENT_SUBMIT_FAILED", "Gagal mengumpulkan tugas", err)
	}
	return nil
}

func (s *Service) ListSubmissions(ctx context.Context, actor authz.Principal, assignmentID string) ([]SubmissionResponse, error) {
	if _, err := s.requireTeacherOwner(ctx, actor, assignmentID); err != nil {
		return nil, err
	}
	rows, err := s.repository.ListSubmissions(ctx, assignmentID)
	if err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, "SUBMISSIONS_READ_FAILED", "Gagal memuat pengumpulan tugas", err)
	}
	result := make([]SubmissionResponse, len(rows))
	for index, row := range rows {
		result[index] = toSubmissionResponse(row)
	}
	return result, nil
}

func (s *Service) Grade(ctx context.Context, actor authz.Principal, submissionID string, request GradeRequest) error {
	assignment, err := s.repository.SubmissionAssignment(ctx, submissionID)
	if err != nil {
		return mapError(err)
	}
	if actor.Role != "teacher" {
		return apperror.New(http.StatusForbidden, "ASSIGNMENT_GRADE_DENIED", "Hanya guru yang ditugaskan dapat memberi nilai")
	}
	if err := s.academics.RequireCourseManager(ctx, actor, assignment.CourseID); err != nil {
		return err
	}
	if request.Score > assignment.MaxScore {
		return apperror.New(http.StatusUnprocessableEntity, "ASSIGNMENT_SCORE_INVALID", "Nilai tidak boleh melebihi skor maksimal")
	}
	if err := s.repository.Grade(ctx, submissionID, actor.UserID, request.Score, strings.TrimSpace(request.Feedback), s.now().UTC()); err != nil {
		return mapError(err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "assignment.graded", EntityType: "assignment_submission", EntityID: submissionID, Metadata: map[string]any{"score": request.Score}})
	return nil
}

func (s *Service) requireTeacherOwner(ctx context.Context, actor authz.Principal, id string) (Assignment, error) {
	if actor.Role != "teacher" {
		return Assignment{}, apperror.New(http.StatusForbidden, "ASSIGNMENT_WRITE_DENIED", "Hanya guru yang ditugaskan dapat mengelola tugas")
	}
	assignment, err := s.repository.Find(ctx, id)
	if err != nil {
		return Assignment{}, mapError(err)
	}
	if err := s.academics.RequireCourseManager(ctx, actor, assignment.CourseID); err != nil {
		return Assignment{}, err
	}
	return assignment, nil
}

func mapError(err error) error {
	if errors.Is(err, ErrNotFound) {
		return apperror.New(http.StatusNotFound, "ASSIGNMENT_NOT_FOUND", "Tugas tidak ditemukan")
	}
	if errors.Is(err, ErrLocked) {
		return apperror.New(http.StatusConflict, "ASSIGNMENT_LOCKED", "Tugas yang telah dipublikasikan tidak dapat diubah")
	}
	return apperror.Wrap(http.StatusInternalServerError, "ASSIGNMENT_WRITE_FAILED", "Gagal memproses tugas", err)
}
