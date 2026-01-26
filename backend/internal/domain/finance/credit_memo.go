package finance

import (
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CreditMemoStatus represents the status of a credit memo
type CreditMemoStatus string

const (
	CreditMemoStatusPending  CreditMemoStatus = "PENDING"  // Created, waiting to be applied
	CreditMemoStatusApplied  CreditMemoStatus = "APPLIED"  // Applied to receivables
	CreditMemoStatusPartial  CreditMemoStatus = "PARTIAL"  // Partially applied
	CreditMemoStatusVoided   CreditMemoStatus = "VOIDED"   // Voided/cancelled
	CreditMemoStatusRefunded CreditMemoStatus = "REFUNDED" // Refunded to customer
)

// IsValid checks if the status is a valid CreditMemoStatus
func (s CreditMemoStatus) IsValid() bool {
	switch s {
	case CreditMemoStatusPending, CreditMemoStatusApplied, CreditMemoStatusPartial,
		CreditMemoStatusVoided, CreditMemoStatusRefunded:
		return true
	}
	return false
}

// String returns the string representation of CreditMemoStatus
func (s CreditMemoStatus) String() string {
	return string(s)
}

// IsTerminal returns true if the credit memo is in a terminal state
func (s CreditMemoStatus) IsTerminal() bool {
	return s == CreditMemoStatusApplied || s == CreditMemoStatusVoided || s == CreditMemoStatusRefunded
}

// CanApply returns true if the credit memo can be applied to receivables
func (s CreditMemoStatus) CanApply() bool {
	return s == CreditMemoStatusPending || s == CreditMemoStatusPartial
}

// CreditMemoApplication represents an application of credit to a receivable
type CreditMemoApplication struct {
	ID           uuid.UUID       `json:"id"`
	CreditMemoID uuid.UUID       `json:"credit_memo_id"`
	ReceivableID uuid.UUID       `json:"receivable_id"` // Reference to the receivable
	Amount       decimal.Decimal `json:"amount"`
	AppliedAt    time.Time       `json:"applied_at"`
	Remark       string          `json:"remark"`
}

// NewCreditMemoApplication creates a new credit memo application
func NewCreditMemoApplication(creditMemoID, receivableID uuid.UUID, amount valueobject.Money, remark string) *CreditMemoApplication {
	return &CreditMemoApplication{
		ID:           uuid.New(),
		CreditMemoID: creditMemoID,
		ReceivableID: receivableID,
		Amount:       amount.Amount(),
		AppliedAt:    time.Now(),
		Remark:       remark,
	}
}

// GetAmountMoney returns the amount as Money value object
func (a *CreditMemoApplication) GetAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(a.Amount)
}

// CreditMemoItem represents a line item in the credit memo
type CreditMemoItem struct {
	ID             uuid.UUID       `json:"id"`
	CreditMemoID   uuid.UUID       `json:"credit_memo_id"`
	SalesReturnID  uuid.UUID       `json:"sales_return_id"` // Reference to sales return
	ReturnItemID   uuid.UUID       `json:"return_item_id"`  // Reference to return item
	ProductID      uuid.UUID       `json:"product_id"`
	ProductName    string          `json:"product_name"`
	ProductCode    string          `json:"product_code"`
	ReturnQuantity decimal.Decimal `json:"return_quantity"`
	UnitPrice      decimal.Decimal `json:"unit_price"`
	CreditAmount   decimal.Decimal `json:"credit_amount"` // ReturnQuantity * UnitPrice
	Unit           string          `json:"unit"`
	Reason         string          `json:"reason"`
	CreatedAt      time.Time       `json:"created_at"`
}

// GetCreditAmountMoney returns the credit amount as Money value object
func (i *CreditMemoItem) GetCreditAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(i.CreditAmount)
}

// GetUnitPriceMoney returns the unit price as Money value object
func (i *CreditMemoItem) GetUnitPriceMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(i.UnitPrice)
}

// CreditMemo represents a credit memo aggregate root (红冲单)
// It is created when a sales return is completed to offset accounts receivable
type CreditMemo struct {
	shared.TenantAggregateRoot
	MemoNumber           string                  `json:"memo_number"`
	SalesReturnID        uuid.UUID               `json:"sales_return_id"` // Reference to the sales return
	SalesReturnNumber    string                  `json:"sales_return_number"`
	SalesOrderID         uuid.UUID               `json:"sales_order_id"` // Original sales order
	SalesOrderNumber     string                  `json:"sales_order_number"`
	CustomerID           uuid.UUID               `json:"customer_id"`
	CustomerName         string                  `json:"customer_name"`
	OriginalReceivableID *uuid.UUID              `json:"original_receivable_id"` // Original AR being offset
	Items                []CreditMemoItem        `json:"items"`
	TotalCredit          decimal.Decimal         `json:"total_credit"`     // Total credit amount
	AppliedAmount        decimal.Decimal         `json:"applied_amount"`   // Amount applied to receivables
	RemainingAmount      decimal.Decimal         `json:"remaining_amount"` // Remaining credit to apply
	Status               CreditMemoStatus        `json:"status"`
	Applications         []CreditMemoApplication `json:"applications"`
	Reason               string                  `json:"reason"` // Return reason
	Remark               string                  `json:"remark"`
	AppliedAt            *time.Time              `json:"applied_at"` // When fully applied
	VoidedAt             *time.Time              `json:"voided_at"`
	VoidReason           string                  `json:"void_reason"`
	RefundedAt           *time.Time              `json:"refunded_at"`   // When refunded to customer
	RefundMethod         string                  `json:"refund_method"` // How the refund was made
}

// NewCreditMemo creates a new credit memo from a sales return
func NewCreditMemo(
	tenantID uuid.UUID,
	memoNumber string,
	salesReturnID uuid.UUID,
	salesReturnNumber string,
	salesOrderID uuid.UUID,
	salesOrderNumber string,
	customerID uuid.UUID,
	customerName string,
	totalCredit valueobject.Money,
	reason string,
) (*CreditMemo, error) {
	// Validate inputs
	if memoNumber == "" {
		return nil, shared.NewDomainError("INVALID_MEMO_NUMBER", "Memo number cannot be empty")
	}
	if len(memoNumber) > 50 {
		return nil, shared.NewDomainError("INVALID_MEMO_NUMBER", "Memo number cannot exceed 50 characters")
	}
	if salesReturnID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_SALES_RETURN", "Sales return ID cannot be empty")
	}
	if salesOrderID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_SALES_ORDER", "Sales order ID cannot be empty")
	}
	if customerID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_CUSTOMER", "Customer ID cannot be empty")
	}
	if customerName == "" {
		return nil, shared.NewDomainError("INVALID_CUSTOMER_NAME", "Customer name cannot be empty")
	}
	if totalCredit.Amount().LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("INVALID_AMOUNT", "Total credit must be positive")
	}

	cm := &CreditMemo{
		TenantAggregateRoot: shared.NewTenantAggregateRoot(tenantID),
		MemoNumber:          memoNumber,
		SalesReturnID:       salesReturnID,
		SalesReturnNumber:   salesReturnNumber,
		SalesOrderID:        salesOrderID,
		SalesOrderNumber:    salesOrderNumber,
		CustomerID:          customerID,
		CustomerName:        customerName,
		Items:               make([]CreditMemoItem, 0),
		TotalCredit:         totalCredit.Amount(),
		AppliedAmount:       decimal.Zero,
		RemainingAmount:     totalCredit.Amount(),
		Status:              CreditMemoStatusPending,
		Applications:        make([]CreditMemoApplication, 0),
		Reason:              reason,
	}

	cm.AddDomainEvent(NewCreditMemoCreatedEvent(cm))

	return cm, nil
}

// AddItem adds a line item to the credit memo
func (cm *CreditMemo) AddItem(
	salesReturnID uuid.UUID,
	returnItemID uuid.UUID,
	productID uuid.UUID,
	productName, productCode, unit string,
	returnQuantity decimal.Decimal,
	unitPrice valueobject.Money,
	reason string,
) (*CreditMemoItem, error) {
	if cm.Status != CreditMemoStatusPending {
		return nil, shared.NewDomainError("INVALID_STATE", "Cannot add items to a non-pending credit memo")
	}

	creditAmount := returnQuantity.Mul(unitPrice.Amount())

	item := &CreditMemoItem{
		ID:             uuid.New(),
		CreditMemoID:   cm.ID,
		SalesReturnID:  salesReturnID,
		ReturnItemID:   returnItemID,
		ProductID:      productID,
		ProductName:    productName,
		ProductCode:    productCode,
		ReturnQuantity: returnQuantity,
		UnitPrice:      unitPrice.Amount(),
		CreditAmount:   creditAmount,
		Unit:           unit,
		Reason:         reason,
		CreatedAt:      time.Now(),
	}

	cm.Items = append(cm.Items, *item)
	cm.UpdatedAt = time.Now()
	cm.IncrementVersion()

	return item, nil
}

// SetOriginalReceivable links this credit memo to the original receivable
func (cm *CreditMemo) SetOriginalReceivable(receivableID uuid.UUID) error {
	if receivableID == uuid.Nil {
		return shared.NewDomainError("INVALID_RECEIVABLE", "Receivable ID cannot be empty")
	}

	cm.OriginalReceivableID = &receivableID
	cm.UpdatedAt = time.Now()
	cm.IncrementVersion()

	return nil
}

// ApplyToReceivable applies credit to a specific receivable
// Returns error if amount exceeds remaining credit or credit memo is not applicable
func (cm *CreditMemo) ApplyToReceivable(receivableID uuid.UUID, amount valueobject.Money, remark string) error {
	if !cm.Status.CanApply() {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot apply credit memo in %s status", cm.Status))
	}
	if receivableID == uuid.Nil {
		return shared.NewDomainError("INVALID_RECEIVABLE", "Receivable ID cannot be empty")
	}
	if amount.Amount().LessThanOrEqual(decimal.Zero) {
		return shared.NewDomainError("INVALID_AMOUNT", "Application amount must be positive")
	}
	if amount.Amount().GreaterThan(cm.RemainingAmount) {
		return shared.NewDomainError("EXCEEDS_REMAINING", fmt.Sprintf("Application amount %.2f exceeds remaining credit %.2f", amount.Amount().InexactFloat64(), cm.RemainingAmount.InexactFloat64()))
	}

	// Create application record
	application := NewCreditMemoApplication(cm.ID, receivableID, amount, remark)
	cm.Applications = append(cm.Applications, *application)

	// Update amounts
	cm.AppliedAmount = cm.AppliedAmount.Add(amount.Amount())
	cm.RemainingAmount = cm.TotalCredit.Sub(cm.AppliedAmount)

	// Update status
	if cm.RemainingAmount.IsZero() {
		now := time.Now()
		cm.Status = CreditMemoStatusApplied
		cm.AppliedAt = &now
		cm.AddDomainEvent(NewCreditMemoAppliedEvent(cm))
	} else {
		cm.Status = CreditMemoStatusPartial
		cm.AddDomainEvent(NewCreditMemoPartiallyAppliedEvent(cm, amount))
	}

	cm.UpdatedAt = time.Now()
	cm.IncrementVersion()

	return nil
}

// Void voids the credit memo
// Only allowed if not fully applied
func (cm *CreditMemo) Void(reason string) error {
	if cm.Status.IsTerminal() {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot void credit memo in %s status", cm.Status))
	}
	if cm.AppliedAmount.GreaterThan(decimal.Zero) {
		return shared.NewDomainError("HAS_APPLICATIONS", "Cannot void credit memo with existing applications")
	}
	if reason == "" {
		return shared.NewDomainError("INVALID_REASON", "Void reason is required")
	}

	now := time.Now()
	cm.Status = CreditMemoStatusVoided
	cm.VoidedAt = &now
	cm.VoidReason = reason
	cm.RemainingAmount = decimal.Zero
	cm.UpdatedAt = now
	cm.IncrementVersion()

	cm.AddDomainEvent(NewCreditMemoVoidedEvent(cm))

	return nil
}

// Refund marks the credit memo as refunded
// This is for cases where the remaining credit is refunded to customer instead of applied
func (cm *CreditMemo) Refund(method string) error {
	if !cm.Status.CanApply() {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot refund credit memo in %s status", cm.Status))
	}
	if cm.RemainingAmount.LessThanOrEqual(decimal.Zero) {
		return shared.NewDomainError("NO_REMAINING", "No remaining credit to refund")
	}
	if method == "" {
		return shared.NewDomainError("INVALID_METHOD", "Refund method is required")
	}

	refundAmount := cm.RemainingAmount

	now := time.Now()
	cm.Status = CreditMemoStatusRefunded
	cm.RefundedAt = &now
	cm.RefundMethod = method
	cm.AppliedAmount = cm.TotalCredit // Consider full amount as handled
	cm.RemainingAmount = decimal.Zero
	cm.UpdatedAt = now
	cm.IncrementVersion()

	cm.AddDomainEvent(NewCreditMemoRefundedEvent(cm, refundAmount))

	return nil
}

// SetRemark sets the remark
func (cm *CreditMemo) SetRemark(remark string) {
	cm.Remark = remark
	cm.UpdatedAt = time.Now()
	cm.IncrementVersion()
}

// Helper methods

// GetTotalCreditMoney returns total credit as Money
func (cm *CreditMemo) GetTotalCreditMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(cm.TotalCredit)
}

// GetAppliedAmountMoney returns applied amount as Money
func (cm *CreditMemo) GetAppliedAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(cm.AppliedAmount)
}

// GetRemainingAmountMoney returns remaining amount as Money
func (cm *CreditMemo) GetRemainingAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(cm.RemainingAmount)
}

// IsPending returns true if credit memo is pending
func (cm *CreditMemo) IsPending() bool {
	return cm.Status == CreditMemoStatusPending
}

// IsPartial returns true if credit memo is partially applied
func (cm *CreditMemo) IsPartial() bool {
	return cm.Status == CreditMemoStatusPartial
}

// IsApplied returns true if credit memo is fully applied
func (cm *CreditMemo) IsApplied() bool {
	return cm.Status == CreditMemoStatusApplied
}

// IsVoided returns true if credit memo is voided
func (cm *CreditMemo) IsVoided() bool {
	return cm.Status == CreditMemoStatusVoided
}

// IsRefunded returns true if credit memo is refunded
func (cm *CreditMemo) IsRefunded() bool {
	return cm.Status == CreditMemoStatusRefunded
}

// ItemCount returns the number of items
func (cm *CreditMemo) ItemCount() int {
	return len(cm.Items)
}

// ApplicationCount returns the number of applications
func (cm *CreditMemo) ApplicationCount() int {
	return len(cm.Applications)
}

// AppliedPercentage returns the percentage of credit that has been applied (0-100)
func (cm *CreditMemo) AppliedPercentage() decimal.Decimal {
	if cm.TotalCredit.IsZero() {
		return decimal.NewFromInt(100)
	}
	return cm.AppliedAmount.Div(cm.TotalCredit).Mul(decimal.NewFromInt(100)).Round(2)
}
