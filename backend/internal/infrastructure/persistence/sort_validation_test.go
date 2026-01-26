package persistence

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateSortOrder(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string returns DESC", "", "DESC"},
		{"ASC uppercase returns ASC", "ASC", "ASC"},
		{"asc lowercase returns ASC", "asc", "ASC"},
		{"DESC uppercase returns DESC", "DESC", "DESC"},
		{"desc lowercase returns DESC", "DESC", "DESC"},
		{"invalid value returns DESC", "INVALID", "DESC"},
		{"sql injection attempt returns DESC", "ASC; DROP TABLE users;--", "DESC"},
		{"whitespace only returns DESC", "   ", "DESC"},
		{"whitespace around ASC returns ASC", "  asc  ", "ASC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateSortOrder(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateSortField(t *testing.T) {
	allowedFields := map[string]bool{
		"id":         true,
		"created_at": true,
		"updated_at": true,
		"name":       true,
	}

	tests := []struct {
		name         string
		input        string
		allowedMap   map[string]bool
		defaultField string
		expected     string
	}{
		{"empty string returns default", "", allowedFields, "created_at", "created_at"},
		{"valid field returns field", "name", allowedFields, "created_at", "name"},
		{"valid field id returns field", "id", allowedFields, "created_at", "id"},
		{"invalid field returns default", "invalid_field", allowedFields, "created_at", "created_at"},
		{"sql injection attempt returns default", "id; DROP TABLE users;--", allowedFields, "created_at", "created_at"},
		{"case sensitive - uppercase invalid", "NAME", allowedFields, "created_at", "created_at"},
		{"whitespace only returns default", "   ", allowedFields, "created_at", "created_at"},
		{"whitespace around valid field returns field", "  name  ", allowedFields, "created_at", "name"},
		{"field with spaces injection returns default", "name users", allowedFields, "created_at", "created_at"},
		{"field with quotes injection returns default", "name'--", allowedFields, "created_at", "created_at"},
		{"empty default with valid field", "name", allowedFields, "", "name"},
		{"empty default with invalid field", "invalid", allowedFields, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateSortField(tt.input, tt.allowedMap, tt.defaultField)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSortFieldsWhitelists(t *testing.T) {
	// Test that all predefined whitelists contain expected common fields
	whitelists := map[string]map[string]bool{
		"UserSortFields":                 UserSortFields,
		"TenantSortFields":               TenantSortFields,
		"CustomerSortFields":             CustomerSortFields,
		"ProductSortFields":              ProductSortFields,
		"CategorySortFields":             CategorySortFields,
		"SupplierSortFields":             SupplierSortFields,
		"WarehouseSortFields":            WarehouseSortFields,
		"InventorySortFields":            InventorySortFields,
		"InventoryTransactionSortFields": InventoryTransactionSortFields,
		"StockBatchSortFields":           StockBatchSortFields,
		"SalesOrderSortFields":           SalesOrderSortFields,
		"SalesReturnSortFields":          SalesReturnSortFields,
		"PurchaseOrderSortFields":        PurchaseOrderSortFields,
		"PurchaseReturnSortFields":       PurchaseReturnSortFields,
		"AccountReceivableSortFields":    AccountReceivableSortFields,
		"AccountPayableSortFields":       AccountPayableSortFields,
		"ExpenseRecordSortFields":        ExpenseRecordSortFields,
		"OtherIncomeRecordSortFields":    OtherIncomeRecordSortFields,
		"StockTakingSortFields":          StockTakingSortFields,
		"RoleSortFields":                 RoleSortFields,
	}

	commonFields := []string{"id", "created_at", "updated_at"}

	for name, whitelist := range whitelists {
		t.Run(name+" contains common fields", func(t *testing.T) {
			for _, field := range commonFields {
				assert.True(t, whitelist[field], "%s should contain '%s'", name, field)
			}
		})

		t.Run(name+" is not empty", func(t *testing.T) {
			assert.Greater(t, len(whitelist), 3, "%s should have more than 3 fields", name)
		})
	}
}

func TestSQLInjectionPrevention(t *testing.T) {
	// Test various SQL injection payloads
	injectionPayloads := []string{
		"id; DROP TABLE users;--",
		"id' OR '1'='1",
		"id\"; DROP TABLE users;--",
		"id UNION SELECT * FROM users",
		"id ORDER BY 1",
		"id, (SELECT password FROM users)",
		"CASE WHEN 1=1 THEN id ELSE name END",
		"id/**/;DROP TABLE users",
		"id\n; DROP TABLE users",
		"id\t; DROP TABLE users",
		"' OR ''='",
		"1; EXEC xp_cmdshell('dir')",
	}

	for _, payload := range injectionPayloads {
		t.Run("field: "+payload[:min(len(payload), 30)], func(t *testing.T) {
			result := ValidateSortField(payload, UserSortFields, "created_at")
			// All injection attempts should return the default
			assert.Equal(t, "created_at", result, "SQL injection payload should be rejected: %s", payload)
		})

		t.Run("order: "+payload[:min(len(payload), 30)], func(t *testing.T) {
			result := ValidateSortOrder(payload)
			// All injection attempts should return DESC
			assert.Equal(t, "DESC", result, "SQL injection payload should be rejected: %s", payload)
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
