package persistence

import (
	"strings"
)

// ValidateSortOrder validates and normalizes the sort order to ASC or DESC.
// Returns "DESC" as the default if the input is invalid or empty.
func ValidateSortOrder(orderDir string) string {
	normalized := strings.ToUpper(strings.TrimSpace(orderDir))
	if normalized == "ASC" {
		return "ASC"
	}
	return "DESC"
}

// ValidateSortField validates the sort field against a whitelist of allowed fields.
// Returns the defaultField if the input is invalid, empty, or not in the whitelist.
func ValidateSortField(sortField string, allowedFields map[string]bool, defaultField string) string {
	trimmed := strings.TrimSpace(sortField)
	if trimmed == "" {
		return defaultField
	}
	if allowedFields[trimmed] {
		return trimmed
	}
	return defaultField
}

// Common allowed sort fields for entities with base fields
// These are the common fields present in most entities

// CommonSortFields contains fields common to most entities
var CommonSortFields = map[string]bool{
	"id":         true,
	"created_at": true,
	"updated_at": true,
}

// UserSortFields contains allowed sort fields for users
var UserSortFields = map[string]bool{
	"id":            true,
	"created_at":    true,
	"updated_at":    true,
	"username":      true,
	"email":         true,
	"display_name":  true,
	"status":        true,
	"last_login_at": true,
}

// TenantSortFields contains allowed sort fields for tenants
var TenantSortFields = map[string]bool{
	"id":         true,
	"created_at": true,
	"updated_at": true,
	"code":       true,
	"name":       true,
	"short_name": true,
	"status":     true,
	"plan":       true,
	"expires_at": true,
}

// CustomerSortFields contains allowed sort fields for customers
var CustomerSortFields = map[string]bool{
	"id":              true,
	"created_at":      true,
	"updated_at":      true,
	"code":            true,
	"name":            true,
	"contact_name":    true,
	"contact_phone":   true,
	"status":          true,
	"customer_level":  true,
	"total_purchases": true,
	"balance":         true,
	"credit_limit":    true,
}

// ProductSortFields contains allowed sort fields for products
var ProductSortFields = map[string]bool{
	"id":           true,
	"created_at":   true,
	"updated_at":   true,
	"code":         true,
	"name":         true,
	"category_id":  true,
	"status":       true,
	"base_price":   true,
	"cost_price":   true,
	"sale_price":   true,
	"stock_qty":    true,
	"min_stock":    true,
	"max_stock":    true,
	"barcode":      true,
	"sku":          true,
	"sort_order":   true,
	"is_available": true,
}

// CategorySortFields contains allowed sort fields for categories
var CategorySortFields = map[string]bool{
	"id":            true,
	"created_at":    true,
	"updated_at":    true,
	"code":          true,
	"name":          true,
	"parent_id":     true,
	"level":         true,
	"sort_order":    true,
	"status":        true,
	"product_count": true,
}

// SupplierSortFields contains allowed sort fields for suppliers
var SupplierSortFields = map[string]bool{
	"id":            true,
	"created_at":    true,
	"updated_at":    true,
	"code":          true,
	"name":          true,
	"contact_name":  true,
	"contact_phone": true,
	"status":        true,
	"balance":       true,
	"credit_limit":  true,
}

// WarehouseSortFields contains allowed sort fields for warehouses
var WarehouseSortFields = map[string]bool{
	"id":            true,
	"created_at":    true,
	"updated_at":    true,
	"code":          true,
	"name":          true,
	"status":        true,
	"type":          true,
	"is_default":    true,
	"sort_order":    true,
	"contact_name":  true,
	"contact_phone": true,
	"address":       true,
}

// InventorySortFields contains allowed sort fields for inventory
var InventorySortFields = map[string]bool{
	"id":             true,
	"created_at":     true,
	"updated_at":     true,
	"product_id":     true,
	"warehouse_id":   true,
	"quantity":       true,
	"available_qty":  true,
	"locked_qty":     true,
	"cost":           true,
	"product_code":   true,
	"product_name":   true,
	"warehouse_name": true,
}

// InventoryTransactionSortFields contains allowed sort fields for inventory transactions
var InventoryTransactionSortFields = map[string]bool{
	"id":               true,
	"created_at":       true,
	"updated_at":       true,
	"transaction_type": true,
	"product_id":       true,
	"warehouse_id":     true,
	"quantity":         true,
	"reference_type":   true,
	"reference_id":     true,
}

// StockBatchSortFields contains allowed sort fields for stock batches
var StockBatchSortFields = map[string]bool{
	"id":              true,
	"created_at":      true,
	"updated_at":      true,
	"batch_number":    true,
	"product_id":      true,
	"warehouse_id":    true,
	"quantity":        true,
	"available_qty":   true,
	"cost_price":      true,
	"production_date": true,
	"expiry_date":     true,
}

// SalesOrderSortFields contains allowed sort fields for sales orders
var SalesOrderSortFields = map[string]bool{
	"id":              true,
	"created_at":      true,
	"updated_at":      true,
	"order_number":    true,
	"customer_id":     true,
	"customer_name":   true,
	"status":          true,
	"total_amount":    true,
	"discount_amount": true,
	"payable_amount":  true,
	"confirmed_at":    true,
	"shipped_at":      true,
	"completed_at":    true,
}

// SalesReturnSortFields contains allowed sort fields for sales returns
var SalesReturnSortFields = map[string]bool{
	"id":            true,
	"created_at":    true,
	"updated_at":    true,
	"return_number": true,
	"return_date":   true,
	"customer_id":   true,
	"customer_name": true,
	"status":        true,
	"total_amount":  true,
	"refund_amount": true,
	"reason":        true,
}

// PurchaseOrderSortFields contains allowed sort fields for purchase orders
var PurchaseOrderSortFields = map[string]bool{
	"id":              true,
	"created_at":      true,
	"updated_at":      true,
	"order_number":    true,
	"supplier_id":     true,
	"supplier_name":   true,
	"status":          true,
	"total_amount":    true,
	"discount_amount": true,
	"payable_amount":  true,
	"confirmed_at":    true,
	"completed_at":    true,
}

// PurchaseReturnSortFields contains allowed sort fields for purchase returns
var PurchaseReturnSortFields = map[string]bool{
	"id":            true,
	"created_at":    true,
	"updated_at":    true,
	"return_number": true,
	"return_date":   true,
	"supplier_id":   true,
	"supplier_name": true,
	"status":        true,
	"total_amount":  true,
	"refund_amount": true,
	"reason":        true,
}

// AccountReceivableSortFields contains allowed sort fields for accounts receivable
var AccountReceivableSortFields = map[string]bool{
	"id":            true,
	"created_at":    true,
	"updated_at":    true,
	"customer_id":   true,
	"customer_name": true,
	"order_id":      true,
	"order_number":  true,
	"amount":        true,
	"paid_amount":   true,
	"due_date":      true,
	"status":        true,
	"balance":       true,
}

// AccountPayableSortFields contains allowed sort fields for accounts payable
var AccountPayableSortFields = map[string]bool{
	"id":            true,
	"created_at":    true,
	"updated_at":    true,
	"supplier_id":   true,
	"supplier_name": true,
	"order_id":      true,
	"order_number":  true,
	"amount":        true,
	"paid_amount":   true,
	"due_date":      true,
	"status":        true,
	"balance":       true,
}

// ExpenseRecordSortFields contains allowed sort fields for expense records
var ExpenseRecordSortFields = map[string]bool{
	"id":             true,
	"created_at":     true,
	"updated_at":     true,
	"expense_number": true,
	"expense_date":   true,
	"category":       true,
	"amount":         true,
	"status":         true,
	"description":    true,
	"payment_method": true,
}

// OtherIncomeRecordSortFields contains allowed sort fields for other income records
var OtherIncomeRecordSortFields = map[string]bool{
	"id":             true,
	"created_at":     true,
	"updated_at":     true,
	"income_number":  true,
	"income_date":    true,
	"category":       true,
	"amount":         true,
	"status":         true,
	"description":    true,
	"payment_method": true,
}

// StockTakingSortFields contains allowed sort fields for stock taking
var StockTakingSortFields = map[string]bool{
	"id":             true,
	"created_at":     true,
	"updated_at":     true,
	"taking_number":  true,
	"taking_date":    true,
	"status":         true,
	"warehouse_id":   true,
	"warehouse_name": true,
	"total_items":    true,
}

// RoleSortFields contains allowed sort fields for roles
var RoleSortFields = map[string]bool{
	"id":             true,
	"created_at":     true,
	"updated_at":     true,
	"code":           true,
	"name":           true,
	"sort_order":     true,
	"is_enabled":     true,
	"is_system_role": true,
}
