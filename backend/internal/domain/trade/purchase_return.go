package trade

import (
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PurchaseReturnStatus represents the status of a purchase return
type PurchaseReturnStatus string

const (
	PurchaseReturnStatusDraft     PurchaseReturnStatus = "DRAFT"
	PurchaseReturnStatusPending   PurchaseReturnStatus = "PENDING"   // Waiting for approval
	PurchaseReturnStatusApproved  PurchaseReturnStatus = "APPROVED"  // Approved, ready for shipping
	PurchaseReturnStatusRejected  PurchaseReturnStatus = "REJECTED"  // Rejected by approver
	PurchaseReturnStatusShipped   PurchaseReturnStatus = "SHIPPED"   // Goods shipped back to supplier
	PurchaseReturnStatusCompleted PurchaseReturnStatus = "COMPLETED" // Return completed, supplier confirmed
	PurchaseReturnStatusCancelled PurchaseReturnStatus = "CANCELLED"
)

// IsValid checks if the status is a valid PurchaseReturnStatus
func (s PurchaseReturnStatus) IsValid() bool {
	switch s {
	case PurchaseReturnStatusDraft, PurchaseReturnStatusPending, PurchaseReturnStatusApproved,
		PurchaseReturnStatusRejected, PurchaseReturnStatusShipped, PurchaseReturnStatusCompleted,
		PurchaseReturnStatusCancelled:
		return true
	}
	return false
}

// String returns the string representation of PurchaseReturnStatus
func (s PurchaseReturnStatus) String() string {
	return string(s)
}

// CanTransitionTo checks if the status can transition to the target status
func (s PurchaseReturnStatus) CanTransitionTo(target PurchaseReturnStatus) bool {
	switch s {
	case PurchaseReturnStatusDraft:
		return target == PurchaseReturnStatusPending || target == PurchaseReturnStatusCancelled
	case PurchaseReturnStatusPending:
		return target == PurchaseReturnStatusApproved || target == PurchaseReturnStatusRejected || target == PurchaseReturnStatusCancelled
	case PurchaseReturnStatusApproved:
		return target == PurchaseReturnStatusShipped || target == PurchaseReturnStatusCancelled
	case PurchaseReturnStatusShipped:
		return target == PurchaseReturnStatusCompleted
	case PurchaseReturnStatusRejected, PurchaseReturnStatusCompleted, PurchaseReturnStatusCancelled:
		return false // Terminal states
	}
	return false
}

// PurchaseReturnItem represents a line item in a purchase return
type PurchaseReturnItem struct {
	ID                  uuid.UUID
	ReturnID            uuid.UUID
	PurchaseOrderItemID uuid.UUID // Reference to original order item
	ProductID           uuid.UUID
	ProductName         string
	ProductCode         string
	OriginalQuantity    decimal.Decimal // Quantity in original order (received)
	ReturnQuantity      decimal.Decimal // Quantity being returned
	UnitCost            decimal.Decimal // Cost per unit (from original order)
	RefundAmount        decimal.Decimal // ReturnQuantity * UnitCost
	Unit                string
	Reason              string
	ConditionOnReturn   string // e.g., "defective", "wrong_item", "excess"
	BatchNumber         string // Batch being returned
	ShippedQuantity     decimal.Decimal
	ShippedAt           *time.Time
	SupplierReceivedQty decimal.Decimal // Quantity confirmed by supplier
	SupplierReceivedAt  *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// NewPurchaseReturnItem creates a new purchase return item
func NewPurchaseReturnItem(
	returnID uuid.UUID,
	purchaseOrderItemID uuid.UUID,
	productID uuid.UUID,
	productName, productCode, unit string,
	originalQuantity, returnQuantity decimal.Decimal,
	unitCost valueobject.Money,
) (*PurchaseReturnItem, error) {
	if productID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_PRODUCT", "Product ID cannot be empty")
	}
	if purchaseOrderItemID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_ORDER_ITEM", "Purchase order item ID cannot be empty")
	}
	if productName == "" {
		return nil, shared.NewDomainError("INVALID_PRODUCT_NAME", "Product name cannot be empty")
	}
	if returnQuantity.LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("INVALID_QUANTITY", "Return quantity must be positive")
	}
	if returnQuantity.GreaterThan(originalQuantity) {
		return nil, shared.NewDomainError("INVALID_QUANTITY", "Return quantity cannot exceed received quantity")
	}
	if unit == "" {
		return nil, shared.NewDomainError("INVALID_UNIT", "Unit cannot be empty")
	}

	now := time.Now()
	refundAmount := returnQuantity.Mul(unitCost.Amount())

	return &PurchaseReturnItem{
		ID:                  uuid.New(),
		ReturnID:            returnID,
		PurchaseOrderItemID: purchaseOrderItemID,
		ProductID:           productID,
		ProductName:         productName,
		ProductCode:         productCode,
		OriginalQuantity:    originalQuantity,
		ReturnQuantity:      returnQuantity,
		UnitCost:            unitCost.Amount(),
		RefundAmount:        refundAmount,
		Unit:                unit,
		ShippedQuantity:     decimal.Zero,
		SupplierReceivedQty: decimal.Zero,
		CreatedAt:           now,
		UpdatedAt:           now,
	}, nil
}

// UpdateReturnQuantity updates the return quantity and recalculates the refund amount
func (i *PurchaseReturnItem) UpdateReturnQuantity(quantity decimal.Decimal) error {
	if quantity.LessThanOrEqual(decimal.Zero) {
		return shared.NewDomainError("INVALID_QUANTITY", "Return quantity must be positive")
	}
	if quantity.GreaterThan(i.OriginalQuantity) {
		return shared.NewDomainError("INVALID_QUANTITY", "Return quantity cannot exceed received quantity")
	}

	i.ReturnQuantity = quantity
	i.RefundAmount = quantity.Mul(i.UnitCost)
	i.UpdatedAt = time.Now()

	return nil
}

// SetReason sets the return reason for the item
func (i *PurchaseReturnItem) SetReason(reason string) {
	i.Reason = reason
	i.UpdatedAt = time.Now()
}

// SetCondition sets the condition of the returned item
func (i *PurchaseReturnItem) SetCondition(condition string) {
	i.ConditionOnReturn = condition
	i.UpdatedAt = time.Now()
}

// SetBatchNumber sets the batch number being returned
func (i *PurchaseReturnItem) SetBatchNumber(batchNumber string) {
	i.BatchNumber = batchNumber
	i.UpdatedAt = time.Now()
}

// MarkShipped marks the item as shipped
func (i *PurchaseReturnItem) MarkShipped(quantity decimal.Decimal) error {
	if quantity.LessThanOrEqual(decimal.Zero) {
		return shared.NewDomainError("INVALID_QUANTITY", "Ship quantity must be positive")
	}
	if quantity.GreaterThan(i.ReturnQuantity) {
		return shared.NewDomainError("INVALID_QUANTITY", "Ship quantity cannot exceed return quantity")
	}

	now := time.Now()
	i.ShippedQuantity = quantity
	i.ShippedAt = &now
	i.UpdatedAt = now

	return nil
}

// ConfirmSupplierReceived confirms the supplier received the goods
func (i *PurchaseReturnItem) ConfirmSupplierReceived(quantity decimal.Decimal) error {
	if quantity.LessThanOrEqual(decimal.Zero) {
		return shared.NewDomainError("INVALID_QUANTITY", "Received quantity must be positive")
	}
	if quantity.GreaterThan(i.ShippedQuantity) {
		return shared.NewDomainError("INVALID_QUANTITY", "Received quantity cannot exceed shipped quantity")
	}

	now := time.Now()
	i.SupplierReceivedQty = quantity
	i.SupplierReceivedAt = &now
	i.UpdatedAt = now

	return nil
}

// GetRefundAmountMoney returns the refund amount as Money value object
func (i *PurchaseReturnItem) GetRefundAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(i.RefundAmount)
}

// GetUnitCostMoney returns the unit cost as Money value object
func (i *PurchaseReturnItem) GetUnitCostMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(i.UnitCost)
}

// IsFullyShipped returns true if the full return quantity has been shipped
func (i *PurchaseReturnItem) IsFullyShipped() bool {
	return i.ShippedQuantity.GreaterThanOrEqual(i.ReturnQuantity)
}

// IsFullyReceived returns true if the supplier has confirmed receiving all shipped goods
func (i *PurchaseReturnItem) IsFullyReceived() bool {
	return i.SupplierReceivedQty.GreaterThanOrEqual(i.ShippedQuantity) && i.ShippedQuantity.GreaterThan(decimal.Zero)
}

// PurchaseReturn represents a purchase return aggregate root
// It manages the return of goods to a supplier for a previous purchase order
type PurchaseReturn struct {
	shared.TenantAggregateRoot
	ReturnNumber        string
	PurchaseOrderID     uuid.UUID // Reference to original purchase order
	PurchaseOrderNumber string
	SupplierID          uuid.UUID
	SupplierName        string
	WarehouseID         *uuid.UUID // Warehouse where goods are shipped from
	Items               []PurchaseReturnItem
	TotalRefund         decimal.Decimal // Sum of all item refunds
	Status              PurchaseReturnStatus
	Reason              string // Overall return reason
	Remark              string
	SubmittedAt         *time.Time // When submitted for approval
	ApprovedAt          *time.Time
	ApprovedBy          *uuid.UUID
	ApprovalNote        string
	RejectedAt          *time.Time
	RejectedBy          *uuid.UUID
	RejectionReason     string
	ShippedAt           *time.Time
	ShippedBy           *uuid.UUID
	ShippingNote        string
	TrackingNumber      string
	CompletedAt         *time.Time
	CancelledAt         *time.Time
	CancelReason        string
}

// NewPurchaseReturn creates a new purchase return
func NewPurchaseReturn(
	tenantID uuid.UUID,
	returnNumber string,
	purchaseOrder *PurchaseOrder,
) (*PurchaseReturn, error) {
	if returnNumber == "" {
		return nil, shared.NewDomainError("INVALID_RETURN_NUMBER", "Return number cannot be empty")
	}
	if len(returnNumber) > 50 {
		return nil, shared.NewDomainError("INVALID_RETURN_NUMBER", "Return number cannot exceed 50 characters")
	}
	if purchaseOrder == nil {
		return nil, shared.NewDomainError("INVALID_ORDER", "Purchase order cannot be nil")
	}
	// Can only return from orders that have received goods
	if !purchaseOrder.IsPartialReceived() && !purchaseOrder.IsCompleted() {
		return nil, shared.NewDomainError("INVALID_ORDER_STATUS", "Can only create returns for orders with received goods")
	}

	pr := &PurchaseReturn{
		TenantAggregateRoot: shared.NewTenantAggregateRoot(tenantID),
		ReturnNumber:        returnNumber,
		PurchaseOrderID:     purchaseOrder.ID,
		PurchaseOrderNumber: purchaseOrder.OrderNumber,
		SupplierID:          purchaseOrder.SupplierID,
		SupplierName:        purchaseOrder.SupplierName,
		WarehouseID:         purchaseOrder.WarehouseID,
		Items:               make([]PurchaseReturnItem, 0),
		TotalRefund:         decimal.Zero,
		Status:              PurchaseReturnStatusDraft,
	}

	pr.AddDomainEvent(NewPurchaseReturnCreatedEvent(pr))

	return pr, nil
}

// AddItem adds a new item to the return
// Only allowed in DRAFT status
func (r *PurchaseReturn) AddItem(
	purchaseOrderItem *PurchaseOrderItem,
	returnQuantity decimal.Decimal,
) (*PurchaseReturnItem, error) {
	if r.Status != PurchaseReturnStatusDraft {
		return nil, shared.NewDomainError("INVALID_STATE", "Cannot add items to a non-draft return")
	}
	if purchaseOrderItem == nil {
		return nil, shared.NewDomainError("INVALID_ITEM", "Purchase order item cannot be nil")
	}

	// Check if item already exists in return
	for _, item := range r.Items {
		if item.PurchaseOrderItemID == purchaseOrderItem.ID {
			return nil, shared.NewDomainError("DUPLICATE_ITEM", "Item already exists in return, update quantity instead")
		}
	}

	// Can only return received quantity
	if returnQuantity.GreaterThan(purchaseOrderItem.ReceivedQuantity) {
		return nil, shared.NewDomainError("INVALID_QUANTITY", "Return quantity cannot exceed received quantity")
	}

	item, err := NewPurchaseReturnItem(
		r.ID,
		purchaseOrderItem.ID,
		purchaseOrderItem.ProductID,
		purchaseOrderItem.ProductName,
		purchaseOrderItem.ProductCode,
		purchaseOrderItem.Unit,
		purchaseOrderItem.ReceivedQuantity,
		returnQuantity,
		purchaseOrderItem.GetUnitCostMoney(),
	)
	if err != nil {
		return nil, err
	}

	r.Items = append(r.Items, *item)
	r.recalculateTotalRefund()
	r.UpdatedAt = time.Now()
	r.IncrementVersion()

	return item, nil
}

// UpdateItemQuantity updates the return quantity of an existing item
// Only allowed in DRAFT status
func (r *PurchaseReturn) UpdateItemQuantity(itemID uuid.UUID, quantity decimal.Decimal) error {
	if r.Status != PurchaseReturnStatusDraft {
		return shared.NewDomainError("INVALID_STATE", "Cannot update items in a non-draft return")
	}

	for idx := range r.Items {
		if r.Items[idx].ID == itemID {
			if err := r.Items[idx].UpdateReturnQuantity(quantity); err != nil {
				return err
			}
			r.recalculateTotalRefund()
			r.UpdatedAt = time.Now()
			r.IncrementVersion()
			return nil
		}
	}

	return shared.NewDomainError("ITEM_NOT_FOUND", "Return item not found")
}

// RemoveItem removes an item from the return
// Only allowed in DRAFT status
func (r *PurchaseReturn) RemoveItem(itemID uuid.UUID) error {
	if r.Status != PurchaseReturnStatusDraft {
		return shared.NewDomainError("INVALID_STATE", "Cannot remove items from a non-draft return")
	}

	for idx, item := range r.Items {
		if item.ID == itemID {
			r.Items = append(r.Items[:idx], r.Items[idx+1:]...)
			r.recalculateTotalRefund()
			r.UpdatedAt = time.Now()
			r.IncrementVersion()
			return nil
		}
	}

	return shared.NewDomainError("ITEM_NOT_FOUND", "Return item not found")
}

// SetReason sets the overall return reason
func (r *PurchaseReturn) SetReason(reason string) {
	r.Reason = reason
	r.UpdatedAt = time.Now()
	r.IncrementVersion()
}

// SetRemark sets the return remark
func (r *PurchaseReturn) SetRemark(remark string) {
	r.Remark = remark
	r.UpdatedAt = time.Now()
	r.IncrementVersion()
}

// SetWarehouse sets the warehouse from which goods will be shipped
// Allowed in DRAFT, PENDING, or APPROVED status (before shipping)
func (r *PurchaseReturn) SetWarehouse(warehouseID uuid.UUID) error {
	if r.Status != PurchaseReturnStatusDraft && r.Status != PurchaseReturnStatusPending && r.Status != PurchaseReturnStatusApproved {
		return shared.NewDomainError("INVALID_STATE", "Cannot set warehouse for return in current status")
	}
	if warehouseID == uuid.Nil {
		return shared.NewDomainError("INVALID_WAREHOUSE", "Warehouse ID cannot be empty")
	}

	r.WarehouseID = &warehouseID
	r.UpdatedAt = time.Now()
	r.IncrementVersion()

	return nil
}

// Submit submits the return for approval
// Transitions from DRAFT to PENDING
func (r *PurchaseReturn) Submit() error {
	if !r.Status.CanTransitionTo(PurchaseReturnStatusPending) {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot submit return in %s status", r.Status))
	}
	if len(r.Items) == 0 {
		return shared.NewDomainError("NO_ITEMS", "Cannot submit return without items")
	}
	if r.TotalRefund.LessThanOrEqual(decimal.Zero) {
		return shared.NewDomainError("INVALID_AMOUNT", "Return total refund must be positive")
	}

	now := time.Now()
	r.Status = PurchaseReturnStatusPending
	r.SubmittedAt = &now
	r.UpdatedAt = now
	r.IncrementVersion()

	r.AddDomainEvent(NewPurchaseReturnSubmittedEvent(r))

	return nil
}

// Approve approves the return
// Transitions from PENDING to APPROVED
func (r *PurchaseReturn) Approve(approverID uuid.UUID, note string) error {
	if !r.Status.CanTransitionTo(PurchaseReturnStatusApproved) {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot approve return in %s status", r.Status))
	}
	if approverID == uuid.Nil {
		return shared.NewDomainError("INVALID_APPROVER", "Approver ID cannot be empty")
	}

	now := time.Now()
	r.Status = PurchaseReturnStatusApproved
	r.ApprovedAt = &now
	r.ApprovedBy = &approverID
	r.ApprovalNote = note
	r.UpdatedAt = now
	r.IncrementVersion()

	r.AddDomainEvent(NewPurchaseReturnApprovedEvent(r))

	return nil
}

// Reject rejects the return
// Transitions from PENDING to REJECTED
func (r *PurchaseReturn) Reject(rejecterID uuid.UUID, reason string) error {
	if !r.Status.CanTransitionTo(PurchaseReturnStatusRejected) {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot reject return in %s status", r.Status))
	}
	if rejecterID == uuid.Nil {
		return shared.NewDomainError("INVALID_REJECTER", "Rejecter ID cannot be empty")
	}
	if reason == "" {
		return shared.NewDomainError("INVALID_REASON", "Rejection reason is required")
	}

	now := time.Now()
	r.Status = PurchaseReturnStatusRejected
	r.RejectedAt = &now
	r.RejectedBy = &rejecterID
	r.RejectionReason = reason
	r.UpdatedAt = now
	r.IncrementVersion()

	r.AddDomainEvent(NewPurchaseReturnRejectedEvent(r))

	return nil
}

// Ship marks the return as shipped to the supplier
// Transitions from APPROVED to SHIPPED
// This also triggers inventory deduction
func (r *PurchaseReturn) Ship(shipperID uuid.UUID, note string, trackingNumber string) error {
	if !r.Status.CanTransitionTo(PurchaseReturnStatusShipped) {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot ship return in %s status", r.Status))
	}
	if r.WarehouseID == nil {
		return shared.NewDomainError("NO_WAREHOUSE", "Warehouse must be set before shipping")
	}

	now := time.Now()

	// Mark all items as shipped with their full return quantity
	for idx := range r.Items {
		if err := r.Items[idx].MarkShipped(r.Items[idx].ReturnQuantity); err != nil {
			return err
		}
	}

	r.Status = PurchaseReturnStatusShipped
	r.ShippedAt = &now
	if shipperID != uuid.Nil {
		r.ShippedBy = &shipperID
	}
	r.ShippingNote = note
	r.TrackingNumber = trackingNumber
	r.UpdatedAt = now
	r.IncrementVersion()

	r.AddDomainEvent(NewPurchaseReturnShippedEvent(r))

	return nil
}

// Complete marks the return as completed
// This should be called after supplier has confirmed receipt
// Transitions from SHIPPED to COMPLETED
func (r *PurchaseReturn) Complete() error {
	if !r.Status.CanTransitionTo(PurchaseReturnStatusCompleted) {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot complete return in %s status", r.Status))
	}

	now := time.Now()
	r.Status = PurchaseReturnStatusCompleted
	r.CompletedAt = &now
	r.UpdatedAt = now
	r.IncrementVersion()

	r.AddDomainEvent(NewPurchaseReturnCompletedEvent(r))

	return nil
}

// Cancel cancels the return
// Allowed in DRAFT, PENDING, or APPROVED status (before shipping)
func (r *PurchaseReturn) Cancel(reason string) error {
	if !r.Status.CanTransitionTo(PurchaseReturnStatusCancelled) {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot cancel return in %s status", r.Status))
	}
	if reason == "" {
		return shared.NewDomainError("INVALID_REASON", "Cancel reason is required")
	}

	wasApproved := r.Status == PurchaseReturnStatusApproved
	now := time.Now()
	r.Status = PurchaseReturnStatusCancelled
	r.CancelledAt = &now
	r.CancelReason = reason
	r.UpdatedAt = now
	r.IncrementVersion()

	r.AddDomainEvent(NewPurchaseReturnCancelledEvent(r, wasApproved))

	return nil
}

// recalculateTotalRefund recalculates the total refund amount
func (r *PurchaseReturn) recalculateTotalRefund() {
	total := decimal.Zero
	for _, item := range r.Items {
		total = total.Add(item.RefundAmount)
	}
	r.TotalRefund = total
}

// GetTotalRefundMoney returns total refund as Money
func (r *PurchaseReturn) GetTotalRefundMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(r.TotalRefund)
}

// ItemCount returns the number of items in the return
func (r *PurchaseReturn) ItemCount() int {
	return len(r.Items)
}

// TotalReturnQuantity returns the sum of all item return quantities
func (r *PurchaseReturn) TotalReturnQuantity() decimal.Decimal {
	total := decimal.Zero
	for _, item := range r.Items {
		total = total.Add(item.ReturnQuantity)
	}
	return total
}

// TotalShippedQuantity returns the sum of all shipped quantities
func (r *PurchaseReturn) TotalShippedQuantity() decimal.Decimal {
	total := decimal.Zero
	for _, item := range r.Items {
		total = total.Add(item.ShippedQuantity)
	}
	return total
}

// IsDraft returns true if return is in draft status
func (r *PurchaseReturn) IsDraft() bool {
	return r.Status == PurchaseReturnStatusDraft
}

// IsPending returns true if return is pending approval
func (r *PurchaseReturn) IsPending() bool {
	return r.Status == PurchaseReturnStatusPending
}

// IsApproved returns true if return is approved
func (r *PurchaseReturn) IsApproved() bool {
	return r.Status == PurchaseReturnStatusApproved
}

// IsRejected returns true if return is rejected
func (r *PurchaseReturn) IsRejected() bool {
	return r.Status == PurchaseReturnStatusRejected
}

// IsShipped returns true if return has been shipped
func (r *PurchaseReturn) IsShipped() bool {
	return r.Status == PurchaseReturnStatusShipped
}

// IsCompleted returns true if return is completed
func (r *PurchaseReturn) IsCompleted() bool {
	return r.Status == PurchaseReturnStatusCompleted
}

// IsCancelled returns true if return is cancelled
func (r *PurchaseReturn) IsCancelled() bool {
	return r.Status == PurchaseReturnStatusCancelled
}

// IsTerminal returns true if return is in a terminal state
func (r *PurchaseReturn) IsTerminal() bool {
	return r.IsCompleted() || r.IsCancelled() || r.IsRejected()
}

// CanModify returns true if the return can be modified
func (r *PurchaseReturn) CanModify() bool {
	return r.IsDraft()
}

// CanShip returns true if the return can be shipped
func (r *PurchaseReturn) CanShip() bool {
	return r.IsApproved() && r.WarehouseID != nil
}

// GetItem returns an item by its ID
func (r *PurchaseReturn) GetItem(itemID uuid.UUID) *PurchaseReturnItem {
	for idx := range r.Items {
		if r.Items[idx].ID == itemID {
			return &r.Items[idx]
		}
	}
	return nil
}

// GetItemByOrderItem returns an item by its original order item ID
func (r *PurchaseReturn) GetItemByOrderItem(orderItemID uuid.UUID) *PurchaseReturnItem {
	for idx := range r.Items {
		if r.Items[idx].PurchaseOrderItemID == orderItemID {
			return &r.Items[idx]
		}
	}
	return nil
}
