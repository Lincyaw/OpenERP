package shared

import (
	"github.com/google/uuid"
)

// AggregateRoot is the base interface for all aggregate roots
type AggregateRoot interface {
	Entity
	GetVersion() int
	IncrementVersion()
	AddDomainEvent(event DomainEvent)
	GetDomainEvents() []DomainEvent
	ClearDomainEvents()
}

// BaseAggregateRoot provides common fields for aggregate roots
type BaseAggregateRoot struct {
	BaseEntity
	Version      int           `gorm:"not null;default:1"`
	domainEvents []DomainEvent `gorm:"-"`
}

// GetVersion returns the aggregate version for optimistic locking
func (a *BaseAggregateRoot) GetVersion() int {
	return a.Version
}

// IncrementVersion increments the version number
func (a *BaseAggregateRoot) IncrementVersion() {
	a.Version++
}

// AddDomainEvent adds a domain event to be published
func (a *BaseAggregateRoot) AddDomainEvent(event DomainEvent) {
	a.domainEvents = append(a.domainEvents, event)
}

// GetDomainEvents returns all pending domain events
func (a *BaseAggregateRoot) GetDomainEvents() []DomainEvent {
	return a.domainEvents
}

// ClearDomainEvents clears the pending domain events
func (a *BaseAggregateRoot) ClearDomainEvents() {
	a.domainEvents = nil
}

// NewBaseAggregateRoot creates a new base aggregate root
func NewBaseAggregateRoot() BaseAggregateRoot {
	return BaseAggregateRoot{
		BaseEntity:   NewBaseEntity(),
		Version:      1,
		domainEvents: make([]DomainEvent, 0),
	}
}

// TenantAggregateRoot extends BaseAggregateRoot with multi-tenant support
type TenantAggregateRoot struct {
	BaseAggregateRoot
	TenantID uuid.UUID `gorm:"type:uuid;not null;index"`
}

// NewTenantAggregateRoot creates a new tenant-scoped aggregate root
func NewTenantAggregateRoot(tenantID uuid.UUID) TenantAggregateRoot {
	return TenantAggregateRoot{
		BaseAggregateRoot: NewBaseAggregateRoot(),
		TenantID:          tenantID,
	}
}
