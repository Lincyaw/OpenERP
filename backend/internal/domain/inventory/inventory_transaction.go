package inventory

import (
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TransactionType represents the type of inventory transaction
type TransactionType string

const (
	// TransactionTypeInbound represents stock coming into inventory (purchase receiving, return)
	TransactionTypeInbound TransactionType = "INBOUND"
	// TransactionTypeOutbound represents stock leaving inventory (sales shipment)
	TransactionTypeOutbound TransactionType = "OUTBOUND"
	// TransactionTypeAdjustmentIncrease represents positive stock adjustment
	TransactionTypeAdjustmentIncrease TransactionType = "ADJUSTMENT_INCREASE"
	// TransactionTypeAdjustmentDecrease represents negative stock adjustment
	TransactionTypeAdjustmentDecrease TransactionType = "ADJUSTMENT_DECREASE"
	// TransactionTypeTransferIn represents stock transferred in from another warehouse
	TransactionTypeTransferIn TransactionType = "TRANSFER_IN"
	// TransactionTypeTransferOut represents stock transferred out to another warehouse
	TransactionTypeTransferOut TransactionType = "TRANSFER_OUT"
	// TransactionTypeReturn represents stock returned (sales return in, purchase return out)
	TransactionTypeReturn TransactionType = "RETURN"
	// TransactionTypeLock represents stock locked for pending orders
	TransactionTypeLock TransactionType = "LOCK"
	// TransactionTypeUnlock represents stock unlocked (order cancelled)
	TransactionTypeUnlock TransactionType = "UNLOCK"
)

// String returns the string representation of TransactionType
func (t TransactionType) String() string {
	return string(t)
}

// IsValid returns true if the transaction type is valid
func (t TransactionType) IsValid() bool {
	switch t {
	case TransactionTypeInbound,
		TransactionTypeOutbound,
		TransactionTypeAdjustmentIncrease,
		TransactionTypeAdjustmentDecrease,
		TransactionTypeTransferIn,
		TransactionTypeTransferOut,
		TransactionTypeReturn,
		TransactionTypeLock,
		TransactionTypeUnlock:
		return true
	}
	return false
}

// IsIncrease returns true if this transaction type increases available quantity
func (t TransactionType) IsIncrease() bool {
	switch t {
	case TransactionTypeInbound,
		TransactionTypeAdjustmentIncrease,
		TransactionTypeTransferIn,
		TransactionTypeReturn,
		TransactionTypeUnlock:
		return true
	}
	return false
}

// IsDecrease returns true if this transaction type decreases available quantity
func (t TransactionType) IsDecrease() bool {
	switch t {
	case TransactionTypeOutbound,
		TransactionTypeAdjustmentDecrease,
		TransactionTypeTransferOut,
		TransactionTypeLock:
		return true
	}
	return false
}

// SourceType represents the source document type for a transaction
type SourceType string

const (
	// SourceTypePurchaseOrder is a purchase order
	SourceTypePurchaseOrder SourceType = "PURCHASE_ORDER"
	// SourceTypeSalesOrder is a sales order
	SourceTypeSalesOrder SourceType = "SALES_ORDER"
	// SourceTypeSalesReturn is a sales return
	SourceTypeSalesReturn SourceType = "SALES_RETURN"
	// SourceTypePurchaseReturn is a purchase return
	SourceTypePurchaseReturn SourceType = "PURCHASE_RETURN"
	// SourceTypeStockTaking is a stock taking/count
	SourceTypeStockTaking SourceType = "STOCK_TAKING"
	// SourceTypeManualAdjustment is a manual adjustment
	SourceTypeManualAdjustment SourceType = "MANUAL_ADJUSTMENT"
	// SourceTypeTransfer is a warehouse transfer
	SourceTypeTransfer SourceType = "TRANSFER"
	// SourceTypeInitialStock is initial stock setup
	SourceTypeInitialStock SourceType = "INITIAL_STOCK"
)

// String returns the string representation of SourceType
func (s SourceType) String() string {
	return string(s)
}

// IsValid returns true if the source type is valid
func (s SourceType) IsValid() bool {
	switch s {
	case SourceTypePurchaseOrder,
		SourceTypeSalesOrder,
		SourceTypeSalesReturn,
		SourceTypePurchaseReturn,
		SourceTypeStockTaking,
		SourceTypeManualAdjustment,
		SourceTypeTransfer,
		SourceTypeInitialStock:
		return true
	}
	return false
}

// InventoryTransaction represents an immutable record of a stock movement.
// Once created, transactions cannot be modified - corrections must be made with new transactions.
type InventoryTransaction struct {
	shared.BaseEntity
	TenantID        uuid.UUID       `gorm:"type:uuid;not null;index:idx_inv_tx_tenant_time,priority:1"`
	InventoryItemID uuid.UUID       `gorm:"type:uuid;not null;index:idx_inv_tx_item"`
	WarehouseID     uuid.UUID       `gorm:"type:uuid;not null;index:idx_inv_tx_warehouse"`
	ProductID       uuid.UUID       `gorm:"type:uuid;not null;index:idx_inv_tx_product"`
	TransactionType TransactionType `gorm:"type:varchar(30);not null;index:idx_inv_tx_type"`
	Quantity        decimal.Decimal `gorm:"type:decimal(18,4);not null"`                                       // Always positive, direction determined by type
	UnitCost        decimal.Decimal `gorm:"type:decimal(18,4);not null"`                                       // Cost per unit at time of transaction
	TotalCost       decimal.Decimal `gorm:"type:decimal(18,4);not null"`                                       // Total cost (Quantity * UnitCost)
	BalanceBefore   decimal.Decimal `gorm:"type:decimal(18,4);not null"`                                       // Available quantity before transaction
	BalanceAfter    decimal.Decimal `gorm:"type:decimal(18,4);not null"`                                       // Available quantity after transaction
	SourceType      SourceType      `gorm:"type:varchar(30);not null;index:idx_inv_tx_source"`                 // Type of source document
	SourceID        string          `gorm:"type:varchar(50);not null;index:idx_inv_tx_source"`                 // ID of source document
	SourceLineID    string          `gorm:"type:varchar(50)"`                                                  // ID of source line item (optional)
	BatchID         *uuid.UUID      `gorm:"type:uuid;index"`                                                   // Related batch (optional)
	LockID          *uuid.UUID      `gorm:"type:uuid;index"`                                                   // Related lock (optional)
	Reference       string          `gorm:"type:varchar(100)"`                                                 // Reference number/code
	Reason          string          `gorm:"type:varchar(255)"`                                                 // Reason for transaction
	CostMethod      string          `gorm:"type:varchar(30)"`                                                  // Cost calculation method used (e.g., "moving_average", "fifo")
	OperatorID      *uuid.UUID      `gorm:"type:uuid"`                                                         // User who performed the operation
	TransactionDate time.Time       `gorm:"type:timestamptz;not null;index:idx_inv_tx_tenant_time,priority:2"` // When the transaction occurred
}

// TableName returns the table name for GORM
func (InventoryTransaction) TableName() string {
	return "inventory_transactions"
}

// NewInventoryTransaction creates a new inventory transaction
func NewInventoryTransaction(
	tenantID uuid.UUID,
	inventoryItemID uuid.UUID,
	warehouseID uuid.UUID,
	productID uuid.UUID,
	txType TransactionType,
	quantity decimal.Decimal,
	unitCost decimal.Decimal,
	balanceBefore decimal.Decimal,
	balanceAfter decimal.Decimal,
	sourceType SourceType,
	sourceID string,
) (*InventoryTransaction, error) {
	if tenantID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_TENANT", "Tenant ID cannot be empty")
	}
	if inventoryItemID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_INVENTORY_ITEM", "Inventory item ID cannot be empty")
	}
	if warehouseID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_WAREHOUSE", "Warehouse ID cannot be empty")
	}
	if productID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_PRODUCT", "Product ID cannot be empty")
	}
	if !txType.IsValid() {
		return nil, shared.NewDomainError("INVALID_TRANSACTION_TYPE", "Invalid transaction type")
	}
	if quantity.IsNegative() || quantity.IsZero() {
		return nil, shared.NewDomainError("INVALID_QUANTITY", "Quantity must be positive")
	}
	if unitCost.IsNegative() {
		return nil, shared.NewDomainError("INVALID_COST", "Unit cost cannot be negative")
	}
	if !sourceType.IsValid() {
		return nil, shared.NewDomainError("INVALID_SOURCE_TYPE", "Invalid source type")
	}
	if sourceID == "" {
		return nil, shared.NewDomainError("INVALID_SOURCE_ID", "Source ID cannot be empty")
	}

	totalCost := quantity.Mul(unitCost)

	tx := &InventoryTransaction{
		BaseEntity:      shared.NewBaseEntity(),
		TenantID:        tenantID,
		InventoryItemID: inventoryItemID,
		WarehouseID:     warehouseID,
		ProductID:       productID,
		TransactionType: txType,
		Quantity:        quantity,
		UnitCost:        unitCost,
		TotalCost:       totalCost,
		BalanceBefore:   balanceBefore,
		BalanceAfter:    balanceAfter,
		SourceType:      sourceType,
		SourceID:        sourceID,
		TransactionDate: time.Now(),
	}

	return tx, nil
}

// WithBatchID sets the batch ID for the transaction
func (t *InventoryTransaction) WithBatchID(batchID uuid.UUID) *InventoryTransaction {
	t.BatchID = &batchID
	return t
}

// WithLockID sets the lock ID for the transaction
func (t *InventoryTransaction) WithLockID(lockID uuid.UUID) *InventoryTransaction {
	t.LockID = &lockID
	return t
}

// WithSourceLineID sets the source line ID for the transaction
func (t *InventoryTransaction) WithSourceLineID(lineID string) *InventoryTransaction {
	t.SourceLineID = lineID
	return t
}

// WithReference sets the reference number for the transaction
func (t *InventoryTransaction) WithReference(reference string) *InventoryTransaction {
	t.Reference = reference
	return t
}

// WithReason sets the reason for the transaction
func (t *InventoryTransaction) WithReason(reason string) *InventoryTransaction {
	t.Reason = reason
	return t
}

// WithOperatorID sets the operator ID for the transaction
func (t *InventoryTransaction) WithOperatorID(operatorID uuid.UUID) *InventoryTransaction {
	t.OperatorID = &operatorID
	return t
}

// WithTransactionDate sets the transaction date
func (t *InventoryTransaction) WithTransactionDate(date time.Time) *InventoryTransaction {
	t.TransactionDate = date
	return t
}

// WithCostMethod sets the cost calculation method used for this transaction
func (t *InventoryTransaction) WithCostMethod(method string) *InventoryTransaction {
	t.CostMethod = method
	return t
}

// GetSignedQuantity returns the quantity with sign based on transaction type
// Positive for increases, negative for decreases
func (t *InventoryTransaction) GetSignedQuantity() decimal.Decimal {
	if t.TransactionType.IsDecrease() {
		return t.Quantity.Neg()
	}
	return t.Quantity
}

// GetSignedTotalCost returns the total cost with sign based on transaction type
func (t *InventoryTransaction) GetSignedTotalCost() decimal.Decimal {
	if t.TransactionType.IsDecrease() {
		return t.TotalCost.Neg()
	}
	return t.TotalCost
}

// IsInbound returns true if this is an inbound transaction
func (t *InventoryTransaction) IsInbound() bool {
	return t.TransactionType.IsIncrease()
}

// IsOutbound returns true if this is an outbound transaction
func (t *InventoryTransaction) IsOutbound() bool {
	return t.TransactionType.IsDecrease()
}

// QuantityChange returns the net quantity change
func (t *InventoryTransaction) QuantityChange() decimal.Decimal {
	return t.BalanceAfter.Sub(t.BalanceBefore)
}

// TransactionBuilder provides a fluent interface for building transactions
type TransactionBuilder struct {
	tx  *InventoryTransaction
	err error
}

// NewTransactionBuilder creates a new transaction builder
func NewTransactionBuilder(
	tenantID uuid.UUID,
	inventoryItemID uuid.UUID,
	warehouseID uuid.UUID,
	productID uuid.UUID,
	txType TransactionType,
	quantity decimal.Decimal,
	unitCost decimal.Decimal,
	balanceBefore decimal.Decimal,
	balanceAfter decimal.Decimal,
	sourceType SourceType,
	sourceID string,
) *TransactionBuilder {
	tx, err := NewInventoryTransaction(
		tenantID,
		inventoryItemID,
		warehouseID,
		productID,
		txType,
		quantity,
		unitCost,
		balanceBefore,
		balanceAfter,
		sourceType,
		sourceID,
	)
	return &TransactionBuilder{tx: tx, err: err}
}

// WithBatchID sets the batch ID
func (b *TransactionBuilder) WithBatchID(batchID uuid.UUID) *TransactionBuilder {
	if b.err != nil || b.tx == nil {
		return b
	}
	b.tx.WithBatchID(batchID)
	return b
}

// WithLockID sets the lock ID
func (b *TransactionBuilder) WithLockID(lockID uuid.UUID) *TransactionBuilder {
	if b.err != nil || b.tx == nil {
		return b
	}
	b.tx.WithLockID(lockID)
	return b
}

// WithSourceLineID sets the source line ID
func (b *TransactionBuilder) WithSourceLineID(lineID string) *TransactionBuilder {
	if b.err != nil || b.tx == nil {
		return b
	}
	b.tx.WithSourceLineID(lineID)
	return b
}

// WithReference sets the reference
func (b *TransactionBuilder) WithReference(reference string) *TransactionBuilder {
	if b.err != nil || b.tx == nil {
		return b
	}
	b.tx.WithReference(reference)
	return b
}

// WithReason sets the reason
func (b *TransactionBuilder) WithReason(reason string) *TransactionBuilder {
	if b.err != nil || b.tx == nil {
		return b
	}
	b.tx.WithReason(reason)
	return b
}

// WithOperatorID sets the operator ID
func (b *TransactionBuilder) WithOperatorID(operatorID uuid.UUID) *TransactionBuilder {
	if b.err != nil || b.tx == nil {
		return b
	}
	b.tx.WithOperatorID(operatorID)
	return b
}

// WithTransactionDate sets the transaction date
func (b *TransactionBuilder) WithTransactionDate(date time.Time) *TransactionBuilder {
	if b.err != nil || b.tx == nil {
		return b
	}
	b.tx.WithTransactionDate(date)
	return b
}

// WithCostMethod sets the cost calculation method
func (b *TransactionBuilder) WithCostMethod(method string) *TransactionBuilder {
	if b.err != nil || b.tx == nil {
		return b
	}
	b.tx.WithCostMethod(method)
	return b
}

// Build returns the built transaction or an error
func (b *TransactionBuilder) Build() (*InventoryTransaction, error) {
	if b.err != nil {
		return nil, b.err
	}
	return b.tx, nil
}

// CreateInboundTransaction is a helper to create an inbound transaction
func CreateInboundTransaction(
	tenantID, inventoryItemID, warehouseID, productID uuid.UUID,
	quantity, unitCost, balanceBefore, balanceAfter decimal.Decimal,
	sourceType SourceType,
	sourceID string,
) (*InventoryTransaction, error) {
	return NewInventoryTransaction(
		tenantID,
		inventoryItemID,
		warehouseID,
		productID,
		TransactionTypeInbound,
		quantity,
		unitCost,
		balanceBefore,
		balanceAfter,
		sourceType,
		sourceID,
	)
}

// CreateOutboundTransaction is a helper to create an outbound transaction
func CreateOutboundTransaction(
	tenantID, inventoryItemID, warehouseID, productID uuid.UUID,
	quantity, unitCost, balanceBefore, balanceAfter decimal.Decimal,
	sourceType SourceType,
	sourceID string,
) (*InventoryTransaction, error) {
	return NewInventoryTransaction(
		tenantID,
		inventoryItemID,
		warehouseID,
		productID,
		TransactionTypeOutbound,
		quantity,
		unitCost,
		balanceBefore,
		balanceAfter,
		sourceType,
		sourceID,
	)
}

// CreateAdjustmentTransaction is a helper to create an adjustment transaction
func CreateAdjustmentTransaction(
	tenantID, inventoryItemID, warehouseID, productID uuid.UUID,
	quantity, unitCost, balanceBefore, balanceAfter decimal.Decimal,
	sourceType SourceType,
	sourceID string,
	reason string,
) (*InventoryTransaction, error) {
	txType := TransactionTypeAdjustmentIncrease
	if balanceAfter.LessThan(balanceBefore) {
		txType = TransactionTypeAdjustmentDecrease
	}

	tx, err := NewInventoryTransaction(
		tenantID,
		inventoryItemID,
		warehouseID,
		productID,
		txType,
		quantity,
		unitCost,
		balanceBefore,
		balanceAfter,
		sourceType,
		sourceID,
	)
	if err != nil {
		return nil, err
	}

	tx.WithReason(reason)
	return tx, nil
}
