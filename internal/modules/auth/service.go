package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"lms-cn-api/internal/authz"
	"lms-cn-api/internal/modules/audit"
	"lms-cn-api/internal/modules/users"
	"lms-cn-api/pkg/apperror"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repository *Repository
	users      *users.Repository
	tokens     *TokenManager
	audit      audit.Recorder
	accessTTL  time.Duration
	refreshTTL time.Duration
	now        func() time.Time
}

func NewService(repository *Repository, usersRepository *users.Repository, tokens *TokenManager, auditRecorder audit.Recorder, accessTTL, refreshTTL time.Duration) *Service {
	return &Service{
		repository: repository, users: usersRepository, tokens: tokens, audit: auditRecorder,
		accessTTL: accessTTL, refreshTTL: refreshTTL, now: time.Now,
	}
}

func (s *Service) Login(ctx context.Context, request LoginRequest) (SessionResponse, string, error) {
	user, err := s.users.FindByIdentifier(ctx, strings.TrimSpace(request.Identifier))
	if err != nil || !user.IsActive() || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password)) != nil {
		return SessionResponse{}, "", apperror.New(http.StatusUnauthorized, "INVALID_CREDENTIALS", "Identifier atau password tidak valid")
	}

	now := s.now().UTC()
	rawRefresh, refreshHash, err := generateRefreshToken()
	if err != nil {
		return SessionResponse{}, "", apperror.Wrap(http.StatusInternalServerError, "SESSION_CREATE_FAILED", "Gagal membuat sesi", err)
	}
	session := Session{
		ID: uuid.NewString(), UserID: user.ID, RefreshTokenHash: refreshHash,
		ExpiresAt: now.Add(s.refreshTTL), LastUsedAt: now,
	}
	if err := s.repository.Create(ctx, &session); err != nil {
		return SessionResponse{}, "", apperror.Wrap(http.StatusInternalServerError, "SESSION_CREATE_FAILED", "Gagal membuat sesi", err)
	}
	response, err := s.sessionResponse(user, session.ID, now)
	if err != nil {
		return SessionResponse{}, "", err
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: user.ID, Action: "auth.login", EntityType: "auth_session", EntityID: session.ID})
	return response, rawRefresh, nil
}

func (s *Service) Refresh(ctx context.Context, rawRefresh string) (SessionResponse, string, error) {
	if strings.TrimSpace(rawRefresh) == "" {
		return SessionResponse{}, "", apperror.New(http.StatusUnauthorized, "INVALID_SESSION", "Sesi tidak valid atau telah berakhir")
	}
	nextRaw, nextHash, err := generateRefreshToken()
	if err != nil {
		return SessionResponse{}, "", apperror.Wrap(http.StatusInternalServerError, "SESSION_REFRESH_FAILED", "Gagal memperbarui sesi", err)
	}
	now := s.now().UTC()
	session, err := s.repository.Rotate(ctx, hashToken(rawRefresh), nextHash, now)
	if err != nil {
		return SessionResponse{}, "", apperror.New(http.StatusUnauthorized, "INVALID_SESSION", "Sesi tidak valid atau telah berakhir")
	}
	response, err := s.sessionResponse(session.User, session.ID, now)
	return response, nextRaw, err
}

func (s *Service) Logout(ctx context.Context, principal authz.Principal) error {
	if err := s.repository.Revoke(ctx, principal.SessionID, s.now().UTC()); err != nil {
		return apperror.Wrap(http.StatusInternalServerError, "LOGOUT_FAILED", "Gagal mengakhiri sesi", err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: principal.UserID, Action: "auth.logout", EntityType: "auth_session", EntityID: principal.SessionID})
	return nil
}

func (s *Service) Me(ctx context.Context, principal authz.Principal) (users.Response, error) {
	user, err := s.users.FindByID(ctx, principal.UserID)
	if err != nil || !user.IsActive() {
		return users.Response{}, apperror.New(http.StatusUnauthorized, "INVALID_SESSION", "Sesi tidak valid atau telah berakhir")
	}
	return users.ToResponse(user), nil
}

func (s *Service) ChangePassword(ctx context.Context, principal authz.Principal, request ChangePasswordRequest) error {
	user, err := s.users.FindByID(ctx, principal.UserID)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.CurrentPassword)) != nil {
		return apperror.New(http.StatusBadRequest, "CURRENT_PASSWORD_INVALID", "Password saat ini tidak valid")
	}
	if request.CurrentPassword == request.NewPassword {
		return apperror.New(http.StatusUnprocessableEntity, "PASSWORD_UNCHANGED", "Password baru harus berbeda")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(request.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperror.Wrap(http.StatusInternalServerError, "PASSWORD_HASH_FAILED", "Gagal memperbarui password", err)
	}
	user.PasswordHash = string(hash)
	user.MustChangePassword = false
	if err := s.users.Update(ctx, &user); err != nil {
		return apperror.Wrap(http.StatusInternalServerError, "PASSWORD_UPDATE_FAILED", "Gagal memperbarui password", err)
	}
	if err := s.repository.RevokeAllForUser(ctx, user.ID, s.now().UTC()); err != nil {
		return apperror.Wrap(http.StatusInternalServerError, "SESSION_REVOKE_FAILED", "Password berubah tetapi sesi gagal diakhiri", err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: user.ID, Action: "auth.password_changed", EntityType: "user", EntityID: user.ID})
	return nil
}

func (s *Service) VerifyAccessToken(rawToken string) (authz.Principal, error) {
	return s.tokens.VerifyAccessToken(rawToken)
}

func (s *Service) ValidateSession(ctx context.Context, principal authz.Principal) error {
	session, err := s.repository.FindByID(ctx, principal.SessionID)
	if err != nil || session.UserID != principal.UserID || !session.IsValid(s.now().UTC()) {
		return errors.New("session is not active")
	}
	return nil
}

func (s *Service) sessionResponse(user users.User, sessionID string, now time.Time) (SessionResponse, error) {
	accessToken, err := s.tokens.CreateAccessToken(authz.Principal{UserID: user.ID, SessionID: sessionID, Role: string(user.Role), MustChangePassword: user.MustChangePassword}, now)
	if err != nil {
		return SessionResponse{}, apperror.Wrap(http.StatusInternalServerError, "ACCESS_TOKEN_FAILED", "Gagal membuat access token", err)
	}
	return SessionResponse{AccessToken: accessToken, ExpiresIn: int64(s.accessTTL.Seconds()), User: users.ToResponse(user)}, nil
}
