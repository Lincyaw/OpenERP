package logger

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	gormlogger "gorm.io/gorm/logger"
)

// GormLogger implements GORM's logger interface using zap
type GormLogger struct {
	logger                    *zap.Logger
	logLevel                  gormlogger.LogLevel
	slowThreshold             time.Duration
	ignoreRecordNotFoundError bool
}

// GormLoggerOption is a function that configures a GormLogger
type GormLoggerOption func(*GormLogger)

// WithSlowThreshold sets the slow query threshold
func WithSlowThreshold(threshold time.Duration) GormLoggerOption {
	return func(l *GormLogger) {
		l.slowThreshold = threshold
	}
}

// WithIgnoreRecordNotFoundError configures whether to ignore record not found errors
func WithIgnoreRecordNotFoundError(ignore bool) GormLoggerOption {
	return func(l *GormLogger) {
		l.ignoreRecordNotFoundError = ignore
	}
}

// NewGormLogger creates a new GORM logger backed by zap
func NewGormLogger(zapLogger *zap.Logger, level gormlogger.LogLevel, opts ...GormLoggerOption) *GormLogger {
	gl := &GormLogger{
		logger:                    zapLogger.Named("gorm"),
		logLevel:                  level,
		slowThreshold:             200 * time.Millisecond,
		ignoreRecordNotFoundError: true,
	}

	for _, opt := range opts {
		opt(gl)
	}

	return gl
}

// LogMode implements gormlogger.Interface
func (l *GormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *l
	newLogger.logLevel = level
	return &newLogger
}

// Info implements gormlogger.Interface
func (l *GormLogger) Info(ctx context.Context, msg string, data ...any) {
	if l.logLevel >= gormlogger.Info {
		l.logger.Sugar().Infof(msg, data...)
	}
}

// Warn implements gormlogger.Interface
func (l *GormLogger) Warn(ctx context.Context, msg string, data ...any) {
	if l.logLevel >= gormlogger.Warn {
		l.logger.Sugar().Warnf(msg, data...)
	}
}

// Error implements gormlogger.Interface
func (l *GormLogger) Error(ctx context.Context, msg string, data ...any) {
	if l.logLevel >= gormlogger.Error {
		l.logger.Sugar().Errorf(msg, data...)
	}
}

// Trace implements gormlogger.Interface - logs SQL queries
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.logLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	// Get request ID from context if available
	requestID := GetRequestID(ctx)

	fields := []zap.Field{
		zap.Duration("elapsed", elapsed),
		zap.Int64("rows", rows),
		zap.String("sql", sql),
	}

	if requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}

	switch {
	case err != nil && l.logLevel >= gormlogger.Error:
		if l.ignoreRecordNotFoundError && errors.Is(err, gormlogger.ErrRecordNotFound) {
			return
		}
		fields = append(fields, zap.Error(err))
		l.logger.Error("SQL Error", fields...)

	case elapsed > l.slowThreshold && l.slowThreshold != 0 && l.logLevel >= gormlogger.Warn:
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.slowThreshold)
		l.logger.Warn(slowLog, fields...)

	case l.logLevel >= gormlogger.Info:
		l.logger.Debug("SQL Query", fields...)
	}
}

// MapGormLogLevel maps string log level to GORM log level
func MapGormLogLevel(level string) gormlogger.LogLevel {
	switch level {
	case "silent":
		return gormlogger.Silent
	case "error":
		return gormlogger.Error
	case "warn":
		return gormlogger.Warn
	case "info", "debug":
		return gormlogger.Info
	default:
		return gormlogger.Warn
	}
}
