package printing

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/erp/backend/internal/domain/printing"
	"go.uber.org/zap"
)

const (
	defaultChromeTimeout = 30 * time.Second
	defaultScale         = 1.0
)

// ChromedpConfig contains configuration for the chromedp renderer
type ChromedpConfig struct {
	// DefaultTimeout for rendering operations
	DefaultTimeout time.Duration
	// RemoteURL is the URL of a remote Chrome/Chromium instance (optional)
	// If empty, chromedp will launch a new browser instance
	RemoteURL string
	// Headless mode (default: true)
	Headless bool
	// DisableGPU disables GPU hardware acceleration (default: true for server environments)
	DisableGPU bool
	// NoSandbox runs Chrome without sandbox (required for Docker/root)
	NoSandbox bool
	// Scale for rendering (default: 1.0)
	Scale float64
	// PrintBackground prints background graphics (default: true)
	PrintBackground bool
	// Logger for debug output
	Logger *zap.Logger
}

// ChromedpRenderer renders HTML to PDF using Chrome DevTools Protocol
type ChromedpRenderer struct {
	config      *ChromedpConfig
	logger      *zap.Logger
	allocCtx    context.Context
	allocCancel context.CancelFunc
}

// NewChromedpRenderer creates a new chromedp-based PDF renderer
func NewChromedpRenderer(config *ChromedpConfig) (*ChromedpRenderer, error) {
	if config == nil {
		config = &ChromedpConfig{}
	}

	// Set defaults
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = defaultChromeTimeout
	}
	if config.Scale == 0 {
		config.Scale = defaultScale
	}
	// Default to headless and disable GPU for server environments
	if !config.Headless {
		config.Headless = true
	}
	if !config.DisableGPU {
		config.DisableGPU = true
	}

	logger := config.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	renderer := &ChromedpRenderer{
		config: config,
		logger: logger,
	}

	// Create allocator context
	if err := renderer.initAllocator(); err != nil {
		return nil, err
	}

	return renderer, nil
}

// initAllocator initializes the Chrome allocator
func (r *ChromedpRenderer) initAllocator() error {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", r.config.Headless),
		chromedp.Flag("disable-gpu", r.config.DisableGPU),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-dev-shm-usage", true), // Important for Docker
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("disable-translate", true),
		// Font rendering
		chromedp.Flag("font-render-hinting", "none"),
	)

	if r.config.NoSandbox {
		opts = append(opts, chromedp.Flag("no-sandbox", true))
	}

	if r.config.RemoteURL != "" {
		r.allocCtx, r.allocCancel = chromedp.NewRemoteAllocator(context.Background(), r.config.RemoteURL)
	} else {
		r.allocCtx, r.allocCancel = chromedp.NewExecAllocator(context.Background(), opts...)
	}

	return nil
}

// Render converts HTML content to PDF
func (r *ChromedpRenderer) Render(ctx context.Context, req *RenderRequest) (*RenderResult, error) {
	if req == nil {
		return nil, NewRenderError(ErrCodeInvalidHTML, "render request is nil", nil)
	}
	if strings.TrimSpace(req.HTML) == "" {
		return nil, NewRenderError(ErrCodeInvalidHTML, "HTML content is empty", nil)
	}
	if !req.PaperSize.IsValid() {
		return nil, NewRenderError(ErrCodeInvalidPaperSize, "invalid paper size: "+string(req.PaperSize), nil)
	}

	startTime := time.Now()

	// Determine timeout
	timeout := req.Timeout
	if timeout == 0 {
		timeout = r.config.DefaultTimeout
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create browser context
	browserCtx, browserCancel := chromedp.NewContext(r.allocCtx,
		chromedp.WithLogf(func(format string, args ...interface{}) {
			r.logger.Debug(fmt.Sprintf(format, args...))
		}),
	)
	defer browserCancel()

	// Build the complete HTML with header/footer if needed
	html := r.buildCompleteHTML(req)

	// Build print parameters
	printParams := r.buildPrintParams(req)

	var pdfData []byte

	// Execute rendering
	err := chromedp.Run(browserCtx,
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			frameTree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}
			return page.SetDocumentContent(frameTree.Frame.ID, html).Do(ctx)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			data, _, err := page.PrintToPDF().
				WithPrintBackground(printParams.printBackground).
				WithPaperWidth(printParams.paperWidth).
				WithPaperHeight(printParams.paperHeight).
				WithMarginTop(printParams.marginTop).
				WithMarginRight(printParams.marginRight).
				WithMarginBottom(printParams.marginBottom).
				WithMarginLeft(printParams.marginLeft).
				WithScale(printParams.scale).
				WithLandscape(printParams.landscape).
				WithDisplayHeaderFooter(printParams.displayHeaderFooter).
				WithHeaderTemplate(printParams.headerTemplate).
				WithFooterTemplate(printParams.footerTemplate).
				Do(ctx)
			if err != nil {
				return err
			}
			pdfData = data
			return nil
		}),
	)

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, NewRenderError(ErrCodeRenderTimeout,
				fmt.Sprintf("PDF rendering timed out after %v", timeout), err)
		}
		if ctx.Err() == context.Canceled {
			return nil, NewRenderError(ErrCodeRenderTimeout, "PDF rendering was cancelled", err)
		}

		r.logger.Error("chromedp rendering failed", zap.Error(err))
		return nil, NewRenderError(ErrCodeRenderFailed, "chromedp execution failed: "+err.Error(), err)
	}

	if len(pdfData) == 0 {
		return nil, NewRenderError(ErrCodeRenderFailed, "generated PDF is empty", nil)
	}

	// Count pages
	pageCount := estimatePageCount(pdfData)

	renderDuration := time.Since(startTime)

	r.logger.Info("PDF rendered successfully",
		zap.Int("bytes", len(pdfData)),
		zap.Int("pages", pageCount),
		zap.Duration("duration", renderDuration))

	return &RenderResult{
		PDFData:        pdfData,
		PageCount:      pageCount,
		RenderDuration: renderDuration,
	}, nil
}

// printParams holds the parameters for PDF printing
type printParams struct {
	paperWidth          float64
	paperHeight         float64
	marginTop           float64
	marginRight         float64
	marginBottom        float64
	marginLeft          float64
	scale               float64
	landscape           bool
	printBackground     bool
	displayHeaderFooter bool
	headerTemplate      string
	footerTemplate      string
}

// buildPrintParams constructs the print parameters from the render request
func (r *ChromedpRenderer) buildPrintParams(req *RenderRequest) *printParams {
	params := &printParams{
		scale:           r.config.Scale,
		printBackground: true,
	}

	// Paper size in inches (Chrome uses inches)
	width, height := req.PaperSize.Dimensions()
	params.paperWidth = mmToInches(float64(width))
	params.paperHeight = mmToInches(float64(height))

	// For receipt paper, use a very tall page to avoid pagination issues
	if req.PaperSize.IsReceipt() || req.PaperSize.IsContinuous() {
		// Use a tall page height for continuous paper
		// Chrome will use the actual content height
		params.paperHeight = mmToInches(3000) // ~3 meters, enough for most receipts
	}

	// Orientation
	params.landscape = req.Orientation == printing.OrientationLandscape

	// Margins in inches
	params.marginTop = mmToInches(float64(req.Margins.Top))
	params.marginRight = mmToInches(float64(req.Margins.Right))
	params.marginBottom = mmToInches(float64(req.Margins.Bottom))
	params.marginLeft = mmToInches(float64(req.Margins.Left))

	// Header and footer
	if req.HeaderHTML != "" || req.FooterHTML != "" {
		params.displayHeaderFooter = true
		params.headerTemplate = req.HeaderHTML
		params.footerTemplate = req.FooterHTML

		// If header/footer is provided, ensure minimum margins for them
		if params.headerTemplate != "" && params.marginTop < mmToInches(10) {
			params.marginTop = mmToInches(10)
		}
		if params.footerTemplate != "" && params.marginBottom < mmToInches(10) {
			params.marginBottom = mmToInches(10)
		}
	}

	return params
}

// buildCompleteHTML builds the complete HTML document
func (r *ChromedpRenderer) buildCompleteHTML(req *RenderRequest) string {
	// If the HTML already has DOCTYPE and html tags, return as-is
	if strings.Contains(strings.ToLower(req.HTML), "<!doctype") ||
		strings.Contains(strings.ToLower(req.HTML), "<html") {
		return req.HTML
	}

	// Wrap the HTML in a complete document
	var buf bytes.Buffer
	buf.WriteString("<!DOCTYPE html><html><head>")
	buf.WriteString("<meta charset=\"UTF-8\">")
	buf.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">")
	if req.Title != "" {
		buf.WriteString("<title>")
		buf.WriteString(req.Title)
		buf.WriteString("</title>")
	}
	buf.WriteString("</head><body>")
	buf.WriteString(req.HTML)
	buf.WriteString("</body></html>")

	return buf.String()
}

// Close releases resources held by the renderer
func (r *ChromedpRenderer) Close() error {
	if r.allocCancel != nil {
		r.allocCancel()
	}
	return nil
}

// mmToInches converts millimeters to inches
func mmToInches(mm float64) float64 {
	return mm / 25.4
}

// inchesToMM converts inches to millimeters
func inchesToMM(inches float64) float64 {
	return inches * 25.4
}

// dataURLToBytes converts a data URL to bytes
func dataURLToBytes(dataURL string) ([]byte, error) {
	// Format: data:image/png;base64,iVBORw0KGgo...
	idx := strings.Index(dataURL, ",")
	if idx == -1 {
		return nil, fmt.Errorf("invalid data URL format")
	}
	return base64.StdEncoding.DecodeString(dataURL[idx+1:])
}

// Ensure ChromedpRenderer implements PDFRenderer
var _ PDFRenderer = (*ChromedpRenderer)(nil)
