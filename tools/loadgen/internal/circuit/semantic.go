// Package circuit provides circuit-board-like components for the load generator.
// This file defines semantic types for categorizing API parameters and responses.
package circuit

// SemanticType represents the semantic classification of a parameter or response field.
// It follows a hierarchical naming convention: category.entity.field
// Examples: entity.customer.id, order.sales.number, finance.payment.amount
type SemanticType string

// Semantic type categories
const (
	// CategoryEntity represents entity-related semantic types
	CategoryEntity = "entity"
	// CategoryOrder represents order-related semantic types
	CategoryOrder = "order"
	// CategoryFinance represents finance-related semantic types
	CategoryFinance = "finance"
	// CategoryCommon represents common/shared semantic types
	CategoryCommon = "common"
	// CategoryInventory represents inventory-related semantic types
	CategoryInventory = "inventory"
	// CategorySystem represents system-related semantic types
	CategorySystem = "system"
)

// Entity semantic types - identifiers for domain entities
const (
	// Customer entity types
	EntityCustomerID   SemanticType = "entity.customer.id"
	EntityCustomerCode SemanticType = "entity.customer.code"
	EntityCustomerName SemanticType = "entity.customer.name"

	// Supplier entity types
	EntitySupplierID   SemanticType = "entity.supplier.id"
	EntitySupplierCode SemanticType = "entity.supplier.code"
	EntitySupplierName SemanticType = "entity.supplier.name"

	// Product entity types
	EntityProductID   SemanticType = "entity.product.id"
	EntityProductCode SemanticType = "entity.product.code"
	EntityProductSKU  SemanticType = "entity.product.sku"
	EntityProductName SemanticType = "entity.product.name"

	// Category entity types
	EntityCategoryID   SemanticType = "entity.category.id"
	EntityCategoryCode SemanticType = "entity.category.code"
	EntityCategoryName SemanticType = "entity.category.name"

	// Warehouse entity types
	EntityWarehouseID   SemanticType = "entity.warehouse.id"
	EntityWarehouseCode SemanticType = "entity.warehouse.code"
	EntityWarehouseName SemanticType = "entity.warehouse.name"

	// Location entity types
	EntityLocationID   SemanticType = "entity.location.id"
	EntityLocationCode SemanticType = "entity.location.code"

	// User entity types
	EntityUserID       SemanticType = "entity.user.id"
	EntityUserUsername SemanticType = "entity.user.username"
	EntityUserEmail    SemanticType = "entity.user.email"

	// Role entity types
	EntityRoleID   SemanticType = "entity.role.id"
	EntityRoleCode SemanticType = "entity.role.code"
	EntityRoleName SemanticType = "entity.role.name"

	// Tenant entity types
	EntityTenantID   SemanticType = "entity.tenant.id"
	EntityTenantCode SemanticType = "entity.tenant.code"
	EntityTenantName SemanticType = "entity.tenant.name"

	// Unit entity types
	EntityUnitID   SemanticType = "entity.unit.id"
	EntityUnitCode SemanticType = "entity.unit.code"
	EntityUnitName SemanticType = "entity.unit.name"

	// Brand entity types
	EntityBrandID   SemanticType = "entity.brand.id"
	EntityBrandCode SemanticType = "entity.brand.code"
	EntityBrandName SemanticType = "entity.brand.name"
)

// Order semantic types - order-related fields
const (
	// Sales order types
	OrderSalesID     SemanticType = "order.sales.id"
	OrderSalesNumber SemanticType = "order.sales.number"
	OrderSalesStatus SemanticType = "order.sales.status"

	// Purchase order types
	OrderPurchaseID     SemanticType = "order.purchase.id"
	OrderPurchaseNumber SemanticType = "order.purchase.number"
	OrderPurchaseStatus SemanticType = "order.purchase.status"

	// Order item types
	OrderItemID       SemanticType = "order.item.id"
	OrderItemQuantity SemanticType = "order.item.quantity"
	OrderItemPrice    SemanticType = "order.item.price"

	// Shipment types
	OrderShipmentID     SemanticType = "order.shipment.id"
	OrderShipmentNumber SemanticType = "order.shipment.number"
	OrderShipmentStatus SemanticType = "order.shipment.status"

	// Receipt types
	OrderReceiptID     SemanticType = "order.receipt.id"
	OrderReceiptNumber SemanticType = "order.receipt.number"
	OrderReceiptStatus SemanticType = "order.receipt.status"
)

// Finance semantic types - financial fields
const (
	// Payment types
	FinancePaymentID     SemanticType = "finance.payment.id"
	FinancePaymentNumber SemanticType = "finance.payment.number"
	FinancePaymentAmount SemanticType = "finance.payment.amount"
	FinancePaymentStatus SemanticType = "finance.payment.status"

	// Invoice types
	FinanceInvoiceID     SemanticType = "finance.invoice.id"
	FinanceInvoiceNumber SemanticType = "finance.invoice.number"
	FinanceInvoiceAmount SemanticType = "finance.invoice.amount"
	FinanceInvoiceStatus SemanticType = "finance.invoice.status"

	// Account types
	FinanceAccountID      SemanticType = "finance.account.id"
	FinanceAccountCode    SemanticType = "finance.account.code"
	FinanceAccountBalance SemanticType = "finance.account.balance"

	// Currency types
	FinanceCurrencyID   SemanticType = "finance.currency.id"
	FinanceCurrencyCode SemanticType = "finance.currency.code"
	FinanceCurrencyRate SemanticType = "finance.currency.rate"

	// Price types
	FinancePriceID     SemanticType = "finance.price.id"
	FinancePriceAmount SemanticType = "finance.price.amount"
	FinancePriceType   SemanticType = "finance.price.type"
)

// Inventory semantic types - inventory-related fields
const (
	// Stock types
	InventoryStockID       SemanticType = "inventory.stock.id"
	InventoryStockQuantity SemanticType = "inventory.stock.quantity"
	InventoryStockBatch    SemanticType = "inventory.stock.batch"

	// Movement types
	InventoryMovementID     SemanticType = "inventory.movement.id"
	InventoryMovementNumber SemanticType = "inventory.movement.number"
	InventoryMovementType   SemanticType = "inventory.movement.type"

	// Adjustment types
	InventoryAdjustmentID     SemanticType = "inventory.adjustment.id"
	InventoryAdjustmentNumber SemanticType = "inventory.adjustment.number"
	InventoryAdjustmentReason SemanticType = "inventory.adjustment.reason"
)

// Common semantic types - shared/generic fields
const (
	// Identifier types
	CommonID   SemanticType = "common.id"
	CommonUUID SemanticType = "common.uuid"
	CommonCode SemanticType = "common.code"

	// Name types
	CommonName        SemanticType = "common.name"
	CommonDescription SemanticType = "common.description"
	CommonTitle       SemanticType = "common.title"

	// Status types
	CommonStatus SemanticType = "common.status"
	CommonState  SemanticType = "common.state"
	CommonType   SemanticType = "common.type"

	// Quantity types
	CommonQuantity SemanticType = "common.quantity"
	CommonAmount   SemanticType = "common.amount"
	CommonCount    SemanticType = "common.count"

	// Date/time types
	CommonDate      SemanticType = "common.date"
	CommonTime      SemanticType = "common.time"
	CommonDateTime  SemanticType = "common.datetime"
	CommonTimestamp SemanticType = "common.timestamp"
	CommonCreatedAt SemanticType = "common.created_at"
	CommonUpdatedAt SemanticType = "common.updated_at"

	// Contact types
	CommonEmail   SemanticType = "common.email"
	CommonPhone   SemanticType = "common.phone"
	CommonAddress SemanticType = "common.address"

	// Pagination types
	CommonPage     SemanticType = "common.page"
	CommonPageSize SemanticType = "common.page_size"
	CommonLimit    SemanticType = "common.limit"
	CommonOffset   SemanticType = "common.offset"
	CommonTotal    SemanticType = "common.total"

	// Sort types
	CommonSortBy    SemanticType = "common.sort_by"
	CommonSortOrder SemanticType = "common.sort_order"

	// Search types
	CommonKeyword SemanticType = "common.keyword"
	CommonQuery   SemanticType = "common.query"
	CommonFilter  SemanticType = "common.filter"

	// Boolean types
	CommonEnabled  SemanticType = "common.enabled"
	CommonActive   SemanticType = "common.active"
	CommonDeleted  SemanticType = "common.deleted"
	CommonRequired SemanticType = "common.required"

	// Numeric types
	CommonPrice    SemanticType = "common.price"
	CommonDiscount SemanticType = "common.discount"
	CommonTax      SemanticType = "common.tax"
	CommonRate     SemanticType = "common.rate"
	CommonPercent  SemanticType = "common.percent"

	// Reference types
	CommonParentID SemanticType = "common.parent_id"
	CommonRefID    SemanticType = "common.ref_id"
	CommonVersion  SemanticType = "common.version"

	// Note types
	CommonNote    SemanticType = "common.note"
	CommonRemark  SemanticType = "common.remark"
	CommonComment SemanticType = "common.comment"
)

// System semantic types - system/auth-related fields
const (
	// Auth types
	SystemAccessToken  SemanticType = "system.access_token"
	SystemRefreshToken SemanticType = "system.refresh_token"
	SystemAPIKey       SemanticType = "system.api_key"

	// Permission types
	SystemPermissionID   SemanticType = "system.permission.id"
	SystemPermissionCode SemanticType = "system.permission.code"

	// Audit types
	SystemAuditID     SemanticType = "system.audit.id"
	SystemAuditAction SemanticType = "system.audit.action"
	SystemAuditActor  SemanticType = "system.audit.actor"

	// Config types
	SystemConfigKey   SemanticType = "system.config.key"
	SystemConfigValue SemanticType = "system.config.value"
)

// UnknownSemanticType represents an unclassified semantic type
const UnknownSemanticType SemanticType = "unknown"

// String returns the string representation of the semantic type.
func (s SemanticType) String() string {
	return string(s)
}

// Category returns the category part of the semantic type.
// For "entity.customer.id", returns "entity".
func (s SemanticType) Category() string {
	str := string(s)
	for i := 0; i < len(str); i++ {
		if str[i] == '.' {
			return str[:i]
		}
	}
	return str
}

// Entity returns the entity part of the semantic type.
// For "entity.customer.id", returns "customer".
func (s SemanticType) Entity() string {
	str := string(s)
	firstDot := -1
	for i := 0; i < len(str); i++ {
		if str[i] == '.' {
			if firstDot == -1 {
				firstDot = i
			} else {
				return str[firstDot+1 : i]
			}
		}
	}
	if firstDot != -1 {
		return str[firstDot+1:]
	}
	return ""
}

// Field returns the field part of the semantic type.
// For "entity.customer.id", returns "id".
func (s SemanticType) Field() string {
	str := string(s)
	lastDot := -1
	for i := len(str) - 1; i >= 0; i-- {
		if str[i] == '.' {
			lastDot = i
			break
		}
	}
	if lastDot != -1 && lastDot < len(str)-1 {
		return str[lastDot+1:]
	}
	return ""
}

// IsEntity returns true if this is an entity semantic type.
func (s SemanticType) IsEntity() bool {
	return s.Category() == CategoryEntity
}

// IsID returns true if this semantic type represents an identifier.
func (s SemanticType) IsID() bool {
	field := s.Field()
	return field == "id" || field == "uuid"
}

// IsCode returns true if this semantic type represents a code.
func (s SemanticType) IsCode() bool {
	return s.Field() == "code"
}

// IsName returns true if this semantic type represents a name.
func (s SemanticType) IsName() bool {
	return s.Field() == "name"
}

// AllSemanticTypes returns all defined semantic types for reference.
func AllSemanticTypes() []SemanticType {
	return []SemanticType{
		// Entity types
		EntityCustomerID, EntityCustomerCode, EntityCustomerName,
		EntitySupplierID, EntitySupplierCode, EntitySupplierName,
		EntityProductID, EntityProductCode, EntityProductSKU, EntityProductName,
		EntityCategoryID, EntityCategoryCode, EntityCategoryName,
		EntityWarehouseID, EntityWarehouseCode, EntityWarehouseName,
		EntityLocationID, EntityLocationCode,
		EntityUserID, EntityUserUsername, EntityUserEmail,
		EntityRoleID, EntityRoleCode, EntityRoleName,
		EntityTenantID, EntityTenantCode, EntityTenantName,
		EntityUnitID, EntityUnitCode, EntityUnitName,
		EntityBrandID, EntityBrandCode, EntityBrandName,

		// Order types
		OrderSalesID, OrderSalesNumber, OrderSalesStatus,
		OrderPurchaseID, OrderPurchaseNumber, OrderPurchaseStatus,
		OrderItemID, OrderItemQuantity, OrderItemPrice,
		OrderShipmentID, OrderShipmentNumber, OrderShipmentStatus,
		OrderReceiptID, OrderReceiptNumber, OrderReceiptStatus,

		// Finance types
		FinancePaymentID, FinancePaymentNumber, FinancePaymentAmount, FinancePaymentStatus,
		FinanceInvoiceID, FinanceInvoiceNumber, FinanceInvoiceAmount, FinanceInvoiceStatus,
		FinanceAccountID, FinanceAccountCode, FinanceAccountBalance,
		FinanceCurrencyID, FinanceCurrencyCode, FinanceCurrencyRate,
		FinancePriceID, FinancePriceAmount, FinancePriceType,

		// Inventory types
		InventoryStockID, InventoryStockQuantity, InventoryStockBatch,
		InventoryMovementID, InventoryMovementNumber, InventoryMovementType,
		InventoryAdjustmentID, InventoryAdjustmentNumber, InventoryAdjustmentReason,

		// Common types
		CommonID, CommonUUID, CommonCode,
		CommonName, CommonDescription, CommonTitle,
		CommonStatus, CommonState, CommonType,
		CommonQuantity, CommonAmount, CommonCount,
		CommonDate, CommonTime, CommonDateTime, CommonTimestamp, CommonCreatedAt, CommonUpdatedAt,
		CommonEmail, CommonPhone, CommonAddress,
		CommonPage, CommonPageSize, CommonLimit, CommonOffset, CommonTotal,
		CommonSortBy, CommonSortOrder,
		CommonKeyword, CommonQuery, CommonFilter,
		CommonEnabled, CommonActive, CommonDeleted, CommonRequired,
		CommonPrice, CommonDiscount, CommonTax, CommonRate, CommonPercent,
		CommonParentID, CommonRefID, CommonVersion,
		CommonNote, CommonRemark, CommonComment,

		// System types
		SystemAccessToken, SystemRefreshToken, SystemAPIKey,
		SystemPermissionID, SystemPermissionCode,
		SystemAuditID, SystemAuditAction, SystemAuditActor,
		SystemConfigKey, SystemConfigValue,
	}
}
