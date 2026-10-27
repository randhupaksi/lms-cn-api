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
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Create(ctx context.Context, actor authz.Principal, request CreateRequest) (Response, error) {
	if err := actor.RequireRole(string(RoleSuperadmin), string(RoleAdmin)); err != nil {
		return Response{}, err
	}
	if request.Role == RoleAdmin && actor.Role != string(RoleSuperadmin) {
		return Response{}, apperror.New(http.StatusForbidden, "ROLE_ASSIGNMENT_DENIED", "Hanya superadmin dapat membuat administrator")
	}
	if request.Role != RoleAdmin && request.Role != RoleTeacher && request.Role != RoleStudent {
		return Response{}, apperror.New(422, "INVALID_CREATE_ROLE", "Role akun baru tidak valid")
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
	if err := s.repository.CreateAudited(ctx, &user, audit.Record{ActorID: actor.UserID, Action: "user.created", EntityType: "user", EntityID: user.ID, Metadata: map[string]any{"role": user.Role}}); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return Response{}, apperror.New(http.StatusConflict, "IDENTIFIER_EXISTS", "Identifier sudah digunakan")
		}
		return Response{}, apperror.Wrap(http.StatusInternalServerError, "USER_CREATE_FAILED", "Gagal membuat pengguna", err)
	}
	return ToResponse(user), nil
}

func (s *Service) List(ctx context.Context, actor authz.Principal, page pagination.Request, search, role string) ([]Response, int64, error) {
	if err := actor.RequireRole(string(RoleSuperadmin), string(RoleAdmin)); err != nil {
		return nil, 0, err
	}
	if actor.Role != string(RoleSuperadmin) && role != "" && role != "teacher" && role != "student" {
		return nil, 0, apperror.New(http.StatusForbidden, "ROLE_ACCESS_DENIED", "Daftar role ini hanya tersedia untuk superadmin")
	}
	users, total, err := s.repository.List(ctx, page, search, role, actor.Role == string(RoleSuperadmin))
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
	if !actor.IsSchoolAdmin() {
		return Response{}, actor.RequireRole(string(RoleAdmin), string(RoleSuperadmin))
	}
	var result *User
	err := s.repository.Mutate(ctx, id, request.Status == StatusInactive, func(repo *Repository, user *User) error {
		if err := canManageTarget(actor, *user); err != nil {
			return err
		}
		if user.ID == actor.UserID && request.Status == StatusInactive {
			return apperror.New(http.StatusConflict, "SELF_DEACTIVATION_DENIED", "Akun sendiri tidak dapat dinonaktifkan")
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
		if user.Role == RoleSuperadmin && user.Status == StatusInactive {
			if err := requireOtherSuperadmin(ctx, repo, user.ID); err != nil {
				return err
			}
		}
		result = user
		return nil
	}, audit.Record{ActorID: actor.UserID, Action: "user.updated", EntityType: "user", EntityID: id, Metadata: map[string]any{"status": request.Status}})
	if err != nil {
		return Response{}, mutationError(err)
	}
	return ToResponse(*result), nil
}

func (s *Service) ResetCredential(ctx context.Context, actor authz.Principal, id string, request ResetCredentialRequest) error {
	if err := actor.RequireRole(string(RoleSuperadmin), string(RoleAdmin)); err != nil {
		return err
	}
	user, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return mapUserError(err)
	}
	if err := canManageTarget(actor, user); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(request.TemporaryPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperror.Wrap(http.StatusInternalServerError, "PASSWORD_HASH_FAILED", "Gagal menyiapkan credential", err)
	}
	err = s.repository.Mutate(ctx, id, true, func(_ *Repository, target *User) error {
		if err := canManageTarget(actor, *target); err != nil {
			return err
		}
		target.PasswordHash = string(hash)
		target.MustChangePassword = true
		return nil
	}, audit.Record{ActorID: actor.UserID, Action: "user.credential_reset", EntityType: "user", EntityID: id})
	if err != nil {
		return mutationError(err)
	}
	return nil
}

func canManageTarget(actor authz.Principal, target User) error {
	if actor.Role == string(RoleSuperadmin) || (actor.Role == string(RoleAdmin) && (target.Role == RoleTeacher || target.Role == RoleStudent)) {
		return nil
	}
	return apperror.New(http.StatusForbidden, "PRIVILEGED_ACCOUNT_PROTECTED", "Akun administrator hanya dapat dikelola superadmin")
}

func requireOtherSuperadmin(ctx context.Context, repo *Repository, id string) error {
	count, err := repo.OtherActiveSuperadmins(ctx, id)
	if err != nil {
		return err
	}
	if count == 0 {
		return apperror.New(http.StatusConflict, "LAST_SUPERADMIN_PROTECTED", "Minimal satu superadmin aktif harus tetap tersedia")
	}
	return nil
}

func mutationError(err error) error {
	var app *apperror.Error
	if errors.As(err, &app) {
		return err
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return apperror.New(http.StatusConflict, "IDENTIFIER_EXISTS", "Identifier sudah digunakan")
	}
	return mapUserError(err)
}

func (s *Service) ChangeRole(ctx context.Context, actor authz.Principal, id string, request ChangeRoleRequest) error {
	if err := actor.RequireRole(string(RoleSuperadmin)); err != nil {
		return err
	}
	if actor.UserID == id {
		return apperror.New(http.StatusConflict, "SELF_ROLE_CHANGE_DENIED", "Role akun sendiri tidak dapat diubah")
	}
	if len([]rune(strings.TrimSpace(request.Reason))) < 10 {
		return apperror.New(422, "ROLE_REASON_REQUIRED", "Alasan perubahan minimal 10 karakter")
	}
	current, err := s.repository.FindByID(ctx, actor.UserID)
	if err != nil || current.Role != RoleSuperadmin || !current.IsActive() || bcrypt.CompareHashAndPassword([]byte(current.PasswordHash), []byte(request.CurrentPassword)) != nil {
		return apperror.New(http.StatusForbidden, "REAUTHENTICATION_FAILED", "Konfirmasi kata sandi tidak sesuai")
	}
	meta := map[string]any{"to": request.Role, "reason": strings.TrimSpace(request.Reason)}
	err = s.repository.Mutate(ctx, id, true, func(repo *Repository, user *User) error {
		meta["from"] = user.Role
		if user.Role == request.Role {
			return apperror.New(http.StatusConflict, "ROLE_UNCHANGED", "Role pengguna belum berubah")
		}
		if user.Role == RoleSuperadmin {
			if err := requireOtherSuperadmin(ctx, repo, id); err != nil {
				return err
			}
		}
		linked, err := repo.HasAcademicRecords(ctx, *user)
		if err != nil {
			return err
		}
		if linked {
			return apperror.New(http.StatusConflict, "ACADEMIC_ROLE_LOCKED", "Akun masih memiliki relasi atau riwayat akademik. Gunakan akun terpisah untuk role baru.")
		}
		user.Role = request.Role
		return nil
	}, audit.Record{ActorID: actor.UserID, Action: "user.role_changed", EntityType: "user", EntityID: id, Metadata: meta})
	if err != nil {
		return mutationError(err)
	}
	return nil
}

func (s *Service) Sessions(ctx context.Context, actor authz.Principal, id string) ([]SessionResponse, error) {
	if err := actor.RequireRole(string(RoleSuperadmin)); err != nil {
		return nil, err
	}
	if _, err := s.repository.FindByID(ctx, id); err != nil {
		return nil, mapUserError(err)
	}
	rows, err := s.repository.Sessions(ctx, id)
	if err != nil {
		return nil, mapUserError(err)
	}
	return rows, nil
}

func (s *Service) RevokeSessions(ctx context.Context, actor authz.Principal, id string) error {
	if err := actor.RequireRole(string(RoleSuperadmin)); err != nil {
		return err
	}
	err := s.repository.Mutate(ctx, id, true, func(_ *Repository, _ *User) error { return nil }, audit.Record{ActorID: actor.UserID, Action: "user.sessions_revoked", EntityType: "user", EntityID: id})
	if err != nil {
		return mutationError(err)
	}
	return nil
}

func mapUserError(err error) error {
	if errors.Is(err, ErrNotFound) {
		return apperror.New(http.StatusNotFound, "USER_NOT_FOUND", "Pengguna tidak ditemukan")
	}
	return apperror.Wrap(http.StatusInternalServerError, "USER_READ_FAILED", "Gagal memuat pengguna", err)
}
