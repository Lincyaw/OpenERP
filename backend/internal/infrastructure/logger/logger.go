package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config holds logger configuration
type Config struct {
	Level      string // debug, info, warn, error
	Format     string // json, console
	Output     string // stdout, stderr, or file path
	TimeFormat string // ISO8601, RFC3339, or custom format
}

// DefaultConfig returns a default configuration suitable for development
func DefaultConfig() *Config {
	return &Config{
		Level:      "info",
		Format:     "console",
		Output:     "stdout",
		TimeFormat: "2006-01-02T15:04:05.000Z07:00",
	}
}

// ProductionConfig returns a configuration suitable for production
func ProductionConfig() *Config {
	return &Config{
		Level:      "info",
		Format:     "json",
		Output:     "stdout",
		TimeFormat: "2006-01-02T15:04:05.000Z07:00",
	}
}

// New creates a new zap logger with the given configuration
func New(cfg *Config) (*zap.Logger, error) {
	level := parseLevel(cfg.Level)
	encoder := createEncoder(cfg)
	writer := createWriter(cfg.Output)

	core := zapcore.NewCore(encoder, writer, level)
	logger := zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	return logger, nil
}

// NewForEnvironment creates a logger appropriate for the given environment
func NewForEnvironment(env string) (*zap.Logger, error) {
	var cfg *Config
	if env == "production" {
		cfg = ProductionConfig()
	} else {
		cfg = DefaultConfig()
	}
	return New(cfg)
}

// parseLevel converts a string level to zapcore.Level
func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// createEncoder creates the appropriate encoder based on format
func createEncoder(cfg *Config) zapcore.Encoder {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout(cfg.TimeFormat),
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	if cfg.Format == "console" {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		return zapcore.NewConsoleEncoder(encoderConfig)
	}

	return zapcore.NewJSONEncoder(encoderConfig)
}

// createWriter creates the appropriate writer based on output
func createWriter(output string) zapcore.WriteSyncer {
	switch strings.ToLower(output) {
	case "stdout":
		return zapcore.AddSync(os.Stdout)
	case "stderr":
		return zapcore.AddSync(os.Stderr)
	default:
		// File output - create or append to file
		file, err := os.OpenFile(output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			// Fallback to stdout if file cannot be opened
			return zapcore.AddSync(os.Stdout)
		}
		return zapcore.AddSync(file)
	}
}

// With creates a child logger with the given fields
func With(logger *zap.Logger, fields ...zap.Field) *zap.Logger {
	return logger.With(fields...)
}

// Named creates a named logger
func Named(logger *zap.Logger, name string) *zap.Logger {
	return logger.Named(name)
}

// Sync flushes any buffered log entries
func Sync(logger *zap.Logger) error {
	return logger.Sync()
}
