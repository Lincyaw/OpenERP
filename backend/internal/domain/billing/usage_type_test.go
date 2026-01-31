package billing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsageType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		ut       UsageType
		expected bool
	}{
		{"API_CALLS is valid", UsageTypeAPICalls, true},
		{"STORAGE_BYTES is valid", UsageTypeStorageBytes, true},
		{"ACTIVE_USERS is valid", UsageTypeActiveUsers, true},
		{"ORDERS_CREATED is valid", UsageTypeOrdersCreated, true},
		{"PRODUCTS_SKU is valid", UsageTypeProductsSKU, true},
		{"WAREHOUSES is valid", UsageTypeWarehouses, true},
		{"CUSTOMERS is valid", UsageTypeCustomers, true},
		{"SUPPLIERS is valid", UsageTypeSuppliers, true},
		{"REPORTS_GENERATED is valid", UsageTypeReportsGenerated, true},
		{"DATA_EXPORTS is valid", UsageTypeDataExports, true},
		{"DATA_IMPORT_ROWS is valid", UsageTypeDataImportRows, true},
		{"INTEGRATION_CALLS is valid", UsageTypeIntegrationCalls, true},
		{"NOTIFICATIONS_SENT is valid", UsageTypeNotificationsSent, true},
		{"ATTACHMENT_BYTES is valid", UsageTypeAttachmentBytes, true},
		{"invalid type", UsageType("INVALID"), false},
		{"empty type", UsageType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.ut.IsValid())
		})
	}
}

func TestUsageType_Unit(t *testing.T) {
	tests := []struct {
		name     string
		ut       UsageType
		expected UsageUnit
	}{
		{"API_CALLS returns requests", UsageTypeAPICalls, UsageUnitRequests},
		{"STORAGE_BYTES returns bytes", UsageTypeStorageBytes, UsageUnitBytes},
		{"ACTIVE_USERS returns count", UsageTypeActiveUsers, UsageUnitCount},
		{"ORDERS_CREATED returns requests", UsageTypeOrdersCreated, UsageUnitRequests},
		{"PRODUCTS_SKU returns count", UsageTypeProductsSKU, UsageUnitCount},
		{"WAREHOUSES returns count", UsageTypeWarehouses, UsageUnitCount},
		{"ATTACHMENT_BYTES returns bytes", UsageTypeAttachmentBytes, UsageUnitBytes},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.ut.Unit())
		})
	}
}

func TestUsageType_IsCountable(t *testing.T) {
	countableTypes := []UsageType{
		UsageTypeActiveUsers,
		UsageTypeProductsSKU,
		UsageTypeWarehouses,
		UsageTypeCustomers,
		UsageTypeSuppliers,
	}

	nonCountableTypes := []UsageType{
		UsageTypeAPICalls,
		UsageTypeStorageBytes,
		UsageTypeOrdersCreated,
		UsageTypeReportsGenerated,
	}

	for _, ut := range countableTypes {
		t.Run(string(ut)+" is countable", func(t *testing.T) {
			assert.True(t, ut.IsCountable())
		})
	}

	for _, ut := range nonCountableTypes {
		t.Run(string(ut)+" is not countable", func(t *testing.T) {
			assert.False(t, ut.IsCountable())
		})
	}
}

func TestUsageType_IsAccumulative(t *testing.T) {
	accumulativeTypes := []UsageType{
		UsageTypeAPICalls,
		UsageTypeOrdersCreated,
		UsageTypeReportsGenerated,
		UsageTypeDataExports,
		UsageTypeDataImportRows,
		UsageTypeIntegrationCalls,
		UsageTypeNotificationsSent,
	}

	nonAccumulativeTypes := []UsageType{
		UsageTypeStorageBytes,
		UsageTypeActiveUsers,
		UsageTypeProductsSKU,
		UsageTypeWarehouses,
	}

	for _, ut := range accumulativeTypes {
		t.Run(string(ut)+" is accumulative", func(t *testing.T) {
			assert.True(t, ut.IsAccumulative())
		})
	}

	for _, ut := range nonAccumulativeTypes {
		t.Run(string(ut)+" is not accumulative", func(t *testing.T) {
			assert.False(t, ut.IsAccumulative())
		})
	}
}

func TestUsageType_IsStorage(t *testing.T) {
	storageTypes := []UsageType{
		UsageTypeStorageBytes,
		UsageTypeAttachmentBytes,
	}

	nonStorageTypes := []UsageType{
		UsageTypeAPICalls,
		UsageTypeActiveUsers,
		UsageTypeOrdersCreated,
	}

	for _, ut := range storageTypes {
		t.Run(string(ut)+" is storage", func(t *testing.T) {
			assert.True(t, ut.IsStorage())
		})
	}

	for _, ut := range nonStorageTypes {
		t.Run(string(ut)+" is not storage", func(t *testing.T) {
			assert.False(t, ut.IsStorage())
		})
	}
}

func TestUsageType_DisplayName(t *testing.T) {
	tests := []struct {
		ut       UsageType
		expected string
	}{
		{UsageTypeAPICalls, "API Calls"},
		{UsageTypeStorageBytes, "Storage"},
		{UsageTypeActiveUsers, "Active Users"},
		{UsageTypeOrdersCreated, "Orders Created"},
		{UsageTypeProductsSKU, "Products/SKUs"},
		{UsageType("UNKNOWN"), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(string(tt.ut), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.ut.DisplayName())
		})
	}
}

func TestAllUsageTypes(t *testing.T) {
	types := AllUsageTypes()
	assert.Len(t, types, 14)

	// Verify all returned types are valid
	for _, ut := range types {
		assert.True(t, ut.IsValid(), "Type %s should be valid", ut)
	}
}

func TestParseUsageType(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    UsageType
		expectError bool
	}{
		{"valid API_CALLS", "API_CALLS", UsageTypeAPICalls, false},
		{"valid STORAGE_BYTES", "STORAGE_BYTES", UsageTypeStorageBytes, false},
		{"invalid type", "INVALID", "", true},
		{"empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseUsageType(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestUsageUnit_IsValid(t *testing.T) {
	tests := []struct {
		unit     UsageUnit
		expected bool
	}{
		{UsageUnitRequests, true},
		{UsageUnitBytes, true},
		{UsageUnitCount, true},
		{UsageUnit("invalid"), false},
		{UsageUnit(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.unit), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.unit.IsValid())
		})
	}
}

func TestUsageUnit_FormatValue(t *testing.T) {
	tests := []struct {
		name     string
		unit     UsageUnit
		value    int64
		expected string
	}{
		{"requests format", UsageUnitRequests, 1000, "1000 requests"},
		{"count format", UsageUnitCount, 50, "50"},
		{"bytes - small", UsageUnitBytes, 500, "500 B"},
		{"bytes - KB", UsageUnitBytes, 1024, "1.00 KB"},
		{"bytes - MB", UsageUnitBytes, 1048576, "1.00 MB"},
		{"bytes - GB", UsageUnitBytes, 1073741824, "1.00 GB"},
		{"bytes - TB", UsageUnitBytes, 1099511627776, "1.00 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.unit.FormatValue(tt.value))
		})
	}
}

func TestResetPeriod_IsValid(t *testing.T) {
	tests := []struct {
		period   ResetPeriod
		expected bool
	}{
		{ResetPeriodDaily, true},
		{ResetPeriodWeekly, true},
		{ResetPeriodMonthly, true},
		{ResetPeriodYearly, true},
		{ResetPeriodNever, true},
		{ResetPeriod("INVALID"), false},
		{ResetPeriod(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.period), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.period.IsValid())
		})
	}
}

func TestResetPeriod_DisplayName(t *testing.T) {
	tests := []struct {
		period   ResetPeriod
		expected string
	}{
		{ResetPeriodDaily, "Daily"},
		{ResetPeriodWeekly, "Weekly"},
		{ResetPeriodMonthly, "Monthly"},
		{ResetPeriodYearly, "Yearly"},
		{ResetPeriodNever, "Never"},
		{ResetPeriod("UNKNOWN"), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(string(tt.period), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.period.DisplayName())
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1023, "1023 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1048576, "1.00 MB"},
		{1572864, "1.50 MB"},
		{1073741824, "1.00 GB"},
		{1610612736, "1.50 GB"},
		{1099511627776, "1.00 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatBytes(tt.bytes))
		})
	}
}
