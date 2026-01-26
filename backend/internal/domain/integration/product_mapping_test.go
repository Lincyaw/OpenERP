package integration

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// ProductMapping Tests
// ---------------------------------------------------------------------------

func TestNewProductMapping(t *testing.T) {
	tenantID := uuid.New()
	productID := uuid.New()

	t.Run("Valid mapping creation", func(t *testing.T) {
		mapping, err := NewProductMapping(
			tenantID,
			productID,
			PlatformCodeTaobao,
			"TAOBAO_PROD_001",
		)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, mapping.ID)
		assert.Equal(t, tenantID, mapping.TenantID)
		assert.Equal(t, productID, mapping.LocalProductID)
		assert.Equal(t, PlatformCodeTaobao, mapping.PlatformCode)
		assert.Equal(t, "TAOBAO_PROD_001", mapping.PlatformProductID)
		assert.True(t, mapping.IsActive)
		assert.True(t, mapping.SyncEnabled)
		assert.Equal(t, SyncStatusPending, mapping.LastSyncStatus)
		assert.Empty(t, mapping.SKUMappings)
	})

	t.Run("Invalid tenant ID", func(t *testing.T) {
		_, err := NewProductMapping(
			uuid.Nil,
			productID,
			PlatformCodeTaobao,
			"TAOBAO_PROD_001",
		)
		assert.ErrorIs(t, err, ErrMappingInvalidTenantID)
	})

	t.Run("Invalid product ID", func(t *testing.T) {
		_, err := NewProductMapping(
			tenantID,
			uuid.Nil,
			PlatformCodeTaobao,
			"TAOBAO_PROD_001",
		)
		assert.ErrorIs(t, err, ErrMappingInvalidProductID)
	})

	t.Run("Invalid platform code", func(t *testing.T) {
		_, err := NewProductMapping(
			tenantID,
			productID,
			PlatformCode("INVALID"),
			"PROD_001",
		)
		assert.ErrorIs(t, err, ErrMappingInvalidPlatformCode)
	})

	t.Run("Empty platform product ID", func(t *testing.T) {
		_, err := NewProductMapping(
			tenantID,
			productID,
			PlatformCodeTaobao,
			"",
		)
		assert.ErrorIs(t, err, ErrMappingInvalidPlatformID)
	})
}

func TestProductMapping_Validate(t *testing.T) {
	t.Run("Valid mapping", func(t *testing.T) {
		mapping := &ProductMapping{
			ID:                uuid.New(),
			TenantID:          uuid.New(),
			LocalProductID:    uuid.New(),
			PlatformCode:      PlatformCodeJD,
			PlatformProductID: "JD_PROD_001",
		}
		err := mapping.Validate()
		assert.NoError(t, err)
	})

	t.Run("Invalid tenant ID", func(t *testing.T) {
		mapping := &ProductMapping{
			ID:                uuid.New(),
			TenantID:          uuid.Nil,
			LocalProductID:    uuid.New(),
			PlatformCode:      PlatformCodeJD,
			PlatformProductID: "JD_PROD_001",
		}
		err := mapping.Validate()
		assert.ErrorIs(t, err, ErrMappingInvalidTenantID)
	})
}

func TestProductMapping_AddSKUMapping(t *testing.T) {
	mapping, _ := NewProductMapping(
		uuid.New(),
		uuid.New(),
		PlatformCodePDD,
		"PDD_PROD_001",
	)

	skuID := uuid.New()

	t.Run("Add valid SKU mapping", func(t *testing.T) {
		err := mapping.AddSKUMapping(skuID, "PDD_SKU_001")
		assert.NoError(t, err)
		assert.Equal(t, 1, len(mapping.SKUMappings))
		assert.Equal(t, skuID, mapping.SKUMappings[0].LocalSKUID)
		assert.Equal(t, "PDD_SKU_001", mapping.SKUMappings[0].PlatformSkuID)
		assert.True(t, mapping.SKUMappings[0].IsActive)
	})

	t.Run("Add duplicate SKU mapping is idempotent", func(t *testing.T) {
		err := mapping.AddSKUMapping(skuID, "PDD_SKU_001")
		assert.NoError(t, err)
		assert.Equal(t, 1, len(mapping.SKUMappings))
	})

	t.Run("Add different SKU mapping", func(t *testing.T) {
		newSkuID := uuid.New()
		err := mapping.AddSKUMapping(newSkuID, "PDD_SKU_002")
		assert.NoError(t, err)
		assert.Equal(t, 2, len(mapping.SKUMappings))
	})

	t.Run("Invalid local SKU ID", func(t *testing.T) {
		err := mapping.AddSKUMapping(uuid.Nil, "PDD_SKU_003")
		assert.Error(t, err)
	})

	t.Run("Empty platform SKU ID", func(t *testing.T) {
		err := mapping.AddSKUMapping(uuid.New(), "")
		assert.Error(t, err)
	})
}

func TestProductMapping_RemoveSKUMapping(t *testing.T) {
	mapping, _ := NewProductMapping(
		uuid.New(),
		uuid.New(),
		PlatformCodeDouyin,
		"DY_PROD_001",
	)

	sku1 := uuid.New()
	sku2 := uuid.New()
	_ = mapping.AddSKUMapping(sku1, "DY_SKU_001")
	_ = mapping.AddSKUMapping(sku2, "DY_SKU_002")
	assert.Equal(t, 2, len(mapping.SKUMappings))

	t.Run("Remove existing SKU mapping", func(t *testing.T) {
		mapping.RemoveSKUMapping("DY_SKU_001")
		assert.Equal(t, 1, len(mapping.SKUMappings))
		assert.Equal(t, "DY_SKU_002", mapping.SKUMappings[0].PlatformSkuID)
	})

	t.Run("Remove non-existent SKU mapping is no-op", func(t *testing.T) {
		mapping.RemoveSKUMapping("DY_SKU_999")
		assert.Equal(t, 1, len(mapping.SKUMappings))
	})
}

func TestProductMapping_GetLocalSKUID(t *testing.T) {
	mapping, _ := NewProductMapping(
		uuid.New(),
		uuid.New(),
		PlatformCodeWechat,
		"WX_PROD_001",
	)

	skuID := uuid.New()
	_ = mapping.AddSKUMapping(skuID, "WX_SKU_001")

	t.Run("Get existing SKU", func(t *testing.T) {
		result, found := mapping.GetLocalSKUID("WX_SKU_001")
		assert.True(t, found)
		assert.Equal(t, skuID, result)
	})

	t.Run("Get non-existent SKU", func(t *testing.T) {
		result, found := mapping.GetLocalSKUID("WX_SKU_999")
		assert.False(t, found)
		assert.Equal(t, uuid.Nil, result)
	})

	t.Run("Get inactive SKU returns not found", func(t *testing.T) {
		mapping.SKUMappings[0].IsActive = false
		result, found := mapping.GetLocalSKUID("WX_SKU_001")
		assert.False(t, found)
		assert.Equal(t, uuid.Nil, result)
	})
}

func TestProductMapping_GetPlatformSkuID(t *testing.T) {
	mapping, _ := NewProductMapping(
		uuid.New(),
		uuid.New(),
		PlatformCodeKuaishou,
		"KS_PROD_001",
	)

	skuID := uuid.New()
	_ = mapping.AddSKUMapping(skuID, "KS_SKU_001")

	t.Run("Get existing platform SKU", func(t *testing.T) {
		result, found := mapping.GetPlatformSkuID(skuID)
		assert.True(t, found)
		assert.Equal(t, "KS_SKU_001", result)
	})

	t.Run("Get non-existent platform SKU", func(t *testing.T) {
		result, found := mapping.GetPlatformSkuID(uuid.New())
		assert.False(t, found)
		assert.Equal(t, "", result)
	})
}

func TestProductMapping_Activate_Deactivate(t *testing.T) {
	mapping, _ := NewProductMapping(
		uuid.New(),
		uuid.New(),
		PlatformCodeTaobao,
		"TB_PROD_001",
	)

	assert.True(t, mapping.IsActive)

	t.Run("Deactivate", func(t *testing.T) {
		originalTime := mapping.UpdatedAt
		time.Sleep(time.Millisecond) // Ensure time difference
		mapping.Deactivate()
		assert.False(t, mapping.IsActive)
		assert.True(t, mapping.UpdatedAt.After(originalTime))
	})

	t.Run("Activate", func(t *testing.T) {
		originalTime := mapping.UpdatedAt
		time.Sleep(time.Millisecond)
		mapping.Activate()
		assert.True(t, mapping.IsActive)
		assert.True(t, mapping.UpdatedAt.After(originalTime))
	})
}

func TestProductMapping_EnableSync_DisableSync(t *testing.T) {
	mapping, _ := NewProductMapping(
		uuid.New(),
		uuid.New(),
		PlatformCodeJD,
		"JD_PROD_001",
	)

	assert.True(t, mapping.SyncEnabled)

	t.Run("Disable sync", func(t *testing.T) {
		mapping.DisableSync()
		assert.False(t, mapping.SyncEnabled)
	})

	t.Run("Enable sync", func(t *testing.T) {
		mapping.EnableSync()
		assert.True(t, mapping.SyncEnabled)
	})
}

func TestProductMapping_RecordSyncSuccess(t *testing.T) {
	mapping, _ := NewProductMapping(
		uuid.New(),
		uuid.New(),
		PlatformCodePDD,
		"PDD_PROD_001",
	)

	assert.Nil(t, mapping.LastSyncAt)
	assert.Equal(t, SyncStatusPending, mapping.LastSyncStatus)

	mapping.RecordSyncSuccess()

	assert.NotNil(t, mapping.LastSyncAt)
	assert.Equal(t, SyncStatusSuccess, mapping.LastSyncStatus)
	assert.Empty(t, mapping.LastSyncError)
}

func TestProductMapping_RecordSyncFailure(t *testing.T) {
	mapping, _ := NewProductMapping(
		uuid.New(),
		uuid.New(),
		PlatformCodeDouyin,
		"DY_PROD_001",
	)

	errorMsg := "Platform API returned error: rate limited"
	mapping.RecordSyncFailure(errorMsg)

	assert.NotNil(t, mapping.LastSyncAt)
	assert.Equal(t, SyncStatusFailed, mapping.LastSyncStatus)
	assert.Equal(t, errorMsg, mapping.LastSyncError)
}

// ---------------------------------------------------------------------------
// OrderSyncDirection Tests
// ---------------------------------------------------------------------------

func TestOrderSyncDirection_IsValid(t *testing.T) {
	tests := []struct {
		direction OrderSyncDirection
		expected  bool
	}{
		{OrderSyncDirectionInbound, true},
		{OrderSyncDirectionOutbound, true},
		{OrderSyncDirection("INVALID"), false},
		{OrderSyncDirection(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.direction), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.direction.IsValid())
		})
	}
}

// ---------------------------------------------------------------------------
// PlatformOrderSyncRecord Tests
// ---------------------------------------------------------------------------

func TestPlatformOrderSyncRecord_Structure(t *testing.T) {
	localOrderID := uuid.New()
	record := PlatformOrderSyncRecord{
		ID:               uuid.New(),
		TenantID:         uuid.New(),
		PlatformCode:     PlatformCodeTaobao,
		PlatformOrderID:  "TB123456",
		LocalOrderID:     &localOrderID,
		LocalOrderNumber: "SO-2024-0001",
		Direction:        OrderSyncDirectionInbound,
		Status:           SyncStatusSuccess,
		PlatformStatus:   PlatformOrderStatusPaid,
		SyncedAt:         time.Now(),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	assert.NotEqual(t, uuid.Nil, record.ID)
	assert.Equal(t, PlatformCodeTaobao, record.PlatformCode)
	assert.Equal(t, "TB123456", record.PlatformOrderID)
	assert.Equal(t, localOrderID, *record.LocalOrderID)
	assert.Equal(t, OrderSyncDirectionInbound, record.Direction)
	assert.Equal(t, SyncStatusSuccess, record.Status)
}

// ---------------------------------------------------------------------------
// OrderSyncConfig Tests
// ---------------------------------------------------------------------------

func TestOrderSyncConfig_Structure(t *testing.T) {
	config := OrderSyncConfig{
		TenantID:            uuid.New(),
		PlatformCode:        PlatformCodeJD,
		IsEnabled:           true,
		SyncIntervalMinutes: 15,
		AutoCreateCustomer:  true,
		DefaultWarehouseID:  uuid.New(),
		AutoLockStock:       true,
		OrderPrefixFormat:   "JD{YYYYMMDD}",
		StatusMappings: map[PlatformOrderStatus]string{
			PlatformOrderStatusPaid:      "CONFIRM",
			PlatformOrderStatusCancelled: "CANCEL",
		},
	}

	assert.Equal(t, PlatformCodeJD, config.PlatformCode)
	assert.True(t, config.IsEnabled)
	assert.Equal(t, 15, config.SyncIntervalMinutes)
	assert.True(t, config.AutoCreateCustomer)
	assert.True(t, config.AutoLockStock)
	assert.Equal(t, "JD{YYYYMMDD}", config.OrderPrefixFormat)
	assert.Equal(t, "CONFIRM", config.StatusMappings[PlatformOrderStatusPaid])
}
