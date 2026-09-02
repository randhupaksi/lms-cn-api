package results

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

func (h *Handler) RegisterStaffRoutes(group *gin.RouterGroup) {
	group.GET("", h.listByExam)
	group.POST("/exams/:examID/publish", h.publishByExam)
}

func (h *Handler) RegisterStudentRoutes(group *gin.RouterGroup) {
	group.GET("", h.listStudent)
	group.GET("/:resultID", h.findStudent)
}

func (h *Handler) listByExam(c *gin.Context) {
	examID := c.Query("exam_id")
	if examID == "" {
		response.Error(c, http.StatusBadRequest, "EXAM_ID_REQUIRED", "Ujian wajib dipilih")
		return
	}
	principal, _ := middleware.Principal(c)
	page := pagination.FromContext(c)
	data, total, err := h.service.ListByExam(c.Request.Context(), principal, examID, page)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.SuccessWithMeta(c, http.StatusOK, "Hasil ujian berhasil dimuat", data, page.Meta(total))
}

func (h *Handler) publishByExam(c *gin.Context) {
	examID, err := request.RequireID(c, "examID")
	if err != nil {
		response.FromError(c, err)
		return
	}
	principal, _ := middleware.Principal(c)
	count, err := h.service.PublishByExam(c.Request.Context(), principal, examID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Hasil ujian berhasil dipublikasikan", gin.H{"published_count": count})
}

func (h *Handler) listStudent(c *gin.Context) {
	principal, _ := middleware.Principal(c)
	page := pagination.FromContext(c)
	data, total, err := h.service.ListStudent(c.Request.Context(), principal, page)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.SuccessWithMeta(c, http.StatusOK, "Hasil siswa berhasil dimuat", data, page.Meta(total))
}

func (h *Handler) findStudent(c *gin.Context) {
	resultID, err := request.RequireID(c, "resultID")
	if err != nil {
		response.FromError(c, err)
		return
	}
	principal, _ := middleware.Principal(c)
	data, err := h.service.FindStudent(c.Request.Context(), principal, resultID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Hasil siswa berhasil dimuat", data)
}
