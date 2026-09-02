package academics

import (
	"context"
	"net/http"

	"lms-cn-api/internal/authz"
	"lms-cn-api/internal/middleware"
	"lms-cn-api/pkg/pagination"
	"lms-cn-api/pkg/request"
	"lms-cn-api/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) RegisterRoutes(api *gin.RouterGroup) {
	api.GET("/academic-years", h.listAcademicYears)
	api.POST("/academic-years", h.createAcademicYear)
	api.GET("/class-groups", h.listClassGroups)
	api.POST("/class-groups", h.createClassGroup)
	api.GET("/subjects", h.listSubjects)
	api.POST("/subjects", h.createSubject)
	api.GET("/courses", h.listCourses)
	api.POST("/courses", h.createCourse)
	api.GET("/courses/:courseID/members", h.courseMembers)
	api.PUT("/courses/:courseID/teachers", h.assignTeachers)
	api.PUT("/courses/:courseID/students", h.assignStudents)
}

func (h *Handler) courseMembers(c *gin.Context) {
	courseID, err := request.RequireID(c, "courseID")
	if err != nil {
		response.FromError(c, err)
		return
	}
	principal, _ := middleware.Principal(c)
	data, err := h.service.CourseMembers(c.Request.Context(), principal, courseID)
	respond(c, http.StatusOK, "Anggota course berhasil dimuat", data, err)
}

func (h *Handler) listAcademicYears(c *gin.Context) {
	principal, _ := middleware.Principal(c)
	data, err := h.service.ListAcademicYears(c.Request.Context(), principal)
	respond(c, http.StatusOK, "Tahun ajaran berhasil dimuat", data, err)
}

func (h *Handler) createAcademicYear(c *gin.Context) {
	var body CreateAcademicYearRequest
	if !bind(c, &body) {
		return
	}
	principal, _ := middleware.Principal(c)
	data, err := h.service.CreateAcademicYear(c.Request.Context(), principal, body)
	respond(c, http.StatusCreated, "Tahun ajaran berhasil dibuat", data, err)
}

func (h *Handler) listClassGroups(c *gin.Context) {
	principal, _ := middleware.Principal(c)
	data, err := h.service.ListClassGroups(c.Request.Context(), principal, c.Query("academic_year_id"))
	respond(c, http.StatusOK, "Kelas berhasil dimuat", data, err)
}

func (h *Handler) createClassGroup(c *gin.Context) {
	var body CreateClassGroupRequest
	if !bind(c, &body) {
		return
	}
	principal, _ := middleware.Principal(c)
	data, err := h.service.CreateClassGroup(c.Request.Context(), principal, body)
	respond(c, http.StatusCreated, "Kelas berhasil dibuat", data, err)
}

func (h *Handler) listSubjects(c *gin.Context) {
	principal, _ := middleware.Principal(c)
	data, err := h.service.ListSubjects(c.Request.Context(), principal)
	respond(c, http.StatusOK, "Mata pelajaran berhasil dimuat", data, err)
}

func (h *Handler) createSubject(c *gin.Context) {
	var body CreateSubjectRequest
	if !bind(c, &body) {
		return
	}
	principal, _ := middleware.Principal(c)
	data, err := h.service.CreateSubject(c.Request.Context(), principal, body)
	respond(c, http.StatusCreated, "Mata pelajaran berhasil dibuat", data, err)
}

func (h *Handler) listCourses(c *gin.Context) {
	principal, _ := middleware.Principal(c)
	page := pagination.FromContext(c)
	data, total, err := h.service.ListCourses(c.Request.Context(), principal, page)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.SuccessWithMeta(c, http.StatusOK, "Course berhasil dimuat", data, page.Meta(total))
}

func (h *Handler) createCourse(c *gin.Context) {
	var body CreateCourseRequest
	if !bind(c, &body) {
		return
	}
	principal, _ := middleware.Principal(c)
	data, err := h.service.CreateCourse(c.Request.Context(), principal, body)
	respond(c, http.StatusCreated, "Course berhasil dibuat", data, err)
}

func (h *Handler) assignTeachers(c *gin.Context) {
	h.assignMembers(c, h.service.AssignTeachers, "Assignment guru berhasil diperbarui")
}

func (h *Handler) assignStudents(c *gin.Context) {
	h.assignMembers(c, h.service.AssignStudents, "Peserta course berhasil diperbarui")
}

func (h *Handler) assignMembers(c *gin.Context, action func(ctx context.Context, actor authz.Principal, courseID string, userIDs []string) error, message string) {
	courseID, err := request.RequireID(c, "courseID")
	if err != nil {
		response.FromError(c, err)
		return
	}
	var body AssignMembersRequest
	if !bind(c, &body) {
		return
	}
	principal, _ := middleware.Principal(c)
	if err := action(c.Request.Context(), principal, courseID, body.UserIDs); err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, http.StatusOK, message, nil)
}

func bind(c *gin.Context, target any) bool {
	if err := request.BindJSON(c, target); err != nil {
		response.FromError(c, err)
		return false
	}
	return true
}

func respond(c *gin.Context, status int, message string, data any, err error) {
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, status, message, data)
}
