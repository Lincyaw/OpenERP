package billing

import "fmt"

// UsageType represents the type of resource being metered
type UsageType string

const (
	// UsageTypeAPICalls tracks the number of API requests made
	UsageTypeAPICalls UsageType = "API_CALLS"

	// UsageTypeStorageBytes tracks storage consumption in bytes
	UsageTypeStorageBytes UsageType = "STORAGE_BYTES"

	// UsageTypeActiveUsers tracks the number of active users in a billing period
	UsageTypeActiveUsers UsageType = "ACTIVE_USERS"

	// UsageTypeOrdersCreated tracks the number of orders (sales + purchase) created
	UsageTypeOrdersCreated UsageType = "ORDERS_CREATED"

	// UsageTypeProductsSKU tracks the number of unique products/SKUs
	UsageTypeProductsSKU UsageType = "PRODUCTS_SKU"

	// UsageTypeWarehouses tracks the number of warehouses
	UsageTypeWarehouses UsageType = "WAREHOUSES"

	// UsageTypeCustomers tracks the number of customers
	UsageTypeCustomers UsageType = "CUSTOMERS"

	// UsageTypeSuppliers tracks the number of suppliers
	UsageTypeSuppliers UsageType = "SUPPLIERS"

	// UsageTypeReportsGenerated tracks the number of reports generated
	UsageTypeReportsGenerated UsageType = "REPORTS_GENERATED"

	// UsageTypeDataExports tracks the number of data exports
	UsageTypeDataExports UsageType = "DATA_EXPORTS"

	// UsageTypeDataImportRows tracks the number of rows imported
	UsageTypeDataImportRows UsageType = "DATA_IMPORT_ROWS"

	// UsageTypeIntegrationCalls tracks external integration API calls
	UsageTypeIntegrationCalls UsageType = "INTEGRATION_CALLS"

	// UsageTypeNotificationsSent tracks notifications (email/SMS) sent
	UsageTypeNotificationsSent UsageType = "NOTIFICATIONS_SENT"

	// UsageTypeAttachmentBytes tracks attachment storage in bytes
	UsageTypeAttachmentBytes UsageType = "ATTACHMENT_BYTES"
)

// String returns the string representation of UsageType
func (u UsageType) String() string {
	return string(u)
}

// IsValid returns true if the usage type is valid
func (u UsageType) IsValid() bool {
	switch u {
	case UsageTypeAPICalls,
		UsageTypeStorageBytes,
		UsageTypeActiveUsers,
		UsageTypeOrdersCreated,
		UsageTypeProductsSKU,
		UsageTypeWarehouses,
		UsageTypeCustomers,
		UsageTypeSuppliers,
		UsageTypeReportsGenerated,
		UsageTypeDataExports,
		UsageTypeDataImportRows,
		UsageTypeIntegrationCalls,
		UsageTypeNotificationsSent,
		UsageTypeAttachmentBytes:
		return true
	}
	return false
}

// Unit returns the measurement unit for this usage type
func (u UsageType) Unit() UsageUnit {
	switch u {
	case UsageTypeStorageBytes, UsageTypeAttachmentBytes:
		return UsageUnitBytes
	case UsageTypeActiveUsers, UsageTypeProductsSKU, UsageTypeWarehouses,
		UsageTypeCustomers, UsageTypeSuppliers:
		return UsageUnitCount
	default:
		return UsageUnitRequests
	}
}

// IsCountable returns true if this usage type represents a countable resource
// (e.g., users, products) rather than an event-based metric (e.g., API calls)
func (u UsageType) IsCountable() bool {
	switch u {
	case UsageTypeActiveUsers, UsageTypeProductsSKU, UsageTypeWarehouses,
		UsageTypeCustomers, UsageTypeSuppliers:
		return true
	}
	return false
}

// IsAccumulative returns true if this usage type accumulates over time
// (e.g., API calls, orders created) rather than being a point-in-time snapshot
func (u UsageType) IsAccumulative() bool {
	switch u {
	case UsageTypeAPICalls, UsageTypeOrdersCreated, UsageTypeReportsGenerated,
		UsageTypeDataExports, UsageTypeDataImportRows, UsageTypeIntegrationCalls,
		UsageTypeNotificationsSent:
		return true
	}
	return false
}

// IsStorage returns true if this usage type represents storage consumption
func (u UsageType) IsStorage() bool {
	switch u {
	case UsageTypeStorageBytes, UsageTypeAttachmentBytes:
		return true
	}
	return false
}

// DisplayName returns a human-readable name for the usage type
func (u UsageType) DisplayName() string {
	switch u {
	case UsageTypeAPICalls:
		return "API Calls"
	case UsageTypeStorageBytes:
		return "Storage"
	case UsageTypeActiveUsers:
		return "Active Users"
	case UsageTypeOrdersCreated:
		return "Orders Created"
	case UsageTypeProductsSKU:
		return "Products/SKUs"
	case UsageTypeWarehouses:
		return "Warehouses"
	case UsageTypeCustomers:
		return "Customers"
	case UsageTypeSuppliers:
		return "Suppliers"
	case UsageTypeReportsGenerated:
		return "Reports Generated"
	case UsageTypeDataExports:
		return "Data Exports"
	case UsageTypeDataImportRows:
		return "Import Rows"
	case UsageTypeIntegrationCalls:
		return "Integration Calls"
	case UsageTypeNotificationsSent:
		return "Notifications Sent"
	case UsageTypeAttachmentBytes:
		return "Attachment Storage"
	default:
		return string(u)
	}
}

// AllUsageTypes returns all valid usage types
func AllUsageTypes() []UsageType {
	return []UsageType{
		UsageTypeAPICalls,
		UsageTypeStorageBytes,
		UsageTypeActiveUsers,
		UsageTypeOrdersCreated,
		UsageTypeProductsSKU,
		UsageTypeWarehouses,
		UsageTypeCustomers,
		UsageTypeSuppliers,
		UsageTypeReportsGenerated,
		UsageTypeDataExports,
		UsageTypeDataImportRows,
		UsageTypeIntegrationCalls,
		UsageTypeNotificationsSent,
		UsageTypeAttachmentBytes,
	}
}

// ParseUsageType parses a string into a UsageType
func ParseUsageType(s string) (UsageType, error) {
	u := UsageType(s)
	if !u.IsValid() {
		return "", fmt.Errorf("invalid usage type: %s", s)
	}
	return u, nil
}

// UsageUnit represents the unit of measurement for usage
type UsageUnit string

const (
	// UsageUnitRequests represents request/call count
	UsageUnitRequests UsageUnit = "requests"

	// UsageUnitBytes represents storage in bytes
	UsageUnitBytes UsageUnit = "bytes"

	// UsageUnitCount represents a simple count
	UsageUnitCount UsageUnit = "count"
)

// String returns the string representation of UsageUnit
func (u UsageUnit) String() string {
	return string(u)
}

// IsValid returns true if the usage unit is valid
func (u UsageUnit) IsValid() bool {
	switch u {
	case UsageUnitRequests, UsageUnitBytes, UsageUnitCount:
		return true
	}
	return false
}

// FormatValue formats a value with the appropriate unit suffix
func (u UsageUnit) FormatValue(value int64) string {
	switch u {
	case UsageUnitBytes:
		return formatBytes(value)
	case UsageUnitRequests:
		return fmt.Sprintf("%d requests", value)
	case UsageUnitCount:
		return fmt.Sprintf("%d", value)
	default:
		return fmt.Sprintf("%d", value)
	}
}

// formatBytes formats bytes into human-readable format
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// ResetPeriod represents when usage counters reset
type ResetPeriod string

const (
	// ResetPeriodDaily resets usage daily
	ResetPeriodDaily ResetPeriod = "DAILY"

	// ResetPeriodWeekly resets usage weekly
	ResetPeriodWeekly ResetPeriod = "WEEKLY"

	// ResetPeriodMonthly resets usage monthly (most common for billing)
	ResetPeriodMonthly ResetPeriod = "MONTHLY"

	// ResetPeriodYearly resets usage yearly
	ResetPeriodYearly ResetPeriod = "YEARLY"

	// ResetPeriodNever never resets (for lifetime limits)
	ResetPeriodNever ResetPeriod = "NEVER"
)

// String returns the string representation of ResetPeriod
func (r ResetPeriod) String() string {
	return string(r)
}

// IsValid returns true if the reset period is valid
func (r ResetPeriod) IsValid() bool {
	switch r {
	case ResetPeriodDaily, ResetPeriodWeekly, ResetPeriodMonthly,
		ResetPeriodYearly, ResetPeriodNever:
		return true
	}
	return false
}

// DisplayName returns a human-readable name for the reset period
func (r ResetPeriod) DisplayName() string {
	switch r {
	case ResetPeriodDaily:
		return "Daily"
	case ResetPeriodWeekly:
		return "Weekly"
	case ResetPeriodMonthly:
		return "Monthly"
	case ResetPeriodYearly:
		return "Yearly"
	case ResetPeriodNever:
		return "Never"
	default:
		return string(r)
	}
}
