package identity

import (
	"time"

	"github.com/erp/backend/internal/domain/shared"
)

// Aggregate type constant
const AggregateTypeTenant = "Tenant"

// Event type constants
const (
	EventTypeTenantCreated                  = "TenantCreated"
	EventTypeTenantUpdated                  = "TenantUpdated"
	EventTypeTenantStatusChanged            = "TenantStatusChanged"
	EventTypeTenantPlanChanged              = "TenantPlanChanged"
	EventTypeTenantDeleted                  = "TenantDeleted"
	EventTypeTenantSuspended                = "TenantSuspended"
	EventTypeTenantActivated                = "TenantActivated"
	EventTypeSubscriptionUpgraded           = "SubscriptionUpgraded"
	EventTypeSubscriptionDowngradeScheduled = "SubscriptionDowngradeScheduled"
	EventTypeQuotaUpdated                   = "QuotaUpdated"
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

// SubscriptionUpgradedEvent is published when a tenant upgrades their plan
type SubscriptionUpgradedEvent struct {
	shared.BaseDomainEvent
	TenantCode string     `json:"tenant_code"`
	OldPlan    TenantPlan `json:"old_plan"`
	NewPlan    TenantPlan `json:"new_plan"`
}

// NewSubscriptionUpgradedEvent creates a new SubscriptionUpgradedEvent
func NewSubscriptionUpgradedEvent(tenant *Tenant, oldPlan, newPlan TenantPlan) *SubscriptionUpgradedEvent {
	return &SubscriptionUpgradedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeSubscriptionUpgraded, AggregateTypeTenant, tenant.ID, tenant.ID),
		TenantCode:      tenant.Code,
		OldPlan:         oldPlan,
		NewPlan:         newPlan,
	}
}

// SubscriptionDowngradeScheduledEvent is published when a downgrade is scheduled
type SubscriptionDowngradeScheduledEvent struct {
	shared.BaseDomainEvent
	TenantCode  string     `json:"tenant_code"`
	CurrentPlan TenantPlan `json:"current_plan"`
	NewPlan     TenantPlan `json:"new_plan"`
	EffectiveAt time.Time  `json:"effective_at"`
}

// NewSubscriptionDowngradeScheduledEvent creates a new SubscriptionDowngradeScheduledEvent
func NewSubscriptionDowngradeScheduledEvent(tenant *Tenant, currentPlan, newPlan TenantPlan, effectiveAt time.Time) *SubscriptionDowngradeScheduledEvent {
	return &SubscriptionDowngradeScheduledEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeSubscriptionDowngradeScheduled, AggregateTypeTenant, tenant.ID, tenant.ID),
		TenantCode:      tenant.Code,
		CurrentPlan:     currentPlan,
		NewPlan:         newPlan,
		EffectiveAt:     effectiveAt,
	}
}

// QuotaUpdatedEvent is published when tenant quotas are updated
type QuotaUpdatedEvent struct {
	shared.BaseDomainEvent
	TenantCode string      `json:"tenant_code"`
	OldQuota   TenantQuota `json:"old_quota"`
	NewQuota   TenantQuota `json:"new_quota"`
}

// NewQuotaUpdatedEvent creates a new QuotaUpdatedEvent
func NewQuotaUpdatedEvent(tenant *Tenant, oldQuota, newQuota TenantQuota) *QuotaUpdatedEvent {
	return &QuotaUpdatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeQuotaUpdated, AggregateTypeTenant, tenant.ID, tenant.ID),
		TenantCode:      tenant.Code,
		OldQuota:        oldQuota,
		NewQuota:        newQuota,
	}
}

// TenantSuspendedEvent is published when a tenant is suspended
type TenantSuspendedEvent struct {
	shared.BaseDomainEvent
	TenantCode            string     `json:"tenant_code"`
	Reason                string     `json:"reason,omitempty"`
	ScheduledReactivateAt *time.Time `json:"scheduled_reactivate_at,omitempty"`
}

// NewTenantSuspendedEvent creates a new TenantSuspendedEvent
func NewTenantSuspendedEvent(tenant *Tenant, reason string, scheduledReactivateAt *time.Time) *TenantSuspendedEvent {
	return &TenantSuspendedEvent{
		BaseDomainEvent:       shared.NewBaseDomainEvent(EventTypeTenantSuspended, AggregateTypeTenant, tenant.ID, tenant.ID),
		TenantCode:            tenant.Code,
		Reason:                reason,
		ScheduledReactivateAt: scheduledReactivateAt,
	}
}

// TenantActivatedEvent is published when a suspended tenant is activated
type TenantActivatedEvent struct {
	shared.BaseDomainEvent
	TenantCode string `json:"tenant_code"`
}

// NewTenantActivatedEvent creates a new TenantActivatedEvent
func NewTenantActivatedEvent(tenant *Tenant) *TenantActivatedEvent {
	return &TenantActivatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeTenantActivated, AggregateTypeTenant, tenant.ID, tenant.ID),
		TenantCode:      tenant.Code,
	}
}
