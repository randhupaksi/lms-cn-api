package monitoring

import (
	"context"
	"errors"
	"net/http"
	"time"

	"lms-cn-api/internal/authz"
	"lms-cn-api/internal/modules/academics"
	"lms-cn-api/pkg/apperror"
	"lms-cn-api/pkg/pagination"
)

type Service struct {
	repository *Repository
	academics  *academics.Service
	now        func() time.Time
}

func NewService(repository *Repository, academicsService *academics.Service) *Service {
	return &Service{repository: repository, academics: academicsService, now: time.Now}
}

func (s *Service) ExamStatus(ctx context.Context, actor authz.Principal, examID string) (Summary, error) {
	courseID, err := s.repository.ExamCourseID(ctx, examID)
	if errors.Is(err, ErrExamNotFound) {
		return Summary{}, apperror.New(http.StatusNotFound, "EXAM_NOT_FOUND", "Ujian tidak ditemukan")
	}
	if err != nil {
		return Summary{}, apperror.Wrap(http.StatusInternalServerError, "MONITORING_READ_FAILED", "Gagal memuat monitoring ujian", err)
	}
	if err := s.academics.RequireCourseManager(ctx, actor, courseID); err != nil {
		return Summary{}, err
	}
	participants, err := s.repository.Participants(ctx, examID)
	if err != nil {
		return Summary{}, apperror.Wrap(http.StatusInternalServerError, "MONITORING_READ_FAILED", "Gagal memuat monitoring ujian", err)
	}
	now := s.now().UTC()
	result := Summary{ExamID: examID, ServerTime: now, Total: len(participants), Participants: participants}
	for index := range result.Participants {
		participant := &result.Participants[index]
		if participant.Status == "in_progress" && participant.DeadlineAt != nil && !now.Before(*participant.DeadlineAt) {
			participant.Status = "expired"
		}
		switch participant.Status {
		case "not_started":
			result.NotStarted++
		case "in_progress":
			result.InProgress++
		case "submitted":
			result.Submitted++
		case "expired":
			result.Expired++
		}
	}
	return result, nil
}

func (s *Service) ExamStatusPage(ctx context.Context, actor authz.Principal, examID, search string, page pagination.Request) (Summary, int64, error) {
	courseID, err := s.repository.ExamCourseID(ctx, examID)
	if errors.Is(err, ErrExamNotFound) {
		return Summary{}, 0, apperror.New(http.StatusNotFound, "EXAM_NOT_FOUND", "Ujian tidak ditemukan")
	}
	if err != nil {
		return Summary{}, 0, apperror.Wrap(http.StatusInternalServerError, "MONITORING_READ_FAILED", "Gagal memuat monitoring ujian", err)
	}
	if err := s.academics.RequireCourseManager(ctx, actor, courseID); err != nil {
		return Summary{}, 0, err
	}
	now := s.now().UTC()
	counts, err := s.repository.ParticipantCounts(ctx, examID, now)
	if err != nil {
		return Summary{}, 0, apperror.Wrap(http.StatusInternalServerError, "MONITORING_READ_FAILED", "Gagal memuat monitoring ujian", err)
	}
	participants, total, err := s.repository.ParticipantsPage(ctx, examID, search, page)
	if err != nil {
		return Summary{}, 0, apperror.Wrap(http.StatusInternalServerError, "MONITORING_READ_FAILED", "Gagal memuat monitoring ujian", err)
	}
	for index := range participants {
		participant := &participants[index]
		if participant.Status == "in_progress" && participant.DeadlineAt != nil && !now.Before(*participant.DeadlineAt) {
			participant.Status = "expired"
		}
	}
	return Summary{ExamID: examID, ServerTime: now, Total: counts.Total, NotStarted: counts.NotStarted, InProgress: counts.InProgress, Submitted: counts.Submitted, Expired: counts.Expired, Participants: participants}, total, nil
}
