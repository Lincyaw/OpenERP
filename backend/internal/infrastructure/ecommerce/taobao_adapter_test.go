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

func TestTaobaoConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *TaobaoConfig
		wantErr error
	}{
		{
			name: "valid config",
			config: &TaobaoConfig{
				AppKey:     "test_app_key",
				AppSecret:  "test_app_secret",
				SessionKey: "test_session_key",
			},
			wantErr: nil,
		},
		{
			name: "missing app key",
			config: &TaobaoConfig{
				AppSecret:  "test_app_secret",
				SessionKey: "test_session_key",
			},
			wantErr: ErrTaobaoConfigMissingAppKey,
		},
		{
			name: "missing app secret",
			config: &TaobaoConfig{
				AppKey:     "test_app_key",
				SessionKey: "test_session_key",
			},
			wantErr: ErrTaobaoConfigMissingAppSecret,
		},
		{
			name: "missing session key",
			config: &TaobaoConfig{
				AppKey:    "test_app_key",
				AppSecret: "test_app_secret",
			},
			wantErr: ErrTaobaoConfigMissingSessionKey,
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

func TestTaobaoConfig_Sign(t *testing.T) {
	config := &TaobaoConfig{
		AppKey:    "test_key",
		AppSecret: "test_secret",
	}

	params := map[string]string{
		"method":    "taobao.trades.sold.get",
		"app_key":   "test_key",
		"timestamp": "2024-01-01 00:00:00",
	}

	// Sign should be deterministic
	sign1 := config.Sign(params)
	sign2 := config.Sign(params)
	assert.Equal(t, sign1, sign2)
	assert.Len(t, sign1, 32)                // MD5 produces 32 hex characters
	assert.Equal(t, sign1, upperHex(sign1)) // Should be uppercase
}

func TestNewTaobaoConfig(t *testing.T) {
	config := NewTaobaoConfig("app", "secret", "session")
	assert.Equal(t, "app", config.AppKey)
	assert.Equal(t, "secret", config.AppSecret)
	assert.Equal(t, "session", config.SessionKey)
	assert.Equal(t, TaobaoProductionAPIURL, config.APIBaseURL)
	assert.False(t, config.IsSandbox)
}

func TestNewSandboxTaobaoConfig(t *testing.T) {
	config := NewSandboxTaobaoConfig("app", "secret", "session")
	assert.Equal(t, TaobaoSandboxAPIURL, config.APIBaseURL)
	assert.True(t, config.IsSandbox)
}

// ---------------------------------------------------------------------------
// Adapter Tests
// ---------------------------------------------------------------------------

func TestNewTaobaoAdapter(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := NewTaobaoConfig("app", "secret", "session")
		adapter, err := NewTaobaoAdapter(config)
		require.NoError(t, err)
		assert.NotNil(t, adapter)
		assert.Equal(t, integration.PlatformCodeTaobao, adapter.PlatformCode())
	})

	t.Run("invalid config", func(t *testing.T) {
		config := &TaobaoConfig{} // Empty config
		adapter, err := NewTaobaoAdapter(config)
		assert.Error(t, err)
		assert.Nil(t, adapter)
	})
}

func TestTaobaoAdapter_PlatformCode(t *testing.T) {
	adapter := createTestAdapter(t)
	assert.Equal(t, integration.PlatformCodeTaobao, adapter.PlatformCode())
}

func TestTaobaoAdapter_IsEnabled(t *testing.T) {
	adapter := createTestAdapter(t)
	tenantID := uuid.New()

	t.Run("tenant with config", func(t *testing.T) {
		config := NewTaobaoConfig("app", "secret", "session")
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

func TestTaobaoAdapter_SetTenantConfig(t *testing.T) {
	adapter := createTestAdapter(t)
	tenantID := uuid.New()

	t.Run("valid config", func(t *testing.T) {
		config := NewTaobaoConfig("tenant_app", "tenant_secret", "tenant_session")
		err := adapter.SetTenantConfig(tenantID, config)
		assert.NoError(t, err)
	})

	t.Run("invalid config", func(t *testing.T) {
		config := &TaobaoConfig{} // Empty
		err := adapter.SetTenantConfig(tenantID, config)
		assert.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// Order Pulling Tests
// ---------------------------------------------------------------------------

func TestTaobaoAdapter_PullOrders(t *testing.T) {
	tenantID := uuid.New()

	t.Run("successful pull", func(t *testing.T) {
		server := createMockTaobaoServer(t, func(w http.ResponseWriter, r *http.Request) {
			resp := TaobaoTradesGetResponse{
				TradesSoldGetResponse: &TradesSoldGetResponse{
					TotalResults: 2,
					HasNext:      false,
					Trades: &TaobaoTrades{
						Trade: []TaobaoTrade{
							{
								Tid:             123456789,
								Status:          "WAIT_SELLER_SEND_GOODS",
								BuyerNick:       "test_buyer",
								Created:         "2024-01-15 10:30:00",
								PayTime:         "2024-01-15 10:35:00",
								Payment:         "199.00",
								TotalFee:        "199.00",
								PostFee:         "0.00",
								ReceiverName:    "张三",
								ReceiverState:   "浙江省",
								ReceiverCity:    "杭州市",
								ReceiverAddress: "西湖区XX路XX号",
								ReceiverMobile:  "13800138000",
								Orders: &TaobaoOrders{
									Order: []TaobaoOrder{
										{
											Oid:      111,
											NumIid:   9999,
											Title:    "测试商品",
											Price:    "199.00",
											Num:      1,
											TotalFee: "199.00",
											Payment:  "199.00",
										},
									},
								},
							},
							{
								Tid:       987654321,
								Status:    "TRADE_FINISHED",
								BuyerNick: "buyer2",
								Created:   "2024-01-14 09:00:00",
								Payment:   "59.90",
								TotalFee:  "59.90",
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		})
		defer server.Close()

		adapter := createTestAdapterWithServer(t, server.URL, tenantID)

		req := &integration.OrderPullRequest{
			TenantID:     tenantID,
			PlatformCode: integration.PlatformCodeTaobao,
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
		assert.Equal(t, "123456789", order1.PlatformOrderID)
		assert.Equal(t, integration.PlatformOrderStatusPaid, order1.Status)
		assert.Equal(t, "test_buyer", order1.BuyerNickname)
		assert.Equal(t, "张三", order1.ReceiverName)
		assert.Equal(t, "浙江省", order1.ReceiverProvince)
		assert.True(t, order1.TotalAmount.Equal(decimal.NewFromFloat(199.00)))
		assert.Len(t, order1.Items, 1)
		assert.Equal(t, "测试商品", order1.Items[0].ProductName)
	})

	t.Run("validation error", func(t *testing.T) {
		adapter := createTestAdapter(t)

		req := &integration.OrderPullRequest{
			// Missing required fields
			PlatformCode: integration.PlatformCodeTaobao,
		}

		resp, err := adapter.PullOrders(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("API error", func(t *testing.T) {
		server := createMockTaobaoServer(t, func(w http.ResponseWriter, r *http.Request) {
			resp := TaobaoTradesGetResponse{
				TaobaoResponse: TaobaoResponse{
					ErrorResponse: &TaobaoErrorResponse{
						Code:   "isv.invalid-session",
						Msg:    "Invalid session",
						SubMsg: "Session expired",
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		})
		defer server.Close()

		adapter := createTestAdapterWithServer(t, server.URL, tenantID)

		req := &integration.OrderPullRequest{
			TenantID:     tenantID,
			PlatformCode: integration.PlatformCodeTaobao,
			StartTime:    time.Now().AddDate(0, 0, -7),
			EndTime:      time.Now(),
			PageNo:       1,
			PageSize:     50,
		}

		resp, err := adapter.PullOrders(context.Background(), req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "isv.invalid-session")
		assert.Nil(t, resp)
	})
}

func TestTaobaoAdapter_GetOrder(t *testing.T) {
	tenantID := uuid.New()

	t.Run("successful get", func(t *testing.T) {
		server := createMockTaobaoServer(t, func(w http.ResponseWriter, r *http.Request) {
			resp := TaobaoTradeGetResponse{
				TradeFullinfoGetResponse: &TradeFullinfoGetResponse{
					Trade: &TaobaoTrade{
						Tid:          123456789,
						Status:       "WAIT_SELLER_SEND_GOODS",
						BuyerNick:    "test_buyer",
						Payment:      "199.00",
						ReceiverName: "张三",
						Orders: &TaobaoOrders{
							Order: []TaobaoOrder{
								{
									Oid:      111,
									NumIid:   9999,
									Title:    "测试商品",
									Price:    "199.00",
									Num:      1,
									TotalFee: "199.00",
								},
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		})
		defer server.Close()

		adapter := createTestAdapterWithServer(t, server.URL, tenantID)

		order, err := adapter.GetOrder(context.Background(), tenantID, "123456789")
		require.NoError(t, err)
		assert.Equal(t, "123456789", order.PlatformOrderID)
		assert.Equal(t, integration.PlatformOrderStatusPaid, order.Status)
	})

	t.Run("order not found", func(t *testing.T) {
		server := createMockTaobaoServer(t, func(w http.ResponseWriter, r *http.Request) {
			resp := TaobaoTradeGetResponse{
				TradeFullinfoGetResponse: &TradeFullinfoGetResponse{
					Trade: nil, // No trade found
				},
			}
			json.NewEncoder(w).Encode(resp)
		})
		defer server.Close()

		adapter := createTestAdapterWithServer(t, server.URL, tenantID)

		order, err := adapter.GetOrder(context.Background(), tenantID, "999999999")
		assert.ErrorIs(t, err, integration.ErrOrderSyncOrderNotFound)
		assert.Nil(t, order)
	})
}

func TestTaobaoAdapter_UpdateOrderStatus(t *testing.T) {
	tenantID := uuid.New()

	t.Run("send shipment", func(t *testing.T) {
		server := createMockTaobaoServer(t, func(w http.ResponseWriter, r *http.Request) {
			resp := TaobaoLogisticsSendResponse{
				LogisticsOfflineSendResponse: &LogisticsOfflineSendResponse{
					Shipping: &TaobaoShipping{
						IsSuccess: true,
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		})
		defer server.Close()

		adapter := createTestAdapterWithServer(t, server.URL, tenantID)

		req := &integration.OrderStatusUpdateRequest{
			TenantID:        tenantID,
			PlatformCode:    integration.PlatformCodeTaobao,
			PlatformOrderID: "123456789",
			Status:          integration.PlatformOrderStatusShipped,
			ShippingCompany: "顺丰",
			TrackingNumber:  "SF1234567890",
		}

		err := adapter.UpdateOrderStatus(context.Background(), req)
		assert.NoError(t, err)
	})

	t.Run("validation error - missing shipping info", func(t *testing.T) {
		adapter := createTestAdapter(t)

		req := &integration.OrderStatusUpdateRequest{
			TenantID:        tenantID,
			PlatformCode:    integration.PlatformCodeTaobao,
			PlatformOrderID: "123456789",
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

func TestTaobaoAdapter_SyncInventory(t *testing.T) {
	tenantID := uuid.New()

	t.Run("successful item quantity update", func(t *testing.T) {
		server := createMockTaobaoServer(t, func(w http.ResponseWriter, r *http.Request) {
			resp := TaobaoItemQuantityUpdateResponse{
				ItemQuantityUpdateResponse: &ItemQuantityUpdateResponse{
					Item: &TaobaoItem{
						NumIid: 123456,
						Num:    100,
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		})
		defer server.Close()

		adapter := createTestAdapterWithServer(t, server.URL, tenantID)

		items := []integration.InventorySync{
			{
				PlatformProductID: "123456",
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

	t.Run("successful SKU quantity update", func(t *testing.T) {
		server := createMockTaobaoServer(t, func(w http.ResponseWriter, r *http.Request) {
			resp := TaobaoSkuQuantityUpdateResponse{
				ItemSkuUpdateResponse: &ItemSkuUpdateResponse{
					Sku: &TaobaoSku{
						SkuID:    789,
						NumIid:   123456,
						Quantity: 50,
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		})
		defer server.Close()

		adapter := createTestAdapterWithServer(t, server.URL, tenantID)

		items := []integration.InventorySync{
			{
				PlatformProductID: "123456",
				PlatformSkuID:     "789",
				AvailableQuantity: decimal.NewFromInt(50),
			},
		}

		result, err := adapter.SyncInventory(context.Background(), tenantID, items)
		require.NoError(t, err)
		assert.Equal(t, integration.SyncStatusSuccess, result.Status)
	})

	t.Run("partial failure", func(t *testing.T) {
		callCount := 0
		server := createMockTaobaoServer(t, func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount == 1 {
				// First call succeeds
				resp := TaobaoItemQuantityUpdateResponse{
					ItemQuantityUpdateResponse: &ItemQuantityUpdateResponse{
						Item: &TaobaoItem{NumIid: 111, Num: 100},
					},
				}
				json.NewEncoder(w).Encode(resp)
			} else {
				// Second call fails
				resp := TaobaoItemQuantityUpdateResponse{
					TaobaoResponse: TaobaoResponse{
						ErrorResponse: &TaobaoErrorResponse{
							Code: "isv.item-not-exist",
							Msg:  "Item not found",
						},
					},
				}
				json.NewEncoder(w).Encode(resp)
			}
		})
		defer server.Close()

		adapter := createTestAdapterWithServer(t, server.URL, tenantID)

		items := []integration.InventorySync{
			{PlatformProductID: "111", AvailableQuantity: decimal.NewFromInt(100)},
			{PlatformProductID: "222", AvailableQuantity: decimal.NewFromInt(50)},
		}

		result, err := adapter.SyncInventory(context.Background(), tenantID, items)
		require.NoError(t, err)
		assert.Equal(t, integration.SyncStatusPartial, result.Status)
		assert.Equal(t, 2, result.TotalCount)
		assert.Equal(t, 1, result.SuccessCount)
		assert.Equal(t, 1, result.FailedCount)
		assert.Len(t, result.FailedItems, 1)
		assert.Equal(t, "222", result.FailedItems[0].ItemID)
	})
}

// ---------------------------------------------------------------------------
// Product Operations Tests
// ---------------------------------------------------------------------------

func TestTaobaoAdapter_GetProduct(t *testing.T) {
	tenantID := uuid.New()

	t.Run("successful get", func(t *testing.T) {
		server := createMockTaobaoServer(t, func(w http.ResponseWriter, r *http.Request) {
			resp := TaobaoItemGetResponse{
				ItemGetResponse: &ItemGetResponse{
					Item: &TaobaoFullItem{
						NumIid:        123456,
						Title:         "测试商品",
						Desc:          "商品描述",
						Price:         "99.00",
						Num:           100,
						OuterId:       "SKU001",
						ApproveStatus: "onsale",
						ItemImg: &ItemImgs{
							ItemImg: []ItemImg{
								{URL: "https://img.taobao.com/1.jpg"},
								{URL: "https://img.taobao.com/2.jpg"},
							},
						},
						Skus: &TaobaoSkus{
							Sku: []TaobaoFullSku{
								{
									SkuID:          789,
									NumIid:         123456,
									PropertiesName: "颜色:红色",
									Price:          "99.00",
									Quantity:       50,
									OuterId:        "SKU001-RED",
								},
								{
									SkuID:          790,
									NumIid:         123456,
									PropertiesName: "颜色:蓝色",
									Price:          "99.00",
									Quantity:       50,
									OuterId:        "SKU001-BLUE",
								},
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		})
		defer server.Close()

		adapter := createTestAdapterWithServer(t, server.URL, tenantID)

		product, err := adapter.GetProduct(context.Background(), tenantID, "123456")
		require.NoError(t, err)
		assert.Equal(t, "123456", product.PlatformProductID)
		assert.Equal(t, "测试商品", product.ProductName)
		assert.Equal(t, "SKU001", product.ProductCode)
		assert.True(t, product.Price.Equal(decimal.NewFromFloat(99.00)))
		assert.True(t, product.Quantity.Equal(decimal.NewFromInt(100)))
		assert.True(t, product.IsOnSale)
		assert.Len(t, product.ImageURLs, 2)
		assert.Len(t, product.SKUs, 2)
		assert.Equal(t, "789", product.SKUs[0].PlatformSkuID)
		assert.Equal(t, "颜色:红色", product.SKUs[0].SkuName)
	})

	t.Run("product not found", func(t *testing.T) {
		server := createMockTaobaoServer(t, func(w http.ResponseWriter, r *http.Request) {
			resp := TaobaoItemGetResponse{
				ItemGetResponse: &ItemGetResponse{
					Item: nil,
				},
			}
			json.NewEncoder(w).Encode(resp)
		})
		defer server.Close()

		adapter := createTestAdapterWithServer(t, server.URL, tenantID)

		product, err := adapter.GetProduct(context.Background(), tenantID, "999999")
		assert.ErrorIs(t, err, integration.ErrProductSyncMappingNotFound)
		assert.Nil(t, product)
	})
}

func TestTaobaoAdapter_SyncProducts(t *testing.T) {
	tenantID := uuid.New()

	t.Run("successful sync", func(t *testing.T) {
		server := createMockTaobaoServer(t, func(w http.ResponseWriter, r *http.Request) {
			// Mock successful update response
			resp := map[string]any{
				"item_update_response": map[string]any{
					"item": map[string]any{
						"num_iid": 123456,
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		})
		defer server.Close()

		adapter := createTestAdapterWithServer(t, server.URL, tenantID)

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

func TestMapTaobaoOrderStatusToPlatformStatus(t *testing.T) {
	tests := []struct {
		taobaoStatus   string
		expectedStatus integration.PlatformOrderStatus
	}{
		{"WAIT_BUYER_PAY", integration.PlatformOrderStatusPending},
		{"WAIT_SELLER_SEND_GOODS", integration.PlatformOrderStatusPaid},
		{"WAIT_BUYER_CONFIRM_GOODS", integration.PlatformOrderStatusShipped},
		{"TRADE_BUYER_SIGNED", integration.PlatformOrderStatusDelivered},
		{"TRADE_FINISHED", integration.PlatformOrderStatusCompleted},
		{"TRADE_CLOSED", integration.PlatformOrderStatusClosed},
		{"TRADE_CLOSED_BY_TAOBAO", integration.PlatformOrderStatusCancelled},
		{"UNKNOWN_STATUS", integration.PlatformOrderStatusPending}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.taobaoStatus, func(t *testing.T) {
			result := mapTaobaoOrderStatusToPlatformStatus(tt.taobaoStatus)
			assert.Equal(t, tt.expectedStatus, result)
		})
	}
}

func TestMapToTaobaoOrderStatus(t *testing.T) {
	tests := []struct {
		platformStatus integration.PlatformOrderStatus
		expectedStatus string
	}{
		{integration.PlatformOrderStatusPending, "WAIT_BUYER_PAY"},
		{integration.PlatformOrderStatusPaid, "WAIT_SELLER_SEND_GOODS"},
		{integration.PlatformOrderStatusShipped, "WAIT_BUYER_CONFIRM_GOODS"},
		{integration.PlatformOrderStatusDelivered, "TRADE_BUYER_SIGNED"},
		{integration.PlatformOrderStatusCompleted, "TRADE_FINISHED"},
		{integration.PlatformOrderStatusClosed, "TRADE_CLOSED"},
		{integration.PlatformOrderStatusCancelled, "TRADE_CLOSED_BY_TAOBAO"},
	}

	for _, tt := range tests {
		t.Run(string(tt.platformStatus), func(t *testing.T) {
			result := mapToTaobaoOrderStatus(tt.platformStatus)
			assert.Equal(t, tt.expectedStatus, result)
		})
	}
}

func TestMapShippingCompanyToTaobaoCode(t *testing.T) {
	tests := []struct {
		company      string
		expectedCode string
	}{
		{"顺丰", "SF"},
		{"顺丰速运", "SF"},
		{"圆通", "YTO"},
		{"中通", "ZTO"},
		{"申通", "STO"},
		{"韵达", "YUNDA"},
		{"EMS", "EMS"},
		{"ems", "EMS"},
		{"京东", "JD"},
		{"极兔", "JTSD"},
		{"未知快递", "OTHER"},
	}

	for _, tt := range tests {
		t.Run(tt.company, func(t *testing.T) {
			result := mapShippingCompanyToTaobaoCode(tt.company)
			assert.Equal(t, tt.expectedCode, result)
		})
	}
}

// ---------------------------------------------------------------------------
// Type Conversion Tests
// ---------------------------------------------------------------------------

func TestParseDecimal(t *testing.T) {
	tests := []struct {
		input    string
		expected decimal.Decimal
	}{
		{"99.00", decimal.NewFromFloat(99.00)},
		{"0.01", decimal.NewFromFloat(0.01)},
		{"1234567.89", decimal.NewFromFloat(1234567.89)},
		{"", decimal.Zero},
		{"invalid", decimal.Zero},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseDecimal(tt.input)
			assert.True(t, result.Equal(tt.expected), "expected %s but got %s", tt.expected.String(), result.String())
		})
	}
}

func TestConvertTaobaoTradeToPlatformOrder(t *testing.T) {
	adapter := createTestAdapter(t)

	trade := &TaobaoTrade{
		Tid:              123456789,
		Status:           "WAIT_SELLER_SEND_GOODS",
		BuyerNick:        "test_buyer",
		Created:          "2024-01-15 10:30:00",
		PayTime:          "2024-01-15 10:35:00",
		Payment:          "199.00",
		TotalFee:         "199.00",
		PostFee:          "10.00",
		DiscountFee:      "20.00",
		ReceiverName:     "张三",
		ReceiverState:    "浙江省",
		ReceiverCity:     "杭州市",
		ReceiverDistrict: "西湖区",
		ReceiverAddress:  "XX路XX号",
		ReceiverZip:      "310000",
		ReceiverMobile:   "13800138000",
		BuyerMessage:     "请包装好",
		SellerMemo:       "VIP客户",
		Orders: &TaobaoOrders{
			Order: []TaobaoOrder{
				{
					Oid:               111,
					NumIid:            9999,
					SkuID:             "88888",
					Title:             "测试商品",
					SkuPropertiesName: "颜色:红色;尺码:M",
					Price:             "219.00",
					Num:               1,
					TotalFee:          "219.00",
					Payment:           "199.00",
					DiscountFee:       "20.00",
					PicPath:           "https://img.example.com/1.jpg",
				},
			},
		},
	}

	order := adapter.convertTaobaoTradeToPlatformOrder(trade)

	assert.Equal(t, "123456789", order.PlatformOrderID)
	assert.Equal(t, integration.PlatformCodeTaobao, order.PlatformCode)
	assert.Equal(t, integration.PlatformOrderStatusPaid, order.Status)
	assert.Equal(t, "test_buyer", order.BuyerNickname)
	assert.Equal(t, "张三", order.ReceiverName)
	assert.Equal(t, "浙江省", order.ReceiverProvince)
	assert.Equal(t, "杭州市", order.ReceiverCity)
	assert.Equal(t, "西湖区", order.ReceiverDistrict)
	assert.Equal(t, "XX路XX号", order.ReceiverAddress)
	assert.Equal(t, "310000", order.ReceiverPostalCode)
	assert.Equal(t, "13800138000", order.ReceiverPhone)
	assert.True(t, order.TotalAmount.Equal(decimal.NewFromFloat(199.00)))
	assert.True(t, order.ProductAmount.Equal(decimal.NewFromFloat(199.00)))
	assert.True(t, order.FreightAmount.Equal(decimal.NewFromFloat(10.00)))
	assert.True(t, order.DiscountAmount.Equal(decimal.NewFromFloat(20.00)))
	assert.Equal(t, "CNY", order.Currency)
	assert.Equal(t, "请包装好", order.BuyerMessage)
	assert.Equal(t, "VIP客户", order.SellerMemo)
	assert.NotNil(t, order.PaidAt)
	assert.NotEmpty(t, order.RawData)

	// Check order items
	require.Len(t, order.Items, 1)
	item := order.Items[0]
	assert.Equal(t, "111", item.PlatformItemID)
	assert.Equal(t, "9999", item.PlatformProductID)
	assert.Equal(t, "88888", item.PlatformSkuID)
	assert.Equal(t, "测试商品", item.ProductName)
	assert.Equal(t, "颜色:红色;尺码:M", item.SkuName)
	assert.Equal(t, "https://img.example.com/1.jpg", item.ImageURL)
	assert.True(t, item.Quantity.Equal(decimal.NewFromInt(1)))
	assert.True(t, item.UnitPrice.Equal(decimal.NewFromFloat(219.00)))
	assert.True(t, item.TotalPrice.Equal(decimal.NewFromFloat(219.00)))
	assert.True(t, item.DiscountAmount.Equal(decimal.NewFromFloat(20.00)))
}

// ---------------------------------------------------------------------------
// Helper Functions
// ---------------------------------------------------------------------------

func createTestAdapter(t *testing.T) *TaobaoAdapter {
	config := NewTaobaoConfig("test_app_key", "test_app_secret", "test_session_key")
	adapter, err := NewTaobaoAdapter(config)
	require.NoError(t, err)
	return adapter
}

func createTestAdapterWithServer(t *testing.T, serverURL string, tenantID uuid.UUID) *TaobaoAdapter {
	config := &TaobaoConfig{
		AppKey:         "test_app_key",
		AppSecret:      "test_app_secret",
		SessionKey:     "test_session_key",
		APIBaseURL:     serverURL,
		TimeoutSeconds: 30,
	}
	adapter, err := NewTaobaoAdapter(config)
	require.NoError(t, err)

	// Set config for specific tenant
	err = adapter.SetTenantConfig(tenantID, config)
	require.NoError(t, err)

	return adapter
}

func createMockTaobaoServer(_ *testing.T, handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func upperHex(s string) string {
	result := make([]byte, len(s))
	for i, c := range s {
		if c >= 'a' && c <= 'f' {
			result[i] = byte(c - 32)
		} else {
			result[i] = byte(c)
		}
	}
	return string(result)
}
