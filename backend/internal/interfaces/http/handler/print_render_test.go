package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	printingapp "github.com/erp/backend/internal/application/printing"
	"github.com/erp/backend/internal/domain/printing"
	infra "github.com/erp/backend/internal/infrastructure/printing"
	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/erp/backend/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Test Helpers
// ============================================================================

func setupRenderTemplateTestRouter(handler *PrintHandler) *gin.Engine {
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
	group.POST("/templates/:id/render", handler.RenderTemplate)

	return r
}

// createTestPrintService creates a PrintService with minimal dependencies for testing
func createTestPrintService() *printingapp.PrintService {
	// Create template store with default templates
	templateStore, _ := infra.NewTemplateStore(&infra.TemplateStoreConfig{})

	// Create template engine
	templateEngine := infra.NewTemplateEngine()

	// Create service with nil dependencies (we'll test error paths)
	return printingapp.NewPrintService(
		templateStore,
		nil, // jobRepo - not needed for render
		templateEngine,
		nil, // pdfRenderer - not needed for render
		nil, // pdfStorage - not needed for render
		nil, // dataProviders - will render with nil data
		nil, // logger
	)
}

// ============================================================================
// RenderTemplate Handler Tests
// ============================================================================

func TestRenderTemplate_Success(t *testing.T) {
	// This test verifies the handler correctly processes a valid request
	// Without data providers, the service renders with nil data (which succeeds)

	printService := createTestPrintService()
	handler := NewPrintHandler(printService, nil)
	router := setupRenderTemplateTestRouter(handler)

	// Get a valid template ID from the template store
	templates := printService.GetDocumentTypes()
	require.NotEmpty(t, templates, "Should have document types")

	// Use a known template ID format (generated from doc type + paper size)
	// For SALES_ORDER with A4 paper
	templateID := uuid.NewSHA1(
		uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
		[]byte("print-template:SALES_ORDER:A4:PORTRAIT"),
	).String()

	reqBody := RenderTemplateHTTPRequest{
		DocumentID:   uuid.New().String(),
		DocumentType: "SALES_ORDER",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/printing/templates/"+templateID+"/render", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Without data providers, the service renders with nil data (which succeeds)
	assert.Equal(t, http.StatusOK, w.Code)

	var response dto.APIResponse[RenderTemplateHTTPResponse]
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.NotEmpty(t, response.Data.HTML)
	assert.Equal(t, templateID, response.Data.TemplateID)
	assert.Equal(t, "A4", response.Data.PaperSize)
	assert.Equal(t, "PORTRAIT", response.Data.Orientation)
}

func TestRenderTemplate_InvalidTemplateID(t *testing.T) {
	printService := createTestPrintService()
	handler := NewPrintHandler(printService, nil)
	router := setupRenderTemplateTestRouter(handler)

	reqBody := RenderTemplateHTTPRequest{
		DocumentID:   uuid.New().String(),
		DocumentType: "SALES_ORDER",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/printing/templates/invalid-uuid/render", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response dto.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.NotNil(t, response.Error)
	assert.Contains(t, response.Error.Message, "Invalid template ID format")
}

func TestRenderTemplate_TemplateNotFound(t *testing.T) {
	printService := createTestPrintService()
	handler := NewPrintHandler(printService, nil)
	router := setupRenderTemplateTestRouter(handler)

	// Use a valid UUID format but non-existent template
	nonExistentTemplateID := uuid.New().String()

	reqBody := RenderTemplateHTTPRequest{
		DocumentID:   uuid.New().String(),
		DocumentType: "SALES_ORDER",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/printing/templates/"+nonExistentTemplateID+"/render", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response dto.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.NotNil(t, response.Error)
	assert.Contains(t, response.Error.Message, "Template not found")
}

func TestRenderTemplate_MissingDocumentID(t *testing.T) {
	printService := createTestPrintService()
	handler := NewPrintHandler(printService, nil)
	router := setupRenderTemplateTestRouter(handler)

	templateID := uuid.New().String()

	reqBody := map[string]string{
		"document_type": "SALES_ORDER",
		// document_id is missing
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/printing/templates/"+templateID+"/render", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRenderTemplate_MissingDocumentType(t *testing.T) {
	printService := createTestPrintService()
	handler := NewPrintHandler(printService, nil)
	router := setupRenderTemplateTestRouter(handler)

	templateID := uuid.New().String()

	reqBody := map[string]string{
		"document_id": uuid.New().String(),
		// document_type is missing
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/printing/templates/"+templateID+"/render", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRenderTemplate_InvalidDocumentID(t *testing.T) {
	printService := createTestPrintService()
	handler := NewPrintHandler(printService, nil)
	router := setupRenderTemplateTestRouter(handler)

	templateID := uuid.New().String()

	reqBody := map[string]string{
		"document_id":   "not-a-uuid",
		"document_type": "SALES_ORDER",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/printing/templates/"+templateID+"/render", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRenderTemplate_InvalidDocumentType(t *testing.T) {
	printService := createTestPrintService()
	handler := NewPrintHandler(printService, nil)
	router := setupRenderTemplateTestRouter(handler)

	// Use a valid template ID
	templateID := uuid.NewSHA1(
		uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
		[]byte("print-template:SALES_ORDER:A4:PORTRAIT"),
	).String()

	reqBody := RenderTemplateHTTPRequest{
		DocumentID:   uuid.New().String(),
		DocumentType: "INVALID_TYPE",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/printing/templates/"+templateID+"/render", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 400 (Bad Request) for invalid document type
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response dto.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.NotNil(t, response.Error)
	assert.Contains(t, response.Error.Message, "Invalid document type")
}

func TestRenderTemplate_DocumentTypeMismatch(t *testing.T) {
	printService := createTestPrintService()
	handler := NewPrintHandler(printService, nil)
	router := setupRenderTemplateTestRouter(handler)

	// Use a SALES_ORDER template
	templateID := uuid.NewSHA1(
		uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
		[]byte("print-template:SALES_ORDER:A4:PORTRAIT"),
	).String()

	// But request with PURCHASE_ORDER document type
	reqBody := RenderTemplateHTTPRequest{
		DocumentID:   uuid.New().String(),
		DocumentType: "PURCHASE_ORDER",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/printing/templates/"+templateID+"/render", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 400 (Bad Request) for document type mismatch
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response dto.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.NotNil(t, response.Error)
	assert.Contains(t, response.Error.Message, "does not match")
}

func TestRenderTemplate_EmptyBody(t *testing.T) {
	printService := createTestPrintService()
	handler := NewPrintHandler(printService, nil)
	router := setupRenderTemplateTestRouter(handler)

	templateID := uuid.New().String()

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/printing/templates/"+templateID+"/render", bytes.NewBuffer([]byte{}))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRenderTemplate_InvalidJSON(t *testing.T) {
	printService := createTestPrintService()
	handler := NewPrintHandler(printService, nil)
	router := setupRenderTemplateTestRouter(handler)

	templateID := uuid.New().String()

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/printing/templates/"+templateID+"/render", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// Test All Document Types
// ============================================================================

func TestRenderTemplate_AllDocumentTypes(t *testing.T) {
	printService := createTestPrintService()
	handler := NewPrintHandler(printService, nil)
	router := setupRenderTemplateTestRouter(handler)

	// Test document types that have A4 templates
	// Note: SALES_RECEIPT uses receipt paper sizes (58mm, 80mm), not A4
	docTypesWithA4 := []printing.DocType{
		printing.DocTypeSalesOrder,
		printing.DocTypeSalesDelivery,
		printing.DocTypeSalesReturn,
		printing.DocTypePurchaseOrder,
		printing.DocTypePurchaseReceiving,
		printing.DocTypePurchaseReturn,
		printing.DocTypeReceiptVoucher,
		printing.DocTypePaymentVoucher,
		printing.DocTypeStockTaking,
	}

	for _, docType := range docTypesWithA4 {
		t.Run(string(docType), func(t *testing.T) {
			// Generate template ID for this doc type
			templateID := uuid.NewSHA1(
				uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
				[]byte("print-template:"+string(docType)+":A4:PORTRAIT"),
			).String()

			reqBody := RenderTemplateHTTPRequest{
				DocumentID:   uuid.New().String(),
				DocumentType: string(docType),
			}
			body, _ := json.Marshal(reqBody)

			req, _ := http.NewRequest(http.MethodPost, "/api/v1/printing/templates/"+templateID+"/render", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Without data providers, the service renders with nil data (which succeeds)
			// This validates the handler correctly processes each document type
			assert.Equal(t, http.StatusOK, w.Code, "Expected 200 for doc type %s, got %d: %s", docType, w.Code, w.Body.String())
		})
	}
}

func TestRenderTemplate_SalesReceiptWithReceiptPaper(t *testing.T) {
	// SALES_RECEIPT uses receipt paper sizes (58mm, 80mm), not A4
	printService := createTestPrintService()
	handler := NewPrintHandler(printService, nil)
	router := setupRenderTemplateTestRouter(handler)

	// Generate template ID for SALES_RECEIPT with 58mm receipt paper
	templateID := uuid.NewSHA1(
		uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
		[]byte("print-template:SALES_RECEIPT:RECEIPT_58MM:PORTRAIT"),
	).String()

	reqBody := RenderTemplateHTTPRequest{
		DocumentID:   uuid.New().String(),
		DocumentType: "SALES_RECEIPT",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/printing/templates/"+templateID+"/render", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 for SALES_RECEIPT with receipt paper")
}

func TestRenderTemplate_EmptyTemplateID(t *testing.T) {
	printService := createTestPrintService()
	handler := NewPrintHandler(printService, nil)
	router := setupRenderTemplateTestRouter(handler)

	reqBody := RenderTemplateHTTPRequest{
		DocumentID:   uuid.New().String(),
		DocumentType: "SALES_ORDER",
	}
	body, _ := json.Marshal(reqBody)

	// Empty template ID in path - gin treats this as a different route
	// The route /templates//render doesn't match /templates/:id/render
	// So it returns 404 (route not found) or 400 depending on gin version
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/printing/templates//render", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return either 404 (route not found) or 400 (bad request)
	// Both are acceptable for an empty template ID
	assert.True(t, w.Code == http.StatusNotFound || w.Code == http.StatusBadRequest,
		"Expected 404 or 400 for empty template ID, got %d", w.Code)
}
