package governance

import (
	"context"
	"testing"

	"lms-cn-api/internal/authz"
	"lms-cn-api/pkg/apperror"
)

func TestLeadershipActionAuthorizationBeforeRepository(t *testing.T) {
	service := NewService(nil, nil)
	ctx := context.Background()
	if err := service.CreateLeadershipAction(ctx, authz.Principal{Role: "teacher"}, CreateLeadershipActionInput{}); err == nil {
		t.Fatal("teacher created a leadership action")
	}
	if _, _, err := service.ListLeadershipActions(ctx, authz.Principal{Role: "student"}, "", "", 1, 20); err == nil {
		t.Fatal("student listed leadership actions")
	}
	if _, err := service.ActionAssignees(ctx, authz.Principal{Role: "admin"}); err == nil {
		t.Fatal("admin read the school-wide assignee directory")
	}
}

func TestLeadershipActionAssigneeScope(t *testing.T) {
	assignee := "teacher-a"
	value := LeadershipAction{AssigneeID: &assignee}
	for _, test := range []struct {
		role, user string
		allowed    bool
	}{
		{"principal", "principal-a", true}, {"superadmin", "superadmin-a", true},
		{"teacher", "teacher-a", true}, {"teacher", "teacher-b", false},
		{"admin", "admin-a", false}, {"student", "teacher-a", false},
	} {
		if got := canAccessLeadershipAction(authz.Principal{Role: test.role, UserID: test.user}, value); got != test.allowed {
			t.Fatalf("role %s user %s: got %v", test.role, test.user, got)
		}
	}
}

func TestLeadershipActionStatusPolicy(t *testing.T) {
	tests := []struct {
		name, role, current, next, wantCode string
	}{
		{"teacher starts assigned work", "teacher", "open", "in_progress", ""},
		{"admin blocks assigned work", "admin", "in_progress", "blocked", ""},
		{"teacher completes assigned work", "teacher", "blocked", "completed", ""},
		{"staff cannot cancel", "teacher", "open", "cancelled", "ACTION_TRANSITION_DENIED"},
		{"staff cannot reopen completed work", "admin", "completed", "in_progress", "ACTION_TRANSITION_DENIED"},
		{"leader can reopen work", "principal", "completed", "in_progress", ""},
		{"superadmin can cancel work", "superadmin", "open", "cancelled", ""},
		{"unchanged status conflicts", "principal", "open", "open", "ACTION_STATUS_UNCHANGED"},
		{"students cannot update", "student", "open", "in_progress", "LEADERSHIP_ACTION_DENIED"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validLeadershipActionStatus(test.role, test.current, test.next)
			if test.wantCode == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			app, ok := apperror.As(err)
			if !ok || app.Code != test.wantCode {
				t.Fatalf("wanted %s, got %v", test.wantCode, err)
			}
		})
	}
}
