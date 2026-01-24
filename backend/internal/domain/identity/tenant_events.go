package identity

import (
	"github.com/erp/backend/internal/domain/shared"
)

// Aggregate type constant
const AggregateTypeTenant = "Tenant"

// Event type constants
const (
	EventTypeTenantCreated       = "TenantCreated"
	EventTypeTenantUpdated       = "TenantUpdated"
	EventTypeTenantStatusChanged = "TenantStatusChanged"
	EventTypeTenantPlanChanged   = "TenantPlanChanged"
	EventTypeTenantDeleted       = "TenantDeleted"
)

// TenantCreatedEvent is published when a new tenant is created
type TenantCreatedEvent struct {
	shared.BaseDomainEvent
	Code   string       `json:"code"`
	Name   string       `json:"name"`
	Status TenantStatus `json:"status"`
	Plan   TenantPlan   `json:"plan"`
}

// NewTenantCreatedEvent creates a new TenantCreatedEvent
func NewTenantCreatedEvent(tenant *Tenant) *TenantCreatedEvent {
	return &TenantCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeTenantCreated, AggregateTypeTenant, tenant.ID, tenant.ID),
		Code:            tenant.Code,
		Name:            tenant.Name,
		Status:          tenant.Status,
		Plan:            tenant.Plan,
	}
}

// TenantUpdatedEvent is published when a tenant is updated
type TenantUpdatedEvent struct {
	shared.BaseDomainEvent
	Code         string `json:"code"`
	Name         string `json:"name"`
	ShortName    string `json:"short_name,omitempty"`
	ContactName  string `json:"contact_name,omitempty"`
	ContactPhone string `json:"contact_phone,omitempty"`
	ContactEmail string `json:"contact_email,omitempty"`
}

// NewTenantUpdatedEvent creates a new TenantUpdatedEvent
func NewTenantUpdatedEvent(tenant *Tenant) *TenantUpdatedEvent {
	return &TenantUpdatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeTenantUpdated, AggregateTypeTenant, tenant.ID, tenant.ID),
		Code:            tenant.Code,
		Name:            tenant.Name,
		ShortName:       tenant.ShortName,
		ContactName:     tenant.ContactName,
		ContactPhone:    tenant.ContactPhone,
		ContactEmail:    tenant.ContactEmail,
	}
}

// TenantStatusChangedEvent is published when a tenant's status changes
type TenantStatusChangedEvent struct {
	shared.BaseDomainEvent
	Code      string       `json:"code"`
	OldStatus TenantStatus `json:"old_status"`
	NewStatus TenantStatus `json:"new_status"`
}

// NewTenantStatusChangedEvent creates a new TenantStatusChangedEvent
func NewTenantStatusChangedEvent(tenant *Tenant, oldStatus, newStatus TenantStatus) *TenantStatusChangedEvent {
	return &TenantStatusChangedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeTenantStatusChanged, AggregateTypeTenant, tenant.ID, tenant.ID),
		Code:            tenant.Code,
		OldStatus:       oldStatus,
		NewStatus:       newStatus,
	}
}

// TenantPlanChangedEvent is published when a tenant's subscription plan changes
type TenantPlanChangedEvent struct {
	shared.BaseDomainEvent
	Code    string     `json:"code"`
	OldPlan TenantPlan `json:"old_plan"`
	NewPlan TenantPlan `json:"new_plan"`
}

// NewTenantPlanChangedEvent creates a new TenantPlanChangedEvent
func NewTenantPlanChangedEvent(tenant *Tenant, oldPlan, newPlan TenantPlan) *TenantPlanChangedEvent {
	return &TenantPlanChangedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeTenantPlanChanged, AggregateTypeTenant, tenant.ID, tenant.ID),
		Code:            tenant.Code,
		OldPlan:         oldPlan,
		NewPlan:         newPlan,
	}
}

// TenantDeletedEvent is published when a tenant is deleted
type TenantDeletedEvent struct {
	shared.BaseDomainEvent
	Code string `json:"code"`
	Name string `json:"name"`
}

// NewTenantDeletedEvent creates a new TenantDeletedEvent
func NewTenantDeletedEvent(tenant *Tenant) *TenantDeletedEvent {
	return &TenantDeletedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeTenantDeleted, AggregateTypeTenant, tenant.ID, tenant.ID),
		Code:            tenant.Code,
		Name:            tenant.Name,
	}
}
