package telemetry_test

import (
	"context"
	"errors"
	"testing"

	"github.com/erp/backend/internal/infrastructure/telemetry"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// setupTestTracer sets up a test tracer with an in-memory span recorder.
// Returns the span recorder for assertions and a cleanup function.
func setupTestTracer(t *testing.T) (*tracetest.SpanRecorder, func()) {
	t.Helper()

	// Create an in-memory span recorder
	sr := tracetest.NewSpanRecorder()

	// Create a TracerProvider with the span recorder
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sr),
	)

	// Save the original provider and set the test provider
	originalProvider := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)

	// Return cleanup function
	cleanup := func() {
		otel.SetTracerProvider(originalProvider)
		_ = tp.Shutdown(context.Background())
	}

	return sr, cleanup
}

func TestStartSpan(t *testing.T) {
	sr, cleanup := setupTestTracer(t)
	defer cleanup()

	ctx := context.Background()

	// Start a span
	ctx, span := telemetry.StartSpan(ctx, "test.operation")
	require.NotNil(t, span)
	span.End()

	// Get recorded spans
	spans := sr.Ended()
	require.Len(t, spans, 1)

	assert.Equal(t, "test.operation", spans[0].Name())
	assert.Equal(t, trace.SpanKindInternal, spans[0].SpanKind())
}

func TestStartSpan_WithOptions(t *testing.T) {
	sr, cleanup := setupTestTracer(t)
	defer cleanup()

	ctx := context.Background()

	// Start a span with options
	ctx, span := telemetry.StartSpan(ctx, "test.operation",
		telemetry.WithAttribute("test_key", "test_value"),
		telemetry.WithSpanKind(trace.SpanKindClient),
	)
	require.NotNil(t, span)
	span.End()

	// Get recorded spans
	spans := sr.Ended()
	require.Len(t, spans, 1)

	assert.Equal(t, trace.SpanKindClient, spans[0].SpanKind())

	// Check attributes
	attrs := spans[0].Attributes()
	var found bool
	for _, attr := range attrs {
		if attr.Key == "test_key" && attr.Value.AsString() == "test_value" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected attribute 'test_key' not found")
}

func TestStartServiceSpan(t *testing.T) {
	sr, cleanup := setupTestTracer(t)
	defer cleanup()

	ctx := context.Background()

	// Start a service span
	ctx, span := telemetry.StartServiceSpan(ctx, "sales_order", "create")
	require.NotNil(t, span)
	span.End()

	// Get recorded spans
	spans := sr.Ended()
	require.Len(t, spans, 1)

	// Verify naming convention
	assert.Equal(t, "sales_order.create", spans[0].Name())
}

func TestSetAttributes(t *testing.T) {
	sr, cleanup := setupTestTracer(t)
	defer cleanup()

	ctx := context.Background()

	ctx, span := telemetry.StartSpan(ctx, "test.operation")

	// Set multiple attributes
	telemetry.SetAttributes(span,
		"string_attr", "value",
		"int_attr", 42,
		"bool_attr", true,
	)

	span.End()

	// Get recorded spans
	spans := sr.Ended()
	require.Len(t, spans, 1)

	// Check attributes
	attrs := spans[0].Attributes()
	attrMap := make(map[string]interface{})
	for _, attr := range attrs {
		attrMap[string(attr.Key)] = attr.Value.AsInterface()
	}

	assert.Equal(t, "value", attrMap["string_attr"])
	assert.Equal(t, int64(42), attrMap["int_attr"])
	assert.Equal(t, true, attrMap["bool_attr"])
}

func TestSetAttribute(t *testing.T) {
	sr, cleanup := setupTestTracer(t)
	defer cleanup()

	ctx := context.Background()

	ctx, span := telemetry.StartSpan(ctx, "test.operation")

	// Set single attribute
	telemetry.SetAttribute(span, "order_id", "12345")

	span.End()

	// Get recorded spans
	spans := sr.Ended()
	require.Len(t, spans, 1)

	// Check attribute
	attrs := spans[0].Attributes()
	var found bool
	for _, attr := range attrs {
		if attr.Key == "order_id" && attr.Value.AsString() == "12345" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected attribute 'order_id' not found")
}

func TestSetAttribute_WithUUID(t *testing.T) {
	sr, cleanup := setupTestTracer(t)
	defer cleanup()

	ctx := context.Background()

	ctx, span := telemetry.StartSpan(ctx, "test.operation")

	// Set UUID attribute using Stringer interface
	orderID := uuid.New()
	telemetry.SetAttribute(span, "order_id", orderID)

	span.End()

	// Get recorded spans
	spans := sr.Ended()
	require.Len(t, spans, 1)

	// Check attribute - UUID should be converted via fmt.Stringer
	attrs := spans[0].Attributes()
	var found bool
	for _, attr := range attrs {
		if attr.Key == "order_id" && attr.Value.AsString() == orderID.String() {
			found = true
			break
		}
	}
	assert.True(t, found, "expected attribute 'order_id' with UUID value not found")
}

func TestRecordError(t *testing.T) {
	sr, cleanup := setupTestTracer(t)
	defer cleanup()

	ctx := context.Background()

	ctx, span := telemetry.StartSpan(ctx, "test.operation")

	// Record an error
	testErr := errors.New("test error")
	telemetry.RecordError(span, testErr)

	span.End()

	// Get recorded spans
	spans := sr.Ended()
	require.Len(t, spans, 1)

	// Check status
	assert.Equal(t, codes.Error, spans[0].Status().Code)
	assert.Equal(t, "test error", spans[0].Status().Description)

	// Check events (error should be recorded as an event)
	events := spans[0].Events()
	require.GreaterOrEqual(t, len(events), 1)
	assert.Equal(t, "exception", events[0].Name)
}

func TestRecordError_NilError(t *testing.T) {
	sr, cleanup := setupTestTracer(t)
	defer cleanup()

	ctx := context.Background()

	ctx, span := telemetry.StartSpan(ctx, "test.operation")

	// Record nil error - should be no-op
	telemetry.RecordError(span, nil)

	span.End()

	// Get recorded spans
	spans := sr.Ended()
	require.Len(t, spans, 1)

	// Status should not be error
	assert.NotEqual(t, codes.Error, spans[0].Status().Code)
}

func TestRecordError_NilSpan(t *testing.T) {
	// Should not panic
	telemetry.RecordError(nil, errors.New("test error"))
}

func TestSetOK(t *testing.T) {
	sr, cleanup := setupTestTracer(t)
	defer cleanup()

	ctx := context.Background()

	ctx, span := telemetry.StartSpan(ctx, "test.operation")

	// Set OK status
	telemetry.SetOK(span)

	span.End()

	// Get recorded spans
	spans := sr.Ended()
	require.Len(t, spans, 1)

	// Check status
	assert.Equal(t, codes.Ok, spans[0].Status().Code)
}

func TestAddEvent(t *testing.T) {
	sr, cleanup := setupTestTracer(t)
	defer cleanup()

	ctx := context.Background()

	ctx, span := telemetry.StartSpan(ctx, "test.operation")

	// Add event with attributes
	telemetry.AddEvent(span, "stock_locked",
		"product_id", "prod-123",
		"quantity", 10,
	)

	span.End()

	// Get recorded spans
	spans := sr.Ended()
	require.Len(t, spans, 1)

	// Check events
	events := spans[0].Events()
	require.Len(t, events, 1)
	assert.Equal(t, "stock_locked", events[0].Name)

	// Check event attributes
	eventAttrs := events[0].Attributes
	attrMap := make(map[string]interface{})
	for _, attr := range eventAttrs {
		attrMap[string(attr.Key)] = attr.Value.AsInterface()
	}
	assert.Equal(t, "prod-123", attrMap["product_id"])
	assert.Equal(t, int64(10), attrMap["quantity"])
}

func TestSpanFromContext(t *testing.T) {
	_, cleanup := setupTestTracer(t)
	defer cleanup()

	ctx := context.Background()

	// No span in context
	span := telemetry.SpanFromContext(ctx)
	assert.NotNil(t, span) // Returns no-op span

	// With span in context
	ctx, createdSpan := telemetry.StartSpan(ctx, "test.operation")
	defer createdSpan.End()

	retrievedSpan := telemetry.SpanFromContext(ctx)
	assert.Equal(t, createdSpan.SpanContext().SpanID(), retrievedSpan.SpanContext().SpanID())
}

func TestGetTraceID(t *testing.T) {
	_, cleanup := setupTestTracer(t)
	defer cleanup()

	ctx := context.Background()

	// No span in context
	traceID := telemetry.GetTraceID(ctx)
	assert.Empty(t, traceID)

	// With span in context
	ctx, span := telemetry.StartSpan(ctx, "test.operation")
	defer span.End()

	traceID = telemetry.GetTraceID(ctx)
	assert.NotEmpty(t, traceID)
	assert.Len(t, traceID, 32) // TraceID is 16 bytes = 32 hex chars
}

func TestGetSpanID(t *testing.T) {
	_, cleanup := setupTestTracer(t)
	defer cleanup()

	ctx := context.Background()

	// No span in context
	spanID := telemetry.GetSpanID(ctx)
	assert.Empty(t, spanID)

	// With span in context
	ctx, span := telemetry.StartSpan(ctx, "test.operation")
	defer span.End()

	spanID = telemetry.GetSpanID(ctx)
	assert.NotEmpty(t, spanID)
	assert.Len(t, spanID, 16) // SpanID is 8 bytes = 16 hex chars
}

func TestContextWithSpan(t *testing.T) {
	_, cleanup := setupTestTracer(t)
	defer cleanup()

	ctx := context.Background()

	// Create a span
	_, span := telemetry.StartSpan(ctx, "test.operation")
	defer span.End()

	// Create new context with span
	newCtx := telemetry.ContextWithSpan(ctx, span)

	// Verify span is in new context
	retrievedSpan := telemetry.SpanFromContext(newCtx)
	assert.Equal(t, span.SpanContext().SpanID(), retrievedSpan.SpanContext().SpanID())
}

func TestNestedSpans(t *testing.T) {
	sr, cleanup := setupTestTracer(t)
	defer cleanup()

	ctx := context.Background()

	// Parent span
	ctx, parentSpan := telemetry.StartSpan(ctx, "parent.operation")

	// Child span
	_, childSpan := telemetry.StartSpan(ctx, "child.operation")
	childSpan.End()

	parentSpan.End()

	// Get recorded spans
	spans := sr.Ended()
	require.Len(t, spans, 2)

	// Find parent and child spans
	var parentIdx, childIdx int = -1, -1
	for i := range spans {
		if spans[i].Name() == "parent.operation" {
			parentIdx = i
		} else if spans[i].Name() == "child.operation" {
			childIdx = i
		}
	}

	require.NotEqual(t, -1, parentIdx, "parent span not found")
	require.NotEqual(t, -1, childIdx, "child span not found")

	parentSpanCtx := spans[parentIdx].SpanContext()
	childSpanCtx := spans[childIdx].SpanContext()
	childParentCtx := spans[childIdx].Parent()

	// Verify parent-child relationship
	assert.Equal(t, parentSpanCtx.TraceID(), childSpanCtx.TraceID())
	assert.Equal(t, parentSpanCtx.SpanID(), childParentCtx.SpanID())
}

func TestSetAttributes_NilSpan(t *testing.T) {
	// Should not panic
	telemetry.SetAttributes(nil, "key", "value")
}

func TestSetAttribute_NilSpan(t *testing.T) {
	// Should not panic
	telemetry.SetAttribute(nil, "key", "value")
}

func TestSetOK_NilSpan(t *testing.T) {
	// Should not panic
	telemetry.SetOK(nil)
}

func TestAddEvent_NilSpan(t *testing.T) {
	// Should not panic
	telemetry.AddEvent(nil, "event_name", "key", "value")
}

func TestAttributeTypes(t *testing.T) {
	sr, cleanup := setupTestTracer(t)
	defer cleanup()

	ctx := context.Background()

	ctx, span := telemetry.StartSpan(ctx, "test.operation")

	// Test various attribute types
	telemetry.SetAttributes(span,
		"string", "value",
		"int", 42,
		"int64", int64(100),
		"float64", 3.14,
		"bool", true,
		"string_slice", []string{"a", "b"},
		"int_slice", []int{1, 2, 3},
		"int64_slice", []int64{10, 20},
		"float64_slice", []float64{1.1, 2.2},
		"bool_slice", []bool{true, false},
	)

	span.End()

	// Get recorded spans
	spans := sr.Ended()
	require.Len(t, spans, 1)

	// Verify all attributes were recorded
	attrs := spans[0].Attributes()
	assert.GreaterOrEqual(t, len(attrs), 10)
}

func TestSetAttributes_OddKeyValues(t *testing.T) {
	sr, cleanup := setupTestTracer(t)
	defer cleanup()

	ctx := context.Background()

	ctx, span := telemetry.StartSpan(ctx, "test.operation")

	// Odd number of key-values - last one should be ignored
	telemetry.SetAttributes(span,
		"key1", "value1",
		"key2", "value2",
		"orphan_key", // No value
	)

	span.End()

	// Get recorded spans
	spans := sr.Ended()
	require.Len(t, spans, 1)

	// Only 2 attributes should be recorded
	attrs := spans[0].Attributes()
	assert.Len(t, attrs, 2)
}

func TestSetAttributes_NonStringKey(t *testing.T) {
	sr, cleanup := setupTestTracer(t)
	defer cleanup()

	ctx := context.Background()

	ctx, span := telemetry.StartSpan(ctx, "test.operation")

	// Non-string key should be skipped
	telemetry.SetAttributes(span,
		"valid_key", "value",
		123, "invalid_key", // This pair should be skipped
	)

	span.End()

	// Get recorded spans
	spans := sr.Ended()
	require.Len(t, spans, 1)

	// Only 1 attribute should be recorded
	attrs := spans[0].Attributes()
	assert.Len(t, attrs, 1)
}
