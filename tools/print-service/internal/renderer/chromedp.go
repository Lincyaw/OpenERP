// Package renderer provides PDF rendering capabilities.
package renderer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// ChromedpOptions contains options for the chromedp renderer.
type ChromedpOptions struct {
	// Timeout is the maximum time to wait for rendering.
	Timeout time.Duration
	// Headless determines if Chrome runs in headless mode.
	Headless bool
	// DisableGPU disables GPU hardware acceleration.
	DisableGPU bool
	// NoSandbox disables the Chrome sandbox (required in some environments).
	NoSandbox bool
	// ChromePath is the path to the Chrome executable.
	// If empty, chromedp will try to find Chrome automatically.
	ChromePath string
	// UserDataDir is the directory for Chrome user data.
	// If empty, a temporary directory is used.
	UserDataDir string
}

// DefaultChromedpOptions returns default chromedp options.
func DefaultChromedpOptions() ChromedpOptions {
	return ChromedpOptions{
		Timeout:    60 * time.Second,
		Headless:   true,
		DisableGPU: true,
		NoSandbox:  false,
	}
}

// ChromedpRenderer renders PDFs using headless Chrome via chromedp.
type ChromedpRenderer struct {
	opts       ChromedpOptions
	allocCtx   context.Context
	cancelFunc context.CancelFunc
	mu         sync.Mutex
	available  bool
}

// NewChromedpRenderer creates a new chromedp renderer.
func NewChromedpRenderer(opts ChromedpOptions) (*ChromedpRenderer, error) {
	r := &ChromedpRenderer{
		opts:      opts,
		available: false,
	}

	// Check if Chrome is available
	if err := r.checkAvailability(); err != nil {
		return r, nil // Return renderer but mark as unavailable
	}

	r.available = true
	return r, nil
}

// checkAvailability checks if Chrome is available.
func (r *ChromedpRenderer) checkAvailability() error {
	// Try to find Chrome executable
	chromePath := r.opts.ChromePath
	if chromePath == "" {
		// Try common Chrome paths
		paths := []string{
			"google-chrome",
			"google-chrome-stable",
			"chromium",
			"chromium-browser",
			"/usr/bin/google-chrome",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		}
		for _, p := range paths {
			if _, err := exec.LookPath(p); err == nil {
				chromePath = p
				break
			}
		}
	}

	if chromePath == "" {
		return fmt.Errorf("Chrome executable not found")
	}

	return nil
}

// initAllocator initializes the Chrome allocator if not already done.
func (r *ChromedpRenderer) initAllocator() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.allocCtx != nil {
		return nil
	}

	// Build Chrome options
	chromeOpts := []chromedp.ExecAllocatorOption{
		chromedp.Headless,
		chromedp.DisableGPU,
	}

	if r.opts.NoSandbox {
		chromeOpts = append(chromeOpts, chromedp.NoSandbox)
	}

	if r.opts.ChromePath != "" {
		chromeOpts = append(chromeOpts, chromedp.ExecPath(r.opts.ChromePath))
	}

	if r.opts.UserDataDir != "" {
		chromeOpts = append(chromeOpts, chromedp.UserDataDir(r.opts.UserDataDir))
	}

	// Create allocator context
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), chromeOpts...)
	r.allocCtx = allocCtx
	r.cancelFunc = cancel

	return nil
}

// RenderHTML renders HTML content to PDF.
func (r *ChromedpRenderer) RenderHTML(ctx context.Context, html string, opts RenderOptions) ([]byte, error) {
	if !r.available {
		return nil, ErrEngineNotAvailable
	}

	if err := r.initAllocator(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRenderFailed, err)
	}

	// Create a temporary file for the HTML
	tmpFile, err := os.CreateTemp("", "render-*.html")
	if err != nil {
		return nil, fmt.Errorf("%w: creating temp file: %v", ErrRenderFailed, err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(html); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("%w: writing temp file: %v", ErrRenderFailed, err)
	}
	tmpFile.Close()

	// Render the file URL
	fileURL := "file://" + tmpFile.Name()
	return r.RenderURL(ctx, fileURL, opts)
}

// RenderURL renders a URL to PDF.
func (r *ChromedpRenderer) RenderURL(ctx context.Context, url string, opts RenderOptions) ([]byte, error) {
	if !r.available {
		return nil, ErrEngineNotAvailable
	}

	if err := r.initAllocator(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRenderFailed, err)
	}

	// Apply timeout
	timeout := r.opts.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create browser context
	browserCtx, browserCancel := chromedp.NewContext(r.allocCtx)
	defer browserCancel()

	// Build print options
	printOpts := r.buildPrintOptions(opts)

	// Render PDF
	var pdfData []byte
	err := chromedp.Run(browserCtx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdfData, _, err = page.PrintToPDF().
				WithPaperWidth(printOpts.paperWidth).
				WithPaperHeight(printOpts.paperHeight).
				WithMarginTop(printOpts.marginTop).
				WithMarginBottom(printOpts.marginBottom).
				WithMarginLeft(printOpts.marginLeft).
				WithMarginRight(printOpts.marginRight).
				WithScale(printOpts.scale).
				WithPrintBackground(printOpts.printBackground).
				WithLandscape(printOpts.landscape).
				WithPreferCSSPageSize(printOpts.preferCSSPageSize).
				WithDisplayHeaderFooter(printOpts.displayHeaderFooter).
				WithHeaderTemplate(printOpts.headerTemplate).
				WithFooterTemplate(printOpts.footerTemplate).
				Do(ctx)
			return err
		}),
	)

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, ErrTimeout
		}
		return nil, fmt.Errorf("%w: %v", ErrRenderFailed, err)
	}

	return pdfData, nil
}

// printOptions holds the converted print options for chromedp.
type printOptions struct {
	paperWidth          float64
	paperHeight         float64
	marginTop           float64
	marginBottom        float64
	marginLeft          float64
	marginRight         float64
	scale               float64
	printBackground     bool
	landscape           bool
	preferCSSPageSize   bool
	displayHeaderFooter bool
	headerTemplate      string
	footerTemplate      string
}

// buildPrintOptions converts RenderOptions to chromedp print options.
func (r *ChromedpRenderer) buildPrintOptions(opts RenderOptions) printOptions {
	// Convert page size to inches (chromedp uses inches)
	var paperWidth, paperHeight float64
	switch opts.PageSize {
	case PageSizeA4:
		paperWidth, paperHeight = 8.27, 11.69
	case PageSizeA3:
		paperWidth, paperHeight = 11.69, 16.54
	case PageSizeLetter:
		paperWidth, paperHeight = 8.5, 11
	case PageSizeLegal:
		paperWidth, paperHeight = 8.5, 14
	default:
		paperWidth, paperHeight = 8.27, 11.69 // Default to A4
	}

	// Swap dimensions for landscape
	landscape := opts.Orientation == OrientationLandscape

	// Convert margins from mm to inches
	marginTop := opts.MarginTop / 25.4
	marginBottom := opts.MarginBottom / 25.4
	marginLeft := opts.MarginLeft / 25.4
	marginRight := opts.MarginRight / 25.4

	scale := opts.Scale
	if scale <= 0 {
		scale = 1.0
	}

	return printOptions{
		paperWidth:          paperWidth,
		paperHeight:         paperHeight,
		marginTop:           marginTop,
		marginBottom:        marginBottom,
		marginLeft:          marginLeft,
		marginRight:         marginRight,
		scale:               scale,
		printBackground:     opts.PrintBackground,
		landscape:           landscape,
		preferCSSPageSize:   opts.PreferCSSPageSize,
		displayHeaderFooter: opts.DisplayHeaderFooter,
		headerTemplate:      opts.HeaderTemplate,
		footerTemplate:      opts.FooterTemplate,
	}
}

// IsAvailable returns true if Chrome is available.
func (r *ChromedpRenderer) IsAvailable() bool {
	return r.available
}

// Name returns the name of the renderer.
func (r *ChromedpRenderer) Name() string {
	return string(EngineChromedp)
}

// Close releases resources held by the renderer.
func (r *ChromedpRenderer) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cancelFunc != nil {
		r.cancelFunc()
		r.cancelFunc = nil
		r.allocCtx = nil
	}
	return nil
}
