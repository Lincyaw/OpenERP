// Package loadctrl provides load control components including traffic shaping
// and rate limiting for the load generator.
package loadctrl

import (
	"fmt"
	"time"
)

// TrafficShaper defines the interface for traffic shaping strategies.
// Implementations control how QPS varies over time during a load test.
//
// Thread Safety: Implementations should be safe for concurrent use by multiple goroutines.
type TrafficShaper interface {
	// GetTargetQPS returns the target QPS for the given elapsed time.
	// The elapsed duration is measured from the start of the load test.
	GetTargetQPS(elapsed time.Duration) float64

	// GetPhase returns a human-readable description of the current phase
	// for the given elapsed time. This is useful for monitoring and logging.
	GetPhase(elapsed time.Duration) string

	// Name returns the name of the shaper type (e.g., "sine", "spike", "step", "custom").
	Name() string

	// Config returns a copy of the shaper's configuration for inspection.
	Config() ShaperConfig
}

// ShaperConfig is a generic configuration container for traffic shapers.
// Each shaper type interprets these fields according to its pattern.
type ShaperConfig struct {
	// Type identifies the shaper type: "sine", "spike", "step", "custom"
	Type string `yaml:"type" json:"type"`

	// BaseQPS is the baseline QPS around which the pattern oscillates.
	// For spike shaper, this is the normal QPS.
	// For step shaper, this is the starting QPS.
	BaseQPS float64 `yaml:"baseQPS" json:"baseQPS"`

	// MinQPS is the minimum QPS value (optional floor for patterns).
	MinQPS float64 `yaml:"minQPS,omitempty" json:"minQPS,omitempty"`

	// MaxQPS is the maximum QPS value (optional ceiling for patterns).
	MaxQPS float64 `yaml:"maxQPS,omitempty" json:"maxQPS,omitempty"`

	// Amplitude is the amplitude of oscillation (for sine wave).
	// Can be absolute (e.g., 50) or relative percentage (e.g., 0.5 for ±50%).
	Amplitude float64 `yaml:"amplitude,omitempty" json:"amplitude,omitempty"`

	// Period is the duration of one complete cycle (for sine wave).
	Period time.Duration `yaml:"period,omitempty" json:"period,omitempty"`

	// SpikeConfig contains spike-specific configuration.
	Spike *SpikeConfig `yaml:"spike,omitempty" json:"spike,omitempty"`

	// StepConfig contains step-specific configuration.
	Step *StepConfig `yaml:"step,omitempty" json:"step,omitempty"`

	// CustomPoints defines custom QPS points for interpolation.
	CustomPoints []CustomPoint `yaml:"customPoints,omitempty" json:"customPoints,omitempty"`
}

// SpikeConfig holds configuration for spike traffic patterns.
type SpikeConfig struct {
	// SpikeQPS is the QPS during spike periods.
	SpikeQPS float64 `yaml:"spikeQPS" json:"spikeQPS"`

	// SpikeDuration is how long each spike lasts.
	SpikeDuration time.Duration `yaml:"spikeDuration" json:"spikeDuration"`

	// SpikeInterval is the time between spike starts.
	SpikeInterval time.Duration `yaml:"spikeInterval" json:"spikeInterval"`
}

// StepConfig holds configuration for step/staircase traffic patterns.
type StepConfig struct {
	// Steps defines the QPS levels and their durations.
	Steps []StepLevel `yaml:"steps" json:"steps"`

	// Loop indicates whether to repeat the steps after completing all.
	Loop bool `yaml:"loop,omitempty" json:"loop,omitempty"`
}

// StepLevel defines a single step in a staircase pattern.
type StepLevel struct {
	// QPS is the target QPS for this step.
	QPS float64 `yaml:"qps" json:"qps"`

	// Duration is how long this step lasts.
	Duration time.Duration `yaml:"duration" json:"duration"`

	// RampDuration is optional time to ramp from previous QPS to this QPS.
	// If zero, the transition is immediate.
	RampDuration time.Duration `yaml:"rampDuration,omitempty" json:"rampDuration,omitempty"`
}

// CustomPoint defines a point in a custom QPS curve.
type CustomPoint struct {
	// Time is the elapsed time for this point.
	Time time.Duration `yaml:"time" json:"time"`

	// QPS is the target QPS at this point.
	QPS float64 `yaml:"qps" json:"qps"`
}

// Validate validates the shaper configuration.
func (c *ShaperConfig) Validate() error {
	if c.Type == "" {
		return fmt.Errorf("shaper type is required")
	}

	if c.BaseQPS < 0 {
		return fmt.Errorf("baseQPS cannot be negative: %f", c.BaseQPS)
	}

	if c.MinQPS < 0 {
		return fmt.Errorf("minQPS cannot be negative: %f", c.MinQPS)
	}

	if c.MaxQPS > 0 && c.MinQPS > c.MaxQPS {
		return fmt.Errorf("minQPS (%f) cannot exceed maxQPS (%f)", c.MinQPS, c.MaxQPS)
	}

	switch c.Type {
	case "sine":
		if c.Period <= 0 {
			return fmt.Errorf("sine shaper requires positive period, got: %v", c.Period)
		}
		if c.Amplitude < 0 {
			return fmt.Errorf("sine amplitude cannot be negative: %f", c.Amplitude)
		}

	case "spike":
		if c.Spike == nil {
			return fmt.Errorf("spike shaper requires spike configuration")
		}
		if c.Spike.SpikeQPS < 0 {
			return fmt.Errorf("spike QPS cannot be negative: %f", c.Spike.SpikeQPS)
		}
		if c.Spike.SpikeDuration <= 0 {
			return fmt.Errorf("spike duration must be positive: %v", c.Spike.SpikeDuration)
		}
		if c.Spike.SpikeInterval <= 0 {
			return fmt.Errorf("spike interval must be positive: %v", c.Spike.SpikeInterval)
		}
		if c.Spike.SpikeDuration >= c.Spike.SpikeInterval {
			return fmt.Errorf("spike duration (%v) must be less than spike interval (%v)",
				c.Spike.SpikeDuration, c.Spike.SpikeInterval)
		}

	case "step":
		if c.Step == nil {
			return fmt.Errorf("step shaper requires step configuration")
		}
		if len(c.Step.Steps) == 0 {
			return fmt.Errorf("step shaper requires at least one step")
		}
		for i, step := range c.Step.Steps {
			if step.QPS < 0 {
				return fmt.Errorf("step %d: QPS cannot be negative: %f", i, step.QPS)
			}
			if step.Duration <= 0 {
				return fmt.Errorf("step %d: duration must be positive: %v", i, step.Duration)
			}
			if step.RampDuration < 0 {
				return fmt.Errorf("step %d: ramp duration cannot be negative: %v", i, step.RampDuration)
			}
			if step.RampDuration > step.Duration {
				return fmt.Errorf("step %d: ramp duration (%v) cannot exceed step duration (%v)",
					i, step.RampDuration, step.Duration)
			}
		}

	case "custom":
		if len(c.CustomPoints) < 2 {
			return fmt.Errorf("custom shaper requires at least 2 points, got: %d", len(c.CustomPoints))
		}
		// Verify points are in chronological order
		for i := 1; i < len(c.CustomPoints); i++ {
			if c.CustomPoints[i].Time <= c.CustomPoints[i-1].Time {
				return fmt.Errorf("custom points must be in chronological order: point %d (%v) <= point %d (%v)",
					i, c.CustomPoints[i].Time, i-1, c.CustomPoints[i-1].Time)
			}
		}
		for i, pt := range c.CustomPoints {
			if pt.QPS < 0 {
				return fmt.Errorf("custom point %d: QPS cannot be negative: %f", i, pt.QPS)
			}
		}

	default:
		return fmt.Errorf("unknown shaper type: %s", c.Type)
	}

	return nil
}

// NewTrafficShaper creates a new traffic shaper based on the configuration.
// Returns an error if the configuration is invalid.
func NewTrafficShaper(config ShaperConfig) (TrafficShaper, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid shaper config: %w", err)
	}

	switch config.Type {
	case "sine":
		return NewSineWaveShaper(config)
	case "spike":
		return NewSpikeShaper(config)
	case "step":
		return NewStepShaper(config)
	case "custom":
		return NewCustomShaper(config)
	default:
		return nil, fmt.Errorf("unknown shaper type: %s", config.Type)
	}
}

// clampQPS clamps the given QPS to the min/max bounds if they are set.
func clampQPS(qps, minQPS, maxQPS float64) float64 {
	if minQPS > 0 && qps < minQPS {
		return minQPS
	}
	if maxQPS > 0 && qps > maxQPS {
		return maxQPS
	}
	return qps
}
