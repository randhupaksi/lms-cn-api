package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"lms-cn-api/pkg/response"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func TestRecoverKeepsRequestAndPanicDataPrivate(t *testing.T) {
	var logs bytes.Buffer
	original := slog.Default()
	t.Cleanup(func() { slog.SetDefault(original) })
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))

	router := gin.New()
	router.Use(RequestID(), RequestLogger(), gzip.Gzip(gzip.DefaultCompression), Recover())
	router.GET("/synthetic/:id", func(_ *gin.Context) { panic("synthetic-private-panic") })
	req := httptest.NewRequest(http.MethodGet, "/synthetic/synthetic-private-path?value=synthetic-private-query", nil)
	req.Header.Set("Cookie", "test=synthetic-private-cookie")
	req.Header.Set("Accept-Encoding", "gzip")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	var envelope response.Envelope
	if err := json.Unmarshal(res.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if res.Code != http.StatusInternalServerError || envelope.Success || envelope.Code != "INTERNAL_ERROR" {
		t.Fatalf("unexpected recovery response: %d, %+v", res.Code, envelope)
	}
	if strings.Contains(logs.String()+res.Body.String(), "synthetic-private") {
		t.Fatal("recovery exposed confidential request or panic data")
	}
	requestID := res.Header().Get("X-Request-ID")
	if requestID == "" || !strings.Contains(logs.String(), requestID) || !strings.Contains(logs.String(), "/synthetic/:id") {
		t.Fatal("recovery log is missing correlation metadata")
	}
}
