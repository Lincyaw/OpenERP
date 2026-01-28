// Package pool provides the parameter pool implementation for the load generator.
// This file defines the ParameterValue structure and related types.
package pool

import (
	"time"

	"github.com/example/erp/tools/loadgen/internal/circuit"
)

// ParameterValue represents a stored parameter value with metadata.
// This is an alias for Value to maintain backward compatibility while
// providing the name specified in the design document.
type ParameterValue = Value

// NewParameterValue creates a new ParameterValue with the given data and semantic type.
func NewParameterValue(data any, semanticType circuit.SemanticType, source ValueSource) *ParameterValue {
	return &ParameterValue{
		Data:         data,
		SemanticType: semanticType,
		Source:       source,
		CreatedAt:    time.Now(),
	}
}

// NewParameterValueWithTTL creates a new ParameterValue with a custom TTL.
func NewParameterValueWithTTL(data any, semanticType circuit.SemanticType, source ValueSource, ttl time.Duration) *ParameterValue {
	now := time.Now()
	pv := &ParameterValue{
		Data:         data,
		SemanticType: semanticType,
		Source:       source,
		CreatedAt:    now,
	}
	if ttl > 0 {
		pv.ExpiresAt = now.Add(ttl)
	}
	return pv
}

// IsExpired returns true if the value has expired.
func (v *Value) IsExpired() bool {
	if v.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(v.ExpiresAt)
}

// IsExpiredAt returns true if the value has expired at the given time.
func (v *Value) IsExpiredAt(t time.Time) bool {
	if v.ExpiresAt.IsZero() {
		return false
	}
	return t.After(v.ExpiresAt)
}

// TTL returns the remaining time-to-live for this value.
// Returns 0 if the value has no expiration or has already expired.
func (v *Value) TTL() time.Duration {
	if v.ExpiresAt.IsZero() {
		return 0
	}
	remaining := time.Until(v.ExpiresAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Clone creates a deep copy of the Value.
// Note: The Data field is not deep-copied as it may contain various types.
func (v *Value) Clone() *Value {
	return &Value{
		Data:         v.Data,
		SemanticType: v.SemanticType,
		Source: ValueSource{
			Endpoint:      v.Source.Endpoint,
			ResponseField: v.Source.ResponseField,
			RequestID:     v.Source.RequestID,
		},
		CreatedAt:  v.CreatedAt,
		ExpiresAt:  v.ExpiresAt,
		UsageCount: v.UsageCount,
	}
}
