package partner

import (
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// BalanceTransactionType represents the type of balance transaction
type BalanceTransactionType string

const (
	// BalanceTransactionTypeRecharge represents customer depositing money (balance increase)
	BalanceTransactionTypeRecharge BalanceTransactionType = "RECHARGE"
	// BalanceTransactionTypeConsume represents customer paying with balance (balance decrease)
	BalanceTransactionTypeConsume BalanceTransactionType = "CONSUME"
	// BalanceTransactionTypeRefund represents refund to customer balance (balance increase)
	BalanceTransactionTypeRefund BalanceTransactionType = "REFUND"
	// BalanceTransactionTypeAdjustment represents manual adjustment (increase or decrease)
	BalanceTransactionTypeAdjustment BalanceTransactionType = "ADJUSTMENT"
	// BalanceTransactionTypeExpire represents balance expiration (balance decrease)
	BalanceTransactionTypeExpire BalanceTransactionType = "EXPIRE"
)

// String returns the string representation of BalanceTransactionType
func (t BalanceTransactionType) String() string {
	return string(t)
}

// IsValid returns true if the transaction type is valid
func (t BalanceTransactionType) IsValid() bool {
	switch t {
	case BalanceTransactionTypeRecharge,
		BalanceTransactionTypeConsume,
		BalanceTransactionTypeRefund,
		BalanceTransactionTypeAdjustment,
		BalanceTransactionTypeExpire:
		return true
	}
	return false
}

// IsIncrease returns true if this transaction type typically increases balance
func (t BalanceTransactionType) IsIncrease() bool {
	switch t {
	case BalanceTransactionTypeRecharge, BalanceTransactionTypeRefund:
		return true
	}
	return false
}

// IsDecrease returns true if this transaction type typically decreases balance
func (t BalanceTransactionType) IsDecrease() bool {
	switch t {
	case BalanceTransactionTypeConsume, BalanceTransactionTypeExpire:
		return true
	}
	return false
}

// BalanceTransactionSourceType represents the source document type for a balance transaction
type BalanceTransactionSourceType string

const (
	// BalanceSourceTypeManual represents manual recharge/adjustment
	BalanceSourceTypeManual BalanceTransactionSourceType = "MANUAL"
	// BalanceSourceTypeSalesOrder represents consumption from sales order
	BalanceSourceTypeSalesOrder BalanceTransactionSourceType = "SALES_ORDER"
	// BalanceSourceTypeSalesReturn represents refund from sales return
	BalanceSourceTypeSalesReturn BalanceTransactionSourceType = "SALES_RETURN"
	// BalanceSourceTypeReceiptVoucher represents recharge from receipt voucher
	BalanceSourceTypeReceiptVoucher BalanceTransactionSourceType = "RECEIPT_VOUCHER"
	// BalanceSourceTypeSystem represents system-initiated transaction (e.g., expiration)
	BalanceSourceTypeSystem BalanceTransactionSourceType = "SYSTEM"
)

// String returns the string representation of BalanceTransactionSourceType
func (s BalanceTransactionSourceType) String() string {
	return string(s)
}

// IsValid returns true if the source type is valid
func (s BalanceTransactionSourceType) IsValid() bool {
	switch s {
	case BalanceSourceTypeManual,
		BalanceSourceTypeSalesOrder,
		BalanceSourceTypeSalesReturn,
		BalanceSourceTypeReceiptVoucher,
		BalanceSourceTypeSystem:
		return true
	}
	return false
}

// BalanceTransaction represents an immutable record of a customer balance change.
// Once created, transactions cannot be modified - corrections must be made with new transactions.
type BalanceTransaction struct {
	shared.BaseEntity
	TenantID        uuid.UUID // Always positive, direction determined by type
	CustomerID      uuid.UUID
	TransactionType BalanceTransactionType
	Amount          decimal.Decimal // Always positive, direction determined by type
	BalanceBefore   decimal.Decimal // Balance before transaction
	BalanceAfter    decimal.Decimal // Balance after transaction
	SourceType      BalanceTransactionSourceType
	SourceID        *string    // ID of source document (optional)
	Reference       string     // Reference number/code
	Remark          string     // Remark/notes
	OperatorID      *uuid.UUID // User who performed the operation
	TransactionDate time.Time
}

// NewBalanceTransaction creates a new balance transaction
func NewBalanceTransaction(
	tenantID uuid.UUID,
	customerID uuid.UUID,
	txType BalanceTransactionType,
	amount decimal.Decimal,
	balanceBefore decimal.Decimal,
	balanceAfter decimal.Decimal,
	sourceType BalanceTransactionSourceType,
) (*BalanceTransaction, error) {
	if tenantID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_TENANT", "Tenant ID cannot be empty")
	}
	if customerID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_CUSTOMER", "Customer ID cannot be empty")
	}
	if !txType.IsValid() {
		return nil, shared.NewDomainError("INVALID_TRANSACTION_TYPE", "Invalid balance transaction type")
	}
	if amount.IsNegative() || amount.IsZero() {
		return nil, shared.NewDomainError("INVALID_AMOUNT", "Amount must be positive")
	}
	if balanceBefore.IsNegative() {
		return nil, shared.NewDomainError("INVALID_BALANCE", "Balance before cannot be negative")
	}
	if balanceAfter.IsNegative() {
		return nil, shared.NewDomainError("INVALID_BALANCE", "Balance after cannot be negative")
	}
	if !sourceType.IsValid() {
		return nil, shared.NewDomainError("INVALID_SOURCE_TYPE", "Invalid source type")
	}

	tx := &BalanceTransaction{
		BaseEntity:      shared.NewBaseEntity(),
		TenantID:        tenantID,
		CustomerID:      customerID,
		TransactionType: txType,
		Amount:          amount,
		BalanceBefore:   balanceBefore,
		BalanceAfter:    balanceAfter,
		SourceType:      sourceType,
		TransactionDate: time.Now(),
	}

	return tx, nil
}

// WithSourceID sets the source document ID for the transaction
func (t *BalanceTransaction) WithSourceID(sourceID string) *BalanceTransaction {
	t.SourceID = &sourceID
	return t
}

// WithReference sets the reference number for the transaction
func (t *BalanceTransaction) WithReference(reference string) *BalanceTransaction {
	t.Reference = reference
	return t
}

// WithRemark sets the remark for the transaction
func (t *BalanceTransaction) WithRemark(remark string) *BalanceTransaction {
	t.Remark = remark
	return t
}

// WithOperatorID sets the operator ID for the transaction
func (t *BalanceTransaction) WithOperatorID(operatorID uuid.UUID) *BalanceTransaction {
	t.OperatorID = &operatorID
	return t
}

// WithTransactionDate sets the transaction date
func (t *BalanceTransaction) WithTransactionDate(date time.Time) *BalanceTransaction {
	t.TransactionDate = date
	return t
}

// GetSignedAmount returns the amount with sign based on transaction type
// Positive for increases (recharge, refund), negative for decreases (consume, expire)
func (t *BalanceTransaction) GetSignedAmount() decimal.Decimal {
	if t.TransactionType.IsDecrease() {
		return t.Amount.Neg()
	}
	// For adjustment, calculate from balance difference
	if t.TransactionType == BalanceTransactionTypeAdjustment {
		return t.BalanceAfter.Sub(t.BalanceBefore)
	}
	return t.Amount
}

// IsIncrease returns true if this transaction increased balance
func (t *BalanceTransaction) IsIncrease() bool {
	return t.BalanceAfter.GreaterThan(t.BalanceBefore)
}

// IsDecrease returns true if this transaction decreased balance
func (t *BalanceTransaction) IsDecrease() bool {
	return t.BalanceAfter.LessThan(t.BalanceBefore)
}

// BalanceChange returns the net balance change
func (t *BalanceTransaction) BalanceChange() decimal.Decimal {
	return t.BalanceAfter.Sub(t.BalanceBefore)
}

// CreateRechargeTransaction creates a recharge transaction
func CreateRechargeTransaction(
	tenantID, customerID uuid.UUID,
	amount, balanceBefore decimal.Decimal,
	sourceType BalanceTransactionSourceType,
) (*BalanceTransaction, error) {
	balanceAfter := balanceBefore.Add(amount)
	return NewBalanceTransaction(
		tenantID,
		customerID,
		BalanceTransactionTypeRecharge,
		amount,
		balanceBefore,
		balanceAfter,
		sourceType,
	)
}

// CreateConsumeTransaction creates a consume transaction
func CreateConsumeTransaction(
	tenantID, customerID uuid.UUID,
	amount, balanceBefore decimal.Decimal,
	sourceType BalanceTransactionSourceType,
) (*BalanceTransaction, error) {
	if balanceBefore.LessThan(amount) {
		return nil, shared.NewDomainError("INSUFFICIENT_BALANCE", "Insufficient balance for consumption")
	}
	balanceAfter := balanceBefore.Sub(amount)
	return NewBalanceTransaction(
		tenantID,
		customerID,
		BalanceTransactionTypeConsume,
		amount,
		balanceBefore,
		balanceAfter,
		sourceType,
	)
}

// CreateRefundTransaction creates a refund transaction
func CreateRefundTransaction(
	tenantID, customerID uuid.UUID,
	amount, balanceBefore decimal.Decimal,
	sourceType BalanceTransactionSourceType,
) (*BalanceTransaction, error) {
	balanceAfter := balanceBefore.Add(amount)
	return NewBalanceTransaction(
		tenantID,
		customerID,
		BalanceTransactionTypeRefund,
		amount,
		balanceBefore,
		balanceAfter,
		sourceType,
	)
}

// CreateAdjustmentTransaction creates an adjustment transaction
func CreateAdjustmentTransaction(
	tenantID, customerID uuid.UUID,
	amount decimal.Decimal,
	isIncrease bool,
	balanceBefore decimal.Decimal,
) (*BalanceTransaction, error) {
	var balanceAfter decimal.Decimal
	if isIncrease {
		balanceAfter = balanceBefore.Add(amount)
	} else {
		if balanceBefore.LessThan(amount) {
			return nil, shared.NewDomainError("INSUFFICIENT_BALANCE", "Insufficient balance for adjustment")
		}
		balanceAfter = balanceBefore.Sub(amount)
	}
	return NewBalanceTransaction(
		tenantID,
		customerID,
		BalanceTransactionTypeAdjustment,
		amount,
		balanceBefore,
		balanceAfter,
		BalanceSourceTypeManual,
	)
}
