package shared

import (
	"time"

	"github.com/google/uuid"
)

// DomainEvent represents an event that occurred in the domain
type DomainEvent interface {
	EventID() uuid.UUID
	EventType() string
	OccurredAt() time.Time
	AggregateID() uuid.UUID
	AggregateType() string
	TenantID() uuid.UUID
}

// VersionedEvent extends DomainEvent with schema versioning support
// Events should implement this interface when they need backward-compatible
// schema evolution (adding/removing fields, changing field types, etc.)
type VersionedEvent interface {
	DomainEvent
	// SchemaVersion returns the version of the event schema (e.g., 1, 2, 3)
	// Default should be 1 for events that don't explicitly set a version
	SchemaVersion() int
}

// BaseDomainEvent provides common fields for all domain events
type BaseDomainEvent struct {
	ID            uuid.UUID `json:"id"`
	Type          string    `json:"type"`
	Timestamp     time.Time `json:"timestamp"`
	AggID         uuid.UUID `json:"aggregate_id"`
	AggType       string    `json:"aggregate_type"`
	TenantIDValue uuid.UUID `json:"tenant_id"`
	Version       int       `json:"schema_version,omitempty"` // Event schema version for backward compatibility
}

// EventID returns the unique event identifier
func (e *BaseDomainEvent) EventID() uuid.UUID {
	return e.ID
}

// EventType returns the type of the event
func (e *BaseDomainEvent) EventType() string {
	return e.Type
}

// OccurredAt returns when the event occurred
func (e *BaseDomainEvent) OccurredAt() time.Time {
	return e.Timestamp
}

// AggregateID returns the ID of the aggregate that produced this event
func (e *BaseDomainEvent) AggregateID() uuid.UUID {
	return e.AggID
}

// AggregateType returns the type of the aggregate
func (e *BaseDomainEvent) AggregateType() string {
	return e.AggType
}

// TenantID returns the tenant ID
func (e *BaseDomainEvent) TenantID() uuid.UUID {
	return e.TenantIDValue
}

// SchemaVersion returns the schema version of the event
// Returns 1 if no version is set (backward compatibility with unversioned events)
func (e *BaseDomainEvent) SchemaVersion() int {
	if e.Version == 0 {
		return 1
	}
	return e.Version
}

// NewBaseDomainEvent creates a new base domain event with default schema version 1
func NewBaseDomainEvent(eventType, aggType string, aggID, tenantID uuid.UUID) BaseDomainEvent {
	return BaseDomainEvent{
		ID:            uuid.New(),
		Type:          eventType,
		Timestamp:     time.Now(),
		AggID:         aggID,
		AggType:       aggType,
		TenantIDValue: tenantID,
		Version:       1,
	}
}

// NewVersionedBaseDomainEvent creates a new base domain event with explicit schema version
// If schemaVersion is less than 1, it defaults to 1 for safety
func NewVersionedBaseDomainEvent(eventType, aggType string, aggID, tenantID uuid.UUID, schemaVersion int) BaseDomainEvent {
	if schemaVersion < 1 {
		schemaVersion = 1
	}
	return BaseDomainEvent{
		ID:            uuid.New(),
		Type:          eventType,
		Timestamp:     time.Now(),
		AggID:         aggID,
		AggType:       aggType,
		TenantIDValue: tenantID,
		Version:       schemaVersion,
	}
}
