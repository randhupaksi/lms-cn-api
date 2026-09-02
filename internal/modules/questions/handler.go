package questions

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
	group.PUT("/:questionID", h.update)
	group.DELETE("/:questionID", h.archive)
}

func (h *Handler) list(c *gin.Context) {
	courseID := c.Query("course_id")
	if courseID == "" {
		response.Error(c, http.StatusBadRequest, "COURSE_ID_REQUIRED", "Course wajib dipilih")
		return
	}
	principal, _ := middleware.Principal(c)
	page := pagination.FromContext(c)
	data, total, err := h.service.List(c.Request.Context(), principal, courseID, page)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.SuccessWithMeta(c, http.StatusOK, "Bank soal berhasil dimuat", data, page.Meta(total))
}

func (h *Handler) create(c *gin.Context) {
	var body WriteRequest
	if err := request.BindJSON(c, &body); err != nil {
		response.FromError(c, err)
		return
	}
	principal, _ := middleware.Principal(c)
	data, err := h.service.Create(c.Request.Context(), principal, body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Soal berhasil dibuat", data)
}

func (h *Handler) update(c *gin.Context) {
	questionID, err := request.RequireID(c, "questionID")
	if err != nil {
		response.FromError(c, err)
		return
	}
	var body WriteRequest
	if err := request.BindJSON(c, &body); err != nil {
		response.FromError(c, err)
		return
	}
	principal, _ := middleware.Principal(c)
	data, err := h.service.Update(c.Request.Context(), principal, questionID, body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Soal berhasil diperbarui", data)
}

func (h *Handler) archive(c *gin.Context) {
	questionID, err := request.RequireID(c, "questionID")
	if err != nil {
		response.FromError(c, err)
		return
	}
	principal, _ := middleware.Principal(c)
	if err := h.service.Archive(c.Request.Context(), principal, questionID); err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Soal berhasil diarsipkan", nil)
}
