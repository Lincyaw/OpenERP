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
		"APP_NAME":            os.Getenv("APP_NAME"),
		"APP_ENV":             os.Getenv("APP_ENV"),
		"APP_PORT":            os.Getenv("APP_PORT"),
		"DB_HOST":             os.Getenv("DB_HOST"),
		"DB_PORT":             os.Getenv("DB_PORT"),
		"DB_USER":             os.Getenv("DB_USER"),
		"DB_PASSWORD":         os.Getenv("DB_PASSWORD"),
		"DB_NAME":             os.Getenv("DB_NAME"),
		"DB_SSL_MODE":         os.Getenv("DB_SSL_MODE"),
		"DB_MAX_OPEN_CONNS":   os.Getenv("DB_MAX_OPEN_CONNS"),
		"DB_MAX_IDLE_CONNS":   os.Getenv("DB_MAX_IDLE_CONNS"),
		"JWT_SECRET":          os.Getenv("JWT_SECRET"),
		"JWT_EXPIRATION_HOURS": os.Getenv("JWT_EXPIRATION_HOURS"),
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

	t.Run("loads values from environment variables", func(t *testing.T) {
		clearEnv()
		os.Setenv("APP_NAME", "test-app")
		os.Setenv("APP_ENV", "testing")
		os.Setenv("APP_PORT", "9000")
		os.Setenv("DB_HOST", "testdb.local")
		os.Setenv("DB_PORT", "5433")
		os.Setenv("DB_USER", "testuser")
		os.Setenv("DB_PASSWORD", "testpass")
		os.Setenv("DB_NAME", "testdb")
		os.Setenv("DB_SSL_MODE", "require")
		os.Setenv("DB_MAX_OPEN_CONNS", "50")
		os.Setenv("DB_MAX_IDLE_CONNS", "10")

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
		os.Setenv("DB_MAX_OPEN_CONNS", "10")
		os.Setenv("DB_MAX_IDLE_CONNS", "20")

		_, err := Load()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "DB_MAX_IDLE_CONNS")
		assert.Contains(t, err.Error(), "cannot exceed DB_MAX_OPEN_CONNS")
	})

	t.Run("validates MaxOpenConns must be positive", func(t *testing.T) {
		clearEnv()
		os.Setenv("DB_MAX_OPEN_CONNS", "0")

		_, err := Load()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "DB_MAX_OPEN_CONNS must be positive")
	})

	t.Run("validates MaxIdleConns cannot be negative", func(t *testing.T) {
		clearEnv()
		os.Setenv("DB_MAX_IDLE_CONNS", "-1")

		_, err := Load()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "DB_MAX_IDLE_CONNS cannot be negative")
	})
}

func TestLoad_ProductionValidation(t *testing.T) {
	originalEnv := map[string]string{
		"APP_ENV":     os.Getenv("APP_ENV"),
		"JWT_SECRET":  os.Getenv("JWT_SECRET"),
		"DB_PASSWORD": os.Getenv("DB_PASSWORD"),
		"DB_SSL_MODE": os.Getenv("DB_SSL_MODE"),
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
		os.Unsetenv("APP_ENV")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("DB_SSL_MODE")
	}

	t.Run("requires JWT_SECRET in production", func(t *testing.T) {
		clearEnv()
		os.Setenv("APP_ENV", "production")
		os.Setenv("DB_PASSWORD", "secure-password")
		os.Setenv("DB_SSL_MODE", "require")

		_, err := Load()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "JWT_SECRET is required in production")
	})

	t.Run("requires JWT_SECRET at least 32 characters in production", func(t *testing.T) {
		clearEnv()
		os.Setenv("APP_ENV", "production")
		os.Setenv("JWT_SECRET", "short-secret")
		os.Setenv("DB_PASSWORD", "secure-password")
		os.Setenv("DB_SSL_MODE", "require")

		_, err := Load()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "JWT_SECRET must be at least 32 characters")
	})

	t.Run("requires DB_PASSWORD in production", func(t *testing.T) {
		clearEnv()
		os.Setenv("APP_ENV", "production")
		os.Setenv("JWT_SECRET", "this-is-a-very-secure-jwt-secret-key-32chars")
		os.Setenv("DB_SSL_MODE", "require")

		_, err := Load()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "DB_PASSWORD is required in production")
	})

	t.Run("requires SSL enabled in production", func(t *testing.T) {
		clearEnv()
		os.Setenv("APP_ENV", "production")
		os.Setenv("JWT_SECRET", "this-is-a-very-secure-jwt-secret-key-32chars")
		os.Setenv("DB_PASSWORD", "secure-password")
		os.Setenv("DB_SSL_MODE", "disable")

		_, err := Load()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "DB_SSL_MODE cannot be 'disable' in production")
	})

	t.Run("passes validation with valid production config", func(t *testing.T) {
		clearEnv()
		os.Setenv("APP_ENV", "production")
		os.Setenv("JWT_SECRET", "this-is-a-very-secure-jwt-secret-key-32chars")
		os.Setenv("DB_PASSWORD", "secure-password")
		os.Setenv("DB_SSL_MODE", "require")

		cfg, err := Load()
		require.NoError(t, err)
		assert.Equal(t, "production", cfg.App.Env)
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

func TestGetEnv(t *testing.T) {
	originalValue := os.Getenv("TEST_ENV_VAR")
	defer func() {
		if originalValue == "" {
			os.Unsetenv("TEST_ENV_VAR")
		} else {
			os.Setenv("TEST_ENV_VAR", originalValue)
		}
	}()

	t.Run("returns env value when set", func(t *testing.T) {
		os.Setenv("TEST_ENV_VAR", "test-value")
		value := getEnv("TEST_ENV_VAR", "default")
		assert.Equal(t, "test-value", value)
	})

	t.Run("returns default when env not set", func(t *testing.T) {
		os.Unsetenv("TEST_ENV_VAR")
		value := getEnv("TEST_ENV_VAR", "default")
		assert.Equal(t, "default", value)
	})
}

func TestGetEnvAsInt(t *testing.T) {
	originalValue := os.Getenv("TEST_INT_VAR")
	defer func() {
		if originalValue == "" {
			os.Unsetenv("TEST_INT_VAR")
		} else {
			os.Setenv("TEST_INT_VAR", originalValue)
		}
	}()

	t.Run("returns int value when valid", func(t *testing.T) {
		os.Setenv("TEST_INT_VAR", "42")
		value := getEnvAsInt("TEST_INT_VAR", 0)
		assert.Equal(t, 42, value)
	})

	t.Run("returns default when env not set", func(t *testing.T) {
		os.Unsetenv("TEST_INT_VAR")
		value := getEnvAsInt("TEST_INT_VAR", 10)
		assert.Equal(t, 10, value)
	})

	t.Run("returns default when value is not a valid integer", func(t *testing.T) {
		os.Setenv("TEST_INT_VAR", "not-an-int")
		value := getEnvAsInt("TEST_INT_VAR", 10)
		assert.Equal(t, 10, value)
	})
}
