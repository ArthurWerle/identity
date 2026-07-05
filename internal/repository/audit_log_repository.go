package repository

import (
	"context"
	"identity/internal/model"

	"gorm.io/gorm"
)

// AuditLogRepository defines the interface for audit log data operations
type AuditLogRepository interface {
	Create(ctx context.Context, log *model.AuditLog) error
	GetAll(ctx context.Context, limit, offset int) ([]model.AuditLog, int64, error)
}

// auditLogRepository implements AuditLogRepository
type auditLogRepository struct {
	db *gorm.DB
}

// NewAuditLogRepository creates a new audit log repository
func NewAuditLogRepository(db *gorm.DB) AuditLogRepository {
	return &auditLogRepository{db: db}
}

// Create creates a new audit log entry
func (r *auditLogRepository) Create(ctx context.Context, log *model.AuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// GetAll retrieves audit log entries ordered by most recent first
func (r *auditLogRepository) GetAll(ctx context.Context, limit, offset int) ([]model.AuditLog, int64, error) {
	var logs []model.AuditLog
	var total int64

	if err := r.db.WithContext(ctx).Model(&model.AuditLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.WithContext(ctx).
		Preload("Actor").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error

	return logs, total, err
}
