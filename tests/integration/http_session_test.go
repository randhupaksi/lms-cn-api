package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"lms-cn-api/internal/modules/auth"
)

func verifyAuthenticatedHTTP(t *testing.T, ctx context.Context, router http.Handler, identifier, password string) {
	t.Helper()
	call := func(method, path, token string, cookie *http.Cookie, body []byte, want int) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequestWithContext(ctx, method, "/api/v1"+path, bytes.NewReader(body))
		if len(body) != 0 {
			req.Header.Set("Content-Type", "application/json")
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if cookie != nil {
			req.AddCookie(cookie)
		}
		result := httptest.NewRecorder()
		router.ServeHTTP(result, req)
		if result.Code != want {
			t.Fatalf("%s %s returned %d, want %d", method, path, result.Code, want)
		}
		return result
	}
	readSession := func(rec *httptest.ResponseRecorder) (string, *http.Cookie) {
		t.Helper()
		var envelope struct {
			Data auth.SessionResponse `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil || envelope.Data.AccessToken == "" || envelope.Data.User.Identifier != identifier {
			t.Fatal("missing authenticated student session")
		}
		cookies := rec.Result().Cookies()
		if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].Value == "" {
			t.Fatal("missing HttpOnly refresh cookie")
		}
		return envelope.Data.AccessToken, cookies[0]
	}

	body, err := json.Marshal(auth.LoginRequest{Identifier: identifier, Password: password})
	if err != nil {
		t.Fatal("cannot encode synthetic login")
	}
	token, cookie := readSession(call(http.MethodPost, "/auth/login", "", nil, body, http.StatusOK))
	call(http.MethodGet, "/auth/me", token, nil, nil, http.StatusOK)
	call(http.MethodGet, "/users", token, nil, nil, http.StatusForbidden)
	rotatedToken, rotatedCookie := readSession(call(http.MethodPost, "/auth/refresh", "", cookie, nil, http.StatusOK))
	if rotatedCookie.Value == cookie.Value {
		t.Fatal("refresh cookie did not rotate")
	}
	call(http.MethodPost, "/auth/logout", rotatedToken, rotatedCookie, nil, http.StatusOK)
	call(http.MethodGet, "/auth/me", rotatedToken, nil, nil, http.StatusUnauthorized)
}
