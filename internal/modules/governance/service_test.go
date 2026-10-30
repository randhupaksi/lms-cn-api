package governance

import (
	"context"
	"github.com/gin-gonic/gin"
	"lms-cn-api/internal/authz"
	"lms-cn-api/internal/middleware"
	"lms-cn-api/pkg/apperror"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDecisionIntegrity(t *testing.T) {
	now := time.Date(2026, 10, 27, 2, 0, 0, 0, time.UTC)
	exam := ExamReadiness{Status: "published", StartsAt: now.Add(24 * time.Hour), EndsAt: now.Add(26 * time.Hour), DurationMinutes: 60}
	value := ScheduleRequest{Status: "pending", RequesterID: "teacher", OriginalStartsAt: exam.StartsAt, OriginalEndsAt: exam.EndsAt, ProposedStartsAt: now.Add(48 * time.Hour), ProposedEndsAt: now.Add(50 * time.Hour)}
	tests := []struct {
		name, actor, status, code string
		change                    func(*ScheduleRequest, *ExamReadiness)
	}{
		{"valid approval", "owner", "approved", "", nil},
		{"self approval", "teacher", "approved", "SELF_APPROVAL_DENIED", nil},
		{"already decided", "owner", "approved", "REQUEST_ALREADY_DECIDED", func(v *ScheduleRequest, _ *ExamReadiness) { v.Status = "rejected" }},
		{"schedule stale", "owner", "approved", "EXAM_SCHEDULE_CHANGED", func(_ *ScheduleRequest, e *ExamReadiness) { e.StartsAt = e.StartsAt.Add(time.Minute) }},
		{"attempt locks schedule", "owner", "approved", "EXAM_SCHEDULE_LOCKED", func(_ *ScheduleRequest, e *ExamReadiness) { e.AttemptCount = 1 }},
		{"closed exam", "owner", "approved", "EXAM_SCHEDULE_LOCKED", func(_ *ScheduleRequest, e *ExamReadiness) { e.Status = "closed" }},
		{"expired proposal", "owner", "approved", "INVALID_SCHEDULE", func(v *ScheduleRequest, _ *ExamReadiness) { v.ProposedStartsAt = now.Add(-time.Hour) }},
		{"short window", "owner", "approved", "INVALID_SCHEDULE", func(v *ScheduleRequest, _ *ExamReadiness) {
			v.ProposedEndsAt = v.ProposedStartsAt.Add(30 * time.Minute)
		}},
		{"cancel owned", "teacher", "cancelled", "", nil},
		{"cancel other", "other", "cancelled", "REQUEST_NOT_OWNED", nil},
		{"reject locked exam", "owner", "rejected", "", func(_ *ScheduleRequest, e *ExamReadiness) { e.AttemptCount = 1 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, e := value, exam
			if tt.change != nil {
				tt.change(&v, &e)
			}
			err := validateDecision(v, e, tt.actor, tt.status, now)
			if tt.code == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			app, ok := apperror.As(err)
			if !ok || app.Code != tt.code {
				t.Fatalf("wanted %s", tt.code)
			}
		})
	}
}
func TestAuthorizationBeforeRepository(t *testing.T) {
	s := NewService(nil, nil)
	ctx := context.Background()
	if err := owner(authz.Principal{Role: "principal"}); err != nil {
		t.Fatal("principal should share school governance access")
	}
	if err := academicCoordinator(authz.Principal{Role: "curriculum"}); err != nil {
		t.Fatal("curriculum should have academic coordination access")
	}
	if err := owner(authz.Principal{Role: "curriculum"}); err == nil {
		t.Fatal("curriculum should not receive leadership decision access")
	}
	for _, role := range []string{"student", "teacher", "admin", "unknown"} {
		if _, err := s.Overview(ctx, authz.Principal{Role: role}); err == nil {
			t.Fatalf("overview admitted %s", role)
		}
		if _, err := s.Settings(ctx, authz.Principal{Role: role}); err == nil {
			t.Fatalf("settings admitted %s", role)
		}
	}
	if err := s.CreateRequest(ctx, authz.Principal{Role: "superadmin"}, ScheduleInput{}); err == nil {
		t.Fatal("superadmin can create own decision workload")
	}
	if err := s.Decide(ctx, authz.Principal{Role: "teacher"}, "id", DecisionInput{}); err == nil {
		t.Fatal("teacher can decide")
	}
}

type fakeSession struct{ role string }

func (f fakeSession) VerifyAccessToken(string) (authz.Principal, error) {
	return authz.Principal{Role: f.role, UserID: "synthetic", SessionID: "session"}, nil
}
func (f fakeSession) ValidateSession(context.Context, authz.Principal) error { return nil }
func TestGovernanceHTTPRoleBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, test := range []struct{ role, path string }{{"teacher", "overview"}, {"admin", "settings"}, {"admin", "reports/export"}, {"teacher", "quality"}, {"student", "requests"}, {"student", "request-exams"}} {
		r := gin.New()
		g := r.Group("/governance")
		g.Use(middleware.Authenticate(fakeSession{role: test.role}))
		NewHandler(NewService(nil, nil)).RegisterRoutes(g)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/governance/"+test.path, nil)
		req.Header.Set("Authorization", "Bearer synthetic")
		r.ServeHTTP(w, req)
		if w.Code != 403 {
			t.Fatalf("%s %s status %d", test.role, test.path, w.Code)
		}
	}
}
func TestCSVFormulaEscaping(t *testing.T) {
	for _, value := range []string{"=1+1", " +SUM(A1)", "\t@formula", "-2", "+3"} {
		if !strings.HasPrefix(safeCSV(value), "'") {
			t.Fatal("unsafe CSV cell")
		}
	}
	if safeCSV("Kelas XII") != "Kelas XII" {
		t.Fatal("plain label modified")
	}
}
