package analytics

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
	group.GET("/dashboard", h.dashboard)
	group.GET("/exams/:examID", h.exam)
}

func (h *Handler) dashboard(c *gin.Context) {
	principal, _ := middleware.Principal(c)
	data, err := h.service.Dashboard(c.Request.Context(), principal)
	h.respond(c, "Ringkasan berhasil dimuat", data, err)
}

func (h *Handler) exam(c *gin.Context) {
	examID, err := request.RequireID(c, "examID")
	if err != nil {
		response.FromError(c, err)
		return
	}
	principal, _ := middleware.Principal(c)
	data, err := h.service.Exam(c.Request.Context(), principal, examID)
	h.respond(c, "Analitik ujian berhasil dimuat", data, err)
}

func (h *Handler) respond(c *gin.Context, message string, data any, err error) {
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, http.StatusOK, message, data)
}
