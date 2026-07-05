package service

import (
	"context"
	"testing"
	"time"

	"identity/internal/model"
	"identity/internal/service/dto"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// noopAudit is an AuditLogger that does nothing, for tests
type noopAudit struct{}

func newNoopAudit() AuditLogger {
	return &noopAudit{}
}

func (a *noopAudit) Log(ctx context.Context, actorUserID *uint, action, targetType, targetID string, details map[string]any) {
}

// mockSessionRepository is an in-memory SessionRepository
type mockSessionRepository struct {
	sessions map[string]*model.Session
	users    *mockUserRepository
}

func newMockSessionRepository(users *mockUserRepository) *mockSessionRepository {
	return &mockSessionRepository{
		sessions: make(map[string]*model.Session),
		users:    users,
	}
}

func (m *mockSessionRepository) Create(ctx context.Context, session *model.Session) error {
	m.sessions[session.ID] = session
	return nil
}

func (m *mockSessionRepository) GetByID(ctx context.Context, id string) (*model.Session, error) {
	session, exists := m.sessions[id]
	if !exists {
		return nil, gorm.ErrRecordNotFound
	}
	if user, err := m.users.GetByID(ctx, session.UserID); err == nil {
		session.User = *user
	}
	return session, nil
}

func (m *mockSessionRepository) UpdateExpiresAt(ctx context.Context, id string, expiresAt time.Time) error {
	session, exists := m.sessions[id]
	if !exists {
		return gorm.ErrRecordNotFound
	}
	session.ExpiresAt = expiresAt
	return nil
}

func (m *mockSessionRepository) Delete(ctx context.Context, id string) error {
	delete(m.sessions, id)
	return nil
}

func (m *mockSessionRepository) DeleteByUserID(ctx context.Context, userID uint) error {
	for id, session := range m.sessions {
		if session.UserID == userID {
			delete(m.sessions, id)
		}
	}
	return nil
}

func (m *mockSessionRepository) DeleteExpired(ctx context.Context) error {
	for id, session := range m.sessions {
		if session.IsExpired() {
			delete(m.sessions, id)
		}
	}
	return nil
}

func setupAuthService(t *testing.T, sessionDuration time.Duration) (AuthService, *mockUserRepository, *mockSessionRepository) {
	t.Helper()
	userRepo := newMockUserRepository()
	sessionRepo := newMockSessionRepository(userRepo)
	svc := NewAuthService(userRepo, sessionRepo, newNoopAudit(), sessionDuration)

	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	if err := userRepo.Create(context.Background(), &model.User{
		Name:         "Test User",
		Email:        "test@example.com",
		PasswordHash: string(hash),
		Enabled:      true,
	}); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	return svc, userRepo, sessionRepo
}

func TestLoginAndValidate(t *testing.T) {
	svc, _, _ := setupAuthService(t, 720*time.Hour)

	resp, err := svc.Login(context.Background(), &dto.LoginRequest{
		Email:    "test@example.com",
		Password: "secret123",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if resp.SessionID == "" {
		t.Fatal("expected a session ID")
	}

	user, err := svc.ValidateSession(context.Background(), resp.SessionID)
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if user.Email != "test@example.com" {
		t.Errorf("expected test@example.com, got %s", user.Email)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	svc, _, _ := setupAuthService(t, 720*time.Hour)

	_, err := svc.Login(context.Background(), &dto.LoginRequest{
		Email:    "test@example.com",
		Password: "wrong",
	})
	if err == nil {
		t.Fatal("expected login to fail")
	}
}

func TestValidateSlidesExpiry(t *testing.T) {
	svc, _, sessionRepo := setupAuthService(t, 720*time.Hour)

	resp, err := svc.Login(context.Background(), &dto.LoginRequest{
		Email:    "test@example.com",
		Password: "secret123",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Simulate a session whose expiry has drifted well behind the full window
	staleExpiry := time.Now().Add(1 * time.Hour)
	sessionRepo.sessions[resp.SessionID].ExpiresAt = staleExpiry

	if _, err := svc.ValidateSession(context.Background(), resp.SessionID); err != nil {
		t.Fatalf("validate failed: %v", err)
	}

	newExpiry := sessionRepo.sessions[resp.SessionID].ExpiresAt
	if !newExpiry.After(staleExpiry.Add(24 * time.Hour)) {
		t.Errorf("expected expiry to slide forward, got %v (was %v)", newExpiry, staleExpiry)
	}
}

func TestValidateFreshSessionDoesNotWrite(t *testing.T) {
	svc, _, sessionRepo := setupAuthService(t, 720*time.Hour)

	resp, err := svc.Login(context.Background(), &dto.LoginRequest{
		Email:    "test@example.com",
		Password: "secret123",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// A just-created session is within the slide threshold: expiry unchanged
	before := sessionRepo.sessions[resp.SessionID].ExpiresAt
	if _, err := svc.ValidateSession(context.Background(), resp.SessionID); err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	after := sessionRepo.sessions[resp.SessionID].ExpiresAt
	if !after.Equal(before) {
		t.Errorf("expected expiry unchanged for fresh session, got %v (was %v)", after, before)
	}
}

func TestValidateExpiredSession(t *testing.T) {
	svc, _, sessionRepo := setupAuthService(t, 720*time.Hour)

	resp, err := svc.Login(context.Background(), &dto.LoginRequest{
		Email:    "test@example.com",
		Password: "secret123",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	sessionRepo.sessions[resp.SessionID].ExpiresAt = time.Now().Add(-time.Minute)

	if _, err := svc.ValidateSession(context.Background(), resp.SessionID); err == nil {
		t.Fatal("expected expired session to be rejected")
	}
	if _, exists := sessionRepo.sessions[resp.SessionID]; exists {
		t.Error("expected expired session to be deleted")
	}
}

func TestForceLogout(t *testing.T) {
	svc, _, _ := setupAuthService(t, 720*time.Hour)

	resp, err := svc.Login(context.Background(), &dto.LoginRequest{
		Email:    "test@example.com",
		Password: "secret123",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	if err := svc.ForceLogout(context.Background(), nil, 1); err != nil {
		t.Fatalf("force logout failed: %v", err)
	}

	if _, err := svc.ValidateSession(context.Background(), resp.SessionID); err == nil {
		t.Fatal("expected session to be invalid after force logout")
	}
}
