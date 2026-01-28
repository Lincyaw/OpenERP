// Package generator provides data generation capabilities for the load generator.
// It implements various generators (faker, pattern, random) that can be used
// to dynamically generate test data based on configuration.
package generator

import (
	"errors"
	"fmt"
	"sync"
)

// ErrGeneratorNotFound is returned when trying to get a generator that doesn't exist.
var ErrGeneratorNotFound = errors.New("generator: generator not found")

// ErrInvalidConfig is returned when a generator configuration is invalid.
var ErrInvalidConfig = errors.New("generator: invalid configuration")

// Generator is the interface that all data generators must implement.
// Generators are used to produce dynamic values for request parameters,
// body fields, and other data that needs to be generated during load testing.
type Generator interface {
	// Generate produces a new value based on the generator's configuration.
	// The returned value can be a string, int, float64, or bool depending
	// on the generator type and configuration.
	Generate() (any, error)

	// Type returns the type identifier of this generator.
	Type() GeneratorType
}

// GeneratorType identifies the type of generator.
type GeneratorType string

const (
	// TypeFaker identifies a faker-based generator that uses gofakeit.
	TypeFaker GeneratorType = "faker"

	// TypePattern identifies a pattern-based generator that uses placeholders.
	TypePattern GeneratorType = "pattern"

	// TypeRandom identifies a random value generator for primitives.
	TypeRandom GeneratorType = "random"

	// TypeSequence identifies a sequential value generator.
	TypeSequence GeneratorType = "sequence"
)

// Config holds configuration for creating a generator.
// Only one of the specific config fields should be populated based on Type.
type Config struct {
	// Type is the generator type: "faker", "random", "sequence", "pattern".
	Type GeneratorType `yaml:"type" json:"type"`

	// Faker is faker-specific configuration.
	Faker *FakerConfig `yaml:"faker,omitempty" json:"faker,omitempty"`

	// Random is random generator configuration.
	Random *RandomConfig `yaml:"random,omitempty" json:"random,omitempty"`

	// Sequence is sequence generator configuration.
	Sequence *SequenceConfig `yaml:"sequence,omitempty" json:"sequence,omitempty"`

	// Pattern is pattern-based generator configuration.
	Pattern *PatternConfig `yaml:"pattern,omitempty" json:"pattern,omitempty"`
}

// FakerConfig configures the faker data generator.
type FakerConfig struct {
	// Type is the faker type: "name", "email", "phone", "address", "company",
	// "firstName", "lastName", "city", "country", "zipCode", "url", "uuid",
	// "sentence", "paragraph", "word", "creditCard", "ipv4", "ipv6", "mac".
	Type string `yaml:"type" json:"type"`

	// Locale is the locale for generated data (reserved for future use).
	// Default: "en"
	Locale string `yaml:"locale,omitempty" json:"locale,omitempty"`
}

// RandomConfig configures random value generation.
type RandomConfig struct {
	// Type is the value type: "int", "float", "string", "uuid", "bool".
	Type string `yaml:"type" json:"type"`

	// Min is the minimum value (for int/float).
	Min float64 `yaml:"min,omitempty" json:"min,omitempty"`

	// Max is the maximum value (for int/float).
	Max float64 `yaml:"max,omitempty" json:"max,omitempty"`

	// Length is the string length.
	// Default: 8
	Length int `yaml:"length,omitempty" json:"length,omitempty"`

	// Charset is the character set for strings.
	// Values: "alphanumeric", "alpha", "numeric", "hex", "alphanum_lower", "alphanum_upper"
	// Default: "alphanumeric"
	Charset string `yaml:"charset,omitempty" json:"charset,omitempty"`
}

// SequenceConfig configures sequential value generation.
type SequenceConfig struct {
	// Prefix is added before the sequence number.
	Prefix string `yaml:"prefix,omitempty" json:"prefix,omitempty"`

	// Suffix is added after the sequence number.
	Suffix string `yaml:"suffix,omitempty" json:"suffix,omitempty"`

	// Start is the starting sequence number.
	// Default: 1
	Start int64 `yaml:"start,omitempty" json:"start,omitempty"`

	// Step is the increment between values.
	// Default: 1
	Step int64 `yaml:"step,omitempty" json:"step,omitempty"`

	// Padding is the minimum width with zero-padding.
	// Default: 0 (no padding)
	Padding int `yaml:"padding,omitempty" json:"padding,omitempty"`
}

// PatternConfig configures pattern-based value generation.
type PatternConfig struct {
	// Pattern is the template pattern with placeholders.
	// Supported placeholders:
	//   {PREFIX}         - A configurable prefix string
	//   {TIMESTAMP}      - Current Unix timestamp in milliseconds
	//   {RANDOM:N}       - Random alphanumeric string of length N
	//   {UUID}           - Random UUID v4
	//   {DATE}           - Current date in YYYY-MM-DD format
	//   {TIME}           - Current time in HH:MM:SS format
	//   {DATETIME}       - Current datetime in ISO 8601 format
	//   {INT:MIN:MAX}    - Random integer between MIN and MAX
	//   {ALPHA:N}        - Random alphabetic string of length N
	//   {HEX:N}          - Random hexadecimal string of length N
	//   {SEQUENCE}       - Auto-incrementing sequence number
	Pattern string `yaml:"pattern" json:"pattern"`

	// Prefix is the value to use for {PREFIX} placeholder.
	Prefix string `yaml:"prefix,omitempty" json:"prefix,omitempty"`
}

// New creates a new generator based on the provided configuration.
func New(cfg Config) (Generator, error) {
	switch cfg.Type {
	case TypeFaker:
		if cfg.Faker == nil {
			return nil, fmt.Errorf("%w: faker config is required for faker type", ErrInvalidConfig)
		}
		return NewFakerGenerator(cfg.Faker)

	case TypeRandom:
		if cfg.Random == nil {
			return nil, fmt.Errorf("%w: random config is required for random type", ErrInvalidConfig)
		}
		return NewRandomGenerator(cfg.Random)

	case TypeSequence:
		if cfg.Sequence == nil {
			// Use defaults if no config provided
			cfg.Sequence = &SequenceConfig{}
		}
		return NewSequenceGenerator(cfg.Sequence)

	case TypePattern:
		if cfg.Pattern == nil {
			return nil, fmt.Errorf("%w: pattern config is required for pattern type", ErrInvalidConfig)
		}
		return NewPatternGenerator(cfg.Pattern)

	default:
		return nil, fmt.Errorf("%w: unknown generator type: %s", ErrInvalidConfig, cfg.Type)
	}
}

// Registry manages a collection of named generators.
// Registry is thread-safe and can be used concurrently.
type Registry struct {
	mu         sync.RWMutex
	generators map[string]Generator
}

// NewRegistry creates a new generator registry.
func NewRegistry() *Registry {
	return &Registry{
		generators: make(map[string]Generator),
	}
}

// Register adds a generator to the registry with the given name.
func (r *Registry) Register(name string, gen Generator) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.generators[name] = gen
}

// Get retrieves a generator by name.
func (r *Registry) Get(name string) (Generator, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	gen, ok := r.generators[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrGeneratorNotFound, name)
	}
	return gen, nil
}

// Has checks if a generator with the given name exists.
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.generators[name]
	return ok
}

// Names returns all registered generator names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.generators))
	for name := range r.generators {
		names = append(names, name)
	}
	return names
}

// Generate generates a value using the named generator.
func (r *Registry) Generate(name string) (any, error) {
	gen, err := r.Get(name)
	if err != nil {
		return nil, err
	}
	return gen.Generate()
}

// LoadFromConfig creates generators from a map of configurations.
// The map key is the semantic type (e.g., "common.code", "common.name").
func (r *Registry) LoadFromConfig(configs map[string]Config) error {
	for name, cfg := range configs {
		gen, err := New(cfg)
		if err != nil {
			return fmt.Errorf("creating generator %q: %w", name, err)
		}
		r.Register(name, gen)
	}
	return nil
}
