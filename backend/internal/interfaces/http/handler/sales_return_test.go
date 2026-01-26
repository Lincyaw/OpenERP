package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tradeapp "github.com/erp/backend/internal/application/trade"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/trade"
	"github.com/erp/backend/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSalesReturnRepository implements trade.SalesReturnRepository for testing
type MockSalesReturnRepository struct {
	mock.Mock
}

func (m *MockSalesReturnRepository) FindByID(ctx context.Context, id uuid.UUID) (*trade.SalesReturn, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*trade.SalesReturn), args.Error(1)
}

func (m *MockSalesReturnRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*trade.SalesReturn, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*trade.SalesReturn), args.Error(1)
}

func (m *MockSalesReturnRepository) FindByReturnNumber(ctx context.Context, tenantID uuid.UUID, returnNumber string) (*trade.SalesReturn, error) {
	args := m.Called(ctx, tenantID, returnNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*trade.SalesReturn), args.Error(1)
}

func (m *MockSalesReturnRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]trade.SalesReturn, error) {
	args := m.Called(ctx, tenantID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trade.SalesReturn), args.Error(1)
}

func (m *MockSalesReturnRepository) FindByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, filter shared.Filter) ([]trade.SalesReturn, error) {
	args := m.Called(ctx, tenantID, customerID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trade.SalesReturn), args.Error(1)
}

func (m *MockSalesReturnRepository) FindBySalesOrder(ctx context.Context, tenantID, salesOrderID uuid.UUID) ([]trade.SalesReturn, error) {
	args := m.Called(ctx, tenantID, salesOrderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trade.SalesReturn), args.Error(1)
}

func (m *MockSalesReturnRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status trade.ReturnStatus, filter shared.Filter) ([]trade.SalesReturn, error) {
	args := m.Called(ctx, tenantID, status, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trade.SalesReturn), args.Error(1)
}

func (m *MockSalesReturnRepository) FindPendingApproval(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]trade.SalesReturn, error) {
	args := m.Called(ctx, tenantID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trade.SalesReturn), args.Error(1)
}

func (m *MockSalesReturnRepository) Save(ctx context.Context, sr *trade.SalesReturn) error {
	args := m.Called(ctx, sr)
	return args.Error(0)
}

func (m *MockSalesReturnRepository) SaveWithLock(ctx context.Context, sr *trade.SalesReturn) error {
	args := m.Called(ctx, sr)
	return args.Error(0)
}

func (m *MockSalesReturnRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSalesReturnRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *MockSalesReturnRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSalesReturnRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status trade.ReturnStatus) (int64, error) {
	args := m.Called(ctx, tenantID, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSalesReturnRepository) CountByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID, customerID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSalesReturnRepository) CountBySalesOrder(ctx context.Context, tenantID, salesOrderID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID, salesOrderID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSalesReturnRepository) CountPendingApproval(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSalesReturnRepository) ExistsByReturnNumber(ctx context.Context, tenantID uuid.UUID, returnNumber string) (bool, error) {
	args := m.Called(ctx, tenantID, returnNumber)
	return args.Bool(0), args.Error(1)
}

func (m *MockSalesReturnRepository) GenerateReturnNumber(ctx context.Context, tenantID uuid.UUID) (string, error) {
	args := m.Called(ctx, tenantID)
	return args.String(0), args.Error(1)
}

func (m *MockSalesReturnRepository) GetReturnedQuantityByOrderItem(ctx context.Context, tenantID, salesOrderItemID uuid.UUID) (map[uuid.UUID]decimal.Decimal, error) {
	args := m.Called(ctx, tenantID, salesOrderItemID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[uuid.UUID]decimal.Decimal), args.Error(1)
}

func (m *MockSalesReturnRepository) GetReturnedQuantityByOrderItems(ctx context.Context, tenantID uuid.UUID, salesOrderItemIDs []uuid.UUID) (map[uuid.UUID]decimal.Decimal, error) {
	args := m.Called(ctx, tenantID, salesOrderItemIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[uuid.UUID]decimal.Decimal), args.Error(1)
}

// Ensure mock implements the interface
var _ trade.SalesReturnRepository = (*MockSalesReturnRepository)(nil)

// Test helpers

func setupSalesReturnTestRouter() (*gin.Engine, *MockSalesReturnRepository, *MockSalesOrderRepository, *SalesReturnHandler) {
	gin.SetMode(gin.TestMode)

	mockReturnRepo := new(MockSalesReturnRepository)
	mockOrderRepo := new(MockSalesOrderRepository)
	service := tradeapp.NewSalesReturnService(mockReturnRepo, mockOrderRepo)
	handler := NewSalesReturnHandler(service)

	router := gin.New()

	return router, mockReturnRepo, mockOrderRepo, handler
}

func createTestSalesReturn(tenantID uuid.UUID, returnNumber string) *trade.SalesReturn {
	now := time.Now()
	warehouseID := uuid.New()
	sr := &trade.SalesReturn{
		ReturnNumber:     returnNumber,
		SalesOrderID:     uuid.New(),
		SalesOrderNumber: "SO-2026-00001",
		CustomerID:       uuid.New(),
		CustomerName:     "Test Customer",
		WarehouseID:      &warehouseID,
		TotalRefund:      decimal.NewFromFloat(199.99),
		Status:           trade.ReturnStatusDraft,
		Reason:           "商品质量问题",
		Items:            []trade.SalesReturnItem{},
	}
	sr.ID = uuid.New()
	sr.TenantID = tenantID
	sr.CreatedAt = now
	sr.UpdatedAt = now
	sr.Version = 1
	return sr
}

func addTestJWTContext(c *gin.Context, userID string) {
	c.Set(middleware.JWTUserIDKey, userID)
}

// Tests

func TestSalesReturnHandler_Create(t *testing.T) {
	t.Run("should create sales return successfully", func(t *testing.T) {
		router, mockReturnRepo, mockOrderRepo, handler := setupSalesReturnTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		salesOrderID := uuid.New()
		salesOrderItemID := uuid.New()

		// Create a test sales order with items (must be shipped to create return)
		testOrder := createTestSalesOrder(tenantID, "SO-2026-00001")
		testOrder.ID = salesOrderID
		testOrder.Status = trade.OrderStatusShipped
		testOrder.Items = []trade.SalesOrderItem{
			{
				ID:             salesOrderItemID,
				OrderID:        salesOrderID,
				ProductID:      uuid.New(),
				ProductName:    "Test Product",
				ProductCode:    "SKU-001",
				Quantity:       decimal.NewFromInt(10),
				UnitPrice:      decimal.NewFromFloat(99.99),
				Amount:         decimal.NewFromFloat(999.90),
				Unit:           "pcs",
				BaseUnit:       "pcs",
				ConversionRate: decimal.NewFromInt(1),
			},
		}

		router.POST("/sales-returns", handler.Create)

		mockOrderRepo.On("FindByIDForTenant", mock.Anything, tenantID, salesOrderID).
			Return(testOrder, nil)
		// Mock the new validation method - no previous returns
		mockReturnRepo.On("GetReturnedQuantityByOrderItems", mock.Anything, tenantID, []uuid.UUID{salesOrderItemID}).
			Return(map[uuid.UUID]decimal.Decimal{salesOrderItemID: decimal.Zero}, nil)
		mockReturnRepo.On("GenerateReturnNumber", mock.Anything, tenantID).
			Return("SR-2026-00001", nil)
		mockReturnRepo.On("Save", mock.Anything, mock.AnythingOfType("*trade.SalesReturn")).
			Return(nil)

		reqBody := CreateSalesReturnRequest{
			SalesOrderID: salesOrderID.String(),
			Items: []CreateSalesReturnItemInput{
				{
					SalesOrderItemID:  salesOrderItemID.String(),
					ReturnQuantity:    5,
					Reason:            "商品损坏",
					ConditionOnReturn: "damaged",
				},
			},
			Reason: "商品质量问题",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/sales-returns", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", tenantID.String())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))

		mockReturnRepo.AssertExpectations(t)
		mockOrderRepo.AssertExpectations(t)
	})

	t.Run("should return error for invalid sales order ID", func(t *testing.T) {
		router, _, _, handler := setupSalesReturnTestRouter()

		router.POST("/sales-returns", handler.Create)

		reqBody := map[string]any{
			"sales_order_id": "invalid-uuid",
			"items": []map[string]any{
				{
					"sales_order_item_id": uuid.New().String(),
					"return_quantity":     5,
				},
			},
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/sales-returns", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("should return error for empty items", func(t *testing.T) {
		router, _, _, handler := setupSalesReturnTestRouter()

		router.POST("/sales-returns", handler.Create)

		reqBody := map[string]any{
			"sales_order_id": uuid.New().String(),
			"items":          []map[string]any{},
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/sales-returns", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSalesReturnHandler_GetByID(t *testing.T) {
	t.Run("should get sales return by ID", func(t *testing.T) {
		router, mockRepo, _, handler := setupSalesReturnTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		returnID := uuid.New()
		testReturn := createTestSalesReturn(tenantID, "SR-2026-00001")
		testReturn.ID = returnID

		router.GET("/sales-returns/:id", handler.GetByID)

		mockRepo.On("FindByIDForTenant", mock.Anything, tenantID, returnID).
			Return(testReturn, nil)

		req, _ := http.NewRequest(http.MethodGet, "/sales-returns/"+returnID.String(), nil)
		req.Header.Set("X-Tenant-ID", tenantID.String())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))

		mockRepo.AssertExpectations(t)
	})

	t.Run("should return 404 for non-existent return", func(t *testing.T) {
		router, mockRepo, _, handler := setupSalesReturnTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		returnID := uuid.New()

		router.GET("/sales-returns/:id", handler.GetByID)

		mockRepo.On("FindByIDForTenant", mock.Anything, tenantID, returnID).
			Return(nil, shared.ErrNotFound)

		req, _ := http.NewRequest(http.MethodGet, "/sales-returns/"+returnID.String(), nil)
		req.Header.Set("X-Tenant-ID", tenantID.String())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		mockRepo.AssertExpectations(t)
	})

	t.Run("should return error for invalid return ID", func(t *testing.T) {
		router, _, _, handler := setupSalesReturnTestRouter()

		router.GET("/sales-returns/:id", handler.GetByID)

		req, _ := http.NewRequest(http.MethodGet, "/sales-returns/invalid-uuid", nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSalesReturnHandler_List(t *testing.T) {
	t.Run("should list sales returns", func(t *testing.T) {
		router, mockRepo, _, handler := setupSalesReturnTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		testReturns := []trade.SalesReturn{
			*createTestSalesReturn(tenantID, "SR-2026-00001"),
			*createTestSalesReturn(tenantID, "SR-2026-00002"),
		}

		router.GET("/sales-returns", handler.List)

		mockRepo.On("FindAllForTenant", mock.Anything, tenantID, mock.AnythingOfType("shared.Filter")).
			Return(testReturns, nil)
		mockRepo.On("CountForTenant", mock.Anything, tenantID, mock.AnythingOfType("shared.Filter")).
			Return(int64(2), nil)

		req, _ := http.NewRequest(http.MethodGet, "/sales-returns?page=1&page_size=20", nil)
		req.Header.Set("X-Tenant-ID", tenantID.String())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
		assert.NotNil(t, response["meta"])

		mockRepo.AssertExpectations(t)
	})
}

func TestSalesReturnHandler_Submit(t *testing.T) {
	t.Run("should submit draft return", func(t *testing.T) {
		router, mockRepo, _, handler := setupSalesReturnTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		returnID := uuid.New()
		testReturn := createTestSalesReturn(tenantID, "SR-2026-00001")
		testReturn.ID = returnID
		testReturn.Status = trade.ReturnStatusDraft
		// Add an item to allow submission
		testReturn.Items = []trade.SalesReturnItem{
			{
				ID:               uuid.New(),
				ReturnID:         returnID,
				SalesOrderItemID: uuid.New(),
				ProductID:        uuid.New(),
				ProductName:      "Test Product",
				ReturnQuantity:   decimal.NewFromInt(5),
				UnitPrice:        decimal.NewFromFloat(39.99),
				RefundAmount:     decimal.NewFromFloat(199.95),
			},
		}

		router.POST("/sales-returns/:id/submit", handler.Submit)

		mockRepo.On("FindByIDForTenant", mock.Anything, tenantID, returnID).
			Return(testReturn, nil)
		mockRepo.On("SaveWithLock", mock.Anything, mock.AnythingOfType("*trade.SalesReturn")).
			Return(nil)

		req, _ := http.NewRequest(http.MethodPost, "/sales-returns/"+returnID.String()+"/submit", nil)
		req.Header.Set("X-Tenant-ID", tenantID.String())
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		mockRepo.AssertExpectations(t)
	})

	t.Run("should fail to submit non-draft return", func(t *testing.T) {
		router, mockRepo, _, handler := setupSalesReturnTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		returnID := uuid.New()
		testReturn := createTestSalesReturn(tenantID, "SR-2026-00001")
		testReturn.ID = returnID
		testReturn.Status = trade.ReturnStatusPending // Already submitted

		router.POST("/sales-returns/:id/submit", handler.Submit)

		mockRepo.On("FindByIDForTenant", mock.Anything, tenantID, returnID).
			Return(testReturn, nil)

		req, _ := http.NewRequest(http.MethodPost, "/sales-returns/"+returnID.String()+"/submit", nil)
		req.Header.Set("X-Tenant-ID", tenantID.String())
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return an error (422 or similar)
		assert.NotEqual(t, http.StatusOK, w.Code)

		mockRepo.AssertExpectations(t)
	})
}

func TestSalesReturnHandler_Approve(t *testing.T) {
	t.Run("should approve pending return", func(t *testing.T) {
		router, mockRepo, _, handler := setupSalesReturnTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		userID := uuid.New()
		returnID := uuid.New()
		testReturn := createTestSalesReturn(tenantID, "SR-2026-00001")
		testReturn.ID = returnID
		testReturn.Status = trade.ReturnStatusPending

		router.POST("/sales-returns/:id/approve", func(c *gin.Context) {
			// Set JWT user ID in context
			c.Set(middleware.JWTUserIDKey, userID.String())
			handler.Approve(c)
		})

		mockRepo.On("FindByIDForTenant", mock.Anything, tenantID, returnID).
			Return(testReturn, nil)
		mockRepo.On("SaveWithLock", mock.Anything, mock.AnythingOfType("*trade.SalesReturn")).
			Return(nil)

		reqBody := ApproveReturnRequest{
			Note: "审批通过",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/sales-returns/"+returnID.String()+"/approve", bytes.NewBuffer(body))
		req.Header.Set("X-Tenant-ID", tenantID.String())
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		mockRepo.AssertExpectations(t)
	})

	t.Run("should fail to approve non-pending return", func(t *testing.T) {
		router, mockRepo, _, handler := setupSalesReturnTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		userID := uuid.New()
		returnID := uuid.New()
		testReturn := createTestSalesReturn(tenantID, "SR-2026-00001")
		testReturn.ID = returnID
		testReturn.Status = trade.ReturnStatusDraft // Not pending

		router.POST("/sales-returns/:id/approve", func(c *gin.Context) {
			c.Set(middleware.JWTUserIDKey, userID.String())
			handler.Approve(c)
		})

		mockRepo.On("FindByIDForTenant", mock.Anything, tenantID, returnID).
			Return(testReturn, nil)

		req, _ := http.NewRequest(http.MethodPost, "/sales-returns/"+returnID.String()+"/approve", nil)
		req.Header.Set("X-Tenant-ID", tenantID.String())
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return error for non-pending return
		assert.NotEqual(t, http.StatusOK, w.Code)

		mockRepo.AssertExpectations(t)
	})

	t.Run("should fail to approve without authentication", func(t *testing.T) {
		router, _, _, handler := setupSalesReturnTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		returnID := uuid.New()

		router.POST("/sales-returns/:id/approve", handler.Approve)

		req, _ := http.NewRequest(http.MethodPost, "/sales-returns/"+returnID.String()+"/approve", nil)
		req.Header.Set("X-Tenant-ID", tenantID.String())
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestSalesReturnHandler_Reject(t *testing.T) {
	t.Run("should reject pending return", func(t *testing.T) {
		router, mockRepo, _, handler := setupSalesReturnTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		userID := uuid.New()
		returnID := uuid.New()
		testReturn := createTestSalesReturn(tenantID, "SR-2026-00001")
		testReturn.ID = returnID
		testReturn.Status = trade.ReturnStatusPending

		router.POST("/sales-returns/:id/reject", func(c *gin.Context) {
			c.Set(middleware.JWTUserIDKey, userID.String())
			handler.Reject(c)
		})

		mockRepo.On("FindByIDForTenant", mock.Anything, tenantID, returnID).
			Return(testReturn, nil)
		mockRepo.On("SaveWithLock", mock.Anything, mock.AnythingOfType("*trade.SalesReturn")).
			Return(nil)

		reqBody := RejectReturnRequest{
			Reason: "退货原因不充分",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/sales-returns/"+returnID.String()+"/reject", bytes.NewBuffer(body))
		req.Header.Set("X-Tenant-ID", tenantID.String())
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		mockRepo.AssertExpectations(t)
	})

	t.Run("should fail reject without reason", func(t *testing.T) {
		router, _, _, handler := setupSalesReturnTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		userID := uuid.New()
		returnID := uuid.New()

		router.POST("/sales-returns/:id/reject", func(c *gin.Context) {
			c.Set(middleware.JWTUserIDKey, userID.String())
			handler.Reject(c)
		})

		reqBody := map[string]any{
			// Missing reason
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/sales-returns/"+returnID.String()+"/reject", bytes.NewBuffer(body))
		req.Header.Set("X-Tenant-ID", tenantID.String())
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSalesReturnHandler_Complete(t *testing.T) {
	t.Run("should complete approved return", func(t *testing.T) {
		router, mockRepo, _, handler := setupSalesReturnTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		returnID := uuid.New()
		testReturn := createTestSalesReturn(tenantID, "SR-2026-00001")
		testReturn.ID = returnID
		testReturn.Status = trade.ReturnStatusApproved

		router.POST("/sales-returns/:id/complete", handler.Complete)

		mockRepo.On("FindByIDForTenant", mock.Anything, tenantID, returnID).
			Return(testReturn, nil)
		mockRepo.On("SaveWithLock", mock.Anything, mock.AnythingOfType("*trade.SalesReturn")).
			Return(nil)

		req, _ := http.NewRequest(http.MethodPost, "/sales-returns/"+returnID.String()+"/complete", nil)
		req.Header.Set("X-Tenant-ID", tenantID.String())
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		mockRepo.AssertExpectations(t)
	})

	t.Run("should fail to complete non-approved return", func(t *testing.T) {
		router, mockRepo, _, handler := setupSalesReturnTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		returnID := uuid.New()
		testReturn := createTestSalesReturn(tenantID, "SR-2026-00001")
		testReturn.ID = returnID
		testReturn.Status = trade.ReturnStatusPending // Not approved

		router.POST("/sales-returns/:id/complete", handler.Complete)

		mockRepo.On("FindByIDForTenant", mock.Anything, tenantID, returnID).
			Return(testReturn, nil)

		req, _ := http.NewRequest(http.MethodPost, "/sales-returns/"+returnID.String()+"/complete", nil)
		req.Header.Set("X-Tenant-ID", tenantID.String())
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return error for non-approved return
		assert.NotEqual(t, http.StatusOK, w.Code)

		mockRepo.AssertExpectations(t)
	})
}

func TestSalesReturnHandler_Cancel(t *testing.T) {
	t.Run("should cancel draft return", func(t *testing.T) {
		router, mockRepo, _, handler := setupSalesReturnTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		returnID := uuid.New()
		testReturn := createTestSalesReturn(tenantID, "SR-2026-00001")
		testReturn.ID = returnID
		testReturn.Status = trade.ReturnStatusDraft

		router.POST("/sales-returns/:id/cancel", handler.Cancel)

		mockRepo.On("FindByIDForTenant", mock.Anything, tenantID, returnID).
			Return(testReturn, nil)
		mockRepo.On("SaveWithLock", mock.Anything, mock.AnythingOfType("*trade.SalesReturn")).
			Return(nil)

		reqBody := CancelReturnRequest{
			Reason: "客户取消退货申请",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/sales-returns/"+returnID.String()+"/cancel", bytes.NewBuffer(body))
		req.Header.Set("X-Tenant-ID", tenantID.String())
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		mockRepo.AssertExpectations(t)
	})

	t.Run("should fail cancel without reason", func(t *testing.T) {
		router, _, _, handler := setupSalesReturnTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		returnID := uuid.New()

		router.POST("/sales-returns/:id/cancel", handler.Cancel)

		reqBody := map[string]any{
			// Missing reason
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/sales-returns/"+returnID.String()+"/cancel", bytes.NewBuffer(body))
		req.Header.Set("X-Tenant-ID", tenantID.String())
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSalesReturnHandler_Delete(t *testing.T) {
	t.Run("should delete draft return", func(t *testing.T) {
		router, mockRepo, _, handler := setupSalesReturnTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		returnID := uuid.New()
		testReturn := createTestSalesReturn(tenantID, "SR-2026-00001")
		testReturn.ID = returnID
		testReturn.Status = trade.ReturnStatusDraft

		router.DELETE("/sales-returns/:id", handler.Delete)

		mockRepo.On("FindByIDForTenant", mock.Anything, tenantID, returnID).
			Return(testReturn, nil)
		mockRepo.On("DeleteForTenant", mock.Anything, tenantID, returnID).
			Return(nil)

		req, _ := http.NewRequest(http.MethodDelete, "/sales-returns/"+returnID.String(), nil)
		req.Header.Set("X-Tenant-ID", tenantID.String())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		mockRepo.AssertExpectations(t)
	})

	t.Run("should fail to delete non-draft return", func(t *testing.T) {
		router, mockRepo, _, handler := setupSalesReturnTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		returnID := uuid.New()
		testReturn := createTestSalesReturn(tenantID, "SR-2026-00001")
		testReturn.ID = returnID
		testReturn.Status = trade.ReturnStatusPending // Not draft

		router.DELETE("/sales-returns/:id", handler.Delete)

		mockRepo.On("FindByIDForTenant", mock.Anything, tenantID, returnID).
			Return(testReturn, nil)

		req, _ := http.NewRequest(http.MethodDelete, "/sales-returns/"+returnID.String(), nil)
		req.Header.Set("X-Tenant-ID", tenantID.String())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return error for non-draft return
		assert.NotEqual(t, http.StatusNoContent, w.Code)

		mockRepo.AssertExpectations(t)
	})
}

func TestSalesReturnHandler_GetStatusSummary(t *testing.T) {
	t.Run("should get status summary", func(t *testing.T) {
		router, mockRepo, _, handler := setupSalesReturnTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

		router.GET("/sales-returns/stats/summary", handler.GetStatusSummary)

		mockRepo.On("CountByStatus", mock.Anything, tenantID, trade.ReturnStatusDraft).Return(int64(3), nil)
		mockRepo.On("CountByStatus", mock.Anything, tenantID, trade.ReturnStatusPending).Return(int64(5), nil)
		mockRepo.On("CountByStatus", mock.Anything, tenantID, trade.ReturnStatusApproved).Return(int64(2), nil)
		mockRepo.On("CountByStatus", mock.Anything, tenantID, trade.ReturnStatusRejected).Return(int64(1), nil)
		mockRepo.On("CountByStatus", mock.Anything, tenantID, trade.ReturnStatusCompleted).Return(int64(50), nil)
		mockRepo.On("CountByStatus", mock.Anything, tenantID, trade.ReturnStatusCancelled).Return(int64(4), nil)

		req, _ := http.NewRequest(http.MethodGet, "/sales-returns/stats/summary", nil)
		req.Header.Set("X-Tenant-ID", tenantID.String())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))

		data := response["data"].(map[string]any)
		assert.Equal(t, float64(3), data["draft"])
		assert.Equal(t, float64(5), data["pending"])
		assert.Equal(t, float64(2), data["approved"])
		assert.Equal(t, float64(1), data["rejected"])
		assert.Equal(t, float64(50), data["completed"])
		assert.Equal(t, float64(4), data["cancelled"])
		assert.Equal(t, float64(65), data["total"])

		mockRepo.AssertExpectations(t)
	})
}

func TestSalesReturnHandler_AddItem(t *testing.T) {
	t.Run("should add item to draft return", func(t *testing.T) {
		router, mockReturnRepo, mockOrderRepo, handler := setupSalesReturnTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		returnID := uuid.New()
		salesOrderID := uuid.New()
		salesOrderItemID := uuid.New()
		testReturn := createTestSalesReturn(tenantID, "SR-2026-00001")
		testReturn.ID = returnID
		testReturn.SalesOrderID = salesOrderID
		testReturn.Status = trade.ReturnStatusDraft

		// Create a test sales order with items
		testOrder := createTestSalesOrder(tenantID, "SO-2026-00001")
		testOrder.ID = salesOrderID
		testOrder.Status = trade.OrderStatusShipped
		testOrder.Items = []trade.SalesOrderItem{
			{
				ID:             salesOrderItemID,
				OrderID:        salesOrderID,
				ProductID:      uuid.New(),
				ProductName:    "Test Product",
				ProductCode:    "SKU-001",
				Quantity:       decimal.NewFromInt(10),
				UnitPrice:      decimal.NewFromFloat(49.99),
				Amount:         decimal.NewFromFloat(499.90),
				Unit:           "pcs",
				BaseUnit:       "pcs",
				ConversionRate: decimal.NewFromInt(1),
			},
		}

		router.POST("/sales-returns/:id/items", handler.AddItem)

		mockReturnRepo.On("FindByIDForTenant", mock.Anything, tenantID, returnID).
			Return(testReturn, nil)
		mockOrderRepo.On("FindByIDForTenant", mock.Anything, tenantID, salesOrderID).
			Return(testOrder, nil)
		// Mock the new validation method - no previous returns
		mockReturnRepo.On("GetReturnedQuantityByOrderItem", mock.Anything, tenantID, salesOrderItemID).
			Return(map[uuid.UUID]decimal.Decimal{salesOrderItemID: decimal.Zero}, nil)
		mockReturnRepo.On("SaveWithLock", mock.Anything, mock.AnythingOfType("*trade.SalesReturn")).
			Return(nil)

		reqBody := AddReturnItemRequest{
			SalesOrderItemID:  salesOrderItemID.String(),
			ReturnQuantity:    3,
			Reason:            "商品有瑕疵",
			ConditionOnReturn: "defective",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/sales-returns/"+returnID.String()+"/items", bytes.NewBuffer(body))
		req.Header.Set("X-Tenant-ID", tenantID.String())
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		mockReturnRepo.AssertExpectations(t)
		mockOrderRepo.AssertExpectations(t)
	})
}

// Suppress unused helper function warning
var _ = addTestJWTContext
