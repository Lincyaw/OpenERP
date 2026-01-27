package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tradeapp "github.com/erp/backend/internal/application/trade"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/trade"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSalesOrderRepository implements trade.SalesOrderRepository for testing
type MockSalesOrderRepository struct {
	mock.Mock
}

func (m *MockSalesOrderRepository) FindByID(ctx context.Context, id uuid.UUID) (*trade.SalesOrder, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*trade.SalesOrder), args.Error(1)
}

func (m *MockSalesOrderRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*trade.SalesOrder, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*trade.SalesOrder), args.Error(1)
}

func (m *MockSalesOrderRepository) FindByOrderNumber(ctx context.Context, tenantID uuid.UUID, orderNumber string) (*trade.SalesOrder, error) {
	args := m.Called(ctx, tenantID, orderNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*trade.SalesOrder), args.Error(1)
}

func (m *MockSalesOrderRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]trade.SalesOrder, error) {
	args := m.Called(ctx, tenantID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trade.SalesOrder), args.Error(1)
}

func (m *MockSalesOrderRepository) FindByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, filter shared.Filter) ([]trade.SalesOrder, error) {
	args := m.Called(ctx, tenantID, customerID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trade.SalesOrder), args.Error(1)
}

func (m *MockSalesOrderRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status trade.OrderStatus, filter shared.Filter) ([]trade.SalesOrder, error) {
	args := m.Called(ctx, tenantID, status, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trade.SalesOrder), args.Error(1)
}

func (m *MockSalesOrderRepository) FindByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID, filter shared.Filter) ([]trade.SalesOrder, error) {
	args := m.Called(ctx, tenantID, warehouseID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]trade.SalesOrder), args.Error(1)
}

func (m *MockSalesOrderRepository) Save(ctx context.Context, order *trade.SalesOrder) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockSalesOrderRepository) SaveWithLock(ctx context.Context, order *trade.SalesOrder) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockSalesOrderRepository) SaveWithLockAndEvents(ctx context.Context, order *trade.SalesOrder, events []shared.DomainEvent) error {
	args := m.Called(ctx, order, events)
	return args.Error(0)
}

func (m *MockSalesOrderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSalesOrderRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *MockSalesOrderRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSalesOrderRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status trade.OrderStatus) (int64, error) {
	args := m.Called(ctx, tenantID, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSalesOrderRepository) CountByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID, customerID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSalesOrderRepository) CountIncompleteByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (int64, error) {
	args := m.Called(ctx, tenantID, customerID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSalesOrderRepository) ExistsByOrderNumber(ctx context.Context, tenantID uuid.UUID, orderNumber string) (bool, error) {
	args := m.Called(ctx, tenantID, orderNumber)
	return args.Bool(0), args.Error(1)
}

func (m *MockSalesOrderRepository) GenerateOrderNumber(ctx context.Context, tenantID uuid.UUID) (string, error) {
	args := m.Called(ctx, tenantID)
	return args.String(0), args.Error(1)
}

func (m *MockSalesOrderRepository) ExistsByProduct(ctx context.Context, tenantID, productID uuid.UUID) (bool, error) {
	args := m.Called(ctx, tenantID, productID)
	return args.Bool(0), args.Error(1)
}

// Ensure mock implements the interface
var _ trade.SalesOrderRepository = (*MockSalesOrderRepository)(nil)

// Test helpers

func setupSalesOrderTestRouter() (*gin.Engine, *MockSalesOrderRepository, *SalesOrderHandler) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockSalesOrderRepository)
	service := tradeapp.NewSalesOrderService(mockRepo)
	handler := NewSalesOrderHandler(service)

	router := gin.New()
	// Add test authentication middleware that sets JWT context values
	router.Use(func(c *gin.Context) {
		setJWTContext(c, uuid.MustParse("00000000-0000-0000-0000-000000000001"), uuid.New())
		c.Next()
	})

	return router, mockRepo, handler
}

func createTestSalesOrder(tenantID uuid.UUID, orderNumber string) *trade.SalesOrder {
	now := time.Now()
	order := &trade.SalesOrder{
		OrderNumber:    orderNumber,
		CustomerID:     uuid.New(),
		CustomerName:   "Test Customer",
		TotalAmount:    decimal.NewFromFloat(999.99),
		DiscountAmount: decimal.Zero,
		PayableAmount:  decimal.NewFromFloat(999.99),
		Status:         trade.OrderStatusDraft,
		Items:          []trade.SalesOrderItem{},
	}
	order.ID = uuid.New()
	order.TenantID = tenantID
	order.CreatedAt = now
	order.UpdatedAt = now
	order.Version = 1
	return order
}

// Tests

func TestSalesOrderHandler_Create(t *testing.T) {
	t.Run("should create sales order successfully", func(t *testing.T) {
		router, mockRepo, handler := setupSalesOrderTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		customerID := uuid.New()

		router.POST("/sales-orders", handler.Create)

		mockRepo.On("GenerateOrderNumber", mock.Anything, tenantID).
			Return("SO-2026-00001", nil)
		mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*trade.SalesOrder")).
			Return(nil)

		reqBody := CreateSalesOrderRequest{
			CustomerID:   customerID.String(),
			CustomerName: "Test Customer",
			Remark:       "Test order",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/sales-orders", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", tenantID.String())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))

		mockRepo.AssertExpectations(t)
	})

	t.Run("should return error for invalid customer ID", func(t *testing.T) {
		router, _, handler := setupSalesOrderTestRouter()

		router.POST("/sales-orders", handler.Create)

		reqBody := map[string]interface{}{
			"customer_id":   "invalid-uuid",
			"customer_name": "Test Customer",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/sales-orders", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("should return error for missing required fields", func(t *testing.T) {
		router, _, handler := setupSalesOrderTestRouter()

		router.POST("/sales-orders", handler.Create)

		reqBody := map[string]interface{}{
			"customer_id": uuid.New().String(),
			// Missing customer_name
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/sales-orders", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSalesOrderHandler_GetByID(t *testing.T) {
	t.Run("should get sales order by ID", func(t *testing.T) {
		router, mockRepo, handler := setupSalesOrderTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		orderID := uuid.New()
		testOrder := createTestSalesOrder(tenantID, "SO-2026-00001")
		testOrder.ID = orderID

		router.GET("/sales-orders/:id", handler.GetByID)

		mockRepo.On("FindByIDForTenant", mock.Anything, tenantID, orderID).
			Return(testOrder, nil)

		req, _ := http.NewRequest(http.MethodGet, "/sales-orders/"+orderID.String(), nil)
		req.Header.Set("X-Tenant-ID", tenantID.String())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))

		mockRepo.AssertExpectations(t)
	})

	t.Run("should return 404 for non-existent order", func(t *testing.T) {
		router, mockRepo, handler := setupSalesOrderTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		orderID := uuid.New()

		router.GET("/sales-orders/:id", handler.GetByID)

		mockRepo.On("FindByIDForTenant", mock.Anything, tenantID, orderID).
			Return(nil, shared.ErrNotFound)

		req, _ := http.NewRequest(http.MethodGet, "/sales-orders/"+orderID.String(), nil)
		req.Header.Set("X-Tenant-ID", tenantID.String())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		mockRepo.AssertExpectations(t)
	})

	t.Run("should return error for invalid order ID", func(t *testing.T) {
		router, _, handler := setupSalesOrderTestRouter()

		router.GET("/sales-orders/:id", handler.GetByID)

		req, _ := http.NewRequest(http.MethodGet, "/sales-orders/invalid-uuid", nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSalesOrderHandler_List(t *testing.T) {
	t.Run("should list sales orders", func(t *testing.T) {
		router, mockRepo, handler := setupSalesOrderTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		testOrders := []trade.SalesOrder{
			*createTestSalesOrder(tenantID, "SO-2026-00001"),
			*createTestSalesOrder(tenantID, "SO-2026-00002"),
		}

		router.GET("/sales-orders", handler.List)

		mockRepo.On("FindAllForTenant", mock.Anything, tenantID, mock.AnythingOfType("shared.Filter")).
			Return(testOrders, nil)
		mockRepo.On("CountForTenant", mock.Anything, tenantID, mock.AnythingOfType("shared.Filter")).
			Return(int64(2), nil)

		req, _ := http.NewRequest(http.MethodGet, "/sales-orders?page=1&page_size=20", nil)
		req.Header.Set("X-Tenant-ID", tenantID.String())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
		assert.NotNil(t, response["meta"])

		mockRepo.AssertExpectations(t)
	})
}

func TestSalesOrderHandler_Confirm(t *testing.T) {
	t.Run("should confirm draft order", func(t *testing.T) {
		router, mockRepo, handler := setupSalesOrderTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		orderID := uuid.New()
		testOrder := createTestSalesOrder(tenantID, "SO-2026-00001")
		testOrder.ID = orderID
		testOrder.Status = trade.OrderStatusDraft
		// Add an item to allow confirmation
		testOrder.Items = []trade.SalesOrderItem{
			{
				ID:          uuid.New(),
				OrderID:     orderID,
				ProductID:   uuid.New(),
				ProductName: "Test Product",
				ProductCode: "SKU-001",
				Quantity:    decimal.NewFromInt(10),
				UnitPrice:   decimal.NewFromFloat(99.99),
				Amount:      decimal.NewFromFloat(999.90),
				Unit:        "pcs",
			},
		}
		testOrder.TotalAmount = decimal.NewFromFloat(999.90)
		testOrder.PayableAmount = decimal.NewFromFloat(999.90)

		router.POST("/sales-orders/:id/confirm", handler.Confirm)

		mockRepo.On("FindByIDForTenant", mock.Anything, tenantID, orderID).
			Return(testOrder, nil)
		mockRepo.On("SaveWithLockAndEvents", mock.Anything, mock.AnythingOfType("*trade.SalesOrder"), mock.Anything).
			Return(nil)

		req, _ := http.NewRequest(http.MethodPost, "/sales-orders/"+orderID.String()+"/confirm", nil)
		req.Header.Set("X-Tenant-ID", tenantID.String())
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		mockRepo.AssertExpectations(t)
	})

	t.Run("should fail to confirm already confirmed order", func(t *testing.T) {
		router, mockRepo, handler := setupSalesOrderTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		orderID := uuid.New()
		testOrder := createTestSalesOrder(tenantID, "SO-2026-00001")
		testOrder.ID = orderID
		testOrder.Status = trade.OrderStatusConfirmed // Already confirmed

		router.POST("/sales-orders/:id/confirm", handler.Confirm)

		mockRepo.On("FindByIDForTenant", mock.Anything, tenantID, orderID).
			Return(testOrder, nil)

		req, _ := http.NewRequest(http.MethodPost, "/sales-orders/"+orderID.String()+"/confirm", nil)
		req.Header.Set("X-Tenant-ID", tenantID.String())
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return an error (422 or similar)
		assert.NotEqual(t, http.StatusOK, w.Code)

		mockRepo.AssertExpectations(t)
	})
}

func TestSalesOrderHandler_Cancel(t *testing.T) {
	t.Run("should cancel draft order", func(t *testing.T) {
		router, mockRepo, handler := setupSalesOrderTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		orderID := uuid.New()
		testOrder := createTestSalesOrder(tenantID, "SO-2026-00001")
		testOrder.ID = orderID
		testOrder.Status = trade.OrderStatusDraft

		router.POST("/sales-orders/:id/cancel", handler.Cancel)

		mockRepo.On("FindByIDForTenant", mock.Anything, tenantID, orderID).
			Return(testOrder, nil)
		mockRepo.On("SaveWithLockAndEvents", mock.Anything, mock.AnythingOfType("*trade.SalesOrder"), mock.Anything).
			Return(nil)

		reqBody := CancelOrderRequest{
			Reason: "Customer requested cancellation",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/sales-orders/"+orderID.String()+"/cancel", bytes.NewBuffer(body))
		req.Header.Set("X-Tenant-ID", tenantID.String())
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		mockRepo.AssertExpectations(t)
	})

	t.Run("should fail cancel without reason", func(t *testing.T) {
		router, _, handler := setupSalesOrderTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		orderID := uuid.New()

		router.POST("/sales-orders/:id/cancel", handler.Cancel)

		reqBody := map[string]interface{}{
			// Missing reason
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/sales-orders/"+orderID.String()+"/cancel", bytes.NewBuffer(body))
		req.Header.Set("X-Tenant-ID", tenantID.String())
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSalesOrderHandler_Delete(t *testing.T) {
	t.Run("should delete draft order", func(t *testing.T) {
		router, mockRepo, handler := setupSalesOrderTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		orderID := uuid.New()
		testOrder := createTestSalesOrder(tenantID, "SO-2026-00001")
		testOrder.ID = orderID
		testOrder.Status = trade.OrderStatusDraft

		router.DELETE("/sales-orders/:id", handler.Delete)

		mockRepo.On("FindByIDForTenant", mock.Anything, tenantID, orderID).
			Return(testOrder, nil)
		mockRepo.On("DeleteForTenant", mock.Anything, tenantID, orderID).
			Return(nil)

		req, _ := http.NewRequest(http.MethodDelete, "/sales-orders/"+orderID.String(), nil)
		req.Header.Set("X-Tenant-ID", tenantID.String())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		mockRepo.AssertExpectations(t)
	})

	t.Run("should fail to delete non-draft order", func(t *testing.T) {
		router, mockRepo, handler := setupSalesOrderTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		orderID := uuid.New()
		testOrder := createTestSalesOrder(tenantID, "SO-2026-00001")
		testOrder.ID = orderID
		testOrder.Status = trade.OrderStatusConfirmed // Not draft

		router.DELETE("/sales-orders/:id", handler.Delete)

		mockRepo.On("FindByIDForTenant", mock.Anything, tenantID, orderID).
			Return(testOrder, nil)

		req, _ := http.NewRequest(http.MethodDelete, "/sales-orders/"+orderID.String(), nil)
		req.Header.Set("X-Tenant-ID", tenantID.String())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return error for non-draft order
		assert.NotEqual(t, http.StatusNoContent, w.Code)

		mockRepo.AssertExpectations(t)
	})
}

func TestSalesOrderHandler_GetStatusSummary(t *testing.T) {
	t.Run("should get status summary", func(t *testing.T) {
		router, mockRepo, handler := setupSalesOrderTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

		router.GET("/sales-orders/stats/summary", handler.GetStatusSummary)

		mockRepo.On("CountByStatus", mock.Anything, tenantID, trade.OrderStatusDraft).Return(int64(5), nil)
		mockRepo.On("CountByStatus", mock.Anything, tenantID, trade.OrderStatusConfirmed).Return(int64(10), nil)
		mockRepo.On("CountByStatus", mock.Anything, tenantID, trade.OrderStatusShipped).Return(int64(8), nil)
		mockRepo.On("CountByStatus", mock.Anything, tenantID, trade.OrderStatusCompleted).Return(int64(100), nil)
		mockRepo.On("CountByStatus", mock.Anything, tenantID, trade.OrderStatusCancelled).Return(int64(3), nil)

		req, _ := http.NewRequest(http.MethodGet, "/sales-orders/stats/summary", nil)
		req.Header.Set("X-Tenant-ID", tenantID.String())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))

		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(5), data["draft"])
		assert.Equal(t, float64(10), data["confirmed"])
		assert.Equal(t, float64(8), data["shipped"])
		assert.Equal(t, float64(100), data["completed"])
		assert.Equal(t, float64(3), data["cancelled"])
		assert.Equal(t, float64(126), data["total"])

		mockRepo.AssertExpectations(t)
	})
}

func TestSalesOrderHandler_AddItem(t *testing.T) {
	t.Run("should add item to draft order", func(t *testing.T) {
		router, mockRepo, handler := setupSalesOrderTestRouter()

		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		orderID := uuid.New()
		productID := uuid.New()
		testOrder := createTestSalesOrder(tenantID, "SO-2026-00001")
		testOrder.ID = orderID
		testOrder.Status = trade.OrderStatusDraft

		router.POST("/sales-orders/:id/items", handler.AddItem)

		mockRepo.On("FindByIDForTenant", mock.Anything, tenantID, orderID).
			Return(testOrder, nil)
		mockRepo.On("SaveWithLock", mock.Anything, mock.AnythingOfType("*trade.SalesOrder")).
			Return(nil)

		reqBody := AddOrderItemRequest{
			ProductID:   productID.String(),
			ProductName: "Test Product",
			ProductCode: "SKU-001",
			Unit:        "pcs",
			Quantity:    10,
			UnitPrice:   99.99,
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/sales-orders/"+orderID.String()+"/items", bytes.NewBuffer(body))
		req.Header.Set("X-Tenant-ID", tenantID.String())
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		mockRepo.AssertExpectations(t)
	})
}

// Suppress unused imports
var _ = errors.New
