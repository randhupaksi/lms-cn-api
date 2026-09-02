package users

import (
	"context"
	"errors"
	"strings"

	"lms-cn-api/pkg/pagination"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("user not found")

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Create(user).Error
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

func (r *Repository) List(ctx context.Context, page pagination.Request, search, role string) ([]User, int64, error) {
	query := r.db.WithContext(ctx).Model(&User{})
	if search = strings.TrimSpace(search); search != "" {
		like := "%" + search + "%"
		query = query.Where("identifier LIKE ? OR full_name LIKE ?", like, like)
	}
	if role != "" {
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

func (r *Repository) Update(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *Repository) RevokeSessions(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Table("auth_sessions").Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", gorm.Expr("CURRENT_TIMESTAMP(6)")).Error
}

func normalizeIdentifier(value string) string { return strings.ToLower(strings.TrimSpace(value)) }
