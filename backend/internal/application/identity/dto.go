package identity

import (
	"time"

	"github.com/google/uuid"
)

// LoginInput contains the input for user login
type LoginInput struct {
	Username string
	Password string
	IP       string // Client IP for login tracking
}

// LoginResult contains the result of a successful login
type LoginResult struct {
	AccessToken           string
	RefreshToken          string
	AccessTokenExpiresAt  time.Time
	RefreshTokenExpiresAt time.Time
	TokenType             string
	User                  UserInfo
}

// UserInfo contains basic user information returned after login
type UserInfo struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	Username    string
	DisplayName string
	Email       string
	Phone       string
	Avatar      string
	Permissions []string
	RoleIDs     []uuid.UUID
}

// RefreshTokenInput contains the input for token refresh
type RefreshTokenInput struct {
	RefreshToken string
	UserID       uuid.UUID // For permission reload
	TenantID     uuid.UUID
}

// RefreshTokenResult contains the result of a token refresh
type RefreshTokenResult struct {
	AccessToken           string
	RefreshToken          string
	AccessTokenExpiresAt  time.Time
	RefreshTokenExpiresAt time.Time
	TokenType             string
}

// LogoutInput contains the input for user logout
type LogoutInput struct {
	UserID   uuid.UUID
	TenantID uuid.UUID
	TokenJTI string // JWT ID for blacklisting (optional)
}

// ChangePasswordInput contains the input for password change
type ChangePasswordInput struct {
	UserID      uuid.UUID
	OldPassword string
	NewPassword string
}

// GetCurrentUserInput contains the input for getting current user info
type GetCurrentUserInput struct {
	UserID   uuid.UUID
	TenantID uuid.UUID
}

// CurrentUserResult contains the current user's information
type CurrentUserResult struct {
	User        UserInfo
	Permissions []string
}

// ForceLogoutInput contains the input for force logout operation
type ForceLogoutInput struct {
	AdminUserID  uuid.UUID // Admin performing the action
	TargetUserID uuid.UUID // User to force logout
	TenantID     uuid.UUID
	Reason       string // Reason for force logout (for audit)
}

// ForceLogoutResult contains the result of force logout operation
type ForceLogoutResult struct {
	Message string
}
