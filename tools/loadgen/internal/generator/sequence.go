// Package generator provides data generation capabilities for the load generator.
package generator

import (
	"fmt"
	"sync/atomic"
)

// SequenceGenerator generates sequential values with optional prefix/suffix.
type SequenceGenerator struct {
	config  *SequenceConfig
	current int64 // atomic counter
}

// NewSequenceGenerator creates a new sequence generator with the given configuration.
func NewSequenceGenerator(cfg *SequenceConfig) (*SequenceGenerator, error) {
	if cfg == nil {
		cfg = &SequenceConfig{}
	}

	// Apply defaults
	if cfg.Step == 0 {
		cfg.Step = 1
	}
	if cfg.Start == 0 {
		cfg.Start = 1
	}

	return &SequenceGenerator{
		config:  cfg,
		current: cfg.Start - cfg.Step, // Start one step behind so first call gives Start
	}, nil
}

// Generate produces the next sequential value.
func (s *SequenceGenerator) Generate() (any, error) {
	next := atomic.AddInt64(&s.current, s.config.Step)

	// Format the number with optional padding
	var numStr string
	if s.config.Padding > 0 {
		format := fmt.Sprintf("%%0%dd", s.config.Padding)
		numStr = fmt.Sprintf(format, next)
	} else {
		numStr = fmt.Sprintf("%d", next)
	}

	// Apply prefix and suffix
	return s.config.Prefix + numStr + s.config.Suffix, nil
}

// Type returns the generator type.
func (s *SequenceGenerator) Type() GeneratorType {
	return TypeSequence
}

// Reset resets the sequence to its starting value.
func (s *SequenceGenerator) Reset() {
	atomic.StoreInt64(&s.current, s.config.Start-s.config.Step)
}

// Current returns the current sequence value without incrementing.
func (s *SequenceGenerator) Current() int64 {
	return atomic.LoadInt64(&s.current)
}
