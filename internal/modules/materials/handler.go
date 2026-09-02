package materials

import (
	"context"
	"net/http"

	"lms-cn-api/internal/authz"
	"lms-cn-api/internal/middleware"
	"lms-cn-api/pkg/request"
	"lms-cn-api/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("", h.list)
	group.POST("", h.create)
	group.PUT("/:materialID", h.update)
	group.POST("/:materialID/publish", h.publish)
	group.POST("/:materialID/complete", h.complete)
}

func (h *Handler) list(c *gin.Context) {
	courseID := c.Query("course_id")
	if courseID == "" {
		response.Error(c, http.StatusBadRequest, "COURSE_ID_REQUIRED", "Course wajib dipilih")
		return
	}
	principal, _ := middleware.Principal(c)
	data, err := h.service.List(c.Request.Context(), principal, courseID)
	respond(c, http.StatusOK, "Materi berhasil dimuat", data, err)
}

func (h *Handler) create(c *gin.Context) { h.write(c, true) }
func (h *Handler) update(c *gin.Context) { h.write(c, false) }

func (h *Handler) write(c *gin.Context, create bool) {
	var body WriteRequest
	if err := request.BindJSON(c, &body); err != nil {
		response.FromError(c, err)
		return
	}
	principal, _ := middleware.Principal(c)
	if create {
		data, err := h.service.Create(c.Request.Context(), principal, body)
		respond(c, http.StatusCreated, "Materi berhasil dibuat", data, err)
		return
	}
	id, err := request.RequireID(c, "materialID")
	if err != nil {
		response.FromError(c, err)
		return
	}
	data, err := h.service.Update(c.Request.Context(), principal, id, body)
	respond(c, http.StatusOK, "Materi berhasil diperbarui", data, err)
}

func (h *Handler) publish(c *gin.Context) {
	h.action(c, h.service.Publish, "Materi berhasil dipublikasikan")
}
func (h *Handler) complete(c *gin.Context) {
	h.action(c, h.service.Complete, "Progress materi berhasil disimpan")
}

func (h *Handler) action(c *gin.Context, action func(ctx context.Context, actor authz.Principal, id string) error, message string) {
	id, err := request.RequireID(c, "materialID")
	if err != nil {
		response.FromError(c, err)
		return
	}
	principal, _ := middleware.Principal(c)
	if err := action(c.Request.Context(), principal, id); err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, http.StatusOK, message, nil)
}

func respond(c *gin.Context, status int, message string, data any, err error) {
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, status, message, data)
}
