package trade

import (
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PurchaseOrderStatus represents the status of a purchase order
type PurchaseOrderStatus string

const (
	PurchaseOrderStatusDraft           PurchaseOrderStatus = "DRAFT"
	PurchaseOrderStatusConfirmed       PurchaseOrderStatus = "CONFIRMED"
	PurchaseOrderStatusPartialReceived PurchaseOrderStatus = "PARTIAL_RECEIVED"
	PurchaseOrderStatusCompleted       PurchaseOrderStatus = "COMPLETED"
	PurchaseOrderStatusCancelled       PurchaseOrderStatus = "CANCELLED"
)

// IsValid checks if the status is a valid PurchaseOrderStatus
func (s PurchaseOrderStatus) IsValid() bool {
	switch s {
	case PurchaseOrderStatusDraft, PurchaseOrderStatusConfirmed, PurchaseOrderStatusPartialReceived,
		PurchaseOrderStatusCompleted, PurchaseOrderStatusCancelled:
		return true
	}
	return false
}

// String returns the string representation of PurchaseOrderStatus
func (s PurchaseOrderStatus) String() string {
	return string(s)
}

// CanTransitionTo checks if the status can transition to the target status
func (s PurchaseOrderStatus) CanTransitionTo(target PurchaseOrderStatus) bool {
	switch s {
	case PurchaseOrderStatusDraft:
		return target == PurchaseOrderStatusConfirmed || target == PurchaseOrderStatusCancelled
	case PurchaseOrderStatusConfirmed:
		return target == PurchaseOrderStatusPartialReceived || target == PurchaseOrderStatusCompleted || target == PurchaseOrderStatusCancelled
	case PurchaseOrderStatusPartialReceived:
		return target == PurchaseOrderStatusPartialReceived || target == PurchaseOrderStatusCompleted
	case PurchaseOrderStatusCompleted, PurchaseOrderStatusCancelled:
		return false // Terminal states
	}
	return false
}

// CanReceive returns true if receiving goods is allowed in this status
func (s PurchaseOrderStatus) CanReceive() bool {
	return s == PurchaseOrderStatusConfirmed || s == PurchaseOrderStatusPartialReceived
}

// PurchaseOrderItem represents a line item in a purchase order
type PurchaseOrderItem struct {
	ID               uuid.UUID       `gorm:"type:uuid;primary_key"`
	OrderID          uuid.UUID       `gorm:"type:uuid;not null;index"`
	ProductID        uuid.UUID       `gorm:"type:uuid;not null"`
	ProductName      string          `gorm:"type:varchar(200);not null"`
	ProductCode      string          `gorm:"type:varchar(50);not null"`
	OrderedQuantity  decimal.Decimal `gorm:"type:decimal(18,4);not null"`           // Quantity ordered (in order unit)
	ReceivedQuantity decimal.Decimal `gorm:"type:decimal(18,4);not null"`           // Quantity already received (in order unit)
	UnitCost         decimal.Decimal `gorm:"type:decimal(18,4);not null"`           // Cost per unit
	Amount           decimal.Decimal `gorm:"type:decimal(18,4);not null"`           // OrderedQuantity * UnitCost
	Unit             string          `gorm:"type:varchar(20);not null"`             // Unit of measure (may be auxiliary unit)
	ConversionRate   decimal.Decimal `gorm:"type:decimal(18,6);not null;default:1"` // Conversion rate to base unit
	BaseQuantity     decimal.Decimal `gorm:"type:decimal(18,4);not null"`           // Ordered quantity in base units (for inventory)
	BaseUnit         string          `gorm:"type:varchar(20);not null"`             // Base unit code
	Remark           string          `gorm:"type:varchar(500)"`
	CreatedAt        time.Time       `gorm:"not null"`
	UpdatedAt        time.Time       `gorm:"not null"`
}

// TableName returns the table name for GORM
func (PurchaseOrderItem) TableName() string {
	return "purchase_order_items"
}

// NewPurchaseOrderItem creates a new purchase order item
// Parameters:
//   - orderID: the parent order ID
//   - productID: the product ID
//   - productName, productCode: product display info
//   - unit: the unit of measure (may be auxiliary unit)
//   - baseUnit: the base unit code for the product
//   - quantity: quantity in the order unit
//   - conversionRate: conversion rate from order unit to base unit (1 if using base unit)
//   - unitCost: cost per order unit
func NewPurchaseOrderItem(orderID, productID uuid.UUID, productName, productCode, unit, baseUnit string, quantity, conversionRate decimal.Decimal, unitCost valueobject.Money) (*PurchaseOrderItem, error) {
	if productID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_PRODUCT", "Product ID cannot be empty")
	}
	if productName == "" {
		return nil, shared.NewDomainError("INVALID_PRODUCT_NAME", "Product name cannot be empty")
	}
	if quantity.LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("INVALID_QUANTITY", "Quantity must be positive")
	}
	if unitCost.Amount().IsNegative() {
		return nil, shared.NewDomainError("INVALID_COST", "Unit cost cannot be negative")
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
	amount := quantity.Mul(unitCost.Amount())
	// Calculate base quantity: quantity * conversionRate
	baseQuantity := quantity.Mul(conversionRate).Round(4)

	return &PurchaseOrderItem{
		ID:               uuid.New(),
		OrderID:          orderID,
		ProductID:        productID,
		ProductName:      productName,
		ProductCode:      productCode,
		OrderedQuantity:  quantity,
		ReceivedQuantity: decimal.Zero,
		UnitCost:         unitCost.Amount(),
		Amount:           amount,
		Unit:             unit,
		ConversionRate:   conversionRate,
		BaseQuantity:     baseQuantity,
		BaseUnit:         baseUnit,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

// UpdateQuantity updates the item ordered quantity and recalculates the amount and base quantity
func (i *PurchaseOrderItem) UpdateQuantity(quantity decimal.Decimal) error {
	if quantity.LessThanOrEqual(decimal.Zero) {
		return shared.NewDomainError("INVALID_QUANTITY", "Quantity must be positive")
	}
	if quantity.LessThan(i.ReceivedQuantity) {
		return shared.NewDomainError("INVALID_QUANTITY", "Ordered quantity cannot be less than received quantity")
	}

	i.OrderedQuantity = quantity
	i.Amount = quantity.Mul(i.UnitCost)
	i.BaseQuantity = quantity.Mul(i.ConversionRate).Round(4)
	i.UpdatedAt = time.Now()

	return nil
}

// UpdateUnitCost updates the unit cost and recalculates the amount
func (i *PurchaseOrderItem) UpdateUnitCost(unitCost valueobject.Money) error {
	if unitCost.Amount().IsNegative() {
		return shared.NewDomainError("INVALID_COST", "Unit cost cannot be negative")
	}

	i.UnitCost = unitCost.Amount()
	i.Amount = i.OrderedQuantity.Mul(i.UnitCost)
	i.UpdatedAt = time.Now()

	return nil
}

// SetRemark sets the remark for the item
func (i *PurchaseOrderItem) SetRemark(remark string) {
	i.Remark = remark
	i.UpdatedAt = time.Now()
}

// RemainingQuantity returns the quantity still to be received
func (i *PurchaseOrderItem) RemainingQuantity() decimal.Decimal {
	remaining := i.OrderedQuantity.Sub(i.ReceivedQuantity)
	if remaining.IsNegative() {
		return decimal.Zero
	}
	return remaining
}

// IsFullyReceived returns true if all ordered quantity has been received
func (i *PurchaseOrderItem) IsFullyReceived() bool {
	return i.ReceivedQuantity.GreaterThanOrEqual(i.OrderedQuantity)
}

// CanReceive returns true if more goods can be received for this item
func (i *PurchaseOrderItem) CanReceive() bool {
	return i.ReceivedQuantity.LessThan(i.OrderedQuantity)
}

// AddReceivedQuantity adds to the received quantity
func (i *PurchaseOrderItem) AddReceivedQuantity(quantity decimal.Decimal) error {
	if quantity.LessThanOrEqual(decimal.Zero) {
		return shared.NewDomainError("INVALID_QUANTITY", "Receive quantity must be positive")
	}

	newReceived := i.ReceivedQuantity.Add(quantity)
	if newReceived.GreaterThan(i.OrderedQuantity) {
		return shared.NewDomainError("QUANTITY_EXCEEDED", fmt.Sprintf("Cannot receive %s, only %s remaining", quantity.String(), i.RemainingQuantity().String()))
	}

	i.ReceivedQuantity = newReceived
	i.UpdatedAt = time.Now()

	return nil
}

// GetAmountMoney returns the amount as Money value object
func (i *PurchaseOrderItem) GetAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(i.Amount)
}

// GetUnitCostMoney returns the unit cost as Money value object
func (i *PurchaseOrderItem) GetUnitCostMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(i.UnitCost)
}

// ReceiveItem represents a single item being received in a receiving operation
type ReceiveItem struct {
	ProductID   uuid.UUID       `json:"product_id"`
	Quantity    decimal.Decimal `json:"quantity"`
	UnitCost    decimal.Decimal `json:"unit_cost,omitempty"` // Optional: override cost if different
	BatchNumber string          `json:"batch_number,omitempty"`
	ExpiryDate  *time.Time      `json:"expiry_date,omitempty"`
}

// PurchaseOrder represents a purchase order aggregate root
// It manages the lifecycle of a supplier order from creation to completion
type PurchaseOrder struct {
	shared.TenantAggregateRoot
	OrderNumber    string              `gorm:"type:varchar(50);not null;uniqueIndex:idx_purchase_order_tenant_number,priority:2"`
	SupplierID     uuid.UUID           `gorm:"type:uuid;not null;index"`
	SupplierName   string              `gorm:"type:varchar(200);not null"`
	WarehouseID    *uuid.UUID          `gorm:"type:uuid;index"` // Target warehouse for receiving
	Items          []PurchaseOrderItem `gorm:"foreignKey:OrderID;references:ID"`
	TotalAmount    decimal.Decimal     `gorm:"type:decimal(18,4);not null;default:0"` // Sum of all items
	DiscountAmount decimal.Decimal     `gorm:"type:decimal(18,4);not null;default:0"` // Order-level discount
	PayableAmount  decimal.Decimal     `gorm:"type:decimal(18,4);not null;default:0"` // TotalAmount - DiscountAmount
	Status         PurchaseOrderStatus `gorm:"type:varchar(20);not null;default:'DRAFT'"`
	Remark         string              `gorm:"type:text"`
	ConfirmedAt    *time.Time          `gorm:"index"`
	CompletedAt    *time.Time
	CancelledAt    *time.Time
	CancelReason   string `gorm:"type:varchar(500)"`
}

// TableName returns the table name for GORM
func (PurchaseOrder) TableName() string {
	return "purchase_orders"
}

// NewPurchaseOrder creates a new purchase order
func NewPurchaseOrder(tenantID uuid.UUID, orderNumber string, supplierID uuid.UUID, supplierName string) (*PurchaseOrder, error) {
	if orderNumber == "" {
		return nil, shared.NewDomainError("INVALID_ORDER_NUMBER", "Order number cannot be empty")
	}
	if len(orderNumber) > 50 {
		return nil, shared.NewDomainError("INVALID_ORDER_NUMBER", "Order number cannot exceed 50 characters")
	}
	if supplierID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_SUPPLIER", "Supplier ID cannot be empty")
	}
	if supplierName == "" {
		return nil, shared.NewDomainError("INVALID_SUPPLIER_NAME", "Supplier name cannot be empty")
	}

	order := &PurchaseOrder{
		TenantAggregateRoot: shared.NewTenantAggregateRoot(tenantID),
		OrderNumber:         orderNumber,
		SupplierID:          supplierID,
		SupplierName:        supplierName,
		Items:               make([]PurchaseOrderItem, 0),
		TotalAmount:         decimal.Zero,
		DiscountAmount:      decimal.Zero,
		PayableAmount:       decimal.Zero,
		Status:              PurchaseOrderStatusDraft,
	}

	order.AddDomainEvent(NewPurchaseOrderCreatedEvent(order))

	return order, nil
}

// AddItem adds a new item to the order
// Only allowed in DRAFT status
// Parameters:
//   - productID: the product ID
//   - productName, productCode: product display info
//   - unit: the unit of measure (may be auxiliary unit)
//   - baseUnit: the base unit code for the product
//   - quantity: quantity in the order unit
//   - conversionRate: conversion rate from order unit to base unit (1 if using base unit)
//   - unitCost: cost per order unit
func (o *PurchaseOrder) AddItem(productID uuid.UUID, productName, productCode, unit, baseUnit string, quantity, conversionRate decimal.Decimal, unitCost valueobject.Money) (*PurchaseOrderItem, error) {
	if o.Status != PurchaseOrderStatusDraft {
		return nil, shared.NewDomainError("INVALID_STATE", "Cannot add items to a non-draft order")
	}

	// Check if product already exists in order
	for _, item := range o.Items {
		if item.ProductID == productID {
			return nil, shared.NewDomainError("DUPLICATE_PRODUCT", "Product already exists in order, update quantity instead")
		}
	}

	item, err := NewPurchaseOrderItem(o.ID, productID, productName, productCode, unit, baseUnit, quantity, conversionRate, unitCost)
	if err != nil {
		return nil, err
	}

	o.Items = append(o.Items, *item)
	o.recalculateTotals()
	o.UpdatedAt = time.Now()
	o.IncrementVersion()

	return item, nil
}

// UpdateItemQuantity updates the ordered quantity of an existing item
// Only allowed in DRAFT status
func (o *PurchaseOrder) UpdateItemQuantity(itemID uuid.UUID, quantity decimal.Decimal) error {
	if o.Status != PurchaseOrderStatusDraft {
		return shared.NewDomainError("INVALID_STATE", "Cannot update items in a non-draft order")
	}

	for idx := range o.Items {
		if o.Items[idx].ID == itemID {
			if err := o.Items[idx].UpdateQuantity(quantity); err != nil {
				return err
			}
			o.recalculateTotals()
			o.UpdatedAt = time.Now()
			o.IncrementVersion()
			return nil
		}
	}

	return shared.NewDomainError("ITEM_NOT_FOUND", "Order item not found")
}

// UpdateItemCost updates the unit cost of an existing item
// Only allowed in DRAFT status
func (o *PurchaseOrder) UpdateItemCost(itemID uuid.UUID, unitCost valueobject.Money) error {
	if o.Status != PurchaseOrderStatusDraft {
		return shared.NewDomainError("INVALID_STATE", "Cannot update items in a non-draft order")
	}

	for idx := range o.Items {
		if o.Items[idx].ID == itemID {
			if err := o.Items[idx].UpdateUnitCost(unitCost); err != nil {
				return err
			}
			o.recalculateTotals()
			o.UpdatedAt = time.Now()
			o.IncrementVersion()
			return nil
		}
	}

	return shared.NewDomainError("ITEM_NOT_FOUND", "Order item not found")
}

// RemoveItem removes an item from the order
// Only allowed in DRAFT status
func (o *PurchaseOrder) RemoveItem(itemID uuid.UUID) error {
	if o.Status != PurchaseOrderStatusDraft {
		return shared.NewDomainError("INVALID_STATE", "Cannot remove items from a non-draft order")
	}

	for idx, item := range o.Items {
		if item.ID == itemID {
			o.Items = append(o.Items[:idx], o.Items[idx+1:]...)
			o.recalculateTotals()
			o.UpdatedAt = time.Now()
			o.IncrementVersion()
			return nil
		}
	}

	return shared.NewDomainError("ITEM_NOT_FOUND", "Order item not found")
}

// ApplyDiscount applies a discount to the order
// Only allowed in DRAFT status
func (o *PurchaseOrder) ApplyDiscount(discount valueobject.Money) error {
	if o.Status != PurchaseOrderStatusDraft {
		return shared.NewDomainError("INVALID_STATE", "Cannot apply discount to a non-draft order")
	}
	if discount.Amount().IsNegative() {
		return shared.NewDomainError("INVALID_DISCOUNT", "Discount cannot be negative")
	}
	if discount.Amount().GreaterThan(o.TotalAmount) {
		return shared.NewDomainError("INVALID_DISCOUNT", "Discount cannot exceed total amount")
	}

	o.DiscountAmount = discount.Amount()
	o.PayableAmount = o.TotalAmount.Sub(o.DiscountAmount)
	o.UpdatedAt = time.Now()
	o.IncrementVersion()

	return nil
}

// SetRemark sets the order remark
func (o *PurchaseOrder) SetRemark(remark string) {
	o.Remark = remark
	o.UpdatedAt = time.Now()
	o.IncrementVersion()
}

// SetWarehouse sets the target warehouse for receiving
// Only allowed in DRAFT or CONFIRMED status
func (o *PurchaseOrder) SetWarehouse(warehouseID uuid.UUID) error {
	if o.Status != PurchaseOrderStatusDraft && o.Status != PurchaseOrderStatusConfirmed {
		return shared.NewDomainError("INVALID_STATE", "Cannot set warehouse for order in current status")
	}
	if warehouseID == uuid.Nil {
		return shared.NewDomainError("INVALID_WAREHOUSE", "Warehouse ID cannot be empty")
	}

	o.WarehouseID = &warehouseID
	o.UpdatedAt = time.Now()
	o.IncrementVersion()

	return nil
}

// Confirm confirms the order, transitioning from DRAFT to CONFIRMED
// Requires at least one item in the order
func (o *PurchaseOrder) Confirm() error {
	if !o.Status.CanTransitionTo(PurchaseOrderStatusConfirmed) {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot confirm order in %s status", o.Status))
	}
	if len(o.Items) == 0 {
		return shared.NewDomainError("NO_ITEMS", "Cannot confirm order without items")
	}
	if o.PayableAmount.LessThanOrEqual(decimal.Zero) {
		return shared.NewDomainError("INVALID_AMOUNT", "Order payable amount must be positive")
	}

	now := time.Now()
	o.Status = PurchaseOrderStatusConfirmed
	o.ConfirmedAt = &now
	o.UpdatedAt = now
	o.IncrementVersion()

	o.AddDomainEvent(NewPurchaseOrderConfirmedEvent(o))

	return nil
}

// Receive processes receipt of goods for one or more items
// Only allowed in CONFIRMED or PARTIAL_RECEIVED status
// Returns the list of items that were updated and their received quantities
func (o *PurchaseOrder) Receive(receiveItems []ReceiveItem) ([]ReceivedItemInfo, error) {
	if !o.Status.CanReceive() {
		return nil, shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot receive goods for order in %s status", o.Status))
	}
	if len(receiveItems) == 0 {
		return nil, shared.NewDomainError("NO_ITEMS", "Receive items cannot be empty")
	}
	if o.WarehouseID == nil {
		return nil, shared.NewDomainError("NO_WAREHOUSE", "Warehouse must be set before receiving")
	}

	receivedInfos := make([]ReceivedItemInfo, 0, len(receiveItems))

	// Process each receive item
	for _, ri := range receiveItems {
		if ri.Quantity.LessThanOrEqual(decimal.Zero) {
			return nil, shared.NewDomainError("INVALID_QUANTITY", fmt.Sprintf("Receive quantity for product %s must be positive", ri.ProductID))
		}

		// Find the matching order item
		var found bool
		for idx := range o.Items {
			if o.Items[idx].ProductID == ri.ProductID {
				// Check if can receive this quantity
				if err := o.Items[idx].AddReceivedQuantity(ri.Quantity); err != nil {
					return nil, err
				}

				// Use the item's unit cost unless overridden
				unitCost := o.Items[idx].UnitCost
				if !ri.UnitCost.IsZero() {
					unitCost = ri.UnitCost
				}

				receivedInfos = append(receivedInfos, ReceivedItemInfo{
					ItemID:      o.Items[idx].ID,
					ProductID:   ri.ProductID,
					ProductName: o.Items[idx].ProductName,
					ProductCode: o.Items[idx].ProductCode,
					Quantity:    ri.Quantity,
					UnitCost:    unitCost,
					Unit:        o.Items[idx].Unit,
					BatchNumber: ri.BatchNumber,
					ExpiryDate:  ri.ExpiryDate,
				})

				found = true
				break
			}
		}

		if !found {
			return nil, shared.NewDomainError("ITEM_NOT_FOUND", fmt.Sprintf("Product %s not found in order", ri.ProductID))
		}
	}

	// Update order status based on whether all items are fully received
	if o.isAllItemsReceived() {
		o.Status = PurchaseOrderStatusCompleted
		now := time.Now()
		o.CompletedAt = &now
	} else {
		o.Status = PurchaseOrderStatusPartialReceived
	}

	o.UpdatedAt = time.Now()
	o.IncrementVersion()

	o.AddDomainEvent(NewPurchaseOrderReceivedEvent(o, receivedInfos))

	return receivedInfos, nil
}

// Cancel cancels the order
// Allowed only in DRAFT or CONFIRMED status (no goods received yet)
func (o *PurchaseOrder) Cancel(reason string) error {
	if !o.Status.CanTransitionTo(PurchaseOrderStatusCancelled) {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot cancel order in %s status", o.Status))
	}
	if reason == "" {
		return shared.NewDomainError("INVALID_REASON", "Cancel reason is required")
	}

	// Check if any goods have been received
	if o.hasReceivedAnyGoods() {
		return shared.NewDomainError("ALREADY_RECEIVED", "Cannot cancel order after goods have been received")
	}

	wasConfirmed := o.Status == PurchaseOrderStatusConfirmed
	now := time.Now()
	o.Status = PurchaseOrderStatusCancelled
	o.CancelledAt = &now
	o.CancelReason = reason
	o.UpdatedAt = now
	o.IncrementVersion()

	o.AddDomainEvent(NewPurchaseOrderCancelledEvent(o, wasConfirmed))

	return nil
}

// recalculateTotals recalculates the order totals
func (o *PurchaseOrder) recalculateTotals() {
	total := decimal.Zero
	for _, item := range o.Items {
		total = total.Add(item.Amount)
	}
	o.TotalAmount = total
	o.PayableAmount = o.TotalAmount.Sub(o.DiscountAmount)

	// Ensure payable doesn't go negative if discount was set before items
	if o.PayableAmount.IsNegative() {
		o.DiscountAmount = o.TotalAmount
		o.PayableAmount = decimal.Zero
	}
}

// isAllItemsReceived checks if all items have been fully received
func (o *PurchaseOrder) isAllItemsReceived() bool {
	for _, item := range o.Items {
		if !item.IsFullyReceived() {
			return false
		}
	}
	return len(o.Items) > 0
}

// hasReceivedAnyGoods checks if any goods have been received
func (o *PurchaseOrder) hasReceivedAnyGoods() bool {
	for _, item := range o.Items {
		if item.ReceivedQuantity.GreaterThan(decimal.Zero) {
			return true
		}
	}
	return false
}

// TotalReceivedQuantity returns the total quantity of all received items
func (o *PurchaseOrder) TotalReceivedQuantity() decimal.Decimal {
	total := decimal.Zero
	for _, item := range o.Items {
		total = total.Add(item.ReceivedQuantity)
	}
	return total
}

// TotalOrderedQuantity returns the total ordered quantity
func (o *PurchaseOrder) TotalOrderedQuantity() decimal.Decimal {
	total := decimal.Zero
	for _, item := range o.Items {
		total = total.Add(item.OrderedQuantity)
	}
	return total
}

// TotalRemainingQuantity returns the total quantity still to be received
func (o *PurchaseOrder) TotalRemainingQuantity() decimal.Decimal {
	total := decimal.Zero
	for _, item := range o.Items {
		total = total.Add(item.RemainingQuantity())
	}
	return total
}

// GetTotalAmountMoney returns total amount as Money
func (o *PurchaseOrder) GetTotalAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(o.TotalAmount)
}

// GetDiscountAmountMoney returns discount amount as Money
func (o *PurchaseOrder) GetDiscountAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(o.DiscountAmount)
}

// GetPayableAmountMoney returns payable amount as Money
func (o *PurchaseOrder) GetPayableAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(o.PayableAmount)
}

// ItemCount returns the number of items in the order
func (o *PurchaseOrder) ItemCount() int {
	return len(o.Items)
}

// IsDraft returns true if order is in draft status
func (o *PurchaseOrder) IsDraft() bool {
	return o.Status == PurchaseOrderStatusDraft
}

// IsConfirmed returns true if order is confirmed
func (o *PurchaseOrder) IsConfirmed() bool {
	return o.Status == PurchaseOrderStatusConfirmed
}

// IsPartialReceived returns true if order is partially received
func (o *PurchaseOrder) IsPartialReceived() bool {
	return o.Status == PurchaseOrderStatusPartialReceived
}

// IsCompleted returns true if order is completed
func (o *PurchaseOrder) IsCompleted() bool {
	return o.Status == PurchaseOrderStatusCompleted
}

// IsCancelled returns true if order is cancelled
func (o *PurchaseOrder) IsCancelled() bool {
	return o.Status == PurchaseOrderStatusCancelled
}

// IsTerminal returns true if order is in a terminal state (completed or cancelled)
func (o *PurchaseOrder) IsTerminal() bool {
	return o.IsCompleted() || o.IsCancelled()
}

// CanModify returns true if the order can be modified (items, discount, etc.)
func (o *PurchaseOrder) CanModify() bool {
	return o.IsDraft()
}

// CanReceiveGoods returns true if the order can receive goods
func (o *PurchaseOrder) CanReceiveGoods() bool {
	return o.Status.CanReceive()
}

// GetItem returns an item by its ID
func (o *PurchaseOrder) GetItem(itemID uuid.UUID) *PurchaseOrderItem {
	for idx := range o.Items {
		if o.Items[idx].ID == itemID {
			return &o.Items[idx]
		}
	}
	return nil
}

// GetItemByProduct returns an item by product ID
func (o *PurchaseOrder) GetItemByProduct(productID uuid.UUID) *PurchaseOrderItem {
	for idx := range o.Items {
		if o.Items[idx].ProductID == productID {
			return &o.Items[idx]
		}
	}
	return nil
}

// GetReceivableItems returns items that can still receive goods
func (o *PurchaseOrder) GetReceivableItems() []PurchaseOrderItem {
	items := make([]PurchaseOrderItem, 0)
	for _, item := range o.Items {
		if item.CanReceive() {
			items = append(items, item)
		}
	}
	return items
}

// ReceiveProgress returns the receiving progress as a percentage (0-100)
func (o *PurchaseOrder) ReceiveProgress() decimal.Decimal {
	totalOrdered := o.TotalOrderedQuantity()
	if totalOrdered.IsZero() {
		return decimal.Zero
	}
	totalReceived := o.TotalReceivedQuantity()
	return totalReceived.Div(totalOrdered).Mul(decimal.NewFromInt(100)).Round(2)
}
