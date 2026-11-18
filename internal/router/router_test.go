package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"lms-cn-api/internal/config"
	"lms-cn-api/pkg/response"

	"gorm.io/gorm"
)

func TestRegisteredRoutesRequireAuthenticationByDefault(t *testing.T) {
	const prefix = "/api/v1"
	engine, err := New(&config.Config{
		APIPrefix: prefix, AllowedOrigins: []string{"https://synthetic.example"},
		AccessTokenTTL: time.Minute, RefreshTokenTTL: time.Hour,
		LoginRateLimit: 10, LoginRateWindow: time.Minute,
	}, &gorm.DB{})
	if err != nil {
		t.Fatal(err)
	}
	public := map[string]bool{
		"GET " + prefix + "/health":         true,
		"GET " + prefix + "/health/live":    true,
		"GET " + prefix + "/school-profile": true,
		"POST " + prefix + "/auth/login":    true,
		"POST " + prefix + "/auth/refresh":  true,
	}
	parameters := regexp.MustCompile(`[:*][^/]+`)
	protected := 0
	for _, route := range engine.Routes() {
		name := route.Method + " " + route.Path
		if public[name] {
			delete(public, name)
			continue
		}
		protected++
		t.Run(name, func(t *testing.T) {
			path := parameters.ReplaceAllString(route.Path, "synthetic")
			req := httptest.NewRequest(route.Method, path, nil)
			res := httptest.NewRecorder()
			engine.ServeHTTP(res, req)
			var envelope response.Envelope
			if err := json.Unmarshal(res.Body.Bytes(), &envelope); err != nil || res.Code != http.StatusUnauthorized || envelope.Success || envelope.Code != "UNAUTHENTICATED" {
				t.Fatalf("expected unauthenticated API response, got status=%d code=%q decode=%v", res.Code, envelope.Code, err)
			}
		})
	}
	if len(public) != 0 || protected == 0 {
		t.Fatalf("unexpected route inventory: missing public routes=%v, protected routes=%d", public, protected)
	}
	t.Logf("checked %d protected routes", protected)
}
