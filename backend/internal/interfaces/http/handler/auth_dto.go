package handler

import (
	"time"

	"github.com/google/uuid"
)

// =====================
// Auth Request DTOs
// =====================

// LoginRequest represents the request body for user login
type LoginRequest struct {
	Username string `json:"username" binding:"required,min=3,max=100"`
	Password string `json:"password" binding:"required,min=8,max=128"`
}

// RefreshTokenRequest represents the request body for token refresh
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// ChangePasswordRequest represents the request body for password change
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=128"`
}

// =====================
// Auth Response DTOs
// =====================

// TokenResponse represents the token data in auth responses
type TokenResponse struct {
	AccessToken           string    `json:"access_token"`
	RefreshToken          string    `json:"refresh_token"`
	AccessTokenExpiresAt  time.Time `json:"access_token_expires_at"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
	TokenType             string    `json:"token_type"`
}

// AuthUserResponse represents user data in auth responses
type AuthUserResponse struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email,omitempty"`
	Phone       string    `json:"phone,omitempty"`
	Avatar      string    `json:"avatar,omitempty"`
	Permissions []string  `json:"permissions"`
	RoleIDs     []string  `json:"role_ids"`
}

// LoginResponse represents the response body for successful login
type LoginResponse struct {
	Token TokenResponse    `json:"token"`
	User  AuthUserResponse `json:"user"`
}

// RefreshTokenResponse represents the response body for successful token refresh
type RefreshTokenResponse struct {
	Token TokenResponse `json:"token"`
}

// CurrentUserResponse represents the response body for current user info
type CurrentUserResponse struct {
	User        AuthUserResponse `json:"user"`
	Permissions []string         `json:"permissions"`
}

// LogoutResponse represents the response body for logout
type LogoutResponse struct {
	Message string `json:"message"`
}
