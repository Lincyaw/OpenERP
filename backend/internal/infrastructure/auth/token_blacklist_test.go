package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/erp/backend/internal/infrastructure/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryTokenBlacklist_AddToBlacklist(t *testing.T) {
	blacklist := auth.NewInMemoryTokenBlacklist()
	ctx := context.Background()

	// Add a token to blacklist
	err := blacklist.AddToBlacklist(ctx, "test-jti-1", 1*time.Hour)
	require.NoError(t, err)

	// Verify it's blacklisted
	isBlacklisted, err := blacklist.IsBlacklisted(ctx, "test-jti-1")
	require.NoError(t, err)
	assert.True(t, isBlacklisted)

	// Verify a different JTI is not blacklisted
	isBlacklisted, err = blacklist.IsBlacklisted(ctx, "test-jti-2")
	require.NoError(t, err)
	assert.False(t, isBlacklisted)
}

func TestInMemoryTokenBlacklist_ExpirationCleanup(t *testing.T) {
	blacklist := auth.NewInMemoryTokenBlacklist()
	ctx := context.Background()

	// Add a token with very short TTL
	err := blacklist.AddToBlacklist(ctx, "test-jti-expire", 1*time.Millisecond)
	require.NoError(t, err)

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Should no longer be blacklisted
	isBlacklisted, err := blacklist.IsBlacklisted(ctx, "test-jti-expire")
	require.NoError(t, err)
	assert.False(t, isBlacklisted)
}

func TestInMemoryTokenBlacklist_UserTokenInvalidation(t *testing.T) {
	blacklist := auth.NewInMemoryTokenBlacklist()
	ctx := context.Background()

	// Token issued before invalidation
	tokenIssuedAt := time.Now().Add(-1 * time.Hour)

	// Initially, token should not be invalidated
	invalidated, err := blacklist.IsUserTokenInvalidated(ctx, "user-1", tokenIssuedAt)
	require.NoError(t, err)
	assert.False(t, invalidated)

	// Invalidate all user tokens
	err = blacklist.AddUserTokensToBlacklist(ctx, "user-1", 1*time.Hour)
	require.NoError(t, err)

	// Token issued before invalidation should be invalid
	invalidated, err = blacklist.IsUserTokenInvalidated(ctx, "user-1", tokenIssuedAt)
	require.NoError(t, err)
	assert.True(t, invalidated)

	// Token issued after invalidation should be valid
	futureToken := time.Now().Add(1 * time.Second)
	time.Sleep(2 * time.Millisecond) // Ensure future token is after invalidation
	invalidated, err = blacklist.IsUserTokenInvalidated(ctx, "user-1", futureToken)
	require.NoError(t, err)
	assert.False(t, invalidated)

	// Different user should not be affected
	invalidated, err = blacklist.IsUserTokenInvalidated(ctx, "user-2", tokenIssuedAt)
	require.NoError(t, err)
	assert.False(t, invalidated)
}

func TestInMemoryTokenBlacklist_MultipleTokens(t *testing.T) {
	blacklist := auth.NewInMemoryTokenBlacklist()
	ctx := context.Background()

	// Add multiple tokens
	for i := 0; i < 10; i++ {
		jti := "test-jti-" + string(rune('a'+i))
		err := blacklist.AddToBlacklist(ctx, jti, 1*time.Hour)
		require.NoError(t, err)
	}

	// Verify all are blacklisted
	for i := 0; i < 10; i++ {
		jti := "test-jti-" + string(rune('a'+i))
		isBlacklisted, err := blacklist.IsBlacklisted(ctx, jti)
		require.NoError(t, err)
		assert.True(t, isBlacklisted, "token %s should be blacklisted", jti)
	}

	// Non-blacklisted token should return false
	isBlacklisted, err := blacklist.IsBlacklisted(ctx, "not-blacklisted")
	require.NoError(t, err)
	assert.False(t, isBlacklisted)
}

func TestInMemoryTokenBlacklist_Interface(t *testing.T) {
	// Ensure InMemoryTokenBlacklist implements TokenBlacklist interface
	var _ auth.TokenBlacklist = (*auth.InMemoryTokenBlacklist)(nil)
	var _ auth.TokenBlacklist = auth.NewInMemoryTokenBlacklist()
}

func TestRedisTokenBlacklist_Interface(t *testing.T) {
	// Ensure RedisTokenBlacklist implements TokenBlacklist interface
	var _ auth.TokenBlacklist = (*auth.RedisTokenBlacklist)(nil)
}
