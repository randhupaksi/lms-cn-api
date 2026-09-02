package academics

import (
	"time"

	"lms-cn-api/internal/modules/users"
)

type CreateAcademicYearRequest struct {
	Name     string `json:"name" binding:"required,max=80"`
	StartsOn string `json:"starts_on" binding:"required,datetime=2006-01-02"`
	EndsOn   string `json:"ends_on" binding:"required,datetime=2006-01-02"`
	Status   string `json:"status" binding:"required,oneof=active inactive"`
}

type CreateClassGroupRequest struct {
	AcademicYearID string `json:"academic_year_id" binding:"required"`
	Name           string `json:"name" binding:"required,max=80"`
	GradeLevel     int    `json:"grade_level" binding:"required,min=1,max=12"`
}

type CreateSubjectRequest struct {
	Code string `json:"code" binding:"required,max=32"`
	Name string `json:"name" binding:"required,max=120"`
}

type CreateCourseRequest struct {
	AcademicYearID string `json:"academic_year_id" binding:"required"`
	ClassGroupID   string `json:"class_group_id" binding:"required"`
	SubjectID      string `json:"subject_id" binding:"required"`
	Name           string `json:"name" binding:"required,max=160"`
}

type AssignMembersRequest struct {
	UserIDs []string `json:"user_ids" binding:"required,min=1,dive,required"`
}

type CourseMembersResponse struct {
	Teachers []users.Response `json:"teachers"`
	Students []users.Response `json:"students"`
}

type AcademicYearResponse struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	StartsOn time.Time `json:"starts_on"`
	EndsOn   time.Time `json:"ends_on"`
	Status   string    `json:"status"`
}

type ClassGroupResponse struct {
	ID             string `json:"id"`
	AcademicYearID string `json:"academic_year_id"`
	Name           string `json:"name"`
	GradeLevel     int    `json:"grade_level"`
}

type SubjectResponse struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type CourseResponse struct {
	ID           string               `json:"id"`
	Name         string               `json:"name"`
	Status       string               `json:"status"`
	AcademicYear AcademicYearResponse `json:"academic_year"`
	ClassGroup   ClassGroupResponse   `json:"class_group"`
	Subject      SubjectResponse      `json:"subject"`
}

func toAcademicYearResponse(value AcademicYear) AcademicYearResponse {
	return AcademicYearResponse{ID: value.ID, Name: value.Name, StartsOn: value.StartsOn, EndsOn: value.EndsOn, Status: value.Status}
}

func toClassGroupResponse(value ClassGroup) ClassGroupResponse {
	return ClassGroupResponse{ID: value.ID, AcademicYearID: value.AcademicYearID, Name: value.Name, GradeLevel: value.GradeLevel}
}

func toSubjectResponse(value Subject) SubjectResponse {
	return SubjectResponse{ID: value.ID, Code: value.Code, Name: value.Name}
}

func toCourseResponse(value Course) CourseResponse {
	return CourseResponse{
		ID: value.ID, Name: value.Name, Status: value.Status,
		AcademicYear: toAcademicYearResponse(value.AcademicYear),
		ClassGroup:   toClassGroupResponse(value.ClassGroup), Subject: toSubjectResponse(value.Subject),
	}
}
