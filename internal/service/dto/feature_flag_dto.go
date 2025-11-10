package dto

import (
	"time"
)

// CreateFeatureFlagRequest represents the request to create a new feature flag
type CreateFeatureFlagRequest struct {
	Key         string `json:"key" binding:"required" example:"dark_mode"`
	Description string `json:"description" example:"Enable dark mode interface"`
	Enabled     bool   `json:"enabled" example:"true"`
}

// UpdateFeatureFlagRequest represents the request to update a feature flag
type UpdateFeatureFlagRequest struct {
	Description *string `json:"description,omitempty" example:"Enable dark mode interface"`
	Enabled     *bool   `json:"enabled,omitempty" example:"true"`
}

// FeatureFlagResponse represents the response for a feature flag
type FeatureFlagResponse struct {
	ID          uint      `json:"id" example:"1"`
	Key         string    `json:"key" example:"dark_mode"`
	Description string    `json:"description" example:"Enable dark mode interface"`
	Enabled     bool      `json:"enabled" example:"true"`
	CreatedAt   time.Time `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt   time.Time `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// FeatureFlagListResponse represents a paginated list of feature flags
type FeatureFlagListResponse struct {
	FeatureFlags []FeatureFlagResponse `json:"feature_flags"`
	Total        int64                 `json:"total" example:"50"`
	Page         int                   `json:"page" example:"1"`
	PageSize     int                   `json:"page_size" example:"10"`
	TotalPages   int                   `json:"total_pages" example:"5"`
}
