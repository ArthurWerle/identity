package handler

import (
	"embed"
	"html/template"
	"identity/internal/middleware"
	"identity/internal/model"
	"identity/internal/repository"
	"identity/internal/service"
	"identity/internal/service/dto"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed templates/*.html
var templateFS embed.FS

// WebHandler handles web interface requests
type WebHandler struct {
	authService        service.AuthService
	userService        service.UserService
	featureFlagService service.FeatureFlagService
	auditLogRepo       repository.AuditLogRepository
	logger             *slog.Logger
	templates          *template.Template
	cookieSecure       bool
	environment        string
}

// NewWebHandler creates a new web handler
func NewWebHandler(
	authService service.AuthService,
	userService service.UserService,
	featureFlagService service.FeatureFlagService,
	auditLogRepo repository.AuditLogRepository,
	logger *slog.Logger,
	cookieSecure bool,
	environment string,
) *WebHandler {
	tmpl := template.Must(template.ParseFS(templateFS, "templates/*.html"))

	return &WebHandler{
		authService:        authService,
		userService:        userService,
		featureFlagService: featureFlagService,
		auditLogRepo:       auditLogRepo,
		logger:             logger,
		templates:          tmpl,
		cookieSecure:       cookieSecure,
		environment:        environment,
	}
}

// PageData contains common data for all pages
type PageData struct {
	Title        string
	Environment  string
	User         *model.User
	Error        string
	Success      string
	ActiveTab    string
	Flags        []FlagWithUserCount
	Users        []UserWithFlagCount
	SelectedUser *model.User
	AllFlags     []FlagWithAssignment
	AuditLogs    []AuditRow
}

// AuditRow is a template-friendly audit log entry
type AuditRow struct {
	CreatedAt string
	Action    string
	Actor     string
	Target    string
	Details   string
}

// FlagWithUserCount represents a feature flag with user count
type FlagWithUserCount struct {
	ID          uint
	Key         string
	Description string
	Enabled     bool
	UserCount   int
}

// UserWithFlagCount represents a user with flag count
type UserWithFlagCount struct {
	ID        uint
	Name      string
	Email     string
	Enabled   bool
	FlagCount int
}

// FlagWithAssignment represents a flag with assignment status
type FlagWithAssignment struct {
	ID          uint
	Key         string
	Description string
	Enabled     bool
	IsAssigned  bool
}

// LoginPage renders the login page
func (h *WebHandler) LoginPage(c *gin.Context) {
	data := PageData{
		Title: "Login",
	}
	h.renderTemplate(c, "layout.html", "login.html", data)
}

// LoginSubmit handles login form submission
func (h *WebHandler) LoginSubmit(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBind(&req); err != nil {
		data := PageData{
			Title: "Login",
			Error: "Please enter valid email and password",
		}
		h.renderTemplate(c, "layout.html", "login.html", data)
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("web login failed", "error", err, "email", req.Email)
		data := PageData{
			Title: "Login",
			Error: err.Error(),
		}
		h.renderTemplate(c, "layout.html", "login.html", data)
		return
	}

	// Set session cookie
	c.SetCookie(
		SessionCookieName,
		resp.SessionID,
		int(h.authService.SessionDuration().Seconds()),
		"/",
		"",
		h.cookieSecure,
		true,
	)

	c.Redirect(http.StatusFound, "/admin")
}

// Logout handles logout
func (h *WebHandler) Logout(c *gin.Context) {
	sessionID, _ := c.Cookie(SessionCookieName)
	if sessionID != "" {
		_ = h.authService.Logout(c.Request.Context(), sessionID)
	}

	c.SetCookie(SessionCookieName, "", -1, "/", "", h.cookieSecure, true)
	c.Redirect(http.StatusFound, "/admin/login")
}

// Dashboard renders the main dashboard
func (h *WebHandler) Dashboard(c *gin.Context) {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/admin/login")
		return
	}

	data := PageData{
		Title:     "Dashboard",
		User:      user,
		ActiveTab: "flags",
	}

	// Load flags
	data.Flags = h.loadFlags(c)

	h.renderTemplate(c, "layout.html", "dashboard.html", data)
}

// FlagsTab renders the flags tab content
func (h *WebHandler) FlagsTab(c *gin.Context) {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		c.Status(http.StatusUnauthorized)
		return
	}

	data := PageData{
		Title:     "Feature Flags",
		User:      user,
		ActiveTab: "flags",
		Flags:     h.loadFlags(c),
	}

	// Check if this is an HTMX request
	if c.GetHeader("HX-Request") == "true" {
		h.templates.ExecuteTemplate(c.Writer, "flags-content", data)
		return
	}

	h.renderTemplate(c, "layout.html", "dashboard.html", data)
}

// UsersTab renders the users tab content
func (h *WebHandler) UsersTab(c *gin.Context) {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		c.Status(http.StatusUnauthorized)
		return
	}

	data := PageData{
		Title:     "Users",
		User:      user,
		ActiveTab: "users",
		Users:     h.loadUsers(c),
	}

	// Check if this is an HTMX request
	if c.GetHeader("HX-Request") == "true" {
		h.templates.ExecuteTemplate(c.Writer, "users-content", data)
		return
	}

	h.renderTemplate(c, "layout.html", "dashboard.html", data)
}

// CreateFlag creates a new feature flag
func (h *WebHandler) CreateFlag(c *gin.Context) {
	key := c.PostForm("key")
	description := c.PostForm("description")

	if key == "" {
		c.String(http.StatusBadRequest, "Key is required")
		return
	}

	_, err := h.featureFlagService.CreateFeatureFlag(c.Request.Context(), &dto.CreateFeatureFlagRequest{
		Key:         key,
		Description: description,
		Enabled:     false,
	})

	if err != nil {
		h.logger.Error("failed to create flag", "error", err)
	}

	// Return updated flags table
	data := PageData{
		Flags: h.loadFlags(c),
	}
	h.templates.ExecuteTemplate(c.Writer, "flags-table", data)
}

// ToggleFlag toggles a feature flag's global status
func (h *WebHandler) ToggleFlag(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid flag ID")
		return
	}

	// Get current flag
	flag, err := h.featureFlagService.GetFeatureFlag(c.Request.Context(), uint(id))
	if err != nil {
		c.String(http.StatusNotFound, "Flag not found")
		return
	}

	// Toggle enabled status
	newEnabled := !flag.Enabled
	_, err = h.featureFlagService.UpdateFeatureFlag(c.Request.Context(), uint(id), &dto.UpdateFeatureFlagRequest{
		Enabled: &newEnabled,
	})

	if err != nil {
		h.logger.Error("failed to toggle flag", "error", err)
		c.String(http.StatusInternalServerError, "Failed to update flag")
		return
	}

	// Return updated row
	flags := h.loadFlags(c)
	for _, f := range flags {
		if f.ID == uint(id) {
			h.templates.ExecuteTemplate(c.Writer, "flag-row", f)
			return
		}
	}
}

// DeleteFlag deletes a feature flag
func (h *WebHandler) DeleteFlag(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid flag ID")
		return
	}

	err = h.featureFlagService.DeleteFeatureFlag(c.Request.Context(), uint(id))
	if err != nil {
		h.logger.Error("failed to delete flag", "error", err)
	}

	// Return empty string to remove the row
	c.String(http.StatusOK, "")
}

// UserFlags shows flags for a specific user
func (h *WebHandler) UserFlags(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Get user
	userResp, err := h.userService.GetUser(c.Request.Context(), uint(id))
	if err != nil {
		c.String(http.StatusNotFound, "User not found")
		return
	}

	// Get user's assigned flags
	userFlags, err := h.userService.GetUserFeatureFlags(c.Request.Context(), uint(id))
	if err != nil {
		h.logger.Error("failed to get user flags", "error", err)
	}

	userFlagKeys := make(map[string]bool)
	for _, f := range userFlags {
		userFlagKeys[f.Key] = true
	}

	// Get all flags
	pagination := &dto.PaginationParams{Page: 1, PageSize: 100}
	flagsResp, _ := h.featureFlagService.GetFeatureFlags(c.Request.Context(), pagination)

	allFlags := make([]FlagWithAssignment, 0)
	for _, f := range flagsResp.FeatureFlags {
		allFlags = append(allFlags, FlagWithAssignment{
			ID:          f.ID,
			Key:         f.Key,
			Description: f.Description,
			Enabled:     f.Enabled,
			IsAssigned:  userFlagKeys[f.Key],
		})
	}

	data := PageData{
		SelectedUser: &model.User{
			ID:    userResp.ID,
			Name:  userResp.Name,
			Email: userResp.Email,
		},
		AllFlags: allFlags,
	}

	h.templates.ExecuteTemplate(c.Writer, "user-flags-modal", data)
}

// ToggleUserFlag toggles a flag assignment for a user
func (h *WebHandler) ToggleUserFlag(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid user ID")
		return
	}

	flagKey := c.Param("key")

	// Check if flag is assigned
	userFlags, err := h.userService.GetUserFeatureFlags(c.Request.Context(), uint(userID))
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to get user flags")
		return
	}

	isAssigned := false
	for _, f := range userFlags {
		if f.Key == flagKey {
			isAssigned = true
			break
		}
	}

	if isAssigned {
		err = h.userService.UnassignFeatureFlagFromUser(c.Request.Context(), uint(userID), flagKey)
	} else {
		err = h.userService.AssignFeatureFlagToUser(c.Request.Context(), uint(userID), flagKey)
	}

	if err != nil {
		h.logger.Error("failed to toggle user flag", "error", err)
	}

	// Re-render the modal
	h.UserFlags(c)
}

// CreateUser creates a new user from the admin UI
func (h *WebHandler) CreateUser(c *gin.Context) {
	name := c.PostForm("name")
	email := c.PostForm("email")
	password := c.PostForm("password")

	if name == "" || email == "" || password == "" {
		c.String(http.StatusBadRequest, "Name, email and password are required")
		return
	}

	_, err := h.authService.Register(c.Request.Context(), &dto.RegisterRequest{
		Name:     name,
		Email:    email,
		Password: password,
	})
	if err != nil {
		h.logger.Error("failed to create user", "error", err)
	}

	h.renderUsersList(c)
}

// EditUserModal renders the edit form for a user
func (h *WebHandler) EditUserModal(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid user ID")
		return
	}

	userResp, err := h.userService.GetUser(c.Request.Context(), uint(id))
	if err != nil {
		c.String(http.StatusNotFound, "User not found")
		return
	}

	data := PageData{
		SelectedUser: &model.User{
			ID:      userResp.ID,
			Name:    userResp.Name,
			Email:   userResp.Email,
			Enabled: userResp.Enabled,
		},
	}
	h.templates.ExecuteTemplate(c.Writer, "user-edit-modal", data)
}

// UpdateUser updates a user from the admin UI (name/email/enabled, optional new password)
func (h *WebHandler) UpdateUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid user ID")
		return
	}

	name := c.PostForm("name")
	email := c.PostForm("email")
	enabled := c.PostForm("enabled") == "true"

	_, err = h.userService.UpdateUser(c.Request.Context(), uint(id), &dto.UpdateUserRequest{
		Name:    &name,
		Email:   &email,
		Enabled: &enabled,
	})
	if err != nil {
		h.logger.Error("failed to update user", "error", err)
	}

	if password := c.PostForm("password"); password != "" {
		if err := h.authService.SetPassword(c.Request.Context(), uint(id), password); err != nil {
			h.logger.Error("failed to set password", "error", err)
		}
	}

	h.renderUsersList(c)
}

// DeleteUser deletes a user from the admin UI
func (h *WebHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid user ID")
		return
	}

	if current := middleware.GetUserFromContext(c); current != nil && current.ID == uint(id) {
		c.String(http.StatusBadRequest, "You cannot delete your own account")
		return
	}

	if err := h.userService.DeleteUser(c.Request.Context(), uint(id)); err != nil {
		h.logger.Error("failed to delete user", "error", err)
	}

	// Also kill any active sessions for the deleted user
	if err := h.authService.ForceLogout(c.Request.Context(), nil, uint(id)); err != nil {
		h.logger.Error("failed to clear sessions of deleted user", "error", err)
	}

	h.renderUsersList(c)
}

// ForceLogoutUser deletes all sessions for a user ("log people out" button)
func (h *WebHandler) ForceLogoutUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid user ID")
		return
	}

	if err := h.authService.ForceLogout(c.Request.Context(), nil, uint(id)); err != nil {
		h.logger.Error("failed to force logout user", "error", err)
	}

	h.renderUsersList(c)
}

// AuditTab renders the audit log tab
func (h *WebHandler) AuditTab(c *gin.Context) {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		c.Status(http.StatusUnauthorized)
		return
	}

	data := PageData{
		Title:     "Audit Log",
		User:      user,
		ActiveTab: "audit",
		AuditLogs: h.loadAuditLogs(c),
	}

	if c.GetHeader("HX-Request") == "true" {
		h.templates.ExecuteTemplate(c.Writer, "audit-content", data)
		return
	}

	h.renderTemplate(c, "layout.html", "dashboard.html", data)
}

// Helper methods

func (h *WebHandler) renderUsersList(c *gin.Context) {
	data := PageData{
		Users: h.loadUsers(c),
	}
	h.templates.ExecuteTemplate(c.Writer, "users-list", data)
}

func (h *WebHandler) loadAuditLogs(c *gin.Context) []AuditRow {
	logs, _, err := h.auditLogRepo.GetAll(c.Request.Context(), 100, 0)
	if err != nil {
		h.logger.Error("failed to load audit logs", "error", err)
		return nil
	}

	rows := make([]AuditRow, 0, len(logs))
	for _, entry := range logs {
		actor := "-"
		if entry.Actor != nil && entry.Actor.Name != "" {
			actor = entry.Actor.Name
		} else if entry.ActorUserID != nil {
			actor = "user #" + strconv.FormatUint(uint64(*entry.ActorUserID), 10)
		}

		target := entry.TargetType
		if entry.TargetID != "" {
			target += " " + entry.TargetID
		}

		rows = append(rows, AuditRow{
			CreatedAt: entry.CreatedAt.Format(time.RFC3339),
			Action:    entry.Action,
			Actor:     actor,
			Target:    target,
			Details:   string(entry.Details),
		})
	}

	return rows
}

func (h *WebHandler) loadFlags(c *gin.Context) []FlagWithUserCount {
	pagination := &dto.PaginationParams{Page: 1, PageSize: 100}
	flagsResp, err := h.featureFlagService.GetFeatureFlags(c.Request.Context(), pagination)
	if err != nil {
		h.logger.Error("failed to load flags", "error", err)
		return nil
	}

	flags := make([]FlagWithUserCount, 0, len(flagsResp.FeatureFlags))
	for _, f := range flagsResp.FeatureFlags {
		flags = append(flags, FlagWithUserCount{
			ID:          f.ID,
			Key:         f.Key,
			Description: f.Description,
			Enabled:     f.Enabled,
			UserCount:   0, // TODO: implement user count
		})
	}

	return flags
}

func (h *WebHandler) loadUsers(c *gin.Context) []UserWithFlagCount {
	pagination := &dto.PaginationParams{Page: 1, PageSize: 100}
	usersResp, err := h.userService.GetUsers(c.Request.Context(), pagination)
	if err != nil {
		h.logger.Error("failed to load users", "error", err)
		return nil
	}

	users := make([]UserWithFlagCount, 0, len(usersResp.Users))
	for _, u := range usersResp.Users {
		flagCount := 0
		flags, err := h.userService.GetUserFeatureFlags(c.Request.Context(), u.ID)
		if err == nil {
			flagCount = len(flags)
		}

		users = append(users, UserWithFlagCount{
			ID:        u.ID,
			Name:      u.Name,
			Email:     u.Email,
			Enabled:   u.Enabled,
			FlagCount: flagCount,
		})
	}

	return users
}

func (h *WebHandler) renderTemplate(c *gin.Context, layout, content string, data PageData) {
	c.Header("Content-Type", "text/html; charset=utf-8")

	// Environment label shown in the admin header on every full-page render
	data.Environment = h.environment

	// Parse templates fresh each time for development
	// In production, you might want to cache this
	tmpl := template.Must(template.ParseFS(templateFS, "templates/"+layout, "templates/"+content))

	if err := tmpl.ExecuteTemplate(c.Writer, "layout.html", data); err != nil {
		h.logger.Error("failed to render template", "error", err)
		c.String(http.StatusInternalServerError, "Internal server error")
	}
}
