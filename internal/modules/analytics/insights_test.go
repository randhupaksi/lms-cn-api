package analytics

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"lms-cn-api/internal/authz"
	"lms-cn-api/pkg/apperror"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestInsightWindowUsesWIBCalendarAndFillsGaps(t *testing.T) {
	// UTC evening belongs to the following school day, across a year boundary.
	now := time.Date(2025, 12, 31, 18, 30, 0, 0, time.UTC)
	start, days := insightWindow(now)
	if len(days) != 14 || days[13].Date != "2026-01-01" || days[0].Date != "2025-12-19" {
		t.Fatalf("unexpected school date window: %#v", days)
	}
	if start.Format(time.RFC3339) != "2025-12-18T17:00:00Z" {
		t.Fatalf("unexpected UTC boundary: %v", start)
	}
	fillActivity(days, []activityCount{{Day: "2026-01-01", Total: 3}, {Day: "2025-12-01", Total: 99}}, []activityCount{{Day: "2025-12-19", Total: 2}})
	if days[13].ExamSubmissions != 3 || days[0].AssignmentSubmissions != 2 || days[5].ExamSubmissions != 0 || days[5].AssignmentSubmissions != 0 {
		t.Fatal("activity did not preserve zero days and the bounded series")
	}
}

func TestInsightsCannotSpoofRoleThroughQueryParameters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(NewService(nil, nil)).RegisterRoutes(router.Group("/analytics"))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest("GET", "/analytics/dashboard/insights?role=admin&user_id=synthetic", nil))
	if recorder.Code != 403 {
		t.Fatalf("query parameters must not grant access: status %d", recorder.Code)
	}
}

func TestInsightCourseScopeUsesAuthenticatedMembership(t *testing.T) {
	db, err := gorm.Open(mysql.New(mysql.Config{DSN: "synthetic:synthetic@tcp(127.0.0.1:1)/synthetic", SkipInitializeWithVersion: true}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ role, contains string }{{"teacher", "scope_member.teacher_id = 'synthetic-id'"}, {"student", "scope_member.student_id = 'synthetic-id'"}, {"invalid", "1 = 0"}, {"admin", "SELECT"}} {
		t.Run(test.role, func(t *testing.T) {
			sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
				return insightCourseScope(tx.Table("exams e"), test.role, "synthetic-id", "e.course_id").Find(&[]struct{ ID string }{})
			})
			if !strings.Contains(sql, test.contains) {
				t.Fatalf("missing scope condition: %s", sql)
			}
			if (test.role == "student" || test.role == "teacher") && (!strings.Contains(sql, "EXISTS") || strings.Contains(sql, "JOIN course_")) {
				t.Fatalf("scope may multiply aggregate rows: %s", sql)
			}
			if test.role == "admin" && strings.Contains(sql, "scope_member.") {
				t.Fatalf("unexpected membership restriction: %s", sql)
			}
		})
	}
}

func TestInsightsRejectUnknownRoleBeforeDatabaseAccess(t *testing.T) {
	service := NewService(nil, nil)
	_, err := service.DashboardInsights(context.Background(), authz.Principal{Role: "unknown"})
	appErr, ok := apperror.As(err)
	if !ok || appErr.Status != 403 {
		t.Fatalf("expected forbidden, got %v", err)
	}
}
