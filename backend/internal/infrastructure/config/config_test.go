package config

import (
	"os"
	"testing"

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
