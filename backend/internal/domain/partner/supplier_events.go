package partner

import (
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Aggregate type constant for Supplier
const AggregateTypeSupplier = "Supplier"

// Event type constants for Supplier
const (
	EventTypeSupplierCreated             = "SupplierCreated"
	EventTypeSupplierUpdated             = "SupplierUpdated"
	EventTypeSupplierStatusChanged       = "SupplierStatusChanged"
	EventTypeSupplierPaymentTermsChanged = "SupplierPaymentTermsChanged"
	EventTypeSupplierBalanceChanged      = "SupplierBalanceChanged"
	EventTypeSupplierDeleted             = "SupplierDeleted"
)

// SupplierCreatedEvent is published when a new supplier is created
type SupplierCreatedEvent struct {
	shared.BaseDomainEvent
	SupplierID uuid.UUID    `json:"supplier_id"`
	Code       string       `json:"code"`
	Name       string       `json:"name"`
	Type       SupplierType `json:"type"`
}

// NewSupplierCreatedEvent creates a new SupplierCreatedEvent
func NewSupplierCreatedEvent(supplier *Supplier) *SupplierCreatedEvent {
	return &SupplierCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeSupplierCreated, AggregateTypeSupplier, supplier.ID, supplier.TenantID),
		SupplierID:      supplier.ID,
		Code:            supplier.Code,
		Name:            supplier.Name,
		Type:            supplier.Type,
	}
}

// SupplierUpdatedEvent is published when a supplier is updated
type SupplierUpdatedEvent struct {
	shared.BaseDomainEvent
	SupplierID  uuid.UUID `json:"supplier_id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	ShortName   string    `json:"short_name,omitempty"`
	ContactName string    `json:"contact_name,omitempty"`
	Phone       string    `json:"phone,omitempty"`
	Email       string    `json:"email,omitempty"`
}

// NewSupplierUpdatedEvent creates a new SupplierUpdatedEvent
func NewSupplierUpdatedEvent(supplier *Supplier) *SupplierUpdatedEvent {
	return &SupplierUpdatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeSupplierUpdated, AggregateTypeSupplier, supplier.ID, supplier.TenantID),
		SupplierID:      supplier.ID,
		Code:            supplier.Code,
		Name:            supplier.Name,
		ShortName:       supplier.ShortName,
		ContactName:     supplier.ContactName,
		Phone:           supplier.Phone,
		Email:           supplier.Email,
	}
}

// SupplierStatusChangedEvent is published when a supplier's status changes
type SupplierStatusChangedEvent struct {
	shared.BaseDomainEvent
	SupplierID uuid.UUID      `json:"supplier_id"`
	Code       string         `json:"code"`
	OldStatus  SupplierStatus `json:"old_status"`
	NewStatus  SupplierStatus `json:"new_status"`
}

// NewSupplierStatusChangedEvent creates a new SupplierStatusChangedEvent
func NewSupplierStatusChangedEvent(supplier *Supplier, oldStatus, newStatus SupplierStatus) *SupplierStatusChangedEvent {
	return &SupplierStatusChangedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeSupplierStatusChanged, AggregateTypeSupplier, supplier.ID, supplier.TenantID),
		SupplierID:      supplier.ID,
		Code:            supplier.Code,
		OldStatus:       oldStatus,
		NewStatus:       newStatus,
	}
}

// SupplierPaymentTermsChangedEvent is published when a supplier's payment terms change
type SupplierPaymentTermsChangedEvent struct {
	shared.BaseDomainEvent
	SupplierID     uuid.UUID       `json:"supplier_id"`
	Code           string          `json:"code"`
	OldCreditDays  int             `json:"old_credit_days"`
	NewCreditDays  int             `json:"new_credit_days"`
	OldCreditLimit decimal.Decimal `json:"old_credit_limit"`
	NewCreditLimit decimal.Decimal `json:"new_credit_limit"`
}

// NewSupplierPaymentTermsChangedEvent creates a new SupplierPaymentTermsChangedEvent
func NewSupplierPaymentTermsChangedEvent(supplier *Supplier, oldCreditDays, newCreditDays int, oldCreditLimit, newCreditLimit decimal.Decimal) *SupplierPaymentTermsChangedEvent {
	return &SupplierPaymentTermsChangedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeSupplierPaymentTermsChanged, AggregateTypeSupplier, supplier.ID, supplier.TenantID),
		SupplierID:      supplier.ID,
		Code:            supplier.Code,
		OldCreditDays:   oldCreditDays,
		NewCreditDays:   newCreditDays,
		OldCreditLimit:  oldCreditLimit,
		NewCreditLimit:  newCreditLimit,
	}
}

// SupplierBalanceChangedEvent is published when a supplier's balance changes
type SupplierBalanceChangedEvent struct {
	shared.BaseDomainEvent
	SupplierID uuid.UUID       `json:"supplier_id"`
	Code       string          `json:"code"`
	OldBalance decimal.Decimal `json:"old_balance"`
	NewBalance decimal.Decimal `json:"new_balance"`
	Reason     string          `json:"reason"` // "purchase", "payment", "adjustment"
}

// NewSupplierBalanceChangedEvent creates a new SupplierBalanceChangedEvent
func NewSupplierBalanceChangedEvent(supplier *Supplier, oldBalance, newBalance decimal.Decimal, reason string) *SupplierBalanceChangedEvent {
	return &SupplierBalanceChangedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeSupplierBalanceChanged, AggregateTypeSupplier, supplier.ID, supplier.TenantID),
		SupplierID:      supplier.ID,
		Code:            supplier.Code,
		OldBalance:      oldBalance,
		NewBalance:      newBalance,
		Reason:          reason,
	}
}

// SupplierDeletedEvent is published when a supplier is deleted
type SupplierDeletedEvent struct {
	shared.BaseDomainEvent
	SupplierID uuid.UUID `json:"supplier_id"`
	Code       string    `json:"code"`
	Name       string    `json:"name"`
}

// NewSupplierDeletedEvent creates a new SupplierDeletedEvent
func NewSupplierDeletedEvent(supplier *Supplier) *SupplierDeletedEvent {
	return &SupplierDeletedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeSupplierDeleted, AggregateTypeSupplier, supplier.ID, supplier.TenantID),
		SupplierID:      supplier.ID,
		Code:            supplier.Code,
		Name:            supplier.Name,
	}
}
