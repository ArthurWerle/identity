package model

import (
	"time"

	"gorm.io/gorm"
)

// FeatureFlag represents a feature flag in the system
type FeatureFlag struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Key         string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"key"`
	Description string         `gorm:"type:text" json:"description"`
	Enabled     bool           `gorm:"default:false;not null" json:"enabled"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Many-to-many relationship with Users
	Users []User `gorm:"many2many:user_feature_flags;" json:"users,omitempty"`
}

// TableName specifies the table name for the FeatureFlag model
func (FeatureFlag) TableName() string {
	return "feature_flags"
}
