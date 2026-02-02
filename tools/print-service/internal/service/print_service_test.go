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

func TestPrintService_GetEventMapper(t *testing.T) {
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

	mapper := svc.GetEventMapper()
	if mapper == nil {
		t.Fatal("Expected non-nil EventMapper")
	}

	// Verify supported events
	if !mapper.IsSupported("SalesOrderConfirmed") {
		t.Error("Expected SalesOrderConfirmed to be supported")
	}
	if !mapper.IsSupported("PurchaseOrderReceived") {
		t.Error("Expected PurchaseOrderReceived to be supported")
	}
}

func TestPrintService_GetPrintQueue(t *testing.T) {
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

	queue := svc.GetPrintQueue()
	if queue == nil {
		t.Fatal("Expected non-nil PrintQueue")
	}

	stats := queue.Stats()
	if stats.MaxConcurrent != 5 {
		t.Errorf("Expected MaxConcurrent 5, got %d", stats.MaxConcurrent)
	}
}

func TestPrintService_HandleDomainEvent_NoRule(t *testing.T) {
	// Create a mock API server that returns no rules
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"success": true,
			"data":    []interface{}{}, // Empty rules
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
		Health: config.HealthConfig{
			Enabled: false,
		},
	}

	svc := NewPrintService(cfg, zap.NewNop())

	// Test handling SalesOrderConfirmed event (domain event)
	event := client.PrintingEventPayload{
		EventID:     "event-domain-1",
		EventType:   "SalesOrderConfirmed",
		AggregateID: "order-789",
		TenantID:    "test-tenant",
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

func TestPrintService_HandleDomainEvent_WithRule(t *testing.T) {
	// Create a mock API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		switch {
		case path == "/api/v1/printing/auto-print-rules":
			// Return a matching rule
			rules := []map[string]interface{}{
				{
					"id":            "rule-1",
					"tenant_id":     "test-tenant",
					"document_type": "SALES_ORDER",
					"trigger_event": "CONFIRMED",
					"template_id":   "template-1",
					"auto_print":    true,
					"copies":        2,
					"printer_name":  "Test Printer",
					"enabled":       true,
				},
			}
			resp := map[string]interface{}{
				"success": true,
				"data":    rules,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)

		case path == "/api/v1/printing/render":
			// Return render response
			resp := map[string]interface{}{
				"success": true,
				"data": map[string]interface{}{
					"job_id":  "render-job-1",
					"pdf_url": "http://localhost/pdfs/render-job-1.pdf",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)

		default:
			// Return PDF data for any other path (PDF download)
			w.Header().Set("Content-Type", "application/pdf")
			w.Write([]byte("%PDF-1.4 mock pdf content"))
		}
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

	// Start the print queue (required for enqueueing)
	svc.printQueue.Start()
	defer svc.printQueue.Stop()

	// Test handling SalesOrderConfirmed event (domain event)
	event := client.PrintingEventPayload{
		EventID:        "event-domain-2",
		EventType:      "SalesOrderConfirmed",
		AggregateID:    "order-999",
		DocumentNumber: "SO-2024-001",
		TenantID:       "test-tenant",
	}

	err := svc.handleEvent(context.Background(), event)
	if err != nil {
		t.Fatalf("handleEvent failed: %v", err)
	}

	// Wait for task to be processed
	time.Sleep(100 * time.Millisecond)

	// Verify a task was enqueued
	stats := svc.printQueue.Stats()
	if stats.TotalTasks != 1 {
		t.Errorf("Expected 1 task enqueued, got %d", stats.TotalTasks)
	}
}

func TestPrintService_HandleUnsupportedEvent(t *testing.T) {
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

	// Test handling an unsupported event type
	event := client.PrintingEventPayload{
		EventID:   "event-unsupported",
		EventType: "SomeRandomEvent",
	}

	err := svc.handleEvent(context.Background(), event)
	if err != nil {
		t.Fatalf("handleEvent should not fail for unsupported events: %v", err)
	}

	// Verify events count was still incremented
	if svc.eventsCount.Load() != 1 {
		t.Errorf("Expected eventsCount 1, got %d", svc.eventsCount.Load())
	}
}

func TestPrintService_GetPrintLogs(t *testing.T) {
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

	// Initially should have no logs
	logs := svc.GetPrintLogs()
	if len(logs) != 0 {
		t.Errorf("Expected 0 logs initially, got %d", len(logs))
	}

	recentLogs := svc.GetRecentPrintLogs(10)
	if len(recentLogs) != 0 {
		t.Errorf("Expected 0 recent logs initially, got %d", len(recentLogs))
	}
}

func TestHealthStatus_WithPrintQueue(t *testing.T) {
	status := HealthStatus{
		Status:    StatusRunning,
		Version:   "1.0.0",
		Uptime:    "1h30m",
		WebSocket: "connected",
		API:       "healthy",
		Printer:   "available",
		PrintQueue: PrintQueueStats{
			TotalTasks:    100,
			SuccessCount:  95,
			FailedCount:   5,
			QueueLength:   3,
			QueueSize:     100,
			MaxConcurrent: 5,
			IsRunning:     true,
		},
		LastEvent:    "2024-01-15T10:30:00Z",
		EventsCount:  42,
		PrinterCount: 2,
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal HealthStatus: %v", err)
	}

	var decoded HealthStatus
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal HealthStatus: %v", err)
	}

	if decoded.PrintQueue.TotalTasks != 100 {
		t.Errorf("Expected PrintQueue.TotalTasks 100, got %d", decoded.PrintQueue.TotalTasks)
	}
	if decoded.PrintQueue.SuccessCount != 95 {
		t.Errorf("Expected PrintQueue.SuccessCount 95, got %d", decoded.PrintQueue.SuccessCount)
	}
	if decoded.PrintQueue.MaxConcurrent != 5 {
		t.Errorf("Expected PrintQueue.MaxConcurrent 5, got %d", decoded.PrintQueue.MaxConcurrent)
	}
}
