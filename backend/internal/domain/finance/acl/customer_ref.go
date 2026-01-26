// Package acl provides Anti-Corruption Layer (ACL) components for the Finance bounded context.
// ACL protects the Finance domain from direct dependencies on external bounded contexts
// such as Partner (Customer) while maintaining data consistency through event-driven updates.
//
// DDD-H04: Cross-context reference with ACL
package acl

import (
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// CustomerID is a value object representing a customer identifier within the Finance context.
// This provides type safety and isolates the Finance domain from the Partner domain's
// internal representation of customer identifiers.
//
// Unlike using uuid.UUID directly, CustomerID:
// - Provides explicit semantic meaning in the Finance domain
// - Prevents accidental mixing with other UUID-based IDs
// - Allows for potential future representation changes without affecting the domain
type CustomerID struct {
	value uuid.UUID
}

// NewCustomerID creates a new CustomerID from a UUID.
// Returns an error if the UUID is nil/empty.
func NewCustomerID(id uuid.UUID) (CustomerID, error) {
	if id == uuid.Nil {
		return CustomerID{}, shared.NewDomainError("INVALID_CUSTOMER_ID", "Customer ID cannot be empty")
	}
	return CustomerID{value: id}, nil
}

// MustNewCustomerID creates a new CustomerID, panicking if the ID is invalid.
// Use only when the ID is guaranteed to be valid (e.g., from database).
func MustNewCustomerID(id uuid.UUID) CustomerID {
	cid, err := NewCustomerID(id)
	if err != nil {
		panic(err)
	}
	return cid
}

// ParseCustomerID parses a string into a CustomerID.
// Returns an error if the string is not a valid UUID or is empty.
func ParseCustomerID(s string) (CustomerID, error) {
	if s == "" {
		return CustomerID{}, shared.NewDomainError("INVALID_CUSTOMER_ID", "Customer ID cannot be empty")
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return CustomerID{}, shared.NewDomainError("INVALID_CUSTOMER_ID", "Customer ID is not a valid UUID")
	}
	return NewCustomerID(id)
}

// UUID returns the underlying UUID value.
// This is used for persistence and integration with external systems.
func (c CustomerID) UUID() uuid.UUID {
	return c.value
}

// String returns the string representation of the CustomerID.
func (c CustomerID) String() string {
	return c.value.String()
}

// IsEmpty returns true if the CustomerID is empty (nil UUID).
func (c CustomerID) IsEmpty() bool {
	return c.value == uuid.Nil
}

// Equals checks if two CustomerIDs are equal.
func (c CustomerID) Equals(other CustomerID) bool {
	return c.value == other.value
}

// CustomerReference is a value object that holds denormalized customer information
// needed by the Finance context. This is the Finance context's local view of a customer,
// maintained through event-driven synchronization from the Partner context.
//
// This pattern follows DDD's recommendation for cross-bounded-context references:
// - Store minimal necessary information (ID and name for display)
// - Update through domain events from the source context
// - Never directly query the source context from the domain layer
type CustomerReference struct {
	id   CustomerID
	name string
	code string // Customer code for display (e.g., "CUST-001")
}

// NewCustomerReference creates a new CustomerReference.
// Returns an error if the ID is empty or the name is empty.
func NewCustomerReference(id CustomerID, name, code string) (CustomerReference, error) {
	if id.IsEmpty() {
		return CustomerReference{}, shared.NewDomainError("INVALID_CUSTOMER_ID", "Customer ID cannot be empty")
	}
	if name == "" {
		return CustomerReference{}, shared.NewDomainError("INVALID_CUSTOMER_NAME", "Customer name cannot be empty")
	}
	// Code is optional but useful for display

	return CustomerReference{
		id:   id,
		name: name,
		code: code,
	}, nil
}

// NewCustomerReferenceFromUUID creates a CustomerReference from raw UUID and name.
// This is a convenience method for creating references from database records.
func NewCustomerReferenceFromUUID(id uuid.UUID, name, code string) (CustomerReference, error) {
	customerID, err := NewCustomerID(id)
	if err != nil {
		return CustomerReference{}, err
	}
	return NewCustomerReference(customerID, name, code)
}

// MustNewCustomerReference creates a CustomerReference, panicking if invalid.
// Use only when inputs are guaranteed to be valid (e.g., from database).
func MustNewCustomerReference(id uuid.UUID, name, code string) CustomerReference {
	ref, err := NewCustomerReferenceFromUUID(id, name, code)
	if err != nil {
		panic(err)
	}
	return ref
}

// ID returns the CustomerID.
func (r CustomerReference) ID() CustomerID {
	return r.id
}

// UUID returns the underlying UUID of the customer ID.
// Convenience method to avoid r.ID().UUID() calls.
func (r CustomerReference) UUID() uuid.UUID {
	return r.id.UUID()
}

// Name returns the customer name.
func (r CustomerReference) Name() string {
	return r.name
}

// Code returns the customer code.
func (r CustomerReference) Code() string {
	return r.code
}

// DisplayName returns a formatted display name (code + name if code exists).
func (r CustomerReference) DisplayName() string {
	if r.code != "" {
		return r.code + " - " + r.name
	}
	return r.name
}

// IsEmpty returns true if the reference is empty.
func (r CustomerReference) IsEmpty() bool {
	return r.id.IsEmpty()
}

// Equals checks if two CustomerReferences are equal (by ID).
func (r CustomerReference) Equals(other CustomerReference) bool {
	return r.id.Equals(other.id)
}

// WithUpdatedInfo returns a new CustomerReference with updated name and code.
// This is used when processing CustomerUpdatedEvent.
func (r CustomerReference) WithUpdatedInfo(name, code string) (CustomerReference, error) {
	if name == "" {
		return CustomerReference{}, shared.NewDomainError("INVALID_CUSTOMER_NAME", "Customer name cannot be empty")
	}
	return CustomerReference{
		id:   r.id,
		name: name,
		code: code,
	}, nil
}

// EmptyCustomerReference returns an empty CustomerReference.
// Used as a zero value or for optional customer references.
func EmptyCustomerReference() CustomerReference {
	return CustomerReference{}
}
