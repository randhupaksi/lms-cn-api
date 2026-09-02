package users

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"lms-cn-api/internal/authz"
	"lms-cn-api/internal/modules/audit"
	"lms-cn-api/pkg/apperror"
	"lms-cn-api/pkg/pagination"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service struct {
	repository *Repository
	audit      audit.Recorder
}

func NewService(repository *Repository, auditRecorder audit.Recorder) *Service {
	return &Service{repository: repository, audit: auditRecorder}
}

func (s *Service) Create(ctx context.Context, actor authz.Principal, request CreateRequest) (Response, error) {
	if err := actor.RequireRole(string(RoleAdmin)); err != nil {
		return Response{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(request.TemporaryPassword), bcrypt.DefaultCost)
	if err != nil {
		return Response{}, apperror.Wrap(http.StatusInternalServerError, "PASSWORD_HASH_FAILED", "Gagal menyiapkan credential", err)
	}
	user := User{
		ID: uuid.NewString(), Identifier: normalizeIdentifier(request.Identifier),
		FullName: strings.TrimSpace(request.FullName), Role: request.Role,
		Status: StatusActive, PasswordHash: string(hash), MustChangePassword: true,
	}
	if err := s.repository.Create(ctx, &user); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return Response{}, apperror.New(http.StatusConflict, "IDENTIFIER_EXISTS", "Identifier sudah digunakan")
		}
		return Response{}, apperror.Wrap(http.StatusInternalServerError, "USER_CREATE_FAILED", "Gagal membuat pengguna", err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "user.created", EntityType: "user", EntityID: user.ID})
	return ToResponse(user), nil
}

func (s *Service) List(ctx context.Context, actor authz.Principal, page pagination.Request, search, role string) ([]Response, int64, error) {
	if err := actor.RequireRole(string(RoleAdmin)); err != nil {
		return nil, 0, err
	}
	users, total, err := s.repository.List(ctx, page, search, role)
	if err != nil {
		return nil, 0, apperror.Wrap(http.StatusInternalServerError, "USERS_READ_FAILED", "Gagal memuat pengguna", err)
	}
	result := make([]Response, len(users))
	for index, user := range users {
		result[index] = ToResponse(user)
	}
	return result, total, nil
}

func (s *Service) Update(ctx context.Context, actor authz.Principal, id string, request UpdateRequest) (Response, error) {
	if err := actor.RequireRole(string(RoleAdmin)); err != nil {
		return Response{}, err
	}
	user, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return Response{}, mapUserError(err)
	}
	if request.Identifier != "" {
		user.Identifier = normalizeIdentifier(request.Identifier)
	}
	if request.FullName != "" {
		user.FullName = strings.TrimSpace(request.FullName)
	}
	if request.Status != "" {
		user.Status = request.Status
	}
	if err := s.repository.Update(ctx, &user); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return Response{}, apperror.New(http.StatusConflict, "IDENTIFIER_EXISTS", "Identifier sudah digunakan")
		}
		return Response{}, apperror.Wrap(http.StatusInternalServerError, "USER_UPDATE_FAILED", "Gagal memperbarui pengguna", err)
	}
	if user.Status == StatusInactive {
		_ = s.repository.RevokeSessions(ctx, user.ID)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "user.updated", EntityType: "user", EntityID: user.ID, Metadata: map[string]any{"status": user.Status}})
	return ToResponse(user), nil
}

func (s *Service) ResetCredential(ctx context.Context, actor authz.Principal, id string, request ResetCredentialRequest) error {
	if err := actor.RequireRole(string(RoleAdmin)); err != nil {
		return err
	}
	user, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return mapUserError(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(request.TemporaryPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperror.Wrap(http.StatusInternalServerError, "PASSWORD_HASH_FAILED", "Gagal menyiapkan credential", err)
	}
	user.PasswordHash = string(hash)
	user.MustChangePassword = true
	if err := s.repository.Update(ctx, &user); err != nil {
		return apperror.Wrap(http.StatusInternalServerError, "CREDENTIAL_RESET_FAILED", "Gagal mereset credential", err)
	}
	if err := s.repository.RevokeSessions(ctx, user.ID); err != nil {
		return apperror.Wrap(http.StatusInternalServerError, "SESSION_REVOKE_FAILED", "Credential berubah tetapi sesi gagal diakhiri", err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "user.credential_reset", EntityType: "user", EntityID: user.ID})
	return nil
}

func mapUserError(err error) error {
	if errors.Is(err, ErrNotFound) {
		return apperror.New(http.StatusNotFound, "USER_NOT_FOUND", "Pengguna tidak ditemukan")
	}
	return apperror.Wrap(http.StatusInternalServerError, "USER_READ_FAILED", "Gagal memuat pengguna", err)
}
