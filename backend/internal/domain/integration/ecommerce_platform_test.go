package integration

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// PlatformCode Tests
// ---------------------------------------------------------------------------

func TestPlatformCode_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		code     PlatformCode
		expected bool
	}{
		{"Taobao valid", PlatformCodeTaobao, true},
		{"JD valid", PlatformCodeJD, true},
		{"PDD valid", PlatformCodePDD, true},
		{"Douyin valid", PlatformCodeDouyin, true},
		{"Wechat valid", PlatformCodeWechat, true},
		{"Kuaishou valid", PlatformCodeKuaishou, true},
		{"Invalid code", PlatformCode("INVALID"), false},
		{"Empty code", PlatformCode(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.code.IsValid())
		})
	}
}

func TestPlatformCode_DisplayName(t *testing.T) {
	tests := []struct {
		code     PlatformCode
		expected string
	}{
		{PlatformCodeTaobao, "淘宝/天猫"},
		{PlatformCodeJD, "京东"},
		{PlatformCodePDD, "拼多多"},
		{PlatformCodeDouyin, "抖音"},
		{PlatformCodeWechat, "微信小程序"},
		{PlatformCodeKuaishou, "快手"},
		{PlatformCode("UNKNOWN"), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.code.DisplayName())
		})
	}
}

// ---------------------------------------------------------------------------
// PlatformOrderStatus Tests
// ---------------------------------------------------------------------------

func TestPlatformOrderStatus_IsValid(t *testing.T) {
	validStatuses := []PlatformOrderStatus{
		PlatformOrderStatusPending,
		PlatformOrderStatusPaid,
		PlatformOrderStatusShipped,
		PlatformOrderStatusDelivered,
		PlatformOrderStatusCompleted,
		PlatformOrderStatusCancelled,
		PlatformOrderStatusRefunding,
		PlatformOrderStatusRefunded,
		PlatformOrderStatusClosed,
	}

	for _, status := range validStatuses {
		t.Run(string(status), func(t *testing.T) {
			assert.True(t, status.IsValid())
		})
	}

	t.Run("Invalid status", func(t *testing.T) {
		assert.False(t, PlatformOrderStatus("INVALID").IsValid())
	})
}

func TestPlatformOrderStatus_IsFinal(t *testing.T) {
	tests := []struct {
		status   PlatformOrderStatus
		expected bool
	}{
		{PlatformOrderStatusPending, false},
		{PlatformOrderStatusPaid, false},
		{PlatformOrderStatusShipped, false},
		{PlatformOrderStatusDelivered, false},
		{PlatformOrderStatusCompleted, true},
		{PlatformOrderStatusCancelled, true},
		{PlatformOrderStatusRefunding, false},
		{PlatformOrderStatusRefunded, true},
		{PlatformOrderStatusClosed, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.IsFinal())
		})
	}
}

func TestPlatformOrderStatus_RequiresShipment(t *testing.T) {
	tests := []struct {
		status   PlatformOrderStatus
		expected bool
	}{
		{PlatformOrderStatusPending, false},
		{PlatformOrderStatusPaid, true},
		{PlatformOrderStatusShipped, false},
		{PlatformOrderStatusCompleted, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.RequiresShipment())
		})
	}
}

// ---------------------------------------------------------------------------
// SyncStatus Tests
// ---------------------------------------------------------------------------

func TestSyncStatus_IsValid(t *testing.T) {
	validStatuses := []SyncStatus{
		SyncStatusPending,
		SyncStatusInProgress,
		SyncStatusSuccess,
		SyncStatusPartial,
		SyncStatusFailed,
	}

	for _, status := range validStatuses {
		t.Run(string(status), func(t *testing.T) {
			assert.True(t, status.IsValid())
		})
	}

	t.Run("Invalid status", func(t *testing.T) {
		assert.False(t, SyncStatus("INVALID").IsValid())
	})
}

// ---------------------------------------------------------------------------
// OrderPullRequest Tests
// ---------------------------------------------------------------------------

func TestOrderPullRequest_Validate(t *testing.T) {
	tenantID := uuid.New()
	startTime := time.Now().Add(-24 * time.Hour)
	endTime := time.Now()

	t.Run("Valid request", func(t *testing.T) {
		req := &OrderPullRequest{
			TenantID:     tenantID,
			PlatformCode: PlatformCodeTaobao,
			StartTime:    startTime,
			EndTime:      endTime,
			PageNo:       1,
			PageSize:     50,
		}
		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Missing tenant ID", func(t *testing.T) {
		req := &OrderPullRequest{
			TenantID:     uuid.Nil,
			PlatformCode: PlatformCodeTaobao,
			StartTime:    startTime,
			EndTime:      endTime,
		}
		err := req.Validate()
		assert.ErrorIs(t, err, ErrMappingInvalidTenantID)
	})

	t.Run("Invalid platform code", func(t *testing.T) {
		req := &OrderPullRequest{
			TenantID:     tenantID,
			PlatformCode: PlatformCode("INVALID"),
			StartTime:    startTime,
			EndTime:      endTime,
		}
		err := req.Validate()
		assert.ErrorIs(t, err, ErrMappingInvalidPlatformCode)
	})

	t.Run("Missing time range", func(t *testing.T) {
		req := &OrderPullRequest{
			TenantID:     tenantID,
			PlatformCode: PlatformCodeTaobao,
		}
		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Invalid time range", func(t *testing.T) {
		req := &OrderPullRequest{
			TenantID:     tenantID,
			PlatformCode: PlatformCodeTaobao,
			StartTime:    endTime,
			EndTime:      startTime, // start after end
		}
		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Default page values", func(t *testing.T) {
		req := &OrderPullRequest{
			TenantID:     tenantID,
			PlatformCode: PlatformCodeTaobao,
			StartTime:    startTime,
			EndTime:      endTime,
			PageNo:       0, // should default to 1
			PageSize:     0, // should default to 50
		}
		err := req.Validate()
		assert.NoError(t, err)
		assert.Equal(t, 1, req.PageNo)
		assert.Equal(t, 50, req.PageSize)
	})

	t.Run("Page size capped at 100", func(t *testing.T) {
		req := &OrderPullRequest{
			TenantID:     tenantID,
			PlatformCode: PlatformCodeTaobao,
			StartTime:    startTime,
			EndTime:      endTime,
			PageNo:       1,
			PageSize:     200, // should be capped to 50
		}
		err := req.Validate()
		assert.NoError(t, err)
		assert.Equal(t, 50, req.PageSize)
	})
}

// ---------------------------------------------------------------------------
// OrderStatusUpdateRequest Tests
// ---------------------------------------------------------------------------

func TestOrderStatusUpdateRequest_Validate(t *testing.T) {
	tenantID := uuid.New()

	t.Run("Valid request", func(t *testing.T) {
		req := &OrderStatusUpdateRequest{
			TenantID:        tenantID,
			PlatformCode:    PlatformCodeJD,
			PlatformOrderID: "JD123456",
			Status:          PlatformOrderStatusShipped,
			ShippingCompany: "SF Express",
			TrackingNumber:  "SF123456789",
		}
		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("Missing tenant ID", func(t *testing.T) {
		req := &OrderStatusUpdateRequest{
			TenantID:        uuid.Nil,
			PlatformCode:    PlatformCodeJD,
			PlatformOrderID: "JD123456",
			Status:          PlatformOrderStatusShipped,
		}
		err := req.Validate()
		assert.ErrorIs(t, err, ErrMappingInvalidTenantID)
	})

	t.Run("Missing shipping info for shipped status", func(t *testing.T) {
		req := &OrderStatusUpdateRequest{
			TenantID:        tenantID,
			PlatformCode:    PlatformCodeJD,
			PlatformOrderID: "JD123456",
			Status:          PlatformOrderStatusShipped,
			// Missing ShippingCompany and TrackingNumber
		}
		err := req.Validate()
		assert.Error(t, err)
	})

	t.Run("Completed status without shipping info is OK", func(t *testing.T) {
		req := &OrderStatusUpdateRequest{
			TenantID:        tenantID,
			PlatformCode:    PlatformCodeJD,
			PlatformOrderID: "JD123456",
			Status:          PlatformOrderStatusCompleted,
		}
		err := req.Validate()
		assert.NoError(t, err)
	})
}

// ---------------------------------------------------------------------------
// PlatformOrder Tests
// ---------------------------------------------------------------------------

func TestPlatformOrder_Structure(t *testing.T) {
	order := PlatformOrder{
		PlatformOrderID: "TB123456",
		PlatformCode:    PlatformCodeTaobao,
		Status:          PlatformOrderStatusPaid,
		BuyerNickname:   "buyer123",
		ReceiverName:    "John Doe",
		ReceiverPhone:   "13812345678",
		ReceiverAddress: "123 Main St",
		TotalAmount:     decimal.NewFromFloat(199.99),
		ProductAmount:   decimal.NewFromFloat(219.99),
		FreightAmount:   decimal.NewFromFloat(10.00),
		DiscountAmount:  decimal.NewFromFloat(30.00),
		Items: []PlatformOrderItem{
			{
				PlatformItemID:    "ITEM001",
				PlatformProductID: "PROD001",
				ProductName:       "Test Product",
				Quantity:          decimal.NewFromInt(2),
				UnitPrice:         decimal.NewFromFloat(109.995),
				TotalPrice:        decimal.NewFromFloat(219.99),
			},
		},
		CreatedAt: time.Now(),
	}

	assert.Equal(t, "TB123456", order.PlatformOrderID)
	assert.Equal(t, PlatformCodeTaobao, order.PlatformCode)
	assert.Equal(t, PlatformOrderStatusPaid, order.Status)
	assert.Equal(t, 1, len(order.Items))
	assert.True(t, order.TotalAmount.Equal(decimal.NewFromFloat(199.99)))
}
