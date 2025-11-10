package dto

import (
	"time"
)

// CreateUserRequest represents the request to create a new user
type CreateUserRequest struct {
	Name    string `json:"name" binding:"required" example:"John Doe"`
	Email   string `json:"email" binding:"required,email" example:"john@example.com"`
	Enabled bool   `json:"enabled" example:"true"`
}

// UpdateUserRequest represents the request to update a user
type UpdateUserRequest struct {
	Name    *string `json:"name,omitempty" example:"John Doe"`
	Email   *string `json:"email,omitempty" example:"john@example.com"`
	Enabled *bool   `json:"enabled,omitempty" example:"true"`
}

// UserResponse represents the response for a user
type UserResponse struct {
	ID        uint       `json:"id" example:"1"`
	Name      string     `json:"name" example:"John Doe"`
	Email     string     `json:"email" example:"john@example.com"`
	Enabled   bool       `json:"enabled" example:"true"`
	CreatedAt time.Time  `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt time.Time  `json:"updated_at" example:"2024-01-01T00:00:00Z"`
	LastLogin *time.Time `json:"last_login,omitempty" example:"2024-01-01T00:00:00Z"`
}

// UserListResponse represents a paginated list of users
type UserListResponse struct {
	Users      []UserResponse `json:"users"`
	Total      int64          `json:"total" example:"100"`
	Page       int            `json:"page" example:"1"`
	PageSize   int            `json:"page_size" example:"10"`
	TotalPages int            `json:"total_pages" example:"10"`
}
