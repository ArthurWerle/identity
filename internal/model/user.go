package model

import (
	"time"

	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"type:varchar(255);not null" json:"name"`
	Email     string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Enabled   bool           `gorm:"default:true;not null" json:"enabled"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	LastLogin *time.Time     `json:"last_login,omitempty"`

	// Many-to-many relationship with FeatureFlags
	FeatureFlags []FeatureFlag `gorm:"many2many:user_feature_flags;" json:"feature_flags,omitempty"`
}

// TableName specifies the table name for the User model
func (User) TableName() string {
	return "users"
}
