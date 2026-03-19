package dto

// LoginRequest represents the request to login
type LoginRequest struct {
	Email    string `json:"email" form:"email" binding:"required,email" example:"john@example.com"`
	Password string `json:"password" form:"password" binding:"required" example:"password123"`
}

// LoginResponse represents the response after successful login
type LoginResponse struct {
	User      UserResponse `json:"user"`
	SessionID string       `json:"session_id"`
	Message   string       `json:"message"`
}

// RegisterRequest represents the request to register a new user
type RegisterRequest struct {
	Name     string `json:"name" form:"name" binding:"required" example:"John Doe"`
	Email    string `json:"email" form:"email" binding:"required,email" example:"john@example.com"`
	Password string `json:"password" form:"password" binding:"required,min=6" example:"password123"`
}

// ValidateSessionRequest represents the request to validate a session
type ValidateSessionRequest struct {
	SessionID string `json:"session_id" example:"abc123"`
}
