package trade

import (
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ReturnStatus represents the status of a sales return
type ReturnStatus string

const (
	ReturnStatusDraft     ReturnStatus = "DRAFT"
	ReturnStatusPending   ReturnStatus = "PENDING"   // Waiting for approval
	ReturnStatusApproved  ReturnStatus = "APPROVED"  // Approved, ready for processing
	ReturnStatusReceiving ReturnStatus = "RECEIVING" // Receiving returned goods into warehouse
	ReturnStatusRejected  ReturnStatus = "REJECTED"  // Rejected by approver
	ReturnStatusCompleted ReturnStatus = "COMPLETED" // Return completed, stock restored
	ReturnStatusCancelled ReturnStatus = "CANCELLED"
)

// IsValid checks if the status is a valid ReturnStatus
func (s ReturnStatus) IsValid() bool {
	switch s {
	case ReturnStatusDraft, ReturnStatusPending, ReturnStatusApproved,
		ReturnStatusReceiving, ReturnStatusRejected, ReturnStatusCompleted, ReturnStatusCancelled:
		return true
	}
	return false
}

// String returns the string representation of ReturnStatus
func (s ReturnStatus) String() string {
	return string(s)
}

// CanTransitionTo checks if the status can transition to the target status
func (s ReturnStatus) CanTransitionTo(target ReturnStatus) bool {
	switch s {
	case ReturnStatusDraft:
		return target == ReturnStatusPending || target == ReturnStatusCancelled
	case ReturnStatusPending:
		return target == ReturnStatusApproved || target == ReturnStatusRejected || target == ReturnStatusCancelled
	case ReturnStatusApproved:
		return target == ReturnStatusReceiving || target == ReturnStatusCancelled
	case ReturnStatusReceiving:
		return target == ReturnStatusCompleted || target == ReturnStatusCancelled
	case ReturnStatusRejected, ReturnStatusCompleted, ReturnStatusCancelled:
		return false // Terminal states
	}
	return false
}

// SalesReturnItem represents a line item in a sales return
type SalesReturnItem struct {
	ID                uuid.UUID
	ReturnID          uuid.UUID
	SalesOrderItemID  uuid.UUID // Reference to original order item
	ProductID         uuid.UUID
	ProductName       string
	ProductCode       string
	OriginalQuantity  decimal.Decimal // Quantity in original order
	ReturnQuantity    decimal.Decimal // Quantity being returned
	UnitPrice         decimal.Decimal // Price per unit (from original order)
	RefundAmount      decimal.Decimal // ReturnQuantity * UnitPrice
	Unit              string
	ConversionRate    decimal.Decimal // Conversion rate to base unit
	BaseQuantity      decimal.Decimal // Return quantity in base units (for inventory)
	BaseUnit          string          // Base unit code
	Reason            string
	ConditionOnReturn string // e.g., "damaged", "defective", "wrong_item"
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// NewSalesReturnItem creates a new sales return item
// Parameters:
//   - returnID: the parent return ID
//   - salesOrderItemID: the original sales order item ID
//   - productID: the product ID
//   - productName, productCode: product display info
//   - unit: the unit of measure (may be auxiliary unit)
//   - baseUnit: the base unit code for the product
//   - originalQuantity: quantity in original order (in order unit)
//   - returnQuantity: quantity being returned (in order unit)
//   - conversionRate: conversion rate from order unit to base unit (1 if using base unit)
//   - unitPrice: price per order unit
func NewSalesReturnItem(
	returnID uuid.UUID,
	salesOrderItemID uuid.UUID,
	productID uuid.UUID,
	productName, productCode, unit, baseUnit string,
	originalQuantity, returnQuantity, conversionRate decimal.Decimal,
	unitPrice valueobject.Money,
) (*SalesReturnItem, error) {
	if productID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_PRODUCT", "Product ID cannot be empty")
	}
	if salesOrderItemID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_ORDER_ITEM", "Sales order item ID cannot be empty")
	}
	if productName == "" {
		return nil, shared.NewDomainError("INVALID_PRODUCT_NAME", "Product name cannot be empty")
	}
	if returnQuantity.LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("INVALID_QUANTITY", "Return quantity must be positive")
	}
	if returnQuantity.GreaterThan(originalQuantity) {
		return nil, shared.NewDomainError("INVALID_QUANTITY", "Return quantity cannot exceed original quantity")
	}
	if unit == "" {
		return nil, shared.NewDomainError("INVALID_UNIT", "Unit cannot be empty")
	}
	if baseUnit == "" {
		return nil, shared.NewDomainError("INVALID_BASE_UNIT", "Base unit cannot be empty")
	}
	if conversionRate.LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("INVALID_CONVERSION_RATE", "Conversion rate must be positive")
	}

	now := time.Now()
	refundAmount := returnQuantity.Mul(unitPrice.Amount())
	// Calculate base quantity: returnQuantity * conversionRate
	baseQuantity := returnQuantity.Mul(conversionRate).Round(4)

	return &SalesReturnItem{
		ID:               uuid.New(),
		ReturnID:         returnID,
		SalesOrderItemID: salesOrderItemID,
		ProductID:        productID,
		ProductName:      productName,
		ProductCode:      productCode,
		OriginalQuantity: originalQuantity,
		ReturnQuantity:   returnQuantity,
		UnitPrice:        unitPrice.Amount(),
		RefundAmount:     refundAmount,
		Unit:             unit,
		ConversionRate:   conversionRate,
		BaseQuantity:     baseQuantity,
		BaseUnit:         baseUnit,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

// UpdateReturnQuantity updates the return quantity and recalculates the refund amount and base quantity
func (i *SalesReturnItem) UpdateReturnQuantity(quantity decimal.Decimal) error {
	if quantity.LessThanOrEqual(decimal.Zero) {
		return shared.NewDomainError("INVALID_QUANTITY", "Return quantity must be positive")
	}
	if quantity.GreaterThan(i.OriginalQuantity) {
		return shared.NewDomainError("INVALID_QUANTITY", "Return quantity cannot exceed original quantity")
	}

	i.ReturnQuantity = quantity
	i.RefundAmount = quantity.Mul(i.UnitPrice)
	i.BaseQuantity = quantity.Mul(i.ConversionRate).Round(4)
	i.UpdatedAt = time.Now()

	return nil
}

// SetReason sets the return reason for the item
func (i *SalesReturnItem) SetReason(reason string) {
	i.Reason = reason
	i.UpdatedAt = time.Now()
}

// SetCondition sets the condition of the returned item
func (i *SalesReturnItem) SetCondition(condition string) {
	i.ConditionOnReturn = condition
	i.UpdatedAt = time.Now()
}

// GetRefundAmountMoney returns the refund amount as Money value object
func (i *SalesReturnItem) GetRefundAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(i.RefundAmount)
}

// GetUnitPriceMoney returns the unit price as Money value object
func (i *SalesReturnItem) GetUnitPriceMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(i.UnitPrice)
}

// SalesReturn represents a sales return aggregate root
// It manages the return of goods from a customer for a previous sales order
type SalesReturn struct {
	shared.TenantAggregateRoot
	ReturnNumber     string
	SalesOrderID     uuid.UUID // Reference to original sales order
	SalesOrderNumber string
	CustomerID       uuid.UUID
	CustomerName     string
	WarehouseID      *uuid.UUID // Warehouse where returned goods will be stored
	Items            []SalesReturnItem
	TotalRefund      decimal.Decimal // Sum of all item refunds
	Status           ReturnStatus
	Reason           string // Overall return reason
	Remark           string
	SubmittedAt      *time.Time // When submitted for approval
	ApprovedAt       *time.Time
	ApprovedBy       *uuid.UUID
	ApprovalNote     string
	RejectedAt       *time.Time
	RejectedBy       *uuid.UUID
	RejectionReason  string
	ReceivedAt       *time.Time // When receiving process started
	CompletedAt      *time.Time
	CancelledAt      *time.Time
	CancelReason     string
}

// NewSalesReturn creates a new sales return
func NewSalesReturn(
	tenantID uuid.UUID,
	returnNumber string,
	salesOrder *SalesOrder,
) (*SalesReturn, error) {
	if returnNumber == "" {
		return nil, shared.NewDomainError("INVALID_RETURN_NUMBER", "Return number cannot be empty")
	}
	if len(returnNumber) > 50 {
		return nil, shared.NewDomainError("INVALID_RETURN_NUMBER", "Return number cannot exceed 50 characters")
	}
	if salesOrder == nil {
		return nil, shared.NewDomainError("INVALID_ORDER", "Sales order cannot be nil")
	}
	if !salesOrder.IsShipped() && !salesOrder.IsCompleted() {
		return nil, shared.NewDomainError("INVALID_ORDER_STATUS", "Can only create returns for shipped or completed orders")
	}

	sr := &SalesReturn{
		TenantAggregateRoot: shared.NewTenantAggregateRoot(tenantID),
		ReturnNumber:        returnNumber,
		SalesOrderID:        salesOrder.ID,
		SalesOrderNumber:    salesOrder.OrderNumber,
		CustomerID:          salesOrder.CustomerID,
		CustomerName:        salesOrder.CustomerName,
		WarehouseID:         salesOrder.WarehouseID,
		Items:               make([]SalesReturnItem, 0),
		TotalRefund:         decimal.Zero,
		Status:              ReturnStatusDraft,
	}

	sr.AddDomainEvent(NewSalesReturnCreatedEvent(sr))

	return sr, nil
}

// AddItem adds a new item to the return
// Only allowed in DRAFT status
func (r *SalesReturn) AddItem(
	salesOrderItem *SalesOrderItem,
	returnQuantity decimal.Decimal,
) (*SalesReturnItem, error) {
	if r.Status != ReturnStatusDraft {
		return nil, shared.NewDomainError("INVALID_STATE", "Cannot add items to a non-draft return")
	}
	if salesOrderItem == nil {
		return nil, shared.NewDomainError("INVALID_ITEM", "Sales order item cannot be nil")
	}

	// Check if item already exists in return
	for _, item := range r.Items {
		if item.SalesOrderItemID == salesOrderItem.ID {
			return nil, shared.NewDomainError("DUPLICATE_ITEM", "Item already exists in return, update quantity instead")
		}
	}

	item, err := NewSalesReturnItem(
		r.ID,
		salesOrderItem.ID,
		salesOrderItem.ProductID,
		salesOrderItem.ProductName,
		salesOrderItem.ProductCode,
		salesOrderItem.Unit,
		salesOrderItem.BaseUnit,
		salesOrderItem.Quantity,
		returnQuantity,
		salesOrderItem.ConversionRate,
		salesOrderItem.GetUnitPriceMoney(),
	)
	if err != nil {
		return nil, err
	}

	r.Items = append(r.Items, *item)
	r.recalculateTotalRefund()
	r.UpdatedAt = time.Now()

	return item, nil
}

// UpdateItemQuantity updates the return quantity of an existing item
// Only allowed in DRAFT status
func (r *SalesReturn) UpdateItemQuantity(itemID uuid.UUID, quantity decimal.Decimal) error {
	if r.Status != ReturnStatusDraft {
		return shared.NewDomainError("INVALID_STATE", "Cannot update items in a non-draft return")
	}

	for idx := range r.Items {
		if r.Items[idx].ID == itemID {
			if err := r.Items[idx].UpdateReturnQuantity(quantity); err != nil {
				return err
			}
			r.recalculateTotalRefund()
			r.UpdatedAt = time.Now()
			return nil
		}
	}

	return shared.NewDomainError("ITEM_NOT_FOUND", "Return item not found")
}

// RemoveItem removes an item from the return
// Only allowed in DRAFT status
func (r *SalesReturn) RemoveItem(itemID uuid.UUID) error {
	if r.Status != ReturnStatusDraft {
		return shared.NewDomainError("INVALID_STATE", "Cannot remove items from a non-draft return")
	}

	for idx, item := range r.Items {
		if item.ID == itemID {
			r.Items = append(r.Items[:idx], r.Items[idx+1:]...)
			r.recalculateTotalRefund()
			r.UpdatedAt = time.Now()
			return nil
		}
	}

	return shared.NewDomainError("ITEM_NOT_FOUND", "Return item not found")
}

// SetReason sets the overall return reason
func (r *SalesReturn) SetReason(reason string) {
	r.Reason = reason
	r.UpdatedAt = time.Now()
}

// SetRemark sets the return remark
func (r *SalesReturn) SetRemark(remark string) {
	r.Remark = remark
	r.UpdatedAt = time.Now()
}

// SetWarehouse sets the warehouse for returned goods
// Allowed in DRAFT, PENDING, or APPROVED status (before completion)
func (r *SalesReturn) SetWarehouse(warehouseID uuid.UUID) error {
	if r.Status != ReturnStatusDraft && r.Status != ReturnStatusPending && r.Status != ReturnStatusApproved {
		return shared.NewDomainError("INVALID_STATE", "Cannot set warehouse for return in current status")
	}
	if warehouseID == uuid.Nil {
		return shared.NewDomainError("INVALID_WAREHOUSE", "Warehouse ID cannot be empty")
	}

	r.WarehouseID = &warehouseID
	r.UpdatedAt = time.Now()

	return nil
}

// Submit submits the return for approval
// Transitions from DRAFT to PENDING
func (r *SalesReturn) Submit() error {
	if !r.Status.CanTransitionTo(ReturnStatusPending) {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot submit return in %s status", r.Status))
	}
	if len(r.Items) == 0 {
		return shared.NewDomainError("NO_ITEMS", "Cannot submit return without items")
	}
	if r.TotalRefund.LessThanOrEqual(decimal.Zero) {
		return shared.NewDomainError("INVALID_AMOUNT", "Return total refund must be positive")
	}

	now := time.Now()
	r.Status = ReturnStatusPending
	r.SubmittedAt = &now
	r.UpdatedAt = now

	r.AddDomainEvent(NewSalesReturnSubmittedEvent(r))

	return nil
}

// Approve approves the return
// Transitions from PENDING to APPROVED
func (r *SalesReturn) Approve(approverID uuid.UUID, note string) error {
	if !r.Status.CanTransitionTo(ReturnStatusApproved) {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot approve return in %s status", r.Status))
	}
	if approverID == uuid.Nil {
		return shared.NewDomainError("INVALID_APPROVER", "Approver ID cannot be empty")
	}

	now := time.Now()
	r.Status = ReturnStatusApproved
	r.ApprovedAt = &now
	r.ApprovedBy = &approverID
	r.ApprovalNote = note
	r.UpdatedAt = now

	r.AddDomainEvent(NewSalesReturnApprovedEvent(r))

	return nil
}

// Reject rejects the return
// Transitions from PENDING to REJECTED
func (r *SalesReturn) Reject(rejecterID uuid.UUID, reason string) error {
	if !r.Status.CanTransitionTo(ReturnStatusRejected) {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot reject return in %s status", r.Status))
	}
	if rejecterID == uuid.Nil {
		return shared.NewDomainError("INVALID_REJECTER", "Rejecter ID cannot be empty")
	}
	if reason == "" {
		return shared.NewDomainError("INVALID_REASON", "Rejection reason is required")
	}

	now := time.Now()
	r.Status = ReturnStatusRejected
	r.RejectedAt = &now
	r.RejectedBy = &rejecterID
	r.RejectionReason = reason
	r.UpdatedAt = now

	r.AddDomainEvent(NewSalesReturnRejectedEvent(r))

	return nil
}

// Complete marks the return as completed
// This should be called after stock has been restored
// Transitions from RECEIVING to COMPLETED
func (r *SalesReturn) Complete() error {
	if !r.Status.CanTransitionTo(ReturnStatusCompleted) {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot complete return in %s status", r.Status))
	}
	if r.WarehouseID == nil {
		return shared.NewDomainError("NO_WAREHOUSE", "Warehouse must be set before completing return")
	}

	now := time.Now()
	r.Status = ReturnStatusCompleted
	r.CompletedAt = &now
	r.UpdatedAt = now

	r.AddDomainEvent(NewSalesReturnCompletedEvent(r))

	return nil
}

// Receive starts the receiving process for returned goods
// Transitions from APPROVED to RECEIVING
func (r *SalesReturn) Receive() error {
	if !r.Status.CanTransitionTo(ReturnStatusReceiving) {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot start receiving return in %s status", r.Status))
	}
	if r.WarehouseID == nil {
		return shared.NewDomainError("NO_WAREHOUSE", "Warehouse must be set before receiving return")
	}

	now := time.Now()
	r.Status = ReturnStatusReceiving
	r.ReceivedAt = &now
	r.UpdatedAt = now

	r.AddDomainEvent(NewSalesReturnReceivedEvent(r))

	return nil
}

// Cancel cancels the return
// Allowed in DRAFT, PENDING, APPROVED, or RECEIVING status
func (r *SalesReturn) Cancel(reason string) error {
	if !r.Status.CanTransitionTo(ReturnStatusCancelled) {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot cancel return in %s status", r.Status))
	}
	if reason == "" {
		return shared.NewDomainError("INVALID_REASON", "Cancel reason is required")
	}

	// Track if we were in approved or receiving state (may need to reverse inventory operations)
	wasApproved := r.Status == ReturnStatusApproved || r.Status == ReturnStatusReceiving
	now := time.Now()
	r.Status = ReturnStatusCancelled
	r.CancelledAt = &now
	r.CancelReason = reason
	r.UpdatedAt = now

	r.AddDomainEvent(NewSalesReturnCancelledEvent(r, wasApproved))

	return nil
}

// recalculateTotalRefund recalculates the total refund amount
func (r *SalesReturn) recalculateTotalRefund() {
	total := decimal.Zero
	for _, item := range r.Items {
		total = total.Add(item.RefundAmount)
	}
	r.TotalRefund = total
}

// GetTotalRefundMoney returns total refund as Money
func (r *SalesReturn) GetTotalRefundMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(r.TotalRefund)
}

// ItemCount returns the number of items in the return
func (r *SalesReturn) ItemCount() int {
	return len(r.Items)
}

// TotalReturnQuantity returns the sum of all item return quantities
func (r *SalesReturn) TotalReturnQuantity() decimal.Decimal {
	total := decimal.Zero
	for _, item := range r.Items {
		total = total.Add(item.ReturnQuantity)
	}
	return total
}

// IsDraft returns true if return is in draft status
func (r *SalesReturn) IsDraft() bool {
	return r.Status == ReturnStatusDraft
}

// IsPending returns true if return is pending approval
func (r *SalesReturn) IsPending() bool {
	return r.Status == ReturnStatusPending
}

// IsApproved returns true if return is approved
func (r *SalesReturn) IsApproved() bool {
	return r.Status == ReturnStatusApproved
}

// IsReceiving returns true if return is in receiving status
func (r *SalesReturn) IsReceiving() bool {
	return r.Status == ReturnStatusReceiving
}

// IsRejected returns true if return is rejected
func (r *SalesReturn) IsRejected() bool {
	return r.Status == ReturnStatusRejected
}

// IsCompleted returns true if return is completed
func (r *SalesReturn) IsCompleted() bool {
	return r.Status == ReturnStatusCompleted
}

// IsCancelled returns true if return is cancelled
func (r *SalesReturn) IsCancelled() bool {
	return r.Status == ReturnStatusCancelled
}

// IsTerminal returns true if return is in a terminal state
func (r *SalesReturn) IsTerminal() bool {
	return r.IsCompleted() || r.IsCancelled() || r.IsRejected()
}

// CanModify returns true if the return can be modified
func (r *SalesReturn) CanModify() bool {
	return r.IsDraft()
}

// GetItem returns an item by its ID
func (r *SalesReturn) GetItem(itemID uuid.UUID) *SalesReturnItem {
	for idx := range r.Items {
		if r.Items[idx].ID == itemID {
			return &r.Items[idx]
		}
	}
	return nil
}

// GetItemByOrderItem returns an item by its original order item ID
func (r *SalesReturn) GetItemByOrderItem(orderItemID uuid.UUID) *SalesReturnItem {
	for idx := range r.Items {
		if r.Items[idx].SalesOrderItemID == orderItemID {
			return &r.Items[idx]
		}
	}
	return nil
}
