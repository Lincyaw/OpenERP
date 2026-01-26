package ecommerce

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erp/backend/internal/domain/integration"
)

// ---------------------------------------------------------------------------
// Config Tests
// ---------------------------------------------------------------------------

func TestDouyinConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *DouyinConfig
		wantErr error
	}{
		{
			name: "valid config",
			config: &DouyinConfig{
				AppKey:      "test_app_key",
				AppSecret:   "test_app_secret",
				AccessToken: "test_access_token",
				ShopID:      "test_shop_id",
			},
			wantErr: nil,
		},
		{
			name: "missing app key",
			config: &DouyinConfig{
				AppSecret:   "test_app_secret",
				AccessToken: "test_access_token",
				ShopID:      "test_shop_id",
			},
			wantErr: ErrDouyinConfigMissingAppKey,
		},
		{
			name: "missing app secret",
			config: &DouyinConfig{
				AppKey:      "test_app_key",
				AccessToken: "test_access_token",
				ShopID:      "test_shop_id",
			},
			wantErr: ErrDouyinConfigMissingAppSecret,
		},
		{
			name: "missing access token",
			config: &DouyinConfig{
				AppKey:    "test_app_key",
				AppSecret: "test_app_secret",
				ShopID:    "test_shop_id",
			},
			wantErr: ErrDouyinConfigMissingAccessToken,
		},
		{
			name: "missing shop ID",
			config: &DouyinConfig{
				AppKey:      "test_app_key",
				AppSecret:   "test_app_secret",
				AccessToken: "test_access_token",
			},
			wantErr: ErrDouyinConfigMissingShopID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
				// Check defaults are set
				assert.NotEmpty(t, tt.config.APIBaseURL)
				assert.True(t, tt.config.TimeoutSeconds > 0)
			}
		})
	}
}

func TestDouyinConfig_Sign(t *testing.T) {
	config := &DouyinConfig{
		AppKey:    "test_key",
		AppSecret: "test_secret",
	}

	method := "/order/searchList"
	paramJSON := `{"page":0,"size":50}`
	timestamp := "1704067200"
	version := "2"

	// Sign should be deterministic
	sign1 := config.Sign(method, paramJSON, timestamp, version)
	sign2 := config.Sign(method, paramJSON, timestamp, version)
	assert.Equal(t, sign1, sign2)
	assert.Len(t, sign1, 64) // SHA256 produces 64 hex characters
}

func TestDouyinConfig_SignV2(t *testing.T) {
	config := &DouyinConfig{
		AppKey:    "test_key",
		AppSecret: "test_secret",
	}

	params := map[string]string{
		"method":    "/order/searchList",
		"app_key":   "test_key",
		"timestamp": "1704067200",
	}

	// Sign should be deterministic
	sign1 := config.SignV2(params)
	sign2 := config.SignV2(params)
	assert.Equal(t, sign1, sign2)
	assert.Len(t, sign1, 64) // SHA256 produces 64 hex characters
}

func TestNewDouyinConfig(t *testing.T) {
	config := NewDouyinConfig("app", "secret", "token", "shop123")
	assert.Equal(t, "app", config.AppKey)
	assert.Equal(t, "secret", config.AppSecret)
	assert.Equal(t, "token", config.AccessToken)
	assert.Equal(t, "shop123", config.ShopID)
	assert.Equal(t, DouyinProductionAPIURL, config.APIBaseURL)
	assert.False(t, config.IsSandbox)
}

func TestNewSandboxDouyinConfig(t *testing.T) {
	config := NewSandboxDouyinConfig("app", "secret", "token", "shop123")
	assert.Equal(t, DouyinSandboxAPIURL, config.APIBaseURL)
	assert.True(t, config.IsSandbox)
}

// ---------------------------------------------------------------------------
// Adapter Tests
// ---------------------------------------------------------------------------

func TestNewDouyinAdapter(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := NewDouyinConfig("app", "secret", "token", "shop123")
		adapter, err := NewDouyinAdapter(config)
		require.NoError(t, err)
		assert.NotNil(t, adapter)
		assert.Equal(t, integration.PlatformCodeDouyin, adapter.PlatformCode())
	})

	t.Run("invalid config", func(t *testing.T) {
		config := &DouyinConfig{} // Empty config
		adapter, err := NewDouyinAdapter(config)
		assert.Error(t, err)
		assert.Nil(t, adapter)
	})
}

func TestDouyinAdapter_PlatformCode(t *testing.T) {
	adapter := createTestDouyinAdapter(t)
	assert.Equal(t, integration.PlatformCodeDouyin, adapter.PlatformCode())
}

func TestDouyinAdapter_IsEnabled(t *testing.T) {
	adapter := createTestDouyinAdapter(t)
	tenantID := uuid.New()

	t.Run("tenant with config", func(t *testing.T) {
		config := NewDouyinConfig("app", "secret", "token", "shop123")
		err := adapter.SetTenantConfig(tenantID, config)
		require.NoError(t, err)

		enabled, err := adapter.IsEnabled(context.Background(), tenantID)
		assert.NoError(t, err)
		assert.True(t, enabled)
	})

	t.Run("tenant without config uses default", func(t *testing.T) {
		unknownTenant := uuid.New()
		enabled, err := adapter.IsEnabled(context.Background(), unknownTenant)
		assert.NoError(t, err)
		assert.True(t, enabled) // Falls back to default config
	})
}

func TestDouyinAdapter_SetTenantConfig(t *testing.T) {
	adapter := createTestDouyinAdapter(t)
	tenantID := uuid.New()

	t.Run("valid config", func(t *testing.T) {
		config := NewDouyinConfig("tenant_app", "tenant_secret", "tenant_token", "tenant_shop")
		err := adapter.SetTenantConfig(tenantID, config)
		assert.NoError(t, err)
	})

	t.Run("invalid config", func(t *testing.T) {
		config := &DouyinConfig{} // Empty
		err := adapter.SetTenantConfig(tenantID, config)
		assert.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// Order Pulling Tests
// ---------------------------------------------------------------------------

func TestDouyinAdapter_PullOrders(t *testing.T) {
	tenantID := uuid.New()

	t.Run("successful pull", func(t *testing.T) {
		server := createMockDouyinServer(t, func(w http.ResponseWriter, r *http.Request) {
			resp := DouyinOrderListResponse{
				DouyinResponse: DouyinResponse{
					ErrNo:   0,
					Message: "success",
				},
				Data: &DouyinOrderListData{
					Total: 2,
					List: []DouyinOrder{
						{
							OrderID:     "4987654321234567890",
							ShopID:      12345,
							OrderStatus: DouyinOrderStatusPendingShipment,
							CreateTime:  1705312200,
							PayTime:     1705312500,
							PayAmount:   19900, // 199.00 yuan in cents
							OrderAmount: 19900,
							PostAmount:  0,
							PostReceiver: &DouyinPostReceiver{
								Name:     "张三",
								Phone:    "13800138000",
								Province: "浙江省",
								City:     "杭州市",
								Town:     "西湖区",
								Street:   "XX路",
								Detail:   "XX号",
							},
							SkuOrderList: []DouyinSkuOrder{
								{
									SkuOrderID:   "111",
									ProductID:    9999,
									ProductName:  "测试商品",
									SkuID:        88888,
									ItemNum:      1,
									OriginAmount: 19900,
									PayAmount:    19900,
									ProductPic:   "https://img.douyin.com/1.jpg",
								},
							},
						},
						{
							OrderID:     "4987654321234567891",
							ShopID:      12345,
							OrderStatus: DouyinOrderStatusCompleted,
							CreateTime:  1705225800,
							PayAmount:   5990, // 59.90 yuan
							OrderAmount: 5990,
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		})
		defer server.Close()

		adapter := createTestDouyinAdapterWithServer(t, server.URL, tenantID)

		req := &integration.OrderPullRequest{
			TenantID:     tenantID,
			PlatformCode: integration.PlatformCodeDouyin,
			StartTime:    time.Now().AddDate(0, 0, -7),
			EndTime:      time.Now(),
			PageNo:       1,
			PageSize:     50,
		}

		resp, err := adapter.PullOrders(context.Background(), req)
		require.NoError(t, err)
		assert.Equal(t, int64(2), resp.TotalCount)
		assert.False(t, resp.HasMore)
		assert.Len(t, resp.Orders, 2)

		// Check first order
		order1 := resp.Orders[0]
		assert.Equal(t, "4987654321234567890", order1.PlatformOrderID)
		assert.Equal(t, integration.PlatformOrderStatusPaid, order1.Status)
		assert.Equal(t, "张三", order1.ReceiverName)
		assert.Equal(t, "浙江省", order1.ReceiverProvince)
		assert.True(t, order1.TotalAmount.Equal(decimal.NewFromFloat(199.00)))
		assert.Len(t, order1.Items, 1)
		assert.Equal(t, "测试商品", order1.Items[0].ProductName)
	})

	t.Run("validation error", func(t *testing.T) {
		adapter := createTestDouyinAdapter(t)

		req := &integration.OrderPullRequest{
			// Missing required fields
			PlatformCode: integration.PlatformCodeDouyin,
		}

		resp, err := adapter.PullOrders(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("API error", func(t *testing.T) {
		server := createMockDouyinServer(t, func(w http.ResponseWriter, r *http.Request) {
			resp := DouyinOrderListResponse{
				DouyinResponse: DouyinResponse{
					ErrNo:   10000,
					Message: "Invalid access token",
				},
			}
			json.NewEncoder(w).Encode(resp)
		})
		defer server.Close()

		adapter := createTestDouyinAdapterWithServer(t, server.URL, tenantID)

		req := &integration.OrderPullRequest{
			TenantID:     tenantID,
			PlatformCode: integration.PlatformCodeDouyin,
			StartTime:    time.Now().AddDate(0, 0, -7),
			EndTime:      time.Now(),
			PageNo:       1,
			PageSize:     50,
		}

		resp, err := adapter.PullOrders(context.Background(), req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid access token")
		assert.Nil(t, resp)
	})
}

func TestDouyinAdapter_GetOrder(t *testing.T) {
	tenantID := uuid.New()

	t.Run("successful get", func(t *testing.T) {
		server := createMockDouyinServer(t, func(w http.ResponseWriter, r *http.Request) {
			resp := DouyinOrderDetailResponse{
				DouyinResponse: DouyinResponse{
					ErrNo:   0,
					Message: "success",
				},
				Data: &DouyinOrderDetailData{
					ShopOrderDetail: &DouyinOrder{
						OrderID:     "4987654321234567890",
						OrderStatus: DouyinOrderStatusPendingShipment,
						PayAmount:   19900,
						PostReceiver: &DouyinPostReceiver{
							Name:     "张三",
							Province: "浙江省",
						},
						SkuOrderList: []DouyinSkuOrder{
							{
								SkuOrderID:   "111",
								ProductID:    9999,
								ProductName:  "测试商品",
								ItemNum:      1,
								OriginAmount: 19900,
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		})
		defer server.Close()

		adapter := createTestDouyinAdapterWithServer(t, server.URL, tenantID)

		order, err := adapter.GetOrder(context.Background(), tenantID, "4987654321234567890")
		require.NoError(t, err)
		assert.Equal(t, "4987654321234567890", order.PlatformOrderID)
		assert.Equal(t, integration.PlatformOrderStatusPaid, order.Status)
	})

	t.Run("order not found", func(t *testing.T) {
		server := createMockDouyinServer(t, func(w http.ResponseWriter, r *http.Request) {
			resp := DouyinOrderDetailResponse{
				DouyinResponse: DouyinResponse{
					ErrNo:   0,
					Message: "success",
				},
				Data: &DouyinOrderDetailData{
					ShopOrderDetail: nil, // No order found
				},
			}
			json.NewEncoder(w).Encode(resp)
		})
		defer server.Close()

		adapter := createTestDouyinAdapterWithServer(t, server.URL, tenantID)

		order, err := adapter.GetOrder(context.Background(), tenantID, "999999999")
		assert.ErrorIs(t, err, integration.ErrOrderSyncOrderNotFound)
		assert.Nil(t, order)
	})
}

func TestDouyinAdapter_UpdateOrderStatus(t *testing.T) {
	tenantID := uuid.New()

	t.Run("send shipment", func(t *testing.T) {
		server := createMockDouyinServer(t, func(w http.ResponseWriter, r *http.Request) {
			resp := DouyinShipResponse{
				DouyinResponse: DouyinResponse{
					ErrNo:   0,
					Message: "success",
				},
				Data: &DouyinShipData{
					PackID: "pack123",
				},
			}
			json.NewEncoder(w).Encode(resp)
		})
		defer server.Close()

		adapter := createTestDouyinAdapterWithServer(t, server.URL, tenantID)

		req := &integration.OrderStatusUpdateRequest{
			TenantID:        tenantID,
			PlatformCode:    integration.PlatformCodeDouyin,
			PlatformOrderID: "4987654321234567890",
			Status:          integration.PlatformOrderStatusShipped,
			ShippingCompany: "顺丰",
			TrackingNumber:  "SF1234567890",
		}

		err := adapter.UpdateOrderStatus(context.Background(), req)
		assert.NoError(t, err)
	})

	t.Run("validation error - missing shipping info", func(t *testing.T) {
		adapter := createTestDouyinAdapter(t)

		req := &integration.OrderStatusUpdateRequest{
			TenantID:        tenantID,
			PlatformCode:    integration.PlatformCodeDouyin,
			PlatformOrderID: "4987654321234567890",
			Status:          integration.PlatformOrderStatusShipped,
			// Missing ShippingCompany and TrackingNumber
		}

		err := adapter.UpdateOrderStatus(context.Background(), req)
		assert.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// Inventory Sync Tests
// ---------------------------------------------------------------------------

func TestDouyinAdapter_SyncInventory(t *testing.T) {
	tenantID := uuid.New()

	t.Run("successful stock update", func(t *testing.T) {
		server := createMockDouyinServer(t, func(w http.ResponseWriter, r *http.Request) {
			resp := DouyinStockUpdateResponse{
				DouyinResponse: DouyinResponse{
					ErrNo:   0,
					Message: "success",
				},
				Data: &DouyinStockUpdateData{
					Success: true,
				},
			}
			json.NewEncoder(w).Encode(resp)
		})
		defer server.Close()

		adapter := createTestDouyinAdapterWithServer(t, server.URL, tenantID)

		items := []integration.InventorySync{
			{
				PlatformProductID: "123456",
				PlatformSkuID:     "789",
				AvailableQuantity: decimal.NewFromInt(100),
			},
		}

		result, err := adapter.SyncInventory(context.Background(), tenantID, items)
		require.NoError(t, err)
		assert.Equal(t, integration.SyncStatusSuccess, result.Status)
		assert.Equal(t, 1, result.TotalCount)
		assert.Equal(t, 1, result.SuccessCount)
		assert.Equal(t, 0, result.FailedCount)
	})

	t.Run("partial failure", func(t *testing.T) {
		callCount := 0
		server := createMockDouyinServer(t, func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount == 1 {
				// First call succeeds
				resp := DouyinStockUpdateResponse{
					DouyinResponse: DouyinResponse{
						ErrNo:   0,
						Message: "success",
					},
					Data: &DouyinStockUpdateData{Success: true},
				}
				json.NewEncoder(w).Encode(resp)
			} else {
				// Second call fails
				resp := DouyinStockUpdateResponse{
					DouyinResponse: DouyinResponse{
						ErrNo:   20001,
						Message: "SKU not found",
					},
				}
				json.NewEncoder(w).Encode(resp)
			}
		})
		defer server.Close()

		adapter := createTestDouyinAdapterWithServer(t, server.URL, tenantID)

		items := []integration.InventorySync{
			{PlatformProductID: "111", PlatformSkuID: "222", AvailableQuantity: decimal.NewFromInt(100)},
			{PlatformProductID: "333", PlatformSkuID: "444", AvailableQuantity: decimal.NewFromInt(50)},
		}

		result, err := adapter.SyncInventory(context.Background(), tenantID, items)
		require.NoError(t, err)
		assert.Equal(t, integration.SyncStatusPartial, result.Status)
		assert.Equal(t, 2, result.TotalCount)
		assert.Equal(t, 1, result.SuccessCount)
		assert.Equal(t, 1, result.FailedCount)
		assert.Len(t, result.FailedItems, 1)
		assert.Equal(t, "333", result.FailedItems[0].ItemID)
	})
}

// ---------------------------------------------------------------------------
// Product Operations Tests
// ---------------------------------------------------------------------------

func TestDouyinAdapter_GetProduct(t *testing.T) {
	tenantID := uuid.New()

	t.Run("successful get", func(t *testing.T) {
		server := createMockDouyinServer(t, func(w http.ResponseWriter, r *http.Request) {
			resp := DouyinProductDetailResponse{
				DouyinResponse: DouyinResponse{
					ErrNo:   0,
					Message: "success",
				},
				Data: &DouyinProductDetailData{
					Product: &DouyinProduct{
						ProductID:     123456,
						OutProductID:  "SKU001",
						Name:          "测试商品",
						Description:   "商品描述",
						MarketPrice:   9900, // 99.00 yuan in cents
						DiscountPrice: 9900,
						Status:        0, // Online
						Img:           "https://img.douyin.com/1.jpg",
						SkuList: []DouyinSku{
							{
								SkuID:    789,
								OutSkuID: "SKU001-RED",
								Price:    9900,
								StockNum: 50,
								SpecDetail: []DouyinSpecDetail{
									{SpecName: "颜色", ValueName: "红色"},
								},
							},
							{
								SkuID:    790,
								OutSkuID: "SKU001-BLUE",
								Price:    9900,
								StockNum: 50,
								SpecDetail: []DouyinSpecDetail{
									{SpecName: "颜色", ValueName: "蓝色"},
								},
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		})
		defer server.Close()

		adapter := createTestDouyinAdapterWithServer(t, server.URL, tenantID)

		product, err := adapter.GetProduct(context.Background(), tenantID, "123456")
		require.NoError(t, err)
		assert.Equal(t, "123456", product.PlatformProductID)
		assert.Equal(t, "测试商品", product.ProductName)
		assert.Equal(t, "SKU001", product.ProductCode)
		assert.True(t, product.Price.Equal(decimal.NewFromFloat(99.00)))
		assert.True(t, product.Quantity.Equal(decimal.NewFromInt(100))) // 50 + 50
		assert.True(t, product.IsOnSale)
		assert.Len(t, product.ImageURLs, 1)
		assert.Len(t, product.SKUs, 2)
		assert.Equal(t, "789", product.SKUs[0].PlatformSkuID)
		assert.Equal(t, "颜色:红色", product.SKUs[0].SkuName)
	})

	t.Run("product not found", func(t *testing.T) {
		server := createMockDouyinServer(t, func(w http.ResponseWriter, r *http.Request) {
			resp := DouyinProductDetailResponse{
				DouyinResponse: DouyinResponse{
					ErrNo:   0,
					Message: "success",
				},
				Data: &DouyinProductDetailData{
					Product: nil,
				},
			}
			json.NewEncoder(w).Encode(resp)
		})
		defer server.Close()

		adapter := createTestDouyinAdapterWithServer(t, server.URL, tenantID)

		product, err := adapter.GetProduct(context.Background(), tenantID, "999999")
		assert.ErrorIs(t, err, integration.ErrProductSyncMappingNotFound)
		assert.Nil(t, product)
	})

	t.Run("invalid product ID", func(t *testing.T) {
		adapter := createTestDouyinAdapter(t)

		product, err := adapter.GetProduct(context.Background(), tenantID, "invalid_id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid product ID")
		assert.Nil(t, product)
	})
}

func TestDouyinAdapter_SyncProducts(t *testing.T) {
	tenantID := uuid.New()

	t.Run("successful sync", func(t *testing.T) {
		server := createMockDouyinServer(t, func(w http.ResponseWriter, r *http.Request) {
			resp := DouyinResponse{
				ErrNo:   0,
				Message: "success",
			}
			json.NewEncoder(w).Encode(resp)
		})
		defer server.Close()

		adapter := createTestDouyinAdapterWithServer(t, server.URL, tenantID)

		products := []integration.ProductSync{
			{
				PlatformProductID: "123456",
				ProductName:       "更新后的商品",
				Price:             decimal.NewFromFloat(199.00),
				Quantity:          decimal.NewFromInt(50),
			},
		}

		result, err := adapter.SyncProducts(context.Background(), tenantID, products)
		require.NoError(t, err)
		assert.Equal(t, integration.SyncStatusSuccess, result.Status)
		assert.Equal(t, 1, result.SuccessCount)
	})
}

// ---------------------------------------------------------------------------
// Status Mapping Tests
// ---------------------------------------------------------------------------

func TestMapDouyinOrderStatusToPlatformStatus(t *testing.T) {
	tests := []struct {
		douyinStatus   int
		expectedStatus integration.PlatformOrderStatus
	}{
		{DouyinOrderStatusPendingPayment, integration.PlatformOrderStatusPending},
		{DouyinOrderStatusPendingShipment, integration.PlatformOrderStatusPaid},
		{DouyinOrderStatusShipped, integration.PlatformOrderStatusShipped},
		{DouyinOrderStatusCompleted, integration.PlatformOrderStatusCompleted},
		{DouyinOrderStatusCancelled, integration.PlatformOrderStatusCancelled},
		{DouyinOrderStatusRefunding, integration.PlatformOrderStatusRefunding},
		{DouyinOrderStatusRefunded, integration.PlatformOrderStatusRefunded},
		{999, integration.PlatformOrderStatusPending}, // Unknown status
	}

	for _, tt := range tests {
		t.Run(ParseDouyinOrderStatus(tt.douyinStatus), func(t *testing.T) {
			result := mapDouyinOrderStatusToPlatformStatus(tt.douyinStatus)
			assert.Equal(t, tt.expectedStatus, result)
		})
	}
}

func TestMapToDouyinOrderStatus(t *testing.T) {
	tests := []struct {
		platformStatus integration.PlatformOrderStatus
		expectedStatus int
	}{
		{integration.PlatformOrderStatusPending, DouyinOrderStatusPendingPayment},
		{integration.PlatformOrderStatusPaid, DouyinOrderStatusPendingShipment},
		{integration.PlatformOrderStatusShipped, DouyinOrderStatusShipped},
		{integration.PlatformOrderStatusCompleted, DouyinOrderStatusCompleted},
		{integration.PlatformOrderStatusCancelled, DouyinOrderStatusCancelled},
		{integration.PlatformOrderStatusRefunding, DouyinOrderStatusRefunding},
		{integration.PlatformOrderStatusRefunded, DouyinOrderStatusRefunded},
	}

	for _, tt := range tests {
		t.Run(string(tt.platformStatus), func(t *testing.T) {
			result := mapToDouyinOrderStatus(tt.platformStatus)
			assert.Equal(t, tt.expectedStatus, result)
		})
	}
}

func TestMapShippingCompanyToDouyinCode(t *testing.T) {
	tests := []struct {
		company      string
		expectedCode string
	}{
		{"顺丰", "shunfeng"},
		{"顺丰速运", "shunfeng"},
		{"圆通", "yuantong"},
		{"中通", "zhongtong"},
		{"申通", "shentong"},
		{"韵达", "yunda"},
		{"EMS", "ems"},
		{"ems", "ems"},
		{"京东", "jd"},
		{"极兔", "jtexpress"},
		{"未知快递", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.company, func(t *testing.T) {
			result := mapShippingCompanyToDouyinCode(tt.company)
			assert.Equal(t, tt.expectedCode, result)
		})
	}
}

// ---------------------------------------------------------------------------
// Type Conversion Tests
// ---------------------------------------------------------------------------

func TestParseDouyinOrderStatus(t *testing.T) {
	tests := []struct {
		status   int
		expected string
	}{
		{DouyinOrderStatusPendingPayment, "待付款"},
		{DouyinOrderStatusPendingShipment, "待发货"},
		{DouyinOrderStatusShipped, "已发货"},
		{DouyinOrderStatusCompleted, "已完成"},
		{DouyinOrderStatusCancelled, "已取消"},
		{DouyinOrderStatusRefunding, "退款中"},
		{DouyinOrderStatusRefunded, "已退款"},
		{999, "未知状态"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := ParseDouyinOrderStatus(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertDouyinOrderToPlatformOrder(t *testing.T) {
	adapter := createTestDouyinAdapter(t)

	order := &DouyinOrder{
		OrderID:          "4987654321234567890",
		ShopID:           12345,
		OrderStatus:      DouyinOrderStatusPendingShipment,
		CreateTime:       1705312200,
		PayTime:          1705312500,
		PayAmount:        19900,
		OrderAmount:      21900,
		PostAmount:       1000,
		CouponAmount:     2000,
		ShopCouponAmount: 1000,
		PlatformDiscount: 1000,
		BuyerWords:       "请包装好",
		SellerWords:      "VIP客户",
		PostReceiver: &DouyinPostReceiver{
			Name:     "张三",
			Phone:    "13800138000",
			Province: "浙江省",
			City:     "杭州市",
			Town:     "西湖区",
			Street:   "XX路",
			Detail:   "XX号",
			PostCode: "310000",
		},
		SkuOrderList: []DouyinSkuOrder{
			{
				SkuOrderID:   "111",
				ProductID:    9999,
				SkuID:        88888,
				ProductName:  "测试商品",
				SkuSpec:      `{"颜色":"红色","尺码":"M"}`,
				ItemNum:      1,
				OriginAmount: 21900,
				PayAmount:    19900,
				CouponAmount: 2000,
				ProductPic:   "https://img.douyin.com/1.jpg",
			},
		},
	}

	platformOrder := adapter.convertDouyinOrderToPlatformOrder(order)

	assert.Equal(t, "4987654321234567890", platformOrder.PlatformOrderID)
	assert.Equal(t, integration.PlatformCodeDouyin, platformOrder.PlatformCode)
	assert.Equal(t, integration.PlatformOrderStatusPaid, platformOrder.Status)
	assert.Equal(t, "张三", platformOrder.ReceiverName)
	assert.Equal(t, "浙江省", platformOrder.ReceiverProvince)
	assert.Equal(t, "杭州市", platformOrder.ReceiverCity)
	assert.Equal(t, "西湖区", platformOrder.ReceiverDistrict)
	assert.Contains(t, platformOrder.ReceiverAddress, "XX路")
	assert.Equal(t, "310000", platformOrder.ReceiverPostalCode)
	assert.Equal(t, "13800138000", platformOrder.ReceiverPhone)
	assert.True(t, platformOrder.TotalAmount.Equal(decimal.NewFromFloat(199.00)))
	assert.True(t, platformOrder.ProductAmount.Equal(decimal.NewFromFloat(219.00)))
	assert.True(t, platformOrder.FreightAmount.Equal(decimal.NewFromFloat(10.00)))
	assert.True(t, platformOrder.DiscountAmount.Equal(decimal.NewFromFloat(20.00)))
	assert.Equal(t, "CNY", platformOrder.Currency)
	assert.Equal(t, "请包装好", platformOrder.BuyerMessage)
	assert.Equal(t, "VIP客户", platformOrder.SellerMemo)
	assert.NotNil(t, platformOrder.PaidAt)
	assert.NotEmpty(t, platformOrder.RawData)

	// Check order items
	require.Len(t, platformOrder.Items, 1)
	item := platformOrder.Items[0]
	assert.Equal(t, "111", item.PlatformItemID)
	assert.Equal(t, "9999", item.PlatformProductID)
	assert.Equal(t, "88888", item.PlatformSkuID)
	assert.Equal(t, "测试商品", item.ProductName)
	assert.Equal(t, "https://img.douyin.com/1.jpg", item.ImageURL)
	assert.True(t, item.Quantity.Equal(decimal.NewFromInt(1)))
	assert.True(t, item.TotalPrice.Equal(decimal.NewFromFloat(219.00)))
	assert.True(t, item.DiscountAmount.Equal(decimal.NewFromFloat(20.00)))
}

// ---------------------------------------------------------------------------
// Helper Functions
// ---------------------------------------------------------------------------

func createTestDouyinAdapter(t *testing.T) *DouyinAdapter {
	config := NewDouyinConfig("test_app_key", "test_app_secret", "test_access_token", "test_shop_id")
	adapter, err := NewDouyinAdapter(config)
	require.NoError(t, err)
	return adapter
}

func createTestDouyinAdapterWithServer(t *testing.T, serverURL string, tenantID uuid.UUID) *DouyinAdapter {
	config := &DouyinConfig{
		AppKey:         "test_app_key",
		AppSecret:      "test_app_secret",
		AccessToken:    "test_access_token",
		ShopID:         "test_shop_id",
		APIBaseURL:     serverURL,
		TimeoutSeconds: 30,
	}
	adapter, err := NewDouyinAdapter(config)
	require.NoError(t, err)

	// Set config for specific tenant
	err = adapter.SetTenantConfig(tenantID, config)
	require.NoError(t, err)

	return adapter
}

func createMockDouyinServer(_ *testing.T, handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}
