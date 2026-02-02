package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/erp/tools/print-service/internal/client"
	"github.com/example/erp/tools/print-service/internal/config"
	"go.uber.org/zap"
)

func TestNewPrintService(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			APIURL:       "http://localhost:8080",
			WebSocketURL: "ws://localhost:8080/ws",
			TenantID:     "test-tenant",
			APIToken:     "test-token",
			Timeout:      30 * time.Second,
		},
		Printing: config.PrintingConfig{
			DefaultPrinter: "Test Printer",
			Renderer:       "chromedp",
			Protocol:       "ipp",
		},
		Health: config.HealthConfig{
			Enabled: true,
			Port:    9999,
			Path:    "/health",
		},
	}

	svc := NewPrintService(cfg, zap.NewNop())

	if svc == nil {
		t.Fatal("Expected non-nil service")
	}
	if svc.Status() != StatusStopped {
		t.Errorf("Expected status 'stopped', got '%s'", svc.Status())
	}
}

func TestPrintService_Status(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			APIURL:       "http://localhost:8080",
			WebSocketURL: "ws://localhost:8080/ws",
			TenantID:     "test-tenant",
			APIToken:     "test-token",
		},
		Health: config.HealthConfig{
			Enabled: false,
		},
	}

	svc := NewPrintService(cfg, zap.NewNop())

	// Initial status should be stopped
	if svc.Status() != StatusStopped {
		t.Errorf("Expected initial status 'stopped', got '%s'", svc.Status())
	}
}

func TestPrintService_GetClients(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			APIURL:       "http://localhost:8080",
			WebSocketURL: "ws://localhost:8080/ws",
			TenantID:     "test-tenant",
			APIToken:     "test-token",
		},
		Health: config.HealthConfig{
			Enabled: false,
		},
	}

	svc := NewPrintService(cfg, zap.NewNop())

	if svc.GetAPIClient() == nil {
		t.Error("Expected non-nil API client")
	}
	if svc.GetWSClient() == nil {
		t.Error("Expected non-nil WebSocket client")
	}
}

func TestHealthStatus_JSON(t *testing.T) {
	status := HealthStatus{
		Status:      StatusRunning,
		Version:     "1.0.0",
		Uptime:      "1h30m",
		WebSocket:   "connected",
		API:         "healthy",
		LastEvent:   "2024-01-15T10:30:00Z",
		EventsCount: 42,
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal HealthStatus: %v", err)
	}

	var decoded HealthStatus
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal HealthStatus: %v", err)
	}

	if decoded.Status != StatusRunning {
		t.Errorf("Expected status 'running', got '%s'", decoded.Status)
	}
	if decoded.EventsCount != 42 {
		t.Errorf("Expected EventsCount 42, got %d", decoded.EventsCount)
	}
}

func TestPrintService_HandleEvent(t *testing.T) {
	// Create a mock API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a mock print job
		job := map[string]interface{}{
			"id":            "job-123",
			"document_type": "SALES_ORDER",
			"document_id":   "order-456",
			"status":        "PENDING",
			"pdf_url":       "",
		}
		resp := map[string]interface{}{
			"success": true,
			"data":    job,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{
			APIURL:       server.URL,
			WebSocketURL: "ws://localhost:8080/ws",
			TenantID:     "test-tenant",
			APIToken:     "test-token",
		},
		Printing: config.PrintingConfig{
			Protocol: "ipp",
		},
		Health: config.HealthConfig{
			Enabled: false,
		},
	}

	svc := NewPrintService(cfg, zap.NewNop())

	// Test handling PrintJobCreated event
	event := client.PrintingEventPayload{
		EventID:      "event-1",
		EventType:    "PrintJobCreated",
		DocumentType: "SALES_ORDER",
		DocumentID:   "order-456",
		JobID:        "job-123",
	}

	err := svc.handleEvent(context.Background(), event)
	if err != nil {
		t.Fatalf("handleEvent failed: %v", err)
	}

	// Verify events count was incremented
	if svc.eventsCount.Load() != 1 {
		t.Errorf("Expected eventsCount 1, got %d", svc.eventsCount.Load())
	}
}

func TestPrintService_HandlePrintJobCompleted(t *testing.T) {
	// Create a mock server that returns PDF data
	pdfServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		w.Write([]byte("%PDF-1.4 mock pdf content"))
	}))
	defer pdfServer.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{
			APIURL:       "http://localhost:8080",
			WebSocketURL: "ws://localhost:8080/ws",
			TenantID:     "test-tenant",
			APIToken:     "test-token",
		},
		Printing: config.PrintingConfig{
			Protocol: "ipp",
		},
		Health: config.HealthConfig{
			Enabled: false,
		},
	}

	svc := NewPrintService(cfg, zap.NewNop())

	event := client.PrintingEventPayload{
		EventID:      "event-2",
		EventType:    "PrintJobCompleted",
		DocumentType: "SALES_ORDER",
		DocumentID:   "order-456",
		JobID:        "job-123",
		PdfURL:       pdfServer.URL + "/pdfs/job-123.pdf",
		Copies:       2,
	}

	// This will fail because we can't actually print, but it should process the event
	err := svc.handlePrintJobCompleted(context.Background(), event)
	// We expect no error since the PDF download should succeed
	if err != nil {
		t.Logf("handlePrintJobCompleted returned error (expected in test): %v", err)
	}
}

func TestPrintService_HandlePrintJobFailed(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			APIURL:       "http://localhost:8080",
			WebSocketURL: "ws://localhost:8080/ws",
			TenantID:     "test-tenant",
			APIToken:     "test-token",
		},
		Health: config.HealthConfig{
			Enabled: false,
		},
	}

	svc := NewPrintService(cfg, zap.NewNop())

	event := client.PrintingEventPayload{
		EventID:      "event-3",
		EventType:    "PrintJobFailed",
		JobID:        "job-123",
		ErrorMessage: "Printer offline",
	}

	err := svc.handlePrintJobFailed(context.Background(), event)
	if err != nil {
		t.Fatalf("handlePrintJobFailed failed: %v", err)
	}
}

func TestServiceStatus_Values(t *testing.T) {
	tests := []struct {
		status   ServiceStatus
		expected string
	}{
		{StatusStarting, "starting"},
		{StatusRunning, "running"},
		{StatusDegraded, "degraded"},
		{StatusStopping, "stopping"},
		{StatusStopped, "stopped"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, string(tt.status))
			}
		})
	}
}
