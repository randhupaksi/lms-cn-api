package academics

import (
	"context"
	"errors"

	"lms-cn-api/internal/modules/users"
	"lms-cn-api/pkg/pagination"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("academic resource not found")

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) CreateAcademicYear(ctx context.Context, value *AcademicYear) error {
	return r.db.WithContext(ctx).Create(value).Error
}

func (r *Repository) ListAcademicYears(ctx context.Context) ([]AcademicYear, error) {
	var result []AcademicYear
	return result, r.db.WithContext(ctx).Order("starts_on DESC").Find(&result).Error
}

func (r *Repository) CreateClassGroup(ctx context.Context, value *ClassGroup) error {
	return r.db.WithContext(ctx).Create(value).Error
}

func (r *Repository) ListClassGroups(ctx context.Context, academicYearID string) ([]ClassGroup, error) {
	query := r.db.WithContext(ctx).Model(&ClassGroup{})
	if academicYearID != "" {
		query = query.Where("academic_year_id = ?", academicYearID)
	}
	var result []ClassGroup
	return result, query.Order("grade_level ASC, name ASC").Find(&result).Error
}

func (r *Repository) CreateSubject(ctx context.Context, value *Subject) error {
	return r.db.WithContext(ctx).Create(value).Error
}

func (r *Repository) ListSubjects(ctx context.Context) ([]Subject, error) {
	var result []Subject
	return result, r.db.WithContext(ctx).Order("name ASC").Find(&result).Error
}

func (r *Repository) CreateCourse(ctx context.Context, value *Course) error {
	return r.db.WithContext(ctx).Create(value).Error
}

func (r *Repository) FindCourse(ctx context.Context, id string) (Course, error) {
	var value Course
	err := r.db.WithContext(ctx).Preload("AcademicYear").Preload("ClassGroup").Preload("Subject").First(&value, "courses.id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Course{}, ErrNotFound
	}
	return value, err
}

func (r *Repository) ListCourses(ctx context.Context, principalRole, userID string, page pagination.Request) ([]Course, int64, error) {
	query := r.db.WithContext(ctx).Model(&Course{})
	switch principalRole {
	case string(users.RoleTeacher):
		query = query.Joins("JOIN course_teachers ct ON ct.course_id = courses.id AND ct.teacher_id = ?", userID)
	case string(users.RoleStudent):
		query = query.Joins("JOIN course_students cs ON cs.course_id = courses.id AND cs.student_id = ?", userID)
	}
	var total int64
	if err := query.Session(&gorm.Session{}).
		Select("COUNT(DISTINCT courses.id)").
		Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	var result []Course
	err := query.Distinct("courses.*").Preload("AcademicYear").Preload("ClassGroup").Preload("Subject").
		Order("courses.name ASC").Offset(page.Offset()).Limit(page.PerPage).Find(&result).Error
	return result, total, err
}

func (r *Repository) ReplaceTeachers(ctx context.Context, courseID string, userIDs []string) error {
	return r.replaceMembers(ctx, "course_teachers", "teacher_id", string(users.RoleTeacher), courseID, userIDs)
}

func (r *Repository) ReplaceStudents(ctx context.Context, courseID string, userIDs []string) error {
	return r.replaceMembers(ctx, "course_students", "student_id", string(users.RoleStudent), courseID, userIDs)
}

func (r *Repository) CourseMembers(ctx context.Context, courseID string) ([]users.User, []users.User, error) {
	var teachers []users.User
	if err := r.db.WithContext(ctx).Table("users u").Select("u.*").Joins("JOIN course_teachers ct ON ct.teacher_id = u.id").Where("ct.course_id = ?", courseID).Order("u.full_name ASC").Scan(&teachers).Error; err != nil {
		return nil, nil, err
	}
	var students []users.User
	if err := r.db.WithContext(ctx).Table("users u").Select("u.*").Joins("JOIN course_students cs ON cs.student_id = u.id").Where("cs.course_id = ?", courseID).Order("u.full_name ASC").Scan(&students).Error; err != nil {
		return nil, nil, err
	}
	return teachers, students, nil
}

func (r *Repository) replaceMembers(ctx context.Context, table, userColumn, role, courseID string, userIDs []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&users.User{}).Where("id IN ? AND role = ? AND status = ?", userIDs, role, users.StatusActive).Count(&count).Error; err != nil {
			return err
		}
		if count != int64(len(userIDs)) {
			return ErrNotFound
		}
		var membership any
		switch table {
		case "course_teachers":
			membership = &CourseTeacher{}
		case "course_students":
			membership = &CourseStudent{}
		default:
			return errors.New("unsupported course membership table")
		}
		if err := tx.Where("course_id = ?", courseID).Delete(membership).Error; err != nil {
			return err
		}
		rows := make([]map[string]any, len(userIDs))
		for index, userID := range userIDs {
			rows[index] = map[string]any{"course_id": courseID, userColumn: userID}
		}
		return tx.Table(table).Create(&rows).Error
	})
}

func (r *Repository) IsTeacherAssigned(ctx context.Context, courseID, teacherID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("course_teachers").Where("course_id = ? AND teacher_id = ?", courseID, teacherID).Count(&count).Error
	return count > 0, err
}

func (r *Repository) IsStudentEnrolled(ctx context.Context, courseID, studentID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("course_students").Where("course_id = ? AND student_id = ?", courseID, studentID).Count(&count).Error
	return count > 0, err
}
