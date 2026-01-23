package logger

import (
	"context"

	"go.uber.org/zap"
)

// contextKey is a type for context keys used by the logger package
type contextKey string

const (
	// LoggerKey is the context key for the logger
	LoggerKey contextKey = "logger"
	// RequestIDKey is the context key for request ID
	RequestIDKey contextKey = "request_id"
	// TenantIDKey is the context key for tenant ID
	TenantIDKey contextKey = "tenant_id"
	// UserIDKey is the context key for user ID
	UserIDKey contextKey = "user_id"
)

// WithContext returns a new context with the logger attached
func WithContext(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, LoggerKey, logger)
}

// FromContext retrieves the logger from context, returns default logger if not found
func FromContext(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value(LoggerKey).(*zap.Logger); ok {
		return logger
	}
	// Return a no-op logger if not found
	return zap.NewNop()
}

// WithRequestID adds request ID to context and returns enriched logger
func WithRequestID(ctx context.Context, logger *zap.Logger, requestID string) (context.Context, *zap.Logger) {
	ctx = context.WithValue(ctx, RequestIDKey, requestID)
	enrichedLogger := logger.With(zap.String("request_id", requestID))
	return WithContext(ctx, enrichedLogger), enrichedLogger
}

// WithTenantID adds tenant ID to context and returns enriched logger
func WithTenantID(ctx context.Context, logger *zap.Logger, tenantID string) (context.Context, *zap.Logger) {
	ctx = context.WithValue(ctx, TenantIDKey, tenantID)
	enrichedLogger := logger.With(zap.String("tenant_id", tenantID))
	return WithContext(ctx, enrichedLogger), enrichedLogger
}

// WithUserID adds user ID to context and returns enriched logger
func WithUserID(ctx context.Context, logger *zap.Logger, userID string) (context.Context, *zap.Logger) {
	ctx = context.WithValue(ctx, UserIDKey, userID)
	enrichedLogger := logger.With(zap.String("user_id", userID))
	return WithContext(ctx, enrichedLogger), enrichedLogger
}

// GetRequestID retrieves request ID from context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// GetTenantID retrieves tenant ID from context
func GetTenantID(ctx context.Context) string {
	if tenantID, ok := ctx.Value(TenantIDKey).(string); ok {
		return tenantID
	}
	return ""
}

// GetUserID retrieves user ID from context
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}
