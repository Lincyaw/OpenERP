package loadctrl

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// TrafficShaper Interface Tests
// ============================================================================

func TestNewTrafficShaper(t *testing.T) {
	tests := []struct {
		name        string
		config      ShaperConfig
		expectError bool
		errorMsg    string
		shaperType  string
	}{
		{
			name: "valid sine config",
			config: ShaperConfig{
				Type:      "sine",
				BaseQPS:   100,
				Amplitude: 50,
				Period:    60 * time.Second,
			},
			expectError: false,
			shaperType:  "sine",
		},
		{
			name: "valid spike config",
			config: ShaperConfig{
				Type:    "spike",
				BaseQPS: 100,
				Spike: &SpikeConfig{
					SpikeQPS:      500,
					SpikeDuration: 5 * time.Second,
					SpikeInterval: 30 * time.Second,
				},
			},
			expectError: false,
			shaperType:  "spike",
		},
		{
			name: "valid step config",
			config: ShaperConfig{
				Type:    "step",
				BaseQPS: 50,
				Step: &StepConfig{
					Steps: []StepLevel{
						{QPS: 100, Duration: 30 * time.Second},
						{QPS: 200, Duration: 30 * time.Second},
					},
					Loop: false,
				},
			},
			expectError: false,
			shaperType:  "step",
		},
		{
			name: "valid custom config",
			config: ShaperConfig{
				Type: "custom",
				CustomPoints: []CustomPoint{
					{Time: 0, QPS: 50},
					{Time: 30 * time.Second, QPS: 150},
					{Time: 60 * time.Second, QPS: 100},
				},
			},
			expectError: false,
			shaperType:  "custom",
		},
		{
			name: "unknown type",
			config: ShaperConfig{
				Type:    "unknown",
				BaseQPS: 100,
			},
			expectError: true,
			errorMsg:    "unknown shaper type",
		},
		{
			name: "missing type",
			config: ShaperConfig{
				BaseQPS: 100,
			},
			expectError: true,
			errorMsg:    "shaper type is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shaper, err := NewTrafficShaper(tt.config)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.shaperType, shaper.Name())
			}
		})
	}
}

func TestShaperConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      ShaperConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "missing type",
			config:      ShaperConfig{BaseQPS: 100},
			expectError: true,
			errorMsg:    "shaper type is required",
		},
		{
			name:        "negative baseQPS",
			config:      ShaperConfig{Type: "sine", BaseQPS: -10, Period: time.Second},
			expectError: true,
			errorMsg:    "baseQPS cannot be negative",
		},
		{
			name:        "sine without period",
			config:      ShaperConfig{Type: "sine", BaseQPS: 100},
			expectError: true,
			errorMsg:    "positive period",
		},
		{
			name:        "spike without config",
			config:      ShaperConfig{Type: "spike", BaseQPS: 100},
			expectError: true,
			errorMsg:    "spike configuration",
		},
		{
			name: "spike duration >= interval",
			config: ShaperConfig{
				Type:    "spike",
				BaseQPS: 100,
				Spike: &SpikeConfig{
					SpikeQPS:      500,
					SpikeDuration: 30 * time.Second,
					SpikeInterval: 30 * time.Second,
				},
			},
			expectError: true,
			errorMsg:    "must be less than",
		},
		{
			name:        "step without config",
			config:      ShaperConfig{Type: "step", BaseQPS: 100},
			expectError: true,
			errorMsg:    "step configuration",
		},
		{
			name: "step with no steps",
			config: ShaperConfig{
				Type:    "step",
				BaseQPS: 100,
				Step:    &StepConfig{Steps: []StepLevel{}},
			},
			expectError: true,
			errorMsg:    "at least one step",
		},
		{
			name: "custom with single point",
			config: ShaperConfig{
				Type:         "custom",
				CustomPoints: []CustomPoint{{Time: 0, QPS: 100}},
			},
			expectError: true,
			errorMsg:    "at least 2 points",
		},
		{
			name: "custom with unordered points",
			config: ShaperConfig{
				Type: "custom",
				CustomPoints: []CustomPoint{
					{Time: 30 * time.Second, QPS: 100},
					{Time: 10 * time.Second, QPS: 50},
				},
			},
			expectError: true,
			errorMsg:    "chronological order",
		},
		{
			name: "minQPS exceeds maxQPS",
			config: ShaperConfig{
				Type:    "sine",
				BaseQPS: 100,
				MinQPS:  200,
				MaxQPS:  100,
				Period:  time.Second,
			},
			expectError: true,
			errorMsg:    "cannot exceed maxQPS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ============================================================================
// SineWaveShaper Tests
// ============================================================================

func TestSineWaveShaper_Basic(t *testing.T) {
	config := ShaperConfig{
		Type:      "sine",
		BaseQPS:   100,
		Amplitude: 50, // absolute amplitude
		Period:    60 * time.Second,
	}

	shaper, err := NewSineWaveShaper(config)
	require.NoError(t, err)

	// At t=0, sin(0) = 0, so QPS = base = 100
	assert.InDelta(t, 100.0, shaper.GetTargetQPS(0), 0.001)

	// At t=15s (1/4 period), sin(π/2) = 1, so QPS = base + amplitude = 150
	assert.InDelta(t, 150.0, shaper.GetTargetQPS(15*time.Second), 0.001)

	// At t=30s (1/2 period), sin(π) = 0, so QPS = base = 100
	assert.InDelta(t, 100.0, shaper.GetTargetQPS(30*time.Second), 0.001)

	// At t=45s (3/4 period), sin(3π/2) = -1, so QPS = base - amplitude = 50
	assert.InDelta(t, 50.0, shaper.GetTargetQPS(45*time.Second), 0.001)

	// At t=60s (full period), sin(2π) = 0, so QPS = base = 100
	assert.InDelta(t, 100.0, shaper.GetTargetQPS(60*time.Second), 0.001)
}

func TestSineWaveShaper_AcceptanceCriteria(t *testing.T) {
	// Acceptance criteria: 60s period ±50% amplitude
	config := ShaperConfig{
		Type:      "sine",
		BaseQPS:   100,
		Amplitude: 0.5, // 50% relative amplitude
		Period:    60 * time.Second,
	}

	shaper, err := NewSineWaveShaper(config)
	require.NoError(t, err)

	// Effective amplitude = 100 * 0.5 = 50
	assert.InDelta(t, 50.0, shaper.GetAmplitude(), 0.001)

	// At t=0, QPS = 100
	assert.InDelta(t, 100.0, shaper.GetTargetQPS(0), 0.001)

	// At t=15s (peak), QPS = 150 (+50%)
	assert.InDelta(t, 150.0, shaper.GetTargetQPS(15*time.Second), 0.001)

	// At t=45s (trough), QPS = 50 (-50%)
	assert.InDelta(t, 50.0, shaper.GetTargetQPS(45*time.Second), 0.001)

	// Verify min/max
	min, max := shaper.GetMinMaxQPS()
	assert.InDelta(t, 50.0, min, 0.001)
	assert.InDelta(t, 150.0, max, 0.001)
}

func TestSineWaveShaper_RelativeAmplitude(t *testing.T) {
	// Test that amplitude <= 1 is treated as percentage
	config := ShaperConfig{
		Type:      "sine",
		BaseQPS:   200,
		Amplitude: 0.25, // 25% = 50 QPS
		Period:    10 * time.Second,
	}

	shaper, err := NewSineWaveShaper(config)
	require.NoError(t, err)

	assert.InDelta(t, 50.0, shaper.GetAmplitude(), 0.001)

	// At peak (t=2.5s for 10s period): 200 + 50 = 250
	assert.InDelta(t, 250.0, shaper.GetTargetQPS(2500*time.Millisecond), 0.001)

	// At trough (t=7.5s): 200 - 50 = 150
	assert.InDelta(t, 150.0, shaper.GetTargetQPS(7500*time.Millisecond), 0.001)
}

func TestSineWaveShaper_AbsoluteAmplitude(t *testing.T) {
	// Test that amplitude > 1 is treated as absolute
	config := ShaperConfig{
		Type:      "sine",
		BaseQPS:   100,
		Amplitude: 30, // absolute 30 QPS
		Period:    10 * time.Second,
	}

	shaper, err := NewSineWaveShaper(config)
	require.NoError(t, err)

	assert.InDelta(t, 30.0, shaper.GetAmplitude(), 0.001)

	// At peak: 100 + 30 = 130
	assert.InDelta(t, 130.0, shaper.GetTargetQPS(2500*time.Millisecond), 0.001)

	// At trough: 100 - 30 = 70
	assert.InDelta(t, 70.0, shaper.GetTargetQPS(7500*time.Millisecond), 0.001)
}

func TestSineWaveShaper_MinMaxClamping(t *testing.T) {
	config := ShaperConfig{
		Type:      "sine",
		BaseQPS:   100,
		Amplitude: 80, // would go to 20 and 180
		MinQPS:    50,
		MaxQPS:    150,
		Period:    60 * time.Second,
	}

	shaper, err := NewSineWaveShaper(config)
	require.NoError(t, err)

	// At peak (would be 180, clamped to 150)
	assert.InDelta(t, 150.0, shaper.GetTargetQPS(15*time.Second), 0.001)

	// At trough (would be 20, clamped to 50)
	assert.InDelta(t, 50.0, shaper.GetTargetQPS(45*time.Second), 0.001)
}

func TestSineWaveShaper_NeverNegative(t *testing.T) {
	// Even without clamping, QPS should never go negative
	config := ShaperConfig{
		Type:      "sine",
		BaseQPS:   30,
		Amplitude: 50, // would go to -20 at trough
		Period:    10 * time.Second,
	}

	shaper, err := NewSineWaveShaper(config)
	require.NoError(t, err)

	// At trough (would be -20, but should be 0)
	assert.InDelta(t, 0.0, shaper.GetTargetQPS(7500*time.Millisecond), 0.001)
}

func TestSineWaveShaper_Phase(t *testing.T) {
	config := ShaperConfig{
		Type:      "sine",
		BaseQPS:   100,
		Amplitude: 50,
		Period:    60 * time.Second,
	}

	shaper, err := NewSineWaveShaper(config)
	require.NoError(t, err)

	// Check phase descriptions at different times
	phase0 := shaper.GetPhase(0)
	assert.Contains(t, phase0, "rising to peak")

	phase15 := shaper.GetPhase(15 * time.Second)
	assert.Contains(t, phase15, "falling from peak")

	phase45 := shaper.GetPhase(45 * time.Second)
	assert.Contains(t, phase45, "rising from trough")

	// Period 2
	phase65 := shaper.GetPhase(65 * time.Second)
	assert.Contains(t, phase65, "period 2")
}

func TestSineWaveShaper_Metadata(t *testing.T) {
	config := ShaperConfig{
		Type:      "sine",
		BaseQPS:   100,
		Amplitude: 50,
		Period:    60 * time.Second,
	}

	shaper, err := NewSineWaveShaper(config)
	require.NoError(t, err)

	assert.Equal(t, "sine", shaper.Name())
	assert.Equal(t, 60*time.Second, shaper.GetPeriod())
	assert.Equal(t, config.Type, shaper.Config().Type)
}

// ============================================================================
// SpikeShaper Tests
// ============================================================================

func TestSpikeShaper_Basic(t *testing.T) {
	config := ShaperConfig{
		Type:    "spike",
		BaseQPS: 100,
		Spike: &SpikeConfig{
			SpikeQPS:      500,
			SpikeDuration: 5 * time.Second,
			SpikeInterval: 30 * time.Second,
		},
	}

	shaper, err := NewSpikeShaper(config)
	require.NoError(t, err)

	// During first spike (0-5s): 500 QPS
	assert.InDelta(t, 500.0, shaper.GetTargetQPS(0), 0.001)
	assert.InDelta(t, 500.0, shaper.GetTargetQPS(2*time.Second), 0.001)
	assert.InDelta(t, 500.0, shaper.GetTargetQPS(4*time.Second+999*time.Millisecond), 0.001)

	// After first spike (5-30s): 100 QPS
	assert.InDelta(t, 100.0, shaper.GetTargetQPS(5*time.Second), 0.001)
	assert.InDelta(t, 100.0, shaper.GetTargetQPS(15*time.Second), 0.001)
	assert.InDelta(t, 100.0, shaper.GetTargetQPS(29*time.Second+999*time.Millisecond), 0.001)

	// During second spike (30-35s): 500 QPS
	assert.InDelta(t, 500.0, shaper.GetTargetQPS(30*time.Second), 0.001)
	assert.InDelta(t, 500.0, shaper.GetTargetQPS(32*time.Second), 0.001)
}

func TestSpikeShaper_Phase(t *testing.T) {
	config := ShaperConfig{
		Type:    "spike",
		BaseQPS: 100,
		Spike: &SpikeConfig{
			SpikeQPS:      500,
			SpikeDuration: 5 * time.Second,
			SpikeInterval: 30 * time.Second,
		},
	}

	shaper, err := NewSpikeShaper(config)
	require.NoError(t, err)

	// During spike
	phase1 := shaper.GetPhase(2 * time.Second)
	assert.Contains(t, phase1, "active")
	assert.Contains(t, phase1, "spike 1")

	// After spike
	phase2 := shaper.GetPhase(10 * time.Second)
	assert.Contains(t, phase2, "normal")

	// Second spike
	phase3 := shaper.GetPhase(31 * time.Second)
	assert.Contains(t, phase3, "spike 2")
}

func TestSpikeShaper_Metadata(t *testing.T) {
	config := ShaperConfig{
		Type:    "spike",
		BaseQPS: 100,
		Spike: &SpikeConfig{
			SpikeQPS:      500,
			SpikeDuration: 5 * time.Second,
			SpikeInterval: 30 * time.Second,
		},
	}

	shaper, err := NewSpikeShaper(config)
	require.NoError(t, err)

	assert.Equal(t, "spike", shaper.Name())
	assert.Equal(t, 30*time.Second, shaper.GetSpikeInterval())
	assert.Equal(t, 5*time.Second, shaper.GetSpikeDuration())
	assert.InDelta(t, 500.0, shaper.GetSpikeQPS(), 0.001)
	assert.InDelta(t, 100.0, shaper.GetBaseQPS(), 0.001)
}

func TestSpikeShaper_TimeCalculations(t *testing.T) {
	config := ShaperConfig{
		Type:    "spike",
		BaseQPS: 100,
		Spike: &SpikeConfig{
			SpikeQPS:      500,
			SpikeDuration: 5 * time.Second,
			SpikeInterval: 30 * time.Second,
		},
	}

	shaper, err := NewSpikeShaper(config)
	require.NoError(t, err)

	// During spike: remaining duration should decrease
	assert.Equal(t, 3*time.Second, shaper.RemainingSpikeDuration(2*time.Second))
	assert.Equal(t, time.Duration(0), shaper.RemainingSpikeDuration(10*time.Second))

	// Spike numbers
	assert.Equal(t, 1, shaper.CurrentSpikeNumber(0))
	assert.Equal(t, 1, shaper.CurrentSpikeNumber(29*time.Second))
	assert.Equal(t, 2, shaper.CurrentSpikeNumber(30*time.Second))
}

func TestSpikeShaper_MinMaxClamping(t *testing.T) {
	config := ShaperConfig{
		Type:    "spike",
		BaseQPS: 100,
		MinQPS:  50,
		MaxQPS:  300,
		Spike: &SpikeConfig{
			SpikeQPS:      500, // would exceed max
			SpikeDuration: 5 * time.Second,
			SpikeInterval: 30 * time.Second,
		},
	}

	shaper, err := NewSpikeShaper(config)
	require.NoError(t, err)

	// During spike: clamped to maxQPS
	assert.InDelta(t, 300.0, shaper.GetTargetQPS(2*time.Second), 0.001)

	// Normal: base QPS
	assert.InDelta(t, 100.0, shaper.GetTargetQPS(10*time.Second), 0.001)
}

// ============================================================================
// StepShaper Tests
// ============================================================================

func TestStepShaper_BasicSteps(t *testing.T) {
	config := ShaperConfig{
		Type:    "step",
		BaseQPS: 50,
		Step: &StepConfig{
			Steps: []StepLevel{
				{QPS: 100, Duration: 30 * time.Second},
				{QPS: 200, Duration: 30 * time.Second},
				{QPS: 150, Duration: 30 * time.Second},
			},
			Loop: false,
		},
	}

	shaper, err := NewStepShaper(config)
	require.NoError(t, err)

	// Step 1 (0-30s): 100 QPS
	assert.InDelta(t, 100.0, shaper.GetTargetQPS(0), 0.001)
	assert.InDelta(t, 100.0, shaper.GetTargetQPS(15*time.Second), 0.001)
	assert.InDelta(t, 100.0, shaper.GetTargetQPS(29*time.Second), 0.001)

	// Step 2 (30-60s): 200 QPS
	assert.InDelta(t, 200.0, shaper.GetTargetQPS(30*time.Second), 0.001)
	assert.InDelta(t, 200.0, shaper.GetTargetQPS(45*time.Second), 0.001)

	// Step 3 (60-90s): 150 QPS
	assert.InDelta(t, 150.0, shaper.GetTargetQPS(60*time.Second), 0.001)
	assert.InDelta(t, 150.0, shaper.GetTargetQPS(75*time.Second), 0.001)

	// After all steps: stays at last QPS
	assert.InDelta(t, 150.0, shaper.GetTargetQPS(100*time.Second), 0.001)
}

func TestStepShaper_WithRamps(t *testing.T) {
	config := ShaperConfig{
		Type:    "step",
		BaseQPS: 50, // starting QPS for first ramp
		Step: &StepConfig{
			Steps: []StepLevel{
				{QPS: 100, Duration: 30 * time.Second, RampDuration: 10 * time.Second},
				{QPS: 200, Duration: 30 * time.Second, RampDuration: 10 * time.Second},
			},
			Loop: false,
		},
	}

	shaper, err := NewStepShaper(config)
	require.NoError(t, err)

	// Step 1: ramp from baseQPS (50) to 100 over 10s
	// At t=0: start of ramp, QPS = 50
	assert.InDelta(t, 50.0, shaper.GetTargetQPS(0), 0.001)

	// At t=5s: halfway through ramp, QPS = 75
	assert.InDelta(t, 75.0, shaper.GetTargetQPS(5*time.Second), 0.001)

	// At t=10s: end of ramp, QPS = 100
	assert.InDelta(t, 100.0, shaper.GetTargetQPS(10*time.Second), 0.001)

	// At t=20s: hold phase, QPS = 100
	assert.InDelta(t, 100.0, shaper.GetTargetQPS(20*time.Second), 0.001)

	// Step 2: ramp from 100 to 200 over 10s
	// At t=30s: start of step 2 ramp, QPS = 100
	assert.InDelta(t, 100.0, shaper.GetTargetQPS(30*time.Second), 0.001)

	// At t=35s: halfway through ramp, QPS = 150
	assert.InDelta(t, 150.0, shaper.GetTargetQPS(35*time.Second), 0.001)

	// At t=40s: end of ramp, QPS = 200
	assert.InDelta(t, 200.0, shaper.GetTargetQPS(40*time.Second), 0.001)
}

func TestStepShaper_Looping(t *testing.T) {
	config := ShaperConfig{
		Type:    "step",
		BaseQPS: 50,
		Step: &StepConfig{
			Steps: []StepLevel{
				{QPS: 100, Duration: 30 * time.Second},
				{QPS: 200, Duration: 30 * time.Second},
			},
			Loop: true,
		},
	}

	shaper, err := NewStepShaper(config)
	require.NoError(t, err)

	// Total duration is 60s
	assert.Equal(t, 60*time.Second, shaper.GetTotalDuration())

	// First cycle
	assert.InDelta(t, 100.0, shaper.GetTargetQPS(15*time.Second), 0.001)
	assert.InDelta(t, 200.0, shaper.GetTargetQPS(45*time.Second), 0.001)

	// Second cycle (should loop back)
	assert.InDelta(t, 100.0, shaper.GetTargetQPS(75*time.Second), 0.001)  // 75 = 60 + 15
	assert.InDelta(t, 200.0, shaper.GetTargetQPS(105*time.Second), 0.001) // 105 = 60 + 45
}

func TestStepShaper_Phase(t *testing.T) {
	config := ShaperConfig{
		Type:    "step",
		BaseQPS: 50,
		Step: &StepConfig{
			Steps: []StepLevel{
				{QPS: 100, Duration: 30 * time.Second, RampDuration: 10 * time.Second},
				{QPS: 200, Duration: 30 * time.Second},
			},
			Loop: false,
		},
	}

	shaper, err := NewStepShaper(config)
	require.NoError(t, err)

	// During ramp phase
	phase1 := shaper.GetPhase(5 * time.Second)
	assert.Contains(t, phase1, "ramping")
	assert.Contains(t, phase1, "step 1/2")

	// During hold phase
	phase2 := shaper.GetPhase(20 * time.Second)
	assert.Contains(t, phase2, "holding")
	assert.Contains(t, phase2, "step 1/2")

	// Second step
	phase3 := shaper.GetPhase(40 * time.Second)
	assert.Contains(t, phase3, "step 2/2")
}

func TestStepShaper_Metadata(t *testing.T) {
	config := ShaperConfig{
		Type:    "step",
		BaseQPS: 50,
		Step: &StepConfig{
			Steps: []StepLevel{
				{QPS: 100, Duration: 30 * time.Second},
				{QPS: 200, Duration: 30 * time.Second},
			},
			Loop: true,
		},
	}

	shaper, err := NewStepShaper(config)
	require.NoError(t, err)

	assert.Equal(t, "step", shaper.Name())
	assert.Equal(t, 60*time.Second, shaper.GetTotalDuration())
	assert.True(t, shaper.IsLooping())
	assert.Equal(t, 2, shaper.GetStepCount())

	step, ok := shaper.GetStep(0)
	assert.True(t, ok)
	assert.InDelta(t, 100.0, step.QPS, 0.001)
}

func TestStepShaper_TimeCalculations(t *testing.T) {
	config := ShaperConfig{
		Type:    "step",
		BaseQPS: 50,
		Step: &StepConfig{
			Steps: []StepLevel{
				{QPS: 100, Duration: 30 * time.Second},
				{QPS: 200, Duration: 30 * time.Second},
			},
			Loop: false,
		},
	}

	shaper, err := NewStepShaper(config)
	require.NoError(t, err)

	// Step index
	assert.Equal(t, 0, shaper.CurrentStepIndex(15*time.Second))
	assert.Equal(t, 1, shaper.CurrentStepIndex(45*time.Second))

	// Time until next step
	assert.Equal(t, 20*time.Second, shaper.TimeUntilNextStep(10*time.Second))
}

// ============================================================================
// CustomShaper Tests
// ============================================================================

func TestCustomShaper_Basic(t *testing.T) {
	config := ShaperConfig{
		Type: "custom",
		CustomPoints: []CustomPoint{
			{Time: 0, QPS: 50},
			{Time: 30 * time.Second, QPS: 150},
			{Time: 60 * time.Second, QPS: 100},
		},
	}

	shaper, err := NewCustomShaper(config)
	require.NoError(t, err)

	// At defined points
	assert.InDelta(t, 50.0, shaper.GetTargetQPS(0), 0.001)
	assert.InDelta(t, 150.0, shaper.GetTargetQPS(30*time.Second), 0.001)
	assert.InDelta(t, 100.0, shaper.GetTargetQPS(60*time.Second), 0.001)

	// Linear interpolation between points
	// At t=15s: halfway between 50 and 150 = 100
	assert.InDelta(t, 100.0, shaper.GetTargetQPS(15*time.Second), 0.001)

	// At t=45s: halfway between 150 and 100 = 125
	assert.InDelta(t, 125.0, shaper.GetTargetQPS(45*time.Second), 0.001)
}

func TestCustomShaper_BeforeAndAfterCurve(t *testing.T) {
	config := ShaperConfig{
		Type: "custom",
		CustomPoints: []CustomPoint{
			{Time: 10 * time.Second, QPS: 50},  // starts at 10s
			{Time: 30 * time.Second, QPS: 150}, // ends at 30s
		},
	}

	shaper, err := NewCustomShaper(config)
	require.NoError(t, err)

	// Before first point: use first point's QPS
	assert.InDelta(t, 50.0, shaper.GetTargetQPS(0), 0.001)
	assert.InDelta(t, 50.0, shaper.GetTargetQPS(5*time.Second), 0.001)

	// After last point: use last point's QPS
	assert.InDelta(t, 150.0, shaper.GetTargetQPS(30*time.Second), 0.001)
	assert.InDelta(t, 150.0, shaper.GetTargetQPS(60*time.Second), 0.001)
}

func TestCustomShaper_Phase(t *testing.T) {
	config := ShaperConfig{
		Type: "custom",
		CustomPoints: []CustomPoint{
			{Time: 0, QPS: 50},
			{Time: 30 * time.Second, QPS: 150},
			{Time: 60 * time.Second, QPS: 100},
		},
	}

	shaper, err := NewCustomShaper(config)
	require.NoError(t, err)

	// Ramping up
	phase1 := shaper.GetPhase(15 * time.Second)
	assert.Contains(t, phase1, "ramping up")
	assert.Contains(t, phase1, "segment 1/2")

	// Ramping down
	phase2 := shaper.GetPhase(45 * time.Second)
	assert.Contains(t, phase2, "ramping down")
	assert.Contains(t, phase2, "segment 2/2")

	// After curve complete
	phase3 := shaper.GetPhase(90 * time.Second)
	assert.Contains(t, phase3, "complete")
}

func TestCustomShaper_Metadata(t *testing.T) {
	config := ShaperConfig{
		Type: "custom",
		CustomPoints: []CustomPoint{
			{Time: 0, QPS: 50},
			{Time: 30 * time.Second, QPS: 150},
			{Time: 60 * time.Second, QPS: 100},
		},
	}

	shaper, err := NewCustomShaper(config)
	require.NoError(t, err)

	assert.Equal(t, "custom", shaper.Name())
	assert.Equal(t, 60*time.Second, shaper.GetTotalDuration())
	assert.Equal(t, 3, shaper.GetPointCount())

	pt, ok := shaper.GetPoint(1)
	assert.True(t, ok)
	assert.InDelta(t, 150.0, pt.QPS, 0.001)

	min, max := shaper.GetMinMaxQPS()
	assert.InDelta(t, 50.0, min, 0.001)
	assert.InDelta(t, 150.0, max, 0.001)
}

func TestCustomShaper_SortingPoints(t *testing.T) {
	// Points provided out of order should be sorted
	// Note: Validation rejects unordered points, but NewCustomShaper sorts them
	config := ShaperConfig{
		Type: "custom",
		CustomPoints: []CustomPoint{
			{Time: 0, QPS: 50},
			{Time: 60 * time.Second, QPS: 100},
			{Time: 30 * time.Second, QPS: 150},
		},
	}

	// This should fail validation
	err := config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "chronological order")
}

func TestCustomShaper_SegmentCalculations(t *testing.T) {
	config := ShaperConfig{
		Type: "custom",
		CustomPoints: []CustomPoint{
			{Time: 0, QPS: 50},
			{Time: 30 * time.Second, QPS: 150},
			{Time: 60 * time.Second, QPS: 100},
		},
	}

	shaper, err := NewCustomShaper(config)
	require.NoError(t, err)

	// Current segment
	start, end := shaper.CurrentSegment(15 * time.Second)
	assert.Equal(t, 0, start)
	assert.Equal(t, 1, end)

	start, end = shaper.CurrentSegment(45 * time.Second)
	assert.Equal(t, 1, start)
	assert.Equal(t, 2, end)

	// Time until next point
	assert.Equal(t, 15*time.Second, shaper.TimeUntilNextPoint(15*time.Second))
	assert.Equal(t, time.Duration(0), shaper.TimeUntilNextPoint(90*time.Second))
}

func TestCustomShaper_MinMaxClamping(t *testing.T) {
	config := ShaperConfig{
		Type:   "custom",
		MinQPS: 75,
		MaxQPS: 125,
		CustomPoints: []CustomPoint{
			{Time: 0, QPS: 50},                 // would be clamped to 75
			{Time: 30 * time.Second, QPS: 150}, // would be clamped to 125
			{Time: 60 * time.Second, QPS: 100},
		},
	}

	shaper, err := NewCustomShaper(config)
	require.NoError(t, err)

	// At t=0: clamped from 50 to 75
	assert.InDelta(t, 75.0, shaper.GetTargetQPS(0), 0.001)

	// At t=30s: clamped from 150 to 125
	assert.InDelta(t, 125.0, shaper.GetTargetQPS(30*time.Second), 0.001)

	// At t=60s: 100 is within bounds
	assert.InDelta(t, 100.0, shaper.GetTargetQPS(60*time.Second), 0.001)
}

// ============================================================================
// Integration Tests - QPS Curve Verification
// ============================================================================

func TestSineWaveShaper_FullCycleAccuracy(t *testing.T) {
	// Test the acceptance criteria: 60s period ±50% amplitude
	config := ShaperConfig{
		Type:      "sine",
		BaseQPS:   100,
		Amplitude: 0.5, // 50% relative
		Period:    60 * time.Second,
	}

	shaper, err := NewSineWaveShaper(config)
	require.NoError(t, err)

	// Sample points throughout the cycle
	samples := 100
	for i := 0; i <= samples; i++ {
		elapsed := time.Duration(float64(i) / float64(samples) * float64(60*time.Second))
		qps := shaper.GetTargetQPS(elapsed)

		// Expected: 100 + 50 * sin(2π * t / 60s)
		expected := 100.0 + 50.0*math.Sin(2*math.Pi*float64(elapsed)/float64(60*time.Second))

		assert.InDelta(t, expected, qps, 0.001,
			"Mismatch at t=%v: expected %.3f, got %.3f", elapsed, expected, qps)
	}
}

func TestAllShapers_PositiveQPS(t *testing.T) {
	// Verify all shapers produce non-negative QPS values

	shapers := []TrafficShaper{}

	// Sine with extreme amplitude
	sineConfig := ShaperConfig{
		Type:      "sine",
		BaseQPS:   10,
		Amplitude: 50, // would go negative without protection
		Period:    10 * time.Second,
	}
	sine, _ := NewSineWaveShaper(sineConfig)
	shapers = append(shapers, sine)

	// Spike
	spikeConfig := ShaperConfig{
		Type:    "spike",
		BaseQPS: 0, // zero base
		Spike: &SpikeConfig{
			SpikeQPS:      100,
			SpikeDuration: 1 * time.Second,
			SpikeInterval: 5 * time.Second,
		},
	}
	spike, _ := NewSpikeShaper(spikeConfig)
	shapers = append(shapers, spike)

	// Step
	stepConfig := ShaperConfig{
		Type:    "step",
		BaseQPS: 0,
		Step: &StepConfig{
			Steps: []StepLevel{
				{QPS: 0, Duration: 10 * time.Second},
				{QPS: 100, Duration: 10 * time.Second, RampDuration: 5 * time.Second},
			},
		},
	}
	step, _ := NewStepShaper(stepConfig)
	shapers = append(shapers, step)

	// Custom
	customConfig := ShaperConfig{
		Type: "custom",
		CustomPoints: []CustomPoint{
			{Time: 0, QPS: 0},
			{Time: 30 * time.Second, QPS: 100},
		},
	}
	custom, _ := NewCustomShaper(customConfig)
	shapers = append(shapers, custom)

	// Test all shapers at various times
	for _, shaper := range shapers {
		t.Run(shaper.Name(), func(t *testing.T) {
			for elapsed := time.Duration(0); elapsed <= 60*time.Second; elapsed += time.Second {
				qps := shaper.GetTargetQPS(elapsed)
				assert.GreaterOrEqual(t, qps, 0.0,
					"Shaper %s produced negative QPS at t=%v: %f", shaper.Name(), elapsed, qps)
			}
		})
	}
}

// ============================================================================
// Additional Coverage Tests
// ============================================================================

func TestAllShapers_Config(t *testing.T) {
	// Test Config() method on all shapers
	configs := []ShaperConfig{
		{Type: "sine", BaseQPS: 100, Amplitude: 50, Period: 60 * time.Second},
		{Type: "spike", BaseQPS: 100, Spike: &SpikeConfig{
			SpikeQPS: 500, SpikeDuration: 5 * time.Second, SpikeInterval: 30 * time.Second,
		}},
		{Type: "step", BaseQPS: 50, Step: &StepConfig{
			Steps: []StepLevel{{QPS: 100, Duration: 30 * time.Second}},
		}},
		{Type: "custom", CustomPoints: []CustomPoint{
			{Time: 0, QPS: 50}, {Time: 30 * time.Second, QPS: 100},
		}},
	}

	for _, cfg := range configs {
		t.Run(cfg.Type, func(t *testing.T) {
			shaper, err := NewTrafficShaper(cfg)
			require.NoError(t, err)

			returned := shaper.Config()
			assert.Equal(t, cfg.Type, returned.Type)
			assert.Equal(t, cfg.BaseQPS, returned.BaseQPS)
		})
	}
}

func TestSpikeShaper_TimeUntilNextSpike(t *testing.T) {
	config := ShaperConfig{
		Type:    "spike",
		BaseQPS: 100,
		Spike: &SpikeConfig{
			SpikeQPS:      500,
			SpikeDuration: 5 * time.Second,
			SpikeInterval: 30 * time.Second,
		},
	}
	shaper, err := NewSpikeShaper(config)
	require.NoError(t, err)

	// During spike at t=2s, next spike at t=30s (28s away)
	assert.Equal(t, 28*time.Second, shaper.TimeUntilNextSpike(2*time.Second))

	// After spike at t=10s, next spike at t=30s (20s away)
	assert.Equal(t, 20*time.Second, shaper.TimeUntilNextSpike(10*time.Second))

	// At exactly spike start
	assert.Equal(t, 30*time.Second, shaper.TimeUntilNextSpike(0))

	// Right before next spike
	assert.Equal(t, 1*time.Second, shaper.TimeUntilNextSpike(29*time.Second))
}

func TestCustomShaper_GetPoints(t *testing.T) {
	config := ShaperConfig{
		Type: "custom",
		CustomPoints: []CustomPoint{
			{Time: 0, QPS: 50},
			{Time: 30 * time.Second, QPS: 150},
			{Time: 60 * time.Second, QPS: 100},
		},
	}

	shaper, err := NewCustomShaper(config)
	require.NoError(t, err)

	points := shaper.GetPoints()
	assert.Len(t, points, 3)
	assert.Equal(t, 50.0, points[0].QPS)
	assert.Equal(t, 150.0, points[1].QPS)
	assert.Equal(t, 100.0, points[2].QPS)

	// Verify it returns a copy (modifying returned slice shouldn't affect shaper)
	points[0].QPS = 999
	originalPoints := shaper.GetPoints()
	assert.Equal(t, 50.0, originalPoints[0].QPS)
}

func TestStepShaper_GetStep_Invalid(t *testing.T) {
	config := ShaperConfig{
		Type:    "step",
		BaseQPS: 50,
		Step: &StepConfig{
			Steps: []StepLevel{
				{QPS: 100, Duration: 30 * time.Second},
			},
		},
	}

	shaper, err := NewStepShaper(config)
	require.NoError(t, err)

	// Invalid indices
	_, ok := shaper.GetStep(-1)
	assert.False(t, ok)

	_, ok = shaper.GetStep(10)
	assert.False(t, ok)
}

func TestCustomShaper_GetPoint_Invalid(t *testing.T) {
	config := ShaperConfig{
		Type: "custom",
		CustomPoints: []CustomPoint{
			{Time: 0, QPS: 50},
			{Time: 30 * time.Second, QPS: 100},
		},
	}

	shaper, err := NewCustomShaper(config)
	require.NoError(t, err)

	// Invalid indices
	_, ok := shaper.GetPoint(-1)
	assert.False(t, ok)

	_, ok = shaper.GetPoint(10)
	assert.False(t, ok)
}

func TestNewShaperConstructor_Errors(t *testing.T) {
	// Test error cases for individual constructors

	// SineWaveShaper wrong type
	_, err := NewSineWaveShaper(ShaperConfig{Type: "spike"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected type 'sine'")

	// SpikeShaper wrong type
	_, err = NewSpikeShaper(ShaperConfig{Type: "sine"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected type 'spike'")

	// StepShaper wrong type
	_, err = NewStepShaper(ShaperConfig{Type: "sine"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected type 'step'")

	// CustomShaper wrong type
	_, err = NewCustomShaper(ShaperConfig{Type: "sine"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected type 'custom'")
}

func TestClampQPS(t *testing.T) {
	// Test the clampQPS helper function behavior through shapers

	// Test MinQPS clamping
	config := ShaperConfig{
		Type:    "step",
		BaseQPS: 0,
		MinQPS:  50,
		MaxQPS:  150,
		Step: &StepConfig{
			Steps: []StepLevel{
				{QPS: 10, Duration: 10 * time.Second},  // below min
				{QPS: 100, Duration: 10 * time.Second}, // within range
				{QPS: 200, Duration: 10 * time.Second}, // above max
			},
		},
	}

	shaper, err := NewStepShaper(config)
	require.NoError(t, err)

	// Step 1: 10 -> clamped to 50
	assert.InDelta(t, 50.0, shaper.GetTargetQPS(5*time.Second), 0.001)

	// Step 2: 100 -> unchanged
	assert.InDelta(t, 100.0, shaper.GetTargetQPS(15*time.Second), 0.001)

	// Step 3: 200 -> clamped to 150
	assert.InDelta(t, 150.0, shaper.GetTargetQPS(25*time.Second), 0.001)
}

func TestCustomShaper_HoldSteady(t *testing.T) {
	// Test GetPhase when two consecutive points have same QPS
	config := ShaperConfig{
		Type: "custom",
		CustomPoints: []CustomPoint{
			{Time: 0, QPS: 100},
			{Time: 30 * time.Second, QPS: 100}, // same as first
		},
	}

	shaper, err := NewCustomShaper(config)
	require.NoError(t, err)

	phase := shaper.GetPhase(15 * time.Second)
	assert.Contains(t, phase, "holding steady")
}

func TestStepShaper_NoRamp(t *testing.T) {
	// Test steps without ramp duration (instant transitions)
	config := ShaperConfig{
		Type:    "step",
		BaseQPS: 50,
		Step: &StepConfig{
			Steps: []StepLevel{
				{QPS: 100, Duration: 30 * time.Second, RampDuration: 0},
				{QPS: 200, Duration: 30 * time.Second, RampDuration: 0},
			},
			Loop: false,
		},
	}

	shaper, err := NewStepShaper(config)
	require.NoError(t, err)

	// Instant jump to 100 at t=0
	assert.InDelta(t, 100.0, shaper.GetTargetQPS(0), 0.001)
	assert.InDelta(t, 100.0, shaper.GetTargetQPS(1*time.Second), 0.001)

	// Instant jump to 200 at t=30s
	assert.InDelta(t, 200.0, shaper.GetTargetQPS(30*time.Second), 0.001)
}

func TestStepShaper_LoopingCycle(t *testing.T) {
	config := ShaperConfig{
		Type:    "step",
		BaseQPS: 50,
		Step: &StepConfig{
			Steps: []StepLevel{
				{QPS: 100, Duration: 20 * time.Second},
				{QPS: 200, Duration: 20 * time.Second},
			},
			Loop: true,
		},
	}

	shaper, err := NewStepShaper(config)
	require.NoError(t, err)

	// Verify phase shows correct cycle number
	phase1 := shaper.GetPhase(10 * time.Second)
	assert.Contains(t, phase1, "cycle 1")

	phase2 := shaper.GetPhase(50 * time.Second) // 50 = 40 + 10, cycle 2
	assert.Contains(t, phase2, "cycle 2")

	phase3 := shaper.GetPhase(90 * time.Second) // 90 = 80 + 10, cycle 3
	assert.Contains(t, phase3, "cycle 3")
}

// ============================================================================
// Benchmarks
// ============================================================================

func BenchmarkSineWaveShaper_GetTargetQPS(b *testing.B) {
	config := ShaperConfig{
		Type:      "sine",
		BaseQPS:   100,
		Amplitude: 50,
		Period:    60 * time.Second,
	}
	shaper, _ := NewSineWaveShaper(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		elapsed := time.Duration(i%60000) * time.Millisecond
		shaper.GetTargetQPS(elapsed)
	}
}

func BenchmarkSpikeShaper_GetTargetQPS(b *testing.B) {
	config := ShaperConfig{
		Type:    "spike",
		BaseQPS: 100,
		Spike: &SpikeConfig{
			SpikeQPS:      500,
			SpikeDuration: 5 * time.Second,
			SpikeInterval: 30 * time.Second,
		},
	}
	shaper, _ := NewSpikeShaper(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		elapsed := time.Duration(i%60000) * time.Millisecond
		shaper.GetTargetQPS(elapsed)
	}
}

func BenchmarkStepShaper_GetTargetQPS(b *testing.B) {
	config := ShaperConfig{
		Type:    "step",
		BaseQPS: 50,
		Step: &StepConfig{
			Steps: []StepLevel{
				{QPS: 100, Duration: 30 * time.Second},
				{QPS: 200, Duration: 30 * time.Second},
			},
		},
	}
	shaper, _ := NewStepShaper(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		elapsed := time.Duration(i%60000) * time.Millisecond
		shaper.GetTargetQPS(elapsed)
	}
}

func BenchmarkCustomShaper_GetTargetQPS(b *testing.B) {
	config := ShaperConfig{
		Type: "custom",
		CustomPoints: []CustomPoint{
			{Time: 0, QPS: 50},
			{Time: 30 * time.Second, QPS: 150},
			{Time: 60 * time.Second, QPS: 100},
		},
	}
	shaper, _ := NewCustomShaper(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		elapsed := time.Duration(i%60000) * time.Millisecond
		shaper.GetTargetQPS(elapsed)
	}
}
