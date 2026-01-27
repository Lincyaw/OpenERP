package warmup

import (
	"testing"
	"time"

	"github.com/example/erp/tools/loadgen/internal/circuit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, 10, cfg.Iterations)
	assert.Nil(t, cfg.Fill)
	assert.Equal(t, 5, cfg.MinPoolSize)
	assert.Equal(t, 5*time.Minute, cfg.Timeout)
	assert.Equal(t, 3, cfg.RetryCount)
	assert.Equal(t, time.Second, cfg.RetryDelay)
	assert.True(t, cfg.ContinueOnError)
	assert.False(t, cfg.Verbose)
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "valid empty config",
			config:  Config{},
			wantErr: false,
		},
		{
			name: "valid config with fill",
			config: Config{
				Iterations: 10,
				Fill: []circuit.SemanticType{
					"entity.customer.id",
					"entity.product.id",
				},
				MinPoolSize: 5,
				Timeout:     time.Minute,
				RetryCount:  3,
				RetryDelay:  time.Second,
			},
			wantErr: false,
		},
		{
			name: "negative iterations",
			config: Config{
				Iterations: -1,
			},
			wantErr: true,
		},
		{
			name: "negative minPoolSize",
			config: Config{
				MinPoolSize: -1,
			},
			wantErr: true,
		},
		{
			name: "negative timeout",
			config: Config{
				Timeout: -time.Second,
			},
			wantErr: true,
		},
		{
			name: "negative retryCount",
			config: Config{
				RetryCount: -1,
			},
			wantErr: true,
		},
		{
			name: "negative retryDelay",
			config: Config{
				RetryDelay: -time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidConfig)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_ApplyDefaults(t *testing.T) {
	t.Run("applies defaults to sentinel values (-1)", func(t *testing.T) {
		cfg := Config{
			Iterations:  -1,
			MinPoolSize: -1,
			Timeout:     -1,
			RetryCount:  -1,
			RetryDelay:  -1,
		}
		cfg.ApplyDefaults()

		defaults := DefaultConfig()
		assert.Equal(t, defaults.Iterations, cfg.Iterations)
		assert.Equal(t, defaults.MinPoolSize, cfg.MinPoolSize)
		assert.Equal(t, defaults.Timeout, cfg.Timeout)
		assert.Equal(t, defaults.RetryCount, cfg.RetryCount)
		assert.Equal(t, defaults.RetryDelay, cfg.RetryDelay)
	})

	t.Run("preserves zero values", func(t *testing.T) {
		cfg := Config{
			Iterations:  0,
			MinPoolSize: 0,
			Timeout:     0,
			RetryCount:  0,
			RetryDelay:  0,
		}
		cfg.ApplyDefaults()

		assert.Equal(t, 0, cfg.Iterations)
		assert.Equal(t, 0, cfg.MinPoolSize)
		assert.Equal(t, time.Duration(0), cfg.Timeout)
		assert.Equal(t, 0, cfg.RetryCount)
		assert.Equal(t, time.Duration(0), cfg.RetryDelay)
	})

	t.Run("preserves explicitly set positive values", func(t *testing.T) {
		cfg := Config{
			Iterations:  20,
			MinPoolSize: 10,
			Timeout:     10 * time.Minute,
			RetryCount:  5,
			RetryDelay:  2 * time.Second,
		}
		cfg.ApplyDefaults()

		assert.Equal(t, 20, cfg.Iterations)
		assert.Equal(t, 10, cfg.MinPoolSize)
		assert.Equal(t, 10*time.Minute, cfg.Timeout)
		assert.Equal(t, 5, cfg.RetryCount)
		assert.Equal(t, 2*time.Second, cfg.RetryDelay)
	})
}

func TestConfig_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected bool
	}{
		{
			name:     "empty config",
			config:   Config{},
			expected: true,
		},
		{
			name: "config with iterations only",
			config: Config{
				Iterations: 10,
			},
			expected: false, // Has iterations, not empty
		},
		{
			name: "config with fill only",
			config: Config{
				Fill: []circuit.SemanticType{"entity.customer.id"},
			},
			expected: false, // Has fill, not empty
		},
		{
			name: "config with both iterations and fill",
			config: Config{
				Iterations: 10,
				Fill:       []circuit.SemanticType{"entity.customer.id"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsEmpty()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfig_Clone(t *testing.T) {
	original := Config{
		Iterations:      20,
		Fill:            []circuit.SemanticType{"entity.customer.id", "entity.product.id"},
		MinPoolSize:     10,
		Timeout:         10 * time.Minute,
		RetryCount:      5,
		RetryDelay:      2 * time.Second,
		ContinueOnError: true,
		Verbose:         true,
	}

	clone := original.Clone()

	// Verify values are equal
	assert.Equal(t, original.Iterations, clone.Iterations)
	assert.Equal(t, original.Fill, clone.Fill)
	assert.Equal(t, original.MinPoolSize, clone.MinPoolSize)
	assert.Equal(t, original.Timeout, clone.Timeout)
	assert.Equal(t, original.RetryCount, clone.RetryCount)
	assert.Equal(t, original.RetryDelay, clone.RetryDelay)
	assert.Equal(t, original.ContinueOnError, clone.ContinueOnError)
	assert.Equal(t, original.Verbose, clone.Verbose)

	// Verify Fill slice is a copy (not same underlying array)
	require.NotNil(t, clone.Fill)
	clone.Fill[0] = "modified"
	assert.NotEqual(t, original.Fill[0], clone.Fill[0])
}

func TestConfig_Clone_NilFill(t *testing.T) {
	original := Config{
		Iterations: 10,
		Fill:       nil,
	}

	clone := original.Clone()

	assert.Nil(t, clone.Fill)
}
