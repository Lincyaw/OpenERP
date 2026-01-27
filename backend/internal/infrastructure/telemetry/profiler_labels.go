// Package telemetry provides Pyroscope continuous profiling integration.
package telemetry

import (
	"context"
	"maps"
	"runtime/pprof"
	"sort"
	"strings"

	"github.com/grafana/pyroscope-go"
)

// Constants for profiling labels.
const (
	// ProfilingLabelController is the label key for the handler/controller name.
	ProfilingLabelController = "controller"
	// ProfilingLabelRoute is the label key for the route pattern.
	ProfilingLabelRoute = "route"
	// ProfilingLabelMethod is the label key for the HTTP method.
	ProfilingLabelMethod = "method"
	// ProfilingLabelTenantID is the label key for the tenant ID.
	ProfilingLabelTenantID = "tenant_id"
	// ProfilingLabelOperation is the label key for the operation name.
	ProfilingLabelOperation = "operation"
	// ProfilingLabelRegion is the label key for code regions (e.g., "db_query", "external_api").
	ProfilingLabelRegion = "region"
)

// MaxLabelValueLength is the maximum allowed length for label values
// to prevent high cardinality and memory issues.
const MaxLabelValueLength = 128

// HighCardinalityLabels contains label keys that should be validated
// to prevent accidentally using high-cardinality values.
//
// WARNING: Do not modify this map at runtime. It is used by sanitizeLabels
// to filter out labels that could cause memory issues in Pyroscope.
//
// Note: tenant_id is intentionally NOT in this list, as it is typically
// low-to-medium cardinality. For systems with >1000 tenants, consider
// disabling tenant labeling or implementing sampling.
var HighCardinalityLabels = map[string]bool{
	"user_id":    true,
	"request_id": true,
	"order_id":   true,
	"trace_id":   true,
	"span_id":    true,
	"session_id": true,
}

// WithProfilingLabels wraps a function with profiling labels for Pyroscope.
// Labels allow slicing and filtering profiling data in the Pyroscope UI.
//
// This function uses pyroscope.TagWrapper which is compatible with Go's
// native pprof labels API.
//
// Example usage:
//
//	telemetry.WithProfilingLabels(ctx, map[string]string{
//	    "controller": "ProductHandler",
//	    "operation": "CreateProduct",
//	}, func(c context.Context) {
//	    // expensive operation
//	    processProducts(c)
//	})
//
// Note: Avoid using high-cardinality labels like user_id, request_id, or order_id
// as they can significantly increase memory usage in Pyroscope.
// The labels map is copied internally, so it is safe to modify the original
// map after calling this function.
func WithProfilingLabels(ctx context.Context, labels map[string]string, fn func(context.Context)) {
	if len(labels) == 0 {
		fn(ctx)
		return
	}

	// Make a defensive copy to prevent race conditions if the caller
	// modifies the map after passing it
	labelsCopy := make(map[string]string, len(labels))
	maps.Copy(labelsCopy, labels)

	// Sanitize and convert labels to slice format
	labelPairs := sanitizeLabels(labelsCopy)
	if len(labelPairs) == 0 {
		fn(ctx)
		return
	}

	pyroscope.TagWrapper(ctx, pyroscope.Labels(labelPairs...), fn)
}

// WithPprofLabels is an alternative implementation using Go's native pprof API.
// This is useful when you don't have the Pyroscope SDK available but still
// want labels in standard pprof output.
//
// Both pyroscope.TagWrapper and pprof.Do are compatible and produce the same
// label behavior. Use this when you want to ensure compatibility with standard
// Go profiling tools.
//
// The labels map is copied internally, so it is safe to modify the original
// map after calling this function.
func WithPprofLabels(ctx context.Context, labels map[string]string, fn func(context.Context)) {
	if len(labels) == 0 {
		fn(ctx)
		return
	}

	// Make a defensive copy to prevent race conditions if the caller
	// modifies the map after passing it
	labelsCopy := make(map[string]string, len(labels))
	maps.Copy(labelsCopy, labels)

	// Sanitize and convert labels to slice format
	labelPairs := sanitizeLabels(labelsCopy)
	if len(labelPairs) == 0 {
		fn(ctx)
		return
	}

	// Convert to pprof.Labels format
	pprofLabels := pprof.Labels(labelPairs...)
	pprof.Do(ctx, pprofLabels, fn)
}

// ProfilingScope provides a builder pattern for adding profiling labels.
// This is useful when you want to incrementally add labels.
type ProfilingScope struct {
	labels map[string]string
}

// NewProfilingScope creates a new ProfilingScope with an initial set of labels.
func NewProfilingScope(labels map[string]string) *ProfilingScope {
	scope := &ProfilingScope{
		labels: make(map[string]string),
	}
	maps.Copy(scope.labels, labels)
	return scope
}

// WithLabel adds a single label to the scope.
func (s *ProfilingScope) WithLabel(key, value string) *ProfilingScope {
	s.labels[key] = value
	return s
}

// WithController adds the controller label.
func (s *ProfilingScope) WithController(controller string) *ProfilingScope {
	return s.WithLabel(ProfilingLabelController, controller)
}

// WithRoute adds the route label.
func (s *ProfilingScope) WithRoute(route string) *ProfilingScope {
	return s.WithLabel(ProfilingLabelRoute, route)
}

// WithMethod adds the method label.
func (s *ProfilingScope) WithMethod(method string) *ProfilingScope {
	return s.WithLabel(ProfilingLabelMethod, method)
}

// WithTenantID adds the tenant_id label.
func (s *ProfilingScope) WithTenantID(tenantID string) *ProfilingScope {
	return s.WithLabel(ProfilingLabelTenantID, tenantID)
}

// WithOperation adds the operation label.
func (s *ProfilingScope) WithOperation(operation string) *ProfilingScope {
	return s.WithLabel(ProfilingLabelOperation, operation)
}

// WithRegion adds the region label for code regions.
func (s *ProfilingScope) WithRegion(region string) *ProfilingScope {
	return s.WithLabel(ProfilingLabelRegion, region)
}

// Labels returns the current labels map.
func (s *ProfilingScope) Labels() map[string]string {
	result := make(map[string]string, len(s.labels))
	maps.Copy(result, s.labels)
	return result
}

// Run executes the function with the accumulated labels.
func (s *ProfilingScope) Run(ctx context.Context, fn func(context.Context)) {
	WithProfilingLabels(ctx, s.labels, fn)
}

// sanitizeLabels validates and sanitizes labels for Pyroscope.
// - Filters out high-cardinality labels with a warning
// - Truncates values that are too long
// - Removes empty keys/values
// - Returns a deterministic slice of key-value pairs
func sanitizeLabels(labels map[string]string) []string {
	if len(labels) == 0 {
		return nil
	}

	// Pre-allocate capacity for worst case
	pairs := make([]string, 0, len(labels)*2)

	// Sort keys for deterministic output
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := labels[key]

		// Skip empty keys or values
		if key == "" || value == "" {
			continue
		}

		// Skip high-cardinality labels
		if HighCardinalityLabels[key] {
			// In production, we silently skip these rather than logging
			// to avoid log spam in hot paths
			continue
		}

		// Truncate long values
		if len(value) > MaxLabelValueLength {
			value = value[:MaxLabelValueLength]
		}

		// Sanitize key (replace spaces with underscores, lowercase)
		sanitizedKey := sanitizeLabelKey(key)
		if sanitizedKey == "" {
			continue
		}

		pairs = append(pairs, sanitizedKey, value)
	}

	return pairs
}

// sanitizeLabelKey ensures label keys follow the snake_case convention.
func sanitizeLabelKey(key string) string {
	// Convert to lowercase and replace spaces with underscores
	key = strings.ToLower(key)
	key = strings.ReplaceAll(key, " ", "_")
	key = strings.ReplaceAll(key, "-", "_")

	// Remove any characters that aren't alphanumeric or underscore
	result := make([]byte, 0, len(key))
	for i := 0; i < len(key); i++ {
		c := key[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' {
			result = append(result, c)
		}
	}

	return string(result)
}

// HTTPRequestLabels creates a standard set of labels for HTTP request profiling.
// This is a convenience function for creating consistent labels across handlers.
func HTTPRequestLabels(controller, route, method, tenantID string) map[string]string {
	labels := make(map[string]string, 4)

	if controller != "" {
		labels[ProfilingLabelController] = controller
	}
	if route != "" {
		labels[ProfilingLabelRoute] = route
	}
	if method != "" {
		labels[ProfilingLabelMethod] = method
	}
	if tenantID != "" {
		labels[ProfilingLabelTenantID] = tenantID
	}

	return labels
}

// OperationLabels creates labels for a named operation.
func OperationLabels(operation string, extraLabels map[string]string) map[string]string {
	labels := make(map[string]string, len(extraLabels)+1)
	labels[ProfilingLabelOperation] = operation
	maps.Copy(labels, extraLabels)

	return labels
}

// RegionLabels creates labels for a code region (e.g., database, external API).
func RegionLabels(region string, extraLabels map[string]string) map[string]string {
	labels := make(map[string]string, len(extraLabels)+1)
	labels[ProfilingLabelRegion] = region
	maps.Copy(labels, extraLabels)

	return labels
}
