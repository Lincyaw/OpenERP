package finance

import (
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// DebitMemoStatus represents the status of a debit memo
type DebitMemoStatus string

const (
	DebitMemoStatusPending  DebitMemoStatus = "PENDING"  // Created, waiting to be applied
	DebitMemoStatusApplied  DebitMemoStatus = "APPLIED"  // Applied to payables
	DebitMemoStatusPartial  DebitMemoStatus = "PARTIAL"  // Partially applied
	DebitMemoStatusVoided   DebitMemoStatus = "VOIDED"   // Voided/cancelled
	DebitMemoStatusRefunded DebitMemoStatus = "REFUNDED" // Refund received from supplier
)

// IsValid checks if the status is a valid DebitMemoStatus
func (s DebitMemoStatus) IsValid() bool {
	switch s {
	case DebitMemoStatusPending, DebitMemoStatusApplied, DebitMemoStatusPartial,
		DebitMemoStatusVoided, DebitMemoStatusRefunded:
		return true
	}
	return false
}

// String returns the string representation of DebitMemoStatus
func (s DebitMemoStatus) String() string {
	return string(s)
}

// IsTerminal returns true if the debit memo is in a terminal state
func (s DebitMemoStatus) IsTerminal() bool {
	return s == DebitMemoStatusApplied || s == DebitMemoStatusVoided || s == DebitMemoStatusRefunded
}

// CanApply returns true if the debit memo can be applied to payables
func (s DebitMemoStatus) CanApply() bool {
	return s == DebitMemoStatusPending || s == DebitMemoStatusPartial
}

// DebitMemoApplication represents an application of debit to a payable
type DebitMemoApplication struct {
	ID          uuid.UUID       `gorm:"type:uuid;primary_key"`
	DebitMemoID uuid.UUID       `gorm:"type:uuid;not null;index"`
	PayableID   uuid.UUID       `gorm:"type:uuid;not null;index"` // Reference to the payable
	Amount      decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	AppliedAt   time.Time       `gorm:"not null"`
	Remark      string          `gorm:"type:varchar(500)"`
}

// TableName returns the table name for GORM
func (DebitMemoApplication) TableName() string {
	return "debit_memo_applications"
}

// NewDebitMemoApplication creates a new debit memo application
func NewDebitMemoApplication(debitMemoID, payableID uuid.UUID, amount valueobject.Money, remark string) *DebitMemoApplication {
	return &DebitMemoApplication{
		ID:          uuid.New(),
		DebitMemoID: debitMemoID,
		PayableID:   payableID,
		Amount:      amount.Amount(),
		AppliedAt:   time.Now(),
		Remark:      remark,
	}
}

// GetAmountMoney returns the amount as Money value object
func (a *DebitMemoApplication) GetAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(a.Amount)
}

// DebitMemoItem represents a line item in the debit memo
type DebitMemoItem struct {
	ID               uuid.UUID       `gorm:"type:uuid;primary_key"`
	DebitMemoID      uuid.UUID       `gorm:"type:uuid;not null;index"`
	PurchaseReturnID uuid.UUID       `gorm:"type:uuid;not null"` // Reference to purchase return
	ReturnItemID     uuid.UUID       `gorm:"type:uuid;not null"` // Reference to return item
	ProductID        uuid.UUID       `gorm:"type:uuid;not null"`
	ProductName      string          `gorm:"type:varchar(200);not null"`
	ProductCode      string          `gorm:"type:varchar(50);not null"`
	ReturnQuantity   decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	UnitCost         decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	DebitAmount      decimal.Decimal `gorm:"type:decimal(18,4);not null"` // ReturnQuantity * UnitCost
	Unit             string          `gorm:"type:varchar(20);not null"`
	Reason           string          `gorm:"type:varchar(500)"`
	CreatedAt        time.Time       `gorm:"not null"`
}

// TableName returns the table name for GORM
func (DebitMemoItem) TableName() string {
	return "debit_memo_items"
}

// GetDebitAmountMoney returns the debit amount as Money value object
func (i *DebitMemoItem) GetDebitAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(i.DebitAmount)
}

// GetUnitCostMoney returns the unit cost as Money value object
func (i *DebitMemoItem) GetUnitCostMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(i.UnitCost)
}

// DebitMemo represents a debit memo aggregate root (红冲单 for purchase returns)
// It is created when a purchase return is completed to offset accounts payable
type DebitMemo struct {
	shared.TenantAggregateRoot
	MemoNumber             string                 `gorm:"type:varchar(50);not null;uniqueIndex:idx_debit_memo_tenant_number,priority:2"`
	PurchaseReturnID       uuid.UUID              `gorm:"type:uuid;not null;index"` // Reference to the purchase return
	PurchaseReturnNumber   string                 `gorm:"type:varchar(50);not null"`
	PurchaseOrderID        uuid.UUID              `gorm:"type:uuid;not null;index"` // Original purchase order
	PurchaseOrderNumber    string                 `gorm:"type:varchar(50);not null"`
	SupplierID             uuid.UUID              `gorm:"type:uuid;not null;index"`
	SupplierName           string                 `gorm:"type:varchar(200);not null"`
	OriginalPayableID      *uuid.UUID             `gorm:"type:uuid;index"` // Original AP being offset
	Items                  []DebitMemoItem        `gorm:"foreignKey:DebitMemoID;references:ID"`
	TotalDebit             decimal.Decimal        `gorm:"type:decimal(18,4);not null"` // Total debit amount
	AppliedAmount          decimal.Decimal        `gorm:"type:decimal(18,4);not null"` // Amount applied to payables
	RemainingAmount        decimal.Decimal        `gorm:"type:decimal(18,4);not null"` // Remaining debit to apply
	Status                 DebitMemoStatus        `gorm:"type:varchar(20);not null;default:'PENDING'"`
	Applications           []DebitMemoApplication `gorm:"foreignKey:DebitMemoID;references:ID"`
	Reason                 string                 `gorm:"type:text"` // Return reason
	Remark                 string                 `gorm:"type:text"`
	AppliedAt              *time.Time             // When fully applied
	VoidedAt               *time.Time
	VoidReason             string     `gorm:"type:varchar(500)"`
	RefundReceivedAt       *time.Time // When refund received from supplier
	RefundMethod           string     `gorm:"type:varchar(50)"` // How the refund was received
}

// TableName returns the table name for GORM
func (DebitMemo) TableName() string {
	return "debit_memos"
}

// NewDebitMemo creates a new debit memo from a purchase return
func NewDebitMemo(
	tenantID uuid.UUID,
	memoNumber string,
	purchaseReturnID uuid.UUID,
	purchaseReturnNumber string,
	purchaseOrderID uuid.UUID,
	purchaseOrderNumber string,
	supplierID uuid.UUID,
	supplierName string,
	totalDebit valueobject.Money,
	reason string,
) (*DebitMemo, error) {
	// Validate inputs
	if memoNumber == "" {
		return nil, shared.NewDomainError("INVALID_MEMO_NUMBER", "Memo number cannot be empty")
	}
	if len(memoNumber) > 50 {
		return nil, shared.NewDomainError("INVALID_MEMO_NUMBER", "Memo number cannot exceed 50 characters")
	}
	if purchaseReturnID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_PURCHASE_RETURN", "Purchase return ID cannot be empty")
	}
	if purchaseOrderID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_PURCHASE_ORDER", "Purchase order ID cannot be empty")
	}
	if supplierID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_SUPPLIER", "Supplier ID cannot be empty")
	}
	if supplierName == "" {
		return nil, shared.NewDomainError("INVALID_SUPPLIER_NAME", "Supplier name cannot be empty")
	}
	if totalDebit.Amount().LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("INVALID_AMOUNT", "Total debit must be positive")
	}

	dm := &DebitMemo{
		TenantAggregateRoot:  shared.NewTenantAggregateRoot(tenantID),
		MemoNumber:           memoNumber,
		PurchaseReturnID:     purchaseReturnID,
		PurchaseReturnNumber: purchaseReturnNumber,
		PurchaseOrderID:      purchaseOrderID,
		PurchaseOrderNumber:  purchaseOrderNumber,
		SupplierID:           supplierID,
		SupplierName:         supplierName,
		Items:                make([]DebitMemoItem, 0),
		TotalDebit:           totalDebit.Amount(),
		AppliedAmount:        decimal.Zero,
		RemainingAmount:      totalDebit.Amount(),
		Status:               DebitMemoStatusPending,
		Applications:         make([]DebitMemoApplication, 0),
		Reason:               reason,
	}

	dm.AddDomainEvent(NewDebitMemoCreatedEvent(dm))

	return dm, nil
}

// AddItem adds a line item to the debit memo
func (dm *DebitMemo) AddItem(
	purchaseReturnID uuid.UUID,
	returnItemID uuid.UUID,
	productID uuid.UUID,
	productName, productCode, unit string,
	returnQuantity decimal.Decimal,
	unitCost valueobject.Money,
	reason string,
) (*DebitMemoItem, error) {
	if dm.Status != DebitMemoStatusPending {
		return nil, shared.NewDomainError("INVALID_STATE", "Cannot add items to a non-pending debit memo")
	}

	debitAmount := returnQuantity.Mul(unitCost.Amount())

	item := &DebitMemoItem{
		ID:               uuid.New(),
		DebitMemoID:      dm.ID,
		PurchaseReturnID: purchaseReturnID,
		ReturnItemID:     returnItemID,
		ProductID:        productID,
		ProductName:      productName,
		ProductCode:      productCode,
		ReturnQuantity:   returnQuantity,
		UnitCost:         unitCost.Amount(),
		DebitAmount:      debitAmount,
		Unit:             unit,
		Reason:           reason,
		CreatedAt:        time.Now(),
	}

	dm.Items = append(dm.Items, *item)
	dm.UpdatedAt = time.Now()
	dm.IncrementVersion()

	return item, nil
}

// SetOriginalPayable links this debit memo to the original payable
func (dm *DebitMemo) SetOriginalPayable(payableID uuid.UUID) error {
	if payableID == uuid.Nil {
		return shared.NewDomainError("INVALID_PAYABLE", "Payable ID cannot be empty")
	}

	dm.OriginalPayableID = &payableID
	dm.UpdatedAt = time.Now()
	dm.IncrementVersion()

	return nil
}

// ApplyToPayable applies debit to a specific payable
// Returns error if amount exceeds remaining debit or debit memo is not applicable
func (dm *DebitMemo) ApplyToPayable(payableID uuid.UUID, amount valueobject.Money, remark string) error {
	if !dm.Status.CanApply() {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot apply debit memo in %s status", dm.Status))
	}
	if payableID == uuid.Nil {
		return shared.NewDomainError("INVALID_PAYABLE", "Payable ID cannot be empty")
	}
	if amount.Amount().LessThanOrEqual(decimal.Zero) {
		return shared.NewDomainError("INVALID_AMOUNT", "Application amount must be positive")
	}
	if amount.Amount().GreaterThan(dm.RemainingAmount) {
		return shared.NewDomainError("EXCEEDS_REMAINING", fmt.Sprintf("Application amount %.2f exceeds remaining debit %.2f", amount.Amount().InexactFloat64(), dm.RemainingAmount.InexactFloat64()))
	}

	// Create application record
	application := NewDebitMemoApplication(dm.ID, payableID, amount, remark)
	dm.Applications = append(dm.Applications, *application)

	// Update amounts
	dm.AppliedAmount = dm.AppliedAmount.Add(amount.Amount())
	dm.RemainingAmount = dm.TotalDebit.Sub(dm.AppliedAmount)

	// Update status
	if dm.RemainingAmount.IsZero() {
		now := time.Now()
		dm.Status = DebitMemoStatusApplied
		dm.AppliedAt = &now
		dm.AddDomainEvent(NewDebitMemoAppliedEvent(dm))
	} else {
		dm.Status = DebitMemoStatusPartial
		dm.AddDomainEvent(NewDebitMemoPartiallyAppliedEvent(dm, amount))
	}

	dm.UpdatedAt = time.Now()
	dm.IncrementVersion()

	return nil
}

// Void voids the debit memo
// Only allowed if not fully applied
func (dm *DebitMemo) Void(reason string) error {
	if dm.Status.IsTerminal() {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot void debit memo in %s status", dm.Status))
	}
	if dm.AppliedAmount.GreaterThan(decimal.Zero) {
		return shared.NewDomainError("HAS_APPLICATIONS", "Cannot void debit memo with existing applications")
	}
	if reason == "" {
		return shared.NewDomainError("INVALID_REASON", "Void reason is required")
	}

	now := time.Now()
	dm.Status = DebitMemoStatusVoided
	dm.VoidedAt = &now
	dm.VoidReason = reason
	dm.RemainingAmount = decimal.Zero
	dm.UpdatedAt = now
	dm.IncrementVersion()

	dm.AddDomainEvent(NewDebitMemoVoidedEvent(dm))

	return nil
}

// ReceiveRefund marks the debit memo as refund received
// This is for cases where the supplier sends refund instead of offsetting payables
func (dm *DebitMemo) ReceiveRefund(method string) error {
	if !dm.Status.CanApply() {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot receive refund for debit memo in %s status", dm.Status))
	}
	if dm.RemainingAmount.LessThanOrEqual(decimal.Zero) {
		return shared.NewDomainError("NO_REMAINING", "No remaining debit to refund")
	}
	if method == "" {
		return shared.NewDomainError("INVALID_METHOD", "Refund method is required")
	}

	refundAmount := dm.RemainingAmount

	now := time.Now()
	dm.Status = DebitMemoStatusRefunded
	dm.RefundReceivedAt = &now
	dm.RefundMethod = method
	dm.AppliedAmount = dm.TotalDebit // Consider full amount as handled
	dm.RemainingAmount = decimal.Zero
	dm.UpdatedAt = now
	dm.IncrementVersion()

	dm.AddDomainEvent(NewDebitMemoRefundReceivedEvent(dm, refundAmount))

	return nil
}

// SetRemark sets the remark
func (dm *DebitMemo) SetRemark(remark string) {
	dm.Remark = remark
	dm.UpdatedAt = time.Now()
	dm.IncrementVersion()
}

// Helper methods

// GetTotalDebitMoney returns total debit as Money
func (dm *DebitMemo) GetTotalDebitMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(dm.TotalDebit)
}

// GetAppliedAmountMoney returns applied amount as Money
func (dm *DebitMemo) GetAppliedAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(dm.AppliedAmount)
}

// GetRemainingAmountMoney returns remaining amount as Money
func (dm *DebitMemo) GetRemainingAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(dm.RemainingAmount)
}

// IsPending returns true if debit memo is pending
func (dm *DebitMemo) IsPending() bool {
	return dm.Status == DebitMemoStatusPending
}

// IsPartial returns true if debit memo is partially applied
func (dm *DebitMemo) IsPartial() bool {
	return dm.Status == DebitMemoStatusPartial
}

// IsApplied returns true if debit memo is fully applied
func (dm *DebitMemo) IsApplied() bool {
	return dm.Status == DebitMemoStatusApplied
}

// IsVoided returns true if debit memo is voided
func (dm *DebitMemo) IsVoided() bool {
	return dm.Status == DebitMemoStatusVoided
}

// IsRefunded returns true if debit memo is refunded
func (dm *DebitMemo) IsRefunded() bool {
	return dm.Status == DebitMemoStatusRefunded
}

// ItemCount returns the number of items
func (dm *DebitMemo) ItemCount() int {
	return len(dm.Items)
}

// ApplicationCount returns the number of applications
func (dm *DebitMemo) ApplicationCount() int {
	return len(dm.Applications)
}

// AppliedPercentage returns the percentage of debit that has been applied (0-100)
func (dm *DebitMemo) AppliedPercentage() decimal.Decimal {
	if dm.TotalDebit.IsZero() {
		return decimal.NewFromInt(100)
	}
	return dm.AppliedAmount.Div(dm.TotalDebit).Mul(decimal.NewFromInt(100)).Round(2)
}
