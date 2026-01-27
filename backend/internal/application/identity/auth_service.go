package identity

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/auth"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AuthServiceConfig contains configuration for the auth service
type AuthServiceConfig struct {
	MaxLoginAttempts int           // Maximum failed login attempts before lock
	LockDuration     time.Duration // How long to lock account after max attempts
}

// DefaultAuthServiceConfig returns default configuration
func DefaultAuthServiceConfig() AuthServiceConfig {
	return AuthServiceConfig{
		MaxLoginAttempts: 5,
		LockDuration:     15 * time.Minute,
	}
}

// AuthService handles authentication operations
type AuthService struct {
	userRepo       identity.UserRepository
	roleRepo       identity.RoleRepository
	jwtService     *auth.JWTService
	tokenBlacklist auth.TokenBlacklist
	config         AuthServiceConfig
	logger         *zap.Logger
}

// NewAuthService creates a new authentication service
func NewAuthService(
	userRepo identity.UserRepository,
	roleRepo identity.RoleRepository,
	jwtService *auth.JWTService,
	config AuthServiceConfig,
	logger *zap.Logger,
) *AuthService {
	return &AuthService{
		userRepo:       userRepo,
		roleRepo:       roleRepo,
		jwtService:     jwtService,
		tokenBlacklist: nil, // Optional, set via SetTokenBlacklist
		config:         config,
		logger:         logger,
	}
}

// SetTokenBlacklist sets the token blacklist for the service
// This allows token revocation on logout and password change
func (s *AuthService) SetTokenBlacklist(blacklist auth.TokenBlacklist) {
	s.tokenBlacklist = blacklist
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(ctx context.Context, input LoginInput) (*LoginResult, error) {
	s.logger.Info("Login attempt", zap.String("username", input.Username))

	// Find user by username
	user, err := s.userRepo.FindByUsername(ctx, input.Username)
	if err != nil {
		s.logger.Warn("User not found during login", zap.String("username", input.Username))
		return nil, shared.NewDomainError("INVALID_CREDENTIALS", "Invalid username or password")
	}

	// Check if user can login
	if !user.CanLogin() {
		if user.IsLocked() {
			s.logger.Warn("Login attempt for locked account", zap.String("username", input.Username))
			return nil, shared.NewDomainError("ACCOUNT_LOCKED", "Account is locked. Please try again later or contact support")
		}
		if user.IsDeactivated() {
			s.logger.Warn("Login attempt for deactivated account", zap.String("username", input.Username))
			return nil, shared.NewDomainError("ACCOUNT_DEACTIVATED", "Account has been deactivated")
		}
		if user.IsPending() {
			s.logger.Warn("Login attempt for pending account", zap.String("username", input.Username))
			return nil, shared.NewDomainError("ACCOUNT_PENDING", "Account is pending activation")
		}
		return nil, shared.NewDomainError("ACCOUNT_INACTIVE", "Account is not active")
	}

	// Verify password
	if !user.VerifyPassword(input.Password) {
		// Record failed attempt
		locked := user.RecordLoginFailure(s.config.MaxLoginAttempts, s.config.LockDuration)
		if err := s.userRepo.Update(ctx, user); err != nil {
			s.logger.Error("Failed to update user after login failure", zap.Error(err))
		}

		if locked {
			s.logger.Warn("Account locked after too many failed attempts",
				zap.String("username", input.Username),
				zap.Int("attempts", s.config.MaxLoginAttempts))
			return nil, shared.NewDomainError("ACCOUNT_LOCKED", "Too many failed login attempts. Account has been locked")
		}

		s.logger.Warn("Invalid password attempt",
			zap.String("username", input.Username),
			zap.Int("failed_attempts", user.FailedAttempts))
		return nil, shared.NewDomainError("INVALID_CREDENTIALS", "Invalid username or password")
	}

	// Load user roles
	if err := s.userRepo.LoadUserRoles(ctx, user); err != nil {
		s.logger.Error("Failed to load user roles", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to load user roles")
	}

	// Collect all permissions from roles
	permissions, err := s.collectUserPermissions(ctx, user.RoleIDs)
	if err != nil {
		s.logger.Error("Failed to collect user permissions", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to load user permissions")
	}

	// Generate token pair
	tokenInput := auth.GenerateTokenInput{
		TenantID:    user.TenantID,
		UserID:      user.ID,
		Username:    user.Username,
		RoleIDs:     user.RoleIDs,
		Permissions: permissions,
	}

	tokenPair, err := s.jwtService.GenerateTokenPair(tokenInput)
	if err != nil {
		s.logger.Error("Failed to generate token pair", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to generate authentication tokens")
	}

	// Record successful login
	user.RecordLoginSuccess(input.IP)
	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error("Failed to update user after successful login", zap.Error(err))
		// Don't fail the login - just log the error
	}

	s.logger.Info("User logged in successfully",
		zap.String("username", input.Username),
		zap.String("user_id", user.ID.String()))

	return &LoginResult{
		AccessToken:           tokenPair.AccessToken,
		RefreshToken:          tokenPair.RefreshToken,
		AccessTokenExpiresAt:  tokenPair.AccessTokenExpiresAt,
		RefreshTokenExpiresAt: tokenPair.RefreshTokenExpiresAt,
		TokenType:             tokenPair.TokenType,
		User: UserInfo{
			ID:          user.ID,
			TenantID:    user.TenantID,
			Username:    user.Username,
			DisplayName: user.GetDisplayNameOrUsername(),
			Email:       user.Email,
			Phone:       user.Phone,
			Avatar:      user.Avatar,
			Permissions: permissions,
			RoleIDs:     user.RoleIDs,
		},
	}, nil
}

// RefreshToken refreshes the access token using a valid refresh token
func (s *AuthService) RefreshToken(ctx context.Context, input RefreshTokenInput) (*RefreshTokenResult, error) {
	s.logger.Info("Token refresh attempt")

	// First, validate the refresh token to extract user info
	refreshClaims, err := s.jwtService.ValidateRefreshToken(input.RefreshToken)
	if err != nil {
		s.logger.Warn("Refresh token validation failed", zap.Error(err))

		switch err {
		case auth.ErrExpiredToken:
			return nil, shared.NewDomainError("TOKEN_EXPIRED", "Refresh token has expired")
		case auth.ErrInvalidToken:
			return nil, shared.NewDomainError("TOKEN_INVALID", "Invalid refresh token")
		case auth.ErrMaxRefreshExceeded:
			return nil, shared.NewDomainError("TOKEN_MAX_REFRESH", "Maximum token refresh count exceeded. Please log in again")
		default:
			return nil, shared.NewDomainError("TOKEN_ERROR", "Failed to validate refresh token")
		}
	}

	// Check if the refresh token or user tokens have been invalidated
	if s.tokenBlacklist != nil {
		// Check if user's tokens have been globally invalidated (logout, password change)
		tokenIssuedAt := refreshClaims.GetIssuedAtTime()
		invalidated, err := s.tokenBlacklist.IsUserTokenInvalidated(ctx, refreshClaims.UserID, tokenIssuedAt)
		if err != nil {
			s.logger.Error("Failed to check token invalidation during refresh",
				zap.String("user_id", refreshClaims.UserID),
				zap.Error(err))
			// Don't fail on error - allow refresh for availability
		} else if invalidated {
			s.logger.Warn("Refresh token rejected - user tokens invalidated",
				zap.String("user_id", refreshClaims.UserID))
			return nil, shared.NewDomainError("TOKEN_REVOKED", "Session has been invalidated. Please log in again")
		}
	}

	// Parse user ID from refresh token claims
	userID, err := uuid.Parse(refreshClaims.UserID)
	if err != nil {
		s.logger.Error("Invalid user ID in refresh token", zap.Error(err))
		return nil, shared.NewDomainError("TOKEN_INVALID", "Invalid user ID in token")
	}

	s.logger.Info("Token refresh for user", zap.String("user_id", userID.String()))

	// Find user to verify they still exist and are active
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		s.logger.Warn("User not found during token refresh", zap.String("user_id", userID.String()))
		return nil, shared.NewDomainError("USER_NOT_FOUND", "User not found")
	}

	// Check if user can still access the system
	if !user.CanLogin() {
		s.logger.Warn("Token refresh for inactive user", zap.String("user_id", userID.String()))
		return nil, shared.NewDomainError("ACCOUNT_INACTIVE", "Account is no longer active")
	}

	// Load user roles
	if err := s.userRepo.LoadUserRoles(ctx, user); err != nil {
		s.logger.Error("Failed to load user roles during refresh", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to load user roles")
	}

	// Collect updated permissions
	permissions, err := s.collectUserPermissions(ctx, user.RoleIDs)
	if err != nil {
		s.logger.Error("Failed to collect permissions during refresh", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to load user permissions")
	}

	// Refresh the token pair
	tokenPair, err := s.jwtService.RefreshTokenPair(input.RefreshToken, permissions)
	if err != nil {
		s.logger.Warn("Token refresh failed", zap.Error(err))

		// Map JWT errors to domain errors
		switch err {
		case auth.ErrExpiredToken:
			return nil, shared.NewDomainError("TOKEN_EXPIRED", "Refresh token has expired")
		case auth.ErrInvalidToken:
			return nil, shared.NewDomainError("TOKEN_INVALID", "Invalid refresh token")
		case auth.ErrMaxRefreshExceeded:
			return nil, shared.NewDomainError("TOKEN_MAX_REFRESH", "Maximum token refresh count exceeded. Please log in again")
		default:
			return nil, shared.NewDomainError("TOKEN_ERROR", "Failed to refresh token")
		}
	}

	s.logger.Info("Token refreshed successfully", zap.String("user_id", userID.String()))

	return &RefreshTokenResult{
		AccessToken:           tokenPair.AccessToken,
		RefreshToken:          tokenPair.RefreshToken,
		AccessTokenExpiresAt:  tokenPair.AccessTokenExpiresAt,
		RefreshTokenExpiresAt: tokenPair.RefreshTokenExpiresAt,
		TokenType:             tokenPair.TokenType,
	}, nil
}

// Logout handles user logout by invalidating all tokens for the user
// This ensures both access and refresh tokens are immediately invalidated
func (s *AuthService) Logout(ctx context.Context, input LogoutInput) error {
	s.logger.Info("User logout",
		zap.String("user_id", input.UserID.String()),
		zap.String("tenant_id", input.TenantID.String()),
		zap.String("token_jti", input.TokenJTI))

	// Invalidate all tokens for the user to ensure complete logout
	// This is more secure than just blacklisting the access token JTI
	// because it also invalidates any refresh tokens the user might have
	if s.tokenBlacklist != nil {
		// Use refresh token expiration as TTL since that's the longest-lived token
		ttl := s.jwtService.GetRefreshTokenExpiration()

		if err := s.tokenBlacklist.AddUserTokensToBlacklist(ctx, input.UserID.String(), ttl); err != nil {
			s.logger.Error("Failed to invalidate user tokens on logout",
				zap.String("user_id", input.UserID.String()),
				zap.Error(err))
			// Don't fail logout if invalidation fails - tokens will still expire naturally
			// But log the error for monitoring
		} else {
			s.logger.Info("All user tokens invalidated on logout",
				zap.String("user_id", input.UserID.String()))
		}
	}

	return nil
}

// GetCurrentUser retrieves the current user's information
func (s *AuthService) GetCurrentUser(ctx context.Context, input GetCurrentUserInput) (*CurrentUserResult, error) {
	user, err := s.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return nil, shared.NewDomainError("USER_NOT_FOUND", "User not found")
	}

	// Load roles
	if err := s.userRepo.LoadUserRoles(ctx, user); err != nil {
		s.logger.Error("Failed to load user roles", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to load user roles")
	}

	// Collect permissions
	permissions, err := s.collectUserPermissions(ctx, user.RoleIDs)
	if err != nil {
		s.logger.Error("Failed to collect permissions", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to load user permissions")
	}

	return &CurrentUserResult{
		User: UserInfo{
			ID:          user.ID,
			TenantID:    user.TenantID,
			Username:    user.Username,
			DisplayName: user.GetDisplayNameOrUsername(),
			Email:       user.Email,
			Phone:       user.Phone,
			Avatar:      user.Avatar,
			Permissions: permissions,
			RoleIDs:     user.RoleIDs,
		},
		Permissions: permissions,
	}, nil
}

// ChangePassword changes a user's password and invalidates all existing tokens
func (s *AuthService) ChangePassword(ctx context.Context, input ChangePasswordInput) error {
	user, err := s.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return shared.NewDomainError("USER_NOT_FOUND", "User not found")
	}

	if err := user.ChangePassword(input.OldPassword, input.NewPassword); err != nil {
		return err
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error("Failed to update user after password change", zap.Error(err))
		return shared.NewDomainError("INTERNAL_ERROR", "Failed to update password")
	}

	s.logger.Info("User password changed", zap.String("user_id", input.UserID.String()))

	// Invalidate all existing tokens for security
	// User must re-login on all devices after password change
	if s.tokenBlacklist != nil {
		// Use refresh token expiration as TTL since that's the longest-lived token
		ttl := s.jwtService.GetRefreshTokenExpiration()

		if err := s.tokenBlacklist.AddUserTokensToBlacklist(ctx, input.UserID.String(), ttl); err != nil {
			s.logger.Error("Failed to invalidate user tokens after password change",
				zap.String("user_id", input.UserID.String()),
				zap.Error(err))
			// Don't fail password change if token invalidation fails
			// The password was changed successfully, just log the error
		} else {
			s.logger.Info("All user tokens invalidated after password change",
				zap.String("user_id", input.UserID.String()))
		}
	}

	return nil
}

// ForceLogout invalidates all tokens for a user, forcing them to re-authenticate
// This is typically used by administrators when a user's account is compromised
func (s *AuthService) ForceLogout(ctx context.Context, input ForceLogoutInput) (*ForceLogoutResult, error) {
	s.logger.Info("Force logout initiated",
		zap.String("admin_user_id", input.AdminUserID.String()),
		zap.String("target_user_id", input.TargetUserID.String()),
		zap.String("tenant_id", input.TenantID.String()),
		zap.String("reason", input.Reason))

	// Verify target user exists
	user, err := s.userRepo.FindByID(ctx, input.TargetUserID)
	if err != nil {
		return nil, shared.NewDomainError("USER_NOT_FOUND", "Target user not found")
	}

	// Verify target user is in the same tenant
	if user.TenantID != input.TenantID {
		s.logger.Warn("Force logout attempt across tenant boundary",
			zap.String("admin_user_id", input.AdminUserID.String()),
			zap.String("target_user_id", input.TargetUserID.String()),
			zap.String("admin_tenant_id", input.TenantID.String()),
			zap.String("target_tenant_id", user.TenantID.String()))
		return nil, shared.NewDomainError("FORBIDDEN", "Cannot force logout user from different tenant")
	}

	// Check if token blacklist is available
	if s.tokenBlacklist == nil {
		s.logger.Warn("Token blacklist not configured, force logout has limited effect")
		return &ForceLogoutResult{
			Message: "Force logout completed (token blacklist not configured - user sessions will expire naturally)",
		}, nil
	}

	// Use refresh token expiration as TTL since that's the longest-lived token
	ttl := s.jwtService.GetRefreshTokenExpiration()

	// Invalidate all tokens for the target user
	if err := s.tokenBlacklist.AddUserTokensToBlacklist(ctx, input.TargetUserID.String(), ttl); err != nil {
		s.logger.Error("Failed to invalidate user tokens on force logout",
			zap.String("target_user_id", input.TargetUserID.String()),
			zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to invalidate user sessions")
	}

	s.logger.Info("Force logout completed successfully",
		zap.String("admin_user_id", input.AdminUserID.String()),
		zap.String("target_user_id", input.TargetUserID.String()),
		zap.String("reason", input.Reason))

	return &ForceLogoutResult{
		Message: "All sessions for user have been invalidated",
	}, nil
}

// IsTokenBlacklisted checks if a token (by JTI) is blacklisted
func (s *AuthService) IsTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	if s.tokenBlacklist == nil {
		return false, nil
	}
	return s.tokenBlacklist.IsBlacklisted(ctx, jti)
}

// IsUserTokenInvalidated checks if a user's tokens have been globally invalidated
func (s *AuthService) IsUserTokenInvalidated(ctx context.Context, userID string, tokenIssuedAt time.Time) (bool, error) {
	if s.tokenBlacklist == nil {
		return false, nil
	}
	return s.tokenBlacklist.IsUserTokenInvalidated(ctx, userID, tokenIssuedAt)
}

// collectUserPermissions collects all unique permissions from the user's roles
func (s *AuthService) collectUserPermissions(ctx context.Context, roleIDs []uuid.UUID) ([]string, error) {
	if len(roleIDs) == 0 {
		return []string{}, nil
	}

	// Find all roles
	roles, err := s.roleRepo.FindByIDs(ctx, roleIDs)
	if err != nil {
		return nil, err
	}

	// Collect unique permissions
	permSet := make(map[string]struct{})
	for _, role := range roles {
		if !role.IsEnabled {
			continue
		}
		// Load permissions for the role
		if err := s.roleRepo.LoadPermissions(ctx, role); err != nil {
			s.logger.Warn("Failed to load permissions for role",
				zap.String("role_id", role.ID.String()),
				zap.Error(err))
			continue
		}
		for _, perm := range role.Permissions {
			permSet[perm.Code] = struct{}{}
		}
	}

	// Convert to slice
	permissions := make([]string, 0, len(permSet))
	for perm := range permSet {
		permissions = append(permissions, perm)
	}

	return permissions, nil
}
