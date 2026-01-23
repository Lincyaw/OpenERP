package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration
type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Log      LogConfig
	Event    EventConfig
	HTTP     HTTPConfig
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level  string // debug, info, warn, error
	Format string // json, console
	Output string // stdout, stderr, or file path
}

// AppConfig holds application-specific settings
type AppConfig struct {
	Name string
	Env  string
	Port string
}

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime int // in minutes
	ConnMaxIdleTime int // in minutes
}

// RedisConfig holds Redis connection settings
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// JWTConfig holds JWT settings
type JWTConfig struct {
	Secret          string
	ExpirationHours int
}

// EventConfig holds event processing configuration
type EventConfig struct {
	ProcessorEnabled  bool
	BatchSize         int
	PollInterval      time.Duration
	MaxRetries        int
	CleanupEnabled    bool
	CleanupRetention  time.Duration
}

// HTTPConfig holds HTTP server configuration
type HTTPConfig struct {
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
	MaxBodySize       int64 // Maximum request body size in bytes
	RateLimitEnabled  bool
	RateLimitRequests int           // Requests per window
	RateLimitWindow   time.Duration // Window duration
	CORSAllowOrigins  []string
	CORSAllowMethods  []string
	CORSAllowHeaders  []string
	TrustedProxies    []string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		App: AppConfig{
			Name: getEnv("APP_NAME", "erp-backend"),
			Env:  getEnv("APP_ENV", "development"),
			Port: getEnv("APP_PORT", "8080"),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnvAsInt("DB_PORT", 5432),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", ""),
			DBName:          getEnv("DB_NAME", "erp"),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvAsInt("DB_CONN_MAX_LIFETIME", 60),
			ConnMaxIdleTime: getEnvAsInt("DB_CONN_MAX_IDLE_TIME", 30),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvAsInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:          getEnv("JWT_SECRET", ""),
			ExpirationHours: getEnvAsInt("JWT_EXPIRATION_HOURS", 24),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "console"),
			Output: getEnv("LOG_OUTPUT", "stdout"),
		},
		Event: EventConfig{
			ProcessorEnabled:  getEnvAsBool("EVENT_PROCESSOR_ENABLED", true),
			BatchSize:         getEnvAsInt("EVENT_PROCESSOR_BATCH_SIZE", 100),
			PollInterval:      getEnvAsDuration("EVENT_PROCESSOR_INTERVAL", 5*time.Second),
			MaxRetries:        getEnvAsInt("EVENT_MAX_RETRIES", 5),
			CleanupEnabled:    getEnvAsBool("EVENT_CLEANUP_ENABLED", true),
			CleanupRetention:  getEnvAsDuration("EVENT_CLEANUP_RETENTION", 168*time.Hour),
		},
		HTTP: HTTPConfig{
			ReadTimeout:       getEnvAsDuration("HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:      getEnvAsDuration("HTTP_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:       getEnvAsDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
			MaxHeaderBytes:    getEnvAsInt("HTTP_MAX_HEADER_BYTES", 1<<20), // 1MB
			MaxBodySize:       getEnvAsInt64("HTTP_MAX_BODY_SIZE", 10<<20), // 10MB
			RateLimitEnabled:  getEnvAsBool("HTTP_RATE_LIMIT_ENABLED", true),
			RateLimitRequests: getEnvAsInt("HTTP_RATE_LIMIT_REQUESTS", 100),
			RateLimitWindow:   getEnvAsDuration("HTTP_RATE_LIMIT_WINDOW", time.Minute),
			CORSAllowOrigins:  getEnvAsStringSlice("HTTP_CORS_ORIGINS", []string{"*"}),
			CORSAllowMethods:  getEnvAsStringSlice("HTTP_CORS_METHODS", []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}),
			CORSAllowHeaders:  getEnvAsStringSlice("HTTP_CORS_HEADERS", []string{"Content-Type", "Authorization", "X-Request-ID", "X-Tenant-ID"}),
			TrustedProxies:    getEnvAsStringSlice("HTTP_TRUSTED_PROXIES", nil),
		},
	}

	// Validate configuration
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate performs validation on the configuration
func (c *Config) validate() error {
	// Validate connection pool settings
	if c.Database.MaxOpenConns <= 0 {
		return fmt.Errorf("DB_MAX_OPEN_CONNS must be positive")
	}
	if c.Database.MaxIdleConns < 0 {
		return fmt.Errorf("DB_MAX_IDLE_CONNS cannot be negative")
	}
	if c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		return fmt.Errorf("DB_MAX_IDLE_CONNS (%d) cannot exceed DB_MAX_OPEN_CONNS (%d)",
			c.Database.MaxIdleConns, c.Database.MaxOpenConns)
	}

	// Production-specific validations
	if c.App.Env == "production" {
		if c.JWT.Secret == "" {
			return fmt.Errorf("JWT_SECRET is required in production")
		}
		if len(c.JWT.Secret) < 32 {
			return fmt.Errorf("JWT_SECRET must be at least 32 characters in production")
		}
		if c.Database.Password == "" {
			return fmt.Errorf("DB_PASSWORD is required in production")
		}
		if c.Database.SSLMode == "disable" {
			return fmt.Errorf("DB_SSL_MODE cannot be 'disable' in production")
		}
	}

	return nil
}

// DSN returns the database connection string with properly escaped values
func (d *DatabaseConfig) DSN() string {
	// Build URL-style DSN for proper handling of special characters
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(d.User, d.Password),
		Host:   fmt.Sprintf("%s:%d", d.Host, d.Port),
		Path:   d.DBName,
	}
	q := u.Query()
	q.Set("sslmode", d.SSLMode)
	u.RawQuery = q.Encode()
	return u.String()
}

// getEnv gets an environment variable with a default fallback
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as int with a default fallback
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsBool gets an environment variable as bool with a default fallback
func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// getEnvAsDuration gets an environment variable as duration with a default fallback
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// getEnvAsInt64 gets an environment variable as int64 with a default fallback
func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsStringSlice gets an environment variable as string slice (comma-separated)
func getEnvAsStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		parts := strings.Split(value, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultValue
}
