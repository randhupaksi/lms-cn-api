package middleware

import (
	"context"
	"net/http"
	"strings"

	"lms-cn-api/internal/authz"
	"lms-cn-api/pkg/response"

	"github.com/gin-gonic/gin"
)

const principalKey = "auth_principal"

type AccessTokenVerifier interface {
	VerifyAccessToken(rawToken string) (authz.Principal, error)
	ValidateSession(ctx context.Context, principal authz.Principal) error
}

func Authenticate(verifier AccessTokenVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if !strings.HasPrefix(rawHeader, "Bearer ") {
			response.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi login diperlukan")
			c.Abort()
			return
		}

		principal, err := verifier.VerifyAccessToken(strings.TrimSpace(strings.TrimPrefix(rawHeader, "Bearer ")))
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "INVALID_SESSION", "Sesi tidak valid atau telah berakhir")
			c.Abort()
			return
		}
		if err := verifier.ValidateSession(c.Request.Context(), principal); err != nil {
			response.Error(c, http.StatusUnauthorized, "INVALID_SESSION", "Sesi tidak valid atau telah berakhir")
			c.Abort()
			return
		}
		if principal.MustChangePassword && !passwordChangeAllowed(c.Request.URL.Path) {
			response.Error(c, http.StatusForbidden, "PASSWORD_CHANGE_REQUIRED", "Kata sandi wajib diperbarui sebelum melanjutkan")
			c.Abort()
			return
		}

		c.Set(principalKey, principal)
		c.Next()
	}
}

func passwordChangeAllowed(path string) bool {
	return strings.HasSuffix(path, "/auth/change-password") || strings.HasSuffix(path, "/auth/logout") || strings.HasSuffix(path, "/auth/me")
}

func Principal(c *gin.Context) (authz.Principal, bool) {
	value, exists := c.Get(principalKey)
	if !exists {
		return authz.Principal{}, false
	}
	principal, ok := value.(authz.Principal)
	return principal, ok
}

func RequireRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := Principal(c)
		if !ok {
			response.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Sesi login diperlukan")
			c.Abort()
			return
		}
		if err := principal.RequireRole(roles...); err != nil {
			response.FromError(c, err)
			c.Abort()
			return
		}
		c.Next()
	}
}
