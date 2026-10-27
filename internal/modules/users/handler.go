package users

import (
	"net/http"

	"lms-cn-api/internal/middleware"
	"lms-cn-api/pkg/pagination"
	"lms-cn-api/pkg/request"
	"lms-cn-api/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("", h.list)
	group.POST("", h.create)
	group.PATCH("/:userID", h.update)
	group.POST("/:userID/reset-credential", h.resetCredential)
	group.PATCH("/:userID/role", h.changeRole)
	group.GET("/:userID/sessions", h.sessions)
	group.POST("/:userID/revoke-sessions", h.revokeSessions)
}

func (h *Handler) changeRole(c *gin.Context) {
	var body ChangeRoleRequest
	if err := request.BindJSON(c, &body); err != nil {
		response.FromError(c, err)
		return
	}
	actor, _ := middleware.Principal(c)
	if err := h.service.ChangeRole(c.Request.Context(), actor, c.Param("userID"), body); err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Role diperbarui dan sesi lama diakhiri", nil)
}
func (h *Handler) sessions(c *gin.Context) {
	actor, _ := middleware.Principal(c)
	rows, err := h.service.Sessions(c.Request.Context(), actor, c.Param("userID"))
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Sesi aktif berhasil dimuat", rows)
}
func (h *Handler) revokeSessions(c *gin.Context) {
	actor, _ := middleware.Principal(c)
	if err := h.service.RevokeSessions(c.Request.Context(), actor, c.Param("userID")); err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Seluruh sesi pengguna telah diakhiri", nil)
}

func (h *Handler) list(c *gin.Context) {
	principal, _ := middleware.Principal(c)
	page := pagination.FromContext(c)
	result, total, err := h.service.List(c.Request.Context(), principal, page, c.Query("search"), c.Query("role"))
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.SuccessWithMeta(c, http.StatusOK, "Pengguna berhasil dimuat", result, page.Meta(total))
}

func (h *Handler) create(c *gin.Context) {
	var body CreateRequest
	if err := request.BindJSON(c, &body); err != nil {
		response.FromError(c, err)
		return
	}
	principal, _ := middleware.Principal(c)
	result, err := h.service.Create(c.Request.Context(), principal, body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Pengguna berhasil dibuat", result)
}

func (h *Handler) update(c *gin.Context) {
	userID, err := request.RequireID(c, "userID")
	if err != nil {
		response.FromError(c, err)
		return
	}
	var body UpdateRequest
	if err := request.BindJSON(c, &body); err != nil {
		response.FromError(c, err)
		return
	}
	principal, _ := middleware.Principal(c)
	result, err := h.service.Update(c.Request.Context(), principal, userID, body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Pengguna berhasil diperbarui", result)
}

func (h *Handler) resetCredential(c *gin.Context) {
	userID, err := request.RequireID(c, "userID")
	if err != nil {
		response.FromError(c, err)
		return
	}
	var body ResetCredentialRequest
	if err := request.BindJSON(c, &body); err != nil {
		response.FromError(c, err)
		return
	}
	principal, _ := middleware.Principal(c)
	if err := h.service.ResetCredential(c.Request.Context(), principal, userID, body); err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Credential berhasil direset", nil)
}
