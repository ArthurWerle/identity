package model

import (
	"time"

	"gorm.io/datatypes"
)

// AuditLog represents an audit trail entry for auth and feature flag actions
type AuditLog struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	ActorUserID *uint          `gorm:"index" json:"actor_user_id,omitempty"`
	Action      string         `gorm:"type:text;not null" json:"action"`
	TargetType  string         `gorm:"type:text" json:"target_type,omitempty"`
	TargetID    string         `gorm:"type:text" json:"target_id,omitempty"`
	Details     datatypes.JSON `gorm:"type:jsonb" json:"details,omitempty"`
	IP          string         `gorm:"type:text" json:"ip,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`

	// Relationship
	Actor *User `gorm:"foreignKey:ActorUserID" json:"actor,omitempty"`
}

// TableName specifies the table name for the AuditLog model
func (AuditLog) TableName() string {
	return "audit_logs"
}
