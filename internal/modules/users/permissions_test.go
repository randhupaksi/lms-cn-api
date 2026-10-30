package users

import (
	"context"
	"lms-cn-api/internal/authz"
	"testing"
)

func TestAdministrativeTargetBoundary(t *testing.T) {
	for _, actor := range []Role{RoleSuperadmin, RolePrincipal, RoleCurriculum, RoleAdmin, RoleTeacher, RoleStudent} {
		for _, target := range []Role{RoleSuperadmin, RolePrincipal, RoleCurriculum, RoleAdmin, RoleTeacher, RoleStudent} {
			allowed := actor == RoleSuperadmin ||
				(actor == RolePrincipal && (target == RoleCurriculum || target == RoleAdmin || target == RoleTeacher || target == RoleStudent)) ||
				(actor == RoleAdmin && (target == RoleTeacher || target == RoleStudent))
			err := canManageTarget(authz.Principal{Role: string(actor)}, User{Role: target})
			if (err == nil) != allowed {
				t.Fatalf("actor %s target %s", actor, target)
			}
		}
	}
}
func TestSensitiveActionsDenyBeforeDatabase(t *testing.T) {
	s := NewService(nil)
	ctx := context.Background()
	for _, role := range []string{"principal", "admin", "teacher", "student", "unknown"} {
		actor := authz.Principal{Role: role}
		if err := s.ChangeRole(ctx, actor, "target", ChangeRoleRequest{Role: RoleSuperadmin}); err == nil {
			t.Fatal("role escalation admitted")
		}
		if _, err := s.Sessions(ctx, actor, "target"); err == nil {
			t.Fatal("session inspection admitted")
		}
		if err := s.RevokeSessions(ctx, actor, "target"); err == nil {
			t.Fatal("session revocation admitted")
		}
	}
}
