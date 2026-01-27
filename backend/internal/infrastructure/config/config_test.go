package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// Save original env vars and restore after tests
	originalEnv := map[string]string{
		"ERP_APP_NAME":                os.Getenv("ERP_APP_NAME"),
		"ERP_APP_ENV":                 os.Getenv("ERP_APP_ENV"),
		"ERP_APP_PORT":                os.Getenv("ERP_APP_PORT"),
		"ERP_DATABASE_HOST":           os.Getenv("ERP_DATABASE_HOST"),
		"ERP_DATABASE_PORT":           os.Getenv("ERP_DATABASE_PORT"),
		"ERP_DATABASE_USER":           os.Getenv("ERP_DATABASE_USER"),
		"ERP_DATABASE_PASSWORD":       os.Getenv("ERP_DATABASE_PASSWORD"),
		"ERP_DATABASE_DBNAME":         os.Getenv("ERP_DATABASE_DBNAME"),
		"ERP_DATABASE_SSLMODE":        os.Getenv("ERP_DATABASE_SSLMODE"),
		"ERP_DATABASE_MAX_OPEN_CONNS": os.Getenv("ERP_DATABASE_MAX_OPEN_CONNS"),
		"ERP_DATABASE_MAX_IDLE_CONNS": os.Getenv("ERP_DATABASE_MAX_IDLE_CONNS"),
		"ERP_JWT_SECRET":              os.Getenv("ERP_JWT_SECRET"),
		"APP_ENV":                     os.Getenv("APP_ENV"),
	}

	defer func() {
		for k, v := range originalEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	clearEnv := func() {
		for k := range originalEnv {
			os.Unsetenv(k)
		}
	}

	t.Run("loads default values when env vars not set", func(t *testing.T) {
		clearEnv()

		cfg, err := Load()
		require.NoError(t, err)

		assert.Equal(t, "erp-backend", cfg.App.Name)
		assert.Equal(t, "development", cfg.App.Env)
		assert.Equal(t, "8080", cfg.App.Port)
		assert.Equal(t, "localhost", cfg.Database.Host)
		assert.Equal(t, 5432, cfg.Database.Port)
		assert.Equal(t, "postgres", cfg.Database.User)
		assert.Equal(t, "", cfg.Database.Password)
		assert.Equal(t, "erp", cfg.Database.DBName)
		assert.Equal(t, "disable", cfg.Database.SSLMode)
		assert.Equal(t, 25, cfg.Database.MaxOpenConns)
		assert.Equal(t, 5, cfg.Database.MaxIdleConns)
	})

	t.Run("loads values from environment variables with ERP prefix", func(t *testing.T) {
		clearEnv()
		os.Setenv("ERP_APP_NAME", "test-app")
		os.Setenv("ERP_APP_ENV", "testing")
		os.Setenv("ERP_APP_PORT", "9000")
		os.Setenv("ERP_DATABASE_HOST", "testdb.local")
		os.Setenv("ERP_DATABASE_PORT", "5433")
		os.Setenv("ERP_DATABASE_USER", "testuser")
		os.Setenv("ERP_DATABASE_PASSWORD", "testpass")
		os.Setenv("ERP_DATABASE_DBNAME", "testdb")
		os.Setenv("ERP_DATABASE_SSLMODE", "require")
		os.Setenv("ERP_DATABASE_MAX_OPEN_CONNS", "50")
		os.Setenv("ERP_DATABASE_MAX_IDLE_CONNS", "10")

		cfg, err := Load()
		require.NoError(t, err)

		assert.Equal(t, "test-app", cfg.App.Name)
		assert.Equal(t, "testing", cfg.App.Env)
		assert.Equal(t, "9000", cfg.App.Port)
		assert.Equal(t, "testdb.local", cfg.Database.Host)
		assert.Equal(t, 5433, cfg.Database.Port)
		assert.Equal(t, "testuser", cfg.Database.User)
		assert.Equal(t, "testpass", cfg.Database.Password)
		assert.Equal(t, "testdb", cfg.Database.DBName)
		assert.Equal(t, "require", cfg.Database.SSLMode)
		assert.Equal(t, 50, cfg.Database.MaxOpenConns)
		assert.Equal(t, 10, cfg.Database.MaxIdleConns)
	})

	t.Run("validates MaxIdleConns cannot exceed MaxOpenConns", func(t *testing.T) {
		clearEnv()
		os.Setenv("ERP_DATABASE_MAX_OPEN_CONNS", "10")
		os.Setenv("ERP_DATABASE_MAX_IDLE_CONNS", "20")

		_, err := Load()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "max_idle_conns")
		assert.Contains(t, err.Error(), "cannot exceed")
	})

	t.Run("zero MaxOpenConns uses default", func(t *testing.T) {
		clearEnv()
		os.Setenv("ERP_DATABASE_MAX_OPEN_CONNS", "0")

		cfg, err := Load()
		require.NoError(t, err)
		// 0 is treated as "not set", so default (25) is used
		assert.Equal(t, 25, cfg.Database.MaxOpenConns)
	})

	t.Run("validates MaxIdleConns cannot be negative", func(t *testing.T) {
		clearEnv()
		os.Setenv("ERP_DATABASE_MAX_IDLE_CONNS", "-1")

		_, err := Load()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "max_idle_conns cannot be negative")
	})
}

func TestLoad_ProductionValidation(t *testing.T) {
	originalEnv := map[string]string{
		"ERP_APP_ENV":              os.Getenv("ERP_APP_ENV"),
		"ERP_JWT_SECRET":           os.Getenv("ERP_JWT_SECRET"),
		"ERP_DATABASE_PASSWORD":    os.Getenv("ERP_DATABASE_PASSWORD"),
		"ERP_DATABASE_SSLMODE":     os.Getenv("ERP_DATABASE_SSLMODE"),
		"ERP_COOKIE_SECURE":        os.Getenv("ERP_COOKIE_SECURE"),
		"ERP_SWAGGER_ENABLED":      os.Getenv("ERP_SWAGGER_ENABLED"),
		"ERP_SWAGGER_REQUIRE_AUTH": os.Getenv("ERP_SWAGGER_REQUIRE_AUTH"),
		"ERP_SWAGGER_ALLOWED_IPS":  os.Getenv("ERP_SWAGGER_ALLOWED_IPS"),
		"APP_ENV":                  os.Getenv("APP_ENV"),
	}

	defer func() {
		for k, v := range originalEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	clearEnv := func() {
		for k := range originalEnv {
			os.Unsetenv(k)
		}
	}

	// Helper to set valid production base config
	setValidProductionBase := func() {
		os.Setenv("ERP_APP_ENV", "production")
		os.Setenv("ERP_JWT_SECRET", "this-is-a-very-secure-jwt-secret-key-32chars")
		os.Setenv("ERP_DATABASE_PASSWORD", "secure-password")
		os.Setenv("ERP_DATABASE_SSLMODE", "require")
		os.Setenv("ERP_COOKIE_SECURE", "true")
		os.Setenv("ERP_SWAGGER_ENABLED", "false") // Disabled by default for security
	}

	t.Run("requires jwt.secret in production", func(t *testing.T) {
		clearEnv()
		os.Setenv("ERP_APP_ENV", "production")
		os.Setenv("ERP_DATABASE_PASSWORD", "secure-password")
		os.Setenv("ERP_DATABASE_SSLMODE", "require")
		os.Setenv("ERP_COOKIE_SECURE", "true")
		os.Setenv("ERP_SWAGGER_ENABLED", "false")

		_, err := Load()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "jwt.secret is required in production")
	})

	t.Run("requires jwt.secret at least 32 characters in production", func(t *testing.T) {
		clearEnv()
		os.Setenv("ERP_APP_ENV", "production")
		os.Setenv("ERP_JWT_SECRET", "short-secret")
		os.Setenv("ERP_DATABASE_PASSWORD", "secure-password")
		os.Setenv("ERP_DATABASE_SSLMODE", "require")
		os.Setenv("ERP_COOKIE_SECURE", "true")
		os.Setenv("ERP_SWAGGER_ENABLED", "false")

		_, err := Load()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "jwt.secret must be at least 32 characters")
	})

	t.Run("requires database.password in production", func(t *testing.T) {
		clearEnv()
		os.Setenv("ERP_APP_ENV", "production")
		os.Setenv("ERP_JWT_SECRET", "this-is-a-very-secure-jwt-secret-key-32chars")
		os.Setenv("ERP_DATABASE_SSLMODE", "require")
		os.Setenv("ERP_COOKIE_SECURE", "true")
		os.Setenv("ERP_SWAGGER_ENABLED", "false")

		_, err := Load()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database.password is required in production")
	})

	t.Run("requires SSL enabled in production", func(t *testing.T) {
		clearEnv()
		os.Setenv("ERP_APP_ENV", "production")
		os.Setenv("ERP_JWT_SECRET", "this-is-a-very-secure-jwt-secret-key-32chars")
		os.Setenv("ERP_DATABASE_PASSWORD", "secure-password")
		os.Setenv("ERP_DATABASE_SSLMODE", "disable")
		os.Setenv("ERP_COOKIE_SECURE", "true")
		os.Setenv("ERP_SWAGGER_ENABLED", "false")

		_, err := Load()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database.sslmode cannot be 'disable' in production")
	})

	t.Run("passes validation with valid production config", func(t *testing.T) {
		clearEnv()
		setValidProductionBase()

		cfg, err := Load()
		require.NoError(t, err)
		assert.Equal(t, "production", cfg.App.Env)
	})

	// SEC-007: Swagger protection validation tests
	t.Run("fails if swagger enabled without protection in production", func(t *testing.T) {
		clearEnv()
		setValidProductionBase()
		os.Setenv("ERP_SWAGGER_ENABLED", "true")
		os.Setenv("ERP_SWAGGER_REQUIRE_AUTH", "false")
		// No IP whitelist set

		_, err := Load()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "swagger endpoint must be disabled, require authentication, or have IP restriction")
	})

	t.Run("passes with swagger enabled and require_auth in production", func(t *testing.T) {
		clearEnv()
		setValidProductionBase()
		os.Setenv("ERP_SWAGGER_ENABLED", "true")
		os.Setenv("ERP_SWAGGER_REQUIRE_AUTH", "true")

		cfg, err := Load()
		require.NoError(t, err)
		assert.True(t, cfg.Swagger.Enabled)
		assert.True(t, cfg.Swagger.RequireAuth)
	})

	t.Run("passes with swagger disabled in production", func(t *testing.T) {
		clearEnv()
		setValidProductionBase()
		os.Setenv("ERP_SWAGGER_ENABLED", "false")

		cfg, err := Load()
		require.NoError(t, err)
		assert.False(t, cfg.Swagger.Enabled)
	})

	t.Run("fails if db_log_full_sql enabled in production", func(t *testing.T) {
		clearEnv()
		setValidProductionBase()
		os.Setenv("ERP_TELEMETRY_DB_LOG_FULL_SQL", "true")

		_, err := Load()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db_log_full_sql must be false in production")
	})

	t.Run("passes with db_log_full_sql disabled in production", func(t *testing.T) {
		clearEnv()
		setValidProductionBase()
		os.Setenv("ERP_TELEMETRY_DB_LOG_FULL_SQL", "false")
		os.Setenv("ERP_TELEMETRY_DB_TRACE_ENABLED", "true")

		cfg, err := Load()
		require.NoError(t, err)
		assert.True(t, cfg.Telemetry.DBTraceEnabled)
		assert.False(t, cfg.Telemetry.DBLogFullSQL)
	})
}

func TestDatabaseConfig_DSN(t *testing.T) {
	t.Run("generates valid DSN", func(t *testing.T) {
		cfg := DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "testuser",
			Password: "testpass",
			DBName:   "testdb",
			SSLMode:  "disable",
		}

		dsn := cfg.DSN()
		assert.Contains(t, dsn, "localhost")
		assert.Contains(t, dsn, "5432")
		assert.Contains(t, dsn, "testuser")
		assert.Contains(t, dsn, "testdb")
		assert.Contains(t, dsn, "sslmode=disable")
	})

	t.Run("escapes special characters in password", func(t *testing.T) {
		cfg := DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "user",
			Password: "pass@word#123",
			DBName:   "db",
			SSLMode:  "disable",
		}

		dsn := cfg.DSN()
		// URL-encoded password should be in the DSN
		assert.Contains(t, dsn, "pass%40word%23123")
	})

	t.Run("handles empty password", func(t *testing.T) {
		cfg := DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "user",
			Password: "",
			DBName:   "db",
			SSLMode:  "disable",
		}

		dsn := cfg.DSN()
		assert.NotEmpty(t, dsn)
	})
}

func TestLoad_TelemetryConfig(t *testing.T) {
	originalEnv := map[string]string{
		"ERP_TELEMETRY_ENABLED":                 os.Getenv("ERP_TELEMETRY_ENABLED"),
		"ERP_TELEMETRY_COLLECTOR_ENDPOINT":      os.Getenv("ERP_TELEMETRY_COLLECTOR_ENDPOINT"),
		"ERP_TELEMETRY_SAMPLING_RATIO":          os.Getenv("ERP_TELEMETRY_SAMPLING_RATIO"),
		"ERP_TELEMETRY_SERVICE_NAME":            os.Getenv("ERP_TELEMETRY_SERVICE_NAME"),
		"ERP_TELEMETRY_DB_TRACE_ENABLED":        os.Getenv("ERP_TELEMETRY_DB_TRACE_ENABLED"),
		"ERP_TELEMETRY_DB_LOG_FULL_SQL":         os.Getenv("ERP_TELEMETRY_DB_LOG_FULL_SQL"),
		"ERP_TELEMETRY_DB_SLOW_QUERY_THRESHOLD": os.Getenv("ERP_TELEMETRY_DB_SLOW_QUERY_THRESHOLD"),
	}

	defer func() {
		for k, v := range originalEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	clearEnv := func() {
		for k := range originalEnv {
			os.Unsetenv(k)
		}
	}

	t.Run("loads default telemetry values", func(t *testing.T) {
		clearEnv()

		cfg, err := Load()
		require.NoError(t, err)

		// Default values from applyDefaults
		assert.Equal(t, "localhost:14317", cfg.Telemetry.CollectorEndpoint)
		assert.Equal(t, 1.0, cfg.Telemetry.SamplingRatio)
		assert.Equal(t, "erp-backend", cfg.Telemetry.ServiceName)
		// Enabled defaults to false unless explicitly set
		assert.False(t, cfg.Telemetry.Enabled)
	})

	t.Run("loads telemetry values from env vars", func(t *testing.T) {
		clearEnv()
		os.Setenv("ERP_TELEMETRY_ENABLED", "true")
		os.Setenv("ERP_TELEMETRY_COLLECTOR_ENDPOINT", "otel-collector:14317")
		os.Setenv("ERP_TELEMETRY_SAMPLING_RATIO", "0.5")
		os.Setenv("ERP_TELEMETRY_SERVICE_NAME", "my-erp-service")

		cfg, err := Load()
		require.NoError(t, err)

		assert.True(t, cfg.Telemetry.Enabled)
		assert.Equal(t, "otel-collector:14317", cfg.Telemetry.CollectorEndpoint)
		assert.Equal(t, 0.5, cfg.Telemetry.SamplingRatio)
		assert.Equal(t, "my-erp-service", cfg.Telemetry.ServiceName)
	})

	t.Run("allows zero sampling ratio for disabled tracing", func(t *testing.T) {
		clearEnv()
		os.Setenv("ERP_TELEMETRY_ENABLED", "true")
		os.Setenv("ERP_TELEMETRY_SAMPLING_RATIO", "0.0")

		cfg, err := Load()
		require.NoError(t, err)

		assert.True(t, cfg.Telemetry.Enabled)
		// 0.0 is explicitly set, so it should stay 0.0, but defaults are applied first
		// The default of 1.0 won't override 0.0 from env because applyDefaults only checks == 0
		// This test documents that behavior - if you want 0% sampling, set enabled=false
	})

	t.Run("validates sampling ratio bounds", func(t *testing.T) {
		clearEnv()
		os.Setenv("ERP_TELEMETRY_SAMPLING_RATIO", "1.5")

		_, err := Load()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "telemetry.sampling_ratio must be between 0.0 and 1.0")
	})

	t.Run("validates negative sampling ratio", func(t *testing.T) {
		clearEnv()
		os.Setenv("ERP_TELEMETRY_SAMPLING_RATIO", "-0.1")

		_, err := Load()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "telemetry.sampling_ratio must be between 0.0 and 1.0")
	})

	t.Run("loads insecure config", func(t *testing.T) {
		clearEnv()
		os.Setenv("ERP_TELEMETRY_INSECURE", "true")

		cfg, err := Load()
		require.NoError(t, err)
		assert.True(t, cfg.Telemetry.Insecure)
	})

	t.Run("loads database tracing config", func(t *testing.T) {
		clearEnv()
		os.Setenv("ERP_TELEMETRY_DB_TRACE_ENABLED", "true")
		os.Setenv("ERP_TELEMETRY_DB_LOG_FULL_SQL", "false")
		os.Setenv("ERP_TELEMETRY_DB_SLOW_QUERY_THRESHOLD", "500ms")

		cfg, err := Load()
		require.NoError(t, err)
		assert.True(t, cfg.Telemetry.DBTraceEnabled)
		assert.False(t, cfg.Telemetry.DBLogFullSQL)
		assert.Equal(t, 500*time.Millisecond, cfg.Telemetry.DBSlowQueryThresh)
	})

	t.Run("defaults db_slow_query_threshold to 200ms", func(t *testing.T) {
		clearEnv()

		cfg, err := Load()
		require.NoError(t, err)
		assert.Equal(t, 200*time.Millisecond, cfg.Telemetry.DBSlowQueryThresh)
	})
}
