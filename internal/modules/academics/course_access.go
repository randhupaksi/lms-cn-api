package academics

import (
	"context"

	"lms-cn-api/internal/authz"
)

// CourseManagerAccess checks whether a principal may manage a course.
type CourseManagerAccess interface {
	RequireCourseManager(context.Context, authz.Principal, string) error
}

// CourseAccess checks manager access and student enrollment for a course.
type CourseAccess interface {
	CourseManagerAccess
	RequireCourseStudent(context.Context, authz.Principal, string) error
}
