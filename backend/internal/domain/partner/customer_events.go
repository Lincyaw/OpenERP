package partner

import (
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Aggregate type constant
const AggregateTypeCustomer = "Customer"

// Event type constants
const (
	EventTypeCustomerCreated        = "CustomerCreated"
	EventTypeCustomerUpdated        = "CustomerUpdated"
	EventTypeCustomerStatusChanged  = "CustomerStatusChanged"
	EventTypeCustomerLevelChanged   = "CustomerLevelChanged"
	EventTypeCustomerBalanceChanged = "CustomerBalanceChanged"
	EventTypeCustomerDeleted        = "CustomerDeleted"
)

// CustomerCreatedEvent is published when a new customer is created
type CustomerCreatedEvent struct {
	shared.BaseDomainEvent
	CustomerID uuid.UUID    `json:"customer_id"`
	Code       string       `json:"code"`
	Name       string       `json:"name"`
	Type       CustomerType `json:"type"`
}

// NewCustomerCreatedEvent creates a new CustomerCreatedEvent
func NewCustomerCreatedEvent(customer *Customer) *CustomerCreatedEvent {
	return &CustomerCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeCustomerCreated, AggregateTypeCustomer, customer.ID, customer.TenantID),
		CustomerID:      customer.ID,
		Code:            customer.Code,
		Name:            customer.Name,
		Type:            customer.Type,
	}
}

// CustomerUpdatedEvent is published when a customer is updated
type CustomerUpdatedEvent struct {
	shared.BaseDomainEvent
	CustomerID  uuid.UUID `json:"customer_id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	ShortName   string    `json:"short_name,omitempty"`
	ContactName string    `json:"contact_name,omitempty"`
	Phone       string    `json:"phone,omitempty"`
	Email       string    `json:"email,omitempty"`
}

// NewCustomerUpdatedEvent creates a new CustomerUpdatedEvent
func NewCustomerUpdatedEvent(customer *Customer) *CustomerUpdatedEvent {
	return &CustomerUpdatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeCustomerUpdated, AggregateTypeCustomer, customer.ID, customer.TenantID),
		CustomerID:      customer.ID,
		Code:            customer.Code,
		Name:            customer.Name,
		ShortName:       customer.ShortName,
		ContactName:     customer.ContactName,
		Phone:           customer.Phone,
		Email:           customer.Email,
	}
}

// CustomerStatusChangedEvent is published when a customer's status changes
type CustomerStatusChangedEvent struct {
	shared.BaseDomainEvent
	CustomerID uuid.UUID      `json:"customer_id"`
	Code       string         `json:"code"`
	OldStatus  CustomerStatus `json:"old_status"`
	NewStatus  CustomerStatus `json:"new_status"`
}

// NewCustomerStatusChangedEvent creates a new CustomerStatusChangedEvent
func NewCustomerStatusChangedEvent(customer *Customer, oldStatus, newStatus CustomerStatus) *CustomerStatusChangedEvent {
	return &CustomerStatusChangedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeCustomerStatusChanged, AggregateTypeCustomer, customer.ID, customer.TenantID),
		CustomerID:      customer.ID,
		Code:            customer.Code,
		OldStatus:       oldStatus,
		NewStatus:       newStatus,
	}
}

// CustomerLevelChangedEvent is published when a customer's level/tier changes
type CustomerLevelChangedEvent struct {
	shared.BaseDomainEvent
	CustomerID uuid.UUID     `json:"customer_id"`
	Code       string        `json:"code"`
	OldLevel   CustomerLevel `json:"old_level"`
	NewLevel   CustomerLevel `json:"new_level"`
}

// NewCustomerLevelChangedEvent creates a new CustomerLevelChangedEvent
func NewCustomerLevelChangedEvent(customer *Customer, oldLevel, newLevel CustomerLevel) *CustomerLevelChangedEvent {
	return &CustomerLevelChangedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeCustomerLevelChanged, AggregateTypeCustomer, customer.ID, customer.TenantID),
		CustomerID:      customer.ID,
		Code:            customer.Code,
		OldLevel:        oldLevel,
		NewLevel:        newLevel,
	}
}

// CustomerBalanceChangedEvent is published when a customer's balance changes
type CustomerBalanceChangedEvent struct {
	shared.BaseDomainEvent
	CustomerID uuid.UUID       `json:"customer_id"`
	Code       string          `json:"code"`
	OldBalance decimal.Decimal `json:"old_balance"`
	NewBalance decimal.Decimal `json:"new_balance"`
	Reason     string          `json:"reason"` // "recharge", "deduction", "refund"
}

// NewCustomerBalanceChangedEvent creates a new CustomerBalanceChangedEvent
func NewCustomerBalanceChangedEvent(customer *Customer, oldBalance, newBalance decimal.Decimal, reason string) *CustomerBalanceChangedEvent {
	return &CustomerBalanceChangedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeCustomerBalanceChanged, AggregateTypeCustomer, customer.ID, customer.TenantID),
		CustomerID:      customer.ID,
		Code:            customer.Code,
		OldBalance:      oldBalance,
		NewBalance:      newBalance,
		Reason:          reason,
	}
}

// CustomerDeletedEvent is published when a customer is deleted
type CustomerDeletedEvent struct {
	shared.BaseDomainEvent
	CustomerID uuid.UUID `json:"customer_id"`
	Code       string    `json:"code"`
	Name       string    `json:"name"`
}

// NewCustomerDeletedEvent creates a new CustomerDeletedEvent
func NewCustomerDeletedEvent(customer *Customer) *CustomerDeletedEvent {
	return &CustomerDeletedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeCustomerDeleted, AggregateTypeCustomer, customer.ID, customer.TenantID),
		CustomerID:      customer.ID,
		Code:            customer.Code,
		Name:            customer.Name,
	}
}
