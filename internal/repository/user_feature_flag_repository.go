package repository

import (
	"context"
	"identity/internal/model"

	"gorm.io/gorm"
)

// UserFeatureFlagRepository defines the interface for user-feature flag assignment operations
type UserFeatureFlagRepository interface {
	AssignFeatureFlagToUser(ctx context.Context, userID uint, featureFlagID uint) error
	UnassignFeatureFlagFromUser(ctx context.Context, userID uint, featureFlagID uint) error
	GetUserFeatureFlags(ctx context.Context, userID uint) ([]model.FeatureFlag, error)
	GetFeatureFlagUsers(ctx context.Context, featureFlagID uint) ([]model.User, error)
	IsFeatureFlagAssignedToUser(ctx context.Context, userID uint, featureFlagID uint) (bool, error)
}

// userFeatureFlagRepository implements UserFeatureFlagRepository
type userFeatureFlagRepository struct {
	db *gorm.DB
}

// NewUserFeatureFlagRepository creates a new user feature flag repository
func NewUserFeatureFlagRepository(db *gorm.DB) UserFeatureFlagRepository {
	return &userFeatureFlagRepository{db: db}
}

// AssignFeatureFlagToUser assigns a feature flag to a user
func (r *userFeatureFlagRepository) AssignFeatureFlagToUser(ctx context.Context, userID uint, featureFlagID uint) error {
	assignment := &model.UserFeatureFlag{
		UserID:        userID,
		FeatureFlagID: featureFlagID,
	}
	return r.db.WithContext(ctx).Create(assignment).Error
}

// UnassignFeatureFlagFromUser removes a feature flag from a user
func (r *userFeatureFlagRepository) UnassignFeatureFlagFromUser(ctx context.Context, userID uint, featureFlagID uint) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND feature_flag_id = ?", userID, featureFlagID).
		Delete(&model.UserFeatureFlag{}).Error
}

// GetUserFeatureFlags retrieves all feature flags for a user
func (r *userFeatureFlagRepository) GetUserFeatureFlags(ctx context.Context, userID uint) ([]model.FeatureFlag, error) {
	var flags []model.FeatureFlag
	err := r.db.WithContext(ctx).
		Joins("JOIN user_feature_flags ON user_feature_flags.feature_flag_id = feature_flags.id").
		Where("user_feature_flags.user_id = ?", userID).
		Find(&flags).Error
	return flags, err
}

// GetFeatureFlagUsers retrieves all users assigned to a feature flag
func (r *userFeatureFlagRepository) GetFeatureFlagUsers(ctx context.Context, featureFlagID uint) ([]model.User, error) {
	var users []model.User
	err := r.db.WithContext(ctx).
		Joins("JOIN user_feature_flags ON user_feature_flags.user_id = users.id").
		Where("user_feature_flags.feature_flag_id = ?", featureFlagID).
		Find(&users).Error
	return users, err
}

// IsFeatureFlagAssignedToUser checks if a feature flag is assigned to a user
func (r *userFeatureFlagRepository) IsFeatureFlagAssignedToUser(ctx context.Context, userID uint, featureFlagID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.UserFeatureFlag{}).
		Where("user_id = ? AND feature_flag_id = ?", userID, featureFlagID).
		Count(&count).Error
	return count > 0, err
}
