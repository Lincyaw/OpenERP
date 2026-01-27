// Package loadctrl provides load control components including traffic shaping.
package loadctrl

import (
	"fmt"
	"math"
	"time"
)

// SineWaveShaper implements a sinusoidal traffic pattern.
// QPS varies smoothly following a sine wave around the base QPS.
//
// The formula is: QPS = BaseQPS + Amplitude * sin(2π * elapsed / Period)
//
// Thread Safety: Safe for concurrent use by multiple goroutines (read-only after creation).
type SineWaveShaper struct {
	config ShaperConfig
}

// NewSineWaveShaper creates a new sine wave traffic shaper.
// The amplitude can be specified as:
//   - Absolute value (e.g., 50 means ±50 QPS from base)
//   - If amplitude > 1, it's treated as absolute
//   - If amplitude <= 1, it's treated as relative (percentage of baseQPS)
func NewSineWaveShaper(config ShaperConfig) (*SineWaveShaper, error) {
	if config.Type != "sine" {
		return nil, fmt.Errorf("expected type 'sine', got '%s'", config.Type)
	}

	if config.Period <= 0 {
		return nil, fmt.Errorf("period must be positive, got: %v", config.Period)
	}

	if config.BaseQPS < 0 {
		return nil, fmt.Errorf("baseQPS cannot be negative: %f", config.BaseQPS)
	}

	return &SineWaveShaper{
		config: config,
	}, nil
}

// GetTargetQPS returns the target QPS for the given elapsed time.
// The QPS follows a sine wave pattern: BaseQPS + Amplitude * sin(2π * t / Period)
func (s *SineWaveShaper) GetTargetQPS(elapsed time.Duration) float64 {
	// Calculate the amplitude to use
	amplitude := s.getEffectiveAmplitude()

	// Calculate the angular position in the sine wave
	// phase = 2π * elapsed / period
	phase := 2 * math.Pi * float64(elapsed) / float64(s.config.Period)

	// Calculate QPS: base + amplitude * sin(phase)
	qps := s.config.BaseQPS + amplitude*math.Sin(phase)

	// Ensure QPS is never negative
	if qps < 0 {
		qps = 0
	}

	// Apply min/max clamping if configured
	return clampQPS(qps, s.config.MinQPS, s.config.MaxQPS)
}

// GetPhase returns a human-readable description of the current phase.
func (s *SineWaveShaper) GetPhase(elapsed time.Duration) string {
	// Calculate which period we're in
	periodNum := int(elapsed / s.config.Period)
	positionInPeriod := float64(elapsed%s.config.Period) / float64(s.config.Period)

	// Determine phase name based on position in the sine wave
	// 0-0.25: Rising (0 to peak)
	// 0.25-0.5: Falling (peak to 0)
	// 0.5-0.75: Falling (0 to trough)
	// 0.75-1.0: Rising (trough to 0)
	var phaseName string
	switch {
	case positionInPeriod < 0.25:
		phaseName = "rising to peak"
	case positionInPeriod < 0.5:
		phaseName = "falling from peak"
	case positionInPeriod < 0.75:
		phaseName = "falling to trough"
	default:
		phaseName = "rising from trough"
	}

	return fmt.Sprintf("period %d: %s (%.1f%% through period)",
		periodNum+1, phaseName, positionInPeriod*100)
}

// Name returns the name of this shaper type.
func (s *SineWaveShaper) Name() string {
	return "sine"
}

// Config returns a copy of the shaper's configuration.
func (s *SineWaveShaper) Config() ShaperConfig {
	return s.config
}

// getEffectiveAmplitude calculates the actual amplitude to use.
// If amplitude <= 1, it's treated as a percentage of baseQPS.
// If amplitude > 1, it's treated as an absolute value.
func (s *SineWaveShaper) getEffectiveAmplitude() float64 {
	amplitude := s.config.Amplitude

	// Treat values <= 1 as percentages (e.g., 0.5 = 50%)
	if amplitude <= 1 && amplitude > 0 {
		amplitude = s.config.BaseQPS * amplitude
	}

	return amplitude
}

// GetMinMaxQPS returns the theoretical minimum and maximum QPS values
// based on the configuration (before clamping).
func (s *SineWaveShaper) GetMinMaxQPS() (min, max float64) {
	amplitude := s.getEffectiveAmplitude()
	min = s.config.BaseQPS - amplitude
	max = s.config.BaseQPS + amplitude

	if min < 0 {
		min = 0
	}

	return min, max
}

// GetPeriod returns the period of the sine wave.
func (s *SineWaveShaper) GetPeriod() time.Duration {
	return s.config.Period
}

// GetAmplitude returns the effective amplitude (absolute value).
func (s *SineWaveShaper) GetAmplitude() float64 {
	return s.getEffectiveAmplitude()
}
