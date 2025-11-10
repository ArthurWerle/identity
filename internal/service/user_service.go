package service

import (
	"context"
	"errors"
	"fmt"
	"identity/internal/model"
	"identity/internal/repository"
	"identity/internal/service/dto"
	"time"

	"gorm.io/gorm"
)

// UserService defines the interface for user business logic
type UserService interface {
	CreateUser(ctx context.Context, req *dto.CreateUserRequest) (*dto.UserResponse, error)
	GetUser(ctx context.Context, id uint) (*dto.UserResponse, error)
	GetUsers(ctx context.Context, pagination *dto.PaginationParams) (*dto.UserListResponse, error)
	UpdateUser(ctx context.Context, id uint, req *dto.UpdateUserRequest) (*dto.UserResponse, error)
	DeleteUser(ctx context.Context, id uint) error
	GetUserFeatureFlags(ctx context.Context, userID uint) ([]dto.FeatureFlagResponse, error)
	AssignFeatureFlagToUser(ctx context.Context, userID uint, featureFlagKey string) error
	UnassignFeatureFlagFromUser(ctx context.Context, userID uint, featureFlagKey string) error
}

// userService implements UserService
type userService struct {
	userRepo        repository.UserRepository
	featureFlagRepo repository.FeatureFlagRepository
	userFFRepo      repository.UserFeatureFlagRepository
}

// NewUserService creates a new user service
func NewUserService(
	userRepo repository.UserRepository,
	featureFlagRepo repository.FeatureFlagRepository,
	userFFRepo repository.UserFeatureFlagRepository,
) UserService {
	return &userService{
		userRepo:        userRepo,
		featureFlagRepo: featureFlagRepo,
		userFFRepo:      userFFRepo,
	}
}

// CreateUser creates a new user
func (s *userService) CreateUser(ctx context.Context, req *dto.CreateUserRequest) (*dto.UserResponse, error) {
	// Validate email uniqueness
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, errors.New("email already exists")
	}

	user := &model.User{
		Name:    req.Name,
		Email:   req.Email,
		Enabled: req.Enabled,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return s.toUserResponse(user), nil
}

// GetUser retrieves a user by ID
func (s *userService) GetUser(ctx context.Context, id uint) (*dto.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return s.toUserResponse(user), nil
}

// GetUsers retrieves all users with pagination
func (s *userService) GetUsers(ctx context.Context, pagination *dto.PaginationParams) (*dto.UserListResponse, error) {
	users, total, err := s.userRepo.GetAll(ctx, pagination.GetLimit(), pagination.GetOffset())
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	userResponses := make([]dto.UserResponse, len(users))
	for i, user := range users {
		userResponses[i] = *s.toUserResponse(&user)
	}

	return &dto.UserListResponse{
		Users:      userResponses,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: dto.CalculateTotalPages(total, pagination.PageSize),
	}, nil
}

// UpdateUser updates a user
func (s *userService) UpdateUser(ctx context.Context, id uint, req *dto.UpdateUserRequest) (*dto.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Update fields if provided
	if req.Name != nil {
		user.Name = *req.Name
	}
	if req.Email != nil {
		// Check if email is already taken by another user
		existingUser, err := s.userRepo.GetByEmail(ctx, *req.Email)
		if err == nil && existingUser != nil && existingUser.ID != id {
			return nil, errors.New("email already exists")
		}
		user.Email = *req.Email
	}
	if req.Enabled != nil {
		user.Enabled = *req.Enabled
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return s.toUserResponse(user), nil
}

// DeleteUser deletes a user
func (s *userService) DeleteUser(ctx context.Context, id uint) error {
	// Check if user exists
	_, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	if err := s.userRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// GetUserFeatureFlags retrieves all feature flags for a user
func (s *userService) GetUserFeatureFlags(ctx context.Context, userID uint) ([]dto.FeatureFlagResponse, error) {
	// Check if user exists
	_, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	flags, err := s.userFFRepo.GetUserFeatureFlags(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user feature flags: %w", err)
	}

	responses := make([]dto.FeatureFlagResponse, len(flags))
	for i, flag := range flags {
		responses[i] = dto.FeatureFlagResponse{
			ID:          flag.ID,
			Key:         flag.Key,
			Description: flag.Description,
			Enabled:     flag.Enabled,
			CreatedAt:   flag.CreatedAt,
			UpdatedAt:   flag.UpdatedAt,
		}
	}

	return responses, nil
}

// AssignFeatureFlagToUser assigns a feature flag to a user
func (s *userService) AssignFeatureFlagToUser(ctx context.Context, userID uint, featureFlagKey string) error {
	// Check if user exists
	_, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Get feature flag by key
	flag, err := s.featureFlagRepo.GetByKey(ctx, featureFlagKey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("feature flag not found")
		}
		return fmt.Errorf("failed to get feature flag: %w", err)
	}

	// Check if already assigned
	isAssigned, err := s.userFFRepo.IsFeatureFlagAssignedToUser(ctx, userID, flag.ID)
	if err != nil {
		return fmt.Errorf("failed to check assignment: %w", err)
	}
	if isAssigned {
		return errors.New("feature flag already assigned to user")
	}

	// Assign
	if err := s.userFFRepo.AssignFeatureFlagToUser(ctx, userID, flag.ID); err != nil {
		return fmt.Errorf("failed to assign feature flag: %w", err)
	}

	return nil
}

// UnassignFeatureFlagFromUser removes a feature flag from a user
func (s *userService) UnassignFeatureFlagFromUser(ctx context.Context, userID uint, featureFlagKey string) error {
	// Check if user exists
	_, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Get feature flag by key
	flag, err := s.featureFlagRepo.GetByKey(ctx, featureFlagKey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("feature flag not found")
		}
		return fmt.Errorf("failed to get feature flag: %w", err)
	}

	// Unassign
	if err := s.userFFRepo.UnassignFeatureFlagFromUser(ctx, userID, flag.ID); err != nil {
		return fmt.Errorf("failed to unassign feature flag: %w", err)
	}

	return nil
}

// toUserResponse converts a model.User to dto.UserResponse
func (s *userService) toUserResponse(user *model.User) *dto.UserResponse {
	var lastLogin *time.Time
	if user.LastLogin != nil {
		lastLogin = user.LastLogin
	}

	return &dto.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Enabled:   user.Enabled,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		LastLogin: lastLogin,
	}
}
