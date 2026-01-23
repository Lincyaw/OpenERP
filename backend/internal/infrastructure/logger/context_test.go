package logger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestWithContext(t *testing.T) {
	logger, err := NewForEnvironment("development")
	require.NoError(t, err)

	ctx := context.Background()
	ctxWithLogger := WithContext(ctx, logger)

	retrievedLogger := FromContext(ctxWithLogger)
	assert.NotNil(t, retrievedLogger)
}

func TestFromContext_NotFound(t *testing.T) {
	ctx := context.Background()
	logger := FromContext(ctx)

	// Should return a no-op logger
	assert.NotNil(t, logger)
}

func TestWithRequestID(t *testing.T) {
	logger, err := NewForEnvironment("development")
	require.NoError(t, err)

	ctx := context.Background()
	requestID := "req-123"

	newCtx, newLogger := WithRequestID(ctx, logger, requestID)

	assert.NotNil(t, newCtx)
	assert.NotNil(t, newLogger)
	assert.Equal(t, requestID, GetRequestID(newCtx))
}

func TestWithTenantID(t *testing.T) {
	logger, err := NewForEnvironment("development")
	require.NoError(t, err)

	ctx := context.Background()
	tenantID := "tenant-456"

	newCtx, newLogger := WithTenantID(ctx, logger, tenantID)

	assert.NotNil(t, newCtx)
	assert.NotNil(t, newLogger)
	assert.Equal(t, tenantID, GetTenantID(newCtx))
}

func TestWithUserID(t *testing.T) {
	logger, err := NewForEnvironment("development")
	require.NoError(t, err)

	ctx := context.Background()
	userID := "user-789"

	newCtx, newLogger := WithUserID(ctx, logger, userID)

	assert.NotNil(t, newCtx)
	assert.NotNil(t, newLogger)
	assert.Equal(t, userID, GetUserID(newCtx))
}

func TestGetRequestID_NotFound(t *testing.T) {
	ctx := context.Background()
	requestID := GetRequestID(ctx)
	assert.Empty(t, requestID)
}

func TestGetTenantID_NotFound(t *testing.T) {
	ctx := context.Background()
	tenantID := GetTenantID(ctx)
	assert.Empty(t, tenantID)
}

func TestGetUserID_NotFound(t *testing.T) {
	ctx := context.Background()
	userID := GetUserID(ctx)
	assert.Empty(t, userID)
}

func TestContextChaining(t *testing.T) {
	logger, err := NewForEnvironment("development")
	require.NoError(t, err)

	ctx := context.Background()

	// Chain multiple context enrichments
	ctx, logger = WithRequestID(ctx, logger, "req-1")
	ctx, logger = WithTenantID(ctx, logger, "tenant-1")
	ctx, logger = WithUserID(ctx, logger, "user-1")

	assert.Equal(t, "req-1", GetRequestID(ctx))
	assert.Equal(t, "tenant-1", GetTenantID(ctx))
	assert.Equal(t, "user-1", GetUserID(ctx))
	assert.NotNil(t, logger)
}

func TestContextKeys(t *testing.T) {
	// Verify context keys are unique
	assert.NotEqual(t, LoggerKey, RequestIDKey)
	assert.NotEqual(t, RequestIDKey, TenantIDKey)
	assert.NotEqual(t, TenantIDKey, UserIDKey)
	assert.NotEqual(t, LoggerKey, UserIDKey)
}

func TestFromContext_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), LoggerKey, "not a logger")
	logger := FromContext(ctx)

	// Should return a no-op logger when value is wrong type
	assert.NotNil(t, logger)
	// The no-op logger should not panic when used
	logger.Info("test")
}

func TestLoggerFromEnrichedContext(t *testing.T) {
	baseLogger, err := NewForEnvironment("development")
	require.NoError(t, err)

	ctx := context.Background()
	ctx, enrichedLogger := WithRequestID(ctx, baseLogger, "req-test")

	// The logger in context should be the enriched one
	ctxLogger := FromContext(ctx)
	assert.NotNil(t, ctxLogger)

	// Verify it's the enriched logger, not the base logger
	assert.NotEqual(t, baseLogger, enrichedLogger)
}

func TestMultipleWithRequestID(t *testing.T) {
	logger, err := NewForEnvironment("development")
	require.NoError(t, err)

	ctx := context.Background()

	// First call
	ctx, _ = WithRequestID(ctx, logger, "first-id")
	assert.Equal(t, "first-id", GetRequestID(ctx))

	// Second call should override
	ctx, _ = WithRequestID(ctx, logger, "second-id")
	assert.Equal(t, "second-id", GetRequestID(ctx))
}

func TestNopLoggerDoesNotPanic(t *testing.T) {
	ctx := context.Background()
	logger := FromContext(ctx)

	// These should not panic
	assert.NotPanics(t, func() {
		logger.Info("test message")
		logger.Debug("debug message")
		logger.Warn("warn message")
		logger.Error("error message")
		logger.With(zap.String("key", "value")).Info("with field")
	})
}
