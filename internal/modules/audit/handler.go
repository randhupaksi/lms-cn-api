package audit

import (
	"net/http"

	"lms-cn-api/internal/middleware"
	"lms-cn-api/pkg/pagination"
	"lms-cn-api/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("", h.list)
}

func (h *Handler) list(c *gin.Context) {
	principal, _ := middleware.Principal(c)
	page := pagination.FromContext(c)
	filter := Filter{Action: c.Query("action"), EntityType: c.Query("entity_type"), ActorID: c.Query("actor_id")}
	data, total, err := h.service.List(c.Request.Context(), principal, page, filter)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.SuccessWithMeta(c, http.StatusOK, "Audit aktivitas berhasil dimuat", data, page.Meta(total))
}
