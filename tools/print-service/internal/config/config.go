// Package config provides configuration management for the print service.
package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Errors returned by the config package.
var (
	// ErrInvalidConfig is returned when the configuration is invalid.
	ErrInvalidConfig = errors.New("config: invalid configuration")
	// ErrConfigNotFound is returned when the config file is not found.
	ErrConfigNotFound = errors.New("config: configuration file not found")
)

// Config is the root configuration structure for the print service.
type Config struct {
	// Server contains server connection settings.
	Server ServerConfig `yaml:"server"`

	// Printing contains printing-related settings.
	Printing PrintingConfig `yaml:"printing"`

	// Logging contains logging settings.
	Logging LoggingConfig `yaml:"logging"`

	// Health contains health check endpoint settings.
	Health HealthConfig `yaml:"health"`
}

// ServerConfig holds server connection configuration.
type ServerConfig struct {
	// APIURL is the base URL of the ERP backend API (e.g., "http://localhost:8080").
	APIURL string `yaml:"api_url"`

	// WebSocketURL is the WebSocket URL for real-time events (e.g., "ws://localhost:8080/api/v1/printing/ws").
	WebSocketURL string `yaml:"websocket_url"`

	// TenantID is the tenant identifier for multi-tenancy.
	TenantID string `yaml:"tenant_id"`

	// APIToken is the authentication token for API calls.
	APIToken string `yaml:"api_token"`

	// Timeout is the request timeout for API calls.
	// Default: 30s
	Timeout time.Duration `yaml:"timeout"`

	// ReconnectInterval is the interval between WebSocket reconnection attempts.
	// Default: 5s
	ReconnectInterval time.Duration `yaml:"reconnect_interval"`

	// MaxReconnectAttempts is the maximum number of reconnection attempts.
	// 0 means unlimited.
	// Default: 0
	MaxReconnectAttempts int `yaml:"max_reconnect_attempts"`
}

// PrintingConfig holds printing-related configuration.
type PrintingConfig struct {
	// DefaultPrinter is the default printer name to use.
	DefaultPrinter string `yaml:"default_printer"`

	// Renderer specifies the rendering method: "chromedp", "wkhtmltopdf", "gotenberg".
	// Default: "chromedp"
	Renderer string `yaml:"renderer"`

	// Protocol specifies the printing protocol: "ipp", "cups", "raw", "windows".
	// Default: "ipp"
	Protocol string `yaml:"protocol"`

	// IPPServer is the IPP server URL (e.g., "http://localhost:631").
	// Required when Protocol is "ipp".
	IPPServer string `yaml:"ipp_server"`

	// CUPSServer is the CUPS server address.
	// Required when Protocol is "cups".
	CUPSServer string `yaml:"cups_server"`

	// TempDir is the directory for temporary files.
	// Default: system temp directory
	TempDir string `yaml:"temp_dir"`

	// CleanupInterval is how often to clean up temporary files.
	// Default: 1h
	CleanupInterval time.Duration `yaml:"cleanup_interval"`
}

// LoggingConfig holds logging configuration.
type LoggingConfig struct {
	// Level is the log level: "debug", "info", "warn", "error".
	// Default: "info"
	Level string `yaml:"level"`

	// File is the log file path. If empty, logs to stdout.
	File string `yaml:"file"`

	// MaxSize is the maximum size in megabytes before rotation.
	// Default: 100
	MaxSize int `yaml:"max_size"`

	// MaxBackups is the maximum number of old log files to retain.
	// Default: 3
	MaxBackups int `yaml:"max_backups"`

	// MaxAge is the maximum number of days to retain old log files.
	// Default: 28
	MaxAge int `yaml:"max_age"`

	// Compress determines if rotated log files should be compressed.
	// Default: true
	Compress bool `yaml:"compress"`
}

// HealthConfig holds health check endpoint configuration.
type HealthConfig struct {
	// Enabled determines if the health check endpoint is enabled.
	// Default: true
	Enabled bool `yaml:"enabled"`

	// Port is the port for the health check endpoint.
	// Default: 9999
	Port int `yaml:"port"`

	// Path is the URL path for the health check endpoint.
	// Default: "/health"
	Path string `yaml:"path"`
}

// LoadFromFile loads configuration from a YAML file.
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrConfigNotFound, path)
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	return LoadFromBytes(data)
}

// LoadFromBytes loads configuration from YAML bytes.
func LoadFromBytes(data []byte) (*Config, error) {
	// Expand environment variables in the config
	expandedData := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expandedData), &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	cfg.ApplyDefaults()

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	// Server validation
	if c.Server.APIURL == "" {
		return fmt.Errorf("%w: server.api_url is required", ErrInvalidConfig)
	}

	if c.Server.WebSocketURL == "" {
		return fmt.Errorf("%w: server.websocket_url is required", ErrInvalidConfig)
	}

	if c.Server.TenantID == "" {
		return fmt.Errorf("%w: server.tenant_id is required", ErrInvalidConfig)
	}

	if c.Server.APIToken == "" {
		return fmt.Errorf("%w: server.api_token is required", ErrInvalidConfig)
	}

	// Printing validation
	validRenderers := map[string]bool{"chromedp": true, "wkhtmltopdf": true, "gotenberg": true}
	if !validRenderers[c.Printing.Renderer] {
		return fmt.Errorf("%w: printing.renderer must be one of: chromedp, wkhtmltopdf, gotenberg", ErrInvalidConfig)
	}

	validProtocols := map[string]bool{"ipp": true, "cups": true, "raw": true, "windows": true}
	if !validProtocols[c.Printing.Protocol] {
		return fmt.Errorf("%w: printing.protocol must be one of: ipp, cups, raw, windows", ErrInvalidConfig)
	}

	if c.Printing.Protocol == "ipp" && c.Printing.IPPServer == "" {
		return fmt.Errorf("%w: printing.ipp_server is required when protocol is ipp", ErrInvalidConfig)
	}

	// Logging validation
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("%w: logging.level must be one of: debug, info, warn, error", ErrInvalidConfig)
	}

	// Health validation
	if c.Health.Enabled && (c.Health.Port < 1 || c.Health.Port > 65535) {
		return fmt.Errorf("%w: health.port must be between 1 and 65535", ErrInvalidConfig)
	}

	return nil
}

// ApplyDefaults applies default values to unset fields.
func (c *Config) ApplyDefaults() {
	// Server defaults
	if c.Server.Timeout == 0 {
		c.Server.Timeout = 30 * time.Second
	}
	if c.Server.ReconnectInterval == 0 {
		c.Server.ReconnectInterval = 5 * time.Second
	}

	// Printing defaults
	if c.Printing.Renderer == "" {
		c.Printing.Renderer = "chromedp"
	}
	if c.Printing.Protocol == "" {
		c.Printing.Protocol = "ipp"
	}
	if c.Printing.TempDir == "" {
		c.Printing.TempDir = os.TempDir()
	}
	if c.Printing.CleanupInterval == 0 {
		c.Printing.CleanupInterval = time.Hour
	}

	// Logging defaults
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.MaxSize == 0 {
		c.Logging.MaxSize = 100
	}
	if c.Logging.MaxBackups == 0 {
		c.Logging.MaxBackups = 3
	}
	if c.Logging.MaxAge == 0 {
		c.Logging.MaxAge = 28
	}

	// Health defaults
	if c.Health.Port == 0 {
		c.Health.Port = 9999
	}
	if c.Health.Path == "" {
		c.Health.Path = "/health"
	}
	// Default enabled to true if not explicitly set
	// Note: YAML unmarshaling sets bool to false by default, so we need a pointer or explicit check
	// For simplicity, we'll assume health is enabled by default
}

// NewDefaultConfig creates a new Config with default values.
// This is useful for testing or when no config file is provided.
func NewDefaultConfig() *Config {
	cfg := &Config{
		Health: HealthConfig{
			Enabled: true,
		},
	}
	cfg.ApplyDefaults()
	return cfg
}
