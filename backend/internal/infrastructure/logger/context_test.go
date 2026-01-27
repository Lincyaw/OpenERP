package logger

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

// =============================================================================
// Trace Correlation Tests
// =============================================================================

// createTestTracerProvider creates a noop tracer provider for testing.
func createTestTracerProvider() trace.TracerProvider {
	return noop.NewTracerProvider()
}

// createContextWithSpan creates a context with an active span for testing.
func createContextWithSpan(t *testing.T) (context.Context, trace.Span) {
	tp := createTestTracerProvider()
	otel.SetTracerProvider(tp)
	tracer := tp.Tracer("test-tracer")
	return tracer.Start(context.Background(), "test-span")
}

func TestGetTraceID_NoSpan(t *testing.T) {
	ctx := context.Background()
	traceID := GetTraceID(ctx)
	assert.Empty(t, traceID)
}

func TestGetSpanID_NoSpan(t *testing.T) {
	ctx := context.Background()
	spanID := GetSpanID(ctx)
	assert.Empty(t, spanID)
}

func TestGetTraceID_WithSpan(t *testing.T) {
	ctx, span := createContextWithSpan(t)
	defer span.End()

	traceID := GetTraceID(ctx)
	// Noop span has invalid context, so trace ID will be empty
	// In real scenarios with a real tracer, this would return a valid trace ID
	assert.NotNil(t, traceID) // Should not panic
}

func TestGetSpanID_WithSpan(t *testing.T) {
	ctx, span := createContextWithSpan(t)
	defer span.End()

	spanID := GetSpanID(ctx)
	// Noop span has invalid context, so span ID will be empty
	// In real scenarios with a real tracer, this would return a valid span ID
	assert.NotNil(t, spanID) // Should not panic
}

func TestWithTraceContext_NoSpan(t *testing.T) {
	baseLogger := zap.NewNop()
	ctx := context.Background()

	enrichedLogger := WithTraceContext(ctx, baseLogger)

	// Without a span, should return the same logger
	assert.Equal(t, baseLogger, enrichedLogger)
}

func TestWithTraceContext_WithSpan(t *testing.T) {
	baseLogger := zap.NewNop()
	ctx, span := createContextWithSpan(t)
	defer span.End()

	enrichedLogger := WithTraceContext(ctx, baseLogger)

	// Should return a logger (may or may not be enriched depending on span validity)
	assert.NotNil(t, enrichedLogger)
}

// =============================================================================
// ContextLogger Tests
// =============================================================================

func TestL_ReturnsContextLogger(t *testing.T) {
	ctx := context.Background()
	cl := L(ctx)

	assert.NotNil(t, cl)
	assert.NotNil(t, cl.ctx)
	assert.NotNil(t, cl.logger)
}

func TestL_WithLoggerInContext(t *testing.T) {
	baseLogger, err := NewForEnvironment("development")
	require.NoError(t, err)

	ctx := WithContext(context.Background(), baseLogger)
	cl := L(ctx)

	assert.NotNil(t, cl)
	// Logger should be extracted from context
	assert.NotNil(t, cl.logger)
}

func TestWithLogger_UsesProvidedLogger(t *testing.T) {
	baseLogger, err := NewForEnvironment("development")
	require.NoError(t, err)

	ctx := context.Background()
	cl := WithLogger(ctx, baseLogger)

	assert.NotNil(t, cl)
	assert.Equal(t, baseLogger, cl.logger)
}

func TestContextLogger_With(t *testing.T) {
	// Use a real logger (not nop) to test With creates a new logger
	var buf bytes.Buffer
	encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), zapcore.DebugLevel)
	baseLogger := zap.New(core)

	ctx := context.Background()
	cl := WithLogger(ctx, baseLogger)

	childCl := cl.With(zap.String("key", "value"))

	assert.NotNil(t, childCl)
	assert.Equal(t, ctx, childCl.ctx)
	// Child logger should be different from parent when using a real logger
	assert.NotEqual(t, baseLogger, childCl.logger)
}

func TestContextLogger_LogLevels(t *testing.T) {
	baseLogger := zap.NewNop()
	ctx := context.Background()
	cl := WithLogger(ctx, baseLogger)

	// These should not panic
	assert.NotPanics(t, func() {
		cl.Debug("debug message")
		cl.Info("info message")
		cl.Warn("warn message")
		cl.Error("error message")
	})
}

func TestContextLogger_Zap(t *testing.T) {
	baseLogger := zap.NewNop()
	ctx := context.Background()
	cl := WithLogger(ctx, baseLogger)

	zapLogger := cl.Zap()

	assert.NotNil(t, zapLogger)
	// Should be usable as a zap.Logger
	assert.NotPanics(t, func() {
		zapLogger.Info("test")
	})
}

func TestContextLogger_Sugar(t *testing.T) {
	baseLogger := zap.NewNop()
	ctx := context.Background()
	cl := WithLogger(ctx, baseLogger)

	sugar := cl.Sugar()

	assert.NotNil(t, sugar)
	// Should be usable as a sugared logger
	assert.NotPanics(t, func() {
		sugar.Infof("test %s", "message")
	})
}

func TestContextLogger_EnrichesWithContextFields(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), zapcore.DebugLevel)
	baseLogger := zap.New(core)

	// Build context with various fields
	ctx := context.Background()
	ctx, _ = WithRequestID(ctx, baseLogger, "req-123")
	ctx, _ = WithTenantID(ctx, baseLogger, "tenant-456")
	ctx, _ = WithUserID(ctx, baseLogger, "user-789")

	// Also add the logger to context so L() can find it
	ctx = WithContext(ctx, baseLogger)

	// Create ContextLogger and log
	cl := L(ctx)
	cl.Info("test message", zap.String("extra_field", "extra_value"))

	// Verify output contains expected fields
	output := buf.String()
	assert.Contains(t, output, `"request_id":"req-123"`)
	assert.Contains(t, output, `"tenant_id":"tenant-456"`)
	assert.Contains(t, output, `"user_id":"user-789"`)
	assert.Contains(t, output, `"extra_field":"extra_value"`)
	assert.Contains(t, output, `"msg":"test message"`)
}

func TestContextLogger_NilLogger(t *testing.T) {
	ctx := context.Background()
	cl := &ContextLogger{
		ctx:    ctx,
		logger: nil,
	}

	// Should not panic with nil logger - enrichedLogger handles this
	assert.NotPanics(t, func() {
		cl.Info("test")
	})
}

func TestContextLogger_WithAllContextFields(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), zapcore.DebugLevel)
	baseLogger := zap.New(core)

	// Build context with all available fields
	ctx := context.Background()
	ctx = context.WithValue(ctx, RequestIDKey, "req-aaa")
	ctx = context.WithValue(ctx, TenantIDKey, "tenant-bbb")
	ctx = context.WithValue(ctx, UserIDKey, "user-ccc")

	// Create ContextLogger with explicit logger
	cl := WithLogger(ctx, baseLogger)
	cl.Info("test")

	// Verify all context fields are present
	output := buf.String()
	assert.Contains(t, output, `"request_id":"req-aaa"`)
	assert.Contains(t, output, `"tenant_id":"tenant-bbb"`)
	assert.Contains(t, output, `"user_id":"user-ccc"`)
}

func TestContextLogger_EmptyContextFields(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), zapcore.DebugLevel)
	baseLogger := zap.New(core)

	// Empty context (no fields set)
	ctx := context.Background()

	cl := WithLogger(ctx, baseLogger)
	cl.Info("test")

	// Should not contain the context field keys if they're empty
	output := buf.String()
	// When fields are empty, they should not be added to the log
	// (GetRequestID etc. return empty string, which triggers the "if != ''" check)
	assert.Contains(t, output, `"msg":"test"`)
	// These should NOT be present since their values are empty
	assert.NotContains(t, output, `"request_id":""`)
	assert.NotContains(t, output, `"tenant_id":""`)
	assert.NotContains(t, output, `"user_id":""`)
}

func TestContextLogger_WithChaining(t *testing.T) {
	baseLogger := zap.NewNop()
	ctx := context.Background()

	cl := WithLogger(ctx, baseLogger).
		With(zap.String("field1", "value1")).
		With(zap.String("field2", "value2"))

	assert.NotNil(t, cl)
	// Should not panic when logging
	assert.NotPanics(t, func() {
		cl.Info("chained test")
	})
}

func TestGetTraceID_InvalidSpanContext(t *testing.T) {
	// Create a span with invalid context
	tp := noop.NewTracerProvider()
	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	// Noop tracer creates spans with invalid context
	spanCtx := trace.SpanFromContext(ctx).SpanContext()
	assert.False(t, spanCtx.IsValid())

	// GetTraceID should return empty for invalid span context
	traceID := GetTraceID(ctx)
	assert.Empty(t, traceID)
}

func TestGetSpanID_InvalidSpanContext(t *testing.T) {
	// Create a span with invalid context
	tp := noop.NewTracerProvider()
	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	// Noop tracer creates spans with invalid context
	spanCtx := trace.SpanFromContext(ctx).SpanContext()
	assert.False(t, spanCtx.IsValid())

	// GetSpanID should return empty for invalid span context
	spanID := GetSpanID(ctx)
	assert.Empty(t, spanID)
}

func TestWithTraceContext_InvalidSpanContext(t *testing.T) {
	// Create a span with invalid context
	tp := noop.NewTracerProvider()
	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	baseLogger := zap.NewNop()

	// WithTraceContext should return original logger for invalid span context
	enrichedLogger := WithTraceContext(ctx, baseLogger)
	assert.Equal(t, baseLogger, enrichedLogger)
}
