// Package parser provides default inference rules for semantic type detection.
package parser

import (
	"regexp"
	"strings"

	"github.com/example/erp/tools/loadgen/internal/circuit"
)

// DefaultInferenceRules returns the default set of inference rules in priority order.
func DefaultInferenceRules() []InferenceRule {
	return []InferenceRule{
		// High priority: exact field name matches
		&ExactFieldNameRule{},

		// Pagination and query parameters (high priority for common API patterns)
		&PaginationRule{},

		// Entity ID patterns from endpoint path
		&EndpointEntityRule{},

		// Field name suffix patterns (e.g., customer_id, product_code)
		&FieldSuffixRule{},

		// Format-based inference (uuid, email, date-time)
		&FormatRule{},

		// Common field name patterns
		&CommonFieldRule{},

		// Fallback: generic type inference
		&GenericTypeRule{},
	}
}

// ExactFieldNameRule matches exact field names to semantic types.
type ExactFieldNameRule struct{}

func (r *ExactFieldNameRule) Name() string { return "exact_field_name" }

func (r *ExactFieldNameRule) Match(fieldName string, dataType string, format string, ctx *InferenceContext) *InferenceResult {
	normalized := strings.ToLower(fieldName)

	// Exact matches for common fields
	exactMatches := map[string]circuit.SemanticType{
		// IDs
		"id":   circuit.CommonID,
		"uuid": circuit.CommonUUID,

		// Names
		"name":        circuit.CommonName,
		"title":       circuit.CommonTitle,
		"description": circuit.CommonDescription,

		// Status
		"status": circuit.CommonStatus,
		"state":  circuit.CommonState,
		"type":   circuit.CommonType,

		// Quantities
		"quantity": circuit.CommonQuantity,
		"amount":   circuit.CommonAmount,
		"count":    circuit.CommonCount,
		"total":    circuit.CommonTotal,

		// Dates
		"date":       circuit.CommonDate,
		"time":       circuit.CommonTime,
		"datetime":   circuit.CommonDateTime,
		"timestamp":  circuit.CommonTimestamp,
		"created_at": circuit.CommonCreatedAt,
		"updated_at": circuit.CommonUpdatedAt,
		"createdat":  circuit.CommonCreatedAt,
		"updatedat":  circuit.CommonUpdatedAt,

		// Contact
		"email":   circuit.CommonEmail,
		"phone":   circuit.CommonPhone,
		"address": circuit.CommonAddress,

		// Pagination
		"page":      circuit.CommonPage,
		"page_size": circuit.CommonPageSize,
		"pagesize":  circuit.CommonPageSize,
		"limit":     circuit.CommonLimit,
		"offset":    circuit.CommonOffset,

		// Sort
		"sort_by":    circuit.CommonSortBy,
		"sortby":     circuit.CommonSortBy,
		"sort_order": circuit.CommonSortOrder,
		"sortorder":  circuit.CommonSortOrder,
		"order_by":   circuit.CommonSortBy,
		"orderby":    circuit.CommonSortBy,

		// Search
		"keyword": circuit.CommonKeyword,
		"query":   circuit.CommonQuery,
		"q":       circuit.CommonQuery,
		"search":  circuit.CommonKeyword,
		"filter":  circuit.CommonFilter,

		// Boolean
		"enabled":  circuit.CommonEnabled,
		"active":   circuit.CommonActive,
		"deleted":  circuit.CommonDeleted,
		"required": circuit.CommonRequired,

		// Financial
		"price":    circuit.CommonPrice,
		"discount": circuit.CommonDiscount,
		"tax":      circuit.CommonTax,
		"rate":     circuit.CommonRate,

		// References
		"parent_id": circuit.CommonParentID,
		"parentid":  circuit.CommonParentID,
		"version":   circuit.CommonVersion,

		// Notes
		"note":    circuit.CommonNote,
		"remark":  circuit.CommonRemark,
		"comment": circuit.CommonComment,
		"notes":   circuit.CommonNote,
		"remarks": circuit.CommonRemark,

		// Auth tokens
		"access_token":  circuit.SystemAccessToken,
		"accesstoken":   circuit.SystemAccessToken,
		"refresh_token": circuit.SystemRefreshToken,
		"refreshtoken":  circuit.SystemRefreshToken,
		"token":         circuit.SystemAccessToken,
		"api_key":       circuit.SystemAPIKey,
		"apikey":        circuit.SystemAPIKey,

		// User fields
		"username": circuit.EntityUserUsername,
		"user_id":  circuit.EntityUserID,
		"userid":   circuit.EntityUserID,
	}

	if semanticType, ok := exactMatches[normalized]; ok {
		return &InferenceResult{
			SemanticType: semanticType,
			Confidence:   0.9,
			Source:       "exact_field_name",
			Reason:       "exact match for field: " + fieldName,
		}
	}

	return nil
}

// EndpointEntityRule infers entity types from endpoint path.
type EndpointEntityRule struct{}

func (r *EndpointEntityRule) Name() string { return "endpoint_entity" }

func (r *EndpointEntityRule) Match(fieldName string, dataType string, format string, ctx *InferenceContext) *InferenceResult {
	if ctx == nil || ctx.EndpointPath == "" {
		return nil
	}

	normalized := strings.ToLower(fieldName)

	// Only match id, code, name, sku fields
	if normalized != "id" && normalized != "code" && normalized != "name" && normalized != "sku" {
		return nil
	}

	// Extract entity from endpoint path
	entity := extractEntityFromPath(ctx.EndpointPath, ctx.EndpointMethod)
	if entity == "" {
		return nil
	}

	// Map entity + field to semantic type
	semanticType := mapEntityField(entity, normalized)
	if semanticType == "" {
		return nil
	}

	confidence := 0.92 // Higher than exact_field_name (0.9) to take priority
	// Higher confidence for POST responses (creating new entity)
	if ctx.EndpointMethod == "POST" && !ctx.IsInput && normalized == "id" {
		confidence = 0.95
	}
	// Higher confidence for path parameters
	if ctx.IsInput && normalized == "id" {
		confidence = 0.93
	}

	return &InferenceResult{
		SemanticType: semanticType,
		Confidence:   confidence,
		Source:       "endpoint_entity",
		Reason:       "inferred from endpoint path: " + ctx.EndpointPath + " -> " + entity + "." + normalized,
	}
}

// extractEntityFromPath extracts the entity name from an API path.
func extractEntityFromPath(path, method string) string {
	// Remove leading /api/v1 or similar
	path = regexp.MustCompile(`^/api/v\d+`).ReplaceAllString(path, "")
	path = strings.TrimPrefix(path, "/")

	// Split path into segments
	segments := strings.Split(path, "/")
	if len(segments) == 0 {
		return ""
	}

	// Find the entity segment (usually first non-parameter segment)
	for _, segment := range segments {
		if segment == "" || strings.HasPrefix(segment, "{") {
			continue
		}
		// Convert plural to singular
		return singularize(segment)
	}

	return ""
}

// singularize converts a plural word to singular.
func singularize(word string) string {
	word = strings.ToLower(word)

	// Common irregular plurals
	irregulars := map[string]string{
		"categories":  "category",
		"companies":   "company",
		"currencies":  "currency",
		"inventories": "inventory",
		"entries":     "entry",
		"policies":    "policy",
		"statuses":    "status",
		"warehouses":  "warehouse",
		"addresses":   "address",
		"processes":   "process",
		"analyses":    "analysis",
		"bases":       "base",
	}

	if singular, ok := irregulars[word]; ok {
		return singular
	}

	// Regular rules
	if strings.HasSuffix(word, "ies") {
		return word[:len(word)-3] + "y"
	}
	if strings.HasSuffix(word, "es") && len(word) > 3 {
		// Check for -ses, -xes, -zes, -ches, -shes
		if strings.HasSuffix(word, "sses") || strings.HasSuffix(word, "xes") ||
			strings.HasSuffix(word, "zes") || strings.HasSuffix(word, "ches") ||
			strings.HasSuffix(word, "shes") {
			return word[:len(word)-2]
		}
		// Words ending in -ses (but not -sses)
		if strings.HasSuffix(word, "ses") && !strings.HasSuffix(word, "sses") {
			return word[:len(word)-1]
		}
	}
	if strings.HasSuffix(word, "s") && !strings.HasSuffix(word, "ss") {
		return word[:len(word)-1]
	}

	return word
}

// mapEntityField maps an entity name and field to a semantic type.
func mapEntityField(entity, field string) circuit.SemanticType {
	// Entity to semantic type mapping
	entityMap := map[string]map[string]circuit.SemanticType{
		"customer": {
			"id":   circuit.EntityCustomerID,
			"code": circuit.EntityCustomerCode,
			"name": circuit.EntityCustomerName,
		},
		"supplier": {
			"id":   circuit.EntitySupplierID,
			"code": circuit.EntitySupplierCode,
			"name": circuit.EntitySupplierName,
		},
		"product": {
			"id":   circuit.EntityProductID,
			"code": circuit.EntityProductCode,
			"sku":  circuit.EntityProductSKU,
			"name": circuit.EntityProductName,
		},
		"category": {
			"id":   circuit.EntityCategoryID,
			"code": circuit.EntityCategoryCode,
			"name": circuit.EntityCategoryName,
		},
		"warehouse": {
			"id":   circuit.EntityWarehouseID,
			"code": circuit.EntityWarehouseCode,
			"name": circuit.EntityWarehouseName,
		},
		"location": {
			"id":   circuit.EntityLocationID,
			"code": circuit.EntityLocationCode,
		},
		"user": {
			"id":   circuit.EntityUserID,
			"name": circuit.EntityUserUsername,
		},
		"role": {
			"id":   circuit.EntityRoleID,
			"code": circuit.EntityRoleCode,
			"name": circuit.EntityRoleName,
		},
		"tenant": {
			"id":   circuit.EntityTenantID,
			"code": circuit.EntityTenantCode,
			"name": circuit.EntityTenantName,
		},
		"unit": {
			"id":   circuit.EntityUnitID,
			"code": circuit.EntityUnitCode,
			"name": circuit.EntityUnitName,
		},
		"brand": {
			"id":   circuit.EntityBrandID,
			"code": circuit.EntityBrandCode,
			"name": circuit.EntityBrandName,
		},
		"sales_order": {
			"id":     circuit.OrderSalesID,
			"number": circuit.OrderSalesNumber,
		},
		"salesorder": {
			"id":     circuit.OrderSalesID,
			"number": circuit.OrderSalesNumber,
		},
		"purchase_order": {
			"id":     circuit.OrderPurchaseID,
			"number": circuit.OrderPurchaseNumber,
		},
		"purchaseorder": {
			"id":     circuit.OrderPurchaseID,
			"number": circuit.OrderPurchaseNumber,
		},
		"payment": {
			"id":     circuit.FinancePaymentID,
			"number": circuit.FinancePaymentNumber,
		},
		"invoice": {
			"id":     circuit.FinanceInvoiceID,
			"number": circuit.FinanceInvoiceNumber,
		},
		"shipment": {
			"id":     circuit.OrderShipmentID,
			"number": circuit.OrderShipmentNumber,
		},
		"receipt": {
			"id":     circuit.OrderReceiptID,
			"number": circuit.OrderReceiptNumber,
		},
		"stock": {
			"id": circuit.InventoryStockID,
		},
		"movement": {
			"id":     circuit.InventoryMovementID,
			"number": circuit.InventoryMovementNumber,
		},
		"adjustment": {
			"id":     circuit.InventoryAdjustmentID,
			"number": circuit.InventoryAdjustmentNumber,
		},
		"account": {
			"id":   circuit.FinanceAccountID,
			"code": circuit.FinanceAccountCode,
		},
		"currency": {
			"id":   circuit.FinanceCurrencyID,
			"code": circuit.FinanceCurrencyCode,
		},
		"permission": {
			"id":   circuit.SystemPermissionID,
			"code": circuit.SystemPermissionCode,
		},
	}

	if fields, ok := entityMap[entity]; ok {
		if semanticType, ok := fields[field]; ok {
			return semanticType
		}
	}

	return ""
}

// FieldSuffixRule matches field name suffixes like customer_id, product_code.
type FieldSuffixRule struct{}

func (r *FieldSuffixRule) Name() string { return "field_suffix" }

func (r *FieldSuffixRule) Match(fieldName string, dataType string, format string, ctx *InferenceContext) *InferenceResult {
	normalized := strings.ToLower(fieldName)

	// Suffix patterns: entity_field
	suffixPatterns := []struct {
		pattern      *regexp.Regexp
		semanticType circuit.SemanticType
		confidence   float64
	}{
		// Customer
		{regexp.MustCompile(`customer[_-]?id$`), circuit.EntityCustomerID, 0.9},
		{regexp.MustCompile(`customer[_-]?code$`), circuit.EntityCustomerCode, 0.9},
		{regexp.MustCompile(`customer[_-]?name$`), circuit.EntityCustomerName, 0.85},

		// Supplier
		{regexp.MustCompile(`supplier[_-]?id$`), circuit.EntitySupplierID, 0.9},
		{regexp.MustCompile(`supplier[_-]?code$`), circuit.EntitySupplierCode, 0.9},
		{regexp.MustCompile(`supplier[_-]?name$`), circuit.EntitySupplierName, 0.85},
		{regexp.MustCompile(`vendor[_-]?id$`), circuit.EntitySupplierID, 0.85},

		// Product
		{regexp.MustCompile(`product[_-]?id$`), circuit.EntityProductID, 0.9},
		{regexp.MustCompile(`product[_-]?code$`), circuit.EntityProductCode, 0.9},
		{regexp.MustCompile(`product[_-]?sku$`), circuit.EntityProductSKU, 0.9},
		{regexp.MustCompile(`product[_-]?name$`), circuit.EntityProductName, 0.85},
		{regexp.MustCompile(`item[_-]?id$`), circuit.EntityProductID, 0.8},
		{regexp.MustCompile(`sku$`), circuit.EntityProductSKU, 0.85},

		// Category
		{regexp.MustCompile(`category[_-]?id$`), circuit.EntityCategoryID, 0.9},
		{regexp.MustCompile(`category[_-]?code$`), circuit.EntityCategoryCode, 0.9},

		// Warehouse
		{regexp.MustCompile(`warehouse[_-]?id$`), circuit.EntityWarehouseID, 0.9},
		{regexp.MustCompile(`warehouse[_-]?code$`), circuit.EntityWarehouseCode, 0.9},

		// Location
		{regexp.MustCompile(`location[_-]?id$`), circuit.EntityLocationID, 0.9},
		{regexp.MustCompile(`location[_-]?code$`), circuit.EntityLocationCode, 0.9},

		// User
		{regexp.MustCompile(`user[_-]?id$`), circuit.EntityUserID, 0.9},
		{regexp.MustCompile(`created[_-]?by$`), circuit.EntityUserID, 0.85},
		{regexp.MustCompile(`updated[_-]?by$`), circuit.EntityUserID, 0.85},
		{regexp.MustCompile(`assigned[_-]?to$`), circuit.EntityUserID, 0.8},

		// Role
		{regexp.MustCompile(`role[_-]?id$`), circuit.EntityRoleID, 0.9},
		{regexp.MustCompile(`role[_-]?code$`), circuit.EntityRoleCode, 0.9},

		// Tenant
		{regexp.MustCompile(`tenant[_-]?id$`), circuit.EntityTenantID, 0.9},
		{regexp.MustCompile(`tenant[_-]?code$`), circuit.EntityTenantCode, 0.9},

		// Unit
		{regexp.MustCompile(`unit[_-]?id$`), circuit.EntityUnitID, 0.9},
		{regexp.MustCompile(`unit[_-]?code$`), circuit.EntityUnitCode, 0.9},

		// Brand
		{regexp.MustCompile(`brand[_-]?id$`), circuit.EntityBrandID, 0.9},
		{regexp.MustCompile(`brand[_-]?code$`), circuit.EntityBrandCode, 0.9},

		// Orders
		{regexp.MustCompile(`sales[_-]?order[_-]?id$`), circuit.OrderSalesID, 0.9},
		{regexp.MustCompile(`order[_-]?id$`), circuit.OrderSalesID, 0.8},
		{regexp.MustCompile(`order[_-]?number$`), circuit.OrderSalesNumber, 0.85},
		{regexp.MustCompile(`purchase[_-]?order[_-]?id$`), circuit.OrderPurchaseID, 0.9},
		{regexp.MustCompile(`po[_-]?id$`), circuit.OrderPurchaseID, 0.8},

		// Finance
		{regexp.MustCompile(`payment[_-]?id$`), circuit.FinancePaymentID, 0.9},
		{regexp.MustCompile(`invoice[_-]?id$`), circuit.FinanceInvoiceID, 0.9},
		{regexp.MustCompile(`account[_-]?id$`), circuit.FinanceAccountID, 0.9},
		{regexp.MustCompile(`currency[_-]?id$`), circuit.FinanceCurrencyID, 0.9},
		{regexp.MustCompile(`currency[_-]?code$`), circuit.FinanceCurrencyCode, 0.9},

		// Inventory
		{regexp.MustCompile(`stock[_-]?id$`), circuit.InventoryStockID, 0.9},
		{regexp.MustCompile(`movement[_-]?id$`), circuit.InventoryMovementID, 0.9},
		{regexp.MustCompile(`batch[_-]?number$`), circuit.InventoryStockBatch, 0.85},
		{regexp.MustCompile(`batch[_-]?no$`), circuit.InventoryStockBatch, 0.85},

		// Generic ID suffix
		{regexp.MustCompile(`[_-]id$`), circuit.CommonID, 0.7},
		{regexp.MustCompile(`[_-]code$`), circuit.CommonCode, 0.7},
		{regexp.MustCompile(`[_-]name$`), circuit.CommonName, 0.7},

		// Parent references
		{regexp.MustCompile(`parent[_-]?id$`), circuit.CommonParentID, 0.9},
		{regexp.MustCompile(`ref[_-]?id$`), circuit.CommonRefID, 0.85},
	}

	for _, sp := range suffixPatterns {
		if sp.pattern.MatchString(normalized) {
			return &InferenceResult{
				SemanticType: sp.semanticType,
				Confidence:   sp.confidence,
				Source:       "field_suffix",
				Reason:       "matched suffix pattern: " + sp.pattern.String(),
			}
		}
	}

	return nil
}

// FormatRule infers semantic types from OpenAPI format hints.
type FormatRule struct{}

func (r *FormatRule) Name() string { return "format" }

func (r *FormatRule) Match(fieldName string, dataType string, format string, ctx *InferenceContext) *InferenceResult {
	if format == "" {
		return nil
	}

	format = strings.ToLower(format)

	formatMap := map[string]struct {
		semanticType circuit.SemanticType
		confidence   float64
	}{
		"uuid":      {circuit.CommonUUID, 0.85},
		"email":     {circuit.CommonEmail, 0.95},
		"date":      {circuit.CommonDate, 0.9},
		"date-time": {circuit.CommonDateTime, 0.9},
		"time":      {circuit.CommonTime, 0.9},
		"uri":       {circuit.CommonAddress, 0.7},
		"phone":     {circuit.CommonPhone, 0.9},
	}

	if mapping, ok := formatMap[format]; ok {
		return &InferenceResult{
			SemanticType: mapping.semanticType,
			Confidence:   mapping.confidence,
			Source:       "format",
			Reason:       "inferred from format: " + format,
		}
	}

	return nil
}

// CommonFieldRule matches common field name patterns.
type CommonFieldRule struct{}

func (r *CommonFieldRule) Name() string { return "common_field" }

func (r *CommonFieldRule) Match(fieldName string, dataType string, format string, ctx *InferenceContext) *InferenceResult {
	normalized := strings.ToLower(fieldName)

	// Common patterns
	patterns := []struct {
		pattern      *regexp.Regexp
		semanticType circuit.SemanticType
		confidence   float64
	}{
		// Timestamps
		{regexp.MustCompile(`^created[_-]?at$`), circuit.CommonCreatedAt, 0.95},
		{regexp.MustCompile(`^updated[_-]?at$`), circuit.CommonUpdatedAt, 0.95},
		{regexp.MustCompile(`^deleted[_-]?at$`), circuit.CommonTimestamp, 0.9},
		{regexp.MustCompile(`[_-]at$`), circuit.CommonTimestamp, 0.75},
		{regexp.MustCompile(`[_-]date$`), circuit.CommonDate, 0.8},
		{regexp.MustCompile(`[_-]time$`), circuit.CommonTime, 0.8},

		// Quantities
		{regexp.MustCompile(`^qty$`), circuit.CommonQuantity, 0.9},
		{regexp.MustCompile(`[_-]qty$`), circuit.CommonQuantity, 0.85},
		{regexp.MustCompile(`[_-]quantity$`), circuit.CommonQuantity, 0.9},
		{regexp.MustCompile(`[_-]count$`), circuit.CommonCount, 0.85},
		{regexp.MustCompile(`[_-]amount$`), circuit.CommonAmount, 0.85},

		// Financial
		{regexp.MustCompile(`^price$`), circuit.CommonPrice, 0.9},
		{regexp.MustCompile(`[_-]price$`), circuit.CommonPrice, 0.85},
		{regexp.MustCompile(`^unit[_-]?price$`), circuit.CommonPrice, 0.9},
		{regexp.MustCompile(`^total[_-]?amount$`), circuit.CommonAmount, 0.9},
		{regexp.MustCompile(`^sub[_-]?total$`), circuit.CommonAmount, 0.85},
		{regexp.MustCompile(`^grand[_-]?total$`), circuit.CommonAmount, 0.85},
		{regexp.MustCompile(`[_-]tax$`), circuit.CommonTax, 0.85},
		{regexp.MustCompile(`[_-]discount$`), circuit.CommonDiscount, 0.85},

		// Status
		{regexp.MustCompile(`[_-]status$`), circuit.CommonStatus, 0.85},
		{regexp.MustCompile(`[_-]state$`), circuit.CommonState, 0.85},
		{regexp.MustCompile(`[_-]type$`), circuit.CommonType, 0.8},

		// Boolean
		{regexp.MustCompile(`^is[_-]`), circuit.CommonEnabled, 0.75},
		{regexp.MustCompile(`^has[_-]`), circuit.CommonEnabled, 0.75},
		{regexp.MustCompile(`^can[_-]`), circuit.CommonEnabled, 0.75},

		// Notes
		{regexp.MustCompile(`[_-]note$`), circuit.CommonNote, 0.85},
		{regexp.MustCompile(`[_-]notes$`), circuit.CommonNote, 0.85},
		{regexp.MustCompile(`[_-]remark$`), circuit.CommonRemark, 0.85},
		{regexp.MustCompile(`[_-]comment$`), circuit.CommonComment, 0.85},
		{regexp.MustCompile(`[_-]description$`), circuit.CommonDescription, 0.85},
	}

	for _, p := range patterns {
		if p.pattern.MatchString(normalized) {
			return &InferenceResult{
				SemanticType: p.semanticType,
				Confidence:   p.confidence,
				Source:       "common_field",
				Reason:       "matched common pattern: " + p.pattern.String(),
			}
		}
	}

	return nil
}

// PaginationRule matches pagination-related fields.
type PaginationRule struct{}

func (r *PaginationRule) Name() string { return "pagination" }

func (r *PaginationRule) Match(fieldName string, dataType string, format string, ctx *InferenceContext) *InferenceResult {
	normalized := strings.ToLower(fieldName)

	paginationFields := map[string]circuit.SemanticType{
		"page":        circuit.CommonPage,
		"page_num":    circuit.CommonPage,
		"pagenum":     circuit.CommonPage,
		"page_no":     circuit.CommonPage,
		"pageno":      circuit.CommonPage,
		"page_size":   circuit.CommonPageSize,
		"pagesize":    circuit.CommonPageSize,
		"per_page":    circuit.CommonPageSize,
		"perpage":     circuit.CommonPageSize,
		"limit":       circuit.CommonLimit,
		"offset":      circuit.CommonOffset,
		"skip":        circuit.CommonOffset,
		"total":       circuit.CommonTotal,
		"total_count": circuit.CommonTotal,
		"totalcount":  circuit.CommonTotal,
		"total_items": circuit.CommonTotal,
		"totalitems":  circuit.CommonTotal,
	}

	if semanticType, ok := paginationFields[normalized]; ok {
		return &InferenceResult{
			SemanticType: semanticType,
			Confidence:   0.95,
			Source:       "pagination",
			Reason:       "pagination field: " + fieldName,
		}
	}

	return nil
}

// GenericTypeRule provides fallback inference based on data type.
type GenericTypeRule struct{}

func (r *GenericTypeRule) Name() string { return "generic_type" }

func (r *GenericTypeRule) Match(fieldName string, dataType string, format string, ctx *InferenceContext) *InferenceResult {
	normalized := strings.ToLower(fieldName)

	// Very generic patterns with low confidence
	if strings.Contains(normalized, "email") {
		return &InferenceResult{
			SemanticType: circuit.CommonEmail,
			Confidence:   0.7,
			Source:       "generic_type",
			Reason:       "field name contains 'email'",
		}
	}

	if strings.Contains(normalized, "phone") || strings.Contains(normalized, "mobile") || strings.Contains(normalized, "tel") {
		return &InferenceResult{
			SemanticType: circuit.CommonPhone,
			Confidence:   0.7,
			Source:       "generic_type",
			Reason:       "field name contains phone-related term",
		}
	}

	if strings.Contains(normalized, "address") {
		return &InferenceResult{
			SemanticType: circuit.CommonAddress,
			Confidence:   0.7,
			Source:       "generic_type",
			Reason:       "field name contains 'address'",
		}
	}

	// Data type based inference (very low confidence)
	if dataType == "boolean" {
		return &InferenceResult{
			SemanticType: circuit.CommonEnabled,
			Confidence:   0.5,
			Source:       "generic_type",
			Reason:       "boolean data type",
		}
	}

	return nil
}
