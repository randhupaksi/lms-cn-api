package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimiterRejectsRequestsAboveLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := NewRateLimiter(2, time.Minute)
	engine := gin.New()
	engine.Use(limiter.Middleware())
	engine.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	for index, expected := range []int{http.StatusNoContent, http.StatusNoContent, http.StatusTooManyRequests} {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.RemoteAddr = "192.0.2.10:1234"
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, request)
		if response.Code != expected {
			t.Fatalf("request %d: expected %d, got %d", index+1, expected, response.Code)
		}
	}
}
