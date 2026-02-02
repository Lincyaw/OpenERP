// Package renderer provides PDF rendering capabilities.
package renderer

import (
	"context"
	"errors"
	"fmt"
)

// Errors returned by the renderer package.
var (
	// ErrRenderFailed is returned when PDF rendering fails.
	ErrRenderFailed = errors.New("renderer: rendering failed")
	// ErrInvalidInput is returned when input is invalid.
	ErrInvalidInput = errors.New("renderer: invalid input")
	// ErrEngineNotAvailable is returned when the rendering engine is not available.
	ErrEngineNotAvailable = errors.New("renderer: engine not available")
	// ErrTimeout is returned when rendering times out.
	ErrTimeout = errors.New("renderer: operation timed out")
)

// EngineType represents the type of rendering engine.
type EngineType string

const (
	// EngineChromedp uses headless Chrome via chromedp.
	EngineChromedp EngineType = "chromedp"
	// EngineWkhtmltopdf uses wkhtmltopdf command-line tool.
	EngineWkhtmltopdf EngineType = "wkhtmltopdf"
)

// PageSize represents standard page sizes.
type PageSize string

const (
	PageSizeA4     PageSize = "A4"
	PageSizeA3     PageSize = "A3"
	PageSizeLetter PageSize = "Letter"
	PageSizeLegal  PageSize = "Legal"
)

// Orientation represents page orientation.
type Orientation string

const (
	OrientationPortrait  Orientation = "portrait"
	OrientationLandscape Orientation = "landscape"
)

// RenderOptions contains options for PDF rendering.
type RenderOptions struct {
	// PageSize is the page size (default: A4).
	PageSize PageSize
	// Orientation is the page orientation (default: portrait).
	Orientation Orientation
	// MarginTop in millimeters (default: 10).
	MarginTop float64
	// MarginBottom in millimeters (default: 10).
	MarginBottom float64
	// MarginLeft in millimeters (default: 10).
	MarginLeft float64
	// MarginRight in millimeters (default: 10).
	MarginRight float64
	// Scale is the scale factor (default: 1.0).
	Scale float64
	// PrintBackground determines if background graphics should be printed.
	PrintBackground bool
	// PreferCSSPageSize uses CSS-defined page size if available.
	PreferCSSPageSize bool
	// HeaderTemplate is the HTML template for the header.
	HeaderTemplate string
	// FooterTemplate is the HTML template for the footer.
	FooterTemplate string
	// DisplayHeaderFooter determines if header/footer should be displayed.
	DisplayHeaderFooter bool
}

// DefaultRenderOptions returns default rendering options.
func DefaultRenderOptions() RenderOptions {
	return RenderOptions{
		PageSize:          PageSizeA4,
		Orientation:       OrientationPortrait,
		MarginTop:         10,
		MarginBottom:      10,
		MarginLeft:        10,
		MarginRight:       10,
		Scale:             1.0,
		PrintBackground:   true,
		PreferCSSPageSize: false,
	}
}

// Renderer is the interface for PDF rendering engines.
type Renderer interface {
	// RenderHTML renders HTML content to PDF.
	RenderHTML(ctx context.Context, html string, opts RenderOptions) ([]byte, error)
	// RenderURL renders a URL to PDF.
	RenderURL(ctx context.Context, url string, opts RenderOptions) ([]byte, error)
	// IsAvailable checks if the rendering engine is available.
	IsAvailable() bool
	// Name returns the name of the rendering engine.
	Name() string
	// Close releases any resources held by the renderer.
	Close() error
}

// Manager manages multiple rendering engines and provides fallback support.
type Manager struct {
	primary   Renderer
	fallback  Renderer
	available []Renderer
}

// NewManager creates a new renderer manager with the specified primary engine.
func NewManager(primaryEngine EngineType, opts ...ManagerOption) (*Manager, error) {
	m := &Manager{
		available: make([]Renderer, 0),
	}

	// Apply options
	config := &managerConfig{}
	for _, opt := range opts {
		opt(config)
	}

	// Create primary renderer
	primary, err := createRenderer(primaryEngine, config)
	if err != nil {
		return nil, fmt.Errorf("creating primary renderer: %w", err)
	}
	m.primary = primary
	m.available = append(m.available, primary)

	// Create fallback renderer if different from primary
	if config.fallbackEngine != "" && config.fallbackEngine != primaryEngine {
		fallback, err := createRenderer(config.fallbackEngine, config)
		if err != nil {
			// Fallback creation failure is not fatal
			// Just log and continue without fallback
		} else {
			m.fallback = fallback
			m.available = append(m.available, fallback)
		}
	}

	return m, nil
}

// managerConfig holds configuration for the manager.
type managerConfig struct {
	fallbackEngine  EngineType
	chromedpOpts    *ChromedpOptions
	wkhtmltopdfOpts *WkhtmltopdfOptions
}

// ManagerOption is a function that configures the manager.
type ManagerOption func(*managerConfig)

// WithFallback sets the fallback rendering engine.
func WithFallback(engine EngineType) ManagerOption {
	return func(c *managerConfig) {
		c.fallbackEngine = engine
	}
}

// WithChromedpOptions sets options for the chromedp renderer.
func WithChromedpOptions(opts *ChromedpOptions) ManagerOption {
	return func(c *managerConfig) {
		c.chromedpOpts = opts
	}
}

// WithWkhtmltopdfOptions sets options for the wkhtmltopdf renderer.
func WithWkhtmltopdfOptions(opts *WkhtmltopdfOptions) ManagerOption {
	return func(c *managerConfig) {
		c.wkhtmltopdfOpts = opts
	}
}

// createRenderer creates a renderer of the specified type.
func createRenderer(engine EngineType, config *managerConfig) (Renderer, error) {
	switch engine {
	case EngineChromedp:
		opts := DefaultChromedpOptions()
		if config.chromedpOpts != nil {
			opts = *config.chromedpOpts
		}
		return NewChromedpRenderer(opts)
	case EngineWkhtmltopdf:
		opts := DefaultWkhtmltopdfOptions()
		if config.wkhtmltopdfOpts != nil {
			opts = *config.wkhtmltopdfOpts
		}
		return NewWkhtmltopdfRenderer(opts)
	default:
		return nil, fmt.Errorf("%w: unknown engine type: %s", ErrInvalidInput, engine)
	}
}

// RenderHTML renders HTML content to PDF using the primary renderer with fallback.
func (m *Manager) RenderHTML(ctx context.Context, html string, opts RenderOptions) ([]byte, error) {
	if html == "" {
		return nil, fmt.Errorf("%w: HTML content is empty", ErrInvalidInput)
	}

	// Try primary renderer
	if m.primary != nil && m.primary.IsAvailable() {
		pdf, err := m.primary.RenderHTML(ctx, html, opts)
		if err == nil {
			return pdf, nil
		}
		// If primary fails and we have a fallback, try it
		if m.fallback != nil && m.fallback.IsAvailable() {
			return m.fallback.RenderHTML(ctx, html, opts)
		}
		return nil, err
	}

	// Try fallback if primary is not available
	if m.fallback != nil && m.fallback.IsAvailable() {
		return m.fallback.RenderHTML(ctx, html, opts)
	}

	return nil, ErrEngineNotAvailable
}

// RenderURL renders a URL to PDF using the primary renderer with fallback.
func (m *Manager) RenderURL(ctx context.Context, url string, opts RenderOptions) ([]byte, error) {
	if url == "" {
		return nil, fmt.Errorf("%w: URL is empty", ErrInvalidInput)
	}

	// Try primary renderer
	if m.primary != nil && m.primary.IsAvailable() {
		pdf, err := m.primary.RenderURL(ctx, url, opts)
		if err == nil {
			return pdf, nil
		}
		// If primary fails and we have a fallback, try it
		if m.fallback != nil && m.fallback.IsAvailable() {
			return m.fallback.RenderURL(ctx, url, opts)
		}
		return nil, err
	}

	// Try fallback if primary is not available
	if m.fallback != nil && m.fallback.IsAvailable() {
		return m.fallback.RenderURL(ctx, url, opts)
	}

	return nil, ErrEngineNotAvailable
}

// IsAvailable returns true if at least one renderer is available.
func (m *Manager) IsAvailable() bool {
	for _, r := range m.available {
		if r.IsAvailable() {
			return true
		}
	}
	return false
}

// AvailableEngines returns a list of available rendering engines.
func (m *Manager) AvailableEngines() []string {
	engines := make([]string, 0)
	for _, r := range m.available {
		if r.IsAvailable() {
			engines = append(engines, r.Name())
		}
	}
	return engines
}

// Close releases resources held by all renderers.
func (m *Manager) Close() error {
	var errs []error
	for _, r := range m.available {
		if err := r.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("closing renderers: %v", errs)
	}
	return nil
}
