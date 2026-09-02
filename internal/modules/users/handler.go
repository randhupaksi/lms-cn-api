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
