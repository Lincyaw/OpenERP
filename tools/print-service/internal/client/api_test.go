package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewAPIClient(t *testing.T) {
	cfg := APIClientConfig{
		BaseURL:  "http://localhost:8080",
		Token:    "test-token",
		TenantID: "test-tenant",
		Timeout:  30 * time.Second,
	}

	client := NewAPIClient(cfg)

	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	if client.config.BaseURL != cfg.BaseURL {
		t.Errorf("Expected BaseURL '%s', got '%s'", cfg.BaseURL, client.config.BaseURL)
	}
}

func TestAPIClient_GetAutoPrintRule(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Authorization header 'Bearer test-token', got '%s'", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-Tenant-ID") != "test-tenant" {
			t.Errorf("Expected X-Tenant-ID header 'test-tenant', got '%s'", r.Header.Get("X-Tenant-ID"))
		}

		// Return response
		rules := []AutoPrintRule{
			{
				ID:           "rule-1",
				TenantID:     "test-tenant",
				DocumentType: "SALES_ORDER",
				TriggerEvent: "CONFIRMED",
				AutoPrint:    true,
				Copies:       2,
				Enabled:      true,
			},
		}
		resp := APIResponse{
			Success: true,
			Data:    mustMarshal(rules),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAPIClient(APIClientConfig{
		BaseURL:  server.URL,
		Token:    "test-token",
		TenantID: "test-tenant",
		Logger:   zap.NewNop(),
	})

	rule, err := client.GetAutoPrintRule(context.Background(), "SALES_ORDER", "CONFIRMED")
	if err != nil {
		t.Fatalf("GetAutoPrintRule failed: %v", err)
	}

	if rule == nil {
		t.Fatal("Expected non-nil rule")
	}
	if rule.ID != "rule-1" {
		t.Errorf("Expected ID 'rule-1', got '%s'", rule.ID)
	}
	if rule.DocumentType != "SALES_ORDER" {
		t.Errorf("Expected DocumentType 'SALES_ORDER', got '%s'", rule.DocumentType)
	}
	if rule.Copies != 2 {
		t.Errorf("Expected Copies 2, got %d", rule.Copies)
	}
}

func TestAPIClient_GetAutoPrintRule_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := APIResponse{
			Success: true,
			Data:    mustMarshal([]AutoPrintRule{}),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAPIClient(APIClientConfig{
		BaseURL:  server.URL,
		Token:    "test-token",
		TenantID: "test-tenant",
		Logger:   zap.NewNop(),
	})

	rule, err := client.GetAutoPrintRule(context.Background(), "SALES_ORDER", "CONFIRMED")
	if err != nil {
		t.Fatalf("GetAutoPrintRule failed: %v", err)
	}

	if rule != nil {
		t.Error("Expected nil rule for not found")
	}
}

func TestAPIClient_RenderTemplate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		// Verify request body
		var req RenderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}
		if req.DocumentType != "SALES_ORDER" {
			t.Errorf("Expected DocumentType 'SALES_ORDER', got '%s'", req.DocumentType)
		}
		if req.DocumentID != "doc-123" {
			t.Errorf("Expected DocumentID 'doc-123', got '%s'", req.DocumentID)
		}

		// Return response
		renderResp := RenderResponse{
			JobID:  "job-456",
			PdfURL: "http://localhost:8080/pdfs/job-456.pdf",
		}
		resp := APIResponse{
			Success: true,
			Data:    mustMarshal(renderResp),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAPIClient(APIClientConfig{
		BaseURL:  server.URL,
		Token:    "test-token",
		TenantID: "test-tenant",
		Logger:   zap.NewNop(),
	})

	result, err := client.RenderTemplate(context.Background(), RenderRequest{
		DocumentType: "SALES_ORDER",
		DocumentID:   "doc-123",
	})
	if err != nil {
		t.Fatalf("RenderTemplate failed: %v", err)
	}

	if result.JobID != "job-456" {
		t.Errorf("Expected JobID 'job-456', got '%s'", result.JobID)
	}
	if result.PdfURL != "http://localhost:8080/pdfs/job-456.pdf" {
		t.Errorf("Expected PdfURL 'http://localhost:8080/pdfs/job-456.pdf', got '%s'", result.PdfURL)
	}
}

func TestAPIClient_GetDocument(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/trade/sales-orders/order-123" {
			t.Errorf("Expected path '/api/v1/trade/sales-orders/order-123', got '%s'", r.URL.Path)
		}

		doc := map[string]interface{}{
			"id":     "order-123",
			"number": "SO-2024-001",
			"total":  1000.00,
		}
		resp := APIResponse{
			Success: true,
			Data:    mustMarshal(doc),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAPIClient(APIClientConfig{
		BaseURL:  server.URL,
		Token:    "test-token",
		TenantID: "test-tenant",
		Logger:   zap.NewNop(),
	})

	data, err := client.GetDocument(context.Background(), "SALES_ORDER", "order-123")
	if err != nil {
		t.Fatalf("GetDocument failed: %v", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Failed to unmarshal document: %v", err)
	}

	if doc["id"] != "order-123" {
		t.Errorf("Expected id 'order-123', got '%v'", doc["id"])
	}
}

func TestAPIClient_GetDocument_UnknownType(t *testing.T) {
	client := NewAPIClient(APIClientConfig{
		BaseURL:  "http://localhost:8080",
		Token:    "test-token",
		TenantID: "test-tenant",
		Logger:   zap.NewNop(),
	})

	_, err := client.GetDocument(context.Background(), "UNKNOWN_TYPE", "doc-123")
	if err == nil {
		t.Fatal("Expected error for unknown document type")
	}
}

func TestAPIClient_HealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("Expected path '/health', got '%s'", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewAPIClient(APIClientConfig{
		BaseURL:  server.URL,
		Token:    "test-token",
		TenantID: "test-tenant",
		Logger:   zap.NewNop(),
	})

	err := client.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}
}

func TestAPIClient_HealthCheck_Unhealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewAPIClient(APIClientConfig{
		BaseURL:  server.URL,
		Token:    "test-token",
		TenantID: "test-tenant",
		Logger:   zap.NewNop(),
	})

	err := client.HealthCheck(context.Background())
	if err == nil {
		t.Fatal("Expected error for unhealthy server")
	}
}

func TestAPIClient_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "NOT_FOUND",
				Message: "Resource not found",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAPIClient(APIClientConfig{
		BaseURL:  server.URL,
		Token:    "test-token",
		TenantID: "test-tenant",
		Logger:   zap.NewNop(),
	})

	_, err := client.GetAutoPrintRuleByID(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("Expected error for API error response")
	}
	if !contains(err.Error(), "NOT_FOUND") {
		t.Errorf("Expected error containing 'NOT_FOUND', got '%s'", err.Error())
	}
}

func TestAPIClient_getDocumentEndpoint(t *testing.T) {
	client := NewAPIClient(APIClientConfig{})

	tests := []struct {
		docType  string
		expected string
	}{
		{"SALES_ORDER", "trade/sales-orders"},
		{"SALES_RETURN", "trade/sales-returns"},
		{"PURCHASE_ORDER", "trade/purchase-orders"},
		{"PURCHASE_RETURN", "trade/purchase-returns"},
		{"RECEIPT_VOUCHER", "finance/receipt-vouchers"},
		{"PAYMENT_VOUCHER", "finance/payment-vouchers"},
		{"STOCK_TAKING", "inventory/stock-takings"},
		{"UNKNOWN", ""},
	}

	for _, tt := range tests {
		t.Run(tt.docType, func(t *testing.T) {
			result := client.getDocumentEndpoint(tt.docType)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func mustMarshal(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
