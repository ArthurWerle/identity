package model

import (
	"time"

	"gorm.io/gorm"
)

// Session represents a user login session
type Session struct {
	ID        string         `gorm:"primaryKey;type:varchar(64)" json:"id"`
	UserID    uint           `gorm:"index;not null" json:"user_id"`
	ExpiresAt time.Time      `gorm:"not null" json:"expires_at"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relationship
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName specifies the table name for the Session model
func (Session) TableName() string {
	return "sessions"
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}
