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

type roleAccount struct {
	identifier string
	role       string
}

func verifyRoleRouteMatrix(t *testing.T, ctx context.Context, router http.Handler, password string, accounts []roleAccount) {
	t.Helper()
	for _, account := range accounts {
		t.Run(account.role, func(t *testing.T) {
			body, err := json.Marshal(auth.LoginRequest{Identifier: account.identifier, Password: password})
			if err != nil {
				t.Fatal("cannot encode synthetic role login")
			}
			loginRequest := httptest.NewRequestWithContext(ctx, http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
			loginRequest.Header.Set("Content-Type", "application/json")
			loginResponse := httptest.NewRecorder()
			router.ServeHTTP(loginResponse, loginRequest)
			if loginResponse.Code != http.StatusOK {
				t.Fatalf("role %s login returned %d", account.role, loginResponse.Code)
			}
			var envelope struct {
				Data auth.SessionResponse `json:"data"`
			}
			if err := json.Unmarshal(loginResponse.Body.Bytes(), &envelope); err != nil || envelope.Data.AccessToken == "" || string(envelope.Data.User.Role) != account.role {
				t.Fatalf("role %s login returned an invalid session", account.role)
			}
			cookies := loginResponse.Result().Cookies()
			if len(cookies) != 1 || !cookies[0].HttpOnly {
				t.Fatalf("role %s login returned no HttpOnly refresh cookie", account.role)
			}

			isSchoolAdmin := account.role == "superadmin" || account.role == "principal" || account.role == "admin"
			canReadQuestions := isSchoolAdmin && account.role != "principal" || account.role == "teacher" || account.role == "curriculum"
			isStaff := account.role != "student"
			studentOnly := account.role == "student"
			cases := []struct {
				name string
				path string
				want int
			}{
				{name: "account directory", path: "/users", want: expectedStatus(isSchoolAdmin)},
				{name: "audit viewer", path: "/audit-logs", want: expectedStatus(isSchoolAdmin)},
				{name: "question bank", path: "/questions", want: expectedStatus(canReadQuestions)},
				{name: "staff results", path: "/results", want: expectedStatus(isStaff)},
				{name: "student results", path: "/student/results", want: expectedStatus(studentOnly)},
			}
			for _, testCase := range cases {
				t.Run(testCase.name, func(t *testing.T) {
					request := httptest.NewRequestWithContext(ctx, http.MethodGet, "/api/v1"+testCase.path, nil)
					request.Header.Set("Authorization", "Bearer "+envelope.Data.AccessToken)
					response := httptest.NewRecorder()
					router.ServeHTTP(response, request)
					if response.Code != testCase.want {
						t.Fatalf("%s returned %d, want %d", testCase.path, response.Code, testCase.want)
					}
				})
			}

			logoutRequest := httptest.NewRequestWithContext(ctx, http.MethodPost, "/api/v1/auth/logout", nil)
			logoutRequest.Header.Set("Authorization", "Bearer "+envelope.Data.AccessToken)
			logoutRequest.AddCookie(cookies[0])
			logoutResponse := httptest.NewRecorder()
			router.ServeHTTP(logoutResponse, logoutRequest)
			if logoutResponse.Code != http.StatusOK {
				t.Fatalf("role %s logout returned %d", account.role, logoutResponse.Code)
			}
		})
	}
}

func expectedStatus(allowed bool) int {
	if allowed {
		return http.StatusOK
	}
	return http.StatusForbidden
}

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
