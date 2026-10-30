package users

import (
	"context"
	"errors"
	"lms-cn-api/internal/modules/audit"
	"strings"
	"time"

	"lms-cn-api/pkg/pagination"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrNotFound = errors.New("user not found")

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *Repository) CreateAudited(ctx context.Context, user *User, event audit.Record) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		return audit.NewService(tx).Record(ctx, event)
	})
}

func (r *Repository) FindByID(ctx context.Context, id string) (User, error) {
	var user User
	err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, ErrNotFound
	}
	return user, err
}

func (r *Repository) FindByIdentifier(ctx context.Context, identifier string) (User, error) {
	var user User
	err := r.db.WithContext(ctx).Where("identifier = ?", normalizeIdentifier(identifier)).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, ErrNotFound
	}
	return user, err
}

func (r *Repository) List(ctx context.Context, page pagination.Request, search, role string, actorRole Role) ([]User, int64, error) {
	query := r.db.WithContext(ctx).Model(&User{})
	switch actorRole {
	case RoleSuperadmin:
	case RolePrincipal:
		query = query.Where("role IN ?", []Role{RoleCurriculum, RoleAdmin, RoleTeacher, RoleStudent})
	default:
		query = query.Where("role IN ?", []Role{RoleTeacher, RoleStudent})
	}
	if search = strings.TrimSpace(search); search != "" {
		like := "%" + search + "%"
		query = query.Where("identifier LIKE ? OR full_name LIKE ?", like, like)
	}
	if role == "teaching_staff" {
		query = query.Where("role IN ?", []Role{RoleTeacher, RoleCurriculum})
	} else if role != "" {
		query = query.Where("role = ?", role)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var result []User
	err := query.Order("full_name ASC").Offset(page.Offset()).Limit(page.PerPage).Find(&result).Error
	return result, total, err
}

// Mutate serializes privileged-account changes and revokes sessions atomically.
func (r *Repository) Mutate(ctx context.Context, id string, revoke bool, apply func(*Repository, *User) error, event audit.Record) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var elevated []User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("role = ?", RoleSuperadmin).Order("id").Find(&elevated).Error; err != nil {
			return err
		}
		var user User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		local := NewRepository(tx)
		if err := apply(local, &user); err != nil {
			return err
		}
		if err := tx.Save(&user).Error; err != nil {
			return err
		}
		if revoke {
			if err := local.RevokeSessions(ctx, id); err != nil {
				return err
			}
		}
		return audit.NewService(tx).Record(ctx, event)
	})
}

func (r *Repository) HasAcademicRecords(ctx context.Context, user User) (bool, error) {
	var count int64
	if user.Role == RoleTeacher || user.Role == RoleCurriculum {
		err := r.db.WithContext(ctx).Raw(`SELECT (SELECT COUNT(*) FROM course_teachers WHERE teacher_id = ?) + (SELECT COUNT(*) FROM questions WHERE author_id = ?) + (SELECT COUNT(*) FROM exams WHERE author_id = ?) + (SELECT COUNT(*) FROM course_materials WHERE author_id = ?) + (SELECT COUNT(*) FROM assignments WHERE author_id = ?) + (SELECT COUNT(*) FROM assignment_submissions WHERE graded_by = ?) AS total`, user.ID, user.ID, user.ID, user.ID, user.ID, user.ID).Scan(&count).Error
		return count > 0, err
	}
	if user.Role == RoleStudent {
		err := r.db.WithContext(ctx).Raw(`SELECT (SELECT COUNT(*) FROM course_students WHERE student_id = ?) + (SELECT COUNT(*) FROM exam_participants WHERE student_id = ?) + (SELECT COUNT(*) FROM attempts WHERE student_id = ?) + (SELECT COUNT(*) FROM assignment_submissions WHERE student_id = ?) + (SELECT COUNT(*) FROM material_progress WHERE student_id = ?) AS total`, user.ID, user.ID, user.ID, user.ID, user.ID).Scan(&count).Error
		return count > 0, err
	}
	return false, nil
}

func (r *Repository) Sessions(ctx context.Context, id string) ([]SessionResponse, error) {
	rows := []SessionResponse{}
	err := r.db.WithContext(ctx).Table("auth_sessions").Select("id, created_at, last_used_at, expires_at").Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", id, time.Now().UTC()).Order("last_used_at DESC").Limit(100).Scan(&rows).Error
	return rows, err
}

func (r *Repository) OtherActiveSuperadmins(ctx context.Context, id string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&User{}).Where("role=? AND status=? AND id<>?", RoleSuperadmin, StatusActive, id).Count(&count).Error
	return count, err
}

func (r *Repository) Update(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *Repository) RevokeSessions(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Table("auth_sessions").Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", gorm.Expr("CURRENT_TIMESTAMP(6)")).Error
}

func normalizeIdentifier(value string) string { return strings.ToLower(strings.TrimSpace(value)) }
