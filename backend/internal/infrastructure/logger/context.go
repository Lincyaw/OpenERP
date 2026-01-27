package logger

import (
	"context"

	"go.opentelemetry.io/otel/trace"
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

// =============================================================================
// Trace Correlation Functions
// =============================================================================

// GetTraceID extracts the trace ID from the context's span.
// Returns an empty string if no active span exists or trace is invalid.
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return ""
	}
	spanCtx := span.SpanContext()
	if !spanCtx.IsValid() {
		return ""
	}
	return spanCtx.TraceID().String()
}

// GetSpanID extracts the span ID from the context's span.
// Returns an empty string if no active span exists or span is invalid.
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return ""
	}
	spanCtx := span.SpanContext()
	if !spanCtx.IsValid() {
		return ""
	}
	return spanCtx.SpanID().String()
}

// WithTraceContext adds trace_id and span_id to the logger from the context's span.
// If no valid span exists, returns the original logger unchanged.
func WithTraceContext(ctx context.Context, logger *zap.Logger) *zap.Logger {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return logger
	}
	spanCtx := span.SpanContext()
	if !spanCtx.IsValid() {
		return logger
	}

	return logger.With(
		zap.String("trace_id", spanCtx.TraceID().String()),
		zap.String("span_id", spanCtx.SpanID().String()),
	)
}

// ContextLogger is a wrapper that provides convenient logging with automatic
// trace correlation. It extracts trace_id, span_id, tenant_id, user_id from
// the context and injects them into every log entry.
type ContextLogger struct {
	ctx    context.Context
	logger *zap.Logger
}

// L returns a ContextLogger from the given context.
// Usage: logger.L(ctx).Info("message", zap.String("key", "value"))
//
// This automatically injects:
//   - trace_id: from OpenTelemetry span context
//   - span_id: from OpenTelemetry span context
//   - tenant_id: if present in context
//   - user_id: if present in context
//   - request_id: if present in context
func L(ctx context.Context) *ContextLogger {
	return &ContextLogger{
		ctx:    ctx,
		logger: FromContext(ctx),
	}
}

// WithLogger returns a ContextLogger using the provided logger instead of
// extracting from context. Useful when you have a pre-configured logger.
func WithLogger(ctx context.Context, logger *zap.Logger) *ContextLogger {
	return &ContextLogger{
		ctx:    ctx,
		logger: logger,
	}
}

// enrichedLogger returns a logger enriched with trace and context fields.
func (cl *ContextLogger) enrichedLogger() *zap.Logger {
	l := cl.logger
	if l == nil {
		l = zap.NewNop()
	}

	// Add trace context
	span := trace.SpanFromContext(cl.ctx)
	if span != nil {
		spanCtx := span.SpanContext()
		if spanCtx.IsValid() {
			l = l.With(
				zap.String("trace_id", spanCtx.TraceID().String()),
				zap.String("span_id", spanCtx.SpanID().String()),
			)
		}
	}

	// Add request ID if present
	if requestID := GetRequestID(cl.ctx); requestID != "" {
		l = l.With(zap.String("request_id", requestID))
	}

	// Add tenant ID if present
	if tenantID := GetTenantID(cl.ctx); tenantID != "" {
		l = l.With(zap.String("tenant_id", tenantID))
	}

	// Add user ID if present
	if userID := GetUserID(cl.ctx); userID != "" {
		l = l.With(zap.String("user_id", userID))
	}

	return l
}

// With creates a child ContextLogger with additional fields.
func (cl *ContextLogger) With(fields ...zap.Field) *ContextLogger {
	return &ContextLogger{
		ctx:    cl.ctx,
		logger: cl.logger.With(fields...),
	}
}

// Debug logs a debug level message with trace context.
func (cl *ContextLogger) Debug(msg string, fields ...zap.Field) {
	cl.enrichedLogger().Debug(msg, fields...)
}

// Info logs an info level message with trace context.
func (cl *ContextLogger) Info(msg string, fields ...zap.Field) {
	cl.enrichedLogger().Info(msg, fields...)
}

// Warn logs a warning level message with trace context.
func (cl *ContextLogger) Warn(msg string, fields ...zap.Field) {
	cl.enrichedLogger().Warn(msg, fields...)
}

// Error logs an error level message with trace context.
func (cl *ContextLogger) Error(msg string, fields ...zap.Field) {
	cl.enrichedLogger().Error(msg, fields...)
}

// Fatal logs a fatal level message with trace context and then calls os.Exit(1).
func (cl *ContextLogger) Fatal(msg string, fields ...zap.Field) {
	cl.enrichedLogger().Fatal(msg, fields...)
}

// Panic logs a panic level message with trace context and then panics.
func (cl *ContextLogger) Panic(msg string, fields ...zap.Field) {
	cl.enrichedLogger().Panic(msg, fields...)
}

// Zap returns the underlying zap.Logger enriched with trace context.
// This is useful when you need to pass the logger to functions that
// expect a *zap.Logger.
func (cl *ContextLogger) Zap() *zap.Logger {
	return cl.enrichedLogger()
}

// Sugar returns a sugared logger enriched with trace context.
func (cl *ContextLogger) Sugar() *zap.SugaredLogger {
	return cl.enrichedLogger().Sugar()
}
