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

// =============================================================================
// Balance Transaction Events (per spec.md section 17.5)
// =============================================================================

// Event type constants for balance transactions
const (
	EventTypeCustomerBalanceTopUp    = "CustomerBalanceTopUp"
	EventTypeCustomerBalanceDeducted = "CustomerBalanceDeducted"
	EventTypeCustomerBalanceRefunded = "CustomerBalanceRefunded"
	EventTypeCustomerBalanceAdjusted = "CustomerBalanceAdjusted"
)

// PaymentMethod represents the method used for payment/recharge
type PaymentMethod string

const (
	PaymentMethodCash   PaymentMethod = "CASH"
	PaymentMethodWechat PaymentMethod = "WECHAT"
	PaymentMethodAlipay PaymentMethod = "ALIPAY"
	PaymentMethodBank   PaymentMethod = "BANK"
)

// IsValid returns true if the payment method is valid
func (p PaymentMethod) IsValid() bool {
	switch p {
	case PaymentMethodCash, PaymentMethodWechat, PaymentMethodAlipay, PaymentMethodBank:
		return true
	}
	return false
}

// String returns the string representation of PaymentMethod
func (p PaymentMethod) String() string {
	return string(p)
}

// CustomerBalanceTopUpEvent is published when a customer tops up their balance
// Subscribers: Finance (to record the receipt)
type CustomerBalanceTopUpEvent struct {
	shared.BaseDomainEvent
	CustomerID      uuid.UUID       `json:"customer_id"`
	Code            string          `json:"code"`
	Amount          decimal.Decimal `json:"amount"`
	BalanceBefore   decimal.Decimal `json:"balance_before"`
	BalanceAfter    decimal.Decimal `json:"balance_after"`
	PaymentMethod   PaymentMethod   `json:"payment_method"`
	PaymentRef      string          `json:"payment_ref,omitempty"`
	TransactionID   uuid.UUID       `json:"transaction_id"`
	TransactionDate string          `json:"transaction_date"`
}

// NewCustomerBalanceTopUpEvent creates a new CustomerBalanceTopUpEvent
func NewCustomerBalanceTopUpEvent(
	tenantID, customerID uuid.UUID,
	code string,
	amount, balanceBefore, balanceAfter decimal.Decimal,
	paymentMethod PaymentMethod,
	paymentRef string,
	transactionID uuid.UUID,
	transactionDate string,
) *CustomerBalanceTopUpEvent {
	return &CustomerBalanceTopUpEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeCustomerBalanceTopUp, AggregateTypeCustomer, customerID, tenantID),
		CustomerID:      customerID,
		Code:            code,
		Amount:          amount,
		BalanceBefore:   balanceBefore,
		BalanceAfter:    balanceAfter,
		PaymentMethod:   paymentMethod,
		PaymentRef:      paymentRef,
		TransactionID:   transactionID,
		TransactionDate: transactionDate,
	}
}

// CustomerBalanceDeductedEvent is published when balance is consumed (e.g., payment for order)
type CustomerBalanceDeductedEvent struct {
	shared.BaseDomainEvent
	CustomerID      uuid.UUID       `json:"customer_id"`
	Code            string          `json:"code"`
	Amount          decimal.Decimal `json:"amount"`
	BalanceBefore   decimal.Decimal `json:"balance_before"`
	BalanceAfter    decimal.Decimal `json:"balance_after"`
	OrderRef        string          `json:"order_ref,omitempty"`
	TransactionID   uuid.UUID       `json:"transaction_id"`
	TransactionDate string          `json:"transaction_date"`
}

// NewCustomerBalanceDeductedEvent creates a new CustomerBalanceDeductedEvent
func NewCustomerBalanceDeductedEvent(
	tenantID, customerID uuid.UUID,
	code string,
	amount, balanceBefore, balanceAfter decimal.Decimal,
	orderRef string,
	transactionID uuid.UUID,
	transactionDate string,
) *CustomerBalanceDeductedEvent {
	return &CustomerBalanceDeductedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeCustomerBalanceDeducted, AggregateTypeCustomer, customerID, tenantID),
		CustomerID:      customerID,
		Code:            code,
		Amount:          amount,
		BalanceBefore:   balanceBefore,
		BalanceAfter:    balanceAfter,
		OrderRef:        orderRef,
		TransactionID:   transactionID,
		TransactionDate: transactionDate,
	}
}

// CustomerBalanceRefundedEvent is published when balance is refunded (e.g., from return)
type CustomerBalanceRefundedEvent struct {
	shared.BaseDomainEvent
	CustomerID      uuid.UUID       `json:"customer_id"`
	Code            string          `json:"code"`
	Amount          decimal.Decimal `json:"amount"`
	BalanceBefore   decimal.Decimal `json:"balance_before"`
	BalanceAfter    decimal.Decimal `json:"balance_after"`
	ReturnRef       string          `json:"return_ref,omitempty"`
	TransactionID   uuid.UUID       `json:"transaction_id"`
	TransactionDate string          `json:"transaction_date"`
}

// NewCustomerBalanceRefundedEvent creates a new CustomerBalanceRefundedEvent
func NewCustomerBalanceRefundedEvent(
	tenantID, customerID uuid.UUID,
	code string,
	amount, balanceBefore, balanceAfter decimal.Decimal,
	returnRef string,
	transactionID uuid.UUID,
	transactionDate string,
) *CustomerBalanceRefundedEvent {
	return &CustomerBalanceRefundedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeCustomerBalanceRefunded, AggregateTypeCustomer, customerID, tenantID),
		CustomerID:      customerID,
		Code:            code,
		Amount:          amount,
		BalanceBefore:   balanceBefore,
		BalanceAfter:    balanceAfter,
		ReturnRef:       returnRef,
		TransactionID:   transactionID,
		TransactionDate: transactionDate,
	}
}

// CustomerBalanceAdjustedEvent is published when balance is manually adjusted
// Subscribers: Audit Log
type CustomerBalanceAdjustedEvent struct {
	shared.BaseDomainEvent
	CustomerID      uuid.UUID       `json:"customer_id"`
	Code            string          `json:"code"`
	Amount          decimal.Decimal `json:"amount"`
	IsIncrease      bool            `json:"is_increase"`
	BalanceBefore   decimal.Decimal `json:"balance_before"`
	BalanceAfter    decimal.Decimal `json:"balance_after"`
	Reason          string          `json:"reason"`
	TransactionID   uuid.UUID       `json:"transaction_id"`
	TransactionDate string          `json:"transaction_date"`
}

// NewCustomerBalanceAdjustedEvent creates a new CustomerBalanceAdjustedEvent
func NewCustomerBalanceAdjustedEvent(
	tenantID, customerID uuid.UUID,
	code string,
	amount decimal.Decimal,
	isIncrease bool,
	balanceBefore, balanceAfter decimal.Decimal,
	reason string,
	transactionID uuid.UUID,
	transactionDate string,
) *CustomerBalanceAdjustedEvent {
	return &CustomerBalanceAdjustedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeCustomerBalanceAdjusted, AggregateTypeCustomer, customerID, tenantID),
		CustomerID:      customerID,
		Code:            code,
		Amount:          amount,
		IsIncrease:      isIncrease,
		BalanceBefore:   balanceBefore,
		BalanceAfter:    balanceAfter,
		Reason:          reason,
		TransactionID:   transactionID,
		TransactionDate: transactionDate,
	}
}
