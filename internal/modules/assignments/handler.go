package assignments

import (
	"net/http"

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
	group.PUT("/:assignmentID", h.update)
	group.POST("/:assignmentID/publish", h.publish)
	group.POST("/:assignmentID/submit", h.submit)
	group.GET("/:assignmentID/submissions", h.listSubmissions)
	group.POST("/submissions/:submissionID/grade", h.grade)
}

func (h *Handler) list(c *gin.Context) {
	courseID := c.Query("course_id")
	if courseID == "" {
		response.Error(c, http.StatusBadRequest, "COURSE_ID_REQUIRED", "Course wajib dipilih")
		return
	}
	principal, _ := middleware.Principal(c)
	data, err := h.service.List(c.Request.Context(), principal, courseID)
	h.respond(c, http.StatusOK, "Tugas berhasil dimuat", data, err)
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
		h.respond(c, http.StatusCreated, "Tugas berhasil dibuat", data, err)
		return
	}
	id, err := request.RequireID(c, "assignmentID")
	if err != nil {
		response.FromError(c, err)
		return
	}
	data, err := h.service.Update(c.Request.Context(), principal, id, body)
	h.respond(c, http.StatusOK, "Tugas berhasil diperbarui", data, err)
}

func (h *Handler) publish(c *gin.Context) {
	id, ok := h.requireID(c, "assignmentID")
	if !ok {
		return
	}
	principal, _ := middleware.Principal(c)
	h.respond(c, http.StatusOK, "Tugas berhasil dipublikasikan", nil, h.service.Publish(c.Request.Context(), principal, id))
}

func (h *Handler) submit(c *gin.Context) {
	id, ok := h.requireID(c, "assignmentID")
	if !ok {
		return
	}
	var body SubmitRequest
	if err := request.BindJSON(c, &body); err != nil {
		response.FromError(c, err)
		return
	}
	principal, _ := middleware.Principal(c)
	h.respond(c, http.StatusOK, "Tugas berhasil dikumpulkan", nil, h.service.Submit(c.Request.Context(), principal, id, body))
}

func (h *Handler) listSubmissions(c *gin.Context) {
	id, ok := h.requireID(c, "assignmentID")
	if !ok {
		return
	}
	principal, _ := middleware.Principal(c)
	data, err := h.service.ListSubmissions(c.Request.Context(), principal, id)
	h.respond(c, http.StatusOK, "Pengumpulan tugas berhasil dimuat", data, err)
}

func (h *Handler) grade(c *gin.Context) {
	id, ok := h.requireID(c, "submissionID")
	if !ok {
		return
	}
	var body GradeRequest
	if err := request.BindJSON(c, &body); err != nil {
		response.FromError(c, err)
		return
	}
	principal, _ := middleware.Principal(c)
	h.respond(c, http.StatusOK, "Nilai tugas berhasil disimpan", nil, h.service.Grade(c.Request.Context(), principal, id, body))
}

func (h *Handler) requireID(c *gin.Context, name string) (string, bool) {
	id, err := request.RequireID(c, name)
	if err != nil {
		response.FromError(c, err)
		return "", false
	}
	return id, true
}

func (h *Handler) respond(c *gin.Context, status int, message string, data any, err error) {
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, status, message, data)
}
