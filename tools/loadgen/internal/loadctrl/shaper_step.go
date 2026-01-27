// Package loadctrl provides load control components including traffic shaping.
package loadctrl

import (
	"fmt"
	"time"
)

// StepShaper implements a staircase/step traffic pattern.
// QPS changes in discrete steps, optionally with smooth ramps between steps.
//
// Pattern (with ramps):
//
//	       Step3
//	       ____
//	      /
//	Step2
//	____/
//	   /
//	Step1
//	____
//
// Pattern (without ramps - instant transitions):
//
//	     Step3
//	     ____
//	     |
//	Step2|
//	____|
//	    |
//	Step1|
//	____
//
// Thread Safety: Safe for concurrent use by multiple goroutines (read-only after creation).
type StepShaper struct {
	config          ShaperConfig
	totalDuration   time.Duration
	cumulativeTimes []time.Duration // cumulative end time for each step
	previousQPS     []float64       // QPS of previous step (for ramping)
}

// NewStepShaper creates a new step traffic shaper.
// Steps are executed in order. If loop is true, the pattern repeats after completion.
func NewStepShaper(config ShaperConfig) (*StepShaper, error) {
	if config.Type != "step" {
		return nil, fmt.Errorf("expected type 'step', got '%s'", config.Type)
	}

	if config.Step == nil {
		return nil, fmt.Errorf("step configuration is required")
	}

	if len(config.Step.Steps) == 0 {
		return nil, fmt.Errorf("at least one step is required")
	}

	// Validate steps
	for i, step := range config.Step.Steps {
		if step.QPS < 0 {
			return nil, fmt.Errorf("step %d: QPS cannot be negative: %f", i, step.QPS)
		}
		if step.Duration <= 0 {
			return nil, fmt.Errorf("step %d: duration must be positive: %v", i, step.Duration)
		}
		if step.RampDuration < 0 {
			return nil, fmt.Errorf("step %d: ramp duration cannot be negative: %v", i, step.RampDuration)
		}
		if step.RampDuration > step.Duration {
			return nil, fmt.Errorf("step %d: ramp duration (%v) cannot exceed step duration (%v)",
				i, step.RampDuration, step.Duration)
		}
	}

	// Calculate cumulative times and total duration
	var total time.Duration
	cumulativeTimes := make([]time.Duration, len(config.Step.Steps))
	previousQPS := make([]float64, len(config.Step.Steps))

	for i, step := range config.Step.Steps {
		total += step.Duration
		cumulativeTimes[i] = total

		// Store the QPS of the previous step for ramping calculations
		if i == 0 {
			// First step ramps from baseQPS (or first step's QPS if no ramp)
			previousQPS[i] = config.BaseQPS
		} else {
			previousQPS[i] = config.Step.Steps[i-1].QPS
		}
	}

	return &StepShaper{
		config:          config,
		totalDuration:   total,
		cumulativeTimes: cumulativeTimes,
		previousQPS:     previousQPS,
	}, nil
}

// GetTargetQPS returns the target QPS for the given elapsed time.
// If in a ramp phase, returns linearly interpolated QPS.
// If in a hold phase, returns the step's target QPS.
func (s *StepShaper) GetTargetQPS(elapsed time.Duration) float64 {
	stepIdx, posInStep := s.getStepAndPosition(elapsed)
	step := s.config.Step.Steps[stepIdx]

	var qps float64

	if step.RampDuration > 0 && posInStep < step.RampDuration {
		// In ramp phase - linear interpolation
		prevQPS := s.previousQPS[stepIdx]
		progress := float64(posInStep) / float64(step.RampDuration)
		qps = prevQPS + (step.QPS-prevQPS)*progress
	} else {
		// In hold phase
		qps = step.QPS
	}

	return clampQPS(qps, s.config.MinQPS, s.config.MaxQPS)
}

// GetPhase returns a human-readable description of the current phase.
func (s *StepShaper) GetPhase(elapsed time.Duration) string {
	stepIdx, posInStep := s.getStepAndPosition(elapsed)
	step := s.config.Step.Steps[stepIdx]

	cycleNum := 1
	if s.config.Step.Loop && s.totalDuration > 0 {
		cycleNum = int(elapsed/s.totalDuration) + 1
	}

	progressInStep := float64(posInStep) / float64(step.Duration) * 100

	if step.RampDuration > 0 && posInStep < step.RampDuration {
		rampProgress := float64(posInStep) / float64(step.RampDuration) * 100
		return fmt.Sprintf("cycle %d, step %d/%d: ramping (%.1f%% ramp, %.1f%% step)",
			cycleNum, stepIdx+1, len(s.config.Step.Steps), rampProgress, progressInStep)
	}

	return fmt.Sprintf("cycle %d, step %d/%d: holding at %.0f QPS (%.1f%% through step)",
		cycleNum, stepIdx+1, len(s.config.Step.Steps), step.QPS, progressInStep)
}

// Name returns the name of this shaper type.
func (s *StepShaper) Name() string {
	return "step"
}

// Config returns a copy of the shaper's configuration.
func (s *StepShaper) Config() ShaperConfig {
	return s.config
}

// getStepAndPosition returns the current step index and position within that step.
func (s *StepShaper) getStepAndPosition(elapsed time.Duration) (stepIdx int, posInStep time.Duration) {
	// Handle looping
	effectiveElapsed := elapsed
	if s.config.Step.Loop && s.totalDuration > 0 {
		effectiveElapsed = elapsed % s.totalDuration
	} else if elapsed >= s.totalDuration {
		// Not looping and past the end - stay at last step
		stepIdx = len(s.config.Step.Steps) - 1
		posInStep = s.config.Step.Steps[stepIdx].Duration
		return stepIdx, posInStep
	}

	// Find which step we're in
	var stepStart time.Duration
	for i, cumTime := range s.cumulativeTimes {
		if effectiveElapsed < cumTime {
			posInStep = effectiveElapsed - stepStart
			return i, posInStep
		}
		stepStart = cumTime
	}

	// Should not reach here, but default to last step
	stepIdx = len(s.config.Step.Steps) - 1
	posInStep = s.config.Step.Steps[stepIdx].Duration
	return stepIdx, posInStep
}

// GetTotalDuration returns the total duration of all steps.
func (s *StepShaper) GetTotalDuration() time.Duration {
	return s.totalDuration
}

// IsLooping returns whether the shaper is configured to loop.
func (s *StepShaper) IsLooping() bool {
	return s.config.Step.Loop
}

// GetStepCount returns the number of steps.
func (s *StepShaper) GetStepCount() int {
	return len(s.config.Step.Steps)
}

// GetStep returns the step at the given index.
func (s *StepShaper) GetStep(idx int) (StepLevel, bool) {
	if idx < 0 || idx >= len(s.config.Step.Steps) {
		return StepLevel{}, false
	}
	return s.config.Step.Steps[idx], true
}

// CurrentStepIndex returns the index of the current step (0-indexed).
func (s *StepShaper) CurrentStepIndex(elapsed time.Duration) int {
	stepIdx, _ := s.getStepAndPosition(elapsed)
	return stepIdx
}

// TimeUntilNextStep returns the duration until the next step begins.
// Returns 0 if at the last step and not looping.
func (s *StepShaper) TimeUntilNextStep(elapsed time.Duration) time.Duration {
	stepIdx, posInStep := s.getStepAndPosition(elapsed)
	step := s.config.Step.Steps[stepIdx]

	remaining := step.Duration - posInStep

	// If at the last step and not looping, return 0 after step ends
	if stepIdx == len(s.config.Step.Steps)-1 && !s.config.Step.Loop {
		if posInStep >= step.Duration {
			return 0
		}
	}

	return remaining
}
