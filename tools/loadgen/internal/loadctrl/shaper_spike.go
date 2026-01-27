// Package loadctrl provides load control components including traffic shaping.
package loadctrl

import (
	"fmt"
	"time"
)

// SpikeShaper implements a burst spike traffic pattern.
// QPS alternates between a base level and spike level at regular intervals.
//
// Pattern:
//
//	    SpikeQPS  ____
//	             |    |
//	             |    |
//	BaseQPS ____|    |____
//	        ↑        ↑
//	        |        SpikeDuration
//	        SpikeInterval
//
// Thread Safety: Safe for concurrent use by multiple goroutines (read-only after creation).
type SpikeShaper struct {
	config ShaperConfig
}

// NewSpikeShaper creates a new spike traffic shaper.
// The shaper alternates between baseQPS and spikeQPS based on the interval and duration.
func NewSpikeShaper(config ShaperConfig) (*SpikeShaper, error) {
	if config.Type != "spike" {
		return nil, fmt.Errorf("expected type 'spike', got '%s'", config.Type)
	}

	if config.Spike == nil {
		return nil, fmt.Errorf("spike configuration is required")
	}

	if config.Spike.SpikeInterval <= 0 {
		return nil, fmt.Errorf("spike interval must be positive: %v", config.Spike.SpikeInterval)
	}

	if config.Spike.SpikeDuration <= 0 {
		return nil, fmt.Errorf("spike duration must be positive: %v", config.Spike.SpikeDuration)
	}

	if config.Spike.SpikeDuration >= config.Spike.SpikeInterval {
		return nil, fmt.Errorf("spike duration (%v) must be less than spike interval (%v)",
			config.Spike.SpikeDuration, config.Spike.SpikeInterval)
	}

	if config.BaseQPS < 0 {
		return nil, fmt.Errorf("baseQPS cannot be negative: %f", config.BaseQPS)
	}

	if config.Spike.SpikeQPS < 0 {
		return nil, fmt.Errorf("spikeQPS cannot be negative: %f", config.Spike.SpikeQPS)
	}

	return &SpikeShaper{
		config: config,
	}, nil
}

// GetTargetQPS returns the target QPS for the given elapsed time.
// Returns spikeQPS during spike periods and baseQPS otherwise.
func (s *SpikeShaper) GetTargetQPS(elapsed time.Duration) float64 {
	if s.isInSpike(elapsed) {
		return clampQPS(s.config.Spike.SpikeQPS, s.config.MinQPS, s.config.MaxQPS)
	}
	return clampQPS(s.config.BaseQPS, s.config.MinQPS, s.config.MaxQPS)
}

// GetPhase returns a human-readable description of the current phase.
func (s *SpikeShaper) GetPhase(elapsed time.Duration) string {
	spikeNum := int(elapsed/s.config.Spike.SpikeInterval) + 1
	posInInterval := elapsed % s.config.Spike.SpikeInterval

	if posInInterval < s.config.Spike.SpikeDuration {
		progress := float64(posInInterval) / float64(s.config.Spike.SpikeDuration) * 100
		return fmt.Sprintf("spike %d: active (%.1f%% through spike)", spikeNum, progress)
	}

	// Time until next spike
	timeUntilNextSpike := s.config.Spike.SpikeInterval - posInInterval
	return fmt.Sprintf("spike %d: normal (%.1fs until next spike)", spikeNum, timeUntilNextSpike.Seconds())
}

// Name returns the name of this shaper type.
func (s *SpikeShaper) Name() string {
	return "spike"
}

// Config returns a copy of the shaper's configuration.
func (s *SpikeShaper) Config() ShaperConfig {
	return s.config
}

// isInSpike returns true if the elapsed time is within a spike period.
func (s *SpikeShaper) isInSpike(elapsed time.Duration) bool {
	// Calculate position within the current interval
	posInInterval := elapsed % s.config.Spike.SpikeInterval

	// We're in a spike if we're within the spike duration
	return posInInterval < s.config.Spike.SpikeDuration
}

// GetSpikeInterval returns the interval between spike starts.
func (s *SpikeShaper) GetSpikeInterval() time.Duration {
	return s.config.Spike.SpikeInterval
}

// GetSpikeDuration returns how long each spike lasts.
func (s *SpikeShaper) GetSpikeDuration() time.Duration {
	return s.config.Spike.SpikeDuration
}

// GetSpikeQPS returns the QPS during spikes.
func (s *SpikeShaper) GetSpikeQPS() float64 {
	return s.config.Spike.SpikeQPS
}

// GetBaseQPS returns the base QPS (outside of spikes).
func (s *SpikeShaper) GetBaseQPS() float64 {
	return s.config.BaseQPS
}

// TimeUntilNextSpike returns the duration until the next spike begins.
func (s *SpikeShaper) TimeUntilNextSpike(elapsed time.Duration) time.Duration {
	posInInterval := elapsed % s.config.Spike.SpikeInterval

	if posInInterval < s.config.Spike.SpikeDuration {
		// Currently in a spike, return time until next spike starts
		return s.config.Spike.SpikeInterval - posInInterval
	}

	// Not in a spike, return remaining time until next spike
	return s.config.Spike.SpikeInterval - posInInterval
}

// RemainingSpikeDuration returns the remaining duration of the current spike.
// Returns 0 if not currently in a spike.
func (s *SpikeShaper) RemainingSpikeDuration(elapsed time.Duration) time.Duration {
	if !s.isInSpike(elapsed) {
		return 0
	}

	posInInterval := elapsed % s.config.Spike.SpikeInterval
	return s.config.Spike.SpikeDuration - posInInterval
}

// CurrentSpikeNumber returns which spike number we're in or approaching (1-indexed).
func (s *SpikeShaper) CurrentSpikeNumber(elapsed time.Duration) int {
	return int(elapsed/s.config.Spike.SpikeInterval) + 1
}
