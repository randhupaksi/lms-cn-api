package analytics

import (
	"context"
	"errors"
	"net/http"
	"time"

	"lms-cn-api/internal/authz"
	"lms-cn-api/internal/modules/academics"
	"lms-cn-api/pkg/apperror"
)

type Service struct {
	repository *Repository
	academics  *academics.Service
	now        func() time.Time
}

func NewService(repository *Repository, academicsService *academics.Service) *Service {
	return &Service{repository: repository, academics: academicsService, now: time.Now}
}

func (s *Service) Dashboard(ctx context.Context, actor authz.Principal) (Dashboard, error) {
	result := Dashboard{Role: actor.Role}
	now := s.now().UTC()
	switch actor.Role {
	case "admin":
		counts, err := s.repository.AdminDashboardCounts(ctx, now)
		if err != nil {
			return Dashboard{}, analyticsReadError(err)
		}
		result.Metrics = []Metric{
			{Key: "active_users", Label: "Pengguna aktif", Value: float64(counts.ActiveUsers)},
			{Key: "active_courses", Label: "Course aktif", Value: float64(counts.ActiveCourses)},
			{Key: "published_exams", Label: "Ujian published", Value: float64(counts.PublishedExams)},
			{Key: "active_attempts", Label: "Attempt berlangsung", Value: float64(counts.ActiveAttempts)},
		}
	case "teacher":
		counts, err := s.repository.TeacherDashboardCounts(ctx, actor.UserID)
		if err != nil {
			return Dashboard{}, analyticsReadError(err)
		}
		result.Metrics = []Metric{
			{Key: "courses", Label: "Course saya", Value: float64(counts.Courses)},
			{Key: "questions", Label: "Bank soal", Value: float64(counts.Questions)},
			{Key: "exams", Label: "Ujian dikelola", Value: float64(counts.Exams)},
			{Key: "unpublished_results", Label: "Hasil belum published", Value: float64(counts.UnpublishedResults)},
		}
	case "student":
		counts, err := s.repository.StudentDashboardCounts(ctx, actor.UserID, now)
		if err != nil {
			return Dashboard{}, analyticsReadError(err)
		}
		result.Metrics = []Metric{
			{Key: "courses", Label: "Course saya", Value: float64(counts.Courses)},
			{Key: "available_exams", Label: "Ujian tersedia", Value: float64(counts.AvailableExams)},
			{Key: "published_results", Label: "Hasil tersedia", Value: float64(counts.PublishedResults)},
			{Key: "completed_materials", Label: "Materi selesai", Value: float64(counts.CompletedMaterials)},
		}
	default:
		return Dashboard{}, apperror.New(http.StatusForbidden, "ANALYTICS_ACCESS_DENIED", "Ringkasan tidak tersedia untuk akun ini")
	}
	return result, nil
}

func analyticsReadError(err error) error {
	return apperror.Wrap(http.StatusInternalServerError, "ANALYTICS_READ_FAILED", "Gagal memuat ringkasan", err)
}

func (s *Service) Exam(ctx context.Context, actor authz.Principal, examID string) (ExamSummary, error) {
	courseID, err := s.repository.CourseID(ctx, examID)
	if errors.Is(err, ErrExamNotFound) {
		return ExamSummary{}, apperror.New(http.StatusNotFound, "EXAM_NOT_FOUND", "Ujian tidak ditemukan")
	}
	if err != nil {
		return ExamSummary{}, apperror.Wrap(http.StatusInternalServerError, "ANALYTICS_READ_FAILED", "Gagal memuat analitik ujian", err)
	}
	if err := s.academics.RequireCourseManager(ctx, actor, courseID); err != nil {
		return ExamSummary{}, err
	}
	summary, err := s.repository.ExamSummary(ctx, examID)
	if err != nil {
		return ExamSummary{}, apperror.Wrap(http.StatusInternalServerError, "ANALYTICS_READ_FAILED", "Gagal memuat analitik ujian", err)
	}
	summary.Items, err = s.repository.ItemAnalysis(ctx, examID)
	if err != nil {
		return ExamSummary{}, apperror.Wrap(http.StatusInternalServerError, "ANALYTICS_READ_FAILED", "Gagal memuat analisis soal", err)
	}
	return summary, nil
}
