package service

import (
	"context"
	"encoding/json"
	"fmt"
	"identity/internal/model"
	"identity/internal/repository"
	"log/slog"
)

// Audit action names
const (
	AuditLoginSuccess     = "login_success"
	AuditLoginFailed      = "login_failed"
	AuditLogout           = "logout"
	AuditForceLogout      = "force_logout"
	AuditUserRegistered   = "user_registered"
	AuditPasswordSet      = "password_set"
	AuditUserCreated      = "user_created"
	AuditUserUpdated      = "user_updated"
	AuditUserDeleted      = "user_deleted"
	AuditFlagCreated      = "flag_created"
	AuditFlagUpdated      = "flag_updated"
	AuditFlagToggled      = "flag_toggled"
	AuditFlagDeleted      = "flag_deleted"
	AuditUserFlagAssigned = "user_flag_assigned"
	AuditUserFlagRemoved  = "user_flag_removed"
)

type actorContextKey struct{}

// WithActor stores the acting user's ID in the context so audit entries can
// attribute actions without threading the actor through every service call.
func WithActor(ctx context.Context, userID uint) context.Context {
	return context.WithValue(ctx, actorContextKey{}, userID)
}

// ActorFromContext returns the acting user's ID from the context, if set
func ActorFromContext(ctx context.Context) *uint {
	if id, ok := ctx.Value(actorContextKey{}).(uint); ok {
		return &id
	}
	return nil
}

// AuditLogger records audit events. Writes are best-effort: failures are
// logged and swallowed so an audit problem never blocks the main action.
// When actorUserID is nil, the actor is resolved from the context (set by the
// auth middleware via WithActor).
type AuditLogger interface {
	Log(ctx context.Context, actorUserID *uint, action, targetType, targetID string, details map[string]any)
}

// auditLogger implements AuditLogger
type auditLogger struct {
	repo   repository.AuditLogRepository
	logger *slog.Logger
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(repo repository.AuditLogRepository, logger *slog.Logger) AuditLogger {
	return &auditLogger{repo: repo, logger: logger}
}

// Log writes an audit entry and emits a structured log line
func (a *auditLogger) Log(ctx context.Context, actorUserID *uint, action, targetType, targetID string, details map[string]any) {
	if actorUserID == nil {
		actorUserID = ActorFromContext(ctx)
	}

	entry := &model.AuditLog{
		ActorUserID: actorUserID,
		Action:      action,
		TargetType:  targetType,
		TargetID:    targetID,
	}

	if details != nil {
		if data, err := json.Marshal(details); err == nil {
			entry.Details = data
		}
	}

	a.logger.Info("audit",
		"action", action,
		"actor_user_id", actorUserID,
		"target_type", targetType,
		"target_id", targetID,
		"details", fmt.Sprintf("%v", details),
	)

	if err := a.repo.Create(ctx, entry); err != nil {
		a.logger.Error("failed to write audit log", "action", action, "error", err)
	}
}
