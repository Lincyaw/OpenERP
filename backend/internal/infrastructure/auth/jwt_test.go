package auth

import (
	"testing"
	"time"

	"github.com/erp/backend/internal/infrastructure/config"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestJWTService() *JWTService {
	cfg := config.JWTConfig{
		Secret:                 "test-secret-key-at-least-32-chars",
		RefreshSecret:          "test-refresh-secret-key-32-chars",
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test-issuer",
		MaxRefreshCount:        10,
	}
	return NewJWTService(cfg)
}

func newTestInput() GenerateTokenInput {
	return GenerateTokenInput{
		TenantID:    uuid.New(),
		UserID:      uuid.New(),
		Username:    "testuser",
		RoleIDs:     []uuid.UUID{uuid.New(), uuid.New()},
		Permissions: []string{"product:read", "product:create", "customer:read"},
	}
}

func TestNewJWTService(t *testing.T) {
	cfg := config.JWTConfig{
		Secret:                 "test-secret",
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test-issuer",
		MaxRefreshCount:        5,
	}

	svc := NewJWTService(cfg)

	assert.NotNil(t, svc)
	assert.Equal(t, []byte(cfg.Secret), svc.accessSecret)
	assert.Equal(t, cfg.AccessTokenExpiration, svc.accessExpiration)
	assert.Equal(t, cfg.RefreshTokenExpiration, svc.refreshExpiration)
	assert.Equal(t, cfg.Issuer, svc.issuer)
	assert.Equal(t, cfg.MaxRefreshCount, svc.maxRefreshCount)
}

func TestNewJWTService_UsesSecretForRefreshIfNotProvided(t *testing.T) {
	cfg := config.JWTConfig{
		Secret:        "test-secret",
		RefreshSecret: "", // Empty
	}

	svc := NewJWTService(cfg)

	assert.Equal(t, []byte(cfg.Secret), svc.refreshSecret)
}

func TestGenerateTokenPair(t *testing.T) {
	svc := newTestJWTService()
	input := newTestInput()

	pair, err := svc.GenerateTokenPair(input)

	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.Equal(t, "Bearer", pair.TokenType)
	assert.True(t, pair.AccessTokenExpiresAt.After(time.Now()))
	assert.True(t, pair.RefreshTokenExpiresAt.After(time.Now()))
	assert.True(t, pair.RefreshTokenExpiresAt.After(pair.AccessTokenExpiresAt))
}

func TestValidateAccessToken_Success(t *testing.T) {
	svc := newTestJWTService()
	input := newTestInput()

	pair, err := svc.GenerateTokenPair(input)
	require.NoError(t, err)

	claims, err := svc.ValidateAccessToken(pair.AccessToken)

	require.NoError(t, err)
	assert.Equal(t, input.TenantID.String(), claims.TenantID)
	assert.Equal(t, input.UserID.String(), claims.UserID)
	assert.Equal(t, input.Username, claims.Username)
	assert.Equal(t, TokenTypeAccess, claims.TokenType)
	assert.Len(t, claims.RoleIDs, len(input.RoleIDs))
	assert.Equal(t, input.Permissions, claims.Permissions)
}

func TestValidateAccessToken_ExpiredToken(t *testing.T) {
	cfg := config.JWTConfig{
		Secret:                 "test-secret-key-at-least-32-chars",
		AccessTokenExpiration:  -1 * time.Hour, // Already expired
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test-issuer",
	}
	svc := NewJWTService(cfg)
	input := newTestInput()

	pair, err := svc.GenerateTokenPair(input)
	require.NoError(t, err)

	_, err = svc.ValidateAccessToken(pair.AccessToken)

	assert.ErrorIs(t, err, ErrExpiredToken)
}

func TestValidateAccessToken_InvalidToken(t *testing.T) {
	svc := newTestJWTService()

	_, err := svc.ValidateAccessToken("invalid-token")

	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidateAccessToken_WrongTokenType(t *testing.T) {
	// Use same secret for both tokens to test token type validation
	cfg := config.JWTConfig{
		Secret:                 "test-secret-key-at-least-32-chars",
		RefreshSecret:          "test-secret-key-at-least-32-chars", // Same as access
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test-issuer",
		MaxRefreshCount:        10,
	}
	svc := NewJWTService(cfg)
	input := newTestInput()

	pair, err := svc.GenerateTokenPair(input)
	require.NoError(t, err)

	// Try to validate refresh token as access token
	_, err = svc.ValidateAccessToken(pair.RefreshToken)

	assert.ErrorIs(t, err, ErrInvalidTokenType)
}

func TestValidateRefreshToken_Success(t *testing.T) {
	svc := newTestJWTService()
	input := newTestInput()

	pair, err := svc.GenerateTokenPair(input)
	require.NoError(t, err)

	claims, err := svc.ValidateRefreshToken(pair.RefreshToken)

	require.NoError(t, err)
	assert.Equal(t, input.TenantID.String(), claims.TenantID)
	assert.Equal(t, input.UserID.String(), claims.UserID)
	assert.Equal(t, TokenTypeRefresh, claims.TokenType)
	assert.Equal(t, 0, claims.RefreshCount)
}

func TestValidateRefreshToken_WrongTokenType(t *testing.T) {
	// Use same secret for both tokens to test token type validation
	cfg := config.JWTConfig{
		Secret:                 "test-secret-key-at-least-32-chars",
		RefreshSecret:          "test-secret-key-at-least-32-chars", // Same as access
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test-issuer",
		MaxRefreshCount:        10,
	}
	svc := NewJWTService(cfg)
	input := newTestInput()

	pair, err := svc.GenerateTokenPair(input)
	require.NoError(t, err)

	// Try to validate access token as refresh token
	_, err = svc.ValidateRefreshToken(pair.AccessToken)

	assert.ErrorIs(t, err, ErrInvalidTokenType)
}

func TestRefreshTokenPair_Success(t *testing.T) {
	svc := newTestJWTService()
	input := newTestInput()

	pair, err := svc.GenerateTokenPair(input)
	require.NoError(t, err)

	newPermissions := []string{"updated:permission"}
	newPair, err := svc.RefreshTokenPair(pair.RefreshToken, newPermissions)

	require.NoError(t, err)
	assert.NotEmpty(t, newPair.AccessToken)
	assert.NotEmpty(t, newPair.RefreshToken)
	assert.NotEqual(t, pair.AccessToken, newPair.AccessToken)
	assert.NotEqual(t, pair.RefreshToken, newPair.RefreshToken)

	// Verify the new access token has updated permissions
	claims, err := svc.ValidateAccessToken(newPair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, newPermissions, claims.Permissions)
}

func TestRefreshTokenPair_IncrementsRefreshCount(t *testing.T) {
	svc := newTestJWTService()
	input := newTestInput()

	pair, err := svc.GenerateTokenPair(input)
	require.NoError(t, err)

	// First refresh
	pair, err = svc.RefreshTokenPair(pair.RefreshToken, nil)
	require.NoError(t, err)

	claims, err := svc.ValidateRefreshToken(pair.RefreshToken)
	require.NoError(t, err)
	assert.Equal(t, 1, claims.RefreshCount)

	// Second refresh
	pair, err = svc.RefreshTokenPair(pair.RefreshToken, nil)
	require.NoError(t, err)

	claims, err = svc.ValidateRefreshToken(pair.RefreshToken)
	require.NoError(t, err)
	assert.Equal(t, 2, claims.RefreshCount)
}

func TestRefreshTokenPair_MaxRefreshExceeded(t *testing.T) {
	cfg := config.JWTConfig{
		Secret:                 "test-secret-key-at-least-32-chars",
		RefreshSecret:          "test-refresh-secret-key-32-chars",
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test-issuer",
		MaxRefreshCount:        2, // Low limit for testing
	}
	svc := NewJWTService(cfg)
	input := newTestInput()

	pair, err := svc.GenerateTokenPair(input)
	require.NoError(t, err)

	// Refresh twice (count goes to 1, then 2)
	pair, err = svc.RefreshTokenPair(pair.RefreshToken, nil)
	require.NoError(t, err)

	pair, err = svc.RefreshTokenPair(pair.RefreshToken, nil)
	require.NoError(t, err)

	// Third refresh should fail (count is now 2, which equals MaxRefreshCount)
	_, err = svc.RefreshTokenPair(pair.RefreshToken, nil)

	assert.ErrorIs(t, err, ErrMaxRefreshExceeded)
}

func TestRefreshTokenPair_InvalidToken(t *testing.T) {
	svc := newTestJWTService()

	_, err := svc.RefreshTokenPair("invalid-token", nil)

	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestRefreshTokenPair_WithAccessToken(t *testing.T) {
	// Use same secret for both tokens to test token type validation
	cfg := config.JWTConfig{
		Secret:                 "test-secret-key-at-least-32-chars",
		RefreshSecret:          "test-secret-key-at-least-32-chars", // Same as refresh
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test-issuer",
		MaxRefreshCount:        10,
	}
	svc := NewJWTService(cfg)
	input := newTestInput()

	pair, err := svc.GenerateTokenPair(input)
	require.NoError(t, err)

	// Try to use access token for refresh
	_, err = svc.RefreshTokenPair(pair.AccessToken, nil)

	assert.ErrorIs(t, err, ErrInvalidTokenType)
}

func TestClaims_GetTenantUUID(t *testing.T) {
	svc := newTestJWTService()
	input := newTestInput()

	pair, err := svc.GenerateTokenPair(input)
	require.NoError(t, err)

	claims, err := svc.ValidateAccessToken(pair.AccessToken)
	require.NoError(t, err)

	tenantUUID, err := claims.GetTenantUUID()

	require.NoError(t, err)
	assert.Equal(t, input.TenantID, tenantUUID)
}

func TestClaims_GetUserUUID(t *testing.T) {
	svc := newTestJWTService()
	input := newTestInput()

	pair, err := svc.GenerateTokenPair(input)
	require.NoError(t, err)

	claims, err := svc.ValidateAccessToken(pair.AccessToken)
	require.NoError(t, err)

	userUUID, err := claims.GetUserUUID()

	require.NoError(t, err)
	assert.Equal(t, input.UserID, userUUID)
}

func TestClaims_GetRoleUUIDs(t *testing.T) {
	svc := newTestJWTService()
	input := newTestInput()

	pair, err := svc.GenerateTokenPair(input)
	require.NoError(t, err)

	claims, err := svc.ValidateAccessToken(pair.AccessToken)
	require.NoError(t, err)

	roleUUIDs, err := claims.GetRoleUUIDs()

	require.NoError(t, err)
	assert.Equal(t, input.RoleIDs, roleUUIDs)
}

func TestClaims_HasPermission(t *testing.T) {
	claims := &Claims{
		Permissions: []string{"product:read", "product:create", "customer:read"},
	}

	assert.True(t, claims.HasPermission("product:read"))
	assert.True(t, claims.HasPermission("product:create"))
	assert.False(t, claims.HasPermission("product:delete"))
}

func TestClaims_HasAnyPermission(t *testing.T) {
	claims := &Claims{
		Permissions: []string{"product:read", "product:create"},
	}

	assert.True(t, claims.HasAnyPermission("product:read", "product:delete"))
	assert.True(t, claims.HasAnyPermission("product:delete", "product:create"))
	assert.False(t, claims.HasAnyPermission("product:delete", "customer:delete"))
}

func TestClaims_HasAllPermissions(t *testing.T) {
	claims := &Claims{
		Permissions: []string{"product:read", "product:create", "customer:read"},
	}

	assert.True(t, claims.HasAllPermissions("product:read"))
	assert.True(t, claims.HasAllPermissions("product:read", "product:create"))
	assert.False(t, claims.HasAllPermissions("product:read", "product:delete"))
}

func TestValidateAccessToken_DifferentSecret(t *testing.T) {
	svc1 := newTestJWTService()
	input := newTestInput()

	pair, err := svc1.GenerateTokenPair(input)
	require.NoError(t, err)

	// Create service with different secret
	cfg := config.JWTConfig{
		Secret:                 "different-secret-key-32-chars!",
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test-issuer",
	}
	svc2 := NewJWTService(cfg)

	_, err = svc2.ValidateAccessToken(pair.AccessToken)

	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestTokenPair_Fields(t *testing.T) {
	svc := newTestJWTService()
	input := newTestInput()

	pair, err := svc.GenerateTokenPair(input)
	require.NoError(t, err)

	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.Equal(t, "Bearer", pair.TokenType)
	assert.False(t, pair.AccessTokenExpiresAt.IsZero())
	assert.False(t, pair.RefreshTokenExpiresAt.IsZero())
}
