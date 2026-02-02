// Package printer provides printer management and printing capabilities.
package printer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Errors returned by the printer package.
var (
	// ErrPrinterNotFound is returned when the specified printer is not found.
	ErrPrinterNotFound = errors.New("printer: printer not found")
	// ErrPrintFailed is returned when printing fails.
	ErrPrintFailed = errors.New("printer: print failed")
	// ErrNoDefaultPrinter is returned when no default printer is configured.
	ErrNoDefaultPrinter = errors.New("printer: no default printer")
	// ErrInvalidInput is returned when input is invalid.
	ErrInvalidInput = errors.New("printer: invalid input")
	// ErrProtocolNotSupported is returned when the protocol is not supported.
	ErrProtocolNotSupported = errors.New("printer: protocol not supported")
	// ErrConnectionFailed is returned when connection to printer fails.
	ErrConnectionFailed = errors.New("printer: connection failed")
)

// Protocol represents the printing protocol.
type Protocol string

const (
	// ProtocolIPP uses Internet Printing Protocol.
	ProtocolIPP Protocol = "ipp"
	// ProtocolCUPS uses CUPS (Common Unix Printing System).
	ProtocolCUPS Protocol = "cups"
	// ProtocolRaw sends raw data directly to the printer.
	ProtocolRaw Protocol = "raw"
	// ProtocolWindows uses Windows printing API.
	ProtocolWindows Protocol = "windows"
)

// PrintQuality represents print quality settings.
type PrintQuality int

const (
	QualityDraft  PrintQuality = 3
	QualityNormal PrintQuality = 4
	QualityHigh   PrintQuality = 5
)

// MediaSize represents paper size.
type MediaSize string

const (
	MediaA4     MediaSize = "iso_a4_210x297mm"
	MediaA3     MediaSize = "iso_a3_297x420mm"
	MediaLetter MediaSize = "na_letter_8.5x11in"
	MediaLegal  MediaSize = "na_legal_8.5x14in"
)

// Orientation represents print orientation.
type Orientation int

const (
	OrientationPortrait  Orientation = 3
	OrientationLandscape Orientation = 4
)

// Sides represents duplex printing options.
type Sides string

const (
	SidesOneSided          Sides = "one-sided"
	SidesTwoSidedLongEdge  Sides = "two-sided-long-edge"
	SidesTwoSidedShortEdge Sides = "two-sided-short-edge"
)

// PrinterInfo contains information about a printer.
type PrinterInfo struct {
	// Name is the printer name.
	Name string `json:"name"`
	// Description is a human-readable description.
	Description string `json:"description,omitempty"`
	// Location is the physical location of the printer.
	Location string `json:"location,omitempty"`
	// URI is the printer URI.
	URI string `json:"uri,omitempty"`
	// IsDefault indicates if this is the default printer.
	IsDefault bool `json:"is_default"`
	// State is the current printer state.
	State PrinterState `json:"state"`
	// StateMessage is a human-readable state message.
	StateMessage string `json:"state_message,omitempty"`
	// MakeAndModel is the printer make and model.
	MakeAndModel string `json:"make_and_model,omitempty"`
	// SupportedMediaSizes lists supported paper sizes.
	SupportedMediaSizes []MediaSize `json:"supported_media_sizes,omitempty"`
	// SupportsDuplex indicates if the printer supports duplex printing.
	SupportsDuplex bool `json:"supports_duplex"`
	// SupportsColor indicates if the printer supports color printing.
	SupportsColor bool `json:"supports_color"`
}

// PrinterState represents the state of a printer.
type PrinterState int

const (
	PrinterStateIdle       PrinterState = 3
	PrinterStateProcessing PrinterState = 4
	PrinterStateStopped    PrinterState = 5
)

// String returns a human-readable state string.
func (s PrinterState) String() string {
	switch s {
	case PrinterStateIdle:
		return "idle"
	case PrinterStateProcessing:
		return "processing"
	case PrinterStateStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// PrintOptions contains options for a print job.
type PrintOptions struct {
	// Copies is the number of copies to print.
	Copies int
	// Media is the paper size.
	Media MediaSize
	// Quality is the print quality.
	Quality PrintQuality
	// Orientation is the page orientation.
	Orientation Orientation
	// Sides is the duplex setting.
	Sides Sides
	// ColorMode is the color mode ("color" or "monochrome").
	ColorMode string
	// PageRanges specifies which pages to print (e.g., "1-5,8,11-13").
	PageRanges string
	// JobName is the name of the print job.
	JobName string
}

// DefaultPrintOptions returns default print options.
func DefaultPrintOptions() PrintOptions {
	return PrintOptions{
		Copies:      1,
		Media:       MediaA4,
		Quality:     QualityNormal,
		Orientation: OrientationPortrait,
		Sides:       SidesOneSided,
		ColorMode:   "color",
	}
}

// PrintJobStatus represents the status of a print job.
type PrintJobStatus struct {
	// JobID is the unique job identifier.
	JobID int `json:"job_id"`
	// State is the current job state.
	State JobState `json:"state"`
	// StateMessage is a human-readable state message.
	StateMessage string `json:"state_message,omitempty"`
	// PrinterName is the name of the printer.
	PrinterName string `json:"printer_name"`
	// JobName is the name of the print job.
	JobName string `json:"job_name,omitempty"`
	// CreatedAt is when the job was created.
	CreatedAt time.Time `json:"created_at"`
	// CompletedAt is when the job completed (if applicable).
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	// PagesPrinted is the number of pages printed.
	PagesPrinted int `json:"pages_printed"`
	// TotalPages is the total number of pages.
	TotalPages int `json:"total_pages"`
}

// JobState represents the state of a print job.
type JobState int

const (
	JobStatePending    JobState = 3
	JobStateHeld       JobState = 4
	JobStateProcessing JobState = 5
	JobStateStopped    JobState = 6
	JobStateCanceled   JobState = 7
	JobStateAborted    JobState = 8
	JobStateCompleted  JobState = 9
)

// String returns a human-readable job state string.
func (s JobState) String() string {
	switch s {
	case JobStatePending:
		return "pending"
	case JobStateHeld:
		return "held"
	case JobStateProcessing:
		return "processing"
	case JobStateStopped:
		return "stopped"
	case JobStateCanceled:
		return "canceled"
	case JobStateAborted:
		return "aborted"
	case JobStateCompleted:
		return "completed"
	default:
		return "unknown"
	}
}

// Backend is the interface for printing backends.
type Backend interface {
	// ListPrinters returns a list of available printers.
	ListPrinters(ctx context.Context) ([]PrinterInfo, error)
	// GetPrinter returns information about a specific printer.
	GetPrinter(ctx context.Context, name string) (*PrinterInfo, error)
	// Print sends a document to the printer.
	Print(ctx context.Context, printerName string, data []byte, opts PrintOptions) (int, error)
	// GetJobStatus returns the status of a print job.
	GetJobStatus(ctx context.Context, jobID int) (*PrintJobStatus, error)
	// CancelJob cancels a print job.
	CancelJob(ctx context.Context, jobID int) error
	// Protocol returns the protocol used by this backend.
	Protocol() Protocol
	// IsAvailable checks if the backend is available.
	IsAvailable() bool
	// Close releases resources held by the backend.
	Close() error
}

// Manager manages printers and print jobs.
type Manager struct {
	backend        Backend
	defaultPrinter string
	logger         *zap.Logger
	mu             sync.RWMutex
	jobs           map[int]*PrintJobStatus
}

// ManagerConfig contains configuration for the printer manager.
type ManagerConfig struct {
	// Protocol is the printing protocol to use.
	Protocol Protocol
	// IPPServer is the IPP server URL (for IPP protocol).
	IPPServer string
	// CUPSServer is the CUPS server address (for CUPS protocol).
	CUPSServer string
	// DefaultPrinter is the default printer name.
	DefaultPrinter string
	// Logger is the logger to use.
	Logger *zap.Logger
}

// NewManager creates a new printer manager.
func NewManager(cfg ManagerConfig) (*Manager, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	m := &Manager{
		defaultPrinter: cfg.DefaultPrinter,
		logger:         cfg.Logger,
		jobs:           make(map[int]*PrintJobStatus),
	}

	// Create backend based on protocol
	var backend Backend
	var err error

	switch cfg.Protocol {
	case ProtocolIPP:
		backend, err = NewIPPBackend(IPPConfig{
			ServerURL: cfg.IPPServer,
			Logger:    cfg.Logger,
		})
	case ProtocolCUPS:
		backend, err = NewCUPSBackend(CUPSConfig{
			ServerAddress: cfg.CUPSServer,
			Logger:        cfg.Logger,
		})
	case ProtocolRaw:
		return nil, fmt.Errorf("%w: raw protocol not yet implemented", ErrProtocolNotSupported)
	case ProtocolWindows:
		return nil, fmt.Errorf("%w: windows protocol not yet implemented", ErrProtocolNotSupported)
	default:
		return nil, fmt.Errorf("%w: %s", ErrProtocolNotSupported, cfg.Protocol)
	}

	if err != nil {
		return nil, fmt.Errorf("creating backend: %w", err)
	}

	m.backend = backend
	return m, nil
}

// ListPrinters returns a list of available printers.
func (m *Manager) ListPrinters(ctx context.Context) ([]PrinterInfo, error) {
	printers, err := m.backend.ListPrinters(ctx)
	if err != nil {
		return nil, err
	}

	// Mark default printer
	for i := range printers {
		if printers[i].Name == m.defaultPrinter {
			printers[i].IsDefault = true
		}
	}

	return printers, nil
}

// GetPrinter returns information about a specific printer.
func (m *Manager) GetPrinter(ctx context.Context, name string) (*PrinterInfo, error) {
	printer, err := m.backend.GetPrinter(ctx, name)
	if err != nil {
		return nil, err
	}

	if printer.Name == m.defaultPrinter {
		printer.IsDefault = true
	}

	return printer, nil
}

// GetDefaultPrinter returns the default printer.
func (m *Manager) GetDefaultPrinter(ctx context.Context) (*PrinterInfo, error) {
	if m.defaultPrinter == "" {
		// Try to find system default
		printers, err := m.backend.ListPrinters(ctx)
		if err != nil {
			return nil, err
		}

		for _, p := range printers {
			if p.IsDefault {
				return &p, nil
			}
		}

		return nil, ErrNoDefaultPrinter
	}

	return m.GetPrinter(ctx, m.defaultPrinter)
}

// SetDefaultPrinter sets the default printer.
func (m *Manager) SetDefaultPrinter(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultPrinter = name
}

// Print sends a document to the specified printer.
func (m *Manager) Print(ctx context.Context, printerName string, data []byte, opts PrintOptions) (int, error) {
	if len(data) == 0 {
		return 0, fmt.Errorf("%w: document data is empty", ErrInvalidInput)
	}

	// Use default printer if not specified
	if printerName == "" {
		if m.defaultPrinter == "" {
			return 0, ErrNoDefaultPrinter
		}
		printerName = m.defaultPrinter
	}

	// Validate printer exists
	printer, err := m.backend.GetPrinter(ctx, printerName)
	if err != nil {
		return 0, err
	}

	m.logger.Info("Sending print job",
		zap.String("printer", printerName),
		zap.String("job_name", opts.JobName),
		zap.Int("copies", opts.Copies),
		zap.Int("data_size", len(data)))

	// Send to printer
	jobID, err := m.backend.Print(ctx, printer.Name, data, opts)
	if err != nil {
		m.logger.Error("Print failed",
			zap.String("printer", printerName),
			zap.Error(err))
		return 0, err
	}

	// Track job
	m.mu.Lock()
	m.jobs[jobID] = &PrintJobStatus{
		JobID:       jobID,
		State:       JobStatePending,
		PrinterName: printerName,
		JobName:     opts.JobName,
		CreatedAt:   time.Now(),
	}
	m.mu.Unlock()

	m.logger.Info("Print job submitted",
		zap.Int("job_id", jobID),
		zap.String("printer", printerName))

	return jobID, nil
}

// PrintToDefault sends a document to the default printer.
func (m *Manager) PrintToDefault(ctx context.Context, data []byte, opts PrintOptions) (int, error) {
	return m.Print(ctx, "", data, opts)
}

// GetJobStatus returns the status of a print job.
func (m *Manager) GetJobStatus(ctx context.Context, jobID int) (*PrintJobStatus, error) {
	// Try to get from backend first
	status, err := m.backend.GetJobStatus(ctx, jobID)
	if err == nil {
		// Update local cache
		m.mu.Lock()
		m.jobs[jobID] = status
		m.mu.Unlock()
		return status, nil
	}

	// Fall back to local cache
	m.mu.RLock()
	defer m.mu.RUnlock()

	if status, ok := m.jobs[jobID]; ok {
		return status, nil
	}

	return nil, fmt.Errorf("%w: job %d not found", ErrPrintFailed, jobID)
}

// CancelJob cancels a print job.
func (m *Manager) CancelJob(ctx context.Context, jobID int) error {
	if err := m.backend.CancelJob(ctx, jobID); err != nil {
		return err
	}

	// Update local cache
	m.mu.Lock()
	if status, ok := m.jobs[jobID]; ok {
		status.State = JobStateCanceled
		now := time.Now()
		status.CompletedAt = &now
	}
	m.mu.Unlock()

	return nil
}

// IsAvailable returns true if the printing backend is available.
func (m *Manager) IsAvailable() bool {
	return m.backend != nil && m.backend.IsAvailable()
}

// Protocol returns the protocol used by the manager.
func (m *Manager) Protocol() Protocol {
	if m.backend == nil {
		return ""
	}
	return m.backend.Protocol()
}

// Close releases resources held by the manager.
func (m *Manager) Close() error {
	if m.backend != nil {
		return m.backend.Close()
	}
	return nil
}
