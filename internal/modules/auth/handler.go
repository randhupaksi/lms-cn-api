package auth

import (
	"net/http"
	"strings"
	"time"

	"lms-cn-api/internal/config"
	"lms-cn-api/internal/middleware"
	"lms-cn-api/pkg/request"
	"lms-cn-api/pkg/response"

	"github.com/gin-gonic/gin"
)

const refreshCookieName = "lms_cn_refresh"

type Handler struct {
	service       *Service
	cookieDomain  string
	cookiePath    string
	cookieSecure  bool
	refreshMaxAge int
}

func NewHandler(service *Service, cfg *config.Config) *Handler {
	return &Handler{
		service: service, cookieDomain: cfg.CookieDomain, cookieSecure: cfg.CookieSecure,
		cookiePath: strings.TrimRight(cfg.APIPrefix, "/") + "/auth", refreshMaxAge: int(cfg.RefreshTokenTTL / time.Second),
	}
}

func (h *Handler) RegisterPublicRoutes(group *gin.RouterGroup, loginMiddleware gin.HandlerFunc) {
	group.POST("/login", loginMiddleware, h.login)
	group.POST("/refresh", h.refresh)
}

func (h *Handler) RegisterProtectedRoutes(group *gin.RouterGroup) {
	group.GET("/me", h.me)
	group.POST("/logout", h.logout)
	group.POST("/change-password", h.changePassword)
}

func (h *Handler) login(c *gin.Context) {
	var body LoginRequest
	if err := request.BindJSON(c, &body); err != nil {
		response.FromError(c, err)
		return
	}
	session, refreshToken, err := h.service.Login(c.Request.Context(), body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	h.setRefreshCookie(c, refreshToken)
	response.Success(c, http.StatusOK, "Login berhasil", session)
}

func (h *Handler) refresh(c *gin.Context) {
	rawRefresh, _ := c.Cookie(refreshCookieName)
	session, nextRefresh, err := h.service.Refresh(c.Request.Context(), rawRefresh)
	if err != nil {
		h.clearRefreshCookie(c)
		response.FromError(c, err)
		return
	}
	h.setRefreshCookie(c, nextRefresh)
	response.Success(c, http.StatusOK, "Sesi diperbarui", session)
}

func (h *Handler) me(c *gin.Context) {
	principal, _ := middleware.Principal(c)
	user, err := h.service.Me(c.Request.Context(), principal)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Profil sesi dimuat", user)
}

func (h *Handler) logout(c *gin.Context) {
	principal, _ := middleware.Principal(c)
	if err := h.service.Logout(c.Request.Context(), principal); err != nil {
		response.FromError(c, err)
		return
	}
	h.clearRefreshCookie(c)
	response.Success(c, http.StatusOK, "Logout berhasil", nil)
}

func (h *Handler) changePassword(c *gin.Context) {
	var body ChangePasswordRequest
	if err := request.BindJSON(c, &body); err != nil {
		response.FromError(c, err)
		return
	}
	principal, _ := middleware.Principal(c)
	if err := h.service.ChangePassword(c.Request.Context(), principal, body); err != nil {
		response.FromError(c, err)
		return
	}
	h.clearRefreshCookie(c)
	response.Success(c, http.StatusOK, "Password berhasil diperbarui. Silakan login kembali", nil)
}

func (h *Handler) setRefreshCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(refreshCookieName, token, h.refreshMaxAge, h.cookiePath, h.cookieDomain, h.cookieSecure, true)
}

func (h *Handler) clearRefreshCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(refreshCookieName, "", -1, h.cookiePath, h.cookieDomain, h.cookieSecure, true)
}
