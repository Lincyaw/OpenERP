package acl

import (
	"context"

	"github.com/google/uuid"
)

// CustomerQueryService defines the interface for querying customer information
// from the Partner bounded context. This is the ACL's port for external queries.
//
// Implementations of this interface should:
// - First check the local cache (CustomerReferenceCache)
// - Fall back to the Partner context if not in cache
// - Update the local cache with fetched data
//
// This interface is defined in the Finance domain but implemented in the
// infrastructure layer, following the Dependency Inversion Principle.
type CustomerQueryService interface {
	// GetCustomerReference retrieves customer information for use in the Finance context.
	// It returns a CustomerReference value object that contains the minimal information
	// needed by Finance domain objects.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - tenantID: The tenant ID for multi-tenancy isolation
	//   - customerID: The customer's UUID
	//
	// Returns:
	//   - CustomerReference: The customer reference value object
	//   - error: Returns error if customer not found or on infrastructure failure
	GetCustomerReference(ctx context.Context, tenantID, customerID uuid.UUID) (CustomerReference, error)

	// GetCustomerReferences retrieves multiple customer references in batch.
	// This is more efficient than multiple single queries.
	//
	// Returns a map of customerID -> CustomerReference.
	// Missing customers will not be in the returned map (no error).
	GetCustomerReferences(ctx context.Context, tenantID uuid.UUID, customerIDs []uuid.UUID) (map[uuid.UUID]CustomerReference, error)

	// CustomerExists checks if a customer exists without fetching full details.
	// Useful for validation before creating financial records.
	CustomerExists(ctx context.Context, tenantID, customerID uuid.UUID) (bool, error)
}

// CustomerReferenceCache defines the interface for caching customer references
// within the Finance context. This cache is updated by event handlers listening
// to Partner context events.
//
// The cache serves two purposes:
// 1. Performance: Avoid repeated queries to Partner context
// 2. Eventual consistency: Store snapshot of customer data for Finance operations
type CustomerReferenceCache interface {
	// Get retrieves a customer reference from cache.
	// Returns (CustomerReference, true) if found, (empty, false) if not in cache.
	Get(ctx context.Context, tenantID, customerID uuid.UUID) (CustomerReference, bool)

	// Set stores a customer reference in cache.
	Set(ctx context.Context, tenantID uuid.UUID, ref CustomerReference) error

	// SetBatch stores multiple customer references efficiently.
	SetBatch(ctx context.Context, tenantID uuid.UUID, refs []CustomerReference) error

	// Invalidate removes a customer reference from cache.
	// Called when a customer is deleted or deactivated.
	Invalidate(ctx context.Context, tenantID, customerID uuid.UUID) error

	// InvalidateAll clears all cached references for a tenant.
	// Useful for cache maintenance or tenant data refresh.
	InvalidateAll(ctx context.Context, tenantID uuid.UUID) error
}

// CustomerEventHandler defines the interface for handling customer-related events
// from the Partner context. This is the reactive part of the ACL that maintains
// data consistency through event-driven updates.
//
// Implementations should:
// - Update the local CustomerReferenceCache
// - Optionally update denormalized customer data in Finance aggregates
type CustomerEventHandler interface {
	// HandleCustomerCreated processes CustomerCreatedEvent from Partner context.
	// Creates a new entry in the local cache.
	HandleCustomerCreated(ctx context.Context, event CustomerCreatedEventDTO) error

	// HandleCustomerUpdated processes CustomerUpdatedEvent from Partner context.
	// Updates existing cache entry and optionally propagates changes to Finance aggregates.
	HandleCustomerUpdated(ctx context.Context, event CustomerUpdatedEventDTO) error

	// HandleCustomerDeleted processes CustomerDeletedEvent from Partner context.
	// Invalidates cache entry. Note: Finance records referencing this customer
	// should remain intact for historical purposes.
	HandleCustomerDeleted(ctx context.Context, event CustomerDeletedEventDTO) error

	// HandleCustomerStatusChanged processes CustomerStatusChangedEvent.
	// May invalidate cache if customer becomes inactive/suspended.
	HandleCustomerStatusChanged(ctx context.Context, event CustomerStatusChangedEventDTO) error
}

// Event DTOs for cross-context communication
// These are local representations of Partner context events,
// isolating the Finance context from Partner's event structure.

// CustomerCreatedEventDTO represents the data from a CustomerCreatedEvent.
type CustomerCreatedEventDTO struct {
	TenantID   uuid.UUID
	CustomerID uuid.UUID
	Code       string
	Name       string
}

// CustomerUpdatedEventDTO represents the data from a CustomerUpdatedEvent.
type CustomerUpdatedEventDTO struct {
	TenantID    uuid.UUID
	CustomerID  uuid.UUID
	Code        string
	Name        string
	ContactName string
	Phone       string
	Email       string
}

// CustomerDeletedEventDTO represents the data from a CustomerDeletedEvent.
type CustomerDeletedEventDTO struct {
	TenantID   uuid.UUID
	CustomerID uuid.UUID
	Code       string
	Name       string
}

// CustomerStatusChangedEventDTO represents the data from a CustomerStatusChangedEvent.
type CustomerStatusChangedEventDTO struct {
	TenantID   uuid.UUID
	CustomerID uuid.UUID
	Code       string
	OldStatus  string
	NewStatus  string
}
