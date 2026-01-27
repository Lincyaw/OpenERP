package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	App       AppConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	JWT       JWTConfig
	Cookie    CookieConfig
	Log       LogConfig
	Event     EventConfig
	HTTP      HTTPConfig
	Scheduler SchedulerConfig
	StockLock StockLockConfig
	Swagger   SwaggerConfig
	Telemetry TelemetryConfig
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
	Secret                 string
	AccessTokenExpiration  time.Duration
	RefreshTokenExpiration time.Duration
	Issuer                 string
	RefreshSecret          string
	MaxRefreshCount        int
	ExpirationHours        int // Deprecated: use AccessTokenExpiration instead
}

// CookieConfig holds cookie settings for refresh token
type CookieConfig struct {
	Domain   string // Domain for cookies (empty = current domain)
	Path     string // Path for cookies
	Secure   bool   // Secure flag (should be true in production for HTTPS)
	SameSite string // SameSite policy: "strict", "lax", or "none"
}

// EventConfig holds event processing configuration
type EventConfig struct {
	ProcessorEnabled bool
	BatchSize        int
	PollInterval     time.Duration
	MaxRetries       int
	CleanupEnabled   bool
	CleanupRetention time.Duration
}

// HTTPConfig holds HTTP server configuration
type HTTPConfig struct {
	ReadTimeout           time.Duration
	WriteTimeout          time.Duration
	IdleTimeout           time.Duration
	MaxHeaderBytes        int
	MaxBodySize           int64
	RateLimitEnabled      bool
	RateLimitRequests     int
	RateLimitWindow       time.Duration
	AuthRateLimitEnabled  bool          // Enable stricter rate limiting for auth endpoints
	AuthRateLimitRequests int           // Max auth attempts (default: 5)
	AuthRateLimitWindow   time.Duration // Auth rate limit window (default: 1 minute)
	CORSAllowOrigins      []string
	CORSAllowMethods      []string
	CORSAllowHeaders      []string
	TrustedProxies        []string
}

// SchedulerConfig holds report scheduler configuration
type SchedulerConfig struct {
	Enabled           bool
	DailyCronSchedule string
	MaxConcurrentJobs int
	JobTimeout        time.Duration
	RetryAttempts     int
	RetryDelay        time.Duration
}

// StockLockConfig holds stock lock expiration configuration
type StockLockConfig struct {
	CheckInterval      time.Duration // How often to check for expired locks
	DefaultExpiration  time.Duration // Default lock expiration (24h as per spec)
	AutoReleaseEnabled bool          // Whether to auto-release expired locks
}

// SwaggerConfig holds Swagger documentation endpoint configuration
type SwaggerConfig struct {
	Enabled     bool     // Whether to enable Swagger endpoint
	RequireAuth bool     // Require authentication to access Swagger
	AllowedIPs  []string // IP whitelist (empty = allow all)
}

// TelemetryConfig holds OpenTelemetry configuration
type TelemetryConfig struct {
	Enabled           bool    // Whether to enable OpenTelemetry
	CollectorEndpoint string  // OTEL Collector endpoint (e.g., "localhost:4317")
	SamplingRatio     float64 // Sampling ratio (0.0-1.0, 1.0 = 100%)
	ServiceName       string  // Service name for traces
	Insecure          bool    // Use insecure (non-TLS) connection (development only)
	// Database tracing options
	DBTraceEnabled    bool          // Enable database query tracing (otelgorm)
	DBLogFullSQL      bool          // Log full SQL statements (dev only, disable in prod for security)
	DBSlowQueryThresh time.Duration // Slow query threshold for warnings (default: 200ms)
}

// Load loads configuration from TOML file and environment variables
// Priority (highest to lowest):
// 1. Environment variables with ERP_ prefix (e.g., ERP_DATABASE_PASSWORD)
// 2. config.toml
// 3. Built-in defaults
func Load() (*Config, error) {
	v := viper.New()

	// Set config file settings
	v.SetConfigName("config")
	v.SetConfigType("toml")
	v.AddConfigPath(".")
	v.AddConfigPath("./backend")
	v.AddConfigPath("/app")

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found is OK, we'll use defaults and env vars
	}

	// Enable environment variable override
	v.SetEnvPrefix("ERP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Build config struct
	cfg := &Config{
		App: AppConfig{
			Name: v.GetString("app.name"),
			Env:  v.GetString("app.env"),
			Port: v.GetString("app.port"),
		},
		Database: DatabaseConfig{
			Host:            v.GetString("database.host"),
			Port:            v.GetInt("database.port"),
			User:            v.GetString("database.user"),
			Password:        v.GetString("database.password"),
			DBName:          v.GetString("database.dbname"),
			SSLMode:         v.GetString("database.sslmode"),
			MaxOpenConns:    v.GetInt("database.max_open_conns"),
			MaxIdleConns:    v.GetInt("database.max_idle_conns"),
			ConnMaxLifetime: v.GetInt("database.conn_max_lifetime"),
			ConnMaxIdleTime: v.GetInt("database.conn_max_idle_time"),
		},
		Redis: RedisConfig{
			Host:     v.GetString("redis.host"),
			Port:     v.GetInt("redis.port"),
			Password: v.GetString("redis.password"),
			DB:       v.GetInt("redis.db"),
		},
		JWT: JWTConfig{
			Secret:                 v.GetString("jwt.secret"),
			AccessTokenExpiration:  v.GetDuration("jwt.access_token_expiration"),
			RefreshTokenExpiration: v.GetDuration("jwt.refresh_token_expiration"),
			Issuer:                 v.GetString("jwt.issuer"),
			RefreshSecret:          v.GetString("jwt.refresh_secret"),
			MaxRefreshCount:        v.GetInt("jwt.max_refresh_count"),
			ExpirationHours:        v.GetInt("jwt.expiration_hours"),
		},
		Cookie: CookieConfig{
			Domain:   v.GetString("cookie.domain"),
			Path:     v.GetString("cookie.path"),
			Secure:   v.GetBool("cookie.secure"),
			SameSite: v.GetString("cookie.same_site"),
		},
		Log: LogConfig{
			Level:  v.GetString("log.level"),
			Format: v.GetString("log.format"),
			Output: v.GetString("log.output"),
		},
		Event: EventConfig{
			ProcessorEnabled: v.GetBool("event.processor_enabled"),
			BatchSize:        v.GetInt("event.batch_size"),
			PollInterval:     v.GetDuration("event.poll_interval"),
			MaxRetries:       v.GetInt("event.max_retries"),
			CleanupEnabled:   v.GetBool("event.cleanup_enabled"),
			CleanupRetention: v.GetDuration("event.cleanup_retention"),
		},
		HTTP: HTTPConfig{
			ReadTimeout:           v.GetDuration("http.read_timeout"),
			WriteTimeout:          v.GetDuration("http.write_timeout"),
			IdleTimeout:           v.GetDuration("http.idle_timeout"),
			MaxHeaderBytes:        v.GetInt("http.max_header_bytes"),
			MaxBodySize:           v.GetInt64("http.max_body_size"),
			RateLimitEnabled:      v.GetBool("http.rate_limit_enabled"),
			RateLimitRequests:     v.GetInt("http.rate_limit_requests"),
			RateLimitWindow:       v.GetDuration("http.rate_limit_window"),
			AuthRateLimitEnabled:  v.GetBool("http.auth_rate_limit_enabled"),
			AuthRateLimitRequests: v.GetInt("http.auth_rate_limit_requests"),
			AuthRateLimitWindow:   v.GetDuration("http.auth_rate_limit_window"),
			CORSAllowOrigins:      v.GetStringSlice("http.cors_allow_origins"),
			CORSAllowMethods:      v.GetStringSlice("http.cors_allow_methods"),
			CORSAllowHeaders:      v.GetStringSlice("http.cors_allow_headers"),
			TrustedProxies:        v.GetStringSlice("http.trusted_proxies"),
		},
		Scheduler: SchedulerConfig{
			Enabled:           v.GetBool("scheduler.enabled"),
			DailyCronSchedule: v.GetString("scheduler.daily_cron_schedule"),
			MaxConcurrentJobs: v.GetInt("scheduler.max_concurrent_jobs"),
			JobTimeout:        v.GetDuration("scheduler.job_timeout"),
			RetryAttempts:     v.GetInt("scheduler.retry_attempts"),
			RetryDelay:        v.GetDuration("scheduler.retry_delay"),
		},
		StockLock: StockLockConfig{
			CheckInterval:      v.GetDuration("stock_lock.check_interval"),
			DefaultExpiration:  v.GetDuration("stock_lock.default_expiration"),
			AutoReleaseEnabled: v.GetBool("stock_lock.auto_release_enabled"),
		},
		Swagger: SwaggerConfig{
			Enabled:     v.GetBool("swagger.enabled"),
			RequireAuth: v.GetBool("swagger.require_auth"),
			AllowedIPs:  v.GetStringSlice("swagger.allowed_ips"),
		},
		Telemetry: TelemetryConfig{
			Enabled:           v.GetBool("telemetry.enabled"),
			CollectorEndpoint: v.GetString("telemetry.collector_endpoint"),
			SamplingRatio:     v.GetFloat64("telemetry.sampling_ratio"),
			ServiceName:       v.GetString("telemetry.service_name"),
			Insecure:          v.GetBool("telemetry.insecure"),
			DBTraceEnabled:    v.GetBool("telemetry.db_trace_enabled"),
			DBLogFullSQL:      v.GetBool("telemetry.db_log_full_sql"),
			DBSlowQueryThresh: v.GetDuration("telemetry.db_slow_query_threshold"),
		},
	}

	// Apply defaults for empty values
	applyDefaults(cfg)

	// Validate configuration
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// applyDefaults sets default values for any empty config fields
func applyDefaults(cfg *Config) {
	if cfg.App.Name == "" {
		cfg.App.Name = "erp-backend"
	}
	if cfg.App.Env == "" {
		cfg.App.Env = "development"
	}
	if cfg.App.Port == "" {
		cfg.App.Port = "8080"
	}
	if cfg.Database.Host == "" {
		cfg.Database.Host = "localhost"
	}
	if cfg.Database.Port == 0 {
		cfg.Database.Port = 5432
	}
	if cfg.Database.User == "" {
		cfg.Database.User = "postgres"
	}
	if cfg.Database.DBName == "" {
		cfg.Database.DBName = "erp"
	}
	if cfg.Database.SSLMode == "" {
		cfg.Database.SSLMode = "disable"
	}
	if cfg.Database.MaxOpenConns == 0 {
		cfg.Database.MaxOpenConns = 25
	}
	if cfg.Database.MaxIdleConns == 0 {
		cfg.Database.MaxIdleConns = 5
	}
	if cfg.Database.ConnMaxLifetime == 0 {
		cfg.Database.ConnMaxLifetime = 60
	}
	if cfg.Database.ConnMaxIdleTime == 0 {
		cfg.Database.ConnMaxIdleTime = 30
	}
	if cfg.Redis.Host == "" {
		cfg.Redis.Host = "localhost"
	}
	if cfg.Redis.Port == 0 {
		cfg.Redis.Port = 6379
	}
	if cfg.JWT.AccessTokenExpiration == 0 {
		cfg.JWT.AccessTokenExpiration = 15 * time.Minute
	}
	if cfg.JWT.RefreshTokenExpiration == 0 {
		cfg.JWT.RefreshTokenExpiration = 168 * time.Hour
	}
	if cfg.JWT.Issuer == "" {
		cfg.JWT.Issuer = "erp-backend"
	}
	if cfg.JWT.MaxRefreshCount == 0 {
		cfg.JWT.MaxRefreshCount = 10
	}
	// Cookie defaults
	if cfg.Cookie.Path == "" {
		cfg.Cookie.Path = "/"
	}
	if cfg.Cookie.SameSite == "" {
		cfg.Cookie.SameSite = "lax"
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	if cfg.Log.Format == "" {
		cfg.Log.Format = "console"
	}
	if cfg.Log.Output == "" {
		cfg.Log.Output = "stdout"
	}
	if cfg.Event.BatchSize == 0 {
		cfg.Event.BatchSize = 100
	}
	if cfg.Event.PollInterval == 0 {
		cfg.Event.PollInterval = 5 * time.Second
	}
	if cfg.Event.MaxRetries == 0 {
		cfg.Event.MaxRetries = 5
	}
	if cfg.Event.CleanupRetention == 0 {
		cfg.Event.CleanupRetention = 168 * time.Hour
	}
	if cfg.HTTP.ReadTimeout == 0 {
		cfg.HTTP.ReadTimeout = 15 * time.Second
	}
	if cfg.HTTP.WriteTimeout == 0 {
		cfg.HTTP.WriteTimeout = 15 * time.Second
	}
	if cfg.HTTP.IdleTimeout == 0 {
		cfg.HTTP.IdleTimeout = 60 * time.Second
	}
	if cfg.HTTP.MaxHeaderBytes == 0 {
		cfg.HTTP.MaxHeaderBytes = 1 << 20 // 1MB
	}
	if cfg.HTTP.MaxBodySize == 0 {
		cfg.HTTP.MaxBodySize = 10 << 20 // 10MB
	}
	if cfg.HTTP.RateLimitRequests == 0 {
		cfg.HTTP.RateLimitRequests = 100
	}
	if cfg.HTTP.RateLimitWindow == 0 {
		cfg.HTTP.RateLimitWindow = time.Minute
	}
	// Auth rate limiting defaults - stricter limits for auth endpoints to prevent brute force
	if cfg.HTTP.AuthRateLimitRequests == 0 {
		cfg.HTTP.AuthRateLimitRequests = 5 // 5 attempts per window
	}
	if cfg.HTTP.AuthRateLimitWindow == 0 {
		cfg.HTTP.AuthRateLimitWindow = time.Minute // 1 minute window
	}
	// NOTE: CORS origins are intentionally not given a default fallback to "*".
	// An empty list means no cross-origin requests are allowed until explicitly configured.
	// This is a secure default - applications MUST configure allowed origins explicitly.
	// In development, use config.toml to set specific origins like ["http://localhost:3000"]
	if len(cfg.HTTP.CORSAllowMethods) == 0 {
		cfg.HTTP.CORSAllowMethods = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}
	}
	if len(cfg.HTTP.CORSAllowHeaders) == 0 {
		cfg.HTTP.CORSAllowHeaders = []string{"Content-Type", "Authorization", "X-Request-ID", "X-Tenant-ID"}
	}
	if cfg.Scheduler.DailyCronSchedule == "" {
		cfg.Scheduler.DailyCronSchedule = "0 2 * * *"
	}
	if cfg.Scheduler.MaxConcurrentJobs == 0 {
		cfg.Scheduler.MaxConcurrentJobs = 3
	}
	if cfg.Scheduler.JobTimeout == 0 {
		cfg.Scheduler.JobTimeout = 30 * time.Minute
	}
	if cfg.Scheduler.RetryAttempts == 0 {
		cfg.Scheduler.RetryAttempts = 3
	}
	if cfg.Scheduler.RetryDelay == 0 {
		cfg.Scheduler.RetryDelay = 5 * time.Minute
	}
	// StockLock defaults
	if cfg.StockLock.CheckInterval == 0 {
		cfg.StockLock.CheckInterval = 5 * time.Minute
	}
	if cfg.StockLock.DefaultExpiration == 0 {
		cfg.StockLock.DefaultExpiration = 24 * time.Hour // 24h as per spec
	}
	// Swagger defaults: enabled by default (will be overridden by validation in production)
	// Note: We set enabled=true here, but production validation enforces proper configuration

	// Telemetry defaults
	if cfg.Telemetry.CollectorEndpoint == "" {
		cfg.Telemetry.CollectorEndpoint = "localhost:4317" // Default gRPC endpoint
	}
	if cfg.Telemetry.SamplingRatio == 0 {
		cfg.Telemetry.SamplingRatio = 1.0 // 100% in development
	}
	if cfg.Telemetry.ServiceName == "" {
		cfg.Telemetry.ServiceName = "erp-backend"
	}
	// Note: Insecure defaults to false for safety (TLS enabled by default)
	// Database tracing defaults - enabled by default when telemetry is enabled
	// DBTraceEnabled defaults to false (needs explicit enable)
	if cfg.Telemetry.DBSlowQueryThresh == 0 {
		cfg.Telemetry.DBSlowQueryThresh = 200 * time.Millisecond // 200ms default as per spec
	}
	// Note: DBLogFullSQL defaults to false for security (disable in production)
}

// validate performs validation on the configuration
func (c *Config) validate() error {
	// Validate connection pool settings
	if c.Database.MaxOpenConns <= 0 {
		return fmt.Errorf("database.max_open_conns must be positive")
	}
	if c.Database.MaxIdleConns < 0 {
		return fmt.Errorf("database.max_idle_conns cannot be negative")
	}
	if c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		return fmt.Errorf("database.max_idle_conns (%d) cannot exceed database.max_open_conns (%d)",
			c.Database.MaxIdleConns, c.Database.MaxOpenConns)
	}

	// Production-specific validations
	if c.App.Env == "production" {
		if c.JWT.Secret == "" {
			return fmt.Errorf("jwt.secret is required in production")
		}
		if len(c.JWT.Secret) < 32 {
			return fmt.Errorf("jwt.secret must be at least 32 characters in production")
		}
		if c.Database.Password == "" {
			return fmt.Errorf("database.password is required in production")
		}
		if c.Database.SSLMode == "disable" {
			return fmt.Errorf("database.sslmode cannot be 'disable' in production")
		}
		// Cookie security for refresh token (SEC-004)
		if !c.Cookie.Secure {
			return fmt.Errorf("cookie.secure must be true in production (HTTPS required for secure cookies)")
		}
		// SameSite=None requires Secure flag
		if c.Cookie.SameSite == "none" && !c.Cookie.Secure {
			return fmt.Errorf("cookie.same_site=none requires cookie.secure=true")
		}
		// CORS must not use wildcard with credentials
		for _, origin := range c.HTTP.CORSAllowOrigins {
			if origin == "*" {
				return fmt.Errorf("cors_allow_origins cannot be '*' in production (use specific origins)")
			}
		}
		// Swagger must be disabled OR protected in production
		if c.Swagger.Enabled {
			if !c.Swagger.RequireAuth && len(c.Swagger.AllowedIPs) == 0 {
				return fmt.Errorf("swagger endpoint must be disabled, require authentication, or have IP restriction in production")
			}
		}
		// Database tracing: full SQL logging is a security risk in production
		if c.Telemetry.DBLogFullSQL {
			return fmt.Errorf("telemetry.db_log_full_sql must be false in production to prevent sensitive data exposure in traces")
		}
	}

	// Validate telemetry configuration (all environments)
	if c.Telemetry.SamplingRatio < 0.0 || c.Telemetry.SamplingRatio > 1.0 {
		return fmt.Errorf("telemetry.sampling_ratio must be between 0.0 and 1.0, got %f", c.Telemetry.SamplingRatio)
	}

	return nil
}

// DSN returns the database connection string with properly escaped values
func (d *DatabaseConfig) DSN() string {
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
