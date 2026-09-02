package attempts

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
	group.GET("/exams", h.listAvailable)
	group.POST("/exams/:examID/start", h.start)
	group.GET("/attempts/:attemptID", h.resume)
	group.PUT("/attempts/:attemptID/answers", h.saveAnswer)
	group.POST("/attempts/:attemptID/submit", h.submit)
}

func (h *Handler) listAvailable(c *gin.Context) {
	principal, _ := middleware.Principal(c)
	data, err := h.service.ListAvailable(c.Request.Context(), principal)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Ujian siswa berhasil dimuat", data)
}

func (h *Handler) start(c *gin.Context) {
	examID, err := request.RequireID(c, "examID")
	if err != nil {
		response.FromError(c, err)
		return
	}
	principal, _ := middleware.Principal(c)
	data, err := h.service.Start(c.Request.Context(), principal, examID, c.GetHeader("Idempotency-Key"))
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Attempt siap dikerjakan", data)
}

func (h *Handler) resume(c *gin.Context) {
	attemptID, err := request.RequireID(c, "attemptID")
	if err != nil {
		response.FromError(c, err)
		return
	}
	principal, _ := middleware.Principal(c)
	data, err := h.service.Resume(c.Request.Context(), principal, attemptID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Attempt berhasil dimuat", data)
}

func (h *Handler) saveAnswer(c *gin.Context) {
	attemptID, err := request.RequireID(c, "attemptID")
	if err != nil {
		response.FromError(c, err)
		return
	}
	var body SaveAnswerRequest
	if err := request.BindJSON(c, &body); err != nil {
		response.FromError(c, err)
		return
	}
	principal, _ := middleware.Principal(c)
	data, err := h.service.SaveAnswer(c.Request.Context(), principal, attemptID, c.GetHeader("Idempotency-Key"), body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Jawaban tersimpan", data)
}

func (h *Handler) submit(c *gin.Context) {
	attemptID, err := request.RequireID(c, "attemptID")
	if err != nil {
		response.FromError(c, err)
		return
	}
	principal, _ := middleware.Principal(c)
	data, err := h.service.Submit(c.Request.Context(), principal, attemptID, c.GetHeader("Idempotency-Key"))
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Ujian berhasil disubmit", data)
}
