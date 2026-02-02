// Package printer provides printer management and printing capabilities.
package printer

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

// CUPSConfig contains configuration for the CUPS backend.
type CUPSConfig struct {
	// ServerAddress is the CUPS server address (optional, uses localhost if empty).
	ServerAddress string
	// Logger is the logger to use.
	Logger *zap.Logger
}

// CUPSBackend implements the Backend interface using CUPS command-line tools.
// This backend is available on Linux and macOS systems with CUPS installed.
type CUPSBackend struct {
	serverAddress string
	logger        *zap.Logger
	available     bool
	lpstatPath    string
	lpPath        string
	cancelPath    string
}

// NewCUPSBackend creates a new CUPS backend.
func NewCUPSBackend(cfg CUPSConfig) (*CUPSBackend, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	backend := &CUPSBackend{
		serverAddress: cfg.ServerAddress,
		logger:        cfg.Logger,
		available:     false,
	}

	// Find CUPS command-line tools
	lpstatPath, err := exec.LookPath("lpstat")
	if err != nil {
		cfg.Logger.Warn("lpstat not found, CUPS backend unavailable")
		return backend, nil
	}
	backend.lpstatPath = lpstatPath

	lpPath, err := exec.LookPath("lp")
	if err != nil {
		cfg.Logger.Warn("lp not found, CUPS backend unavailable")
		return backend, nil
	}
	backend.lpPath = lpPath

	cancelPath, _ := exec.LookPath("cancel")
	backend.cancelPath = cancelPath

	// Test CUPS availability
	if err := backend.testConnection(); err != nil {
		cfg.Logger.Warn("CUPS not available", zap.Error(err))
		return backend, nil
	}

	backend.available = true
	return backend, nil
}

// testConnection tests the connection to CUPS.
func (b *CUPSBackend) testConnection() error {
	args := []string{"-r"}
	if b.serverAddress != "" {
		args = append(args, "-h", b.serverAddress)
	}

	cmd := exec.Command(b.lpstatPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %v - %s", ErrConnectionFailed, err, string(output))
	}

	// Check if scheduler is running
	if strings.Contains(string(output), "scheduler is not running") {
		return fmt.Errorf("%w: CUPS scheduler is not running", ErrConnectionFailed)
	}

	return nil
}

// ListPrinters returns a list of available printers.
func (b *CUPSBackend) ListPrinters(ctx context.Context) ([]PrinterInfo, error) {
	if !b.available {
		return nil, ErrConnectionFailed
	}

	// Get list of printers
	args := []string{"-p", "-d"}
	if b.serverAddress != "" {
		args = append(args, "-h", b.serverAddress)
	}

	cmd := exec.CommandContext(ctx, b.lpstatPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%w: %v - %s", ErrPrintFailed, err, string(output))
	}

	printers := make([]PrinterInfo, 0)
	defaultPrinter := ""

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse default printer line
		if strings.HasPrefix(line, "system default destination:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				defaultPrinter = strings.TrimSpace(parts[1])
			}
			continue
		}

		// Parse printer line: "printer PrinterName is idle. enabled since ..."
		if strings.HasPrefix(line, "printer ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				name := parts[1]
				info := PrinterInfo{
					Name:      name,
					IsDefault: name == defaultPrinter,
					State:     PrinterStateIdle,
				}

				// Parse state
				if strings.Contains(line, "is idle") {
					info.State = PrinterStateIdle
				} else if strings.Contains(line, "now printing") || strings.Contains(line, "processing") {
					info.State = PrinterStateProcessing
				} else if strings.Contains(line, "disabled") || strings.Contains(line, "stopped") {
					info.State = PrinterStateStopped
				}

				// Get additional info
				b.enrichPrinterInfo(ctx, &info)

				printers = append(printers, info)
			}
		}
	}

	return printers, nil
}

// enrichPrinterInfo adds additional information to a printer.
func (b *CUPSBackend) enrichPrinterInfo(ctx context.Context, info *PrinterInfo) {
	// Get printer options
	args := []string{"-l", "-p", info.Name}
	if b.serverAddress != "" {
		args = append(args, "-h", b.serverAddress)
	}

	cmd := exec.CommandContext(ctx, "lpoptions", args...)
	output, _ := cmd.CombinedOutput()

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse media sizes
		if strings.HasPrefix(line, "PageSize/") || strings.HasPrefix(line, "media/") {
			if idx := strings.Index(line, ":"); idx > 0 {
				values := strings.TrimSpace(line[idx+1:])
				for _, v := range strings.Fields(values) {
					v = strings.TrimPrefix(v, "*") // Remove default marker
					info.SupportedMediaSizes = append(info.SupportedMediaSizes, MediaSize(v))
				}
			}
		}

		// Check for duplex support
		if strings.Contains(line, "Duplex") || strings.Contains(line, "sides") {
			if strings.Contains(line, "DuplexNoTumble") || strings.Contains(line, "two-sided") {
				info.SupportsDuplex = true
			}
		}

		// Check for color support
		if strings.Contains(line, "ColorModel") || strings.Contains(line, "print-color-mode") {
			if strings.Contains(line, "RGB") || strings.Contains(line, "CMYK") || strings.Contains(line, "color") {
				info.SupportsColor = true
			}
		}
	}

	// Get printer description
	args = []string{"-v", "-p", info.Name}
	if b.serverAddress != "" {
		args = append(args, "-h", b.serverAddress)
	}

	cmd = exec.CommandContext(ctx, b.lpstatPath, args...)
	output, _ = cmd.CombinedOutput()

	lines = strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "device for") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				info.URI = strings.TrimSpace(parts[1])
			}
		}
	}
}

// GetPrinter returns information about a specific printer.
func (b *CUPSBackend) GetPrinter(ctx context.Context, name string) (*PrinterInfo, error) {
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
func (b *CUPSBackend) Print(ctx context.Context, printerName string, data []byte, opts PrintOptions) (int, error) {
	if !b.available {
		return 0, ErrConnectionFailed
	}

	if len(data) == 0 {
		return 0, fmt.Errorf("%w: document data is empty", ErrInvalidInput)
	}

	// Verify printer exists
	_, err := b.GetPrinter(ctx, printerName)
	if err != nil {
		return 0, err
	}

	// Create temporary file for the document
	tmpFile, err := os.CreateTemp("", "print-*.pdf")
	if err != nil {
		return 0, fmt.Errorf("%w: creating temp file: %v", ErrPrintFailed, err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return 0, fmt.Errorf("%w: writing temp file: %v", ErrPrintFailed, err)
	}
	tmpFile.Close()

	// Build lp command arguments
	args := []string{"-d", printerName}

	if b.serverAddress != "" {
		args = append(args, "-h", b.serverAddress)
	}

	if opts.JobName != "" {
		args = append(args, "-t", opts.JobName)
	}

	if opts.Copies > 0 {
		args = append(args, "-n", strconv.Itoa(opts.Copies))
	}

	// Build options string
	options := b.buildOptions(opts)
	for _, opt := range options {
		args = append(args, "-o", opt)
	}

	// Add file path
	args = append(args, tmpFile.Name())

	b.logger.Debug("Executing lp command",
		zap.String("printer", printerName),
		zap.Strings("args", args))

	// Execute lp command
	cmd := exec.CommandContext(ctx, b.lpPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("%w: %v - %s", ErrPrintFailed, err, stderr.String())
	}

	// Parse job ID from output
	// Output format: "request id is PrinterName-123 (1 file(s))"
	jobID := b.parseJobID(stdout.String())

	return jobID, nil
}

// buildOptions builds CUPS options from PrintOptions.
func (b *CUPSBackend) buildOptions(opts PrintOptions) []string {
	var options []string

	if opts.Media != "" {
		// Convert media size to CUPS format
		media := string(opts.Media)
		// Handle common conversions
		switch opts.Media {
		case MediaA4:
			media = "A4"
		case MediaA3:
			media = "A3"
		case MediaLetter:
			media = "Letter"
		case MediaLegal:
			media = "Legal"
		}
		options = append(options, "media="+media)
	}

	if opts.Quality > 0 {
		var quality string
		switch opts.Quality {
		case QualityDraft:
			quality = "draft"
		case QualityNormal:
			quality = "normal"
		case QualityHigh:
			quality = "high"
		}
		if quality != "" {
			options = append(options, "print-quality="+quality)
		}
	}

	if opts.Orientation > 0 {
		var orientation string
		switch opts.Orientation {
		case OrientationPortrait:
			orientation = "portrait"
		case OrientationLandscape:
			orientation = "landscape"
		}
		if orientation != "" {
			options = append(options, "orientation-requested="+orientation)
		}
	}

	if opts.Sides != "" {
		options = append(options, "sides="+string(opts.Sides))
	}

	if opts.ColorMode != "" {
		options = append(options, "print-color-mode="+opts.ColorMode)
	}

	if opts.PageRanges != "" {
		options = append(options, "page-ranges="+opts.PageRanges)
	}

	return options
}

// parseJobID extracts the job ID from lp output.
func (b *CUPSBackend) parseJobID(output string) int {
	// Output format: "request id is PrinterName-123 (1 file(s))"
	re := regexp.MustCompile(`request id is \S+-(\d+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		jobID, _ := strconv.Atoi(matches[1])
		return jobID
	}
	return 0
}

// GetJobStatus returns the status of a print job.
func (b *CUPSBackend) GetJobStatus(ctx context.Context, jobID int) (*PrintJobStatus, error) {
	if !b.available {
		return nil, ErrConnectionFailed
	}

	// Get job status using lpstat
	args := []string{"-W", "all", "-o"}
	if b.serverAddress != "" {
		args = append(args, "-h", b.serverAddress)
	}

	cmd := exec.CommandContext(ctx, b.lpstatPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%w: %v - %s", ErrPrintFailed, err, string(output))
	}

	// Parse output to find job
	// Format: "PrinterName-123 user 1024 Mon Jan 01 12:00:00 2024"
	lines := strings.Split(string(output), "\n")
	jobIDStr := strconv.Itoa(jobID)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if this line contains our job ID
		if strings.Contains(line, "-"+jobIDStr+" ") || strings.HasSuffix(strings.Fields(line)[0], "-"+jobIDStr) {
			status := &PrintJobStatus{
				JobID: jobID,
				State: JobStatePending,
			}

			parts := strings.Fields(line)
			if len(parts) >= 1 {
				// Extract printer name
				jobParts := strings.Split(parts[0], "-")
				if len(jobParts) >= 2 {
					status.PrinterName = strings.Join(jobParts[:len(jobParts)-1], "-")
				}
			}

			// Determine state based on output
			if strings.Contains(line, "completed") {
				status.State = JobStateCompleted
			} else if strings.Contains(line, "processing") || strings.Contains(line, "printing") {
				status.State = JobStateProcessing
			} else if strings.Contains(line, "held") {
				status.State = JobStateHeld
			} else if strings.Contains(line, "canceled") || strings.Contains(line, "cancelled") {
				status.State = JobStateCanceled
			}

			return status, nil
		}
	}

	// Job not found in active jobs, check completed jobs
	return nil, fmt.Errorf("%w: job %d not found", ErrPrintFailed, jobID)
}

// CancelJob cancels a print job.
func (b *CUPSBackend) CancelJob(ctx context.Context, jobID int) error {
	if !b.available {
		return ErrConnectionFailed
	}

	if b.cancelPath == "" {
		return fmt.Errorf("%w: cancel command not available", ErrPrintFailed)
	}

	args := []string{strconv.Itoa(jobID)}
	if b.serverAddress != "" {
		args = append(args, "-h", b.serverAddress)
	}

	cmd := exec.CommandContext(ctx, b.cancelPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %v - %s", ErrPrintFailed, err, string(output))
	}

	return nil
}

// Protocol returns the protocol used by this backend.
func (b *CUPSBackend) Protocol() Protocol {
	return ProtocolCUPS
}

// IsAvailable returns true if the backend is available.
func (b *CUPSBackend) IsAvailable() bool {
	return b.available
}

// Close releases resources held by the backend.
func (b *CUPSBackend) Close() error {
	// CUPS backend doesn't hold persistent resources
	return nil
}
