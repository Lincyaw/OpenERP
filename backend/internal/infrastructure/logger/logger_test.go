package logger

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "info", cfg.Level)
	assert.Equal(t, "console", cfg.Format)
	assert.Equal(t, "stdout", cfg.Output)
	assert.NotEmpty(t, cfg.TimeFormat)
}

func TestProductionConfig(t *testing.T) {
	cfg := ProductionConfig()

	assert.Equal(t, "info", cfg.Level)
	assert.Equal(t, "json", cfg.Format)
	assert.Equal(t, "stdout", cfg.Output)
	assert.NotEmpty(t, cfg.TimeFormat)
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name:    "default config",
			cfg:     DefaultConfig(),
			wantErr: false,
		},
		{
			name:    "production config",
			cfg:     ProductionConfig(),
			wantErr: false,
		},
		{
			name: "debug level",
			cfg: &Config{
				Level:      "debug",
				Format:     "console",
				Output:     "stdout",
				TimeFormat: "2006-01-02T15:04:05Z07:00",
			},
			wantErr: false,
		},
		{
			name: "json format",
			cfg: &Config{
				Level:      "info",
				Format:     "json",
				Output:     "stdout",
				TimeFormat: "2006-01-02T15:04:05Z07:00",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, logger)
		})
	}
}

func TestNewForEnvironment(t *testing.T) {
	tests := []struct {
		name string
		env  string
	}{
		{name: "development", env: "development"},
		{name: "production", env: "production"},
		{name: "unknown", env: "staging"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewForEnvironment(tt.env)

			require.NoError(t, err)
			assert.NotNil(t, logger)
		})
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		level    string
		expected zapcore.Level
	}{
		{"debug", zapcore.DebugLevel},
		{"DEBUG", zapcore.DebugLevel},
		{"info", zapcore.InfoLevel},
		{"INFO", zapcore.InfoLevel},
		{"warn", zapcore.WarnLevel},
		{"warning", zapcore.WarnLevel},
		{"error", zapcore.ErrorLevel},
		{"fatal", zapcore.FatalLevel},
		{"unknown", zapcore.InfoLevel},
		{"", zapcore.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			result := parseLevel(tt.level)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWith(t *testing.T) {
	logger, err := NewForEnvironment("development")
	require.NoError(t, err)

	childLogger := With(logger, zap.String("key", "value"))
	assert.NotNil(t, childLogger)
	assert.NotEqual(t, logger, childLogger)
}

func TestNamed(t *testing.T) {
	logger, err := NewForEnvironment("development")
	require.NoError(t, err)

	namedLogger := Named(logger, "test-component")
	assert.NotNil(t, namedLogger)
	assert.NotEqual(t, logger, namedLogger)
}

func TestSync(t *testing.T) {
	logger, err := NewForEnvironment("development")
	require.NoError(t, err)

	// Sync should not return error for stdout
	err = Sync(logger)
	// Note: Sync might return error on stdout in some environments, so we just check it doesn't panic
	_ = err
}

func TestCreateWriter(t *testing.T) {
	tests := []struct {
		name   string
		output string
	}{
		{"stdout", "stdout"},
		{"stderr", "stderr"},
		{"STDOUT", "STDOUT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := createWriter(tt.output)
			assert.NotNil(t, writer)
		})
	}
}

func TestCreateWriterFile(t *testing.T) {
	// Create a temp file
	tmpFile, err := os.CreateTemp("", "test-log-*.log")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	writer := createWriter(tmpFile.Name())
	assert.NotNil(t, writer)
}

func TestLogOutput(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer

	// Create encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		MessageKey:     "msg",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(&buf),
		zapcore.InfoLevel,
	)
	logger := zap.New(core)

	// Log a message
	logger.Info("test message", zap.String("key", "value"))

	// Parse the output
	var output map[string]any
	err := json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)

	assert.Equal(t, "test message", output["msg"])
	assert.Equal(t, "info", output["level"])
	assert.Equal(t, "value", output["key"])
}

func TestConsoleEncoder(t *testing.T) {
	cfg := &Config{
		Level:      "info",
		Format:     "console",
		Output:     "stdout",
		TimeFormat: "2006-01-02T15:04:05Z07:00",
	}

	encoder := createEncoder(cfg)
	assert.NotNil(t, encoder)
}

func TestJSONEncoder(t *testing.T) {
	cfg := &Config{
		Level:      "info",
		Format:     "json",
		Output:     "stdout",
		TimeFormat: "2006-01-02T15:04:05Z07:00",
	}

	encoder := createEncoder(cfg)
	assert.NotNil(t, encoder)
}

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer

	encoderConfig := zapcore.EncoderConfig{
		LevelKey:    "level",
		MessageKey:  "msg",
		EncodeLevel: zapcore.LowercaseLevelEncoder,
	}

	// Test debug level - should log debug messages
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(&buf),
		zapcore.DebugLevel,
	)
	logger := zap.New(core)

	logger.Debug("debug message")
	assert.True(t, strings.Contains(buf.String(), "debug message"))

	buf.Reset()

	// Test with info level - should not log debug messages
	core = zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(&buf),
		zapcore.InfoLevel,
	)
	logger = zap.New(core)

	logger.Debug("debug message")
	assert.False(t, strings.Contains(buf.String(), "debug message"))

	logger.Info("info message")
	assert.True(t, strings.Contains(buf.String(), "info message"))
}
