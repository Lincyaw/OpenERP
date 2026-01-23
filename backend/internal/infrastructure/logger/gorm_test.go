package logger

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	gormlogger "gorm.io/gorm/logger"
)

func TestNewGormLogger(t *testing.T) {
	core, _ := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(core)

	gormLog := NewGormLogger(zapLogger, gormlogger.Info)

	assert.NotNil(t, gormLog)
	assert.Equal(t, gormlogger.Info, gormLog.logLevel)
}

func TestGormLoggerWithOptions(t *testing.T) {
	core, _ := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(core)

	gormLog := NewGormLogger(
		zapLogger,
		gormlogger.Info,
		WithSlowThreshold(500*time.Millisecond),
		WithIgnoreRecordNotFoundError(false),
	)

	assert.NotNil(t, gormLog)
	assert.Equal(t, 500*time.Millisecond, gormLog.slowThreshold)
	assert.False(t, gormLog.ignoreRecordNotFoundError)
}

func TestGormLogger_LogMode(t *testing.T) {
	core, _ := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(core)

	gormLog := NewGormLogger(zapLogger, gormlogger.Info)
	newLogger := gormLog.LogMode(gormlogger.Warn)

	// Original should be unchanged
	assert.Equal(t, gormlogger.Info, gormLog.logLevel)

	// New logger should have new level
	newGormLog, ok := newLogger.(*GormLogger)
	require.True(t, ok)
	assert.Equal(t, gormlogger.Warn, newGormLog.logLevel)
}

func TestGormLogger_Info(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(core)

	gormLog := NewGormLogger(zapLogger, gormlogger.Info)
	gormLog.Info(context.Background(), "test message %s", "value")

	logs := recorded.All()
	require.Len(t, logs, 1)
	assert.Contains(t, logs[0].Message, "test message value")
}

func TestGormLogger_Info_Suppressed(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(core)

	// Silent level should suppress info
	gormLog := NewGormLogger(zapLogger, gormlogger.Silent)
	gormLog.Info(context.Background(), "test message")

	assert.Empty(t, recorded.All())
}

func TestGormLogger_Warn(t *testing.T) {
	core, recorded := observer.New(zapcore.WarnLevel)
	zapLogger := zap.New(core)

	gormLog := NewGormLogger(zapLogger, gormlogger.Warn)
	gormLog.Warn(context.Background(), "warning message %d", 42)

	logs := recorded.All()
	require.Len(t, logs, 1)
	assert.Contains(t, logs[0].Message, "warning message 42")
	assert.Equal(t, zapcore.WarnLevel, logs[0].Level)
}

func TestGormLogger_Error(t *testing.T) {
	core, recorded := observer.New(zapcore.ErrorLevel)
	zapLogger := zap.New(core)

	gormLog := NewGormLogger(zapLogger, gormlogger.Error)
	gormLog.Error(context.Background(), "error message")

	logs := recorded.All()
	require.Len(t, logs, 1)
	assert.Equal(t, zapcore.ErrorLevel, logs[0].Level)
}

func TestGormLogger_Trace_Error(t *testing.T) {
	core, recorded := observer.New(zapcore.ErrorLevel)
	zapLogger := zap.New(core)

	gormLog := NewGormLogger(zapLogger, gormlogger.Error)

	begin := time.Now()
	fc := func() (string, int64) {
		return "SELECT * FROM users", 0
	}

	gormLog.Trace(context.Background(), begin, fc, errors.New("test error"))

	logs := recorded.All()
	require.Len(t, logs, 1)
	assert.Contains(t, logs[0].Message, "SQL Error")
}

func TestGormLogger_Trace_RecordNotFoundIgnored(t *testing.T) {
	core, recorded := observer.New(zapcore.ErrorLevel)
	zapLogger := zap.New(core)

	gormLog := NewGormLogger(zapLogger, gormlogger.Error, WithIgnoreRecordNotFoundError(true))

	begin := time.Now()
	fc := func() (string, int64) {
		return "SELECT * FROM users WHERE id = ?", 0
	}

	gormLog.Trace(context.Background(), begin, fc, gormlogger.ErrRecordNotFound)

	// Should be ignored
	assert.Empty(t, recorded.All())
}

func TestGormLogger_Trace_SlowQuery(t *testing.T) {
	core, recorded := observer.New(zapcore.WarnLevel)
	zapLogger := zap.New(core)

	// Set a very low threshold for testing
	gormLog := NewGormLogger(
		zapLogger,
		gormlogger.Warn,
		WithSlowThreshold(1*time.Nanosecond),
	)

	begin := time.Now().Add(-1 * time.Second) // Simulate slow query
	fc := func() (string, int64) {
		return "SELECT * FROM users", 10
	}

	gormLog.Trace(context.Background(), begin, fc, nil)

	logs := recorded.All()
	require.Len(t, logs, 1)
	assert.Contains(t, logs[0].Message, "SLOW SQL")
}

func TestGormLogger_Trace_NormalQuery(t *testing.T) {
	core, recorded := observer.New(zapcore.DebugLevel)
	zapLogger := zap.New(core)

	gormLog := NewGormLogger(zapLogger, gormlogger.Info)

	begin := time.Now()
	fc := func() (string, int64) {
		return "SELECT * FROM users", 5
	}

	gormLog.Trace(context.Background(), begin, fc, nil)

	logs := recorded.All()
	require.Len(t, logs, 1)
	assert.Contains(t, logs[0].Message, "SQL Query")
}

func TestGormLogger_Trace_Silent(t *testing.T) {
	core, recorded := observer.New(zapcore.DebugLevel)
	zapLogger := zap.New(core)

	gormLog := NewGormLogger(zapLogger, gormlogger.Silent)

	begin := time.Now()
	fc := func() (string, int64) {
		return "SELECT * FROM users", 5
	}

	gormLog.Trace(context.Background(), begin, fc, nil)

	// Silent mode should log nothing
	assert.Empty(t, recorded.All())
}

func TestGormLogger_Trace_WithRequestID(t *testing.T) {
	core, recorded := observer.New(zapcore.DebugLevel)
	zapLogger := zap.New(core)

	gormLog := NewGormLogger(zapLogger, gormlogger.Info)

	// Create context with request ID
	ctx := context.WithValue(context.Background(), RequestIDKey, "test-req-id")

	begin := time.Now()
	fc := func() (string, int64) {
		return "SELECT * FROM users", 5
	}

	gormLog.Trace(ctx, begin, fc, nil)

	logs := recorded.All()
	require.Len(t, logs, 1)

	// Check that request_id is in the logged fields
	hasRequestID := false
	for _, field := range logs[0].Context {
		if field.Key == "request_id" {
			hasRequestID = true
			assert.Equal(t, "test-req-id", field.String)
		}
	}
	assert.True(t, hasRequestID, "request_id should be in log fields")
}

func TestMapGormLogLevel(t *testing.T) {
	tests := []struct {
		level    string
		expected gormlogger.LogLevel
	}{
		{"silent", gormlogger.Silent},
		{"error", gormlogger.Error},
		{"warn", gormlogger.Warn},
		{"info", gormlogger.Info},
		{"debug", gormlogger.Info},
		{"unknown", gormlogger.Warn},
		{"", gormlogger.Warn},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			result := MapGormLogLevel(tt.level)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGormLoggerImplementsInterface(t *testing.T) {
	core, _ := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(core)

	gormLog := NewGormLogger(zapLogger, gormlogger.Info)

	// Verify it implements gormlogger.Interface
	var _ gormlogger.Interface = gormLog
}
