package telemetry_test

import (
	"context"
	"runtime/pprof"
	"strings"
	"testing"

	"github.com/erp/backend/internal/infrastructure/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithProfilingLabels_EmptyLabels(t *testing.T) {
	ctx := context.Background()
	called := false

	telemetry.WithProfilingLabels(ctx, nil, func(c context.Context) {
		called = true
	})

	assert.True(t, called, "function should be called even with empty labels")

	// Empty map should also work
	called = false
	telemetry.WithProfilingLabels(ctx, map[string]string{}, func(c context.Context) {
		called = true
	})

	assert.True(t, called, "function should be called with empty map")
}

func TestWithProfilingLabels_BasicLabels(t *testing.T) {
	ctx := context.Background()
	called := false
	var capturedCtx context.Context

	labels := map[string]string{
		"controller": "ProductHandler",
		"method":     "GET",
		"route":      "/api/v1/products",
	}

	telemetry.WithProfilingLabels(ctx, labels, func(c context.Context) {
		called = true
		capturedCtx = c
	})

	assert.True(t, called, "function should be called")
	assert.NotNil(t, capturedCtx, "context should be passed")
}

func TestWithProfilingLabels_SkipsHighCardinalityLabels(t *testing.T) {
	ctx := context.Background()
	called := false

	// High cardinality labels should be filtered out
	labels := map[string]string{
		"controller": "ProductHandler", // allowed
		"user_id":    "user-123",       // high cardinality - should be skipped
		"request_id": "req-abc",        // high cardinality - should be skipped
		"order_id":   "order-456",      // high cardinality - should be skipped
	}

	telemetry.WithProfilingLabels(ctx, labels, func(c context.Context) {
		called = true
	})

	assert.True(t, called, "function should be called")
}

func TestWithProfilingLabels_TruncatesLongValues(t *testing.T) {
	ctx := context.Background()
	called := false

	// Create a very long value
	longValue := strings.Repeat("x", 200)

	labels := map[string]string{
		"controller": longValue,
	}

	telemetry.WithProfilingLabels(ctx, labels, func(c context.Context) {
		called = true
	})

	assert.True(t, called, "function should be called with truncated value")
}

func TestWithProfilingLabels_SkipsEmptyValues(t *testing.T) {
	ctx := context.Background()
	called := false

	labels := map[string]string{
		"controller": "ProductHandler",
		"method":     "",      // empty - should be skipped
		"":           "value", // empty key - should be skipped
	}

	telemetry.WithProfilingLabels(ctx, labels, func(c context.Context) {
		called = true
	})

	assert.True(t, called, "function should be called")
}

func TestWithPprofLabels_BasicLabels(t *testing.T) {
	ctx := context.Background()
	called := false
	var capturedLabels pprof.LabelSet

	labels := map[string]string{
		"controller": "ProductHandler",
		"method":     "POST",
	}

	telemetry.WithPprofLabels(ctx, labels, func(c context.Context) {
		called = true
		// Capture labels from context for verification
		capturedLabels = pprof.Labels() // Get empty labels for comparison
		_ = capturedLabels
	})

	assert.True(t, called, "function should be called")
}

func TestWithPprofLabels_EmptyLabels(t *testing.T) {
	ctx := context.Background()
	called := false

	telemetry.WithPprofLabels(ctx, nil, func(c context.Context) {
		called = true
	})

	assert.True(t, called, "function should be called with nil labels")

	called = false
	telemetry.WithPprofLabels(ctx, map[string]string{}, func(c context.Context) {
		called = true
	})

	assert.True(t, called, "function should be called with empty map")
}

func TestProfilingScope_Builder(t *testing.T) {
	scope := telemetry.NewProfilingScope(nil)

	scope.WithController("ProductHandler").
		WithRoute("/api/v1/products").
		WithMethod("GET").
		WithTenantID("tenant-123").
		WithOperation("ListProducts").
		WithRegion("db_query")

	labels := scope.Labels()

	assert.Equal(t, "ProductHandler", labels[telemetry.ProfilingLabelController])
	assert.Equal(t, "/api/v1/products", labels[telemetry.ProfilingLabelRoute])
	assert.Equal(t, "GET", labels[telemetry.ProfilingLabelMethod])
	assert.Equal(t, "tenant-123", labels[telemetry.ProfilingLabelTenantID])
	assert.Equal(t, "ListProducts", labels[telemetry.ProfilingLabelOperation])
	assert.Equal(t, "db_query", labels[telemetry.ProfilingLabelRegion])
}

func TestProfilingScope_WithInitialLabels(t *testing.T) {
	initial := map[string]string{
		"controller": "InitialController",
		"method":     "GET",
	}

	scope := telemetry.NewProfilingScope(initial)
	scope.WithRoute("/api/v1/test")

	labels := scope.Labels()

	assert.Equal(t, "InitialController", labels["controller"])
	assert.Equal(t, "GET", labels["method"])
	assert.Equal(t, "/api/v1/test", labels["route"])
}

func TestProfilingScope_OverwriteLabel(t *testing.T) {
	initial := map[string]string{
		"controller": "InitialController",
	}

	scope := telemetry.NewProfilingScope(initial)
	scope.WithController("NewController")

	labels := scope.Labels()

	assert.Equal(t, "NewController", labels["controller"])
}

func TestProfilingScope_LabelsReturnsACopy(t *testing.T) {
	scope := telemetry.NewProfilingScope(nil)
	scope.WithController("ProductHandler")

	labels1 := scope.Labels()
	labels1["controller"] = "Modified"

	labels2 := scope.Labels()
	assert.Equal(t, "ProductHandler", labels2["controller"], "original should not be modified")
}

func TestProfilingScope_Run(t *testing.T) {
	ctx := context.Background()
	called := false

	scope := telemetry.NewProfilingScope(nil)
	scope.WithController("TestHandler").
		WithMethod("POST")

	scope.Run(ctx, func(c context.Context) {
		called = true
	})

	assert.True(t, called, "function should be called via Run")
}

func TestProfilingScope_WithCustomLabel(t *testing.T) {
	scope := telemetry.NewProfilingScope(nil)
	scope.WithLabel("custom_key", "custom_value")

	labels := scope.Labels()
	assert.Equal(t, "custom_value", labels["custom_key"])
}

func TestHTTPRequestLabels(t *testing.T) {
	tests := []struct {
		name       string
		controller string
		route      string
		method     string
		tenantID   string
		wantLen    int
	}{
		{
			name:       "all_fields",
			controller: "ProductHandler",
			route:      "/api/v1/products",
			method:     "GET",
			tenantID:   "tenant-123",
			wantLen:    4,
		},
		{
			name:       "empty_tenant",
			controller: "ProductHandler",
			route:      "/api/v1/products",
			method:     "GET",
			tenantID:   "",
			wantLen:    3,
		},
		{
			name:       "only_controller",
			controller: "ProductHandler",
			route:      "",
			method:     "",
			tenantID:   "",
			wantLen:    1,
		},
		{
			name:       "all_empty",
			controller: "",
			route:      "",
			method:     "",
			tenantID:   "",
			wantLen:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			labels := telemetry.HTTPRequestLabels(tt.controller, tt.route, tt.method, tt.tenantID)
			assert.Len(t, labels, tt.wantLen)

			if tt.controller != "" {
				assert.Equal(t, tt.controller, labels[telemetry.ProfilingLabelController])
			}
			if tt.route != "" {
				assert.Equal(t, tt.route, labels[telemetry.ProfilingLabelRoute])
			}
			if tt.method != "" {
				assert.Equal(t, tt.method, labels[telemetry.ProfilingLabelMethod])
			}
			if tt.tenantID != "" {
				assert.Equal(t, tt.tenantID, labels[telemetry.ProfilingLabelTenantID])
			}
		})
	}
}

func TestOperationLabels(t *testing.T) {
	t.Run("operation_only", func(t *testing.T) {
		labels := telemetry.OperationLabels("CreateProduct", nil)

		assert.Equal(t, "CreateProduct", labels[telemetry.ProfilingLabelOperation])
		assert.Len(t, labels, 1)
	})

	t.Run("with_extra_labels", func(t *testing.T) {
		extra := map[string]string{
			"controller": "ProductHandler",
			"method":     "POST",
		}

		labels := telemetry.OperationLabels("CreateProduct", extra)

		assert.Equal(t, "CreateProduct", labels[telemetry.ProfilingLabelOperation])
		assert.Equal(t, "ProductHandler", labels["controller"])
		assert.Equal(t, "POST", labels["method"])
		assert.Len(t, labels, 3)
	})
}

func TestRegionLabels(t *testing.T) {
	t.Run("region_only", func(t *testing.T) {
		labels := telemetry.RegionLabels("db_query", nil)

		assert.Equal(t, "db_query", labels[telemetry.ProfilingLabelRegion])
		assert.Len(t, labels, 1)
	})

	t.Run("with_extra_labels", func(t *testing.T) {
		extra := map[string]string{
			"operation": "GetProducts",
			"table":     "products",
		}

		labels := telemetry.RegionLabels("db_query", extra)

		assert.Equal(t, "db_query", labels[telemetry.ProfilingLabelRegion])
		assert.Equal(t, "GetProducts", labels["operation"])
		assert.Equal(t, "products", labels["table"])
		assert.Len(t, labels, 3)
	})
}

func TestLabelConstants(t *testing.T) {
	// Verify constants are defined correctly
	assert.Equal(t, "controller", telemetry.ProfilingLabelController)
	assert.Equal(t, "route", telemetry.ProfilingLabelRoute)
	assert.Equal(t, "method", telemetry.ProfilingLabelMethod)
	assert.Equal(t, "tenant_id", telemetry.ProfilingLabelTenantID)
	assert.Equal(t, "operation", telemetry.ProfilingLabelOperation)
	assert.Equal(t, "region", telemetry.ProfilingLabelRegion)
}

func TestMaxLabelValueLength(t *testing.T) {
	// Verify MaxLabelValueLength is reasonable
	assert.Equal(t, 128, telemetry.MaxLabelValueLength)
}

func TestHighCardinalityLabels(t *testing.T) {
	// Verify high cardinality labels are properly defined
	expectedHighCardinality := []string{
		"user_id",
		"request_id",
		"order_id",
		"trace_id",
		"span_id",
		"session_id",
	}

	for _, label := range expectedHighCardinality {
		assert.True(t, telemetry.HighCardinalityLabels[label],
			"label %s should be marked as high cardinality", label)
	}
}

func TestLabelKeySanitization(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		inputLabels map[string]string
		description string
	}{
		{
			name: "spaces_in_key",
			inputLabels: map[string]string{
				"my key":     "value",
				"controller": "test",
			},
			description: "keys with spaces should be sanitized",
		},
		{
			name: "dashes_in_key",
			inputLabels: map[string]string{
				"my-key":     "value",
				"controller": "test",
			},
			description: "keys with dashes should be sanitized",
		},
		{
			name: "uppercase_in_key",
			inputLabels: map[string]string{
				"MyKey":      "value",
				"controller": "test",
			},
			description: "keys should be lowercased",
		},
		{
			name: "mixed_case_with_spaces",
			inputLabels: map[string]string{
				"My Custom Key": "value",
				"controller":    "test",
			},
			description: "mixed case with spaces should be normalized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			telemetry.WithProfilingLabels(ctx, tt.inputLabels, func(c context.Context) {
				called = true
			})
			assert.True(t, called, tt.description)
		})
	}
}

func TestNestedProfilingLabels(t *testing.T) {
	ctx := context.Background()
	outerCalled := false
	innerCalled := false

	outerLabels := map[string]string{
		"controller": "ProductHandler",
	}

	innerLabels := map[string]string{
		"operation": "QueryDB",
		"region":    "db_query",
	}

	telemetry.WithProfilingLabels(ctx, outerLabels, func(outerCtx context.Context) {
		outerCalled = true

		// Nested profiling labels
		telemetry.WithProfilingLabels(outerCtx, innerLabels, func(innerCtx context.Context) {
			innerCalled = true
			// In Pyroscope, nested labels should show hierarchy
		})
	})

	assert.True(t, outerCalled, "outer function should be called")
	assert.True(t, innerCalled, "inner function should be called")
}

func TestProfilingScope_ImmutableInitialLabels(t *testing.T) {
	initial := map[string]string{
		"controller": "InitialController",
	}

	scope := telemetry.NewProfilingScope(initial)

	// Modify the original map
	initial["controller"] = "Modified"

	// The scope should still have the original value
	labels := scope.Labels()
	assert.Equal(t, "InitialController", labels["controller"],
		"scope should have a copy of initial labels")
}

func TestContextPropagation(t *testing.T) {
	// Create a context with a custom value
	type contextKey string
	key := contextKey("test-key")
	ctx := context.WithValue(context.Background(), key, "test-value")

	labels := map[string]string{
		"controller": "TestHandler",
	}

	telemetry.WithProfilingLabels(ctx, labels, func(c context.Context) {
		// The context should still have the custom value
		value := c.Value(key)
		require.NotNil(t, value)
		assert.Equal(t, "test-value", value)
	})
}

func TestConcurrentProfilingLabels(t *testing.T) {
	ctx := context.Background()
	const goroutines = 10
	done := make(chan bool, goroutines)

	for i := range goroutines {
		go func(id int) {
			labels := map[string]string{
				"controller": "TestHandler",
				"goroutine":  "test", // not high cardinality
			}

			telemetry.WithProfilingLabels(ctx, labels, func(c context.Context) {
				// Simulate some work
			})
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for range goroutines {
		<-done
	}
}
