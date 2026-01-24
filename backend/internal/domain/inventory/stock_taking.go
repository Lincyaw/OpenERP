package inventory

import (
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// StockTakingStatus represents the status of a stock taking document
type StockTakingStatus string

const (
	StockTakingStatusDraft           StockTakingStatus = "DRAFT"
	StockTakingStatusCounting        StockTakingStatus = "COUNTING"
	StockTakingStatusPendingApproval StockTakingStatus = "PENDING_APPROVAL"
	StockTakingStatusApproved        StockTakingStatus = "APPROVED"
	StockTakingStatusRejected        StockTakingStatus = "REJECTED"
	StockTakingStatusCancelled       StockTakingStatus = "CANCELLED"
)

// IsValid checks if the status is a valid StockTakingStatus
func (s StockTakingStatus) IsValid() bool {
	switch s {
	case StockTakingStatusDraft, StockTakingStatusCounting, StockTakingStatusPendingApproval,
		StockTakingStatusApproved, StockTakingStatusRejected, StockTakingStatusCancelled:
		return true
	}
	return false
}

// String returns the string representation of StockTakingStatus
func (s StockTakingStatus) String() string {
	return string(s)
}

// CanTransitionTo checks if the status can transition to the target status
func (s StockTakingStatus) CanTransitionTo(target StockTakingStatus) bool {
	switch s {
	case StockTakingStatusDraft:
		return target == StockTakingStatusCounting || target == StockTakingStatusCancelled
	case StockTakingStatusCounting:
		return target == StockTakingStatusPendingApproval || target == StockTakingStatusCancelled
	case StockTakingStatusPendingApproval:
		return target == StockTakingStatusApproved || target == StockTakingStatusRejected
	case StockTakingStatusApproved, StockTakingStatusRejected, StockTakingStatusCancelled:
		return false // Terminal states
	}
	return false
}

// StockTakingItem represents a line item in a stock taking document
type StockTakingItem struct {
	ID               uuid.UUID       `gorm:"type:uuid;primary_key"`
	StockTakingID    uuid.UUID       `gorm:"type:uuid;not null;index"`
	ProductID        uuid.UUID       `gorm:"type:uuid;not null"`
	ProductName      string          `gorm:"type:varchar(200);not null"`
	ProductCode      string          `gorm:"type:varchar(50);not null"`
	Unit             string          `gorm:"type:varchar(20);not null"`
	SystemQuantity   decimal.Decimal `gorm:"type:decimal(18,4);not null"` // Quantity in system
	ActualQuantity   decimal.Decimal `gorm:"type:decimal(18,4)"`          // Quantity from physical count (nullable until counted)
	DifferenceQty    decimal.Decimal `gorm:"type:decimal(18,4)"`          // Actual - System
	UnitCost         decimal.Decimal `gorm:"type:decimal(18,4);not null"` // Cost per unit at count time
	DifferenceAmount decimal.Decimal `gorm:"type:decimal(18,4)"`          // Difference * UnitCost
	Counted          bool            `gorm:"not null;default:false"`      // Whether item has been counted
	Remark           string          `gorm:"type:varchar(500)"`
	CreatedAt        time.Time       `gorm:"not null"`
	UpdatedAt        time.Time       `gorm:"not null"`
}

// TableName returns the table name for GORM
func (StockTakingItem) TableName() string {
	return "stock_taking_items"
}

// NewStockTakingItem creates a new stock taking item
func NewStockTakingItem(stockTakingID, productID uuid.UUID, productName, productCode, unit string, systemQty, unitCost decimal.Decimal) *StockTakingItem {
	now := time.Now()
	return &StockTakingItem{
		ID:             uuid.New(),
		StockTakingID:  stockTakingID,
		ProductID:      productID,
		ProductName:    productName,
		ProductCode:    productCode,
		Unit:           unit,
		SystemQuantity: systemQty,
		UnitCost:       unitCost,
		Counted:        false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// RecordCount records the actual count for this item
func (i *StockTakingItem) RecordCount(actualQty decimal.Decimal, remark string) error {
	if actualQty.IsNegative() {
		return shared.NewDomainError("INVALID_QUANTITY", "Actual quantity cannot be negative")
	}

	i.ActualQuantity = actualQty
	i.DifferenceQty = actualQty.Sub(i.SystemQuantity)
	i.DifferenceAmount = i.DifferenceQty.Mul(i.UnitCost)
	i.Counted = true
	i.Remark = remark
	i.UpdatedAt = time.Now()

	return nil
}

// HasDifference returns true if there is a difference between system and actual
func (i *StockTakingItem) HasDifference() bool {
	return i.Counted && !i.DifferenceQty.IsZero()
}

// StockTaking represents a stock taking (inventory count) document
// It is the aggregate root for stock taking operations
type StockTaking struct {
	shared.TenantAggregateRoot
	TakingNumber      string            `gorm:"type:varchar(50);not null;uniqueIndex:idx_stock_taking_number_tenant,priority:2"`
	WarehouseID       uuid.UUID         `gorm:"type:uuid;not null;index"`
	WarehouseName     string            `gorm:"type:varchar(100);not null"`
	Status            StockTakingStatus `gorm:"type:varchar(20);not null;default:'DRAFT'"`
	TakingDate        time.Time         `gorm:"not null"`          // Date of stock taking
	StartedAt         *time.Time        `gorm:""`                  // When counting started
	CompletedAt       *time.Time        `gorm:""`                  // When counting completed
	ApprovedAt        *time.Time        `gorm:""`                  // When approved/rejected
	ApprovedByID      *uuid.UUID        `gorm:"type:uuid"`         // User who approved
	ApprovedByName    string            `gorm:"type:varchar(100)"` // Name of approver
	CreatedByID       uuid.UUID         `gorm:"type:uuid;not null"`
	CreatedByName     string            `gorm:"type:varchar(100);not null"`
	TotalItems        int               `gorm:"not null;default:0"`                    // Total number of items
	CountedItems      int               `gorm:"not null;default:0"`                    // Number of items counted
	DifferenceItems   int               `gorm:"not null;default:0"`                    // Number of items with difference
	TotalDifference   decimal.Decimal   `gorm:"type:decimal(18,4);not null;default:0"` // Total difference amount
	ApprovalNote      string            `gorm:"type:varchar(500)"`                     // Approval/rejection note
	Remark            string            `gorm:"type:varchar(500)"`
	Items             []StockTakingItem `gorm:"foreignKey:StockTakingID;references:ID"`
}

// TableName returns the table name for GORM
func (StockTaking) TableName() string {
	return "stock_takings"
}

// NewStockTaking creates a new stock taking document
func NewStockTaking(tenantID, warehouseID uuid.UUID, warehouseName, takingNumber string, takingDate time.Time, createdByID uuid.UUID, createdByName string) (*StockTaking, error) {
	if warehouseID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_WAREHOUSE", "Warehouse ID cannot be empty")
	}
	if warehouseName == "" {
		return nil, shared.NewDomainError("INVALID_WAREHOUSE_NAME", "Warehouse name cannot be empty")
	}
	if takingNumber == "" {
		return nil, shared.NewDomainError("INVALID_TAKING_NUMBER", "Taking number cannot be empty")
	}
	if createdByID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_CREATOR", "Creator ID cannot be empty")
	}

	st := &StockTaking{
		TenantAggregateRoot: shared.NewTenantAggregateRoot(tenantID),
		TakingNumber:        takingNumber,
		WarehouseID:         warehouseID,
		WarehouseName:       warehouseName,
		Status:              StockTakingStatusDraft,
		TakingDate:          takingDate,
		CreatedByID:         createdByID,
		CreatedByName:       createdByName,
		TotalItems:          0,
		CountedItems:        0,
		DifferenceItems:     0,
		TotalDifference:     decimal.Zero,
		Items:               make([]StockTakingItem, 0),
	}

	st.AddDomainEvent(NewStockTakingCreatedEvent(st))

	return st, nil
}

// AddItem adds an item to the stock taking document
func (st *StockTaking) AddItem(productID uuid.UUID, productName, productCode, unit string, systemQty, unitCost decimal.Decimal) error {
	if st.Status != StockTakingStatusDraft {
		return shared.NewDomainError("INVALID_STATUS", "Can only add items in DRAFT status")
	}
	if productID == uuid.Nil {
		return shared.NewDomainError("INVALID_PRODUCT", "Product ID cannot be empty")
	}

	// Check for duplicate product
	for _, item := range st.Items {
		if item.ProductID == productID {
			return shared.NewDomainError("DUPLICATE_PRODUCT", "Product already exists in stock taking")
		}
	}

	item := NewStockTakingItem(st.ID, productID, productName, productCode, unit, systemQty, unitCost)
	st.Items = append(st.Items, *item)
	st.TotalItems++
	st.UpdatedAt = time.Now()
	st.IncrementVersion()

	return nil
}

// RemoveItem removes an item from the stock taking document
func (st *StockTaking) RemoveItem(productID uuid.UUID) error {
	if st.Status != StockTakingStatusDraft {
		return shared.NewDomainError("INVALID_STATUS", "Can only remove items in DRAFT status")
	}

	for i, item := range st.Items {
		if item.ProductID == productID {
			st.Items = append(st.Items[:i], st.Items[i+1:]...)
			st.TotalItems--
			st.UpdatedAt = time.Now()
			st.IncrementVersion()
			return nil
		}
	}

	return shared.NewDomainError("ITEM_NOT_FOUND", "Product not found in stock taking")
}

// StartCounting transitions the stock taking to counting status
func (st *StockTaking) StartCounting() error {
	if !st.Status.CanTransitionTo(StockTakingStatusCounting) {
		return shared.NewDomainError("INVALID_TRANSITION", fmt.Sprintf("Cannot transition from %s to COUNTING", st.Status))
	}
	if st.TotalItems == 0 {
		return shared.NewDomainError("NO_ITEMS", "Cannot start counting with no items")
	}

	now := time.Now()
	st.Status = StockTakingStatusCounting
	st.StartedAt = &now
	st.UpdatedAt = now
	st.IncrementVersion()

	st.AddDomainEvent(NewStockTakingStartedEvent(st))

	return nil
}

// RecordItemCount records the actual count for an item
func (st *StockTaking) RecordItemCount(productID uuid.UUID, actualQty decimal.Decimal, remark string) error {
	if st.Status != StockTakingStatusCounting {
		return shared.NewDomainError("INVALID_STATUS", "Can only record counts in COUNTING status")
	}

	for i := range st.Items {
		if st.Items[i].ProductID == productID {
			wasCounted := st.Items[i].Counted

			if err := st.Items[i].RecordCount(actualQty, remark); err != nil {
				return err
			}

			// Update counted items count
			if !wasCounted {
				st.CountedItems++
			}

			st.recalculateTotals()
			st.UpdatedAt = time.Now()
			st.IncrementVersion()
			return nil
		}
	}

	return shared.NewDomainError("ITEM_NOT_FOUND", "Product not found in stock taking")
}

// recalculateTotals recalculates the totals after a count is recorded
func (st *StockTaking) recalculateTotals() {
	st.DifferenceItems = 0
	st.TotalDifference = decimal.Zero

	for _, item := range st.Items {
		if item.Counted && item.HasDifference() {
			st.DifferenceItems++
			st.TotalDifference = st.TotalDifference.Add(item.DifferenceAmount)
		}
	}
}

// SubmitForApproval transitions the stock taking to pending approval status
func (st *StockTaking) SubmitForApproval() error {
	if !st.Status.CanTransitionTo(StockTakingStatusPendingApproval) {
		return shared.NewDomainError("INVALID_TRANSITION", fmt.Sprintf("Cannot transition from %s to PENDING_APPROVAL", st.Status))
	}
	if st.CountedItems != st.TotalItems {
		return shared.NewDomainError("INCOMPLETE_COUNT", fmt.Sprintf("Not all items have been counted (%d/%d)", st.CountedItems, st.TotalItems))
	}

	now := time.Now()
	st.Status = StockTakingStatusPendingApproval
	st.CompletedAt = &now
	st.UpdatedAt = now
	st.IncrementVersion()

	st.AddDomainEvent(NewStockTakingSubmittedEvent(st))

	return nil
}

// Approve approves the stock taking and triggers inventory adjustments
func (st *StockTaking) Approve(approverID uuid.UUID, approverName, note string) error {
	if !st.Status.CanTransitionTo(StockTakingStatusApproved) {
		return shared.NewDomainError("INVALID_TRANSITION", fmt.Sprintf("Cannot transition from %s to APPROVED", st.Status))
	}
	if approverID == uuid.Nil {
		return shared.NewDomainError("INVALID_APPROVER", "Approver ID cannot be empty")
	}

	now := time.Now()
	st.Status = StockTakingStatusApproved
	st.ApprovedAt = &now
	st.ApprovedByID = &approverID
	st.ApprovedByName = approverName
	st.ApprovalNote = note
	st.UpdatedAt = now
	st.IncrementVersion()

	st.AddDomainEvent(NewStockTakingApprovedEvent(st))

	return nil
}

// Reject rejects the stock taking
func (st *StockTaking) Reject(approverID uuid.UUID, approverName, reason string) error {
	if !st.Status.CanTransitionTo(StockTakingStatusRejected) {
		return shared.NewDomainError("INVALID_TRANSITION", fmt.Sprintf("Cannot transition from %s to REJECTED", st.Status))
	}
	if approverID == uuid.Nil {
		return shared.NewDomainError("INVALID_APPROVER", "Approver ID cannot be empty")
	}
	if reason == "" {
		return shared.NewDomainError("INVALID_REASON", "Rejection reason is required")
	}

	now := time.Now()
	st.Status = StockTakingStatusRejected
	st.ApprovedAt = &now
	st.ApprovedByID = &approverID
	st.ApprovedByName = approverName
	st.ApprovalNote = reason
	st.UpdatedAt = now
	st.IncrementVersion()

	st.AddDomainEvent(NewStockTakingRejectedEvent(st))

	return nil
}

// Cancel cancels the stock taking
func (st *StockTaking) Cancel(reason string) error {
	if !st.Status.CanTransitionTo(StockTakingStatusCancelled) {
		return shared.NewDomainError("INVALID_TRANSITION", fmt.Sprintf("Cannot transition from %s to CANCELLED", st.Status))
	}

	st.Status = StockTakingStatusCancelled
	st.Remark = reason
	st.UpdatedAt = time.Now()
	st.IncrementVersion()

	st.AddDomainEvent(NewStockTakingCancelledEvent(st))

	return nil
}

// SetRemark sets the remark for the stock taking
func (st *StockTaking) SetRemark(remark string) {
	st.Remark = remark
	st.UpdatedAt = time.Now()
}

// IsComplete returns true if all items have been counted
func (st *StockTaking) IsComplete() bool {
	return st.CountedItems == st.TotalItems && st.TotalItems > 0
}

// GetProgress returns the counting progress as a percentage
func (st *StockTaking) GetProgress() float64 {
	if st.TotalItems == 0 {
		return 0
	}
	return float64(st.CountedItems) / float64(st.TotalItems) * 100
}

// GetItemsWithDifference returns items that have a difference between system and actual quantity
func (st *StockTaking) GetItemsWithDifference() []StockTakingItem {
	result := make([]StockTakingItem, 0)
	for _, item := range st.Items {
		if item.HasDifference() {
			result = append(result, item)
		}
	}
	return result
}

// GetUncountedItems returns items that have not been counted yet
func (st *StockTaking) GetUncountedItems() []StockTakingItem {
	result := make([]StockTakingItem, 0)
	for _, item := range st.Items {
		if !item.Counted {
			result = append(result, item)
		}
	}
	return result
}
