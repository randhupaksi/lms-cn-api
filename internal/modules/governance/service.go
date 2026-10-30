package governance

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"lms-cn-api/internal/authz"
	"lms-cn-api/pkg/apperror"
	"net/http"
	"sort"
	"strings"
	"time"
)

type CourseAccess interface {
	RequireCourseManager(context.Context, authz.Principal, string) error
}
type Service struct {
	repository *Repository
	courses    CourseAccess
	now        func() time.Time
}

func NewService(repository *Repository, courses CourseAccess) *Service {
	return &Service{repository: repository, courses: courses, now: time.Now}
}
func owner(actor authz.Principal) error { return actor.RequireRole("superadmin", "principal") }

type PublicProfile struct {
	SchoolName     string `json:"school_name"`
	Motto          string `json:"motto"`
	ContactEmail   string `json:"contact_email"`
	SupportMessage string `json:"support_message"`
}

func (s *Service) PublicProfile(ctx context.Context) (PublicProfile, error) {
	v, err := s.repository.Settings(ctx)
	return PublicProfile{SchoolName: v.SchoolName, Motto: v.Motto, ContactEmail: v.ContactEmail, SupportMessage: v.SupportMessage}, failure(err)
}
func staff(actor authz.Principal) error {
	return actor.RequireRole("superadmin", "principal", "admin", "teacher")
}
func failure(err error) error {
	if err == nil {
		return nil
	}
	var app *apperror.Error
	if errors.As(err, &app) {
		return err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperror.New(404, "GOVERNANCE_NOT_FOUND", "Data tidak ditemukan")
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return apperror.New(409, "PENDING_REQUEST_EXISTS", "Ujian ini sudah memiliki pengajuan yang menunggu keputusan")
	}
	return apperror.Wrap(500, "GOVERNANCE_FAILED", "Data tata kelola belum dapat diproses", err)
}
func decorate(row *ExamReadiness, now time.Time) {
	row.Issues = []string{}
	if row.QuestionCount == 0 {
		row.Issues = append(row.Issues, "Belum ada soal")
	}
	if row.ParticipantCount == 0 {
		row.Issues = append(row.Issues, "Belum ada peserta")
	}
	if row.TeacherCount == 0 {
		row.Issues = append(row.Issues, "Belum ada guru aktif")
	}
	if row.InactiveParticipants > 0 {
		row.Issues = append(row.Issues, "Ada peserta nonaktif")
	}
	if row.ConflictCount > 0 {
		row.Issues = append(row.Issues, "Jadwal kelas bertabrakan")
	}
	row.Lifecycle = "running"
	if row.Status == "draft" {
		row.Lifecycle = "draft"
	} else if row.Status == "closed" || row.Status == "archived" || !row.EndsAt.After(now) {
		row.Lifecycle = "ended"
	} else if row.StartsAt.After(now) {
		row.Lifecycle = "scheduled"
	}
}
func (s *Service) Overview(ctx context.Context, a authz.Principal) (Overview, error) {
	if err := owner(a); err != nil {
		return Overview{}, err
	}
	now := s.now().UTC()
	value, states, err := s.repository.Overview(ctx, now)
	if err != nil {
		return value, failure(err)
	}
	value.ExamStates = states
	value.Exams, err = s.repository.UpcomingExams(ctx, now)
	if err != nil {
		return value, failure(err)
	}
	for i := range value.Exams {
		decorate(&value.Exams[i], now)
	}
	value.Quality, err = s.repository.Quality(ctx)
	if err != nil {
		return value, failure(err)
	}
	value.Agenda, err = s.Agenda(ctx, a, now, now.AddDate(0, 0, 30))
	return value, err
}
func (s *Service) Exams(ctx context.Context, a authz.Principal, page, size int, search string) ([]ExamReadiness, int64, error) {
	if err := staff(a); err != nil {
		return nil, 0, err
	}
	rows, total, err := s.repository.Exams(ctx, page, size, strings.TrimSpace(search), a.Role, a.UserID)
	for i := range rows {
		decorate(&rows[i], s.now().UTC())
	}
	return rows, total, failure(err)
}
func (s *Service) Quality(ctx context.Context, a authz.Principal) ([]QualityGroup, error) {
	if err := owner(a); err != nil {
		return nil, err
	}
	v, e := s.repository.Quality(ctx)
	return v, failure(e)
}
func (s *Service) Requests(ctx context.Context, a authz.Principal, status string, page, size int) ([]RequestRow, int64, error) {
	if err := staff(a); err != nil {
		return nil, 0, err
	}
	if status != "" && status != "pending" && status != "approved" && status != "rejected" && status != "cancelled" {
		return nil, 0, apperror.New(422, "INVALID_STATUS", "Status pengajuan tidak valid")
	}
	v, n, e := s.repository.Requests(ctx, a.Role, a.UserID, status, page, size)
	return v, n, failure(e)
}
func validateSchedule(exam ExamReadiness, start, end, now time.Time) error {
	if (exam.Status != "draft" && exam.Status != "published") || exam.AttemptCount > 0 {
		return apperror.New(409, "EXAM_SCHEDULE_LOCKED", "Jadwal tidak dapat diubah setelah ada attempt atau ujian ditutup")
	}
	if !start.After(now) || !end.After(start) || end.Sub(start) < time.Duration(exam.DurationMinutes)*time.Minute {
		return apperror.New(422, "INVALID_SCHEDULE", "Jadwal harus di masa depan dan jendelanya cukup untuk durasi ujian")
	}
	return nil
}
func (s *Service) CreateRequest(ctx context.Context, a authz.Principal, input ScheduleInput) error {
	if err := a.RequireRole("teacher", "admin"); err != nil {
		return err
	}
	if len([]rune(strings.TrimSpace(input.Reason))) < 10 {
		return apperror.New(422, "REQUEST_REASON_REQUIRED", "Alasan pengajuan minimal 10 karakter")
	}
	row, err := s.repository.Exam(ctx, input.ExamID)
	if err != nil {
		return failure(err)
	}
	if err = s.courses.RequireCourseManager(ctx, a, row.CourseID); err != nil {
		return err
	}
	value := ScheduleRequest{ID: uuid.NewString(), ExamID: input.ExamID, RequesterID: a.UserID, Reason: strings.TrimSpace(input.Reason), ProposedStartsAt: input.ProposedStartsAt.UTC(), ProposedEndsAt: input.ProposedEndsAt.UTC(), Status: "pending", CreatedAt: s.now().UTC()}
	return failure(s.repository.CreateRequest(ctx, value, func(exam ExamReadiness) error {
		return validateSchedule(exam, value.ProposedStartsAt, value.ProposedEndsAt, s.now().UTC())
	}))
}
func validateDecision(value ScheduleRequest, exam ExamReadiness, actorID, status string, now time.Time) error {
	if value.Status != "pending" {
		return apperror.New(409, "REQUEST_ALREADY_DECIDED", "Pengajuan ini sudah diproses")
	}
	if status == "cancelled" {
		if value.RequesterID != actorID {
			return apperror.New(403, "REQUEST_NOT_OWNED", "Hanya pengaju dapat membatalkan pengajuan")
		}
		return nil
	}
	if value.RequesterID == actorID {
		return apperror.New(403, "SELF_APPROVAL_DENIED", "Pengajuan sendiri harus ditinjau pimpinan lain")
	}
	if status == "approved" {
		if !value.OriginalStartsAt.Equal(exam.StartsAt) || !value.OriginalEndsAt.Equal(exam.EndsAt) {
			return apperror.New(409, "EXAM_SCHEDULE_CHANGED", "Jadwal ujian telah berubah sejak pengajuan dibuat. Ajukan ulang.")
		}
		return validateSchedule(exam, value.ProposedStartsAt, value.ProposedEndsAt, now)
	}
	return nil
}
func (s *Service) Decide(ctx context.Context, a authz.Principal, id string, input DecisionInput) error {
	if err := owner(a); err != nil {
		return err
	}
	if len([]rune(strings.TrimSpace(input.Note))) < 10 {
		return apperror.New(422, "DECISION_NOTE_REQUIRED", "Catatan keputusan minimal 10 karakter")
	}
	return failure(s.repository.Decide(ctx, id, a.UserID, input.Decision, strings.TrimSpace(input.Note), s.now().UTC(), func(v ScheduleRequest, e ExamReadiness) error {
		return validateDecision(v, e, a.UserID, input.Decision, s.now().UTC())
	}))
}
func (s *Service) Cancel(ctx context.Context, a authz.Principal, id string) error {
	if err := staff(a); err != nil {
		return err
	}
	return failure(s.repository.Decide(ctx, id, a.UserID, "cancelled", "Dibatalkan oleh pengaju", s.now().UTC(), func(v ScheduleRequest, e ExamReadiness) error {
		return validateDecision(v, e, a.UserID, "cancelled", s.now().UTC())
	}))
}
func (s *Service) Agenda(ctx context.Context, a authz.Principal, from, to time.Time) ([]Event, error) {
	if err := owner(a); err != nil {
		return nil, err
	}
	if !to.After(from) || to.Sub(from) > 366*24*time.Hour {
		return nil, apperror.New(422, "INVALID_DATE_RANGE", "Rentang agenda maksimal 366 hari")
	}
	events, err := s.repository.Events(ctx, from, to)
	if err != nil {
		return nil, failure(err)
	}
	exams, err := s.repository.ExamAgenda(ctx, from, to)
	if err != nil {
		return nil, failure(err)
	}
	events = append(events, exams...)
	sort.Slice(events, func(i, j int) bool { return events[i].StartsAt.Before(events[j].StartsAt) })
	return events, nil
}
func (s *Service) SaveEvent(ctx context.Context, a authz.Principal, id string, input EventInput) error {
	if err := owner(a); err != nil {
		return err
	}
	if len([]rune(strings.TrimSpace(input.Title))) < 3 {
		return apperror.New(422, "EVENT_TITLE_REQUIRED", "Judul agenda minimal 3 karakter")
	}
	if !input.EndsAt.After(input.StartsAt) {
		return apperror.New(422, "INVALID_EVENT_DATES", "Akhir agenda harus setelah awal agenda")
	}
	create := id == ""
	if create {
		id = uuid.NewString()
	}
	now := s.now().UTC()
	value := Event{ID: id, Title: strings.TrimSpace(input.Title), Description: strings.TrimSpace(input.Description), Category: input.Category, StartsAt: input.StartsAt.UTC(), EndsAt: input.EndsAt.UTC(), Status: "active", CreatedBy: a.UserID, CreatedAt: now, UpdatedAt: now, Version: input.Version}
	return failure(s.repository.SaveEvent(ctx, value, a.UserID, create))
}
func (s *Service) ArchiveEvent(ctx context.Context, a authz.Principal, id string, version int) error {
	if err := owner(a); err != nil {
		return err
	}
	return failure(s.repository.ArchiveEvent(ctx, id, a.UserID, version))
}
func (s *Service) Settings(ctx context.Context, a authz.Principal) (Settings, error) {
	if err := owner(a); err != nil {
		return Settings{}, err
	}
	v, e := s.repository.Settings(ctx)
	return v, failure(e)
}
func (s *Service) SaveSettings(ctx context.Context, a authz.Principal, input SettingsInput) error {
	if err := owner(a); err != nil {
		return err
	}
	if len([]rune(strings.TrimSpace(input.Reason))) < 10 {
		return apperror.New(422, "SETTINGS_REASON_REQUIRED", "Alasan perubahan minimal 10 karakter")
	}
	if len([]rune(strings.TrimSpace(input.SchoolName))) < 2 {
		return apperror.New(http.StatusUnprocessableEntity, "INVALID_SCHOOL_NAME", "Nama sekolah wajib diisi")
	}
	value := Settings{ID: 1, SchoolName: strings.TrimSpace(input.SchoolName), Motto: strings.TrimSpace(input.Motto), ContactEmail: strings.TrimSpace(input.ContactEmail), SupportMessage: strings.TrimSpace(input.SupportMessage), PassingReference: input.PassingReference, Version: input.Version, UpdatedAt: s.now().UTC()}
	return failure(s.repository.SaveSettings(ctx, value, a.UserID, strings.TrimSpace(input.Reason)))
}
func (s *Service) History(ctx context.Context, a authz.Principal) ([]ChangeRow, error) {
	if err := owner(a); err != nil {
		return nil, err
	}
	v, e := s.repository.History(ctx)
	return v, failure(e)
}
func (s *Service) Reports(ctx context.Context, a authz.Principal, year, classID string, page, size int) ([]CourseReport, int64, error) {
	if err := owner(a); err != nil {
		return nil, 0, err
	}
	v, n, e := s.repository.Reports(ctx, year, classID, page, size)
	return v, n, failure(e)
}
