package handler

import (
	"identity/internal/service"
	"identity/internal/service/dto"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	// SessionCookieName is the name of the session cookie
	SessionCookieName = "session_id"
	// SessionHeaderName carries the session for service-to-service calls
	SessionHeaderName = "X-Session-ID"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	authService  service.AuthService
	logger       *slog.Logger
	cookieMaxAge int
	cookieSecure bool
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService service.AuthService, logger *slog.Logger, cookieSecure bool) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		logger:       logger,
		cookieMaxAge: int(authService.SessionDuration().Seconds()),
		cookieSecure: cookieSecure,
	}
}

// sessionIDFromRequest extracts the session ID from the cookie, falling back
// to the X-Session-ID header.
func sessionIDFromRequest(c *gin.Context) string {
	if sessionID, err := c.Cookie(SessionCookieName); err == nil && sessionID != "" {
		return sessionID
	}
	return c.GetHeader(SessionHeaderName)
}

// Login godoc
// @Summary Login user
// @Description Authenticate a user and create a session
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Login credentials"
// @Success 200 {object} dto.LoginResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBind(&req); err != nil {
		h.logger.Error("failed to bind login request", "error", err)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("login failed", "error", err, "email", req.Email)
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "login_failed",
			Message: err.Error(),
		})
		return
	}

	// Set session cookie
	c.SetCookie(
		SessionCookieName,
		resp.SessionID,
		h.cookieMaxAge,
		"/",
		"",
		h.cookieSecure,
		true, // HttpOnly
	)

	h.logger.Info("user logged in", "user_id", resp.User.ID, "email", resp.User.Email)
	c.JSON(http.StatusOK, resp)
}

// Logout godoc
// @Summary Logout user
// @Description Invalidate the current session
// @Tags auth
// @Produce json
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	sessionID := sessionIDFromRequest(c)
	if sessionID == "" {
		// No session, nothing to logout
		c.JSON(http.StatusOK, dto.SuccessResponse{
			Message: "Logged out successfully",
		})
		return
	}

	if err := h.authService.Logout(c.Request.Context(), sessionID); err != nil {
		h.logger.Error("logout failed", "error", err)
		// Still clear the cookie even if backend fails
	}

	// Clear session cookie
	c.SetCookie(
		SessionCookieName,
		"",
		-1,
		"/",
		"",
		h.cookieSecure,
		true,
	)

	h.logger.Info("user logged out", "session_id", sessionID)
	c.JSON(http.StatusOK, dto.SuccessResponse{
		Message: "Logged out successfully",
	})
}

// ValidateSession godoc
// @Summary Validate a session
// @Description Validate a session ID and return user info. Accepts session_id in JSON body or X-Session-ID header.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.ValidateSessionRequest false "Session ID"
// @Success 200 {object} dto.UserResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Router /api/v1/auth/validate [post]
func (h *AuthHandler) ValidateSession(c *gin.Context) {
	var sessionID string

	var req dto.ValidateSessionRequest
	if err := c.ShouldBindJSON(&req); err == nil && req.SessionID != "" {
		sessionID = req.SessionID
	}

	if sessionID == "" {
		sessionID = c.GetHeader("X-Session-ID")
	}

	if sessionID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: "session_id is required (provide in JSON body or X-Session-ID header)",
		})
		return
	}

	resp, err := h.authService.GetUserBySession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "unauthorized",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Me godoc
// @Summary Get current user
// @Description Get the currently authenticated user's information
// @Tags auth
// @Produce json
// @Success 200 {object} dto.UserResponse
// @Failure 401 {object} dto.ErrorResponse
// @Router /api/v1/auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	sessionID := sessionIDFromRequest(c)
	if sessionID == "" {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "unauthorized",
			Message: "Not authenticated",
		})
		return
	}

	resp, err := h.authService.GetUserBySession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "unauthorized",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}
