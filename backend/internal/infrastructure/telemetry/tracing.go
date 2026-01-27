// Package telemetry provides OpenTelemetry integration for distributed tracing.
// This file contains utility functions for business-level tracing in application services.
package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	// TracerName is the default tracer name for business spans
	TracerName = "erp-backend"
)

// SpanOption is a function that configures span start options
type SpanOption func(*spanOptions)

type spanOptions struct {
	attributes []attribute.KeyValue
	kind       trace.SpanKind
}

// WithAttribute adds an attribute to the span
func WithAttribute(key string, value interface{}) SpanOption {
	return func(opts *spanOptions) {
		opts.attributes = append(opts.attributes, toAttribute(key, value))
	}
}

// WithSpanKind sets the span kind
func WithSpanKind(kind trace.SpanKind) SpanOption {
	return func(opts *spanOptions) {
		opts.kind = kind
	}
}

// StartSpan starts a new span with the given name.
// It returns a new context containing the span and the span itself.
// The caller is responsible for calling span.End() when the operation completes.
//
// Example usage:
//
//	ctx, span := telemetry.StartSpan(ctx, "sales_order.create")
//	defer span.End()
//
//	// ... business logic ...
//
//	if err != nil {
//	    telemetry.RecordError(span, err)
//	    return err
//	}
func StartSpan(ctx context.Context, spanName string, opts ...SpanOption) (context.Context, trace.Span) {
	// Apply options
	options := &spanOptions{
		kind: trace.SpanKindInternal,
	}
	for _, opt := range opts {
		opt(options)
	}

	// Get tracer
	tracer := otel.GetTracerProvider().Tracer(TracerName)

	// Build span start options
	startOpts := []trace.SpanStartOption{
		trace.WithSpanKind(options.kind),
	}
	if len(options.attributes) > 0 {
		startOpts = append(startOpts, trace.WithAttributes(options.attributes...))
	}

	return tracer.Start(ctx, spanName, startOpts...)
}

// StartServiceSpan starts a span for a service method.
// It follows the naming convention {service}.{method} (e.g., "sales_order.create").
//
// Example:
//
//	ctx, span := telemetry.StartServiceSpan(ctx, "sales_order", "create")
//	defer span.End()
func StartServiceSpan(ctx context.Context, service, method string, opts ...SpanOption) (context.Context, trace.Span) {
	spanName := fmt.Sprintf("%s.%s", service, method)
	return StartSpan(ctx, spanName, opts...)
}

// SetAttributes adds attributes to an existing span.
// Attributes provide context about the operation being traced.
//
// Example:
//
//	telemetry.SetAttributes(span,
//	    "order_id", orderID.String(),
//	    "customer_id", customerID.String(),
//	    "items_count", len(items),
//	)
func SetAttributes(span trace.Span, keyValues ...interface{}) {
	if span == nil {
		return
	}

	attrs := make([]attribute.KeyValue, 0, len(keyValues)/2)
	for i := 0; i+1 < len(keyValues); i += 2 {
		key, ok := keyValues[i].(string)
		if !ok {
			continue
		}
		attrs = append(attrs, toAttribute(key, keyValues[i+1]))
	}

	span.SetAttributes(attrs...)
}

// SetAttribute adds a single attribute to the span.
func SetAttribute(span trace.Span, key string, value interface{}) {
	if span == nil {
		return
	}
	span.SetAttributes(toAttribute(key, value))
}

// RecordError records an error on the span and sets the span status to error.
// This should be called when an operation fails.
//
// Example:
//
//	if err := s.orderRepo.Save(ctx, order); err != nil {
//	    telemetry.RecordError(span, err)
//	    return nil, err
//	}
func RecordError(span trace.Span, err error, opts ...trace.EventOption) {
	if span == nil || err == nil {
		return
	}
	span.RecordError(err, opts...)
	span.SetStatus(codes.Error, err.Error())
}

// SetOK marks the span as successful.
// This is optional since spans without an error status are considered successful.
func SetOK(span trace.Span) {
	if span == nil {
		return
	}
	span.SetStatus(codes.Ok, "")
}

// AddEvent adds an event to the span with optional attributes.
// Events are time-stamped annotations that can be used to record
// significant occurrences during the span's lifetime.
//
// Example:
//
//	telemetry.AddEvent(span, "stock_locked",
//	    "product_id", productID.String(),
//	    "quantity", quantity,
//	)
func AddEvent(span trace.Span, name string, keyValues ...interface{}) {
	if span == nil {
		return
	}

	attrs := make([]attribute.KeyValue, 0, len(keyValues)/2)
	for i := 0; i+1 < len(keyValues); i += 2 {
		key, ok := keyValues[i].(string)
		if !ok {
			continue
		}
		attrs = append(attrs, toAttribute(key, keyValues[i+1]))
	}

	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SpanFromContext returns the current span from the context.
// This is useful when you need to add attributes or events to an existing span.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// ContextWithSpan returns a new context containing the given span.
func ContextWithSpan(ctx context.Context, span trace.Span) context.Context {
	return trace.ContextWithSpan(ctx, span)
}

// GetTraceID returns the trace ID from the current span in the context.
// Returns empty string if no span is present.
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return ""
	}
	traceID := span.SpanContext().TraceID()
	if !traceID.IsValid() {
		return ""
	}
	return traceID.String()
}

// GetSpanID returns the span ID from the current span in the context.
// Returns empty string if no span is present.
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return ""
	}
	spanID := span.SpanContext().SpanID()
	if !spanID.IsValid() {
		return ""
	}
	return spanID.String()
}

// toAttribute converts a key-value pair to an attribute.KeyValue
func toAttribute(key string, value interface{}) attribute.KeyValue {
	switch v := value.(type) {
	case string:
		return attribute.String(key, v)
	case int:
		return attribute.Int(key, v)
	case int64:
		return attribute.Int64(key, v)
	case float64:
		return attribute.Float64(key, v)
	case bool:
		return attribute.Bool(key, v)
	case []string:
		return attribute.StringSlice(key, v)
	case []int:
		return attribute.IntSlice(key, v)
	case []int64:
		return attribute.Int64Slice(key, v)
	case []float64:
		return attribute.Float64Slice(key, v)
	case []bool:
		return attribute.BoolSlice(key, v)
	case fmt.Stringer:
		return attribute.String(key, v.String())
	default:
		return attribute.String(key, fmt.Sprintf("%v", v))
	}
}

// Common attribute keys for business spans (string keys for trace attributes)
// Note: Metric attributes are defined in metrics.go as attribute.Key types.
// These string constants are for trace span attributes only.
const (
	// Order attributes
	SpanAttrOrderID     = "order_id"
	SpanAttrOrderNumber = "order_number"
	SpanAttrOrderStatus = "order_status"

	// Customer attributes
	SpanAttrCustomerID   = "customer_id"
	SpanAttrCustomerName = "customer_name"

	// Product and inventory attributes
	SpanAttrProductCode = "product_code"
	SpanAttrQuantity    = "quantity"

	// Payment attributes
	SpanAttrPaymentID      = "payment_id"
	SpanAttrPaymentGateway = "payment_gateway"
	SpanAttrAmount         = "amount"

	// Lock attributes
	SpanAttrLockID     = "lock_id"
	SpanAttrSourceType = "source_type"
	SpanAttrSourceID   = "source_id"
)
