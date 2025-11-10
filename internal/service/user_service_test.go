package service

import (
	"context"
	"errors"
	"identity/internal/model"
	"identity/internal/service/dto"
	"testing"
	"time"

	"gorm.io/gorm"
)

// Mock repositories
type mockUserRepository struct {
	users map[uint]*model.User
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users: make(map[uint]*model.User),
	}
}

func (m *mockUserRepository) Create(ctx context.Context, user *model.User) error {
	user.ID = uint(len(m.users) + 1)
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepository) GetByID(ctx context.Context, id uint) (*model.User, error) {
	user, exists := m.users[id]
	if !exists {
		return nil, gorm.ErrRecordNotFound
	}
	return user, nil
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockUserRepository) GetAll(ctx context.Context, limit, offset int) ([]model.User, int64, error) {
	users := make([]model.User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, *user)
	}
	return users, int64(len(users)), nil
}

func (m *mockUserRepository) Update(ctx context.Context, user *model.User) error {
	if _, exists := m.users[user.ID]; !exists {
		return gorm.ErrRecordNotFound
	}
	user.UpdatedAt = time.Now()
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepository) Delete(ctx context.Context, id uint) error {
	if _, exists := m.users[id]; !exists {
		return gorm.ErrRecordNotFound
	}
	delete(m.users, id)
	return nil
}

type mockFeatureFlagRepository struct {
	flags map[uint]*model.FeatureFlag
}

func newMockFeatureFlagRepository() *mockFeatureFlagRepository {
	return &mockFeatureFlagRepository{
		flags: make(map[uint]*model.FeatureFlag),
	}
}

func (m *mockFeatureFlagRepository) Create(ctx context.Context, flag *model.FeatureFlag) error {
	flag.ID = uint(len(m.flags) + 1)
	m.flags[flag.ID] = flag
	return nil
}

func (m *mockFeatureFlagRepository) GetByID(ctx context.Context, id uint) (*model.FeatureFlag, error) {
	flag, exists := m.flags[id]
	if !exists {
		return nil, gorm.ErrRecordNotFound
	}
	return flag, nil
}

func (m *mockFeatureFlagRepository) GetByKey(ctx context.Context, key string) (*model.FeatureFlag, error) {
	for _, flag := range m.flags {
		if flag.Key == key {
			return flag, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockFeatureFlagRepository) GetAll(ctx context.Context, limit, offset int) ([]model.FeatureFlag, int64, error) {
	flags := make([]model.FeatureFlag, 0, len(m.flags))
	for _, flag := range m.flags {
		flags = append(flags, *flag)
	}
	return flags, int64(len(flags)), nil
}

func (m *mockFeatureFlagRepository) Update(ctx context.Context, flag *model.FeatureFlag) error {
	if _, exists := m.flags[flag.ID]; !exists {
		return gorm.ErrRecordNotFound
	}
	m.flags[flag.ID] = flag
	return nil
}

func (m *mockFeatureFlagRepository) Delete(ctx context.Context, id uint) error {
	if _, exists := m.flags[id]; !exists {
		return gorm.ErrRecordNotFound
	}
	delete(m.flags, id)
	return nil
}

type mockUserFeatureFlagRepository struct {
	assignments map[string]bool
}

func newMockUserFeatureFlagRepository() *mockUserFeatureFlagRepository {
	return &mockUserFeatureFlagRepository{
		assignments: make(map[string]bool),
	}
}

func (m *mockUserFeatureFlagRepository) AssignFeatureFlagToUser(ctx context.Context, userID uint, featureFlagID uint) error {
	key := m.key(userID, featureFlagID)
	m.assignments[key] = true
	return nil
}

func (m *mockUserFeatureFlagRepository) UnassignFeatureFlagFromUser(ctx context.Context, userID uint, featureFlagID uint) error {
	key := m.key(userID, featureFlagID)
	delete(m.assignments, key)
	return nil
}

func (m *mockUserFeatureFlagRepository) GetUserFeatureFlags(ctx context.Context, userID uint) ([]model.FeatureFlag, error) {
	return []model.FeatureFlag{}, nil
}

func (m *mockUserFeatureFlagRepository) GetFeatureFlagUsers(ctx context.Context, featureFlagID uint) ([]model.User, error) {
	return []model.User{}, nil
}

func (m *mockUserFeatureFlagRepository) IsFeatureFlagAssignedToUser(ctx context.Context, userID uint, featureFlagID uint) (bool, error) {
	key := m.key(userID, featureFlagID)
	return m.assignments[key], nil
}

func (m *mockUserFeatureFlagRepository) key(userID, featureFlagID uint) string {
	return string(rune(userID)) + "-" + string(rune(featureFlagID))
}

// Tests
func TestUserService_CreateUser(t *testing.T) {
	userRepo := newMockUserRepository()
	featureFlagRepo := newMockFeatureFlagRepository()
	userFFRepo := newMockUserFeatureFlagRepository()
	svc := NewUserService(userRepo, featureFlagRepo, userFFRepo)

	tests := []struct {
		name    string
		req     *dto.CreateUserRequest
		wantErr bool
	}{
		{
			name: "valid user",
			req: &dto.CreateUserRequest{
				Name:    "John Doe",
				Email:   "john@example.com",
				Enabled: true,
			},
			wantErr: false,
		},
		{
			name: "duplicate email",
			req: &dto.CreateUserRequest{
				Name:    "John Doe",
				Email:   "john@example.com",
				Enabled: true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := svc.CreateUser(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && user == nil {
				t.Error("CreateUser() returned nil user")
			}
		})
	}
}

func TestUserService_GetUser(t *testing.T) {
	userRepo := newMockUserRepository()
	featureFlagRepo := newMockFeatureFlagRepository()
	userFFRepo := newMockUserFeatureFlagRepository()
	svc := NewUserService(userRepo, featureFlagRepo, userFFRepo)

	// Create a test user
	createReq := &dto.CreateUserRequest{
		Name:    "John Doe",
		Email:   "john@example.com",
		Enabled: true,
	}
	created, _ := svc.CreateUser(context.Background(), createReq)

	tests := []struct {
		name    string
		id      uint
		wantErr bool
	}{
		{
			name:    "existing user",
			id:      created.ID,
			wantErr: false,
		},
		{
			name:    "non-existing user",
			id:      999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := svc.GetUser(context.Background(), tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && user == nil {
				t.Error("GetUser() returned nil user")
			}
		})
	}
}

func TestUserService_UpdateUser(t *testing.T) {
	userRepo := newMockUserRepository()
	featureFlagRepo := newMockFeatureFlagRepository()
	userFFRepo := newMockUserFeatureFlagRepository()
	svc := NewUserService(userRepo, featureFlagRepo, userFFRepo)

	// Create a test user
	createReq := &dto.CreateUserRequest{
		Name:    "John Doe",
		Email:   "john@example.com",
		Enabled: true,
	}
	created, _ := svc.CreateUser(context.Background(), createReq)

	newName := "Jane Doe"
	updateReq := &dto.UpdateUserRequest{
		Name: &newName,
	}

	updated, err := svc.UpdateUser(context.Background(), created.ID, updateReq)
	if err != nil {
		t.Errorf("UpdateUser() error = %v", err)
		return
	}

	if updated.Name != newName {
		t.Errorf("UpdateUser() name = %v, want %v", updated.Name, newName)
	}
}

func TestUserService_DeleteUser(t *testing.T) {
	userRepo := newMockUserRepository()
	featureFlagRepo := newMockFeatureFlagRepository()
	userFFRepo := newMockUserFeatureFlagRepository()
	svc := NewUserService(userRepo, featureFlagRepo, userFFRepo)

	// Create a test user
	createReq := &dto.CreateUserRequest{
		Name:    "John Doe",
		Email:   "john@example.com",
		Enabled: true,
	}
	created, _ := svc.CreateUser(context.Background(), createReq)

	// Delete user
	err := svc.DeleteUser(context.Background(), created.ID)
	if err != nil {
		t.Errorf("DeleteUser() error = %v", err)
		return
	}

	// Verify user is deleted
	_, err = svc.GetUser(context.Background(), created.ID)
	if err == nil {
		t.Error("DeleteUser() user still exists")
	}
}
