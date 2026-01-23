package partner

import (
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// Aggregate type constant for Warehouse
const AggregateTypeWarehouse = "Warehouse"

// Event type constants for Warehouse
const (
	EventTypeWarehouseCreated       = "WarehouseCreated"
	EventTypeWarehouseUpdated       = "WarehouseUpdated"
	EventTypeWarehouseStatusChanged = "WarehouseStatusChanged"
	EventTypeWarehouseSetAsDefault  = "WarehouseSetAsDefault"
	EventTypeWarehouseDeleted       = "WarehouseDeleted"
)

// WarehouseCreatedEvent is published when a new warehouse is created
type WarehouseCreatedEvent struct {
	shared.BaseDomainEvent
	WarehouseID uuid.UUID     `json:"warehouse_id"`
	Code        string        `json:"code"`
	Name        string        `json:"name"`
	Type        WarehouseType `json:"type"`
}

// NewWarehouseCreatedEvent creates a new WarehouseCreatedEvent
func NewWarehouseCreatedEvent(warehouse *Warehouse) *WarehouseCreatedEvent {
	return &WarehouseCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeWarehouseCreated, AggregateTypeWarehouse, warehouse.ID, warehouse.TenantID),
		WarehouseID:     warehouse.ID,
		Code:            warehouse.Code,
		Name:            warehouse.Name,
		Type:            warehouse.Type,
	}
}

// WarehouseUpdatedEvent is published when a warehouse is updated
type WarehouseUpdatedEvent struct {
	shared.BaseDomainEvent
	WarehouseID uuid.UUID `json:"warehouse_id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	ShortName   string    `json:"short_name,omitempty"`
	ContactName string    `json:"contact_name,omitempty"`
	Phone       string    `json:"phone,omitempty"`
	Email       string    `json:"email,omitempty"`
}

// NewWarehouseUpdatedEvent creates a new WarehouseUpdatedEvent
func NewWarehouseUpdatedEvent(warehouse *Warehouse) *WarehouseUpdatedEvent {
	return &WarehouseUpdatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeWarehouseUpdated, AggregateTypeWarehouse, warehouse.ID, warehouse.TenantID),
		WarehouseID:     warehouse.ID,
		Code:            warehouse.Code,
		Name:            warehouse.Name,
		ShortName:       warehouse.ShortName,
		ContactName:     warehouse.ContactName,
		Phone:           warehouse.Phone,
		Email:           warehouse.Email,
	}
}

// WarehouseStatusChangedEvent is published when a warehouse's status changes
type WarehouseStatusChangedEvent struct {
	shared.BaseDomainEvent
	WarehouseID uuid.UUID       `json:"warehouse_id"`
	Code        string          `json:"code"`
	OldStatus   WarehouseStatus `json:"old_status"`
	NewStatus   WarehouseStatus `json:"new_status"`
}

// NewWarehouseStatusChangedEvent creates a new WarehouseStatusChangedEvent
func NewWarehouseStatusChangedEvent(warehouse *Warehouse, oldStatus, newStatus WarehouseStatus) *WarehouseStatusChangedEvent {
	return &WarehouseStatusChangedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeWarehouseStatusChanged, AggregateTypeWarehouse, warehouse.ID, warehouse.TenantID),
		WarehouseID:     warehouse.ID,
		Code:            warehouse.Code,
		OldStatus:       oldStatus,
		NewStatus:       newStatus,
	}
}

// WarehouseSetAsDefaultEvent is published when a warehouse is set as the default
type WarehouseSetAsDefaultEvent struct {
	shared.BaseDomainEvent
	WarehouseID uuid.UUID `json:"warehouse_id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
}

// NewWarehouseSetAsDefaultEvent creates a new WarehouseSetAsDefaultEvent
func NewWarehouseSetAsDefaultEvent(warehouse *Warehouse) *WarehouseSetAsDefaultEvent {
	return &WarehouseSetAsDefaultEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeWarehouseSetAsDefault, AggregateTypeWarehouse, warehouse.ID, warehouse.TenantID),
		WarehouseID:     warehouse.ID,
		Code:            warehouse.Code,
		Name:            warehouse.Name,
	}
}

// WarehouseDeletedEvent is published when a warehouse is deleted
type WarehouseDeletedEvent struct {
	shared.BaseDomainEvent
	WarehouseID uuid.UUID `json:"warehouse_id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
}

// NewWarehouseDeletedEvent creates a new WarehouseDeletedEvent
func NewWarehouseDeletedEvent(warehouse *Warehouse) *WarehouseDeletedEvent {
	return &WarehouseDeletedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeWarehouseDeleted, AggregateTypeWarehouse, warehouse.ID, warehouse.TenantID),
		WarehouseID:     warehouse.ID,
		Code:            warehouse.Code,
		Name:            warehouse.Name,
	}
}
