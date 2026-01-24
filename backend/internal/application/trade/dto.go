package trade

import (
	"time"

	"github.com/erp/backend/internal/domain/trade"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ==================== Purchase Order DTOs ====================

// CreatePurchaseOrderRequest represents a request to create a purchase order
type CreatePurchaseOrderRequest struct {
	SupplierID   uuid.UUID                      `json:"supplier_id" binding:"required"`
	SupplierName string                         `json:"supplier_name" binding:"required,min=1,max=200"`
	WarehouseID  *uuid.UUID                     `json:"warehouse_id"`
	Items        []CreatePurchaseOrderItemInput `json:"items"`
	Discount     *decimal.Decimal               `json:"discount"`
	Remark       string                         `json:"remark"`
}

// CreatePurchaseOrderItemInput represents an item in the create order request
type CreatePurchaseOrderItemInput struct {
	ProductID   uuid.UUID       `json:"product_id" binding:"required"`
	ProductName string          `json:"product_name" binding:"required,min=1,max=200"`
	ProductCode string          `json:"product_code" binding:"required,min=1,max=50"`
	Unit        string          `json:"unit" binding:"required,min=1,max=20"`
	Quantity    decimal.Decimal `json:"quantity" binding:"required"`
	UnitCost    decimal.Decimal `json:"unit_cost" binding:"required"`
	Remark      string          `json:"remark"`
}

// UpdatePurchaseOrderRequest represents a request to update a purchase order (only in DRAFT status)
type UpdatePurchaseOrderRequest struct {
	WarehouseID *uuid.UUID       `json:"warehouse_id"`
	Discount    *decimal.Decimal `json:"discount"`
	Remark      *string          `json:"remark"`
}

// AddPurchaseOrderItemRequest represents a request to add an item to a purchase order
type AddPurchaseOrderItemRequest struct {
	ProductID   uuid.UUID       `json:"product_id" binding:"required"`
	ProductName string          `json:"product_name" binding:"required,min=1,max=200"`
	ProductCode string          `json:"product_code" binding:"required,min=1,max=50"`
	Unit        string          `json:"unit" binding:"required,min=1,max=20"`
	Quantity    decimal.Decimal `json:"quantity" binding:"required"`
	UnitCost    decimal.Decimal `json:"unit_cost" binding:"required"`
	Remark      string          `json:"remark"`
}

// UpdatePurchaseOrderItemRequest represents a request to update a purchase order item
type UpdatePurchaseOrderItemRequest struct {
	Quantity *decimal.Decimal `json:"quantity"`
	UnitCost *decimal.Decimal `json:"unit_cost"`
	Remark   *string          `json:"remark"`
}

// ConfirmPurchaseOrderRequest represents a request to confirm a purchase order
type ConfirmPurchaseOrderRequest struct {
	WarehouseID *uuid.UUID `json:"warehouse_id"` // Optional warehouse override
}

// ReceiveItemInput represents a single item to receive
type ReceiveItemInput struct {
	ProductID   uuid.UUID        `json:"product_id" binding:"required"`
	Quantity    decimal.Decimal  `json:"quantity" binding:"required"`
	UnitCost    *decimal.Decimal `json:"unit_cost"` // Optional: override cost if different
	BatchNumber string           `json:"batch_number"`
	ExpiryDate  *time.Time       `json:"expiry_date"`
}

// ReceivePurchaseOrderRequest represents a request to receive goods for a purchase order
type ReceivePurchaseOrderRequest struct {
	WarehouseID *uuid.UUID         `json:"warehouse_id"` // Optional warehouse override (must be set if not already)
	Items       []ReceiveItemInput `json:"items" binding:"required,min=1"`
}

// CancelPurchaseOrderRequest represents a request to cancel a purchase order
type CancelPurchaseOrderRequest struct {
	Reason string `json:"reason" binding:"required,min=1,max=500"`
}

// PurchaseOrderListFilter represents filter options for purchase order list
type PurchaseOrderListFilter struct {
	Search      string                     `form:"search"`
	SupplierID  *uuid.UUID                 `form:"supplier_id"`
	WarehouseID *uuid.UUID                 `form:"warehouse_id"`
	Status      *trade.PurchaseOrderStatus `form:"status"`
	Statuses    []string                   `form:"statuses"`
	StartDate   *time.Time                 `form:"start_date"`
	EndDate     *time.Time                 `form:"end_date"`
	MinAmount   *decimal.Decimal           `form:"min_amount"`
	MaxAmount   *decimal.Decimal           `form:"max_amount"`
	Page        int                        `form:"page" binding:"min=1"`
	PageSize    int                        `form:"page_size" binding:"min=1,max=100"`
	OrderBy     string                     `form:"order_by"`
	OrderDir    string                     `form:"order_dir" binding:"omitempty,oneof=asc desc"`
}

// PurchaseOrderResponse represents a purchase order in API responses
type PurchaseOrderResponse struct {
	ID               uuid.UUID                   `json:"id"`
	TenantID         uuid.UUID                   `json:"tenant_id"`
	OrderNumber      string                      `json:"order_number"`
	SupplierID       uuid.UUID                   `json:"supplier_id"`
	SupplierName     string                      `json:"supplier_name"`
	WarehouseID      *uuid.UUID                  `json:"warehouse_id,omitempty"`
	Items            []PurchaseOrderItemResponse `json:"items"`
	ItemCount        int                         `json:"item_count"`
	TotalQuantity    decimal.Decimal             `json:"total_quantity"`
	ReceivedQuantity decimal.Decimal             `json:"received_quantity"`
	TotalAmount      decimal.Decimal             `json:"total_amount"`
	DiscountAmount   decimal.Decimal             `json:"discount_amount"`
	PayableAmount    decimal.Decimal             `json:"payable_amount"`
	Status           string                      `json:"status"`
	ReceiveProgress  decimal.Decimal             `json:"receive_progress"`
	Remark           string                      `json:"remark"`
	ConfirmedAt      *time.Time                  `json:"confirmed_at,omitempty"`
	CompletedAt      *time.Time                  `json:"completed_at,omitempty"`
	CancelledAt      *time.Time                  `json:"cancelled_at,omitempty"`
	CancelReason     string                      `json:"cancel_reason,omitempty"`
	CreatedAt        time.Time                   `json:"created_at"`
	UpdatedAt        time.Time                   `json:"updated_at"`
	Version          int                         `json:"version"`
}

// PurchaseOrderListItemResponse represents a purchase order in list responses (less detail)
type PurchaseOrderListItemResponse struct {
	ID              uuid.UUID       `json:"id"`
	OrderNumber     string          `json:"order_number"`
	SupplierID      uuid.UUID       `json:"supplier_id"`
	SupplierName    string          `json:"supplier_name"`
	WarehouseID     *uuid.UUID      `json:"warehouse_id,omitempty"`
	ItemCount       int             `json:"item_count"`
	TotalAmount     decimal.Decimal `json:"total_amount"`
	PayableAmount   decimal.Decimal `json:"payable_amount"`
	Status          string          `json:"status"`
	ReceiveProgress decimal.Decimal `json:"receive_progress"`
	ConfirmedAt     *time.Time      `json:"confirmed_at,omitempty"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// PurchaseOrderItemResponse represents a purchase order item in API responses
type PurchaseOrderItemResponse struct {
	ID                uuid.UUID       `json:"id"`
	ProductID         uuid.UUID       `json:"product_id"`
	ProductName       string          `json:"product_name"`
	ProductCode       string          `json:"product_code"`
	OrderedQuantity   decimal.Decimal `json:"ordered_quantity"`
	ReceivedQuantity  decimal.Decimal `json:"received_quantity"`
	RemainingQuantity decimal.Decimal `json:"remaining_quantity"`
	UnitCost          decimal.Decimal `json:"unit_cost"`
	Amount            decimal.Decimal `json:"amount"`
	Unit              string          `json:"unit"`
	Remark            string          `json:"remark,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// ReceivedItemResponse represents received item info in responses
type ReceivedItemResponse struct {
	ItemID      uuid.UUID       `json:"item_id"`
	ProductID   uuid.UUID       `json:"product_id"`
	ProductName string          `json:"product_name"`
	ProductCode string          `json:"product_code"`
	Quantity    decimal.Decimal `json:"quantity"`
	UnitCost    decimal.Decimal `json:"unit_cost"`
	Unit        string          `json:"unit"`
	BatchNumber string          `json:"batch_number,omitempty"`
	ExpiryDate  *time.Time      `json:"expiry_date,omitempty"`
}

// ReceiveResultResponse represents the result of a receive operation
type ReceiveResultResponse struct {
	Order           PurchaseOrderResponse  `json:"order"`
	ReceivedItems   []ReceivedItemResponse `json:"received_items"`
	IsFullyReceived bool                   `json:"is_fully_received"`
}

// PurchaseOrderStatusSummary represents a summary of purchase orders by status
type PurchaseOrderStatusSummary struct {
	Draft           int64           `json:"draft"`
	Confirmed       int64           `json:"confirmed"`
	PartialReceived int64           `json:"partial_received"`
	Completed       int64           `json:"completed"`
	Cancelled       int64           `json:"cancelled"`
	Total           int64           `json:"total"`
	PendingReceipt  int64           `json:"pending_receipt"` // CONFIRMED + PARTIAL_RECEIVED
	TotalAmount     decimal.Decimal `json:"total_amount"`
}

// ToPurchaseOrderResponse converts domain PurchaseOrder to response DTO
func ToPurchaseOrderResponse(order *trade.PurchaseOrder) PurchaseOrderResponse {
	items := make([]PurchaseOrderItemResponse, len(order.Items))
	for i := range order.Items {
		items[i] = ToPurchaseOrderItemResponse(&order.Items[i])
	}

	return PurchaseOrderResponse{
		ID:               order.ID,
		TenantID:         order.TenantID,
		OrderNumber:      order.OrderNumber,
		SupplierID:       order.SupplierID,
		SupplierName:     order.SupplierName,
		WarehouseID:      order.WarehouseID,
		Items:            items,
		ItemCount:        order.ItemCount(),
		TotalQuantity:    order.TotalOrderedQuantity(),
		ReceivedQuantity: order.TotalReceivedQuantity(),
		TotalAmount:      order.TotalAmount,
		DiscountAmount:   order.DiscountAmount,
		PayableAmount:    order.PayableAmount,
		Status:           string(order.Status),
		ReceiveProgress:  order.ReceiveProgress(),
		Remark:           order.Remark,
		ConfirmedAt:      order.ConfirmedAt,
		CompletedAt:      order.CompletedAt,
		CancelledAt:      order.CancelledAt,
		CancelReason:     order.CancelReason,
		CreatedAt:        order.CreatedAt,
		UpdatedAt:        order.UpdatedAt,
		Version:          order.Version,
	}
}

// ToPurchaseOrderListItemResponse converts domain PurchaseOrder to list response DTO
func ToPurchaseOrderListItemResponse(order *trade.PurchaseOrder) PurchaseOrderListItemResponse {
	return PurchaseOrderListItemResponse{
		ID:              order.ID,
		OrderNumber:     order.OrderNumber,
		SupplierID:      order.SupplierID,
		SupplierName:    order.SupplierName,
		WarehouseID:     order.WarehouseID,
		ItemCount:       order.ItemCount(),
		TotalAmount:     order.TotalAmount,
		PayableAmount:   order.PayableAmount,
		Status:          string(order.Status),
		ReceiveProgress: order.ReceiveProgress(),
		ConfirmedAt:     order.ConfirmedAt,
		CompletedAt:     order.CompletedAt,
		CreatedAt:       order.CreatedAt,
		UpdatedAt:       order.UpdatedAt,
	}
}

// ToPurchaseOrderListItemResponses converts a slice of domain orders to list responses
func ToPurchaseOrderListItemResponses(orders []trade.PurchaseOrder) []PurchaseOrderListItemResponse {
	responses := make([]PurchaseOrderListItemResponse, len(orders))
	for i := range orders {
		responses[i] = ToPurchaseOrderListItemResponse(&orders[i])
	}
	return responses
}

// ToPurchaseOrderItemResponse converts domain PurchaseOrderItem to response DTO
func ToPurchaseOrderItemResponse(item *trade.PurchaseOrderItem) PurchaseOrderItemResponse {
	return PurchaseOrderItemResponse{
		ID:                item.ID,
		ProductID:         item.ProductID,
		ProductName:       item.ProductName,
		ProductCode:       item.ProductCode,
		OrderedQuantity:   item.OrderedQuantity,
		ReceivedQuantity:  item.ReceivedQuantity,
		RemainingQuantity: item.RemainingQuantity(),
		UnitCost:          item.UnitCost,
		Amount:            item.Amount,
		Unit:              item.Unit,
		Remark:            item.Remark,
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	}
}

// ToReceivedItemResponse converts domain ReceivedItemInfo to response DTO
func ToReceivedItemResponse(info trade.ReceivedItemInfo) ReceivedItemResponse {
	return ReceivedItemResponse{
		ItemID:      info.ItemID,
		ProductID:   info.ProductID,
		ProductName: info.ProductName,
		ProductCode: info.ProductCode,
		Quantity:    info.Quantity,
		UnitCost:    info.UnitCost,
		Unit:        info.Unit,
		BatchNumber: info.BatchNumber,
		ExpiryDate:  info.ExpiryDate,
	}
}

// ToReceivedItemResponses converts a slice of domain ReceivedItemInfo to response DTOs
func ToReceivedItemResponses(infos []trade.ReceivedItemInfo) []ReceivedItemResponse {
	responses := make([]ReceivedItemResponse, len(infos))
	for i := range infos {
		responses[i] = ToReceivedItemResponse(infos[i])
	}
	return responses
}

// ==================== Sales Order DTOs ====================

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
	Search      string             `form:"search"`
	CustomerID  *uuid.UUID         `form:"customer_id"`
	WarehouseID *uuid.UUID         `form:"warehouse_id"`
	Status      *trade.OrderStatus `form:"status"`
	Statuses    []string           `form:"statuses"`
	StartDate   *time.Time         `form:"start_date"`
	EndDate     *time.Time         `form:"end_date"`
	MinAmount   *decimal.Decimal   `form:"min_amount"`
	MaxAmount   *decimal.Decimal   `form:"max_amount"`
	Page        int                `form:"page" binding:"min=1"`
	PageSize    int                `form:"page_size" binding:"min=1,max=100"`
	OrderBy     string             `form:"order_by"`
	OrderDir    string             `form:"order_dir" binding:"omitempty,oneof=asc desc"`
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
	Draft       int64           `json:"draft"`
	Confirmed   int64           `json:"confirmed"`
	Shipped     int64           `json:"shipped"`
	Completed   int64           `json:"completed"`
	Cancelled   int64           `json:"cancelled"`
	Total       int64           `json:"total"`
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

// ==================== Sales Return DTOs ====================

// CreateSalesReturnRequest represents a request to create a sales return
type CreateSalesReturnRequest struct {
	SalesOrderID uuid.UUID                    `json:"sales_order_id" binding:"required"`
	WarehouseID  *uuid.UUID                   `json:"warehouse_id"`
	Items        []CreateSalesReturnItemInput `json:"items" binding:"required,min=1"`
	Reason       string                       `json:"reason"`
	Remark       string                       `json:"remark"`
}

// CreateSalesReturnItemInput represents an item in the create return request
type CreateSalesReturnItemInput struct {
	SalesOrderItemID  uuid.UUID       `json:"sales_order_item_id" binding:"required"`
	ReturnQuantity    decimal.Decimal `json:"return_quantity" binding:"required"`
	Reason            string          `json:"reason"`
	ConditionOnReturn string          `json:"condition_on_return"` // damaged, defective, wrong_item, etc.
}

// UpdateSalesReturnRequest represents a request to update a sales return (only in DRAFT status)
type UpdateSalesReturnRequest struct {
	WarehouseID *uuid.UUID `json:"warehouse_id"`
	Reason      *string    `json:"reason"`
	Remark      *string    `json:"remark"`
}

// UpdateReturnItemRequest represents a request to update a return item
type UpdateReturnItemRequest struct {
	ReturnQuantity    *decimal.Decimal `json:"return_quantity"`
	Reason            *string          `json:"reason"`
	ConditionOnReturn *string          `json:"condition_on_return"`
}

// AddReturnItemRequest represents a request to add an item to a return
type AddReturnItemRequest struct {
	SalesOrderItemID  uuid.UUID       `json:"sales_order_item_id" binding:"required"`
	ReturnQuantity    decimal.Decimal `json:"return_quantity" binding:"required"`
	Reason            string          `json:"reason"`
	ConditionOnReturn string          `json:"condition_on_return"`
}

// ApproveReturnRequest represents a request to approve a return
type ApproveReturnRequest struct {
	Note string `json:"note"`
}

// RejectReturnRequest represents a request to reject a return
type RejectReturnRequest struct {
	Reason string `json:"reason" binding:"required,min=1,max=500"`
}

// CancelReturnRequest represents a request to cancel a return
type CancelReturnRequest struct {
	Reason string `json:"reason" binding:"required,min=1,max=500"`
}

// CompleteReturnRequest represents a request to complete a return
type CompleteReturnRequest struct {
	WarehouseID *uuid.UUID `json:"warehouse_id"` // Optional warehouse override
}

// SalesReturnListFilter represents filter options for sales return list
type SalesReturnListFilter struct {
	Search       string              `form:"search"`
	CustomerID   *uuid.UUID          `form:"customer_id"`
	SalesOrderID *uuid.UUID          `form:"sales_order_id"`
	WarehouseID  *uuid.UUID          `form:"warehouse_id"`
	Status       *trade.ReturnStatus `form:"status"`
	Statuses     []string            `form:"statuses"`
	StartDate    *time.Time          `form:"start_date"`
	EndDate      *time.Time          `form:"end_date"`
	MinAmount    *decimal.Decimal    `form:"min_amount"`
	MaxAmount    *decimal.Decimal    `form:"max_amount"`
	Page         int                 `form:"page" binding:"min=1"`
	PageSize     int                 `form:"page_size" binding:"min=1,max=100"`
	OrderBy      string              `form:"order_by"`
	OrderDir     string              `form:"order_dir" binding:"omitempty,oneof=asc desc"`
}

// SalesReturnResponse represents a sales return in API responses
type SalesReturnResponse struct {
	ID               uuid.UUID                 `json:"id"`
	TenantID         uuid.UUID                 `json:"tenant_id"`
	ReturnNumber     string                    `json:"return_number"`
	SalesOrderID     uuid.UUID                 `json:"sales_order_id"`
	SalesOrderNumber string                    `json:"sales_order_number"`
	CustomerID       uuid.UUID                 `json:"customer_id"`
	CustomerName     string                    `json:"customer_name"`
	WarehouseID      *uuid.UUID                `json:"warehouse_id,omitempty"`
	Items            []SalesReturnItemResponse `json:"items"`
	ItemCount        int                       `json:"item_count"`
	TotalQuantity    decimal.Decimal           `json:"total_quantity"`
	TotalRefund      decimal.Decimal           `json:"total_refund"`
	Status           string                    `json:"status"`
	Reason           string                    `json:"reason,omitempty"`
	Remark           string                    `json:"remark,omitempty"`
	SubmittedAt      *time.Time                `json:"submitted_at,omitempty"`
	ApprovedAt       *time.Time                `json:"approved_at,omitempty"`
	ApprovedBy       *uuid.UUID                `json:"approved_by,omitempty"`
	ApprovalNote     string                    `json:"approval_note,omitempty"`
	RejectedAt       *time.Time                `json:"rejected_at,omitempty"`
	RejectedBy       *uuid.UUID                `json:"rejected_by,omitempty"`
	RejectionReason  string                    `json:"rejection_reason,omitempty"`
	CompletedAt      *time.Time                `json:"completed_at,omitempty"`
	CancelledAt      *time.Time                `json:"cancelled_at,omitempty"`
	CancelReason     string                    `json:"cancel_reason,omitempty"`
	CreatedAt        time.Time                 `json:"created_at"`
	UpdatedAt        time.Time                 `json:"updated_at"`
	Version          int                       `json:"version"`
}

// SalesReturnListItemResponse represents a sales return in list responses (less detail)
type SalesReturnListItemResponse struct {
	ID               uuid.UUID       `json:"id"`
	ReturnNumber     string          `json:"return_number"`
	SalesOrderID     uuid.UUID       `json:"sales_order_id"`
	SalesOrderNumber string          `json:"sales_order_number"`
	CustomerID       uuid.UUID       `json:"customer_id"`
	CustomerName     string          `json:"customer_name"`
	WarehouseID      *uuid.UUID      `json:"warehouse_id,omitempty"`
	ItemCount        int             `json:"item_count"`
	TotalRefund      decimal.Decimal `json:"total_refund"`
	Status           string          `json:"status"`
	SubmittedAt      *time.Time      `json:"submitted_at,omitempty"`
	ApprovedAt       *time.Time      `json:"approved_at,omitempty"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// SalesReturnItemResponse represents a return item in API responses
type SalesReturnItemResponse struct {
	ID                uuid.UUID       `json:"id"`
	SalesOrderItemID  uuid.UUID       `json:"sales_order_item_id"`
	ProductID         uuid.UUID       `json:"product_id"`
	ProductName       string          `json:"product_name"`
	ProductCode       string          `json:"product_code"`
	OriginalQuantity  decimal.Decimal `json:"original_quantity"`
	ReturnQuantity    decimal.Decimal `json:"return_quantity"`
	UnitPrice         decimal.Decimal `json:"unit_price"`
	RefundAmount      decimal.Decimal `json:"refund_amount"`
	Unit              string          `json:"unit"`
	Reason            string          `json:"reason,omitempty"`
	ConditionOnReturn string          `json:"condition_on_return,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// ReturnStatusSummary represents a summary of returns by status
type ReturnStatusSummary struct {
	Draft           int64           `json:"draft"`
	Pending         int64           `json:"pending"`
	Approved        int64           `json:"approved"`
	Rejected        int64           `json:"rejected"`
	Completed       int64           `json:"completed"`
	Cancelled       int64           `json:"cancelled"`
	Total           int64           `json:"total"`
	PendingApproval int64           `json:"pending_approval"` // Same as Pending, for convenience
	TotalRefund     decimal.Decimal `json:"total_refund"`
}

// ToSalesReturnResponse converts domain SalesReturn to response DTO
func ToSalesReturnResponse(sr *trade.SalesReturn) SalesReturnResponse {
	items := make([]SalesReturnItemResponse, len(sr.Items))
	for i := range sr.Items {
		items[i] = ToSalesReturnItemResponse(&sr.Items[i])
	}

	return SalesReturnResponse{
		ID:               sr.ID,
		TenantID:         sr.TenantID,
		ReturnNumber:     sr.ReturnNumber,
		SalesOrderID:     sr.SalesOrderID,
		SalesOrderNumber: sr.SalesOrderNumber,
		CustomerID:       sr.CustomerID,
		CustomerName:     sr.CustomerName,
		WarehouseID:      sr.WarehouseID,
		Items:            items,
		ItemCount:        sr.ItemCount(),
		TotalQuantity:    sr.TotalReturnQuantity(),
		TotalRefund:      sr.TotalRefund,
		Status:           string(sr.Status),
		Reason:           sr.Reason,
		Remark:           sr.Remark,
		SubmittedAt:      sr.SubmittedAt,
		ApprovedAt:       sr.ApprovedAt,
		ApprovedBy:       sr.ApprovedBy,
		ApprovalNote:     sr.ApprovalNote,
		RejectedAt:       sr.RejectedAt,
		RejectedBy:       sr.RejectedBy,
		RejectionReason:  sr.RejectionReason,
		CompletedAt:      sr.CompletedAt,
		CancelledAt:      sr.CancelledAt,
		CancelReason:     sr.CancelReason,
		CreatedAt:        sr.CreatedAt,
		UpdatedAt:        sr.UpdatedAt,
		Version:          sr.Version,
	}
}

// ToSalesReturnListItemResponse converts domain SalesReturn to list response DTO
func ToSalesReturnListItemResponse(sr *trade.SalesReturn) SalesReturnListItemResponse {
	return SalesReturnListItemResponse{
		ID:               sr.ID,
		ReturnNumber:     sr.ReturnNumber,
		SalesOrderID:     sr.SalesOrderID,
		SalesOrderNumber: sr.SalesOrderNumber,
		CustomerID:       sr.CustomerID,
		CustomerName:     sr.CustomerName,
		WarehouseID:      sr.WarehouseID,
		ItemCount:        sr.ItemCount(),
		TotalRefund:      sr.TotalRefund,
		Status:           string(sr.Status),
		SubmittedAt:      sr.SubmittedAt,
		ApprovedAt:       sr.ApprovedAt,
		CompletedAt:      sr.CompletedAt,
		CreatedAt:        sr.CreatedAt,
		UpdatedAt:        sr.UpdatedAt,
	}
}

// ToSalesReturnListItemResponses converts a slice of domain returns to list responses
func ToSalesReturnListItemResponses(returns []trade.SalesReturn) []SalesReturnListItemResponse {
	responses := make([]SalesReturnListItemResponse, len(returns))
	for i := range returns {
		responses[i] = ToSalesReturnListItemResponse(&returns[i])
	}
	return responses
}

// ToSalesReturnItemResponse converts domain SalesReturnItem to response DTO
func ToSalesReturnItemResponse(item *trade.SalesReturnItem) SalesReturnItemResponse {
	return SalesReturnItemResponse{
		ID:                item.ID,
		SalesOrderItemID:  item.SalesOrderItemID,
		ProductID:         item.ProductID,
		ProductName:       item.ProductName,
		ProductCode:       item.ProductCode,
		OriginalQuantity:  item.OriginalQuantity,
		ReturnQuantity:    item.ReturnQuantity,
		UnitPrice:         item.UnitPrice,
		RefundAmount:      item.RefundAmount,
		Unit:              item.Unit,
		Reason:            item.Reason,
		ConditionOnReturn: item.ConditionOnReturn,
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	}
}

// ==================== Purchase Return DTOs ====================

// CreatePurchaseReturnRequest represents a request to create a purchase return
type CreatePurchaseReturnRequest struct {
	PurchaseOrderID uuid.UUID                       `json:"purchase_order_id" binding:"required"`
	WarehouseID     *uuid.UUID                      `json:"warehouse_id"`
	Items           []CreatePurchaseReturnItemInput `json:"items" binding:"required,min=1"`
	Reason          string                          `json:"reason"`
	Remark          string                          `json:"remark"`
}

// CreatePurchaseReturnItemInput represents an item in the create return request
type CreatePurchaseReturnItemInput struct {
	PurchaseOrderItemID uuid.UUID       `json:"purchase_order_item_id" binding:"required"`
	ReturnQuantity      decimal.Decimal `json:"return_quantity" binding:"required"`
	Reason              string          `json:"reason"`
	ConditionOnReturn   string          `json:"condition_on_return"` // defective, excess, wrong_item, etc.
	BatchNumber         string          `json:"batch_number"`
}

// UpdatePurchaseReturnRequest represents a request to update a purchase return (only in DRAFT status)
type UpdatePurchaseReturnRequest struct {
	WarehouseID *uuid.UUID `json:"warehouse_id"`
	Reason      *string    `json:"reason"`
	Remark      *string    `json:"remark"`
}

// UpdatePurchaseReturnItemRequest represents a request to update a return item
type UpdatePurchaseReturnItemRequest struct {
	ReturnQuantity    *decimal.Decimal `json:"return_quantity"`
	Reason            *string          `json:"reason"`
	ConditionOnReturn *string          `json:"condition_on_return"`
	BatchNumber       *string          `json:"batch_number"`
}

// AddPurchaseReturnItemRequest represents a request to add an item to a return
type AddPurchaseReturnItemRequest struct {
	PurchaseOrderItemID uuid.UUID       `json:"purchase_order_item_id" binding:"required"`
	ReturnQuantity      decimal.Decimal `json:"return_quantity" binding:"required"`
	Reason              string          `json:"reason"`
	ConditionOnReturn   string          `json:"condition_on_return"`
	BatchNumber         string          `json:"batch_number"`
}

// ApprovePurchaseReturnRequest represents a request to approve a return
type ApprovePurchaseReturnRequest struct {
	Note string `json:"note"`
}

// RejectPurchaseReturnRequest represents a request to reject a return
type RejectPurchaseReturnRequest struct {
	Reason string `json:"reason" binding:"required,min=1,max=500"`
}

// CancelPurchaseReturnRequest represents a request to cancel a return
type CancelPurchaseReturnRequest struct {
	Reason string `json:"reason" binding:"required,min=1,max=500"`
}

// ShipPurchaseReturnRequest represents a request to ship a return back to supplier
type ShipPurchaseReturnRequest struct {
	TrackingNumber string `json:"tracking_number"`
	Note           string `json:"note"`
}

// CompletePurchaseReturnRequest represents a request to complete a return
type CompletePurchaseReturnRequest struct {
	// No additional fields needed - just marks the return as completed after supplier confirms receipt
}

// PurchaseReturnListFilter represents filter options for purchase return list
type PurchaseReturnListFilter struct {
	Search          string                      `form:"search"`
	SupplierID      *uuid.UUID                  `form:"supplier_id"`
	PurchaseOrderID *uuid.UUID                  `form:"purchase_order_id"`
	WarehouseID     *uuid.UUID                  `form:"warehouse_id"`
	Status          *trade.PurchaseReturnStatus `form:"status"`
	Statuses        []string                    `form:"statuses"`
	StartDate       *time.Time                  `form:"start_date"`
	EndDate         *time.Time                  `form:"end_date"`
	MinAmount       *decimal.Decimal            `form:"min_amount"`
	MaxAmount       *decimal.Decimal            `form:"max_amount"`
	Page            int                         `form:"page" binding:"min=1"`
	PageSize        int                         `form:"page_size" binding:"min=1,max=100"`
	OrderBy         string                      `form:"order_by"`
	OrderDir        string                      `form:"order_dir" binding:"omitempty,oneof=asc desc"`
}

// PurchaseReturnResponse represents a purchase return in API responses
type PurchaseReturnResponse struct {
	ID                  uuid.UUID                    `json:"id"`
	TenantID            uuid.UUID                    `json:"tenant_id"`
	ReturnNumber        string                       `json:"return_number"`
	PurchaseOrderID     uuid.UUID                    `json:"purchase_order_id"`
	PurchaseOrderNumber string                       `json:"purchase_order_number"`
	SupplierID          uuid.UUID                    `json:"supplier_id"`
	SupplierName        string                       `json:"supplier_name"`
	WarehouseID         *uuid.UUID                   `json:"warehouse_id,omitempty"`
	Items               []PurchaseReturnItemResponse `json:"items"`
	ItemCount           int                          `json:"item_count"`
	TotalQuantity       decimal.Decimal              `json:"total_quantity"`
	TotalRefund         decimal.Decimal              `json:"total_refund"`
	Status              string                       `json:"status"`
	Reason              string                       `json:"reason,omitempty"`
	Remark              string                       `json:"remark,omitempty"`
	SubmittedAt         *time.Time                   `json:"submitted_at,omitempty"`
	ApprovedAt          *time.Time                   `json:"approved_at,omitempty"`
	ApprovedBy          *uuid.UUID                   `json:"approved_by,omitempty"`
	ApprovalNote        string                       `json:"approval_note,omitempty"`
	RejectedAt          *time.Time                   `json:"rejected_at,omitempty"`
	RejectedBy          *uuid.UUID                   `json:"rejected_by,omitempty"`
	RejectionReason     string                       `json:"rejection_reason,omitempty"`
	ShippedAt           *time.Time                   `json:"shipped_at,omitempty"`
	ShippedBy           *uuid.UUID                   `json:"shipped_by,omitempty"`
	ShippingNote        string                       `json:"shipping_note,omitempty"`
	TrackingNumber      string                       `json:"tracking_number,omitempty"`
	CompletedAt         *time.Time                   `json:"completed_at,omitempty"`
	CancelledAt         *time.Time                   `json:"cancelled_at,omitempty"`
	CancelReason        string                       `json:"cancel_reason,omitempty"`
	CreatedAt           time.Time                    `json:"created_at"`
	UpdatedAt           time.Time                    `json:"updated_at"`
	Version             int                          `json:"version"`
}

// PurchaseReturnListItemResponse represents a purchase return in list responses (less detail)
type PurchaseReturnListItemResponse struct {
	ID                  uuid.UUID       `json:"id"`
	ReturnNumber        string          `json:"return_number"`
	PurchaseOrderID     uuid.UUID       `json:"purchase_order_id"`
	PurchaseOrderNumber string          `json:"purchase_order_number"`
	SupplierID          uuid.UUID       `json:"supplier_id"`
	SupplierName        string          `json:"supplier_name"`
	WarehouseID         *uuid.UUID      `json:"warehouse_id,omitempty"`
	ItemCount           int             `json:"item_count"`
	TotalRefund         decimal.Decimal `json:"total_refund"`
	Status              string          `json:"status"`
	SubmittedAt         *time.Time      `json:"submitted_at,omitempty"`
	ApprovedAt          *time.Time      `json:"approved_at,omitempty"`
	ShippedAt           *time.Time      `json:"shipped_at,omitempty"`
	CompletedAt         *time.Time      `json:"completed_at,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

// PurchaseReturnItemResponse represents a return item in API responses
type PurchaseReturnItemResponse struct {
	ID                  uuid.UUID       `json:"id"`
	PurchaseOrderItemID uuid.UUID       `json:"purchase_order_item_id"`
	ProductID           uuid.UUID       `json:"product_id"`
	ProductName         string          `json:"product_name"`
	ProductCode         string          `json:"product_code"`
	OriginalQuantity    decimal.Decimal `json:"original_quantity"`
	ReturnQuantity      decimal.Decimal `json:"return_quantity"`
	UnitCost            decimal.Decimal `json:"unit_cost"`
	RefundAmount        decimal.Decimal `json:"refund_amount"`
	Unit                string          `json:"unit"`
	Reason              string          `json:"reason,omitempty"`
	ConditionOnReturn   string          `json:"condition_on_return,omitempty"`
	BatchNumber         string          `json:"batch_number,omitempty"`
	ShippedQuantity     decimal.Decimal `json:"shipped_quantity"`
	ShippedAt           *time.Time      `json:"shipped_at,omitempty"`
	SupplierReceivedQty decimal.Decimal `json:"supplier_received_qty"`
	SupplierReceivedAt  *time.Time      `json:"supplier_received_at,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

// PurchaseReturnStatusSummary represents a summary of purchase returns by status
type PurchaseReturnStatusSummary struct {
	Draft           int64           `json:"draft"`
	Pending         int64           `json:"pending"`
	Approved        int64           `json:"approved"`
	Rejected        int64           `json:"rejected"`
	Shipped         int64           `json:"shipped"`
	Completed       int64           `json:"completed"`
	Cancelled       int64           `json:"cancelled"`
	Total           int64           `json:"total"`
	PendingApproval int64           `json:"pending_approval"` // Same as Pending, for convenience
	PendingShipment int64           `json:"pending_shipment"` // Same as Approved, for convenience
	TotalRefund     decimal.Decimal `json:"total_refund"`
}

// ToPurchaseReturnResponse converts domain PurchaseReturn to response DTO
func ToPurchaseReturnResponse(pr *trade.PurchaseReturn) PurchaseReturnResponse {
	items := make([]PurchaseReturnItemResponse, len(pr.Items))
	for i := range pr.Items {
		items[i] = ToPurchaseReturnItemResponse(&pr.Items[i])
	}

	return PurchaseReturnResponse{
		ID:                  pr.ID,
		TenantID:            pr.TenantID,
		ReturnNumber:        pr.ReturnNumber,
		PurchaseOrderID:     pr.PurchaseOrderID,
		PurchaseOrderNumber: pr.PurchaseOrderNumber,
		SupplierID:          pr.SupplierID,
		SupplierName:        pr.SupplierName,
		WarehouseID:         pr.WarehouseID,
		Items:               items,
		ItemCount:           pr.ItemCount(),
		TotalQuantity:       pr.TotalReturnQuantity(),
		TotalRefund:         pr.TotalRefund,
		Status:              string(pr.Status),
		Reason:              pr.Reason,
		Remark:              pr.Remark,
		SubmittedAt:         pr.SubmittedAt,
		ApprovedAt:          pr.ApprovedAt,
		ApprovedBy:          pr.ApprovedBy,
		ApprovalNote:        pr.ApprovalNote,
		RejectedAt:          pr.RejectedAt,
		RejectedBy:          pr.RejectedBy,
		RejectionReason:     pr.RejectionReason,
		ShippedAt:           pr.ShippedAt,
		ShippedBy:           pr.ShippedBy,
		ShippingNote:        pr.ShippingNote,
		TrackingNumber:      pr.TrackingNumber,
		CompletedAt:         pr.CompletedAt,
		CancelledAt:         pr.CancelledAt,
		CancelReason:        pr.CancelReason,
		CreatedAt:           pr.CreatedAt,
		UpdatedAt:           pr.UpdatedAt,
		Version:             pr.Version,
	}
}

// ToPurchaseReturnListItemResponse converts domain PurchaseReturn to list response DTO
func ToPurchaseReturnListItemResponse(pr *trade.PurchaseReturn) PurchaseReturnListItemResponse {
	return PurchaseReturnListItemResponse{
		ID:                  pr.ID,
		ReturnNumber:        pr.ReturnNumber,
		PurchaseOrderID:     pr.PurchaseOrderID,
		PurchaseOrderNumber: pr.PurchaseOrderNumber,
		SupplierID:          pr.SupplierID,
		SupplierName:        pr.SupplierName,
		WarehouseID:         pr.WarehouseID,
		ItemCount:           pr.ItemCount(),
		TotalRefund:         pr.TotalRefund,
		Status:              string(pr.Status),
		SubmittedAt:         pr.SubmittedAt,
		ApprovedAt:          pr.ApprovedAt,
		ShippedAt:           pr.ShippedAt,
		CompletedAt:         pr.CompletedAt,
		CreatedAt:           pr.CreatedAt,
		UpdatedAt:           pr.UpdatedAt,
	}
}

// ToPurchaseReturnListItemResponses converts a slice of domain returns to list responses
func ToPurchaseReturnListItemResponses(returns []trade.PurchaseReturn) []PurchaseReturnListItemResponse {
	responses := make([]PurchaseReturnListItemResponse, len(returns))
	for i := range returns {
		responses[i] = ToPurchaseReturnListItemResponse(&returns[i])
	}
	return responses
}

// ToPurchaseReturnItemResponse converts domain PurchaseReturnItem to response DTO
func ToPurchaseReturnItemResponse(item *trade.PurchaseReturnItem) PurchaseReturnItemResponse {
	return PurchaseReturnItemResponse{
		ID:                  item.ID,
		PurchaseOrderItemID: item.PurchaseOrderItemID,
		ProductID:           item.ProductID,
		ProductName:         item.ProductName,
		ProductCode:         item.ProductCode,
		OriginalQuantity:    item.OriginalQuantity,
		ReturnQuantity:      item.ReturnQuantity,
		UnitCost:            item.UnitCost,
		RefundAmount:        item.RefundAmount,
		Unit:                item.Unit,
		Reason:              item.Reason,
		ConditionOnReturn:   item.ConditionOnReturn,
		BatchNumber:         item.BatchNumber,
		ShippedQuantity:     item.ShippedQuantity,
		ShippedAt:           item.ShippedAt,
		SupplierReceivedQty: item.SupplierReceivedQty,
		SupplierReceivedAt:  item.SupplierReceivedAt,
		CreatedAt:           item.CreatedAt,
		UpdatedAt:           item.UpdatedAt,
	}
}
