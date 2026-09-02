package monitoring

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
