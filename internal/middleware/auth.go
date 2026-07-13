package middleware

import (
	"identity/internal/model"
	"identity/internal/service"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	// SessionCookieName is the name of the session cookie
	SessionCookieName = "session_id"
	// SessionHeaderName carries the session for service-to-service calls
	SessionHeaderName = "X-Session-ID"
	// UserContextKey is the key used to store the user in the context
	UserContextKey = "user"
)

// SessionIDFromRequest extracts the session ID from the cookie, falling back
// to the X-Session-ID header (used by other services calling on behalf of a
// logged-in user).
func SessionIDFromRequest(c *gin.Context) string {
	if sessionID, err := c.Cookie(SessionCookieName); err == nil && sessionID != "" {
		return sessionID
	}
	return c.GetHeader(SessionHeaderName)
}

// ClearSessionCookie instructs the browser to drop the session cookie. Without
// this, a dead session_id keeps getting replayed on every request (each one
// 401s) and the client is stuck until cookies are cleared manually.
func ClearSessionCookie(c *gin.Context, secure bool) {
	c.SetCookie(SessionCookieName, "", -1, "/", "", secure, true)
}

// clearSessionCookieIfPresent clears the cookie only when the request actually
// carried one, so service-to-service X-Session-ID calls aren't handed a
// pointless Set-Cookie.
func clearSessionCookieIfPresent(c *gin.Context, secure bool) {
	if cookie, err := c.Cookie(SessionCookieName); err == nil && cookie != "" {
		ClearSessionCookie(c, secure)
	}
}

// Auth creates a middleware that validates the session and sets the user in context
func Auth(authService service.AuthService, logger *slog.Logger, cookieSecure bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := SessionIDFromRequest(c)
		if sessionID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Authentication required",
			})
			c.Abort()
			return
		}

		user, err := authService.ValidateSession(c.Request.Context(), sessionID)
		if err != nil {
			logger.Debug("session validation failed", "error", err)
			// The session is dead; if it arrived as a cookie, tell the browser
			// to drop it so it stops replaying a request that will only ever 401.
			clearSessionCookieIfPresent(c, cookieSecure)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid or expired session",
			})
			c.Abort()
			return
		}

		// Set user in context (and as audit actor on the request context)
		c.Set(UserContextKey, user)
		c.Request = c.Request.WithContext(service.WithActor(c.Request.Context(), user.ID))
		c.Next()
	}
}

// OptionalAuth creates a middleware that optionally validates the session
// If valid, sets the user in context; if not, continues without user
func OptionalAuth(authService service.AuthService, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := SessionIDFromRequest(c)
		if sessionID == "" {
			c.Next()
			return
		}

		user, err := authService.ValidateSession(c.Request.Context(), sessionID)
		if err != nil {
			logger.Debug("optional session validation failed", "error", err)
			c.Next()
			return
		}

		// Set user in context (and as audit actor on the request context)
		c.Set(UserContextKey, user)
		c.Request = c.Request.WithContext(service.WithActor(c.Request.Context(), user.ID))
		c.Next()
	}
}

// WebAuth creates a middleware for web routes that redirects to login on auth failure
func WebAuth(authService service.AuthService, logger *slog.Logger, cookieSecure bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie(SessionCookieName)
		if err != nil {
			c.Redirect(http.StatusFound, "/admin/login")
			c.Abort()
			return
		}

		user, err := authService.ValidateSession(c.Request.Context(), sessionID)
		if err != nil {
			logger.Debug("web session validation failed", "error", err)
			// Dead cookie: clear it so the admin UI isn't stuck redirecting
			// against a session_id the server will never accept.
			ClearSessionCookie(c, cookieSecure)
			c.Redirect(http.StatusFound, "/admin/login")
			c.Abort()
			return
		}

		// Set user in context (and as audit actor on the request context)
		c.Set(UserContextKey, user)
		c.Request = c.Request.WithContext(service.WithActor(c.Request.Context(), user.ID))
		c.Next()
	}
}

// GetUserFromContext retrieves the user from the gin context
func GetUserFromContext(c *gin.Context) *model.User {
	if user, exists := c.Get(UserContextKey); exists {
		if u, ok := user.(*model.User); ok {
			return u
		}
	}
	return nil
}
