package trade

import (
	"time"

	"github.com/erp/backend/internal/domain/trade"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CreateSalesOrderRequest represents a request to create a sales order
type CreateSalesOrderRequest struct {
	CustomerID   uuid.UUID                   `json:"customer_id" binding:"required"`
	CustomerName string                      `json:"customer_name" binding:"required,min=1,max=200"`
	WarehouseID  *uuid.UUID                  `json:"warehouse_id"`
	Items        []CreateSalesOrderItemInput `json:"items"`
	Discount     *decimal.Decimal            `json:"discount"`
	Remark       string                      `json:"remark"`
}

// CreateSalesOrderItemInput represents an item in the create order request
type CreateSalesOrderItemInput struct {
	ProductID   uuid.UUID       `json:"product_id" binding:"required"`
	ProductName string          `json:"product_name" binding:"required,min=1,max=200"`
	ProductCode string          `json:"product_code" binding:"required,min=1,max=50"`
	Unit        string          `json:"unit" binding:"required,min=1,max=20"`
	Quantity    decimal.Decimal `json:"quantity" binding:"required"`
	UnitPrice   decimal.Decimal `json:"unit_price" binding:"required"`
	Remark      string          `json:"remark"`
}

// UpdateSalesOrderRequest represents a request to update a sales order (only in DRAFT status)
type UpdateSalesOrderRequest struct {
	WarehouseID *uuid.UUID       `json:"warehouse_id"`
	Discount    *decimal.Decimal `json:"discount"`
	Remark      *string          `json:"remark"`
}

// AddOrderItemRequest represents a request to add an item to an order
type AddOrderItemRequest struct {
	ProductID   uuid.UUID       `json:"product_id" binding:"required"`
	ProductName string          `json:"product_name" binding:"required,min=1,max=200"`
	ProductCode string          `json:"product_code" binding:"required,min=1,max=50"`
	Unit        string          `json:"unit" binding:"required,min=1,max=20"`
	Quantity    decimal.Decimal `json:"quantity" binding:"required"`
	UnitPrice   decimal.Decimal `json:"unit_price" binding:"required"`
	Remark      string          `json:"remark"`
}

// UpdateOrderItemRequest represents a request to update an order item
type UpdateOrderItemRequest struct {
	Quantity  *decimal.Decimal `json:"quantity"`
	UnitPrice *decimal.Decimal `json:"unit_price"`
	Remark    *string          `json:"remark"`
}

// ConfirmOrderRequest represents a request to confirm an order
type ConfirmOrderRequest struct {
	WarehouseID *uuid.UUID `json:"warehouse_id"` // Optional warehouse override
}

// ShipOrderRequest represents a request to ship an order
type ShipOrderRequest struct {
	WarehouseID *uuid.UUID `json:"warehouse_id"` // Optional warehouse override (must be set if not already)
}

// CancelOrderRequest represents a request to cancel an order
type CancelOrderRequest struct {
	Reason string `json:"reason" binding:"required,min=1,max=500"`
}

// SalesOrderListFilter represents filter options for sales order list
type SalesOrderListFilter struct {
	Search      string            `form:"search"`
	CustomerID  *uuid.UUID        `form:"customer_id"`
	WarehouseID *uuid.UUID        `form:"warehouse_id"`
	Status      *trade.OrderStatus `form:"status"`
	Statuses    []string          `form:"statuses"`
	StartDate   *time.Time        `form:"start_date"`
	EndDate     *time.Time        `form:"end_date"`
	MinAmount   *decimal.Decimal  `form:"min_amount"`
	MaxAmount   *decimal.Decimal  `form:"max_amount"`
	Page        int               `form:"page" binding:"min=1"`
	PageSize    int               `form:"page_size" binding:"min=1,max=100"`
	OrderBy     string            `form:"order_by"`
	OrderDir    string            `form:"order_dir" binding:"omitempty,oneof=asc desc"`
}

// SalesOrderResponse represents a sales order in API responses
type SalesOrderResponse struct {
	ID             uuid.UUID                `json:"id"`
	TenantID       uuid.UUID                `json:"tenant_id"`
	OrderNumber    string                   `json:"order_number"`
	CustomerID     uuid.UUID                `json:"customer_id"`
	CustomerName   string                   `json:"customer_name"`
	WarehouseID    *uuid.UUID               `json:"warehouse_id,omitempty"`
	Items          []SalesOrderItemResponse `json:"items"`
	ItemCount      int                      `json:"item_count"`
	TotalQuantity  decimal.Decimal          `json:"total_quantity"`
	TotalAmount    decimal.Decimal          `json:"total_amount"`
	DiscountAmount decimal.Decimal          `json:"discount_amount"`
	PayableAmount  decimal.Decimal          `json:"payable_amount"`
	Status         string                   `json:"status"`
	Remark         string                   `json:"remark"`
	ConfirmedAt    *time.Time               `json:"confirmed_at,omitempty"`
	ShippedAt      *time.Time               `json:"shipped_at,omitempty"`
	CompletedAt    *time.Time               `json:"completed_at,omitempty"`
	CancelledAt    *time.Time               `json:"cancelled_at,omitempty"`
	CancelReason   string                   `json:"cancel_reason,omitempty"`
	CreatedAt      time.Time                `json:"created_at"`
	UpdatedAt      time.Time                `json:"updated_at"`
	Version        int                      `json:"version"`
}

// SalesOrderListItemResponse represents a sales order in list responses (less detail)
type SalesOrderListItemResponse struct {
	ID            uuid.UUID       `json:"id"`
	OrderNumber   string          `json:"order_number"`
	CustomerID    uuid.UUID       `json:"customer_id"`
	CustomerName  string          `json:"customer_name"`
	WarehouseID   *uuid.UUID      `json:"warehouse_id,omitempty"`
	ItemCount     int             `json:"item_count"`
	TotalAmount   decimal.Decimal `json:"total_amount"`
	PayableAmount decimal.Decimal `json:"payable_amount"`
	Status        string          `json:"status"`
	ConfirmedAt   *time.Time      `json:"confirmed_at,omitempty"`
	ShippedAt     *time.Time      `json:"shipped_at,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// SalesOrderItemResponse represents an order item in API responses
type SalesOrderItemResponse struct {
	ID          uuid.UUID       `json:"id"`
	ProductID   uuid.UUID       `json:"product_id"`
	ProductName string          `json:"product_name"`
	ProductCode string          `json:"product_code"`
	Quantity    decimal.Decimal `json:"quantity"`
	UnitPrice   decimal.Decimal `json:"unit_price"`
	Amount      decimal.Decimal `json:"amount"`
	Unit        string          `json:"unit"`
	Remark      string          `json:"remark,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// OrderStatusSummary represents a summary of orders by status
type OrderStatusSummary struct {
	Draft     int64           `json:"draft"`
	Confirmed int64           `json:"confirmed"`
	Shipped   int64           `json:"shipped"`
	Completed int64           `json:"completed"`
	Cancelled int64           `json:"cancelled"`
	Total     int64           `json:"total"`
	TotalAmount decimal.Decimal `json:"total_amount"`
}

// ToSalesOrderResponse converts domain SalesOrder to response DTO
func ToSalesOrderResponse(order *trade.SalesOrder) SalesOrderResponse {
	items := make([]SalesOrderItemResponse, len(order.Items))
	for i := range order.Items {
		items[i] = ToSalesOrderItemResponse(&order.Items[i])
	}

	return SalesOrderResponse{
		ID:             order.ID,
		TenantID:       order.TenantID,
		OrderNumber:    order.OrderNumber,
		CustomerID:     order.CustomerID,
		CustomerName:   order.CustomerName,
		WarehouseID:    order.WarehouseID,
		Items:          items,
		ItemCount:      order.ItemCount(),
		TotalQuantity:  order.TotalQuantity(),
		TotalAmount:    order.TotalAmount,
		DiscountAmount: order.DiscountAmount,
		PayableAmount:  order.PayableAmount,
		Status:         string(order.Status),
		Remark:         order.Remark,
		ConfirmedAt:    order.ConfirmedAt,
		ShippedAt:      order.ShippedAt,
		CompletedAt:    order.CompletedAt,
		CancelledAt:    order.CancelledAt,
		CancelReason:   order.CancelReason,
		CreatedAt:      order.CreatedAt,
		UpdatedAt:      order.UpdatedAt,
		Version:        order.Version,
	}
}

// ToSalesOrderListItemResponse converts domain SalesOrder to list response DTO
func ToSalesOrderListItemResponse(order *trade.SalesOrder) SalesOrderListItemResponse {
	return SalesOrderListItemResponse{
		ID:            order.ID,
		OrderNumber:   order.OrderNumber,
		CustomerID:    order.CustomerID,
		CustomerName:  order.CustomerName,
		WarehouseID:   order.WarehouseID,
		ItemCount:     order.ItemCount(),
		TotalAmount:   order.TotalAmount,
		PayableAmount: order.PayableAmount,
		Status:        string(order.Status),
		ConfirmedAt:   order.ConfirmedAt,
		ShippedAt:     order.ShippedAt,
		CreatedAt:     order.CreatedAt,
		UpdatedAt:     order.UpdatedAt,
	}
}

// ToSalesOrderListItemResponses converts a slice of domain orders to list responses
func ToSalesOrderListItemResponses(orders []trade.SalesOrder) []SalesOrderListItemResponse {
	responses := make([]SalesOrderListItemResponse, len(orders))
	for i := range orders {
		responses[i] = ToSalesOrderListItemResponse(&orders[i])
	}
	return responses
}

// ToSalesOrderItemResponse converts domain SalesOrderItem to response DTO
func ToSalesOrderItemResponse(item *trade.SalesOrderItem) SalesOrderItemResponse {
	return SalesOrderItemResponse{
		ID:          item.ID,
		ProductID:   item.ProductID,
		ProductName: item.ProductName,
		ProductCode: item.ProductCode,
		Quantity:    item.Quantity,
		UnitPrice:   item.UnitPrice,
		Amount:      item.Amount,
		Unit:        item.Unit,
		Remark:      item.Remark,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}
