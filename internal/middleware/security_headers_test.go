package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, test := range []struct {
		name       string
		production bool
		wantHSTS   bool
	}{
		{name: "development", production: false, wantHSTS: false},
		{name: "production", production: true, wantHSTS: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			engine := gin.New()
			engine.Use(SecurityHeaders(test.production))
			engine.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

			if got := response.Header().Get("X-Content-Type-Options"); got != "nosniff" {
				t.Fatalf("expected nosniff header, got %q", got)
			}
			if got := response.Header().Get("Strict-Transport-Security"); (got != "") != test.wantHSTS {
				t.Fatalf("unexpected HSTS header %q", got)
			}
		})
	}
}
