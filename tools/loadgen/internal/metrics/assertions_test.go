// Package metrics provides metrics collection and reporting for the load generator.
package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAssertionValidator(t *testing.T) {
	t.Parallel()

	t.Run("creates validator with config", func(t *testing.T) {
		config := AssertionValidatorConfig{
			ExitOnFailure: ptrBool(true),
		}

		validator := NewAssertionValidator(config)

		assert.NotNil(t, validator)
		assert.Equal(t, config.ExitOnFailure, validator.Config().ExitOnFailure)
	})

	t.Run("creates validator with nil config values", func(t *testing.T) {
		config := AssertionValidatorConfig{}

		validator := NewAssertionValidator(config)

		assert.NotNil(t, validator)
		assert.Nil(t, validator.Config().ExitOnFailure)
	})
}

func TestDefaultAssertionValidatorConfig(t *testing.T) {
	t.Parallel()

	config := DefaultAssertionValidatorConfig()

	require.NotNil(t, config.ExitOnFailure)
	assert.True(t, *config.ExitOnFailure)
}

func TestAssertionValidator_ExitOnFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		exitOnFailure *bool
		expected      bool
	}{
		{
			name:          "nil defaults to true",
			exitOnFailure: nil,
			expected:      true,
		},
		{
			name:          "explicit true",
			exitOnFailure: ptrBool(true),
			expected:      true,
		},
		{
			name:          "explicit false",
			exitOnFailure: ptrBool(false),
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewAssertionValidator(AssertionValidatorConfig{
				ExitOnFailure: tt.exitOnFailure,
			})

			assert.Equal(t, tt.expected, validator.ExitOnFailure())
		})
	}
}

func TestAssertionValidator_HasAssertions(t *testing.T) {
	t.Parallel()

	t.Run("no assertions configured", func(t *testing.T) {
		validator := NewAssertionValidator(AssertionValidatorConfig{})

		assert.False(t, validator.HasAssertions())
	})

	t.Run("global maxErrorRate configured", func(t *testing.T) {
		validator := NewAssertionValidator(AssertionValidatorConfig{
			Global: &GlobalAssertions{
				MaxErrorRate: ptrFloat64(1.0),
			},
		})

		assert.True(t, validator.HasAssertions())
	})

	t.Run("global minSuccessRate configured", func(t *testing.T) {
		validator := NewAssertionValidator(AssertionValidatorConfig{
			Global: &GlobalAssertions{
				MinSuccessRate: ptrFloat64(99.0),
			},
		})

		assert.True(t, validator.HasAssertions())
	})

	t.Run("global maxP95Latency configured", func(t *testing.T) {
		validator := NewAssertionValidator(AssertionValidatorConfig{
			Global: &GlobalAssertions{
				MaxP95Latency: int64(100 * time.Millisecond),
			},
		})

		assert.True(t, validator.HasAssertions())
	})

	t.Run("endpoint assertions configured", func(t *testing.T) {
		validator := NewAssertionValidator(AssertionValidatorConfig{
			EndpointOverrides: map[string]EndpointAssertions{
				"create-product": {
					MaxErrorRate: ptrFloat64(0.5),
				},
			},
		})

		assert.True(t, validator.HasAssertions())
	})

	t.Run("disabled endpoint assertions not counted", func(t *testing.T) {
		validator := NewAssertionValidator(AssertionValidatorConfig{
			EndpointOverrides: map[string]EndpointAssertions{
				"create-product": {
					MaxErrorRate: ptrFloat64(0.5),
					Disabled:     true,
				},
			},
		})

		assert.False(t, validator.HasAssertions())
	})
}

func TestAssertionValidator_Validate_GlobalAssertions(t *testing.T) {
	t.Parallel()

	t.Run("maxErrorRate passes when error rate is below threshold", func(t *testing.T) {
		validator := NewAssertionValidator(AssertionValidatorConfig{
			Global: &GlobalAssertions{
				MaxErrorRate: ptrFloat64(1.0),
			},
		})

		snapshot := Snapshot{
			TotalRequests:   1000,
			SuccessRequests: 995, // 0.5% error rate
			SuccessRate:     99.5,
		}

		results := validator.Validate(snapshot)

		require.Equal(t, 1, results.TotalCount)
		assert.True(t, results.AllPassed)
		assert.Equal(t, "global.maxErrorRate", results.Results[0].Name)
		assert.True(t, results.Results[0].Passed)
	})

	t.Run("maxErrorRate fails when error rate exceeds threshold", func(t *testing.T) {
		validator := NewAssertionValidator(AssertionValidatorConfig{
			Global: &GlobalAssertions{
				MaxErrorRate: ptrFloat64(1.0),
			},
		})

		snapshot := Snapshot{
			TotalRequests:   1000,
			SuccessRequests: 980, // 2.0% error rate
			SuccessRate:     98.0,
		}

		results := validator.Validate(snapshot)

		require.Equal(t, 1, results.TotalCount)
		assert.False(t, results.AllPassed)
		assert.Equal(t, 1, results.FailedCount)
		assert.False(t, results.Results[0].Passed)
		assert.Contains(t, results.Results[0].Actual, "2.00%")
	})

	t.Run("minSuccessRate passes when success rate meets threshold", func(t *testing.T) {
		validator := NewAssertionValidator(AssertionValidatorConfig{
			Global: &GlobalAssertions{
				MinSuccessRate: ptrFloat64(99.0),
			},
		})

		snapshot := Snapshot{
			SuccessRate: 99.5,
		}

		results := validator.Validate(snapshot)

		assert.True(t, results.AllPassed)
		assert.True(t, results.Results[0].Passed)
	})

	t.Run("minSuccessRate fails when success rate is below threshold", func(t *testing.T) {
		validator := NewAssertionValidator(AssertionValidatorConfig{
			Global: &GlobalAssertions{
				MinSuccessRate: ptrFloat64(99.0),
			},
		})

		snapshot := Snapshot{
			SuccessRate: 98.5,
		}

		results := validator.Validate(snapshot)

		assert.False(t, results.AllPassed)
		assert.False(t, results.Results[0].Passed)
	})

	t.Run("maxP95Latency passes when latency is below threshold", func(t *testing.T) {
		validator := NewAssertionValidator(AssertionValidatorConfig{
			Global: &GlobalAssertions{
				MaxP95Latency: int64(100 * time.Millisecond),
			},
		})

		snapshot := Snapshot{
			P95Latency: 50 * time.Millisecond,
		}

		results := validator.Validate(snapshot)

		assert.True(t, results.AllPassed)
		assert.True(t, results.Results[0].Passed)
	})

	t.Run("maxP95Latency fails when latency exceeds threshold", func(t *testing.T) {
		validator := NewAssertionValidator(AssertionValidatorConfig{
			Global: &GlobalAssertions{
				MaxP95Latency: int64(100 * time.Millisecond),
			},
		})

		snapshot := Snapshot{
			P95Latency: 150 * time.Millisecond,
		}

		results := validator.Validate(snapshot)

		assert.False(t, results.AllPassed)
		assert.False(t, results.Results[0].Passed)
	})

	t.Run("minThroughput passes when QPS meets threshold", func(t *testing.T) {
		validator := NewAssertionValidator(AssertionValidatorConfig{
			Global: &GlobalAssertions{
				MinThroughput: ptrFloat64(100.0),
			},
		})

		snapshot := Snapshot{
			QPS: 150.0,
		}

		results := validator.Validate(snapshot)

		assert.True(t, results.AllPassed)
		assert.True(t, results.Results[0].Passed)
	})

	t.Run("minThroughput fails when QPS is below threshold", func(t *testing.T) {
		validator := NewAssertionValidator(AssertionValidatorConfig{
			Global: &GlobalAssertions{
				MinThroughput: ptrFloat64(100.0),
			},
		})

		snapshot := Snapshot{
			QPS: 80.0,
		}

		results := validator.Validate(snapshot)

		assert.False(t, results.AllPassed)
		assert.False(t, results.Results[0].Passed)
	})

	t.Run("multiple global assertions", func(t *testing.T) {
		validator := NewAssertionValidator(AssertionValidatorConfig{
			Global: &GlobalAssertions{
				MaxErrorRate:   ptrFloat64(1.0),
				MinSuccessRate: ptrFloat64(99.0),
				MaxP95Latency:  int64(100 * time.Millisecond),
			},
		})

		snapshot := Snapshot{
			SuccessRate: 99.5,
			P95Latency:  50 * time.Millisecond,
		}

		results := validator.Validate(snapshot)

		assert.Equal(t, 3, results.TotalCount)
		assert.True(t, results.AllPassed)
		assert.Equal(t, 3, results.PassedCount)
	})
}

func TestAssertionValidator_Validate_EndpointAssertions(t *testing.T) {
	t.Parallel()

	t.Run("endpoint maxErrorRate passes", func(t *testing.T) {
		validator := NewAssertionValidator(AssertionValidatorConfig{
			EndpointOverrides: map[string]EndpointAssertions{
				"create-product": {
					MaxErrorRate: ptrFloat64(2.0),
				},
			},
		})

		snapshot := Snapshot{
			EndpointStats: map[string]*EndpointSnapshot{
				"create-product": {
					Name:            "create-product",
					TotalRequests:   100,
					SuccessRequests: 99,
					SuccessRate:     99.0, // 1% error rate
				},
			},
		}

		results := validator.Validate(snapshot)

		assert.True(t, results.AllPassed)
		assert.Equal(t, "endpoint:create-product.maxErrorRate", results.Results[0].Name)
		assert.Equal(t, "create-product", results.Results[0].Endpoint)
	})

	t.Run("endpoint maxErrorRate fails", func(t *testing.T) {
		validator := NewAssertionValidator(AssertionValidatorConfig{
			EndpointOverrides: map[string]EndpointAssertions{
				"create-product": {
					MaxErrorRate: ptrFloat64(1.0),
				},
			},
		})

		snapshot := Snapshot{
			EndpointStats: map[string]*EndpointSnapshot{
				"create-product": {
					Name:            "create-product",
					TotalRequests:   100,
					SuccessRequests: 95,
					SuccessRate:     95.0, // 5% error rate
				},
			},
		}

		results := validator.Validate(snapshot)

		assert.False(t, results.AllPassed)
		assert.False(t, results.Results[0].Passed)
	})

	t.Run("endpoint maxP95Latency assertion", func(t *testing.T) {
		validator := NewAssertionValidator(AssertionValidatorConfig{
			EndpointOverrides: map[string]EndpointAssertions{
				"create-product": {
					MaxP95Latency: int64(50 * time.Millisecond),
				},
			},
		})

		snapshot := Snapshot{
			EndpointStats: map[string]*EndpointSnapshot{
				"create-product": {
					Name:       "create-product",
					P95Latency: 100 * time.Millisecond,
				},
			},
		}

		results := validator.Validate(snapshot)

		assert.False(t, results.AllPassed)
		assert.Contains(t, results.Results[0].Name, "maxP95Latency")
	})

	t.Run("disabled endpoint is skipped", func(t *testing.T) {
		validator := NewAssertionValidator(AssertionValidatorConfig{
			EndpointOverrides: map[string]EndpointAssertions{
				"create-product": {
					MaxErrorRate: ptrFloat64(0.1), // Would fail
					Disabled:     true,
				},
			},
		})

		snapshot := Snapshot{
			EndpointStats: map[string]*EndpointSnapshot{
				"create-product": {
					Name:        "create-product",
					SuccessRate: 50.0, // 50% error rate
				},
			},
		}

		results := validator.Validate(snapshot)

		assert.Equal(t, 0, results.TotalCount)
	})

	t.Run("endpoint not in snapshot is skipped", func(t *testing.T) {
		validator := NewAssertionValidator(AssertionValidatorConfig{
			EndpointOverrides: map[string]EndpointAssertions{
				"nonexistent-endpoint": {
					MaxErrorRate: ptrFloat64(1.0),
				},
			},
		})

		snapshot := Snapshot{
			EndpointStats: map[string]*EndpointSnapshot{
				"create-product": {
					Name:        "create-product",
					SuccessRate: 99.0,
				},
			},
		}

		results := validator.Validate(snapshot)

		assert.Equal(t, 0, results.TotalCount)
	})

	t.Run("multiple endpoints validated in order", func(t *testing.T) {
		validator := NewAssertionValidator(AssertionValidatorConfig{
			EndpointOverrides: map[string]EndpointAssertions{
				"b-endpoint": {
					MaxErrorRate: ptrFloat64(1.0),
				},
				"a-endpoint": {
					MaxErrorRate: ptrFloat64(1.0),
				},
			},
		})

		snapshot := Snapshot{
			EndpointStats: map[string]*EndpointSnapshot{
				"a-endpoint": {SuccessRate: 99.5},
				"b-endpoint": {SuccessRate: 99.5},
			},
		}

		results := validator.Validate(snapshot)

		assert.Equal(t, 2, results.TotalCount)
		// Endpoints are sorted alphabetically
		assert.Equal(t, "endpoint:a-endpoint.maxErrorRate", results.Results[0].Name)
		assert.Equal(t, "endpoint:b-endpoint.maxErrorRate", results.Results[1].Name)
	})
}

func TestAssertionValidator_Validate_Combined(t *testing.T) {
	t.Parallel()

	t.Run("global and endpoint assertions combined", func(t *testing.T) {
		validator := NewAssertionValidator(AssertionValidatorConfig{
			Global: &GlobalAssertions{
				MaxErrorRate: ptrFloat64(5.0),
			},
			EndpointOverrides: map[string]EndpointAssertions{
				"create-product": {
					MaxErrorRate: ptrFloat64(2.0),
				},
			},
		})

		snapshot := Snapshot{
			SuccessRate: 98.0, // 2% error rate (passes global)
			EndpointStats: map[string]*EndpointSnapshot{
				"create-product": {
					Name:            "create-product",
					TotalRequests:   100, // Need non-zero for error rate calc
					SuccessRequests: 95,
					SuccessRate:     95.0, // 5% error rate (fails endpoint)
				},
			},
		}

		results := validator.Validate(snapshot)

		assert.Equal(t, 2, results.TotalCount)
		// Global passes (2% error rate < 5% threshold)
		// Endpoint fails (5% error rate > 2% threshold)
		assert.Equal(t, 1, results.PassedCount, "global assertion should pass")
		assert.Equal(t, 1, results.FailedCount, "endpoint assertion should fail")
		assert.False(t, results.AllPassed, "not all assertions passed")
	})
}

func TestAssertionResults_Methods(t *testing.T) {
	t.Parallel()

	t.Run("FailedResults returns only failed", func(t *testing.T) {
		results := &AssertionResults{
			Results: []AssertionResult{
				{Name: "test1", Passed: true},
				{Name: "test2", Passed: false},
				{Name: "test3", Passed: false},
			},
			FailedCount: 2,
		}

		failed := results.FailedResults()

		assert.Len(t, failed, 2)
		assert.Equal(t, "test2", failed[0].Name)
		assert.Equal(t, "test3", failed[1].Name)
	})

	t.Run("PassedResults returns only passed", func(t *testing.T) {
		results := &AssertionResults{
			Results: []AssertionResult{
				{Name: "test1", Passed: true},
				{Name: "test2", Passed: false},
				{Name: "test3", Passed: true},
			},
			PassedCount: 2,
		}

		passed := results.PassedResults()

		assert.Len(t, passed, 2)
		assert.Equal(t, "test1", passed[0].Name)
		assert.Equal(t, "test3", passed[1].Name)
	})

	t.Run("Summary with no assertions", func(t *testing.T) {
		results := &AssertionResults{
			TotalCount: 0,
		}

		assert.Equal(t, "No assertions configured", results.Summary())
	})

	t.Run("Summary with all passed", func(t *testing.T) {
		results := &AssertionResults{
			TotalCount:  5,
			PassedCount: 5,
			FailedCount: 0,
		}

		summary := results.Summary()

		assert.Contains(t, summary, "5/5 passed")
		assert.NotContains(t, summary, "FAILED")
	})

	t.Run("Summary with failures", func(t *testing.T) {
		results := &AssertionResults{
			TotalCount:  5,
			PassedCount: 3,
			FailedCount: 2,
		}

		summary := results.Summary()

		assert.Contains(t, summary, "3/5 passed")
		assert.Contains(t, summary, "2 FAILED")
	})
}

func TestFormatResults(t *testing.T) {
	t.Parallel()

	t.Run("no assertions", func(t *testing.T) {
		results := &AssertionResults{
			TotalCount: 0,
		}

		output := FormatResults(results, false)

		assert.Equal(t, "No assertions configured", output)
	})

	t.Run("all passed non-verbose", func(t *testing.T) {
		results := &AssertionResults{
			Results: []AssertionResult{
				{Name: "test1", Passed: true, Expected: ">= 99%", Actual: "99.5%"},
			},
			TotalCount:  1,
			PassedCount: 1,
			AllPassed:   true,
		}

		output := FormatResults(results, false)

		assert.Contains(t, output, "ASSERTION RESULTS")
		assert.Contains(t, output, "All 1 assertions PASSED")
		assert.NotContains(t, output, "PASSED ASSERTIONS:")
	})

	t.Run("all passed verbose", func(t *testing.T) {
		results := &AssertionResults{
			Results: []AssertionResult{
				{Name: "test1", Passed: true, Expected: ">= 99%", Actual: "99.5%"},
			},
			TotalCount:  1,
			PassedCount: 1,
			AllPassed:   true,
		}

		output := FormatResults(results, true)

		assert.Contains(t, output, "PASSED ASSERTIONS:")
		assert.Contains(t, output, "test1")
	})

	t.Run("with failures", func(t *testing.T) {
		results := &AssertionResults{
			Results: []AssertionResult{
				{Name: "test1", Passed: true, Expected: ">= 99%", Actual: "99.5%"},
				{Name: "test2", Passed: false, Expected: "<= 1%", Actual: "5%", Description: "Maximum error rate"},
			},
			TotalCount:  2,
			PassedCount: 1,
			FailedCount: 1,
			AllPassed:   false,
		}

		output := FormatResults(results, false)

		assert.Contains(t, output, "1/2 assertions FAILED")
		assert.Contains(t, output, "FAILED ASSERTIONS:")
		assert.Contains(t, output, "test2")
		assert.Contains(t, output, "Maximum error rate")
	})
}

func TestFormatDurationNs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ns       int64
		contains string
	}{
		{
			name:     "nanoseconds",
			ns:       500,
			contains: "ns",
		},
		{
			name:     "microseconds",
			ns:       5000,
			contains: "µs",
		},
		{
			name:     "milliseconds",
			ns:       5000000,
			contains: "ms",
		},
		{
			name:     "seconds",
			ns:       5000000000,
			contains: "s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDurationNs(tt.ns)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestAssertionValidator_AllLatencyAssertions(t *testing.T) {
	t.Parallel()

	t.Run("all latency assertions pass", func(t *testing.T) {
		validator := NewAssertionValidator(AssertionValidatorConfig{
			Global: &GlobalAssertions{
				MaxP50Latency: int64(100 * time.Millisecond),
				MaxP95Latency: int64(200 * time.Millisecond),
				MaxP99Latency: int64(300 * time.Millisecond),
				MaxAvgLatency: int64(50 * time.Millisecond),
			},
		})

		snapshot := Snapshot{
			P50Latency: 50 * time.Millisecond,
			P95Latency: 100 * time.Millisecond,
			P99Latency: 150 * time.Millisecond,
			AvgLatency: 30 * time.Millisecond,
		}

		results := validator.Validate(snapshot)

		assert.Equal(t, 4, results.TotalCount)
		assert.True(t, results.AllPassed)
	})

	t.Run("endpoint latency assertions", func(t *testing.T) {
		validator := NewAssertionValidator(AssertionValidatorConfig{
			EndpointOverrides: map[string]EndpointAssertions{
				"test-endpoint": {
					MaxP50Latency: int64(100 * time.Millisecond),
					MaxP95Latency: int64(200 * time.Millisecond),
					MaxP99Latency: int64(300 * time.Millisecond),
					MaxAvgLatency: int64(50 * time.Millisecond),
				},
			},
		})

		snapshot := Snapshot{
			EndpointStats: map[string]*EndpointSnapshot{
				"test-endpoint": {
					Name:       "test-endpoint",
					P50Latency: 50 * time.Millisecond,
					P95Latency: 100 * time.Millisecond,
					P99Latency: 150 * time.Millisecond,
					AvgLatency: 30 * time.Millisecond,
				},
			},
		}

		results := validator.Validate(snapshot)

		assert.Equal(t, 4, results.TotalCount)
		assert.True(t, results.AllPassed)
	})
}

func TestAssertionValidator_MinThroughputEndpoint(t *testing.T) {
	t.Parallel()

	validator := NewAssertionValidator(AssertionValidatorConfig{
		EndpointOverrides: map[string]EndpointAssertions{
			"high-traffic-endpoint": {
				MinThroughput: ptrFloat64(100.0),
			},
		},
	})

	t.Run("passes when QPS meets threshold", func(t *testing.T) {
		snapshot := Snapshot{
			EndpointStats: map[string]*EndpointSnapshot{
				"high-traffic-endpoint": {
					QPS: 150.0,
				},
			},
		}

		results := validator.Validate(snapshot)

		assert.True(t, results.AllPassed)
	})

	t.Run("fails when QPS is below threshold", func(t *testing.T) {
		snapshot := Snapshot{
			EndpointStats: map[string]*EndpointSnapshot{
				"high-traffic-endpoint": {
					QPS: 50.0,
				},
			},
		}

		results := validator.Validate(snapshot)

		assert.False(t, results.AllPassed)
	})
}

func TestAssertionValidator_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("zero requests handled correctly", func(t *testing.T) {
		validator := NewAssertionValidator(AssertionValidatorConfig{
			Global: &GlobalAssertions{
				MaxErrorRate: ptrFloat64(1.0),
			},
		})

		snapshot := Snapshot{
			TotalRequests: 0,
			SuccessRate:   0,
		}

		results := validator.Validate(snapshot)

		// 100 - 0 = 100% error rate, should fail
		assert.False(t, results.AllPassed)
	})

	t.Run("nil global assertions handled", func(t *testing.T) {
		validator := NewAssertionValidator(AssertionValidatorConfig{
			Global: nil,
		})

		snapshot := Snapshot{
			SuccessRate: 99.0,
		}

		results := validator.Validate(snapshot)

		assert.Equal(t, 0, results.TotalCount)
	})

	t.Run("empty endpoint overrides handled", func(t *testing.T) {
		validator := NewAssertionValidator(AssertionValidatorConfig{
			EndpointOverrides: map[string]EndpointAssertions{},
		})

		snapshot := Snapshot{
			SuccessRate: 99.0,
		}

		results := validator.Validate(snapshot)

		assert.Equal(t, 0, results.TotalCount)
	})
}

// Helper functions for creating pointers
func ptrBool(v bool) *bool {
	return &v
}

func ptrFloat64(v float64) *float64 {
	return &v
}

func TestNewAssertionValidatorConfigFromDurations(t *testing.T) {
	t.Parallel()

	t.Run("converts global assertions with durations", func(t *testing.T) {
		global := &GlobalAssertionsDuration{
			MaxErrorRate:   ptrFloat64(1.0),
			MinSuccessRate: ptrFloat64(99.0),
			MaxP50Latency:  10 * time.Millisecond,
			MaxP95Latency:  50 * time.Millisecond,
			MaxP99Latency:  100 * time.Millisecond,
			MaxAvgLatency:  20 * time.Millisecond,
			MinThroughput:  ptrFloat64(100.0),
		}

		config := NewAssertionValidatorConfigFromDurations(global, nil, ptrBool(true))

		require.NotNil(t, config.Global)
		assert.Equal(t, 1.0, *config.Global.MaxErrorRate)
		assert.Equal(t, 99.0, *config.Global.MinSuccessRate)
		assert.Equal(t, int64(10*time.Millisecond), config.Global.MaxP50Latency)
		assert.Equal(t, int64(50*time.Millisecond), config.Global.MaxP95Latency)
		assert.Equal(t, int64(100*time.Millisecond), config.Global.MaxP99Latency)
		assert.Equal(t, int64(20*time.Millisecond), config.Global.MaxAvgLatency)
		assert.Equal(t, 100.0, *config.Global.MinThroughput)
		require.NotNil(t, config.ExitOnFailure)
		assert.True(t, *config.ExitOnFailure)
	})

	t.Run("converts endpoint assertions with durations", func(t *testing.T) {
		endpoints := map[string]EndpointAssertionsDuration{
			"create-product": {
				MaxErrorRate:  ptrFloat64(0.5),
				MaxP95Latency: 200 * time.Millisecond,
				Disabled:      false,
			},
			"get-products": {
				MinSuccessRate: ptrFloat64(99.9),
				Disabled:       true,
			},
		}

		config := NewAssertionValidatorConfigFromDurations(nil, endpoints, nil)

		assert.Nil(t, config.Global)
		require.Len(t, config.EndpointOverrides, 2)

		createProduct := config.EndpointOverrides["create-product"]
		require.NotNil(t, createProduct.MaxErrorRate)
		assert.Equal(t, 0.5, *createProduct.MaxErrorRate)
		assert.Equal(t, int64(200*time.Millisecond), createProduct.MaxP95Latency)
		assert.False(t, createProduct.Disabled)

		getProducts := config.EndpointOverrides["get-products"]
		require.NotNil(t, getProducts.MinSuccessRate)
		assert.Equal(t, 99.9, *getProducts.MinSuccessRate)
		assert.True(t, getProducts.Disabled)
	})

	t.Run("handles nil global and empty endpoints", func(t *testing.T) {
		config := NewAssertionValidatorConfigFromDurations(nil, nil, nil)

		assert.Nil(t, config.Global)
		assert.Nil(t, config.EndpointOverrides)
		assert.Nil(t, config.ExitOnFailure)
	})
}
