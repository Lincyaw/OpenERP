// Package generator provides data generation capabilities for the load generator.
package generator

import (
	"crypto/rand"
	"fmt"

	"github.com/google/uuid"
)

// RandomGenerator generates random values of various types.
type RandomGenerator struct {
	config *RandomConfig
}

// NewRandomGenerator creates a new random generator with the given configuration.
func NewRandomGenerator(cfg *RandomConfig) (*RandomGenerator, error) {
	if cfg == nil {
		return nil, fmt.Errorf("%w: random config is nil", ErrInvalidConfig)
	}
	if cfg.Type == "" {
		return nil, fmt.Errorf("%w: random type is required", ErrInvalidConfig)
	}

	// Validate type
	validTypes := map[string]bool{
		"int":    true,
		"float":  true,
		"string": true,
		"uuid":   true,
		"bool":   true,
	}
	if !validTypes[cfg.Type] {
		return nil, fmt.Errorf("%w: unknown random type: %s", ErrInvalidConfig, cfg.Type)
	}

	// Apply defaults
	if cfg.Length == 0 {
		cfg.Length = 8
	}
	if cfg.Charset == "" {
		cfg.Charset = "alphanumeric"
	}

	return &RandomGenerator{
		config: cfg,
	}, nil
}

// Generate produces a new random value.
func (r *RandomGenerator) Generate() (any, error) {
	switch r.config.Type {
	case "int":
		return r.generateInt(), nil

	case "float":
		return r.generateFloat(), nil

	case "string":
		return r.generateString(), nil

	case "uuid":
		return uuid.New().String(), nil

	case "bool":
		return r.generateBool(), nil

	default:
		return nil, fmt.Errorf("%w: unknown random type: %s", ErrInvalidConfig, r.config.Type)
	}
}

// Type returns the generator type.
func (r *RandomGenerator) Type() GeneratorType {
	return TypeRandom
}

// generateInt generates a random integer between Min and Max.
func (r *RandomGenerator) generateInt() int {
	min := int(r.config.Min)
	max := int(r.config.Max)

	if max == 0 && min == 0 {
		// Default range if not specified
		max = 100
	}

	if min >= max {
		return min
	}

	return randomInt(min, max)
}

// generateFloat generates a random float between Min and Max.
func (r *RandomGenerator) generateFloat() float64 {
	min := r.config.Min
	max := r.config.Max

	if max == 0 && min == 0 {
		// Default range if not specified
		max = 100.0
	}

	if min >= max {
		return min
	}

	return randomFloat(min, max)
}

// generateString generates a random string of the configured length.
func (r *RandomGenerator) generateString() string {
	charset := r.getCharset()
	return randomString(r.config.Length, charset)
}

// generateBool generates a random boolean.
func (r *RandomGenerator) generateBool() bool {
	bytes := make([]byte, 1)
	if _, err := rand.Read(bytes); err != nil {
		return false
	}
	return bytes[0]%2 == 0
}

// getCharset returns the character set based on configuration.
func (r *RandomGenerator) getCharset() string {
	switch r.config.Charset {
	case "alpha":
		return alphaChars
	case "numeric":
		return numericChars
	case "hex":
		return "0123456789abcdef"
	case "alphanum_lower":
		return "abcdefghijklmnopqrstuvwxyz0123456789"
	case "alphanum_upper":
		return "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	case "alphanumeric":
		fallthrough
	default:
		return alphanumericChars
	}
}

// randomFloat generates a random float between min and max.
func randomFloat(min, max float64) float64 {
	if min >= max {
		return min
	}

	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return min
	}

	// Convert bytes to a float in [0, 1)
	var n uint64
	for i := 0; i < 8; i++ {
		n = (n << 8) | uint64(bytes[i])
	}

	// Map to range [min, max)
	fraction := float64(n) / float64(^uint64(0))
	return min + fraction*(max-min)
}
