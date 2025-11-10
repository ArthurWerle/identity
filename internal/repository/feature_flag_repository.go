package repository

import (
	"context"
	"identity/internal/model"

	"gorm.io/gorm"
)

// FeatureFlagRepository defines the interface for feature flag data operations
type FeatureFlagRepository interface {
	Create(ctx context.Context, flag *model.FeatureFlag) error
	GetByID(ctx context.Context, id uint) (*model.FeatureFlag, error)
	GetByKey(ctx context.Context, key string) (*model.FeatureFlag, error)
	GetAll(ctx context.Context, limit, offset int) ([]model.FeatureFlag, int64, error)
	Update(ctx context.Context, flag *model.FeatureFlag) error
	Delete(ctx context.Context, id uint) error
}

// featureFlagRepository implements FeatureFlagRepository
type featureFlagRepository struct {
	db *gorm.DB
}

// NewFeatureFlagRepository creates a new feature flag repository
func NewFeatureFlagRepository(db *gorm.DB) FeatureFlagRepository {
	return &featureFlagRepository{db: db}
}

// Create creates a new feature flag
func (r *featureFlagRepository) Create(ctx context.Context, flag *model.FeatureFlag) error {
	return r.db.WithContext(ctx).Create(flag).Error
}

// GetByID retrieves a feature flag by ID
func (r *featureFlagRepository) GetByID(ctx context.Context, id uint) (*model.FeatureFlag, error) {
	var flag model.FeatureFlag
	err := r.db.WithContext(ctx).First(&flag, id).Error
	if err != nil {
		return nil, err
	}
	return &flag, nil
}

// GetByKey retrieves a feature flag by key
func (r *featureFlagRepository) GetByKey(ctx context.Context, key string) (*model.FeatureFlag, error) {
	var flag model.FeatureFlag
	err := r.db.WithContext(ctx).
		Where("key = ?", key).
		First(&flag).Error
	if err != nil {
		return nil, err
	}
	return &flag, nil
}

// GetAll retrieves all feature flags with pagination
func (r *featureFlagRepository) GetAll(ctx context.Context, limit, offset int) ([]model.FeatureFlag, int64, error) {
	var flags []model.FeatureFlag
	var total int64

	// Count total records
	if err := r.db.WithContext(ctx).Model(&model.FeatureFlag{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated records
	err := r.db.WithContext(ctx).
		Limit(limit).
		Offset(offset).
		Find(&flags).Error

	return flags, total, err
}

// Update updates a feature flag
func (r *featureFlagRepository) Update(ctx context.Context, flag *model.FeatureFlag) error {
	return r.db.WithContext(ctx).Save(flag).Error
}

// Delete soft deletes a feature flag
func (r *featureFlagRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.FeatureFlag{}, id).Error
}
