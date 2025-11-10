package model

import (
	"time"
)

// UserFeatureFlag represents the many-to-many relationship between users and feature flags
type UserFeatureFlag struct {
	UserID        uint      `gorm:"primaryKey" json:"user_id"`
	FeatureFlagID uint      `gorm:"primaryKey" json:"feature_flag_id"`
	CreatedAt     time.Time `json:"created_at"`

	// Foreign key relationships
	User        User        `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	FeatureFlag FeatureFlag `gorm:"foreignKey:FeatureFlagID;constraint:OnDelete:CASCADE" json:"feature_flag,omitempty"`
}

// TableName specifies the table name for the UserFeatureFlag model
func (UserFeatureFlag) TableName() string {
	return "user_feature_flags"
}
