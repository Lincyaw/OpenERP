package identity

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestDefaultTenantStatsServiceConfig(t *testing.T) {
	config := DefaultTenantStatsServiceConfig()

	assert.Equal(t, 5*time.Minute, config.CacheTTL)
	assert.True(t, config.CacheEnabled)
}

func TestTenantUsageStatsDTO(t *testing.T) {
	tenantID := uuid.New()
	dto := TenantUsageStatsDTO{
		TenantID:       tenantID,
		TenantName:     "Test Tenant",
		TenantCode:     "TEST001",
		Plan:           "pro",
		Status:         "active",
		UserCount:      10,
		ProductCount:   100,
		OrderCount:     50,
		WarehouseCount: 2,
		StorageUsage:   1024000,
		APICallCount:   5000,
		MaxUsers:       20,
		MaxProducts:    500,
		MaxWarehouses:  5,
		LastUpdated:    time.Now().Unix(),
	}

	assert.Equal(t, tenantID, dto.TenantID)
	assert.Equal(t, "Test Tenant", dto.TenantName)
	assert.Equal(t, "TEST001", dto.TenantCode)
	assert.Equal(t, "pro", dto.Plan)
	assert.Equal(t, "active", dto.Status)
	assert.Equal(t, int64(10), dto.UserCount)
	assert.Equal(t, int64(100), dto.ProductCount)
	assert.Equal(t, int64(50), dto.OrderCount)
	assert.Equal(t, int64(2), dto.WarehouseCount)
	assert.Equal(t, int64(1024000), dto.StorageUsage)
	assert.Equal(t, int64(5000), dto.APICallCount)
	assert.Equal(t, 20, dto.MaxUsers)
	assert.Equal(t, 500, dto.MaxProducts)
	assert.Equal(t, 5, dto.MaxWarehouses)
}

func TestPlatformStatsDTO(t *testing.T) {
	dto := PlatformStatsDTO{
		TotalTenants:     100,
		ActiveTenants:    80,
		TrialTenants:     10,
		SuspendedTenants: 5,
		InactiveTenants:  5,
		TenantsByPlan: map[string]int64{
			"free":       30,
			"basic":      40,
			"pro":        20,
			"enterprise": 10,
		},
		TotalUsers:        1000,
		TotalProducts:     5000,
		TotalOrders:       2000,
		TotalStorageUsage: 10240000,
		LastUpdated:       time.Now().Unix(),
	}

	assert.Equal(t, int64(100), dto.TotalTenants)
	assert.Equal(t, int64(80), dto.ActiveTenants)
	assert.Equal(t, int64(10), dto.TrialTenants)
	assert.Equal(t, int64(5), dto.SuspendedTenants)
	assert.Equal(t, int64(5), dto.InactiveTenants)
	assert.Equal(t, int64(30), dto.TenantsByPlan["free"])
	assert.Equal(t, int64(40), dto.TenantsByPlan["basic"])
	assert.Equal(t, int64(20), dto.TenantsByPlan["pro"])
	assert.Equal(t, int64(10), dto.TenantsByPlan["enterprise"])
	assert.Equal(t, int64(1000), dto.TotalUsers)
	assert.Equal(t, int64(5000), dto.TotalProducts)
	assert.Equal(t, int64(2000), dto.TotalOrders)
	assert.Equal(t, int64(10240000), dto.TotalStorageUsage)
}

func TestTenantGrowthPointDTO(t *testing.T) {
	dto := TenantGrowthPointDTO{
		Date:         "2026-02-01",
		TotalTenants: 100,
		NewTenants:   5,
		ChurnedCount: 2,
	}

	assert.Equal(t, "2026-02-01", dto.Date)
	assert.Equal(t, int64(100), dto.TotalTenants)
	assert.Equal(t, int64(5), dto.NewTenants)
	assert.Equal(t, int64(2), dto.ChurnedCount)
}

func TestTenantGrowthTrendDTO(t *testing.T) {
	dto := TenantGrowthTrendDTO{
		Period:    "daily",
		StartDate: "2026-01-01",
		EndDate:   "2026-01-31",
		DataPoints: []TenantGrowthPointDTO{
			{Date: "2026-01-01", TotalTenants: 95, NewTenants: 3, ChurnedCount: 1},
			{Date: "2026-01-02", TotalTenants: 97, NewTenants: 2, ChurnedCount: 0},
		},
	}

	assert.Equal(t, "daily", dto.Period)
	assert.Equal(t, "2026-01-01", dto.StartDate)
	assert.Equal(t, "2026-01-31", dto.EndDate)
	assert.Len(t, dto.DataPoints, 2)
	assert.Equal(t, int64(95), dto.DataPoints[0].TotalTenants)
	assert.Equal(t, int64(97), dto.DataPoints[1].TotalTenants)
}

func TestTenantGrowthInput(t *testing.T) {
	input := TenantGrowthInput{
		Period: "weekly",
		Days:   30,
	}

	assert.Equal(t, "weekly", input.Period)
	assert.Equal(t, 30, input.Days)
}

func TestTenantStatsService_validateGrowthInput(t *testing.T) {
	service := &TenantStatsService{}

	tests := []struct {
		name       string
		input      TenantGrowthInput
		wantErr    bool
		errMessage string
	}{
		{
			name:    "valid daily period",
			input:   TenantGrowthInput{Period: "daily", Days: 30},
			wantErr: false,
		},
		{
			name:    "valid weekly period",
			input:   TenantGrowthInput{Period: "weekly", Days: 90},
			wantErr: false,
		},
		{
			name:    "valid monthly period",
			input:   TenantGrowthInput{Period: "monthly", Days: 365},
			wantErr: false,
		},
		{
			name:       "invalid period",
			input:      TenantGrowthInput{Period: "yearly", Days: 30},
			wantErr:    true,
			errMessage: "Period must be 'daily', 'weekly', or 'monthly'",
		},
		{
			name:       "days less than 1",
			input:      TenantGrowthInput{Period: "daily", Days: 0},
			wantErr:    true,
			errMessage: "Days must be at least 1",
		},
		{
			name:       "days exceeds 365",
			input:      TenantGrowthInput{Period: "daily", Days: 400},
			wantErr:    true,
			errMessage: "Days cannot exceed 365",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateGrowthInput(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMessage)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewTenantStatsService(t *testing.T) {
	// Test with nil redis client - caching should be disabled
	service := NewTenantStatsService(nil, nil, nil, nil, DefaultTenantStatsServiceConfig())

	assert.NotNil(t, service)
	assert.False(t, service.cacheEnabled)
	assert.Equal(t, 5*time.Minute, service.cacheTTL)
}

func TestNewTenantStatsService_CacheDisabled(t *testing.T) {
	config := TenantStatsServiceConfig{
		CacheTTL:     10 * time.Minute,
		CacheEnabled: false,
	}

	service := NewTenantStatsService(nil, nil, nil, nil, config)

	assert.NotNil(t, service)
	assert.False(t, service.cacheEnabled)
	assert.Equal(t, 10*time.Minute, service.cacheTTL)
}
