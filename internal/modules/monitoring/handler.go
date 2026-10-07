package monitoring

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
	group.GET("/exams/:examID", h.examStatus)
}

func (h *Handler) examStatus(c *gin.Context) {
	examID, err := request.RequireID(c, "examID")
	if err != nil {
		response.FromError(c, err)
		return
	}
	principal, _ := middleware.Principal(c)
	page := pagination.FromContext(c)
	data, total, err := h.service.ExamStatusPage(c.Request.Context(), principal, examID, c.Query("search"), page)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.SuccessWithMeta(c, http.StatusOK, "Monitoring ujian berhasil dimuat", data, page.Meta(total))
}
