package academics

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"lms-cn-api/internal/authz"
	"lms-cn-api/internal/modules/audit"
	"lms-cn-api/internal/modules/users"
	"lms-cn-api/pkg/apperror"
	"lms-cn-api/pkg/pagination"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct {
	repository *Repository
	audit      audit.Recorder
}

func NewService(repository *Repository, auditRecorder audit.Recorder) *Service {
	return &Service{repository: repository, audit: auditRecorder}
}

func (s *Service) CreateAcademicYear(ctx context.Context, actor authz.Principal, request CreateAcademicYearRequest) (AcademicYearResponse, error) {
	if err := actor.RequireRole(string(users.RoleAdmin)); err != nil {
		return AcademicYearResponse{}, err
	}
	startsOn, _ := time.Parse("2006-01-02", request.StartsOn)
	endsOn, _ := time.Parse("2006-01-02", request.EndsOn)
	if endsOn.Before(startsOn) {
		return AcademicYearResponse{}, apperror.New(http.StatusUnprocessableEntity, "INVALID_ACADEMIC_DATES", "Tanggal akhir harus setelah tanggal mulai")
	}
	value := AcademicYear{ID: uuid.NewString(), Name: strings.TrimSpace(request.Name), StartsOn: startsOn, EndsOn: endsOn, Status: request.Status}
	if err := s.repository.CreateAcademicYear(ctx, &value); err != nil {
		return AcademicYearResponse{}, mapWriteError(err, "Tahun ajaran sudah tersedia")
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "academic_year.created", EntityType: "academic_year", EntityID: value.ID})
	return toAcademicYearResponse(value), nil
}

func (s *Service) ListAcademicYears(ctx context.Context, actor authz.Principal) ([]AcademicYearResponse, error) {
	if err := actor.RequireRole(string(users.RoleAdmin), string(users.RoleTeacher)); err != nil {
		return nil, err
	}
	values, err := s.repository.ListAcademicYears(ctx)
	if err != nil {
		return nil, internalReadError(err)
	}
	result := make([]AcademicYearResponse, len(values))
	for i, value := range values {
		result[i] = toAcademicYearResponse(value)
	}
	return result, nil
}

func (s *Service) CreateClassGroup(ctx context.Context, actor authz.Principal, request CreateClassGroupRequest) (ClassGroupResponse, error) {
	if err := actor.RequireRole(string(users.RoleAdmin)); err != nil {
		return ClassGroupResponse{}, err
	}
	value := ClassGroup{ID: uuid.NewString(), AcademicYearID: request.AcademicYearID, Name: strings.TrimSpace(request.Name), GradeLevel: request.GradeLevel}
	if err := s.repository.CreateClassGroup(ctx, &value); err != nil {
		return ClassGroupResponse{}, mapWriteError(err, "Kelas sudah tersedia pada tahun ajaran ini")
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "class_group.created", EntityType: "class_group", EntityID: value.ID})
	return toClassGroupResponse(value), nil
}

func (s *Service) ListClassGroups(ctx context.Context, actor authz.Principal, yearID string) ([]ClassGroupResponse, error) {
	if err := actor.RequireRole(string(users.RoleAdmin), string(users.RoleTeacher)); err != nil {
		return nil, err
	}
	values, err := s.repository.ListClassGroups(ctx, yearID)
	if err != nil {
		return nil, internalReadError(err)
	}
	result := make([]ClassGroupResponse, len(values))
	for i, value := range values {
		result[i] = toClassGroupResponse(value)
	}
	return result, nil
}

func (s *Service) CreateSubject(ctx context.Context, actor authz.Principal, request CreateSubjectRequest) (SubjectResponse, error) {
	if err := actor.RequireRole(string(users.RoleAdmin)); err != nil {
		return SubjectResponse{}, err
	}
	value := Subject{ID: uuid.NewString(), Code: strings.ToUpper(strings.TrimSpace(request.Code)), Name: strings.TrimSpace(request.Name)}
	if err := s.repository.CreateSubject(ctx, &value); err != nil {
		return SubjectResponse{}, mapWriteError(err, "Mata pelajaran dengan kode atau nama tersebut sudah tersedia")
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "subject.created", EntityType: "subject", EntityID: value.ID})
	return toSubjectResponse(value), nil
}

func (s *Service) ListSubjects(ctx context.Context, actor authz.Principal) ([]SubjectResponse, error) {
	if err := actor.RequireRole(string(users.RoleAdmin), string(users.RoleTeacher)); err != nil {
		return nil, err
	}
	values, err := s.repository.ListSubjects(ctx)
	if err != nil {
		return nil, internalReadError(err)
	}
	result := make([]SubjectResponse, len(values))
	for i, value := range values {
		result[i] = toSubjectResponse(value)
	}
	return result, nil
}

func (s *Service) CreateCourse(ctx context.Context, actor authz.Principal, request CreateCourseRequest) (CourseResponse, error) {
	if err := actor.RequireRole(string(users.RoleAdmin)); err != nil {
		return CourseResponse{}, err
	}
	value := Course{ID: uuid.NewString(), AcademicYearID: request.AcademicYearID, ClassGroupID: request.ClassGroupID, SubjectID: request.SubjectID, Name: strings.TrimSpace(request.Name), Status: "active"}
	if err := s.repository.CreateCourse(ctx, &value); err != nil {
		return CourseResponse{}, mapWriteError(err, "Course untuk konteks akademik tersebut sudah tersedia")
	}
	created, err := s.repository.FindCourse(ctx, value.ID)
	if err != nil {
		return CourseResponse{}, internalReadError(err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "course.created", EntityType: "course", EntityID: value.ID})
	return toCourseResponse(created), nil
}

func (s *Service) ListCourses(ctx context.Context, actor authz.Principal, page pagination.Request) ([]CourseResponse, int64, error) {
	values, total, err := s.repository.ListCourses(ctx, actor.Role, actor.UserID, page)
	if err != nil {
		return nil, 0, internalReadError(err)
	}
	result := make([]CourseResponse, len(values))
	for i, value := range values {
		result[i] = toCourseResponse(value)
	}
	return result, total, nil
}

func (s *Service) AssignTeachers(ctx context.Context, actor authz.Principal, courseID string, userIDs []string) error {
	if err := actor.RequireRole(string(users.RoleAdmin)); err != nil {
		return err
	}
	if err := s.repository.ReplaceTeachers(ctx, courseID, userIDs); err != nil {
		return mapAssignmentError(err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "course.teachers_assigned", EntityType: "course", EntityID: courseID, Metadata: map[string]any{"count": len(userIDs)}})
	return nil
}

func (s *Service) AssignStudents(ctx context.Context, actor authz.Principal, courseID string, userIDs []string) error {
	if err := actor.RequireRole(string(users.RoleAdmin)); err != nil {
		return err
	}
	if err := s.repository.ReplaceStudents(ctx, courseID, userIDs); err != nil {
		return mapAssignmentError(err)
	}
	_ = s.audit.Record(ctx, audit.Record{ActorID: actor.UserID, Action: "course.students_assigned", EntityType: "course", EntityID: courseID, Metadata: map[string]any{"count": len(userIDs)}})
	return nil
}

func (s *Service) CourseMembers(ctx context.Context, actor authz.Principal, courseID string) (CourseMembersResponse, error) {
	if err := s.RequireCourseManager(ctx, actor, courseID); err != nil {
		return CourseMembersResponse{}, err
	}
	teachers, students, err := s.repository.CourseMembers(ctx, courseID)
	if err != nil {
		return CourseMembersResponse{}, internalReadError(err)
	}
	result := CourseMembersResponse{Teachers: make([]users.Response, len(teachers)), Students: make([]users.Response, len(students))}
	for index, teacher := range teachers {
		result.Teachers[index] = users.ToResponse(teacher)
	}
	for index, student := range students {
		result.Students[index] = users.ToResponse(student)
	}
	return result, nil
}

func (s *Service) RequireCourseManager(ctx context.Context, actor authz.Principal, courseID string) error {
	if actor.Role == string(users.RoleAdmin) {
		return nil
	}
	if actor.Role != string(users.RoleTeacher) {
		return apperror.New(http.StatusForbidden, "COURSE_ACCESS_DENIED", "Course tidak tersedia untuk akun ini")
	}
	allowed, err := s.repository.IsTeacherAssigned(ctx, courseID, actor.UserID)
	if err != nil {
		return internalReadError(err)
	}
	if !allowed {
		return apperror.New(http.StatusForbidden, "COURSE_ACCESS_DENIED", "Course tidak tersedia untuk akun ini")
	}
	return nil
}

func (s *Service) RequireCourseStudent(ctx context.Context, actor authz.Principal, courseID string) error {
	if actor.Role != string(users.RoleStudent) {
		return apperror.New(http.StatusForbidden, "COURSE_ACCESS_DENIED", "Course tidak tersedia untuk akun ini")
	}
	allowed, err := s.repository.IsStudentEnrolled(ctx, courseID, actor.UserID)
	if err != nil {
		return internalReadError(err)
	}
	if !allowed {
		return apperror.New(http.StatusForbidden, "COURSE_ACCESS_DENIED", "Course tidak tersedia untuk akun ini")
	}
	return nil
}

func mapWriteError(err error, conflictMessage string) error {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return apperror.New(http.StatusConflict, "ACADEMIC_CONFLICT", conflictMessage)
	}
	if errors.Is(err, gorm.ErrForeignKeyViolated) {
		return apperror.New(http.StatusUnprocessableEntity, "ACADEMIC_REFERENCE_INVALID", "Relasi akademik tidak valid")
	}
	return apperror.Wrap(http.StatusInternalServerError, "ACADEMIC_WRITE_FAILED", "Gagal menyimpan data akademik", err)
}

func mapAssignmentError(err error) error {
	if errors.Is(err, ErrNotFound) {
		return apperror.New(http.StatusUnprocessableEntity, "ASSIGNMENT_MEMBER_INVALID", "Ada pengguna yang tidak aktif atau role-nya tidak sesuai")
	}
	return apperror.Wrap(http.StatusInternalServerError, "ASSIGNMENT_FAILED", "Gagal menyimpan assignment", err)
}

func internalReadError(err error) error {
	return apperror.Wrap(http.StatusInternalServerError, "ACADEMIC_READ_FAILED", "Gagal memuat data akademik", err)
}
