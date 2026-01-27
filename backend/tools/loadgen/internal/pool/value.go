// Package pool provides parameter pool implementations for the load generator.
// It supports storing and retrieving values by semantic type with TTL expiration.
package pool

import (
	"sync/atomic"
	"time"
)

// SemanticType represents a semantic classification of parameter values.
// Examples: entity.customer.id, order.sales_order.id, common.email
type SemanticType string

// Common semantic types used in ERP systems
const (
	// Entity types
	SemanticTypeCustomerID  SemanticType = "entity.customer.id"
	SemanticTypeProductID   SemanticType = "entity.product.id"
	SemanticTypeSupplierID  SemanticType = "entity.supplier.id"
	SemanticTypeWarehouseID SemanticType = "entity.warehouse.id"
	SemanticTypeUserID      SemanticType = "entity.user.id"

	// Order types
	SemanticTypeSalesOrderID    SemanticType = "order.sales_order.id"
	SemanticTypePurchaseOrderID SemanticType = "order.purchase_order.id"

	// Finance types
	SemanticTypeInvoiceID  SemanticType = "finance.invoice.id"
	SemanticTypeReceiptID  SemanticType = "finance.receipt.id"
	SemanticTypeAccountID  SemanticType = "finance.account.id"
	SemanticTypeCurrencyID SemanticType = "finance.currency.id"

	// Common types
	SemanticTypeEmail     SemanticType = "common.email"
	SemanticTypePhone     SemanticType = "common.phone"
	SemanticTypeAddress   SemanticType = "common.address"
	SemanticTypeSKU       SemanticType = "common.sku"
	SemanticTypeBarcode   SemanticType = "common.barcode"
	SemanticTypeTimestamp SemanticType = "common.timestamp"
	SemanticTypeUUID      SemanticType = "common.uuid"
)

// ParameterValue represents a value stored in the parameter pool.
// It includes metadata about the value's origin and expiration.
// Note: Touch() is called under the parent container's lock, making it thread-safe.
type ParameterValue struct {
	// Value holds the actual parameter value (can be any JSON-compatible type)
	// Note: Value should be treated as immutable after creation.
	Value any

	// SemanticType identifies the semantic classification of this value
	SemanticType SemanticType

	// SourceEndpoint is the endpoint that produced this value (e.g., "POST /customers")
	SourceEndpoint string

	// ResponsePath is the JSONPath to extract this value (e.g., "$.data.id")
	ResponsePath string

	// CreatedAt is when this value was added to the pool
	CreatedAt time.Time

	// ExpiresAt is when this value should be considered expired (zero means no expiration)
	ExpiresAt time.Time

	// accessCount tracks how many times this value has been retrieved (atomic for thread safety)
	accessCount atomic.Int64

	// lastAccessedAt tracks when this value was last retrieved (atomic for thread safety)
	lastAccessedAt atomic.Int64 // stores Unix nanoseconds
}

// NewParameterValue creates a new ParameterValue with the given value and semantic type.
// TTL of 0 means the value never expires.
func NewParameterValue(value any, semanticType SemanticType, ttl time.Duration) *ParameterValue {
	now := time.Now()
	pv := &ParameterValue{
		Value:        value,
		SemanticType: semanticType,
		CreatedAt:    now,
	}
	pv.lastAccessedAt.Store(now.UnixNano())
	if ttl > 0 {
		pv.ExpiresAt = now.Add(ttl)
	}
	return pv
}

// WithSource sets the source endpoint and response path for this value.
func (pv *ParameterValue) WithSource(endpoint, path string) *ParameterValue {
	pv.SourceEndpoint = endpoint
	pv.ResponsePath = path
	return pv
}

// IsExpired returns true if this value has expired.
func (pv *ParameterValue) IsExpired() bool {
	if pv.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(pv.ExpiresAt)
}

// Touch updates the access statistics for this value.
// This method is thread-safe using atomic operations.
func (pv *ParameterValue) Touch() {
	pv.accessCount.Add(1)
	pv.lastAccessedAt.Store(time.Now().UnixNano())
}

// AccessCount returns the number of times this value has been accessed.
func (pv *ParameterValue) AccessCount() int64 {
	return pv.accessCount.Load()
}

// LastAccessedAt returns when this value was last accessed.
func (pv *ParameterValue) LastAccessedAt() time.Time {
	return time.Unix(0, pv.lastAccessedAt.Load())
}

// Clone creates a copy of this ParameterValue.
// Note: The Value field is copied by reference; caller should treat it as immutable.
func (pv *ParameterValue) Clone() *ParameterValue {
	clone := &ParameterValue{
		Value:          pv.Value,
		SemanticType:   pv.SemanticType,
		SourceEndpoint: pv.SourceEndpoint,
		ResponsePath:   pv.ResponsePath,
		CreatedAt:      pv.CreatedAt,
		ExpiresAt:      pv.ExpiresAt,
	}
	clone.accessCount.Store(pv.accessCount.Load())
	clone.lastAccessedAt.Store(pv.lastAccessedAt.Load())
	return clone
}
