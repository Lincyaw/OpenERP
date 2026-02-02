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
	"github.com/example/erp/tools/print-service/internal/printer"
	"github.com/google/uuid"
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
	Status       ServiceStatus   `json:"status"`
	Version      string          `json:"version"`
	BuildTime    string          `json:"build_time,omitempty"`
	GitCommit    string          `json:"git_commit,omitempty"`
	Uptime       string          `json:"uptime"`
	WebSocket    string          `json:"websocket"`
	API          string          `json:"api"`
	Printer      string          `json:"printer"`
	PrintQueue   PrintQueueStats `json:"print_queue,omitempty"`
	LastEvent    string          `json:"last_event,omitempty"`
	EventsCount  int64           `json:"events_count"`
	PrinterCount int             `json:"printer_count,omitempty"`
}

// VersionInfo contains version information set at build time.
type VersionInfo struct {
	Version   string
	BuildTime string
	GitCommit string
}

// PrintService is the main service that coordinates event listening and printing.
type PrintService struct {
	config         *config.Config
	wsClient       *client.WSClient
	apiClient      *client.APIClient
	printerManager *printer.Manager
	eventMapper    *EventMapper
	printQueue     *PrintQueue
	logger         *zap.Logger
	ctx            context.Context
	cancel         context.CancelFunc
	status         atomic.Value // ServiceStatus
	startTime      time.Time
	eventsCount    atomic.Int64
	lastEvent      atomic.Value // time.Time
	healthSrv      *http.Server
	wg             sync.WaitGroup
	versionInfo    VersionInfo
}

// NewPrintService creates a new print service.
func NewPrintService(cfg *config.Config, logger *zap.Logger, versionInfo VersionInfo) *PrintService {
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

	// Create printer manager
	var printerManager *printer.Manager
	printerMgr, err := printer.NewManager(printer.ManagerConfig{
		Protocol:       printer.Protocol(cfg.Printing.Protocol),
		IPPServer:      cfg.Printing.IPPServer,
		CUPSServer:     cfg.Printing.CUPSServer,
		DefaultPrinter: cfg.Printing.DefaultPrinter,
		Logger:         logger,
	})
	if err != nil {
		logger.Warn("Failed to create printer manager", zap.Error(err))
	} else {
		printerManager = printerMgr
	}

	// Create event mapper
	eventMapper := NewEventMapper()

	// Create print queue with configuration
	printQueueConfig := PrintQueueConfig{
		QueueSize:      100, // Buffer 100 print tasks
		MaxConcurrent:  5,   // Max 5 concurrent print tasks
		MaxRetries:     3,   // Retry 3 times
		BaseRetryDelay: 1 * time.Second,
		MaxRetryDelay:  30 * time.Second,
		Logger:         logger,
	}
	printQueue := NewPrintQueue(printQueueConfig)

	svc := &PrintService{
		config:         cfg,
		wsClient:       wsClient,
		apiClient:      apiClient,
		printerManager: printerManager,
		eventMapper:    eventMapper,
		printQueue:     printQueue,
		logger:         logger,
		ctx:            ctx,
		cancel:         cancel,
		versionInfo:    versionInfo,
	}

	svc.status.Store(StatusStopped)

	// Set print queue handler (must be done before Start)
	if err := printQueue.SetHandler(svc.executePrintTask); err != nil {
		logger.Error("Failed to set print queue handler", zap.Error(err))
	}

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

	// Start print queue
	if err := s.printQueue.Start(); err != nil {
		s.logger.Error("Failed to start print queue", zap.Error(err))
		return fmt.Errorf("failed to start print queue: %w", err)
	}

	// Start WebSocket client
	if err := s.wsClient.Start(); err != nil {
		s.status.Store(StatusDegraded)
		s.logger.Error("Failed to start WebSocket client", zap.Error(err))
		return fmt.Errorf("failed to start WebSocket client: %w", err)
	}

	s.status.Store(StatusRunning)
	s.logger.Info("Print service started successfully",
		zap.Strings("supported_events", s.eventMapper.SupportedEvents()))

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

	// Stop print queue (waits for pending tasks)
	s.printQueue.Stop()

	// Close printer manager
	if s.printerManager != nil {
		if err := s.printerManager.Close(); err != nil {
			s.logger.Warn("Printer manager close error", zap.Error(err))
		}
	}

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

	// Log final statistics
	stats := s.printQueue.Stats()
	s.logger.Info("Print service stopped",
		zap.Int64("total_tasks", stats.TotalTasks),
		zap.Int64("successful", stats.SuccessCount),
		zap.Int64("failed", stats.FailedCount))

	s.status.Store(StatusStopped)
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

	// Handle internal print job events
	switch event.EventType {
	case "PrintJobCreated":
		return s.handlePrintJobCreated(ctx, event)
	case "PrintJobCompleted":
		return s.handlePrintJobCompleted(ctx, event)
	case "PrintJobFailed":
		return s.handlePrintJobFailed(ctx, event)
	}

	// Handle domain events using event mapper
	if s.eventMapper.IsSupported(event.EventType) {
		return s.handleDomainEvent(ctx, event)
	}

	s.logger.Debug("Ignoring unsupported event type",
		zap.String("event_type", event.EventType))

	return nil
}

// handleDomainEvent handles domain events (e.g., SalesOrderConfirmed) by triggering auto-print.
func (s *PrintService) handleDomainEvent(ctx context.Context, event client.PrintingEventPayload) error {
	mapping := s.eventMapper.GetMapping(event.EventType)
	if mapping == nil {
		return nil
	}

	s.logger.Info("Processing domain event for auto-print",
		zap.String("event_type", event.EventType),
		zap.String("document_type", string(mapping.DocumentType)),
		zap.String("trigger_event", string(mapping.TriggerEvent)),
		zap.String("aggregate_id", event.AggregateID))

	// Query auto-print rules for this document type and trigger
	rule, err := s.apiClient.GetAutoPrintRule(ctx, string(mapping.DocumentType), string(mapping.TriggerEvent))
	if err != nil {
		s.logger.Error("Failed to get auto-print rule",
			zap.String("document_type", string(mapping.DocumentType)),
			zap.String("trigger_event", string(mapping.TriggerEvent)),
			zap.Error(err))
		return err
	}

	if rule == nil {
		s.logger.Debug("No auto-print rule found",
			zap.String("document_type", string(mapping.DocumentType)),
			zap.String("trigger_event", string(mapping.TriggerEvent)))
		return nil
	}

	if !rule.Enabled || !rule.AutoPrint {
		s.logger.Debug("Auto-print disabled for this rule",
			zap.String("rule_id", rule.ID))
		return nil
	}

	s.logger.Info("Auto-print rule matched",
		zap.String("rule_id", rule.ID),
		zap.String("template_id", rule.TemplateID),
		zap.Int("copies", rule.Copies),
		zap.String("printer", rule.PrinterName))

	// Trigger the complete print flow
	return s.triggerPrintFlow(ctx, event, rule, mapping)
}

// triggerPrintFlow executes the complete print flow: render template -> generate PDF -> enqueue print task.
func (s *PrintService) triggerPrintFlow(ctx context.Context, event client.PrintingEventPayload, rule *client.AutoPrintRule, mapping *EventMapping) error {
	// Step 1: Render template to get PDF
	renderReq := client.RenderRequest{
		TemplateID:   rule.TemplateID,
		DocumentType: string(mapping.DocumentType),
		DocumentID:   event.AggregateID,
	}

	s.logger.Info("Rendering template",
		zap.String("template_id", rule.TemplateID),
		zap.String("document_type", string(mapping.DocumentType)),
		zap.String("document_id", event.AggregateID))

	renderResp, err := s.apiClient.RenderTemplate(ctx, renderReq)
	if err != nil {
		s.logger.Error("Failed to render template",
			zap.String("template_id", rule.TemplateID),
			zap.Error(err))
		return err
	}

	s.logger.Info("Template rendered successfully",
		zap.String("job_id", renderResp.JobID),
		zap.String("pdf_url", renderResp.PdfURL))

	// Step 2: Download PDF
	pdfData, err := s.apiClient.DownloadPDF(ctx, renderResp.PdfURL)
	if err != nil {
		s.logger.Error("Failed to download PDF",
			zap.String("pdf_url", renderResp.PdfURL),
			zap.Error(err))
		return err
	}

	s.logger.Info("PDF downloaded",
		zap.String("job_id", renderResp.JobID),
		zap.Int("size", len(pdfData)))

	// Step 3: Create print task and enqueue
	task := &PrintTask{
		ID:             uuid.New().String(),
		DocumentType:   string(mapping.DocumentType),
		DocumentID:     event.AggregateID,
		DocumentNumber: event.DocumentNumber,
		TemplateID:     rule.TemplateID,
		PrinterName:    rule.PrinterName,
		Copies:         rule.Copies,
		PdfData:        pdfData,
	}

	// Enqueue without blocking (non-blocking)
	if err := s.printQueue.Enqueue(task); err != nil {
		s.logger.Error("Failed to enqueue print task",
			zap.String("task_id", task.ID),
			zap.Error(err))
		return err
	}

	s.logger.Info("Print task enqueued",
		zap.String("task_id", task.ID),
		zap.String("document_type", task.DocumentType),
		zap.String("document_id", task.DocumentID),
		zap.Int("copies", task.Copies))

	return nil
}

// executePrintTask is the handler called by the print queue to execute a print task.
func (s *PrintService) executePrintTask(ctx context.Context, task *PrintTask) error {
	printerName := task.PrinterName
	if printerName == "" {
		printerName = s.config.Printing.DefaultPrinter
	}

	s.logger.Info("Executing print task",
		zap.String("task_id", task.ID),
		zap.String("document_type", task.DocumentType),
		zap.String("document_id", task.DocumentID),
		zap.String("printer", printerName),
		zap.Int("copies", task.Copies))

	// Check if printer manager is available
	if s.printerManager == nil {
		return fmt.Errorf("printer manager not available")
	}

	if !s.printerManager.IsAvailable() {
		return fmt.Errorf("printer backend not available")
	}

	// Build print options
	opts := printer.DefaultPrintOptions()
	opts.Copies = task.Copies
	opts.JobName = fmt.Sprintf("ERP-%s-%s", task.DocumentType, task.ID)

	// Send to printer
	printJobID, err := s.printerManager.Print(ctx, printerName, task.PdfData, opts)
	if err != nil {
		s.logger.Error("Failed to send to printer",
			zap.String("task_id", task.ID),
			zap.String("printer", printerName),
			zap.Error(err))
		return err
	}

	s.logger.Info("Print task completed successfully",
		zap.String("task_id", task.ID),
		zap.Int("print_job_id", printJobID),
		zap.String("printer", printerName))

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

	// Check if printer manager is available
	if s.printerManager == nil {
		s.logger.Warn("Printer manager not available, skipping print")
		return fmt.Errorf("printer manager not available")
	}

	if !s.printerManager.IsAvailable() {
		s.logger.Warn("Printer backend not available, skipping print")
		return fmt.Errorf("printer backend not available")
	}

	// Build print options
	opts := printer.DefaultPrintOptions()
	opts.Copies = copies
	opts.JobName = fmt.Sprintf("ERP-%s-%s", job.DocumentType, job.ID)

	// Send to printer
	printJobID, err := s.printerManager.Print(ctx, printerName, pdfData, opts)
	if err != nil {
		s.logger.Error("Failed to send to printer",
			zap.String("job_id", job.ID),
			zap.String("printer", printerName),
			zap.Error(err))
		return err
	}

	s.logger.Info("Print job submitted successfully",
		zap.String("job_id", job.ID),
		zap.Int("print_job_id", printJobID),
		zap.String("printer", printerName))

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

	// Check printer status
	printerStatus := "unavailable"
	printerCount := 0
	if s.printerManager != nil && s.printerManager.IsAvailable() {
		printerStatus = "available"
		// Try to get printer count
		if printers, err := s.printerManager.ListPrinters(ctx); err == nil {
			printerCount = len(printers)
		}
	}

	// Get last event time
	var lastEventStr string
	if lastEvent := s.lastEvent.Load(); lastEvent != nil {
		lastEventStr = lastEvent.(time.Time).Format(time.RFC3339)
	}

	// Get print queue stats
	queueStats := s.printQueue.Stats()

	health := HealthStatus{
		Status:       status,
		Version:      s.versionInfo.Version,
		BuildTime:    s.versionInfo.BuildTime,
		GitCommit:    s.versionInfo.GitCommit,
		Uptime:       time.Since(s.startTime).String(),
		WebSocket:    wsStatus,
		API:          apiStatus,
		Printer:      printerStatus,
		PrintQueue:   queueStats,
		LastEvent:    lastEventStr,
		EventsCount:  s.eventsCount.Load(),
		PrinterCount: printerCount,
	}

	// Determine HTTP status code
	httpStatus := http.StatusOK
	if status != StatusRunning {
		httpStatus = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	if err := json.NewEncoder(w).Encode(health); err != nil {
		s.logger.Error("Failed to encode health response", zap.Error(err))
	}
}

// GetAPIClient returns the API client for external use.
func (s *PrintService) GetAPIClient() *client.APIClient {
	return s.apiClient
}

// GetWSClient returns the WebSocket client for external use.
func (s *PrintService) GetWSClient() *client.WSClient {
	return s.wsClient
}

// GetPrinterManager returns the printer manager for external use.
func (s *PrintService) GetPrinterManager() *printer.Manager {
	return s.printerManager
}

// GetPrintQueue returns the print queue for external use.
func (s *PrintService) GetPrintQueue() *PrintQueue {
	return s.printQueue
}

// GetEventMapper returns the event mapper for external use.
func (s *PrintService) GetEventMapper() *EventMapper {
	return s.eventMapper
}

// GetPrintLogs returns the print logs.
func (s *PrintService) GetPrintLogs() []PrintLog {
	return s.printQueue.GetLogs()
}

// GetRecentPrintLogs returns the most recent n print logs.
func (s *PrintService) GetRecentPrintLogs(n int) []PrintLog {
	return s.printQueue.GetRecentLogs(n)
}
