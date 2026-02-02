// Package service provides the main print service implementation.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/example/erp/tools/print-service/internal/client"
	"github.com/example/erp/tools/print-service/internal/config"
	"go.uber.org/zap"
)

// ServiceStatus represents the current status of the print service.
type ServiceStatus string

const (
	StatusStarting ServiceStatus = "starting"
	StatusRunning  ServiceStatus = "running"
	StatusDegraded ServiceStatus = "degraded"
	StatusStopping ServiceStatus = "stopping"
	StatusStopped  ServiceStatus = "stopped"
)

// HealthStatus represents the health check response.
type HealthStatus struct {
	Status      ServiceStatus `json:"status"`
	Version     string        `json:"version"`
	Uptime      string        `json:"uptime"`
	WebSocket   string        `json:"websocket"`
	API         string        `json:"api"`
	LastEvent   string        `json:"last_event,omitempty"`
	EventsCount int64         `json:"events_count"`
}

// PrintService is the main service that coordinates event listening and printing.
type PrintService struct {
	config      *config.Config
	wsClient    *client.WSClient
	apiClient   *client.APIClient
	logger      *zap.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	status      atomic.Value // ServiceStatus
	startTime   time.Time
	eventsCount atomic.Int64
	lastEvent   atomic.Value // time.Time
	healthSrv   *http.Server
	wg          sync.WaitGroup
}

// NewPrintService creates a new print service.
func NewPrintService(cfg *config.Config, logger *zap.Logger) *PrintService {
	if logger == nil {
		logger = zap.NewNop()
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create API client
	apiClient := client.NewAPIClient(client.APIClientConfig{
		BaseURL:  cfg.Server.APIURL,
		Token:    cfg.Server.APIToken,
		TenantID: cfg.Server.TenantID,
		Timeout:  cfg.Server.Timeout,
		Logger:   logger,
	})

	// Create WebSocket client
	wsClient := client.NewWSClient(client.WSClientConfig{
		URL:                  cfg.Server.WebSocketURL,
		Token:                cfg.Server.APIToken,
		ReconnectInterval:    cfg.Server.ReconnectInterval,
		MaxReconnectAttempts: cfg.Server.MaxReconnectAttempts,
		PingInterval:         30 * time.Second,
		PongTimeout:          60 * time.Second,
		Logger:               logger,
	})

	svc := &PrintService{
		config:    cfg,
		wsClient:  wsClient,
		apiClient: apiClient,
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
	}

	svc.status.Store(StatusStopped)

	// Set event handler (must be done before Start)
	if err := wsClient.SetHandler(svc.handleEvent); err != nil {
		logger.Error("Failed to set event handler", zap.Error(err))
	}

	return svc
}

// Start starts the print service.
func (s *PrintService) Start() error {
	s.status.Store(StatusStarting)
	s.startTime = time.Now()

	s.logger.Info("Starting print service",
		zap.String("api_url", s.config.Server.APIURL),
		zap.String("ws_url", s.config.Server.WebSocketURL))

	// Start health check server
	if s.config.Health.Enabled {
		if err := s.startHealthServer(); err != nil {
			s.logger.Error("Failed to start health server", zap.Error(err))
			// Continue anyway - health server is not critical
		}
	}

	// Start WebSocket client
	if err := s.wsClient.Start(); err != nil {
		s.status.Store(StatusDegraded)
		s.logger.Error("Failed to start WebSocket client", zap.Error(err))
		return fmt.Errorf("failed to start WebSocket client: %w", err)
	}

	s.status.Store(StatusRunning)
	s.logger.Info("Print service started successfully")

	return nil
}

// Stop stops the print service gracefully.
func (s *PrintService) Stop() {
	s.status.Store(StatusStopping)
	s.logger.Info("Stopping print service")

	// Cancel context
	s.cancel()

	// Stop WebSocket client
	s.wsClient.Stop()

	// Stop health server
	if s.healthSrv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.healthSrv.Shutdown(ctx); err != nil {
			s.logger.Warn("Health server shutdown error", zap.Error(err))
		}
	}

	// Wait for goroutines
	s.wg.Wait()

	s.status.Store(StatusStopped)
	s.logger.Info("Print service stopped")
}

// Status returns the current service status.
func (s *PrintService) Status() ServiceStatus {
	return s.status.Load().(ServiceStatus)
}

// handleEvent processes incoming printing events.
func (s *PrintService) handleEvent(ctx context.Context, event client.PrintingEventPayload) error {
	s.eventsCount.Add(1)
	s.lastEvent.Store(time.Now())

	s.logger.Info("Received event",
		zap.String("event_type", event.EventType),
		zap.String("event_id", event.EventID),
		zap.String("document_type", event.DocumentType),
		zap.String("document_id", event.DocumentID))

	// Handle different event types
	switch event.EventType {
	case "PrintJobCreated":
		return s.handlePrintJobCreated(ctx, event)
	case "PrintJobCompleted":
		return s.handlePrintJobCompleted(ctx, event)
	case "PrintJobFailed":
		return s.handlePrintJobFailed(ctx, event)
	default:
		s.logger.Debug("Ignoring event type",
			zap.String("event_type", event.EventType))
	}

	return nil
}

// handlePrintJobCreated handles a new print job event.
func (s *PrintService) handlePrintJobCreated(ctx context.Context, event client.PrintingEventPayload) error {
	s.logger.Info("Processing print job",
		zap.String("job_id", event.JobID),
		zap.String("document_type", event.DocumentType),
		zap.String("document_id", event.DocumentID))

	// Get the print job details
	job, err := s.apiClient.GetPrintJob(ctx, event.JobID)
	if err != nil {
		s.logger.Error("Failed to get print job",
			zap.String("job_id", event.JobID),
			zap.Error(err))
		return err
	}

	// Check if PDF is already rendered
	if job.PdfURL == "" {
		s.logger.Debug("PDF not yet rendered, waiting for completion",
			zap.String("job_id", event.JobID))
		return nil
	}

	// Download and print
	return s.downloadAndPrint(ctx, job)
}

// handlePrintJobCompleted handles a completed print job event.
func (s *PrintService) handlePrintJobCompleted(ctx context.Context, event client.PrintingEventPayload) error {
	s.logger.Info("Print job completed",
		zap.String("job_id", event.JobID),
		zap.String("pdf_url", event.PdfURL))

	// If we have a PDF URL, download and print
	if event.PdfURL != "" {
		job := &client.PrintJob{
			ID:           event.JobID,
			DocumentType: event.DocumentType,
			DocumentID:   event.DocumentID,
			PdfURL:       event.PdfURL,
			Copies:       event.Copies,
		}
		return s.downloadAndPrint(ctx, job)
	}

	return nil
}

// handlePrintJobFailed handles a failed print job event.
func (s *PrintService) handlePrintJobFailed(ctx context.Context, event client.PrintingEventPayload) error {
	s.logger.Warn("Print job failed",
		zap.String("job_id", event.JobID),
		zap.String("error", event.ErrorMessage))
	return nil
}

// downloadAndPrint downloads a PDF and sends it to the printer.
func (s *PrintService) downloadAndPrint(ctx context.Context, job *client.PrintJob) error {
	s.logger.Info("Downloading PDF",
		zap.String("job_id", job.ID),
		zap.String("pdf_url", job.PdfURL))

	// Download PDF
	pdfData, err := s.apiClient.DownloadPDF(ctx, job.PdfURL)
	if err != nil {
		s.logger.Error("Failed to download PDF",
			zap.String("job_id", job.ID),
			zap.Error(err))
		return err
	}

	s.logger.Info("PDF downloaded",
		zap.String("job_id", job.ID),
		zap.Int("size", len(pdfData)))

	// Print the PDF
	if err := s.print(ctx, job, pdfData); err != nil {
		s.logger.Error("Failed to print",
			zap.String("job_id", job.ID),
			zap.Error(err))
		return err
	}

	s.logger.Info("Print job sent to printer",
		zap.String("job_id", job.ID))

	return nil
}

// print sends a PDF to the printer.
// This is a placeholder implementation - actual printing logic depends on the protocol.
func (s *PrintService) print(ctx context.Context, job *client.PrintJob, pdfData []byte) error {
	printerName := job.PrinterName
	if printerName == "" {
		printerName = s.config.Printing.DefaultPrinter
	}

	copies := job.Copies
	if copies < 1 {
		copies = 1
	}

	s.logger.Info("Printing document",
		zap.String("job_id", job.ID),
		zap.String("printer", printerName),
		zap.Int("copies", copies),
		zap.String("protocol", s.config.Printing.Protocol))

	// TODO: Implement actual printing based on protocol
	// For now, just log the action
	switch s.config.Printing.Protocol {
	case "ipp":
		s.logger.Info("Would send to IPP server",
			zap.String("server", s.config.Printing.IPPServer))
	case "cups":
		s.logger.Info("Would send to CUPS server",
			zap.String("server", s.config.Printing.CUPSServer))
	case "raw":
		s.logger.Info("Would send raw data to printer")
	case "windows":
		s.logger.Info("Would use Windows printing API")
	}

	return nil
}

// startHealthServer starts the health check HTTP server.
func (s *PrintService) startHealthServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc(s.config.Health.Path, s.healthHandler)

	addr := fmt.Sprintf("localhost:%d", s.config.Health.Port)
	s.healthSrv = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.logger.Info("Health server starting",
			zap.String("addr", addr),
			zap.String("path", s.config.Health.Path))

		if err := s.healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Health server error", zap.Error(err))
		}
	}()

	return nil
}

// healthHandler handles health check requests.
func (s *PrintService) healthHandler(w http.ResponseWriter, r *http.Request) {
	status := s.Status()

	// Check WebSocket connection
	wsStatus := "disconnected"
	if s.wsClient.IsConnected() {
		wsStatus = "connected"
	}

	// Check API connectivity
	apiStatus := "unknown"
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := s.apiClient.HealthCheck(ctx); err == nil {
		apiStatus = "healthy"
	} else {
		apiStatus = "unhealthy"
	}

	// Get last event time
	var lastEventStr string
	if lastEvent := s.lastEvent.Load(); lastEvent != nil {
		lastEventStr = lastEvent.(time.Time).Format(time.RFC3339)
	}

	health := HealthStatus{
		Status:      status,
		Version:     "1.0.0",
		Uptime:      time.Since(s.startTime).String(),
		WebSocket:   wsStatus,
		API:         apiStatus,
		LastEvent:   lastEventStr,
		EventsCount: s.eventsCount.Load(),
	}

	// Determine HTTP status code
	httpStatus := http.StatusOK
	if status != StatusRunning {
		httpStatus = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(health)
}

// GetAPIClient returns the API client for external use.
func (s *PrintService) GetAPIClient() *client.APIClient {
	return s.apiClient
}

// GetWSClient returns the WebSocket client for external use.
func (s *PrintService) GetWSClient() *client.WSClient {
	return s.wsClient
}
