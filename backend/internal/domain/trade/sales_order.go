package trade

import (
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// OrderStatus represents the status of a sales order
type OrderStatus string

const (
	OrderStatusDraft     OrderStatus = "DRAFT"
	OrderStatusConfirmed OrderStatus = "CONFIRMED"
	OrderStatusShipped   OrderStatus = "SHIPPED"
	OrderStatusCompleted OrderStatus = "COMPLETED"
	OrderStatusCancelled OrderStatus = "CANCELLED"
)

// IsValid checks if the status is a valid OrderStatus
func (s OrderStatus) IsValid() bool {
	switch s {
	case OrderStatusDraft, OrderStatusConfirmed, OrderStatusShipped, OrderStatusCompleted, OrderStatusCancelled:
		return true
	}
	return false
}

// String returns the string representation of OrderStatus
func (s OrderStatus) String() string {
	return string(s)
}

// CanTransitionTo checks if the status can transition to the target status
func (s OrderStatus) CanTransitionTo(target OrderStatus) bool {
	switch s {
	case OrderStatusDraft:
		return target == OrderStatusConfirmed || target == OrderStatusCancelled
	case OrderStatusConfirmed:
		return target == OrderStatusShipped || target == OrderStatusCancelled
	case OrderStatusShipped:
		return target == OrderStatusCompleted
	case OrderStatusCompleted, OrderStatusCancelled:
		return false // Terminal states
	}
	return false
}

// SalesOrderItem represents a line item in a sales order
type SalesOrderItem struct {
	ID             uuid.UUID
	OrderID        uuid.UUID
	ProductID      uuid.UUID
	ProductName    string
	ProductCode    string
	Quantity       decimal.Decimal // Quantity in the order unit
	UnitPrice      decimal.Decimal // Price per unit
	Amount         decimal.Decimal // Quantity * UnitPrice
	Unit           string          // Unit of measure (may be auxiliary unit)
	ConversionRate decimal.Decimal // Conversion rate to base unit
	BaseQuantity   decimal.Decimal // Quantity in base units (for inventory)
	BaseUnit       string          // Base unit code
	Remark         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewSalesOrderItem creates a new sales order item
// Parameters:
//   - orderID: the parent order ID
//   - productID: the product ID
//   - productName, productCode: product display info
//   - unit: the unit of measure (may be auxiliary unit)
//   - baseUnit: the base unit code for the product
//   - quantity: quantity in the order unit
//   - conversionRate: conversion rate from order unit to base unit (1 if using base unit)
//   - unitPrice: price per order unit
func NewSalesOrderItem(orderID, productID uuid.UUID, productName, productCode, unit, baseUnit string, quantity, conversionRate decimal.Decimal, unitPrice valueobject.Money) (*SalesOrderItem, error) {
	if productID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_PRODUCT", "Product ID cannot be empty")
	}
	if productName == "" {
		return nil, shared.NewDomainError("INVALID_PRODUCT_NAME", "Product name cannot be empty")
	}
	if quantity.LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("INVALID_QUANTITY", "Quantity must be positive")
	}
	if unitPrice.Amount().IsNegative() {
		return nil, shared.NewDomainError("INVALID_PRICE", "Unit price cannot be negative")
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
	amount := quantity.Mul(unitPrice.Amount())
	// Calculate base quantity: quantity * conversionRate
	baseQuantity := quantity.Mul(conversionRate).Round(4)

	return &SalesOrderItem{
		ID:             uuid.New(),
		OrderID:        orderID,
		ProductID:      productID,
		ProductName:    productName,
		ProductCode:    productCode,
		Quantity:       quantity,
		UnitPrice:      unitPrice.Amount(),
		Amount:         amount,
		Unit:           unit,
		ConversionRate: conversionRate,
		BaseQuantity:   baseQuantity,
		BaseUnit:       baseUnit,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// UpdateQuantity updates the item quantity and recalculates the amount and base quantity
func (i *SalesOrderItem) UpdateQuantity(quantity decimal.Decimal) error {
	if quantity.LessThanOrEqual(decimal.Zero) {
		return shared.NewDomainError("INVALID_QUANTITY", "Quantity must be positive")
	}

	i.Quantity = quantity
	i.Amount = quantity.Mul(i.UnitPrice)
	i.BaseQuantity = quantity.Mul(i.ConversionRate).Round(4)
	i.UpdatedAt = time.Now()

	return nil
}

// UpdateUnitPrice updates the unit price and recalculates the amount
func (i *SalesOrderItem) UpdateUnitPrice(unitPrice valueobject.Money) error {
	if unitPrice.Amount().IsNegative() {
		return shared.NewDomainError("INVALID_PRICE", "Unit price cannot be negative")
	}

	i.UnitPrice = unitPrice.Amount()
	i.Amount = i.Quantity.Mul(i.UnitPrice)
	i.UpdatedAt = time.Now()

	return nil
}

// SetRemark sets the remark for the item
func (i *SalesOrderItem) SetRemark(remark string) {
	i.Remark = remark
	i.UpdatedAt = time.Now()
}

// GetAmountMoney returns the amount as Money value object
func (i *SalesOrderItem) GetAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(i.Amount)
}

// GetUnitPriceMoney returns the unit price as Money value object
func (i *SalesOrderItem) GetUnitPriceMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(i.UnitPrice)
}

// SalesOrder represents a sales order aggregate root
// It manages the lifecycle of a customer order from creation to completion
type SalesOrder struct {
	shared.TenantAggregateRoot
	OrderNumber    string
	CustomerID     uuid.UUID
	CustomerName   string
	WarehouseID    *uuid.UUID // Warehouse for shipment (set on confirm/ship)
	Items          []SalesOrderItem
	TotalAmount    decimal.Decimal // Sum of all items
	DiscountAmount decimal.Decimal // Order-level discount
	PayableAmount  decimal.Decimal // TotalAmount - DiscountAmount
	Status         OrderStatus
	Remark         string
	ConfirmedAt    *time.Time
	ShippedAt      *time.Time
	CompletedAt    *time.Time
	CancelledAt    *time.Time
	CancelReason   string
}

// NewSalesOrder creates a new sales order
func NewSalesOrder(tenantID uuid.UUID, orderNumber string, customerID uuid.UUID, customerName string) (*SalesOrder, error) {
	if orderNumber == "" {
		return nil, shared.NewDomainError("INVALID_ORDER_NUMBER", "Order number cannot be empty")
	}
	if len(orderNumber) > 50 {
		return nil, shared.NewDomainError("INVALID_ORDER_NUMBER", "Order number cannot exceed 50 characters")
	}
	if customerID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_CUSTOMER", "Customer ID cannot be empty")
	}
	if customerName == "" {
		return nil, shared.NewDomainError("INVALID_CUSTOMER_NAME", "Customer name cannot be empty")
	}

	order := &SalesOrder{
		TenantAggregateRoot: shared.NewTenantAggregateRoot(tenantID),
		OrderNumber:         orderNumber,
		CustomerID:          customerID,
		CustomerName:        customerName,
		Items:               make([]SalesOrderItem, 0),
		TotalAmount:         decimal.Zero,
		DiscountAmount:      decimal.Zero,
		PayableAmount:       decimal.Zero,
		Status:              OrderStatusDraft,
	}

	order.AddDomainEvent(NewSalesOrderCreatedEvent(order))

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
//   - unitPrice: price per order unit
func (o *SalesOrder) AddItem(productID uuid.UUID, productName, productCode, unit, baseUnit string, quantity, conversionRate decimal.Decimal, unitPrice valueobject.Money) (*SalesOrderItem, error) {
	if o.Status != OrderStatusDraft {
		return nil, shared.NewDomainError("INVALID_STATE", "Cannot add items to a non-draft order")
	}

	// Check if product already exists in order
	for _, item := range o.Items {
		if item.ProductID == productID {
			return nil, shared.NewDomainError("DUPLICATE_PRODUCT", "Product already exists in order, update quantity instead")
		}
	}

	item, err := NewSalesOrderItem(o.ID, productID, productName, productCode, unit, baseUnit, quantity, conversionRate, unitPrice)
	if err != nil {
		return nil, err
	}

	o.Items = append(o.Items, *item)
	o.recalculateTotals()
	o.UpdatedAt = time.Now()

	return item, nil
}

// UpdateItemQuantity updates the quantity of an existing item
// Only allowed in DRAFT status
func (o *SalesOrder) UpdateItemQuantity(itemID uuid.UUID, quantity decimal.Decimal) error {
	if o.Status != OrderStatusDraft {
		return shared.NewDomainError("INVALID_STATE", "Cannot update items in a non-draft order")
	}

	for idx := range o.Items {
		if o.Items[idx].ID == itemID {
			if err := o.Items[idx].UpdateQuantity(quantity); err != nil {
				return err
			}
			o.recalculateTotals()
			o.UpdatedAt = time.Now()
			return nil
		}
	}

	return shared.NewDomainError("ITEM_NOT_FOUND", "Order item not found")
}

// UpdateItemPrice updates the unit price of an existing item
// Only allowed in DRAFT status
func (o *SalesOrder) UpdateItemPrice(itemID uuid.UUID, unitPrice valueobject.Money) error {
	if o.Status != OrderStatusDraft {
		return shared.NewDomainError("INVALID_STATE", "Cannot update items in a non-draft order")
	}

	for idx := range o.Items {
		if o.Items[idx].ID == itemID {
			if err := o.Items[idx].UpdateUnitPrice(unitPrice); err != nil {
				return err
			}
			o.recalculateTotals()
			o.UpdatedAt = time.Now()
			return nil
		}
	}

	return shared.NewDomainError("ITEM_NOT_FOUND", "Order item not found")
}

// RemoveItem removes an item from the order
// Only allowed in DRAFT status
func (o *SalesOrder) RemoveItem(itemID uuid.UUID) error {
	if o.Status != OrderStatusDraft {
		return shared.NewDomainError("INVALID_STATE", "Cannot remove items from a non-draft order")
	}

	for idx, item := range o.Items {
		if item.ID == itemID {
			o.Items = append(o.Items[:idx], o.Items[idx+1:]...)
			o.recalculateTotals()
			o.UpdatedAt = time.Now()
			return nil
		}
	}

	return shared.NewDomainError("ITEM_NOT_FOUND", "Order item not found")
}

// ApplyDiscount applies a discount to the order
// Only allowed in DRAFT status
func (o *SalesOrder) ApplyDiscount(discount valueobject.Money) error {
	if o.Status != OrderStatusDraft {
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

	return nil
}

// SetRemark sets the order remark
func (o *SalesOrder) SetRemark(remark string) {
	o.Remark = remark
	o.UpdatedAt = time.Now()
}

// SetWarehouse sets the warehouse for the order
// Only allowed in DRAFT or CONFIRMED status
func (o *SalesOrder) SetWarehouse(warehouseID uuid.UUID) error {
	if o.Status != OrderStatusDraft && o.Status != OrderStatusConfirmed {
		return shared.NewDomainError("INVALID_STATE", "Cannot set warehouse for order in current status")
	}
	if warehouseID == uuid.Nil {
		return shared.NewDomainError("INVALID_WAREHOUSE", "Warehouse ID cannot be empty")
	}

	o.WarehouseID = &warehouseID
	o.UpdatedAt = time.Now()

	return nil
}

// Confirm confirms the order, transitioning from DRAFT to CONFIRMED
// Requires at least one item in the order
func (o *SalesOrder) Confirm() error {
	if !o.Status.CanTransitionTo(OrderStatusConfirmed) {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot confirm order in %s status", o.Status))
	}
	if len(o.Items) == 0 {
		return shared.NewDomainError("NO_ITEMS", "Cannot confirm order without items")
	}
	if o.PayableAmount.LessThanOrEqual(decimal.Zero) {
		return shared.NewDomainError("INVALID_AMOUNT", "Order payable amount must be positive")
	}

	now := time.Now()
	o.Status = OrderStatusConfirmed
	o.ConfirmedAt = &now
	o.UpdatedAt = now

	o.AddDomainEvent(NewSalesOrderConfirmedEvent(o))

	return nil
}

// Ship marks the order as shipped
// Requires warehouse to be set and stock to be locked (handled by application service)
func (o *SalesOrder) Ship() error {
	if !o.Status.CanTransitionTo(OrderStatusShipped) {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot ship order in %s status", o.Status))
	}
	if o.WarehouseID == nil {
		return shared.NewDomainError("NO_WAREHOUSE", "Warehouse must be set before shipping")
	}

	now := time.Now()
	o.Status = OrderStatusShipped
	o.ShippedAt = &now
	o.UpdatedAt = now

	o.AddDomainEvent(NewSalesOrderShippedEvent(o))

	return nil
}

// Complete marks the order as completed (delivered/received)
func (o *SalesOrder) Complete() error {
	if !o.Status.CanTransitionTo(OrderStatusCompleted) {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot complete order in %s status", o.Status))
	}

	now := time.Now()
	o.Status = OrderStatusCompleted
	o.CompletedAt = &now
	o.UpdatedAt = now

	o.AddDomainEvent(NewSalesOrderCompletedEvent(o))

	return nil
}

// Cancel cancels the order
// Allowed only in DRAFT or CONFIRMED status
// If CONFIRMED, stock locks should be released (handled by application service)
func (o *SalesOrder) Cancel(reason string) error {
	if !o.Status.CanTransitionTo(OrderStatusCancelled) {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot cancel order in %s status", o.Status))
	}
	if reason == "" {
		return shared.NewDomainError("INVALID_REASON", "Cancel reason is required")
	}

	wasConfirmed := o.Status == OrderStatusConfirmed
	now := time.Now()
	o.Status = OrderStatusCancelled
	o.CancelledAt = &now
	o.CancelReason = reason
	o.UpdatedAt = now

	o.AddDomainEvent(NewSalesOrderCancelledEvent(o, wasConfirmed))

	return nil
}

// recalculateTotals recalculates the order totals
func (o *SalesOrder) recalculateTotals() {
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

// GetTotalAmountMoney returns total amount as Money
func (o *SalesOrder) GetTotalAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(o.TotalAmount)
}

// GetDiscountAmountMoney returns discount amount as Money
func (o *SalesOrder) GetDiscountAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(o.DiscountAmount)
}

// GetPayableAmountMoney returns payable amount as Money
func (o *SalesOrder) GetPayableAmountMoney() valueobject.Money {
	return valueobject.NewMoneyCNY(o.PayableAmount)
}

// ItemCount returns the number of items in the order
func (o *SalesOrder) ItemCount() int {
	return len(o.Items)
}

// TotalQuantity returns the sum of all item quantities
func (o *SalesOrder) TotalQuantity() decimal.Decimal {
	total := decimal.Zero
	for _, item := range o.Items {
		total = total.Add(item.Quantity)
	}
	return total
}

// IsDraft returns true if order is in draft status
func (o *SalesOrder) IsDraft() bool {
	return o.Status == OrderStatusDraft
}

// IsConfirmed returns true if order is confirmed
func (o *SalesOrder) IsConfirmed() bool {
	return o.Status == OrderStatusConfirmed
}

// IsShipped returns true if order is shipped
func (o *SalesOrder) IsShipped() bool {
	return o.Status == OrderStatusShipped
}

// IsCompleted returns true if order is completed
func (o *SalesOrder) IsCompleted() bool {
	return o.Status == OrderStatusCompleted
}

// IsCancelled returns true if order is cancelled
func (o *SalesOrder) IsCancelled() bool {
	return o.Status == OrderStatusCancelled
}

// IsTerminal returns true if order is in a terminal state (completed or cancelled)
func (o *SalesOrder) IsTerminal() bool {
	return o.IsCompleted() || o.IsCancelled()
}

// CanModify returns true if the order can be modified (items, discount, etc.)
func (o *SalesOrder) CanModify() bool {
	return o.IsDraft()
}

// GetItem returns an item by its ID
func (o *SalesOrder) GetItem(itemID uuid.UUID) *SalesOrderItem {
	for idx := range o.Items {
		if o.Items[idx].ID == itemID {
			return &o.Items[idx]
		}
	}
	return nil
}

// GetItemByProduct returns an item by product ID
func (o *SalesOrder) GetItemByProduct(productID uuid.UUID) *SalesOrderItem {
	for idx := range o.Items {
		if o.Items[idx].ProductID == productID {
			return &o.Items[idx]
		}
	}
	return nil
}
