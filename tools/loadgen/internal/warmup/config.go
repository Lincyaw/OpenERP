// Package warmup implements the warmup phase for the load generator.
// The warmup phase executes producer endpoints to fill the parameter pool
// before the main load test begins.
package warmup

import (
	"errors"
	"fmt"
	"time"

	"github.com/example/erp/tools/loadgen/internal/circuit"
)

// Errors returned by the warmup package.
var (
	// ErrInvalidConfig is returned when the warmup configuration is invalid.
	ErrInvalidConfig = errors.New("warmup: invalid configuration")
	// ErrPoolNotReady is returned when parameter pool is not sufficiently filled.
	ErrPoolNotReady = errors.New("warmup: parameter pool not ready")
	// ErrWarmupFailed is returned when warmup phase fails.
	ErrWarmupFailed = errors.New("warmup: warmup phase failed")
	// ErrLoginFailed is returned when authentication fails during warmup.
	ErrLoginFailed = errors.New("warmup: login failed")
)

// Config holds the warmup phase configuration.
type Config struct {
	// Iterations is the number of times to execute each producer endpoint.
	// Default: 10
	Iterations int `yaml:"iterations" json:"iterations"`

	// Fill specifies the semantic types to fill in order.
	// The warmup phase executes producer endpoints for each type.
	// Example: ["entity.customer.id", "entity.product.id"]
	Fill []circuit.SemanticType `yaml:"fill" json:"fill"`

	// MinPoolSize is the minimum number of values required for each semantic type
	// before warmup is considered complete.
	// Default: 5
	MinPoolSize int `yaml:"minPoolSize" json:"minPoolSize"`

	// Timeout is the maximum duration for the entire warmup phase.
	// Default: 5m
	Timeout time.Duration `yaml:"timeout" json:"timeout"`

	// RetryCount is the number of retries for failed producer calls.
	// Default: 3
	RetryCount int `yaml:"retryCount" json:"retryCount"`

	// RetryDelay is the delay between retry attempts.
	// Default: 1s
	RetryDelay time.Duration `yaml:"retryDelay" json:"retryDelay"`

	// ContinueOnError determines whether to continue if a producer fails.
	// If false, warmup fails immediately on first error.
	// Default: true
	ContinueOnError bool `yaml:"continueOnError" json:"continueOnError"`

	// Verbose enables detailed logging during warmup.
	// Default: false
	Verbose bool `yaml:"verbose" json:"verbose"`
}

// DefaultConfig returns the default warmup configuration.
func DefaultConfig() Config {
	return Config{
		Iterations:      10,
		Fill:            nil, // Must be explicitly configured
		MinPoolSize:     5,
		Timeout:         5 * time.Minute,
		RetryCount:      3,
		RetryDelay:      time.Second,
		ContinueOnError: true,
		Verbose:         false,
	}
}

// Validate validates the warmup configuration.
func (c *Config) Validate() error {
	if c.Iterations < 0 {
		return fmt.Errorf("%w: iterations must be non-negative", ErrInvalidConfig)
	}

	if c.MinPoolSize < 0 {
		return fmt.Errorf("%w: minPoolSize must be non-negative", ErrInvalidConfig)
	}

	if c.Timeout < 0 {
		return fmt.Errorf("%w: timeout must be non-negative", ErrInvalidConfig)
	}

	if c.RetryCount < 0 {
		return fmt.Errorf("%w: retryCount must be non-negative", ErrInvalidConfig)
	}

	if c.RetryDelay < 0 {
		return fmt.Errorf("%w: retryDelay must be non-negative", ErrInvalidConfig)
	}

	return nil
}

// ApplyDefaults applies default values to sentinel-valued or unset fields.
// Negative values (-1) indicate "use default". Zero values for numeric fields
// are preserved (e.g., RetryCount=0 means no retries).
func (c *Config) ApplyDefaults() {
	defaults := DefaultConfig()

	if c.Iterations < 0 {
		c.Iterations = defaults.Iterations
	}
	if c.MinPoolSize < 0 {
		c.MinPoolSize = defaults.MinPoolSize
	}
	if c.Timeout < 0 {
		c.Timeout = defaults.Timeout
	}
	if c.RetryCount < 0 {
		c.RetryCount = defaults.RetryCount
	}
	if c.RetryDelay < 0 {
		c.RetryDelay = defaults.RetryDelay
	}
}

// IsEmpty returns true if the warmup configuration has no fill requirements.
func (c *Config) IsEmpty() bool {
	return len(c.Fill) == 0 && c.Iterations == 0
}

// Clone returns a deep copy of the configuration.
func (c *Config) Clone() Config {
	clone := *c
	if c.Fill != nil {
		clone.Fill = make([]circuit.SemanticType, len(c.Fill))
		copy(clone.Fill, c.Fill)
	}
	return clone
}
