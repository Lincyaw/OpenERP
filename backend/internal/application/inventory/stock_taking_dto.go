package inventory

import (
	"time"

	"github.com/erp/backend/internal/domain/inventory"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ===================== Request DTOs =====================

// CreateStockTakingRequest represents a request to create a stock taking
type CreateStockTakingRequest struct {
	WarehouseID   uuid.UUID  `json:"warehouse_id" binding:"required"`
	WarehouseName string     `json:"warehouse_name" binding:"required"`
	TakingDate    *time.Time `json:"taking_date"` // Optional, defaults to now
	Remark        string     `json:"remark"`
	CreatedByID   uuid.UUID  `json:"created_by_id" binding:"required"`
	CreatedByName string     `json:"created_by_name" binding:"required"`
}

// UpdateStockTakingRequest represents a request to update a stock taking
type UpdateStockTakingRequest struct {
	Remark string `json:"remark"`
}

// AddStockTakingItemRequest represents a request to add an item to stock taking
type AddStockTakingItemRequest struct {
	ProductID      uuid.UUID       `json:"product_id" binding:"required"`
	ProductName    string          `json:"product_name" binding:"required"`
	ProductCode    string          `json:"product_code" binding:"required"`
	Unit           string          `json:"unit" binding:"required"`
	SystemQuantity decimal.Decimal `json:"system_quantity" binding:"required"`
	UnitCost       decimal.Decimal `json:"unit_cost" binding:"required"`
}

// AddStockTakingItemsRequest represents a bulk request to add items
type AddStockTakingItemsRequest struct {
	Items []AddStockTakingItemRequest `json:"items" binding:"required,min=1"`
}

// RecordCountRequest represents a request to record the actual count for an item
type RecordCountRequest struct {
	ProductID      uuid.UUID       `json:"product_id" binding:"required"`
	ActualQuantity decimal.Decimal `json:"actual_quantity" binding:"required,gte=0"`
	Remark         string          `json:"remark"`
}

// RecordCountsRequest represents a bulk request to record counts
type RecordCountsRequest struct {
	Counts []RecordCountRequest `json:"counts" binding:"required,min=1"`
}

// ApproveStockTakingRequest represents a request to approve a stock taking
type ApproveStockTakingRequest struct {
	ApproverID   uuid.UUID `json:"approver_id" binding:"required"`
	ApproverName string    `json:"approver_name" binding:"required"`
	Note         string    `json:"note"`
}

// RejectStockTakingRequest represents a request to reject a stock taking
type RejectStockTakingRequest struct {
	ApproverID   uuid.UUID `json:"approver_id" binding:"required"`
	ApproverName string    `json:"approver_name" binding:"required"`
	Reason       string    `json:"reason" binding:"required,min=1,max=500"`
}

// CancelStockTakingRequest represents a request to cancel a stock taking
type CancelStockTakingRequest struct {
	Reason string `json:"reason" binding:"max=500"`
}

// StockTakingListFilter represents filter options for stock taking list
type StockTakingListFilter struct {
	Search      string                       `form:"search"`
	WarehouseID *uuid.UUID                   `form:"warehouse_id"`
	Status      *inventory.StockTakingStatus `form:"status"`
	StartDate   *time.Time                   `form:"start_date"`
	EndDate     *time.Time                   `form:"end_date"`
	CreatedByID *uuid.UUID                   `form:"created_by_id"`
	Page        int                          `form:"page" binding:"min=1"`
	PageSize    int                          `form:"page_size" binding:"min=1,max=100"`
	OrderBy     string                       `form:"order_by"`
	OrderDir    string                       `form:"order_dir" binding:"omitempty,oneof=asc desc"`
}

// ===================== Response DTOs =====================

// StockTakingItemResponse represents a stock taking item in API responses
type StockTakingItemResponse struct {
	ID               uuid.UUID       `json:"id"`
	StockTakingID    uuid.UUID       `json:"stock_taking_id"`
	ProductID        uuid.UUID       `json:"product_id"`
	ProductName      string          `json:"product_name"`
	ProductCode      string          `json:"product_code"`
	Unit             string          `json:"unit"`
	SystemQuantity   decimal.Decimal `json:"system_quantity"`
	ActualQuantity   decimal.Decimal `json:"actual_quantity"`
	DifferenceQty    decimal.Decimal `json:"difference_qty"`
	UnitCost         decimal.Decimal `json:"unit_cost"`
	DifferenceAmount decimal.Decimal `json:"difference_amount"`
	Counted          bool            `json:"counted"`
	Remark           string          `json:"remark,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// StockTakingResponse represents a stock taking in API responses
type StockTakingResponse struct {
	ID              uuid.UUID                 `json:"id"`
	TenantID        uuid.UUID                 `json:"tenant_id"`
	TakingNumber    string                    `json:"taking_number"`
	WarehouseID     uuid.UUID                 `json:"warehouse_id"`
	WarehouseName   string                    `json:"warehouse_name"`
	Status          string                    `json:"status"`
	TakingDate      time.Time                 `json:"taking_date"`
	StartedAt       *time.Time                `json:"started_at,omitempty"`
	CompletedAt     *time.Time                `json:"completed_at,omitempty"`
	ApprovedAt      *time.Time                `json:"approved_at,omitempty"`
	ApprovedByID    *uuid.UUID                `json:"approved_by_id,omitempty"`
	ApprovedByName  string                    `json:"approved_by_name,omitempty"`
	CreatedByID     uuid.UUID                 `json:"created_by_id"`
	CreatedByName   string                    `json:"created_by_name"`
	TotalItems      int                       `json:"total_items"`
	CountedItems    int                       `json:"counted_items"`
	DifferenceItems int                       `json:"difference_items"`
	TotalDifference decimal.Decimal           `json:"total_difference"`
	Progress        float64                   `json:"progress"`
	ApprovalNote    string                    `json:"approval_note,omitempty"`
	Remark          string                    `json:"remark,omitempty"`
	Items           []StockTakingItemResponse `json:"items,omitempty"`
	CreatedAt       time.Time                 `json:"created_at"`
	UpdatedAt       time.Time                 `json:"updated_at"`
	Version         int                       `json:"version"`
}

// StockTakingListResponse represents a stock taking in list views (without items)
type StockTakingListResponse struct {
	ID              uuid.UUID       `json:"id"`
	TakingNumber    string          `json:"taking_number"`
	WarehouseID     uuid.UUID       `json:"warehouse_id"`
	WarehouseName   string          `json:"warehouse_name"`
	Status          string          `json:"status"`
	TakingDate      time.Time       `json:"taking_date"`
	CreatedByID     uuid.UUID       `json:"created_by_id"`
	CreatedByName   string          `json:"created_by_name"`
	TotalItems      int             `json:"total_items"`
	CountedItems    int             `json:"counted_items"`
	DifferenceItems int             `json:"difference_items"`
	TotalDifference decimal.Decimal `json:"total_difference"`
	Progress        float64         `json:"progress"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// StockTakingProgressResponse represents progress of a stock taking
type StockTakingProgressResponse struct {
	ID              uuid.UUID       `json:"id"`
	TakingNumber    string          `json:"taking_number"`
	Status          string          `json:"status"`
	TotalItems      int             `json:"total_items"`
	CountedItems    int             `json:"counted_items"`
	DifferenceItems int             `json:"difference_items"`
	TotalDifference decimal.Decimal `json:"total_difference"`
	Progress        float64         `json:"progress"`
	IsComplete      bool            `json:"is_complete"`
}

// ===================== Conversion Functions =====================

// ToStockTakingItemResponse converts domain StockTakingItem to response DTO
func ToStockTakingItemResponse(item *inventory.StockTakingItem) StockTakingItemResponse {
	return StockTakingItemResponse{
		ID:               item.ID,
		StockTakingID:    item.StockTakingID,
		ProductID:        item.ProductID,
		ProductName:      item.ProductName,
		ProductCode:      item.ProductCode,
		Unit:             item.Unit,
		SystemQuantity:   item.SystemQuantity,
		ActualQuantity:   item.ActualQuantity,
		DifferenceQty:    item.DifferenceQty,
		UnitCost:         item.UnitCost,
		DifferenceAmount: item.DifferenceAmount,
		Counted:          item.Counted,
		Remark:           item.Remark,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
	}
}

// ToStockTakingItemResponses converts a slice of domain StockTakingItems to responses
func ToStockTakingItemResponses(items []inventory.StockTakingItem) []StockTakingItemResponse {
	responses := make([]StockTakingItemResponse, len(items))
	for i := range items {
		responses[i] = ToStockTakingItemResponse(&items[i])
	}
	return responses
}

// ToStockTakingResponse converts domain StockTaking to response DTO
func ToStockTakingResponse(st *inventory.StockTaking) StockTakingResponse {
	response := StockTakingResponse{
		ID:              st.ID,
		TenantID:        st.TenantID,
		TakingNumber:    st.TakingNumber,
		WarehouseID:     st.WarehouseID,
		WarehouseName:   st.WarehouseName,
		Status:          string(st.Status),
		TakingDate:      st.TakingDate,
		StartedAt:       st.StartedAt,
		CompletedAt:     st.CompletedAt,
		ApprovedAt:      st.ApprovedAt,
		ApprovedByID:    st.ApprovedByID,
		ApprovedByName:  st.ApprovedByName,
		CreatedByID:     st.CreatedByID,
		CreatedByName:   st.CreatedByName,
		TotalItems:      st.TotalItems,
		CountedItems:    st.CountedItems,
		DifferenceItems: st.DifferenceItems,
		TotalDifference: st.TotalDifference,
		Progress:        st.GetProgress(),
		ApprovalNote:    st.ApprovalNote,
		Remark:          st.Remark,
		CreatedAt:       st.CreatedAt,
		UpdatedAt:       st.UpdatedAt,
		Version:         st.Version,
	}

	if len(st.Items) > 0 {
		response.Items = ToStockTakingItemResponses(st.Items)
	}

	return response
}

// ToStockTakingListResponse converts domain StockTaking to list response DTO
func ToStockTakingListResponse(st *inventory.StockTaking) StockTakingListResponse {
	return StockTakingListResponse{
		ID:              st.ID,
		TakingNumber:    st.TakingNumber,
		WarehouseID:     st.WarehouseID,
		WarehouseName:   st.WarehouseName,
		Status:          string(st.Status),
		TakingDate:      st.TakingDate,
		CreatedByID:     st.CreatedByID,
		CreatedByName:   st.CreatedByName,
		TotalItems:      st.TotalItems,
		CountedItems:    st.CountedItems,
		DifferenceItems: st.DifferenceItems,
		TotalDifference: st.TotalDifference,
		Progress:        st.GetProgress(),
		CreatedAt:       st.CreatedAt,
		UpdatedAt:       st.UpdatedAt,
	}
}

// ToStockTakingListResponses converts a slice of domain StockTakings to list responses
func ToStockTakingListResponses(sts []inventory.StockTaking) []StockTakingListResponse {
	responses := make([]StockTakingListResponse, len(sts))
	for i := range sts {
		responses[i] = ToStockTakingListResponse(&sts[i])
	}
	return responses
}

// ToStockTakingProgressResponse converts domain StockTaking to progress response
func ToStockTakingProgressResponse(st *inventory.StockTaking) StockTakingProgressResponse {
	return StockTakingProgressResponse{
		ID:              st.ID,
		TakingNumber:    st.TakingNumber,
		Status:          string(st.Status),
		TotalItems:      st.TotalItems,
		CountedItems:    st.CountedItems,
		DifferenceItems: st.DifferenceItems,
		TotalDifference: st.TotalDifference,
		Progress:        st.GetProgress(),
		IsComplete:      st.IsComplete(),
	}
}
