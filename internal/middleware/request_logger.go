package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()
		slog.InfoContext(c.Request.Context(), "http request completed",
			"request_id", c.Writer.Header().Get("X-Request-ID"),
			"method", c.Request.Method,
			"path", c.FullPath(),
			"status", c.Writer.Status(),
			"duration_ms", time.Since(startedAt).Milliseconds(),
			"client_ip", c.ClientIP(),
		)
	}
}
