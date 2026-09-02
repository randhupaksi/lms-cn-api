package auth

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrSessionNotFound = errors.New("session not found")

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, session *Session) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *Repository) FindByID(ctx context.Context, id string) (Session, error) {
	var session Session
	err := r.db.WithContext(ctx).Preload("User").First(&session, "auth_sessions.id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Session{}, ErrSessionNotFound
	}
	return session, err
}

func (r *Repository) Rotate(ctx context.Context, currentHash, nextHash string, now time.Time) (Session, error) {
	var result Session
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var session Session
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("User").
			First(&session, "refresh_token_hash = ?", currentHash).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSessionNotFound
		}
		if err != nil {
			return err
		}
		if !session.IsValid(now) {
			return ErrSessionNotFound
		}
		if err := tx.Model(&session).Updates(map[string]any{
			"refresh_token_hash": nextHash,
			"last_used_at":       now,
		}).Error; err != nil {
			return err
		}
		session.RefreshTokenHash = nextHash
		session.LastUsedAt = now
		result = session
		return nil
	})
	return result, err
}

func (r *Repository) Revoke(ctx context.Context, sessionID string, now time.Time) error {
	return r.db.WithContext(ctx).Model(&Session{}).
		Where("id = ? AND revoked_at IS NULL", sessionID).Update("revoked_at", now).Error
}

func (r *Repository) RevokeAllForUser(ctx context.Context, userID string, now time.Time) error {
	return r.db.WithContext(ctx).Model(&Session{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).Update("revoked_at", now).Error
}
