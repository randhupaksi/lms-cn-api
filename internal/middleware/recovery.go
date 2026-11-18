package middleware

import (
	"log/slog"
	"net/http"

	"lms-cn-api/pkg/response"

	"github.com/gin-gonic/gin"
)

// Recover logs only request metadata because a panic value or request dump can
// contain credentials or confidential data.
func Recover() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, _ any) {
		slog.ErrorContext(c.Request.Context(), "http handler panicked",
			"request_id", c.Writer.Header().Get("X-Request-ID"),
			"method", c.Request.Method,
			"path", c.FullPath(),
		)
		c.Abort()
		if !c.Writer.Written() {
			response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Terjadi kesalahan pada server")
		}
	})
}
