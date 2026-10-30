package governance

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"lms-cn-api/internal/authz"
	"lms-cn-api/pkg/apperror"
)

func isSchoolLeader(actor authz.Principal) bool {
	return actor.Role == "superadmin" || actor.Role == "principal"
}

func canAccessLeadershipAction(actor authz.Principal, value LeadershipAction) bool {
	return isSchoolLeader(actor) || ((actor.Role == "admin" || actor.Role == "teacher") && value.AssigneeID != nil && *value.AssigneeID == actor.UserID)
}

func validActionCategory(value string) bool {
	switch value {
	case "academic", "exam", "data_quality", "general":
		return true
	default:
		return false
	}
}

func validActionPriority(value string) bool {
	switch value {
	case "low", "normal", "high":
		return true
	default:
		return false
	}
}

func validActionStatus(value string) bool {
	switch value {
	case "open", "in_progress", "blocked", "completed", "cancelled":
		return true
	default:
		return false
	}
}

func (s *Service) ActionAssignees(ctx context.Context, actor authz.Principal) ([]LeadershipActionAssignee, error) {
	if !isSchoolLeader(actor) {
		return nil, apperror.New(http.StatusForbidden, "LEADERSHIP_ACTION_DENIED", "Daftar penanggung jawab hanya tersedia untuk pimpinan sekolah")
	}
	rows, err := s.repository.ListActionAssignees(ctx)
	if err != nil {
		return nil, failure(err)
	}
	return rows, nil
}

func (s *Service) ListLeadershipActions(ctx context.Context, actor authz.Principal, status, category string, page, size int) ([]LeadershipActionResponse, int64, error) {
	if err := actor.RequireRole("superadmin", "principal", "admin", "teacher"); err != nil {
		return nil, 0, err
	}
	if status != "" && status != "open" && status != "in_progress" && status != "blocked" && status != "completed" && status != "cancelled" {
		return nil, 0, apperror.New(http.StatusBadRequest, "ACTION_STATUS_INVALID", "Status tindak lanjut tidak valid")
	}
	if category != "" && !validActionCategory(category) {
		return nil, 0, apperror.New(http.StatusBadRequest, "ACTION_CATEGORY_INVALID", "Kategori tindak lanjut tidak valid")
	}
	rows, total, err := s.repository.ListLeadershipActions(ctx, actor.Role, actor.UserID, status, category, page, size)
	if err != nil {
		return nil, 0, failure(err)
	}
	now := s.now().UTC()
	result := make([]LeadershipActionResponse, len(rows))
	for index, row := range rows {
		result[index] = toLeadershipActionResponse(row, now)
	}
	return result, total, nil
}

func toLeadershipActionResponse(row leadershipActionListRow, now time.Time) LeadershipActionResponse {
	overdue := row.DueAt != nil && row.DueAt.Before(now) && row.Status != "completed" && row.Status != "cancelled"
	return LeadershipActionResponse{
		ID: row.ID, Title: row.Title, Description: row.Description, Category: row.Category,
		Priority: row.Priority, Status: row.Status, AssigneeID: row.AssigneeID, AssigneeName: row.AssigneeName,
		CreatedBy: row.CreatedBy, CreatedByName: row.CreatedByName, DueAt: row.DueAt,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, CompletedAt: row.CompletedAt,
		LatestNote: row.LatestNote, Version: row.Version, Overdue: overdue,
	}
}

func (s *Service) CreateLeadershipAction(ctx context.Context, actor authz.Principal, input CreateLeadershipActionInput) error {
	if !isSchoolLeader(actor) {
		return apperror.New(http.StatusForbidden, "LEADERSHIP_ACTION_DENIED", "Hanya pimpinan sekolah yang dapat membuat tindak lanjut")
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.Category = strings.TrimSpace(input.Category)
	input.Priority = strings.TrimSpace(input.Priority)
	input.AssigneeID = strings.TrimSpace(input.AssigneeID)
	if len([]rune(input.Title)) < 3 || len([]rune(input.Title)) > 180 || len([]rune(input.Description)) > 2000 {
		return apperror.New(http.StatusUnprocessableEntity, "ACTION_INPUT_INVALID", "Judul atau uraian tindak lanjut tidak valid")
	}
	if !validActionCategory(input.Category) {
		return apperror.New(http.StatusUnprocessableEntity, "ACTION_CATEGORY_INVALID", "Kategori tindak lanjut tidak valid")
	}
	if !validActionPriority(input.Priority) {
		return apperror.New(http.StatusUnprocessableEntity, "ACTION_PRIORITY_INVALID", "Prioritas tindak lanjut tidak valid")
	}
	var assigneeID *string
	if input.AssigneeID != "" {
		assigneeID = &input.AssigneeID
	}
	now := s.now().UTC()
	value := LeadershipAction{
		ID: uuid.NewString(), Title: input.Title, Description: input.Description, Category: input.Category,
		Priority: input.Priority, Status: "open", AssigneeID: assigneeID, DueAt: input.DueAt,
		CreatedBy: actor.UserID, CreatedAt: now, UpdatedAt: now, Version: 1,
	}
	err := s.repository.CreateLeadershipAction(ctx, value, actor.UserID, "Tindak lanjut dibuat.")
	if errors.Is(err, ErrLeadershipAssignee) {
		return apperror.New(http.StatusUnprocessableEntity, "ACTION_ASSIGNEE_INVALID", "Penanggung jawab harus akun guru atau administrator yang aktif")
	}
	return failure(err)
}

func validLeadershipActionStatus(role, current, next string) error {
	if current == next {
		return apperror.New(http.StatusConflict, "ACTION_STATUS_UNCHANGED", "Status tindak lanjut tidak berubah")
	}
	if role == "principal" || role == "superadmin" {
		return nil
	}
	if role != "admin" && role != "teacher" {
		return apperror.New(http.StatusForbidden, "LEADERSHIP_ACTION_DENIED", "Akun ini tidak dapat memperbarui tindak lanjut")
	}
	if current == "completed" || current == "cancelled" || next == "open" || next == "cancelled" {
		return apperror.New(http.StatusForbidden, "ACTION_TRANSITION_DENIED", "Staf hanya dapat memulai, menahan, atau menyelesaikan tindak lanjut yang ditugaskan")
	}
	return nil
}

func (s *Service) UpdateLeadershipActionStatus(ctx context.Context, actor authz.Principal, id string, input UpdateLeadershipActionStatusInput) error {
	if err := actor.RequireRole("superadmin", "principal", "admin", "teacher"); err != nil {
		return err
	}
	if !validActionStatus(input.Status) {
		return apperror.New(http.StatusBadRequest, "ACTION_STATUS_INVALID", "Status tindak lanjut tidak valid")
	}
	if len([]rune(strings.TrimSpace(input.Note))) < 5 || len([]rune(input.Note)) > 1000 {
		return apperror.New(http.StatusUnprocessableEntity, "ACTION_NOTE_REQUIRED", "Catatan tindak lanjut minimal 5 karakter")
	}
	value, err := s.repository.FindLeadershipAction(ctx, id)
	if errors.Is(err, ErrLeadershipActionNotFound) {
		return apperror.New(http.StatusNotFound, "ACTION_NOT_FOUND", "Tindak lanjut tidak ditemukan")
	}
	if err != nil {
		return failure(err)
	}
	if !canAccessLeadershipAction(actor, value) {
		return apperror.New(http.StatusForbidden, "LEADERSHIP_ACTION_DENIED", "Tindak lanjut ini bukan tugas akun Anda")
	}
	if err := validLeadershipActionStatus(actor.Role, value.Status, input.Status); err != nil {
		return err
	}
	err = s.repository.UpdateLeadershipActionStatus(ctx, id, actor.UserID, input.Status, strings.TrimSpace(input.Note), input.Version, s.now().UTC())
	if errors.Is(err, ErrLeadershipActionNotFound) {
		return apperror.New(http.StatusNotFound, "ACTION_NOT_FOUND", "Tindak lanjut tidak ditemukan")
	}
	if errors.Is(err, ErrLeadershipVersion) {
		return apperror.New(http.StatusConflict, "ACTION_STALE", "Tindak lanjut sudah berubah. Muat ulang sebelum menyimpan lagi")
	}
	return failure(err)
}

func (s *Service) LeadershipActionUpdates(ctx context.Context, actor authz.Principal, id string) ([]LeadershipActionUpdateResponse, error) {
	if err := actor.RequireRole("superadmin", "principal", "admin", "teacher"); err != nil {
		return nil, err
	}
	value, err := s.repository.FindLeadershipAction(ctx, id)
	if errors.Is(err, ErrLeadershipActionNotFound) {
		return nil, apperror.New(http.StatusNotFound, "ACTION_NOT_FOUND", "Tindak lanjut tidak ditemukan")
	}
	if err != nil {
		return nil, failure(err)
	}
	if !canAccessLeadershipAction(actor, value) {
		return nil, apperror.New(http.StatusForbidden, "LEADERSHIP_ACTION_DENIED", "Riwayat tindak lanjut ini bukan untuk akun Anda")
	}
	rows, err := s.repository.LeadershipActionUpdates(ctx, id)
	if err != nil {
		return nil, failure(err)
	}
	result := make([]LeadershipActionUpdateResponse, len(rows))
	for index, row := range rows {
		result[index] = LeadershipActionUpdateResponse{
			ID: row.ID, ActorID: row.ActorID, ActorName: row.ActorName, FromStatus: row.FromStatus,
			ToStatus: row.ToStatus, Note: row.Note, CreatedAt: row.CreatedAt,
		}
	}
	return result, nil
}

func (s *Service) LeadershipActionCounts(ctx context.Context, actor authz.Principal) (open, overdue int64, err error) {
	if err = actor.RequireRole("superadmin", "principal", "admin", "teacher"); err != nil {
		return 0, 0, err
	}
	open, err = s.repository.LeadershipActionCount(ctx, actor.Role, actor.UserID, "open", "in_progress", "blocked")
	if err != nil {
		return 0, 0, failure(err)
	}
	overdue, err = s.repository.OverdueLeadershipActionCount(ctx, actor.Role, actor.UserID, s.now().UTC())
	if err != nil {
		return 0, 0, failure(err)
	}
	return open, overdue, nil
}
