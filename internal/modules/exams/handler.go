package exams

import (
	"context"
	"net/http"

	"lms-cn-api/internal/authz"
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
	group.GET("/:examID", h.find)
	group.PUT("/:examID", h.update)
	group.PUT("/:examID/questions", h.setQuestions)
	group.PUT("/:examID/participants", h.setParticipants)
	group.POST("/:examID/publish", h.publish)
	group.POST("/:examID/unpublish", h.unpublish)
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
	response.SuccessWithMeta(c, http.StatusOK, "Ujian berhasil dimuat", data, page.Meta(total))
}

func (h *Handler) find(c *gin.Context) {
	id, ok := examID(c)
	if !ok {
		return
	}
	principal, _ := middleware.Principal(c)
	data, err := h.service.Find(c.Request.Context(), principal, id)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Ujian berhasil dimuat", data)
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
	var data Response
	var err error
	status, message := http.StatusOK, "Ujian berhasil diperbarui"
	if create {
		data, err = h.service.Create(c.Request.Context(), principal, body)
		status, message = http.StatusCreated, "Ujian berhasil dibuat"
	} else {
		id, ok := examID(c)
		if !ok {
			return
		}
		data, err = h.service.Update(c.Request.Context(), principal, id, body)
	}
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, status, message, data)
}

func (h *Handler) setQuestions(c *gin.Context) {
	id, ok := examID(c)
	if !ok {
		return
	}
	var body SetQuestionsRequest
	if err := request.BindJSON(c, &body); err != nil {
		response.FromError(c, err)
		return
	}
	principal, _ := middleware.Principal(c)
	if err := h.service.SetQuestions(c.Request.Context(), principal, id, body); err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Soal ujian berhasil diperbarui", nil)
}

func (h *Handler) setParticipants(c *gin.Context) {
	id, ok := examID(c)
	if !ok {
		return
	}
	var body SetParticipantsRequest
	if err := request.BindJSON(c, &body); err != nil {
		response.FromError(c, err)
		return
	}
	principal, _ := middleware.Principal(c)
	if err := h.service.SetParticipants(c.Request.Context(), principal, id, body); err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Peserta ujian berhasil diperbarui", nil)
}

func (h *Handler) publish(c *gin.Context) {
	h.transition(c, h.service.Publish, "Ujian berhasil dipublikasikan")
}
func (h *Handler) unpublish(c *gin.Context) {
	h.transition(c, h.service.Unpublish, "Publikasi ujian berhasil dibatalkan")
}

func (h *Handler) transition(c *gin.Context, action func(context.Context, authz.Principal, string) error, message string) {
	id, ok := examID(c)
	if !ok {
		return
	}
	principal, _ := middleware.Principal(c)
	if err := action(c.Request.Context(), principal, id); err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, http.StatusOK, message, nil)
}

func examID(c *gin.Context) (string, bool) {
	id, err := request.RequireID(c, "examID")
	if err != nil {
		response.FromError(c, err)
		return "", false
	}
	return id, true
}
