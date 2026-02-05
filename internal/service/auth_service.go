package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"identity/internal/model"
	"identity/internal/repository"
	"identity/internal/service/dto"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	// SessionDuration is the default session duration
	SessionDuration = 24 * time.Hour
	// BcryptCost is the cost factor for bcrypt hashing
	BcryptCost = 10
)

// AuthService defines the interface for authentication business logic
type AuthService interface {
	Login(ctx context.Context, req *dto.LoginRequest) (*dto.LoginResponse, error)
	Logout(ctx context.Context, sessionID string) error
	Register(ctx context.Context, req *dto.RegisterRequest) (*dto.UserResponse, error)
	ValidateSession(ctx context.Context, sessionID string) (*model.User, error)
	GetUserBySession(ctx context.Context, sessionID string) (*dto.UserResponse, error)
	SetPassword(ctx context.Context, userID uint, password string) error
}

// authService implements AuthService
type authService struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
}

// NewAuthService creates a new auth service
func NewAuthService(
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
) AuthService {
	return &authService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	}
}

// Login authenticates a user and creates a session
func (s *authService) Login(ctx context.Context, req *dto.LoginRequest) (*dto.LoginResponse, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid email or password")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user is enabled
	if !user.Enabled {
		return nil, errors.New("user account is disabled")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Generate session ID
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	// Create session
	session := &model.Session{
		ID:        sessionID,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(SessionDuration),
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Update last login
	now := time.Now()
	user.LastLogin = &now
	if err := s.userRepo.Update(ctx, user); err != nil {
		// Log but don't fail
	}

	return &dto.LoginResponse{
		User: dto.UserResponse{
			ID:        user.ID,
			Name:      user.Name,
			Email:     user.Email,
			Enabled:   user.Enabled,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			LastLogin: user.LastLogin,
		},
		SessionID: sessionID,
		Message:   "Login successful",
	}, nil
}

// Logout invalidates a session
func (s *authService) Logout(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return errors.New("session ID is required")
	}

	if err := s.sessionRepo.Delete(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// Register creates a new user with a password
func (s *authService) Register(ctx context.Context, req *dto.RegisterRequest) (*dto.UserResponse, error) {
	// Check if email already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, errors.New("email already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), BcryptCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &model.User{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Enabled:      true,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &dto.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Enabled:   user.Enabled,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}

// ValidateSession checks if a session is valid and returns the associated user
func (s *authService) ValidateSession(ctx context.Context, sessionID string) (*model.User, error) {
	if sessionID == "" {
		return nil, errors.New("session ID is required")
	}

	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid session")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Check if session is expired
	if session.IsExpired() {
		// Delete expired session
		_ = s.sessionRepo.Delete(ctx, sessionID)
		return nil, errors.New("session expired")
	}

	// Check if user is still enabled
	if !session.User.Enabled {
		return nil, errors.New("user account is disabled")
	}

	return &session.User, nil
}

// GetUserBySession retrieves user info by session ID
func (s *authService) GetUserBySession(ctx context.Context, sessionID string) (*dto.UserResponse, error) {
	user, err := s.ValidateSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	return &dto.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Enabled:   user.Enabled,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		LastLogin: user.LastLogin,
	}, nil
}

// SetPassword sets a new password for a user
func (s *authService) SetPassword(ctx context.Context, userID uint, password string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.PasswordHash = string(hashedPassword)
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// generateSessionID generates a random session ID
func generateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
