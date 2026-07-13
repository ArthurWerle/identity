package middleware

import (
	"context"
	"errors"
	"identity/internal/model"
	"identity/internal/service/dto"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// stubAuthService implements service.AuthService for middleware tests. Only
// ValidateSession is exercised here; the rest satisfy the interface.
type stubAuthService struct {
	user        *model.User
	validateErr error
}

func (s *stubAuthService) Login(ctx context.Context, req *dto.LoginRequest) (*dto.LoginResponse, error) {
	return nil, nil
}
func (s *stubAuthService) Logout(ctx context.Context, sessionID string) error { return nil }
func (s *stubAuthService) Register(ctx context.Context, req *dto.RegisterRequest) (*dto.UserResponse, error) {
	return nil, nil
}
func (s *stubAuthService) ValidateSession(ctx context.Context, sessionID string) (*model.User, error) {
	return s.user, s.validateErr
}
func (s *stubAuthService) GetUserBySession(ctx context.Context, sessionID string) (*dto.UserResponse, error) {
	return nil, nil
}
func (s *stubAuthService) SetPassword(ctx context.Context, userID uint, password string) error {
	return nil
}
func (s *stubAuthService) ForceLogout(ctx context.Context, actorUserID *uint, userID uint) error {
	return nil
}
func (s *stubAuthService) SessionDuration() time.Duration { return time.Hour }

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// A dead session arriving as a cookie must produce a Set-Cookie that expires it,
// otherwise the browser replays it forever and the client 401-loops.
func TestAuthClearsCookieOnInvalidSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubAuthService{validateErr: errors.New("invalid session")}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "dead-session"})
	c.Request = req

	Auth(svc, discardLogger(), false)(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	setCookie := w.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, SessionCookieName+"=;") {
		t.Fatalf("expected Set-Cookie clearing %s, got %q", SessionCookieName, setCookie)
	}
	if !strings.Contains(setCookie, "Max-Age=0") {
		t.Fatalf("expected the cookie to be expired (Max-Age=0), got %q", setCookie)
	}
}

// A dead session arriving via the X-Session-ID header (service-to-service) has
// no browser cookie to clear, so no Set-Cookie should be emitted.
func TestAuthDoesNotClearCookieForHeaderSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubAuthService{validateErr: errors.New("invalid session")}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set(SessionHeaderName, "dead-session")
	c.Request = req

	Auth(svc, discardLogger(), false)(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	if sc := w.Header().Get("Set-Cookie"); sc != "" {
		t.Fatalf("expected no Set-Cookie for a header-only session, got %q", sc)
	}
}

// The admin web middleware must clear a dead cookie too, so the admin UI isn't
// stuck redirecting against a session_id the server will never accept.
func TestWebAuthClearsCookieOnInvalidSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubAuthService{validateErr: errors.New("invalid session")}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "dead-session"})
	c.Request = req

	WebAuth(svc, discardLogger(), false)(c)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect, got %d", w.Code)
	}
	if sc := w.Header().Get("Set-Cookie"); !strings.Contains(sc, "Max-Age=0") {
		t.Fatalf("expected the cookie to be expired, got %q", sc)
	}
}
