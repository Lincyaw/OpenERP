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

// BaseDomainEvent provides common fields for all domain events
type BaseDomainEvent struct {
	ID            uuid.UUID `json:"id"`
	Type          string    `json:"type"`
	Timestamp     time.Time `json:"timestamp"`
	AggID         uuid.UUID `json:"aggregate_id"`
	AggType       string    `json:"aggregate_type"`
	TenantIDValue uuid.UUID `json:"tenant_id"`
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

// NewBaseDomainEvent creates a new base domain event
func NewBaseDomainEvent(eventType, aggType string, aggID, tenantID uuid.UUID) BaseDomainEvent {
	return BaseDomainEvent{
		ID:            uuid.New(),
		Type:          eventType,
		Timestamp:     time.Now(),
		AggID:         aggID,
		AggType:       aggType,
		TenantIDValue: tenantID,
	}
}
