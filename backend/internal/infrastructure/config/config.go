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
	Log       LogConfig
	Event     EventConfig
	HTTP      HTTPConfig
	Scheduler SchedulerConfig
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
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
	MaxBodySize       int64
	RateLimitEnabled  bool
	RateLimitRequests int
	RateLimitWindow   time.Duration
	CORSAllowOrigins  []string
	CORSAllowMethods  []string
	CORSAllowHeaders  []string
	TrustedProxies    []string
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
			ReadTimeout:       v.GetDuration("http.read_timeout"),
			WriteTimeout:      v.GetDuration("http.write_timeout"),
			IdleTimeout:       v.GetDuration("http.idle_timeout"),
			MaxHeaderBytes:    v.GetInt("http.max_header_bytes"),
			MaxBodySize:       v.GetInt64("http.max_body_size"),
			RateLimitEnabled:  v.GetBool("http.rate_limit_enabled"),
			RateLimitRequests: v.GetInt("http.rate_limit_requests"),
			RateLimitWindow:   v.GetDuration("http.rate_limit_window"),
			CORSAllowOrigins:  v.GetStringSlice("http.cors_allow_origins"),
			CORSAllowMethods:  v.GetStringSlice("http.cors_allow_methods"),
			CORSAllowHeaders:  v.GetStringSlice("http.cors_allow_headers"),
			TrustedProxies:    v.GetStringSlice("http.trusted_proxies"),
		},
		Scheduler: SchedulerConfig{
			Enabled:           v.GetBool("scheduler.enabled"),
			DailyCronSchedule: v.GetString("scheduler.daily_cron_schedule"),
			MaxConcurrentJobs: v.GetInt("scheduler.max_concurrent_jobs"),
			JobTimeout:        v.GetDuration("scheduler.job_timeout"),
			RetryAttempts:     v.GetInt("scheduler.retry_attempts"),
			RetryDelay:        v.GetDuration("scheduler.retry_delay"),
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
	if len(cfg.HTTP.CORSAllowOrigins) == 0 {
		cfg.HTTP.CORSAllowOrigins = []string{"*"}
	}
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
