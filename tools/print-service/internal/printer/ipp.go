// Package printer provides printer management and printing capabilities.
package printer

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/phin1x/go-ipp"
	"go.uber.org/zap"
)

// IPPConfig contains configuration for the IPP backend.
type IPPConfig struct {
	// ServerURL is the IPP server URL (e.g., "http://localhost:631").
	ServerURL string
	// Username is the username for authentication (optional).
	Username string
	// Password is the password for authentication (optional).
	Password string
	// Logger is the logger to use.
	Logger *zap.Logger
}

// IPPBackend implements the Backend interface using IPP protocol.
type IPPBackend struct {
	serverURL string
	client    *ipp.CUPSClient
	logger    *zap.Logger
	available bool
}

// NewIPPBackend creates a new IPP backend.
func NewIPPBackend(cfg IPPConfig) (*IPPBackend, error) {
	if cfg.ServerURL == "" {
		return nil, fmt.Errorf("%w: server URL is required", ErrInvalidInput)
	}

	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	// Parse server URL
	parsedURL, err := url.Parse(cfg.ServerURL)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid server URL: %v", ErrInvalidInput, err)
	}

	// Determine port
	port := 631
	if parsedURL.Port() != "" {
		p, err := strconv.Atoi(parsedURL.Port())
		if err == nil {
			port = p
		}
	}

	// Determine if using TLS
	useTLS := parsedURL.Scheme == "https" || parsedURL.Scheme == "ipps"

	// Create CUPS client (which extends IPP client with GetPrinters)
	client := ipp.NewCUPSClient(parsedURL.Hostname(), port, cfg.Username, cfg.Password, useTLS)

	backend := &IPPBackend{
		serverURL: cfg.ServerURL,
		client:    client,
		logger:    cfg.Logger,
		available: false,
	}

	// Test connection
	if err := backend.testConnection(); err != nil {
		cfg.Logger.Warn("IPP server not available", zap.Error(err))
		return backend, nil // Return backend but mark as unavailable
	}

	backend.available = true
	return backend, nil
}

// testConnection tests the connection to the IPP server.
func (b *IPPBackend) testConnection() error {
	// Try to get printers to test connection
	if err := b.client.TestConnection(); err != nil {
		return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}
	return nil
}

// ListPrinters returns a list of available printers.
func (b *IPPBackend) ListPrinters(ctx context.Context) ([]PrinterInfo, error) {
	if !b.available {
		return nil, ErrConnectionFailed
	}

	// Get printers from IPP server
	printers, err := b.client.GetPrintersContext(ctx, []string{
		"printer-name",
		"printer-info",
		"printer-location",
		"printer-uri-supported",
		"printer-state",
		"printer-state-message",
		"printer-make-and-model",
		"media-supported",
		"sides-supported",
		"color-supported",
		"printer-is-accepting-jobs",
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPrintFailed, err)
	}

	result := make([]PrinterInfo, 0, len(printers))
	for name, attrs := range printers {
		info := PrinterInfo{
			Name: name,
		}

		// Extract attributes
		if v, ok := attrs["printer-info"]; ok && len(v) > 0 {
			if s, ok := v[0].Value.(string); ok {
				info.Description = s
			}
		}
		if v, ok := attrs["printer-location"]; ok && len(v) > 0 {
			if s, ok := v[0].Value.(string); ok {
				info.Location = s
			}
		}
		if v, ok := attrs["printer-uri-supported"]; ok && len(v) > 0 {
			if s, ok := v[0].Value.(string); ok {
				info.URI = s
			}
		}
		if v, ok := attrs["printer-state"]; ok && len(v) > 0 {
			if i, ok := v[0].Value.(int); ok {
				info.State = PrinterState(i)
			}
		}
		if v, ok := attrs["printer-state-message"]; ok && len(v) > 0 {
			if s, ok := v[0].Value.(string); ok {
				info.StateMessage = s
			}
		}
		if v, ok := attrs["printer-make-and-model"]; ok && len(v) > 0 {
			if s, ok := v[0].Value.(string); ok {
				info.MakeAndModel = s
			}
		}
		if v, ok := attrs["sides-supported"]; ok && len(v) > 0 {
			for _, side := range v {
				if s, ok := side.Value.(string); ok && strings.Contains(s, "two-sided") {
					info.SupportsDuplex = true
					break
				}
			}
		}
		if v, ok := attrs["color-supported"]; ok && len(v) > 0 {
			if supported, ok := v[0].Value.(bool); ok {
				info.SupportsColor = supported
			}
		}
		if v, ok := attrs["media-supported"]; ok {
			for _, media := range v {
				if m, ok := media.Value.(string); ok {
					info.SupportedMediaSizes = append(info.SupportedMediaSizes, MediaSize(m))
				}
			}
		}

		result = append(result, info)
	}

	return result, nil
}

// GetPrinter returns information about a specific printer.
func (b *IPPBackend) GetPrinter(ctx context.Context, name string) (*PrinterInfo, error) {
	if !b.available {
		return nil, ErrConnectionFailed
	}

	printers, err := b.ListPrinters(ctx)
	if err != nil {
		return nil, err
	}

	for _, p := range printers {
		if p.Name == name {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrPrinterNotFound, name)
}

// Print sends a document to the printer.
func (b *IPPBackend) Print(ctx context.Context, printerName string, data []byte, opts PrintOptions) (int, error) {
	if !b.available {
		return 0, ErrConnectionFailed
	}

	if len(data) == 0 {
		return 0, fmt.Errorf("%w: document data is empty", ErrInvalidInput)
	}

	// Get printer to verify it exists
	_, err := b.GetPrinter(ctx, printerName)
	if err != nil {
		return 0, err
	}

	// Build print job attributes
	jobAttrs := make(map[string]interface{})

	if opts.JobName != "" {
		jobAttrs[ipp.AttributeJobName] = opts.JobName
	}

	if opts.Copies > 0 {
		jobAttrs[ipp.AttributeCopies] = opts.Copies
	}

	if opts.Media != "" {
		jobAttrs[ipp.AttributeMedia] = string(opts.Media)
	}

	if opts.Quality > 0 {
		jobAttrs[ipp.AttributePrintQuality] = int(opts.Quality)
	}

	if opts.Orientation > 0 {
		jobAttrs[ipp.AttributeOrientationRequested] = int(opts.Orientation)
	}

	if opts.Sides != "" {
		jobAttrs[ipp.AttributeSides] = string(opts.Sides)
	}

	if opts.ColorMode != "" {
		jobAttrs["print-color-mode"] = opts.ColorMode
	}

	if opts.PageRanges != "" {
		// Parse page ranges (e.g., "1-5,8,11-13")
		ranges := parsePageRanges(opts.PageRanges)
		if len(ranges) > 0 {
			jobAttrs["page-ranges"] = ranges
		}
	}

	b.logger.Debug("Sending print job via IPP",
		zap.String("printer", printerName),
		zap.Any("attributes", jobAttrs))

	// Create document
	doc := ipp.Document{
		Document: bytes.NewReader(data),
		Size:     len(data),
		Name:     opts.JobName,
		MimeType: "application/pdf",
	}

	// Send print job
	jobID, err := b.client.PrintJobContext(ctx, doc, printerName, jobAttrs)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrPrintFailed, err)
	}

	return jobID, nil
}

// parsePageRanges parses a page range string into IPP format.
func parsePageRanges(rangeStr string) []int {
	var ranges []int
	parts := strings.Split(rangeStr, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "-") {
			// Range like "1-5"
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) == 2 {
				start, err1 := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
				end, err2 := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
				if err1 == nil && err2 == nil && start > 0 && end >= start {
					ranges = append(ranges, start, end)
				}
			}
		} else {
			// Single page like "8"
			page, err := strconv.Atoi(part)
			if err == nil && page > 0 {
				ranges = append(ranges, page, page)
			}
		}
	}

	return ranges
}

// GetJobStatus returns the status of a print job.
func (b *IPPBackend) GetJobStatus(ctx context.Context, jobID int) (*PrintJobStatus, error) {
	if !b.available {
		return nil, ErrConnectionFailed
	}

	// Get job attributes
	attrs, err := b.client.GetJobAttributesContext(ctx, jobID, []string{
		"job-id",
		"job-state",
		"job-state-message",
		"job-printer-uri",
		"job-name",
		"time-at-creation",
		"time-at-completed",
		"job-impressions-completed",
		"job-impressions",
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPrintFailed, err)
	}

	status := &PrintJobStatus{
		JobID: jobID,
	}

	// Extract attributes
	if v, ok := attrs["job-state"]; ok && len(v) > 0 {
		if i, ok := v[0].Value.(int); ok {
			status.State = JobState(i)
		}
	}
	if v, ok := attrs["job-state-message"]; ok && len(v) > 0 {
		if s, ok := v[0].Value.(string); ok {
			status.StateMessage = s
		}
	}
	if v, ok := attrs["job-printer-uri"]; ok && len(v) > 0 {
		// Extract printer name from URI
		if uri, ok := v[0].Value.(string); ok {
			parts := strings.Split(uri, "/")
			if len(parts) > 0 {
				status.PrinterName = parts[len(parts)-1]
			}
		}
	}
	if v, ok := attrs["job-name"]; ok && len(v) > 0 {
		if s, ok := v[0].Value.(string); ok {
			status.JobName = s
		}
	}
	if v, ok := attrs["job-impressions-completed"]; ok && len(v) > 0 {
		if i, ok := v[0].Value.(int); ok {
			status.PagesPrinted = i
		}
	}
	if v, ok := attrs["job-impressions"]; ok && len(v) > 0 {
		if i, ok := v[0].Value.(int); ok {
			status.TotalPages = i
		}
	}

	return status, nil
}

// CancelJob cancels a print job.
func (b *IPPBackend) CancelJob(ctx context.Context, jobID int) error {
	if !b.available {
		return ErrConnectionFailed
	}

	if err := b.client.CancelJobContext(ctx, jobID, false); err != nil {
		return fmt.Errorf("%w: %v", ErrPrintFailed, err)
	}

	return nil
}

// Protocol returns the protocol used by this backend.
func (b *IPPBackend) Protocol() Protocol {
	return ProtocolIPP
}

// IsAvailable returns true if the backend is available.
func (b *IPPBackend) IsAvailable() bool {
	return b.available
}

// Close releases resources held by the backend.
func (b *IPPBackend) Close() error {
	// IPP client doesn't hold persistent connections
	return nil
}
