package authz

import (
	"net/http"

	"lms-cn-api/pkg/apperror"
)

type Principal struct {
	UserID             string
	SessionID          string
	Role               string
	MustChangePassword bool
}

// IsSchoolAdmin grants school-wide operational scope, never teacher ownership.
func (p Principal) IsSchoolAdmin() bool {
	return p.Role == "admin" || p.Role == "superadmin" || p.Role == "principal"
}

// IsTeachingStaff identifies staff who may author content only after course assignment.
func (p Principal) IsTeachingStaff() bool {
	return p.Role == "teacher" || p.Role == "curriculum"
}

// CanCoordinateAcademics identifies the school-wide academic coordination scope.
func (p Principal) CanCoordinateAcademics() bool {
	return p.IsSchoolAdmin() || p.Role == "curriculum"
}

func (p Principal) RequireRole(roles ...string) error {
	for _, role := range roles {
		if p.Role == role {
			return nil
		}
	}
	return apperror.New(http.StatusForbidden, "FORBIDDEN", "Kamu tidak memiliki izin untuk tindakan ini")
}
