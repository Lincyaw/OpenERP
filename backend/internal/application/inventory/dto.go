package inventory

import (
	"time"

	"github.com/erp/backend/internal/domain/inventory"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// InventoryItemResponse represents an inventory item in API responses
type InventoryItemResponse struct {
	ID                uuid.UUID       `json:"id"`
	TenantID          uuid.UUID       `json:"tenant_id"`
	WarehouseID       uuid.UUID       `json:"warehouse_id"`
	ProductID         uuid.UUID       `json:"product_id"`
	AvailableQuantity decimal.Decimal `json:"available_quantity"`
	LockedQuantity    decimal.Decimal `json:"locked_quantity"`
	TotalQuantity     decimal.Decimal `json:"total_quantity"`
	UnitCost          decimal.Decimal `json:"unit_cost"`
	TotalValue        decimal.Decimal `json:"total_value"`
	MinQuantity       decimal.Decimal `json:"min_quantity"`
	MaxQuantity       decimal.Decimal `json:"max_quantity"`
	IsBelowMinimum    bool            `json:"is_below_minimum"`
	IsAboveMaximum    bool            `json:"is_above_maximum"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	Version           int             `json:"version"`
}

// InventoryListItemResponse represents an inventory list item
type InventoryListItemResponse struct {
	ID                uuid.UUID       `json:"id"`
	WarehouseID       uuid.UUID       `json:"warehouse_id"`
	ProductID         uuid.UUID       `json:"product_id"`
	AvailableQuantity decimal.Decimal `json:"available_quantity"`
	LockedQuantity    decimal.Decimal `json:"locked_quantity"`
	TotalQuantity     decimal.Decimal `json:"total_quantity"`
	UnitCost          decimal.Decimal `json:"unit_cost"`
	TotalValue        decimal.Decimal `json:"total_value"`
	MinQuantity       decimal.Decimal `json:"min_quantity"`
	IsBelowMinimum    bool            `json:"is_below_minimum"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// InventoryListFilter represents filter options for inventory list
type InventoryListFilter struct {
	Search       string     `form:"search"`
	WarehouseID  *uuid.UUID `form:"warehouse_id"`
	ProductID    *uuid.UUID `form:"product_id"`
	BelowMinimum *bool      `form:"below_minimum"`
	HasStock     *bool      `form:"has_stock"`
	MinQuantity  *float64   `form:"min_quantity"`
	MaxQuantity  *float64   `form:"max_quantity"`
	Page         int        `form:"page" binding:"min=1"`
	PageSize     int        `form:"page_size" binding:"min=1,max=100"`
	OrderBy      string     `form:"order_by"`
	OrderDir     string     `form:"order_dir" binding:"omitempty,oneof=asc desc"`
}

// IncreaseStockRequest represents a request to increase stock
type IncreaseStockRequest struct {
	WarehouseID uuid.UUID       `json:"warehouse_id" binding:"required"`
	ProductID   uuid.UUID       `json:"product_id" binding:"required"`
	Quantity    decimal.Decimal `json:"quantity" binding:"required"`
	UnitCost    decimal.Decimal `json:"unit_cost" binding:"required"`
	SourceType  string          `json:"source_type" binding:"required"` // PURCHASE_ORDER, SALES_RETURN, INITIAL_STOCK, etc.
	SourceID    string          `json:"source_id" binding:"required"`
	BatchNumber string          `json:"batch_number"`
	ExpiryDate  *time.Time      `json:"expiry_date"`
	Reference   string          `json:"reference"`
	Reason      string          `json:"reason"`
	OperatorID  *uuid.UUID      `json:"operator_id"`
}

// LockStockRequest represents a request to lock stock
type LockStockRequest struct {
	WarehouseID uuid.UUID       `json:"warehouse_id" binding:"required"`
	ProductID   uuid.UUID       `json:"product_id" binding:"required"`
	Quantity    decimal.Decimal `json:"quantity" binding:"required"`
	SourceType  string          `json:"source_type" binding:"required"` // e.g., "sales_order"
	SourceID    string          `json:"source_id" binding:"required"`
	ExpireAt    *time.Time      `json:"expire_at"` // Optional, defaults to 30 minutes
}

// LockStockResponse represents the response after locking stock
type LockStockResponse struct {
	LockID          uuid.UUID       `json:"lock_id"`
	InventoryItemID uuid.UUID       `json:"inventory_item_id"`
	WarehouseID     uuid.UUID       `json:"warehouse_id"`
	ProductID       uuid.UUID       `json:"product_id"`
	Quantity        decimal.Decimal `json:"quantity"`
	ExpireAt        time.Time       `json:"expire_at"`
	SourceType      string          `json:"source_type"`
	SourceID        string          `json:"source_id"`
}

// UnlockStockRequest represents a request to unlock stock
type UnlockStockRequest struct {
	LockID uuid.UUID `json:"lock_id" binding:"required"`
}

// DeductStockRequest represents a request to deduct locked stock
type DeductStockRequest struct {
	LockID     uuid.UUID  `json:"lock_id" binding:"required"`
	SourceType string     `json:"source_type" binding:"required"` // e.g., "SALES_ORDER"
	SourceID   string     `json:"source_id" binding:"required"`
	Reference  string     `json:"reference"`
	OperatorID *uuid.UUID `json:"operator_id"`
}

// AdjustStockRequest represents a request to adjust stock
type AdjustStockRequest struct {
	WarehouseID    uuid.UUID       `json:"warehouse_id" binding:"required"`
	ProductID      uuid.UUID       `json:"product_id" binding:"required"`
	ActualQuantity decimal.Decimal `json:"actual_quantity" binding:"required"`
	Reason         string          `json:"reason" binding:"required,min=1,max=255"`
	SourceType     string          `json:"source_type"` // defaults to MANUAL_ADJUSTMENT
	SourceID       string          `json:"source_id"`   // auto-generated if empty
	OperatorID     *uuid.UUID      `json:"operator_id"`
}

// SetThresholdsRequest represents a request to set min/max quantity thresholds
type SetThresholdsRequest struct {
	WarehouseID uuid.UUID        `json:"warehouse_id" binding:"required"`
	ProductID   uuid.UUID        `json:"product_id" binding:"required"`
	MinQuantity *decimal.Decimal `json:"min_quantity"`
	MaxQuantity *decimal.Decimal `json:"max_quantity"`
}

// StockLockResponse represents a stock lock in API responses
type StockLockResponse struct {
	ID              uuid.UUID       `json:"id"`
	InventoryItemID uuid.UUID       `json:"inventory_item_id"`
	Quantity        decimal.Decimal `json:"quantity"`
	SourceType      string          `json:"source_type"`
	SourceID        string          `json:"source_id"`
	ExpireAt        time.Time       `json:"expire_at"`
	Released        bool            `json:"released"`
	Consumed        bool            `json:"consumed"`
	IsActive        bool            `json:"is_active"`
	IsExpired       bool            `json:"is_expired"`
	CreatedAt       time.Time       `json:"created_at"`
}

// TransactionResponse represents an inventory transaction in API responses
type TransactionResponse struct {
	ID              uuid.UUID       `json:"id"`
	TenantID        uuid.UUID       `json:"tenant_id"`
	InventoryItemID uuid.UUID       `json:"inventory_item_id"`
	WarehouseID     uuid.UUID       `json:"warehouse_id"`
	ProductID       uuid.UUID       `json:"product_id"`
	TransactionType string          `json:"transaction_type"`
	Quantity        decimal.Decimal `json:"quantity"`
	SignedQuantity  decimal.Decimal `json:"signed_quantity"`
	UnitCost        decimal.Decimal `json:"unit_cost"`
	TotalCost       decimal.Decimal `json:"total_cost"`
	BalanceBefore   decimal.Decimal `json:"balance_before"`
	BalanceAfter    decimal.Decimal `json:"balance_after"`
	SourceType      string          `json:"source_type"`
	SourceID        string          `json:"source_id"`
	SourceLineID    string          `json:"source_line_id,omitempty"`
	BatchID         *uuid.UUID      `json:"batch_id,omitempty"`
	LockID          *uuid.UUID      `json:"lock_id,omitempty"`
	Reference       string          `json:"reference,omitempty"`
	Reason          string          `json:"reason,omitempty"`
	OperatorID      *uuid.UUID      `json:"operator_id,omitempty"`
	TransactionDate time.Time       `json:"transaction_date"`
	CreatedAt       time.Time       `json:"created_at"`
}

// TransactionListFilter represents filter options for transaction list
type TransactionListFilter struct {
	WarehouseID     *uuid.UUID `form:"warehouse_id"`
	ProductID       *uuid.UUID `form:"product_id"`
	TransactionType string     `form:"transaction_type"`
	SourceType      string     `form:"source_type"`
	SourceID        string     `form:"source_id"`
	StartDate       *time.Time `form:"start_date"`
	EndDate         *time.Time `form:"end_date"`
	Page            int        `form:"page" binding:"min=1"`
	PageSize        int        `form:"page_size" binding:"min=1,max=100"`
	OrderBy         string     `form:"order_by"`
	OrderDir        string     `form:"order_dir" binding:"omitempty,oneof=asc desc"`
}

// InventorySummaryResponse represents inventory summary statistics
type InventorySummaryResponse struct {
	TotalItems         int64              `json:"total_items"`
	TotalValue         decimal.Decimal    `json:"total_value"`
	ItemsBelowMinimum  int64              `json:"items_below_minimum"`
	TotalAvailable     decimal.Decimal    `json:"total_available"`
	TotalLocked        decimal.Decimal    `json:"total_locked"`
	WarehouseBreakdown []WarehouseSummary `json:"warehouse_breakdown,omitempty"`
}

// WarehouseSummary represents inventory summary for a warehouse
type WarehouseSummary struct {
	WarehouseID  uuid.UUID       `json:"warehouse_id"`
	ItemCount    int64           `json:"item_count"`
	TotalValue   decimal.Decimal `json:"total_value"`
	BelowMinimum int64           `json:"below_minimum"`
}

// ToInventoryItemResponse converts domain InventoryItem to response DTO
func ToInventoryItemResponse(item *inventory.InventoryItem) InventoryItemResponse {
	return InventoryItemResponse{
		ID:                item.ID,
		TenantID:          item.TenantID,
		WarehouseID:       item.WarehouseID,
		ProductID:         item.ProductID,
		AvailableQuantity: item.AvailableQuantity,
		LockedQuantity:    item.LockedQuantity,
		TotalQuantity:     item.TotalQuantity(),
		UnitCost:          item.UnitCost,
		TotalValue:        item.GetTotalValue().Amount(),
		MinQuantity:       item.MinQuantity,
		MaxQuantity:       item.MaxQuantity,
		IsBelowMinimum:    item.IsBelowMinimum(),
		IsAboveMaximum:    item.IsAboveMaximum(),
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
		Version:           item.Version,
	}
}

// ToInventoryListItemResponse converts domain InventoryItem to list response DTO
func ToInventoryListItemResponse(item *inventory.InventoryItem) InventoryListItemResponse {
	return InventoryListItemResponse{
		ID:                item.ID,
		WarehouseID:       item.WarehouseID,
		ProductID:         item.ProductID,
		AvailableQuantity: item.AvailableQuantity,
		LockedQuantity:    item.LockedQuantity,
		TotalQuantity:     item.TotalQuantity(),
		UnitCost:          item.UnitCost,
		TotalValue:        item.GetTotalValue().Amount(),
		MinQuantity:       item.MinQuantity,
		IsBelowMinimum:    item.IsBelowMinimum(),
		UpdatedAt:         item.UpdatedAt,
	}
}

// ToInventoryListItemResponses converts a slice of domain InventoryItems to list responses
func ToInventoryListItemResponses(items []inventory.InventoryItem) []InventoryListItemResponse {
	responses := make([]InventoryListItemResponse, len(items))
	for i := range items {
		responses[i] = ToInventoryListItemResponse(&items[i])
	}
	return responses
}

// ToStockLockResponse converts domain StockLock to response DTO
func ToStockLockResponse(lock *inventory.StockLock) StockLockResponse {
	return StockLockResponse{
		ID:              lock.ID,
		InventoryItemID: lock.InventoryItemID,
		Quantity:        lock.Quantity,
		SourceType:      lock.SourceType,
		SourceID:        lock.SourceID,
		ExpireAt:        lock.ExpireAt,
		Released:        lock.Released,
		Consumed:        lock.Consumed,
		IsActive:        lock.IsActive(),
		IsExpired:       lock.IsExpired(),
		CreatedAt:       lock.CreatedAt,
	}
}

// ToStockLockResponses converts a slice of domain StockLocks to responses
func ToStockLockResponses(locks []inventory.StockLock) []StockLockResponse {
	responses := make([]StockLockResponse, len(locks))
	for i := range locks {
		responses[i] = ToStockLockResponse(&locks[i])
	}
	return responses
}

// ToTransactionResponse converts domain InventoryTransaction to response DTO
func ToTransactionResponse(tx *inventory.InventoryTransaction) TransactionResponse {
	return TransactionResponse{
		ID:              tx.ID,
		TenantID:        tx.TenantID,
		InventoryItemID: tx.InventoryItemID,
		WarehouseID:     tx.WarehouseID,
		ProductID:       tx.ProductID,
		TransactionType: string(tx.TransactionType),
		Quantity:        tx.Quantity,
		SignedQuantity:  tx.GetSignedQuantity(),
		UnitCost:        tx.UnitCost,
		TotalCost:       tx.TotalCost,
		BalanceBefore:   tx.BalanceBefore,
		BalanceAfter:    tx.BalanceAfter,
		SourceType:      string(tx.SourceType),
		SourceID:        tx.SourceID,
		SourceLineID:    tx.SourceLineID,
		BatchID:         tx.BatchID,
		LockID:          tx.LockID,
		Reference:       tx.Reference,
		Reason:          tx.Reason,
		OperatorID:      tx.OperatorID,
		TransactionDate: tx.TransactionDate,
		CreatedAt:       tx.CreatedAt,
	}
}

// ToTransactionResponses converts a slice of domain transactions to responses
func ToTransactionResponses(txs []inventory.InventoryTransaction) []TransactionResponse {
	responses := make([]TransactionResponse, len(txs))
	for i := range txs {
		responses[i] = ToTransactionResponse(&txs[i])
	}
	return responses
}
