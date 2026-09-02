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

func (p Principal) RequireRole(roles ...string) error {
	for _, role := range roles {
		if p.Role == role {
			return nil
		}
	}
	return apperror.New(http.StatusForbidden, "FORBIDDEN", "Kamu tidak memiliki izin untuk tindakan ini")
}
