package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	printingapp "github.com/erp/backend/internal/application/printing"
	"github.com/erp/backend/internal/domain/printing"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Mock Repository
// ============================================================================

// MockAutoPrintRuleRepository is a mock implementation of AutoPrintRuleRepository
type MockAutoPrintRuleRepository struct {
	mock.Mock
}

func (m *MockAutoPrintRuleRepository) FindByID(ctx context.Context, id uuid.UUID) (*printing.AutoPrintRule, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*printing.AutoPrintRule), args.Error(1)
}

func (m *MockAutoPrintRuleRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*printing.AutoPrintRule, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*printing.AutoPrintRule), args.Error(1)
}

func (m *MockAutoPrintRuleRepository) FindAll(ctx context.Context, filter shared.Filter) ([]printing.AutoPrintRule, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]printing.AutoPrintRule), args.Error(1)
}

func (m *MockAutoPrintRuleRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]printing.AutoPrintRule, error) {
	args := m.Called(ctx, tenantID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]printing.AutoPrintRule), args.Error(1)
}

func (m *MockAutoPrintRuleRepository) FindByDocTypeAndEvent(ctx context.Context, tenantID uuid.UUID, docType printing.DocType, event printing.TriggerEvent) (*printing.AutoPrintRule, error) {
	args := m.Called(ctx, tenantID, docType, event)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*printing.AutoPrintRule), args.Error(1)
}

func (m *MockAutoPrintRuleRepository) FindEnabledByDocTypeAndEvent(ctx context.Context, tenantID uuid.UUID, docType printing.DocType, event printing.TriggerEvent) (*printing.AutoPrintRule, error) {
	args := m.Called(ctx, tenantID, docType, event)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*printing.AutoPrintRule), args.Error(1)
}

func (m *MockAutoPrintRuleRepository) FindEnabledForTenant(ctx context.Context, tenantID uuid.UUID) ([]printing.AutoPrintRule, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]printing.AutoPrintRule), args.Error(1)
}

func (m *MockAutoPrintRuleRepository) ExistsByDocTypeAndEvent(ctx context.Context, tenantID uuid.UUID, docType printing.DocType, event printing.TriggerEvent) (bool, error) {
	args := m.Called(ctx, tenantID, docType, event)
	return args.Bool(0), args.Error(1)
}

func (m *MockAutoPrintRuleRepository) Save(ctx context.Context, rule *printing.AutoPrintRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *MockAutoPrintRuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAutoPrintRuleRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *MockAutoPrintRuleRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAutoPrintRuleRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

// ============================================================================
// Test Helpers
// ============================================================================

func setupAutoPrintRuleTestRouter(handler *AutoPrintRuleHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Add middleware to set tenant ID and user ID in context
	r.Use(func(c *gin.Context) {
		c.Set(middleware.JWTTenantIDKey, testTenantID.String())
		c.Set(middleware.JWTUserIDKey, testUserID.String())
		c.Next()
	})

	// Register routes
	group := r.Group("/api/v1/printing")
	group.POST("/auto-rules", handler.CreateRule)
	group.GET("/auto-rules", handler.ListRules)
	group.GET("/auto-rules/lookup", handler.LookupRule)
	group.GET("/auto-rules/trigger-events", handler.GetTriggerEvents)
	group.GET("/auto-rules/:id", handler.GetRule)
	group.PUT("/auto-rules/:id", handler.UpdateRule)
	group.DELETE("/auto-rules/:id", handler.DeleteRule)
	group.POST("/auto-rules/:id/enable", handler.EnableRule)
	group.POST("/auto-rules/:id/disable", handler.DisableRule)

	return r
}

var (
	testTenantID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	testUserID   = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	testRuleID   = uuid.MustParse("33333333-3333-3333-3333-333333333333")
)

func createTestAutoPrintRule(tenantID uuid.UUID) *printing.AutoPrintRule {
	return printing.ReconstructAutoPrintRule(
		testRuleID,
		tenantID,
		printing.DocTypeSalesOrder,
		printing.TriggerEventConfirmed,
		nil,
		true,
		1,
		"Test Printer",
		true,
		1,
		shared.BaseEntity{
			ID:        testRuleID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	)
}

// ============================================================================
// Tests
// ============================================================================

func TestAutoPrintRuleHandler_CreateRule(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockAutoPrintRuleRepository)
		service := printingapp.NewAutoPrintRuleService(mockRepo)
		handler := NewAutoPrintRuleHandler(service)
		router := setupAutoPrintRuleTestRouter(handler)

		// Setup mock expectations
		mockRepo.On("ExistsByDocTypeAndEvent", mock.Anything, testTenantID, printing.DocTypeSalesOrder, printing.TriggerEventConfirmed).Return(false, nil)
		mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*printing.AutoPrintRule")).Return(nil)

		// Create request
		reqBody := CreateAutoPrintRuleHTTPRequest{
			DocumentType: "SALES_ORDER",
			TriggerEvent: "CONFIRMED",
			AutoPrint:    true,
			PrinterName:  "Test Printer",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/printing/auto-rules", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("duplicate rule", func(t *testing.T) {
		mockRepo := new(MockAutoPrintRuleRepository)
		service := printingapp.NewAutoPrintRuleService(mockRepo)
		handler := NewAutoPrintRuleHandler(service)
		router := setupAutoPrintRuleTestRouter(handler)

		// Setup mock expectations - rule already exists
		mockRepo.On("ExistsByDocTypeAndEvent", mock.Anything, testTenantID, printing.DocTypeSalesOrder, printing.TriggerEventConfirmed).Return(true, nil)

		// Create request
		reqBody := CreateAutoPrintRuleHTTPRequest{
			DocumentType: "SALES_ORDER",
			TriggerEvent: "CONFIRMED",
			AutoPrint:    true,
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/printing/auto-rules", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid document type", func(t *testing.T) {
		mockRepo := new(MockAutoPrintRuleRepository)
		service := printingapp.NewAutoPrintRuleService(mockRepo)
		handler := NewAutoPrintRuleHandler(service)
		router := setupAutoPrintRuleTestRouter(handler)

		// Create request with invalid document type
		reqBody := CreateAutoPrintRuleHTTPRequest{
			DocumentType: "INVALID_TYPE",
			TriggerEvent: "CONFIRMED",
			AutoPrint:    true,
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/printing/auto-rules", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("missing required fields", func(t *testing.T) {
		mockRepo := new(MockAutoPrintRuleRepository)
		service := printingapp.NewAutoPrintRuleService(mockRepo)
		handler := NewAutoPrintRuleHandler(service)
		router := setupAutoPrintRuleTestRouter(handler)

		// Create request with missing fields
		reqBody := map[string]interface{}{
			"auto_print": true,
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/printing/auto-rules", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAutoPrintRuleHandler_GetRule(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockAutoPrintRuleRepository)
		service := printingapp.NewAutoPrintRuleService(mockRepo)
		handler := NewAutoPrintRuleHandler(service)
		router := setupAutoPrintRuleTestRouter(handler)

		rule := createTestAutoPrintRule(testTenantID)
		mockRepo.On("FindByIDForTenant", mock.Anything, testTenantID, testRuleID).Return(rule, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/printing/auto-rules/"+testRuleID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := new(MockAutoPrintRuleRepository)
		service := printingapp.NewAutoPrintRuleService(mockRepo)
		handler := NewAutoPrintRuleHandler(service)
		router := setupAutoPrintRuleTestRouter(handler)

		mockRepo.On("FindByIDForTenant", mock.Anything, testTenantID, testRuleID).Return(nil, shared.ErrNotFound)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/printing/auto-rules/"+testRuleID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid uuid", func(t *testing.T) {
		mockRepo := new(MockAutoPrintRuleRepository)
		service := printingapp.NewAutoPrintRuleService(mockRepo)
		handler := NewAutoPrintRuleHandler(service)
		router := setupAutoPrintRuleTestRouter(handler)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/printing/auto-rules/invalid-uuid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAutoPrintRuleHandler_ListRules(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockAutoPrintRuleRepository)
		service := printingapp.NewAutoPrintRuleService(mockRepo)
		handler := NewAutoPrintRuleHandler(service)
		router := setupAutoPrintRuleTestRouter(handler)

		rules := []printing.AutoPrintRule{*createTestAutoPrintRule(testTenantID)}
		mockRepo.On("FindAllForTenant", mock.Anything, testTenantID, mock.AnythingOfType("shared.Filter")).Return(rules, nil)
		mockRepo.On("CountForTenant", mock.Anything, testTenantID, mock.AnythingOfType("shared.Filter")).Return(int64(1), nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/printing/auto-rules", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("with filters", func(t *testing.T) {
		mockRepo := new(MockAutoPrintRuleRepository)
		service := printingapp.NewAutoPrintRuleService(mockRepo)
		handler := NewAutoPrintRuleHandler(service)
		router := setupAutoPrintRuleTestRouter(handler)

		rules := []printing.AutoPrintRule{*createTestAutoPrintRule(testTenantID)}
		mockRepo.On("FindAllForTenant", mock.Anything, testTenantID, mock.AnythingOfType("shared.Filter")).Return(rules, nil)
		mockRepo.On("CountForTenant", mock.Anything, testTenantID, mock.AnythingOfType("shared.Filter")).Return(int64(1), nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/printing/auto-rules?document_type=SALES_ORDER&enabled=true", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockRepo.AssertExpectations(t)
	})
}

func TestAutoPrintRuleHandler_UpdateRule(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockAutoPrintRuleRepository)
		service := printingapp.NewAutoPrintRuleService(mockRepo)
		handler := NewAutoPrintRuleHandler(service)
		router := setupAutoPrintRuleTestRouter(handler)

		rule := createTestAutoPrintRule(testTenantID)
		mockRepo.On("FindByIDForTenant", mock.Anything, testTenantID, testRuleID).Return(rule, nil)
		mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*printing.AutoPrintRule")).Return(nil)

		reqBody := UpdateAutoPrintRuleHTTPRequest{
			AutoPrint:   false,
			PrinterName: "New Printer",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/printing/auto-rules/"+testRuleID.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := new(MockAutoPrintRuleRepository)
		service := printingapp.NewAutoPrintRuleService(mockRepo)
		handler := NewAutoPrintRuleHandler(service)
		router := setupAutoPrintRuleTestRouter(handler)

		mockRepo.On("FindByIDForTenant", mock.Anything, testTenantID, testRuleID).Return(nil, shared.ErrNotFound)

		reqBody := UpdateAutoPrintRuleHTTPRequest{
			AutoPrint: false,
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/printing/auto-rules/"+testRuleID.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockRepo.AssertExpectations(t)
	})
}

func TestAutoPrintRuleHandler_DeleteRule(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockAutoPrintRuleRepository)
		service := printingapp.NewAutoPrintRuleService(mockRepo)
		handler := NewAutoPrintRuleHandler(service)
		router := setupAutoPrintRuleTestRouter(handler)

		mockRepo.On("DeleteForTenant", mock.Anything, testTenantID, testRuleID).Return(nil)

		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/printing/auto-rules/"+testRuleID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := new(MockAutoPrintRuleRepository)
		service := printingapp.NewAutoPrintRuleService(mockRepo)
		handler := NewAutoPrintRuleHandler(service)
		router := setupAutoPrintRuleTestRouter(handler)

		mockRepo.On("DeleteForTenant", mock.Anything, testTenantID, testRuleID).Return(shared.ErrNotFound)

		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/printing/auto-rules/"+testRuleID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockRepo.AssertExpectations(t)
	})
}

func TestAutoPrintRuleHandler_EnableRule(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockAutoPrintRuleRepository)
		service := printingapp.NewAutoPrintRuleService(mockRepo)
		handler := NewAutoPrintRuleHandler(service)
		router := setupAutoPrintRuleTestRouter(handler)

		rule := createTestAutoPrintRule(testTenantID)
		rule.Disable() // Start disabled
		mockRepo.On("FindByIDForTenant", mock.Anything, testTenantID, testRuleID).Return(rule, nil)
		mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*printing.AutoPrintRule")).Return(nil)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/printing/auto-rules/"+testRuleID.String()+"/enable", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockRepo.AssertExpectations(t)
	})
}

func TestAutoPrintRuleHandler_DisableRule(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockAutoPrintRuleRepository)
		service := printingapp.NewAutoPrintRuleService(mockRepo)
		handler := NewAutoPrintRuleHandler(service)
		router := setupAutoPrintRuleTestRouter(handler)

		rule := createTestAutoPrintRule(testTenantID)
		mockRepo.On("FindByIDForTenant", mock.Anything, testTenantID, testRuleID).Return(rule, nil)
		mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*printing.AutoPrintRule")).Return(nil)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/printing/auto-rules/"+testRuleID.String()+"/disable", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockRepo.AssertExpectations(t)
	})
}

func TestAutoPrintRuleHandler_LookupRule(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockAutoPrintRuleRepository)
		service := printingapp.NewAutoPrintRuleService(mockRepo)
		handler := NewAutoPrintRuleHandler(service)
		router := setupAutoPrintRuleTestRouter(handler)

		rule := createTestAutoPrintRule(testTenantID)
		mockRepo.On("FindByDocTypeAndEvent", mock.Anything, testTenantID, printing.DocTypeSalesOrder, printing.TriggerEventConfirmed).Return(rule, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/printing/auto-rules/lookup?document_type=SALES_ORDER&trigger_event=CONFIRMED", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := new(MockAutoPrintRuleRepository)
		service := printingapp.NewAutoPrintRuleService(mockRepo)
		handler := NewAutoPrintRuleHandler(service)
		router := setupAutoPrintRuleTestRouter(handler)

		mockRepo.On("FindByDocTypeAndEvent", mock.Anything, testTenantID, printing.DocTypeSalesOrder, printing.TriggerEventConfirmed).Return(nil, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/printing/auto-rules/lookup?document_type=SALES_ORDER&trigger_event=CONFIRMED", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("missing document_type", func(t *testing.T) {
		mockRepo := new(MockAutoPrintRuleRepository)
		service := printingapp.NewAutoPrintRuleService(mockRepo)
		handler := NewAutoPrintRuleHandler(service)
		router := setupAutoPrintRuleTestRouter(handler)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/printing/auto-rules/lookup?trigger_event=CONFIRMED", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing trigger_event", func(t *testing.T) {
		mockRepo := new(MockAutoPrintRuleRepository)
		service := printingapp.NewAutoPrintRuleService(mockRepo)
		handler := NewAutoPrintRuleHandler(service)
		router := setupAutoPrintRuleTestRouter(handler)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/printing/auto-rules/lookup?document_type=SALES_ORDER", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAutoPrintRuleHandler_GetTriggerEvents(t *testing.T) {
	t.Run("success for sales order", func(t *testing.T) {
		mockRepo := new(MockAutoPrintRuleRepository)
		service := printingapp.NewAutoPrintRuleService(mockRepo)
		handler := NewAutoPrintRuleHandler(service)
		router := setupAutoPrintRuleTestRouter(handler)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/printing/auto-rules/trigger-events?document_type=SALES_ORDER", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Parse response
		var response struct {
			Success bool                               `json:"success"`
			Data    []printingapp.TriggerEventResponse `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response.Success)
		assert.NotEmpty(t, response.Data)

		// Sales order should have CREATED, CONFIRMED, SHIPPED, COMPLETED, PAID
		eventCodes := make([]string, len(response.Data))
		for i, e := range response.Data {
			eventCodes[i] = e.Code
		}
		assert.Contains(t, eventCodes, "CREATED")
		assert.Contains(t, eventCodes, "CONFIRMED")
		assert.Contains(t, eventCodes, "SHIPPED")
	})

	t.Run("missing document_type", func(t *testing.T) {
		mockRepo := new(MockAutoPrintRuleRepository)
		service := printingapp.NewAutoPrintRuleService(mockRepo)
		handler := NewAutoPrintRuleHandler(service)
		router := setupAutoPrintRuleTestRouter(handler)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/printing/auto-rules/trigger-events", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid document_type", func(t *testing.T) {
		mockRepo := new(MockAutoPrintRuleRepository)
		service := printingapp.NewAutoPrintRuleService(mockRepo)
		handler := NewAutoPrintRuleHandler(service)
		router := setupAutoPrintRuleTestRouter(handler)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/printing/auto-rules/trigger-events?document_type=INVALID", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})
}
