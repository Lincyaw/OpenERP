// Package loadctrl provides load control components including traffic shaping.
// This file contains validation tests for traffic shaper patterns.
package loadctrl

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// LOADGEN-VAL-004: Traffic Shaper Pattern Validation Tests
// ============================================================================
//
// This file validates the traffic shaper implementations against the specified
// acceptance criteria:
//
// 1. SineWaveShaper: 60s period, ±50% amplitude - QPS follows sine wave math
// 2. SpikeShaper: 10x traffic spikes at specified intervals
// 3. StepShaper: 20% QPS increase every 30 seconds (staircase pattern)
// 4. CustomShaper: Execute according to custom configuration file
//
// Pass Criteria:
// - Sine wave mode QPS changes match mathematical sine function
// - Spike mode produces 10x traffic at specified times
// - Step mode increases 20% QPS every 30 seconds
// - Custom mode executes according to configuration points
// ============================================================================

// ============================================================================
// SineWaveShaper Validation Tests
// ============================================================================

// TestSineWaveShaper_60sPeriod50PercentAmplitude validates the primary acceptance criteria:
// 60 second period with ±50% amplitude
func TestSineWaveShaper_60sPeriod50PercentAmplitude(t *testing.T) {
	// Configuration: 60s period, ±50% amplitude (relative)
	config := ShaperConfig{
		Type:      "sine",
		BaseQPS:   100,
		Amplitude: 0.5, // 50% relative amplitude = ±50 QPS
		Period:    60 * time.Second,
	}

	shaper, err := NewSineWaveShaper(config)
	require.NoError(t, err)

	t.Run("effective_amplitude_is_50_percent", func(t *testing.T) {
		// Effective amplitude = 100 * 0.5 = 50
		assert.InDelta(t, 50.0, shaper.GetAmplitude(), 0.001,
			"Amplitude should be 50%% of baseQPS (50)")
	})

	t.Run("min_max_qps_range", func(t *testing.T) {
		min, max := shaper.GetMinMaxQPS()
		assert.InDelta(t, 50.0, min, 0.001, "Min QPS should be 50 (base - amplitude)")
		assert.InDelta(t, 150.0, max, 0.001, "Max QPS should be 150 (base + amplitude)")
	})

	t.Run("qps_follows_sine_wave_math", func(t *testing.T) {
		// Sample throughout one complete 60-second period
		sampleCount := 120 // 2 samples per second
		maxError := 0.0

		for i := 0; i <= sampleCount; i++ {
			elapsed := time.Duration(float64(i) / float64(sampleCount) * float64(60*time.Second))
			actualQPS := shaper.GetTargetQPS(elapsed)

			// Expected: baseQPS + amplitude * sin(2π * t / period)
			// = 100 + 50 * sin(2π * t / 60s)
			expectedQPS := 100.0 + 50.0*math.Sin(2*math.Pi*float64(elapsed)/float64(60*time.Second))

			diff := math.Abs(actualQPS - expectedQPS)
			if diff > maxError {
				maxError = diff
			}

			assert.InDelta(t, expectedQPS, actualQPS, 0.001,
				"QPS mismatch at t=%v: expected %.4f, got %.4f", elapsed, expectedQPS, actualQPS)
		}

		t.Logf("Max QPS deviation from sine function: %.6f", maxError)
	})

	t.Run("smooth_qps_curve", func(t *testing.T) {
		// Verify the QPS curve is smooth (no sudden jumps)
		// Maximum expected change between 100ms samples should be bounded
		prevQPS := shaper.GetTargetQPS(0)
		const maxJump = 2.0 // Max QPS change per 100ms should be small for smooth curve

		for elapsed := 100 * time.Millisecond; elapsed <= 60*time.Second; elapsed += 100 * time.Millisecond {
			currentQPS := shaper.GetTargetQPS(elapsed)
			jump := math.Abs(currentQPS - prevQPS)

			assert.LessOrEqual(t, jump, maxJump,
				"QPS jump at t=%v exceeds smoothness threshold: %.4f QPS change", elapsed, jump)
			prevQPS = currentQPS
		}
	})
}

// TestSineWaveShaper_SmoothQPSVariation verifies QPS changes smoothly over time
func TestSineWaveShaper_SmoothQPSVariation(t *testing.T) {
	config := ShaperConfig{
		Type:      "sine",
		BaseQPS:   100,
		Amplitude: 0.5, // 50%
		Period:    60 * time.Second,
	}

	shaper, err := NewSineWaveShaper(config)
	require.NoError(t, err)

	// Record QPS at specific phase points
	testCases := []struct {
		elapsed  time.Duration
		expected float64
		phase    string
	}{
		{0 * time.Second, 100.0, "start (sin=0)"},   // sin(0) = 0
		{15 * time.Second, 150.0, "peak (sin=1)"},   // sin(π/2) = 1
		{30 * time.Second, 100.0, "mid (sin=0)"},    // sin(π) = 0
		{45 * time.Second, 50.0, "trough (sin=-1)"}, // sin(3π/2) = -1
		{60 * time.Second, 100.0, "end (sin=0)"},    // sin(2π) = 0
	}

	for _, tc := range testCases {
		t.Run(tc.phase, func(t *testing.T) {
			actual := shaper.GetTargetQPS(tc.elapsed)
			assert.InDelta(t, tc.expected, actual, 0.001,
				"At t=%v (%s): expected %.1f QPS, got %.4f", tc.elapsed, tc.phase, tc.expected, actual)
		})
	}

	// Verify phase descriptions are correct
	t.Run("phase_descriptions", func(t *testing.T) {
		phases := map[time.Duration]string{
			5 * time.Second:  "rising to peak",
			20 * time.Second: "falling from peak",
			35 * time.Second: "falling to trough",
			50 * time.Second: "rising from trough",
		}
		for elapsed, expectedPhase := range phases {
			phase := shaper.GetPhase(elapsed)
			assert.Contains(t, phase, expectedPhase,
				"Phase at t=%v should contain '%s', got '%s'", elapsed, expectedPhase, phase)
		}
	})
}

// TestSineWaveShaper_MultiPeriodAccuracy validates accuracy across multiple periods
func TestSineWaveShaper_MultiPeriodAccuracy(t *testing.T) {
	config := ShaperConfig{
		Type:      "sine",
		BaseQPS:   100,
		Amplitude: 0.5,
		Period:    60 * time.Second,
	}

	shaper, err := NewSineWaveShaper(config)
	require.NoError(t, err)

	// Test across 3 full periods (180 seconds = 3 minutes)
	// Verify periodicity is maintained
	period := 60 * time.Second
	for periodNum := 0; periodNum < 3; periodNum++ {
		offset := time.Duration(periodNum) * period

		// Check key points in each period
		assert.InDelta(t, 100.0, shaper.GetTargetQPS(offset+0*time.Second), 0.001,
			"Period %d start", periodNum+1)
		assert.InDelta(t, 150.0, shaper.GetTargetQPS(offset+15*time.Second), 0.001,
			"Period %d peak", periodNum+1)
		assert.InDelta(t, 100.0, shaper.GetTargetQPS(offset+30*time.Second), 0.001,
			"Period %d mid", periodNum+1)
		assert.InDelta(t, 50.0, shaper.GetTargetQPS(offset+45*time.Second), 0.001,
			"Period %d trough", periodNum+1)
	}
}

// ============================================================================
// SpikeShaper Validation Tests
// ============================================================================

// TestSpikeShaper_10xMultiplierAtIntervals validates 10x traffic spikes
func TestSpikeShaper_10xMultiplierAtIntervals(t *testing.T) {
	baseQPS := 100.0
	multiplier := 10.0
	spikeQPS := baseQPS * multiplier // 1000 QPS

	config := ShaperConfig{
		Type:    "spike",
		BaseQPS: baseQPS,
		Spike: &SpikeConfig{
			SpikeQPS:      spikeQPS,
			SpikeDuration: 5 * time.Second,  // 5 second spikes
			SpikeInterval: 30 * time.Second, // every 30 seconds
		},
	}

	shaper, err := NewSpikeShaper(config)
	require.NoError(t, err)

	t.Run("spike_produces_10x_traffic", func(t *testing.T) {
		// During spike period (0-5s)
		for elapsed := time.Duration(0); elapsed < 5*time.Second; elapsed += time.Second {
			qps := shaper.GetTargetQPS(elapsed)
			actualMultiplier := qps / baseQPS
			assert.InDelta(t, multiplier, actualMultiplier, 0.001,
				"At t=%v: expected 10x multiplier, got %.2fx", elapsed, actualMultiplier)
		}
	})

	t.Run("normal_period_returns_to_base", func(t *testing.T) {
		// After spike period (5-30s)
		normalTimes := []time.Duration{
			5 * time.Second,
			10 * time.Second,
			20 * time.Second,
			29 * time.Second,
		}
		for _, elapsed := range normalTimes {
			qps := shaper.GetTargetQPS(elapsed)
			assert.InDelta(t, baseQPS, qps, 0.001,
				"At t=%v (normal period): expected %.0f QPS, got %.4f", elapsed, baseQPS, qps)
		}
	})

	t.Run("spikes_occur_at_intervals", func(t *testing.T) {
		// Verify multiple spike occurrences over 90 seconds (3 spikes)
		spikeTimes := []time.Duration{
			0 * time.Second,  // First spike start
			30 * time.Second, // Second spike start
			60 * time.Second, // Third spike start
		}
		for i, start := range spikeTimes {
			t.Run(
				"spike_"+string(rune('1'+i)),
				func(t *testing.T) {
					// Check spike QPS during spike window
					for offset := time.Duration(0); offset < 5*time.Second; offset += time.Second {
						elapsed := start + offset
						qps := shaper.GetTargetQPS(elapsed)
						assert.InDelta(t, spikeQPS, qps, 0.001,
							"Spike %d at t=%v: expected %.0f QPS, got %.4f", i+1, elapsed, spikeQPS, qps)
					}
				})
		}
	})

	t.Run("spike_boundaries_are_exact", func(t *testing.T) {
		// Test precise boundary behavior
		// Just before spike ends (4.999s) - should still be in spike
		assert.InDelta(t, spikeQPS, shaper.GetTargetQPS(4999*time.Millisecond), 0.001,
			"Just before spike end should be spike QPS")

		// Exactly at spike end (5s) - should be back to base
		assert.InDelta(t, baseQPS, shaper.GetTargetQPS(5*time.Second), 0.001,
			"At spike end should be base QPS")
	})
}

// TestSpikeShaper_SpikeMetrics validates spike helper methods
func TestSpikeShaper_SpikeMetrics(t *testing.T) {
	config := ShaperConfig{
		Type:    "spike",
		BaseQPS: 100,
		Spike: &SpikeConfig{
			SpikeQPS:      1000, // 10x
			SpikeDuration: 5 * time.Second,
			SpikeInterval: 30 * time.Second,
		},
	}

	shaper, err := NewSpikeShaper(config)
	require.NoError(t, err)

	t.Run("spike_number_tracking", func(t *testing.T) {
		assert.Equal(t, 1, shaper.CurrentSpikeNumber(0), "Spike 1 at t=0")
		assert.Equal(t, 1, shaper.CurrentSpikeNumber(29*time.Second), "Still spike 1 at t=29s")
		assert.Equal(t, 2, shaper.CurrentSpikeNumber(30*time.Second), "Spike 2 at t=30s")
		assert.Equal(t, 3, shaper.CurrentSpikeNumber(60*time.Second), "Spike 3 at t=60s")
	})

	t.Run("remaining_spike_duration", func(t *testing.T) {
		// During spike
		assert.Equal(t, 5*time.Second, shaper.RemainingSpikeDuration(0))
		assert.Equal(t, 3*time.Second, shaper.RemainingSpikeDuration(2*time.Second))
		assert.Equal(t, 1*time.Second, shaper.RemainingSpikeDuration(4*time.Second))

		// After spike
		assert.Equal(t, time.Duration(0), shaper.RemainingSpikeDuration(5*time.Second))
		assert.Equal(t, time.Duration(0), shaper.RemainingSpikeDuration(20*time.Second))
	})
}

// ============================================================================
// StepShaper Validation Tests
// ============================================================================

// TestStepShaper_20PercentIncreaseEvery30Seconds validates the staircase pattern
func TestStepShaper_20PercentIncreaseEvery30Seconds(t *testing.T) {
	// Create staircase pattern: 100 -> 120 -> 144 -> 172.8 (20% increase each step)
	baseQPS := 100.0
	increasePercent := 0.20 // 20%

	// Calculate QPS levels for 4 steps (30s each = 2 minutes)
	steps := []StepLevel{
		{QPS: baseQPS, Duration: 30 * time.Second},                                  // 100
		{QPS: baseQPS * (1 + increasePercent), Duration: 30 * time.Second},          // 120
		{QPS: baseQPS * math.Pow(1+increasePercent, 2), Duration: 30 * time.Second}, // 144
		{QPS: baseQPS * math.Pow(1+increasePercent, 3), Duration: 30 * time.Second}, // 172.8
	}

	config := ShaperConfig{
		Type:    "step",
		BaseQPS: baseQPS,
		Step: &StepConfig{
			Steps: steps,
			Loop:  false,
		},
	}

	shaper, err := NewStepShaper(config)
	require.NoError(t, err)

	t.Run("step_qps_values", func(t *testing.T) {
		expected := []struct {
			time     time.Duration
			qps      float64
			stepName string
		}{
			{0 * time.Second, 100.0, "Step 1: base"},
			{15 * time.Second, 100.0, "Step 1: mid"},
			{30 * time.Second, 120.0, "Step 2: +20%"},
			{45 * time.Second, 120.0, "Step 2: mid"},
			{60 * time.Second, 144.0, "Step 3: +44%"},
			{75 * time.Second, 144.0, "Step 3: mid"},
			{90 * time.Second, 172.8, "Step 4: +72.8%"},
			{105 * time.Second, 172.8, "Step 4: mid"},
		}

		for _, tc := range expected {
			actual := shaper.GetTargetQPS(tc.time)
			assert.InDelta(t, tc.qps, actual, 0.001,
				"At t=%v (%s): expected %.1f QPS, got %.4f", tc.time, tc.stepName, tc.qps, actual)
		}
	})

	t.Run("step_transitions_are_immediate", func(t *testing.T) {
		// Verify no ramp (immediate transitions) without RampDuration

		// Just before step 2 (29.999s)
		assert.InDelta(t, 100.0, shaper.GetTargetQPS(29999*time.Millisecond), 0.001)
		// At step 2 (30s)
		assert.InDelta(t, 120.0, shaper.GetTargetQPS(30*time.Second), 0.001)

		// Just before step 3 (59.999s)
		assert.InDelta(t, 120.0, shaper.GetTargetQPS(59999*time.Millisecond), 0.001)
		// At step 3 (60s)
		assert.InDelta(t, 144.0, shaper.GetTargetQPS(60*time.Second), 0.001)
	})

	t.Run("increase_percentage_verification", func(t *testing.T) {
		// Verify each step increases by 20%
		prevQPS := shaper.GetTargetQPS(0)
		for stepIdx := 1; stepIdx < 4; stepIdx++ {
			elapsed := time.Duration(stepIdx*30) * time.Second
			currentQPS := shaper.GetTargetQPS(elapsed)

			actualIncrease := (currentQPS - prevQPS) / prevQPS
			assert.InDelta(t, increasePercent, actualIncrease, 0.001,
				"Step %d: expected 20%% increase, got %.1f%%", stepIdx+1, actualIncrease*100)
			prevQPS = currentQPS
		}
	})
}

// TestStepShaper_WithRamping validates smooth transitions between steps
func TestStepShaper_WithRamping(t *testing.T) {
	config := ShaperConfig{
		Type:    "step",
		BaseQPS: 50, // Ramp from 50 to first step
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

	t.Run("ramp_from_base_to_step1", func(t *testing.T) {
		// Ramp from baseQPS (50) to step1 QPS (100) over 10 seconds
		rampPoints := []struct {
			elapsed  time.Duration
			expected float64
		}{
			{0 * time.Second, 50.0},   // Start of ramp
			{5 * time.Second, 75.0},   // Halfway through ramp
			{10 * time.Second, 100.0}, // End of ramp
			{20 * time.Second, 100.0}, // Hold phase
		}

		for _, pt := range rampPoints {
			actual := shaper.GetTargetQPS(pt.elapsed)
			assert.InDelta(t, pt.expected, actual, 0.001,
				"At t=%v: expected %.0f QPS, got %.4f", pt.elapsed, pt.expected, actual)
		}
	})

	t.Run("ramp_between_steps", func(t *testing.T) {
		// Ramp from step1 QPS (100) to step2 QPS (200) over 10 seconds
		rampPoints := []struct {
			elapsed  time.Duration
			expected float64
		}{
			{30 * time.Second, 100.0}, // Start of step 2 ramp
			{35 * time.Second, 150.0}, // Halfway through ramp
			{40 * time.Second, 200.0}, // End of ramp
			{50 * time.Second, 200.0}, // Hold phase
		}

		for _, pt := range rampPoints {
			actual := shaper.GetTargetQPS(pt.elapsed)
			assert.InDelta(t, pt.expected, actual, 0.001,
				"At t=%v: expected %.0f QPS, got %.4f", pt.elapsed, pt.expected, actual)
		}
	})

	t.Run("ramp_is_linear", func(t *testing.T) {
		// Verify ramp is linear (constant rate of change)
		// Step 2 ramp: 100 -> 200 over 10 seconds = 10 QPS/second
		expectedRate := 10.0 // QPS per second

		for i := 0; i < 10; i++ {
			elapsed1 := 30*time.Second + time.Duration(i)*time.Second
			elapsed2 := elapsed1 + time.Second

			qps1 := shaper.GetTargetQPS(elapsed1)
			qps2 := shaper.GetTargetQPS(elapsed2)

			actualRate := qps2 - qps1
			assert.InDelta(t, expectedRate, actualRate, 0.01,
				"Ramp rate between t=%v and t=%v: expected %.0f/s, got %.2f/s",
				elapsed1, elapsed2, expectedRate, actualRate)
		}
	})
}

// TestStepShaper_LoopingBehavior validates looping step patterns
func TestStepShaper_LoopingBehavior(t *testing.T) {
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

	t.Run("looping_preserves_pattern", func(t *testing.T) {
		// Total duration is 60 seconds
		assert.Equal(t, 60*time.Second, shaper.GetTotalDuration())

		// Check multiple cycles
		for cycle := 0; cycle < 3; cycle++ {
			offset := time.Duration(cycle) * 60 * time.Second

			// Step 1 in this cycle
			assert.InDelta(t, 100.0, shaper.GetTargetQPS(offset+15*time.Second), 0.001,
				"Cycle %d, Step 1", cycle+1)

			// Step 2 in this cycle
			assert.InDelta(t, 200.0, shaper.GetTargetQPS(offset+45*time.Second), 0.001,
				"Cycle %d, Step 2", cycle+1)
		}
	})
}

// ============================================================================
// CustomShaper Validation Tests
// ============================================================================

// TestCustomShaper_ExecutesAccordingToConfig validates custom curve execution
func TestCustomShaper_ExecutesAccordingToConfig(t *testing.T) {
	// Define a custom traffic curve
	customPoints := []CustomPoint{
		{Time: 0, QPS: 50},                 // Start low
		{Time: 30 * time.Second, QPS: 200}, // Ramp up
		{Time: 60 * time.Second, QPS: 100}, // Partial ramp down
		{Time: 90 * time.Second, QPS: 150}, // Ramp up again
		{Time: 120 * time.Second, QPS: 50}, // End at start
	}

	config := ShaperConfig{
		Type:         "custom",
		CustomPoints: customPoints,
	}

	shaper, err := NewCustomShaper(config)
	require.NoError(t, err)

	t.Run("exact_point_values", func(t *testing.T) {
		// Verify QPS exactly at defined points
		for _, pt := range customPoints {
			actual := shaper.GetTargetQPS(pt.Time)
			assert.InDelta(t, pt.QPS, actual, 0.001,
				"At t=%v: expected %.0f QPS, got %.4f", pt.Time, pt.QPS, actual)
		}
	})

	t.Run("linear_interpolation_between_points", func(t *testing.T) {
		// Segment 1: 0 -> 30s (50 -> 200)
		// At t=15s: halfway, should be 125
		assert.InDelta(t, 125.0, shaper.GetTargetQPS(15*time.Second), 0.001)

		// Segment 2: 30 -> 60s (200 -> 100)
		// At t=45s: halfway, should be 150
		assert.InDelta(t, 150.0, shaper.GetTargetQPS(45*time.Second), 0.001)

		// Segment 3: 60 -> 90s (100 -> 150)
		// At t=75s: halfway, should be 125
		assert.InDelta(t, 125.0, shaper.GetTargetQPS(75*time.Second), 0.001)

		// Segment 4: 90 -> 120s (150 -> 50)
		// At t=105s: halfway, should be 100
		assert.InDelta(t, 100.0, shaper.GetTargetQPS(105*time.Second), 0.001)
	})

	t.Run("before_and_after_curve", func(t *testing.T) {
		// After last point: holds at last QPS
		assert.InDelta(t, 50.0, shaper.GetTargetQPS(150*time.Second), 0.001)
		assert.InDelta(t, 50.0, shaper.GetTargetQPS(300*time.Second), 0.001)
	})

	t.Run("segment_tracking", func(t *testing.T) {
		startIdx, endIdx := shaper.CurrentSegment(45 * time.Second)
		assert.Equal(t, 1, startIdx, "Segment 2 start index")
		assert.Equal(t, 2, endIdx, "Segment 2 end index")
	})
}

// TestCustomShaper_ComplexCurve validates a more complex custom curve
func TestCustomShaper_ComplexCurve(t *testing.T) {
	// Simulate a business hours traffic pattern
	customPoints := []CustomPoint{
		{Time: 0, QPS: 20},                  // Night (low traffic)
		{Time: 30 * time.Second, QPS: 50},   // Early morning
		{Time: 60 * time.Second, QPS: 150},  // Morning peak
		{Time: 90 * time.Second, QPS: 100},  // Mid-day
		{Time: 120 * time.Second, QPS: 180}, // Afternoon peak
		{Time: 150 * time.Second, QPS: 80},  // Evening decline
		{Time: 180 * time.Second, QPS: 20},  // Back to night
	}

	config := ShaperConfig{
		Type:         "custom",
		CustomPoints: customPoints,
	}

	shaper, err := NewCustomShaper(config)
	require.NoError(t, err)

	t.Run("min_max_qps", func(t *testing.T) {
		min, max := shaper.GetMinMaxQPS()
		assert.InDelta(t, 20.0, min, 0.001)
		assert.InDelta(t, 180.0, max, 0.001)
	})

	t.Run("total_duration", func(t *testing.T) {
		assert.Equal(t, 180*time.Second, shaper.GetTotalDuration())
	})

	t.Run("point_count", func(t *testing.T) {
		assert.Equal(t, 7, shaper.GetPointCount())
	})

	t.Run("interpolation_accuracy", func(t *testing.T) {
		// Sample every 5 seconds and verify interpolation
		for elapsed := time.Duration(0); elapsed <= 180*time.Second; elapsed += 5 * time.Second {
			qps := shaper.GetTargetQPS(elapsed)

			// QPS should always be positive and within bounds
			assert.GreaterOrEqual(t, qps, 0.0)
			assert.LessOrEqual(t, qps, 200.0,
				"QPS at t=%v should be within bounds", elapsed)
		}
	})
}

// ============================================================================
// Cross-Shaper Validation Tests
// ============================================================================

// TestAllShapers_ThreadSafety validates concurrent access is safe
func TestAllShapers_ThreadSafety(t *testing.T) {
	shapers := []TrafficShaper{}

	// Create one of each type
	sine, _ := NewSineWaveShaper(ShaperConfig{
		Type: "sine", BaseQPS: 100, Amplitude: 0.5, Period: 60 * time.Second,
	})
	shapers = append(shapers, sine)

	spike, _ := NewSpikeShaper(ShaperConfig{
		Type: "spike", BaseQPS: 100,
		Spike: &SpikeConfig{SpikeQPS: 500, SpikeDuration: 5 * time.Second, SpikeInterval: 30 * time.Second},
	})
	shapers = append(shapers, spike)

	step, _ := NewStepShaper(ShaperConfig{
		Type: "step", BaseQPS: 50,
		Step: &StepConfig{Steps: []StepLevel{{QPS: 100, Duration: 30 * time.Second}}},
	})
	shapers = append(shapers, step)

	custom, _ := NewCustomShaper(ShaperConfig{
		Type:         "custom",
		CustomPoints: []CustomPoint{{Time: 0, QPS: 50}, {Time: 60 * time.Second, QPS: 150}},
	})
	shapers = append(shapers, custom)

	for _, shaper := range shapers {
		t.Run(shaper.Name()+"_concurrent_access", func(t *testing.T) {
			const goroutines = 100
			const iterations = 1000
			done := make(chan bool, goroutines)

			for g := 0; g < goroutines; g++ {
				go func() {
					for i := 0; i < iterations; i++ {
						elapsed := time.Duration(i%60000) * time.Millisecond
						_ = shaper.GetTargetQPS(elapsed)
						_ = shaper.GetPhase(elapsed)
						_ = shaper.Name()
						_ = shaper.Config()
					}
					done <- true
				}()
			}

			// Wait for all goroutines
			for g := 0; g < goroutines; g++ {
				<-done
			}
		})
	}
}

// TestAllShapers_NonNegativeQPS validates all shapers produce non-negative QPS
func TestAllShapers_NonNegativeQPS(t *testing.T) {
	testCases := []struct {
		name   string
		config ShaperConfig
	}{
		{
			name: "sine_extreme_amplitude",
			config: ShaperConfig{
				Type: "sine", BaseQPS: 10, Amplitude: 50, Period: 10 * time.Second,
			},
		},
		{
			name: "spike_zero_base",
			config: ShaperConfig{
				Type: "spike", BaseQPS: 0,
				Spike: &SpikeConfig{SpikeQPS: 100, SpikeDuration: time.Second, SpikeInterval: 5 * time.Second},
			},
		},
		{
			name: "step_zero_qps",
			config: ShaperConfig{
				Type: "step", BaseQPS: 0,
				Step: &StepConfig{Steps: []StepLevel{{QPS: 0, Duration: 10 * time.Second}}},
			},
		},
		{
			name: "custom_zero_points",
			config: ShaperConfig{
				Type:         "custom",
				CustomPoints: []CustomPoint{{Time: 0, QPS: 0}, {Time: 30 * time.Second, QPS: 0}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			shaper, err := NewTrafficShaper(tc.config)
			require.NoError(t, err)

			for elapsed := time.Duration(0); elapsed <= 60*time.Second; elapsed += time.Second {
				qps := shaper.GetTargetQPS(elapsed)
				assert.GreaterOrEqual(t, qps, 0.0,
					"QPS at t=%v must be non-negative, got %.4f", elapsed, qps)
			}
		})
	}
}

// TestAllShapers_MinMaxClamping validates min/max QPS clamping works
func TestAllShapers_MinMaxClamping(t *testing.T) {
	testCases := []struct {
		name   string
		config ShaperConfig
	}{
		{
			name: "sine_clamped",
			config: ShaperConfig{
				Type: "sine", BaseQPS: 100, Amplitude: 80, MinQPS: 50, MaxQPS: 150, Period: 60 * time.Second,
			},
		},
		{
			name: "spike_clamped",
			config: ShaperConfig{
				Type: "spike", BaseQPS: 100, MinQPS: 80, MaxQPS: 300,
				Spike: &SpikeConfig{SpikeQPS: 500, SpikeDuration: 5 * time.Second, SpikeInterval: 30 * time.Second},
			},
		},
		{
			name: "step_clamped",
			config: ShaperConfig{
				Type: "step", BaseQPS: 50, MinQPS: 60, MaxQPS: 150,
				Step: &StepConfig{Steps: []StepLevel{
					{QPS: 30, Duration: 30 * time.Second},  // Below min
					{QPS: 200, Duration: 30 * time.Second}, // Above max
				}},
			},
		},
		{
			name: "custom_clamped",
			config: ShaperConfig{
				Type: "custom", MinQPS: 40, MaxQPS: 120,
				CustomPoints: []CustomPoint{
					{Time: 0, QPS: 10},                 // Below min
					{Time: 30 * time.Second, QPS: 200}, // Above max
					{Time: 60 * time.Second, QPS: 80},  // Within range
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			shaper, err := NewTrafficShaper(tc.config)
			require.NoError(t, err)

			for elapsed := time.Duration(0); elapsed <= 90*time.Second; elapsed += time.Second {
				qps := shaper.GetTargetQPS(elapsed)

				if tc.config.MinQPS > 0 {
					assert.GreaterOrEqual(t, qps, tc.config.MinQPS,
						"QPS at t=%v must be >= minQPS (%.0f), got %.4f", elapsed, tc.config.MinQPS, qps)
				}
				if tc.config.MaxQPS > 0 {
					assert.LessOrEqual(t, qps, tc.config.MaxQPS,
						"QPS at t=%v must be <= maxQPS (%.0f), got %.4f", elapsed, tc.config.MaxQPS, qps)
				}
			}
		})
	}
}

// ============================================================================
// Stability Tests (Run 3 consecutive times to verify no flakiness)
// ============================================================================

// TestShaperStability_SineWave runs sine wave validation 3 times
func TestShaperStability_SineWave(t *testing.T) {
	for run := 1; run <= 3; run++ {
		t.Run("run_"+string(rune('0'+run)), func(t *testing.T) {
			config := ShaperConfig{
				Type: "sine", BaseQPS: 100, Amplitude: 0.5, Period: 60 * time.Second,
			}
			shaper, err := NewSineWaveShaper(config)
			require.NoError(t, err)

			// Key points should be deterministic
			assert.InDelta(t, 100.0, shaper.GetTargetQPS(0), 0.001)
			assert.InDelta(t, 150.0, shaper.GetTargetQPS(15*time.Second), 0.001)
			assert.InDelta(t, 50.0, shaper.GetTargetQPS(45*time.Second), 0.001)
		})
	}
}

// TestShaperStability_Spike runs spike validation 3 times
func TestShaperStability_Spike(t *testing.T) {
	for run := 1; run <= 3; run++ {
		t.Run("run_"+string(rune('0'+run)), func(t *testing.T) {
			config := ShaperConfig{
				Type: "spike", BaseQPS: 100,
				Spike: &SpikeConfig{SpikeQPS: 1000, SpikeDuration: 5 * time.Second, SpikeInterval: 30 * time.Second},
			}
			shaper, err := NewSpikeShaper(config)
			require.NoError(t, err)

			// Deterministic behavior
			assert.InDelta(t, 1000.0, shaper.GetTargetQPS(2*time.Second), 0.001)
			assert.InDelta(t, 100.0, shaper.GetTargetQPS(10*time.Second), 0.001)
		})
	}
}

// TestShaperStability_Step runs step validation 3 times
func TestShaperStability_Step(t *testing.T) {
	for run := 1; run <= 3; run++ {
		t.Run("run_"+string(rune('0'+run)), func(t *testing.T) {
			config := ShaperConfig{
				Type: "step", BaseQPS: 50,
				Step: &StepConfig{Steps: []StepLevel{
					{QPS: 100, Duration: 30 * time.Second},
					{QPS: 200, Duration: 30 * time.Second},
				}},
			}
			shaper, err := NewStepShaper(config)
			require.NoError(t, err)

			// Deterministic behavior
			assert.InDelta(t, 100.0, shaper.GetTargetQPS(15*time.Second), 0.001)
			assert.InDelta(t, 200.0, shaper.GetTargetQPS(45*time.Second), 0.001)
		})
	}
}

// TestShaperStability_Custom runs custom validation 3 times
func TestShaperStability_Custom(t *testing.T) {
	for run := 1; run <= 3; run++ {
		t.Run("run_"+string(rune('0'+run)), func(t *testing.T) {
			config := ShaperConfig{
				Type: "custom",
				CustomPoints: []CustomPoint{
					{Time: 0, QPS: 50},
					{Time: 60 * time.Second, QPS: 150},
				},
			}
			shaper, err := NewCustomShaper(config)
			require.NoError(t, err)

			// Deterministic behavior
			assert.InDelta(t, 50.0, shaper.GetTargetQPS(0), 0.001)
			assert.InDelta(t, 100.0, shaper.GetTargetQPS(30*time.Second), 0.001)
			assert.InDelta(t, 150.0, shaper.GetTargetQPS(60*time.Second), 0.001)
		})
	}
}
