package service

import (
	"context"
	"errors"
	"fmt"
	"identity/internal/model"
	"identity/internal/repository"
	"identity/internal/service/dto"

	"gorm.io/gorm"
)

// FeatureFlagService defines the interface for feature flag business logic
type FeatureFlagService interface {
	CreateFeatureFlag(ctx context.Context, req *dto.CreateFeatureFlagRequest) (*dto.FeatureFlagResponse, error)
	GetFeatureFlag(ctx context.Context, id uint) (*dto.FeatureFlagResponse, error)
	GetFeatureFlagByKey(ctx context.Context, key string) (*dto.FeatureFlagResponse, error)
	GetFeatureFlags(ctx context.Context, pagination *dto.PaginationParams) (*dto.FeatureFlagListResponse, error)
	UpdateFeatureFlag(ctx context.Context, id uint, req *dto.UpdateFeatureFlagRequest) (*dto.FeatureFlagResponse, error)
	DeleteFeatureFlag(ctx context.Context, id uint) error
}

// featureFlagService implements FeatureFlagService
type featureFlagService struct {
	featureFlagRepo repository.FeatureFlagRepository
}

// NewFeatureFlagService creates a new feature flag service
func NewFeatureFlagService(featureFlagRepo repository.FeatureFlagRepository) FeatureFlagService {
	return &featureFlagService{
		featureFlagRepo: featureFlagRepo,
	}
}

// CreateFeatureFlag creates a new feature flag
func (s *featureFlagService) CreateFeatureFlag(ctx context.Context, req *dto.CreateFeatureFlagRequest) (*dto.FeatureFlagResponse, error) {
	// Validate key uniqueness
	existingFlag, err := s.featureFlagRepo.GetByKey(ctx, req.Key)
	if err == nil && existingFlag != nil {
		return nil, errors.New("feature flag key already exists")
	}

	flag := &model.FeatureFlag{
		Key:         req.Key,
		Description: req.Description,
		Enabled:     req.Enabled,
	}

	if err := s.featureFlagRepo.Create(ctx, flag); err != nil {
		return nil, fmt.Errorf("failed to create feature flag: %w", err)
	}

	return s.toFeatureFlagResponse(flag), nil
}

// GetFeatureFlag retrieves a feature flag by ID
func (s *featureFlagService) GetFeatureFlag(ctx context.Context, id uint) (*dto.FeatureFlagResponse, error) {
	flag, err := s.featureFlagRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("feature flag not found")
		}
		return nil, fmt.Errorf("failed to get feature flag: %w", err)
	}

	return s.toFeatureFlagResponse(flag), nil
}

// GetFeatureFlagByKey retrieves a feature flag by key
func (s *featureFlagService) GetFeatureFlagByKey(ctx context.Context, key string) (*dto.FeatureFlagResponse, error) {
	flag, err := s.featureFlagRepo.GetByKey(ctx, key)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("feature flag not found")
		}
		return nil, fmt.Errorf("failed to get feature flag: %w", err)
	}

	return s.toFeatureFlagResponse(flag), nil
}

// GetFeatureFlags retrieves all feature flags with pagination
func (s *featureFlagService) GetFeatureFlags(ctx context.Context, pagination *dto.PaginationParams) (*dto.FeatureFlagListResponse, error) {
	flags, total, err := s.featureFlagRepo.GetAll(ctx, pagination.GetLimit(), pagination.GetOffset())
	if err != nil {
		return nil, fmt.Errorf("failed to get feature flags: %w", err)
	}

	flagResponses := make([]dto.FeatureFlagResponse, len(flags))
	for i, flag := range flags {
		flagResponses[i] = *s.toFeatureFlagResponse(&flag)
	}

	return &dto.FeatureFlagListResponse{
		FeatureFlags: flagResponses,
		Total:        total,
		Page:         pagination.Page,
		PageSize:     pagination.PageSize,
		TotalPages:   dto.CalculateTotalPages(total, pagination.PageSize),
	}, nil
}

// UpdateFeatureFlag updates a feature flag
func (s *featureFlagService) UpdateFeatureFlag(ctx context.Context, id uint, req *dto.UpdateFeatureFlagRequest) (*dto.FeatureFlagResponse, error) {
	flag, err := s.featureFlagRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("feature flag not found")
		}
		return nil, fmt.Errorf("failed to get feature flag: %w", err)
	}

	// Update fields if provided
	if req.Description != nil {
		flag.Description = *req.Description
	}
	if req.Enabled != nil {
		flag.Enabled = *req.Enabled
	}

	if err := s.featureFlagRepo.Update(ctx, flag); err != nil {
		return nil, fmt.Errorf("failed to update feature flag: %w", err)
	}

	return s.toFeatureFlagResponse(flag), nil
}

// DeleteFeatureFlag deletes a feature flag
func (s *featureFlagService) DeleteFeatureFlag(ctx context.Context, id uint) error {
	// Check if feature flag exists
	_, err := s.featureFlagRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("feature flag not found")
		}
		return fmt.Errorf("failed to get feature flag: %w", err)
	}

	if err := s.featureFlagRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete feature flag: %w", err)
	}

	return nil
}

// toFeatureFlagResponse converts a model.FeatureFlag to dto.FeatureFlagResponse
func (s *featureFlagService) toFeatureFlagResponse(flag *model.FeatureFlag) *dto.FeatureFlagResponse {
	return &dto.FeatureFlagResponse{
		ID:          flag.ID,
		Key:         flag.Key,
		Description: flag.Description,
		Enabled:     flag.Enabled,
		CreatedAt:   flag.CreatedAt,
		UpdatedAt:   flag.UpdatedAt,
	}
}
