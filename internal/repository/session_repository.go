package repository

import (
	"context"
	"identity/internal/model"

	"gorm.io/gorm"
)

// SessionRepository defines the interface for session data operations
type SessionRepository interface {
	Create(ctx context.Context, session *model.Session) error
	GetByID(ctx context.Context, id string) (*model.Session, error)
	Delete(ctx context.Context, id string) error
	DeleteByUserID(ctx context.Context, userID uint) error
	DeleteExpired(ctx context.Context) error
}

// sessionRepository implements SessionRepository
type sessionRepository struct {
	db *gorm.DB
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(db *gorm.DB) SessionRepository {
	return &sessionRepository{db: db}
}

// Create creates a new session
func (r *sessionRepository) Create(ctx context.Context, session *model.Session) error {
	return r.db.WithContext(ctx).Create(session).Error
}

// GetByID retrieves a session by ID
func (r *sessionRepository) GetByID(ctx context.Context, id string) (*model.Session, error) {
	var session model.Session
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("id = ? AND deleted_at IS NULL", id).
		First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// Delete soft deletes a session
func (r *sessionRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.Session{}, "id = ?", id).Error
}

// DeleteByUserID deletes all sessions for a user
func (r *sessionRepository) DeleteByUserID(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&model.Session{}).Error
}

// DeleteExpired removes all expired sessions
func (r *sessionRepository) DeleteExpired(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Where("expires_at < NOW()").
		Delete(&model.Session{}).Error
}
