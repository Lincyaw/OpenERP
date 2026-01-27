package printing

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/printing"
	"go.uber.org/zap"
)

const (
	defaultBinaryPath   = "wkhtmltopdf"
	defaultTimeout      = 30 * time.Second
	defaultDPI          = 96
	defaultImageQuality = 94
)

// WkhtmltopdfConfig contains configuration for the wkhtmltopdf renderer
type WkhtmltopdfConfig struct {
	// BinaryPath is the path to the wkhtmltopdf binary
	// If empty, will search in PATH
	BinaryPath string
	// DefaultTimeout for rendering operations
	DefaultTimeout time.Duration
	// TempDir for temporary files during rendering
	TempDir string
	// EnableJavaScript enables JavaScript execution (default: false for security)
	EnableJavaScript bool
	// JavaScriptDelay in milliseconds to wait for JS to complete
	JavaScriptDelay int
	// DPI for rendering (default: 96)
	DPI int
	// ImageQuality (0-100, default: 94)
	ImageQuality int
	// Logger for debug output
	Logger *zap.Logger
}

// tempFiles holds paths to temporary files created during rendering
type tempFiles struct {
	header string
	footer string
}

// cleanup removes all temporary files
func (t *tempFiles) cleanup() {
	if t.header != "" {
		os.Remove(t.header)
	}
	if t.footer != "" {
		os.Remove(t.footer)
	}
}

// WkhtmltopdfRenderer renders HTML to PDF using wkhtmltopdf command-line tool
type WkhtmltopdfRenderer struct {
	config *WkhtmltopdfConfig
	logger *zap.Logger
}

// NewWkhtmltopdfRenderer creates a new wkhtmltopdf-based PDF renderer
func NewWkhtmltopdfRenderer(config *WkhtmltopdfConfig) (*WkhtmltopdfRenderer, error) {
	if config == nil {
		config = &WkhtmltopdfConfig{}
	}

	// Set defaults
	if config.BinaryPath == "" {
		config.BinaryPath = defaultBinaryPath
	}
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = defaultTimeout
	}
	if config.TempDir == "" {
		config.TempDir = os.TempDir()
	}
	if config.DPI == 0 {
		config.DPI = defaultDPI
	}
	if config.ImageQuality == 0 {
		config.ImageQuality = defaultImageQuality
	}

	// Verify wkhtmltopdf is available
	binaryPath, err := resolveBinaryPath(config.BinaryPath)
	if err != nil {
		return nil, NewRenderError(ErrCodeBinaryNotFound,
			fmt.Sprintf("wkhtmltopdf binary not found: %s", config.BinaryPath), err)
	}
	config.BinaryPath = binaryPath

	logger := config.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &WkhtmltopdfRenderer{
		config: config,
		logger: logger,
	}, nil
}

// resolveBinaryPath finds the full path to the binary
func resolveBinaryPath(path string) (string, error) {
	// If it's an absolute path, check if it exists
	if filepath.IsAbs(path) {
		if _, err := os.Stat(path); err != nil {
			return "", err
		}
		return path, nil
	}

	// Search in PATH
	return exec.LookPath(path)
}

// Render converts HTML content to PDF
func (r *WkhtmltopdfRenderer) Render(ctx context.Context, req *RenderRequest) (*RenderResult, error) {
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

	// Create temp file for HTML input
	htmlFile, err := os.CreateTemp(r.config.TempDir, "print-*.html")
	if err != nil {
		return nil, NewRenderError(ErrCodeRenderFailed, "failed to create temp HTML file", err)
	}
	htmlPath := htmlFile.Name()
	defer os.Remove(htmlPath)

	if _, err := htmlFile.WriteString(req.HTML); err != nil {
		htmlFile.Close()
		return nil, NewRenderError(ErrCodeRenderFailed, "failed to write HTML to temp file", err)
	}
	htmlFile.Close()

	// Create temp file for PDF output
	pdfFile, err := os.CreateTemp(r.config.TempDir, "output-*.pdf")
	if err != nil {
		return nil, NewRenderError(ErrCodeRenderFailed, "failed to create temp PDF file", err)
	}
	pdfPath := pdfFile.Name()
	pdfFile.Close()
	defer os.Remove(pdfPath)

	// Build command arguments (also creates temp files for header/footer if needed)
	args, temps := r.buildArgs(req, htmlPath, pdfPath)
	defer temps.cleanup()

	r.logger.Debug("executing wkhtmltopdf",
		zap.String("binary", r.config.BinaryPath),
		zap.Strings("args", args))

	// Execute wkhtmltopdf
	cmd := exec.CommandContext(ctx, r.config.BinaryPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		// Check if context was cancelled (timeout)
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, NewRenderError(ErrCodeRenderTimeout,
				fmt.Sprintf("PDF rendering timed out after %v", timeout), err)
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return nil, NewRenderError(ErrCodeRenderTimeout, "PDF rendering was cancelled", err)
		}

		r.logger.Error("wkhtmltopdf failed",
			zap.Error(err),
			zap.String("stderr", stderr.String()),
			zap.String("stdout", stdout.String()))

		return nil, NewRenderError(ErrCodeRenderFailed,
			"wkhtmltopdf execution failed: "+stderr.String(), err)
	}

	// Read the generated PDF
	pdfData, err := os.ReadFile(pdfPath)
	if err != nil {
		return nil, NewRenderError(ErrCodeRenderFailed, "failed to read generated PDF", err)
	}

	if len(pdfData) == 0 {
		return nil, NewRenderError(ErrCodeRenderFailed, "generated PDF is empty", nil)
	}

	// Count pages (simple estimation from PDF)
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

// buildArgs constructs the command-line arguments for wkhtmltopdf
// Returns the args slice and a tempFiles struct for cleanup
func (r *WkhtmltopdfRenderer) buildArgs(req *RenderRequest, htmlPath, pdfPath string) ([]string, *tempFiles) {
	temps := &tempFiles{}
	args := []string{
		"--quiet",
		"--encoding", "UTF-8",
		"--dpi", strconv.Itoa(r.config.DPI),
		"--image-quality", strconv.Itoa(r.config.ImageQuality),
	}

	// Paper size configuration
	args = append(args, r.buildPaperSizeArgs(req.PaperSize, req.Orientation)...)

	// Margins
	args = append(args,
		"--margin-top", fmt.Sprintf("%dmm", req.Margins.Top),
		"--margin-right", fmt.Sprintf("%dmm", req.Margins.Right),
		"--margin-bottom", fmt.Sprintf("%dmm", req.Margins.Bottom),
		"--margin-left", fmt.Sprintf("%dmm", req.Margins.Left),
	)

	// JavaScript handling
	if r.config.EnableJavaScript {
		args = append(args, "--enable-javascript")
		if r.config.JavaScriptDelay > 0 {
			args = append(args, "--javascript-delay", strconv.Itoa(r.config.JavaScriptDelay))
		}
	} else {
		args = append(args, "--disable-javascript")
	}

	// Local file access (for images)
	if req.EnableLocalFileAccess {
		args = append(args, "--enable-local-file-access")
	} else {
		args = append(args, "--disable-local-file-access")
	}

	// Title
	if req.Title != "" {
		args = append(args, "--title", req.Title)
	}

	// Header
	if req.HeaderHTML != "" {
		headerFile, err := r.writeTempHTML(req.HeaderHTML, "header-*.html")
		if err != nil {
			r.logger.Warn("failed to create header temp file", zap.Error(err))
		} else {
			args = append(args, "--header-html", headerFile)
			temps.header = headerFile
		}
	}

	// Footer
	if req.FooterHTML != "" {
		footerFile, err := r.writeTempHTML(req.FooterHTML, "footer-*.html")
		if err != nil {
			r.logger.Warn("failed to create footer temp file", zap.Error(err))
		} else {
			args = append(args, "--footer-html", footerFile)
			temps.footer = footerFile
		}
	}

	// Input and output
	args = append(args, htmlPath, pdfPath)

	return args, temps
}

// buildPaperSizeArgs generates paper size arguments
func (r *WkhtmltopdfRenderer) buildPaperSizeArgs(paperSize printing.PaperSize, orientation printing.Orientation) []string {
	var args []string

	width, _ := paperSize.Dimensions()

	switch paperSize {
	case printing.PaperSizeA4:
		args = append(args, "--page-size", "A4")
	case printing.PaperSizeA5:
		args = append(args, "--page-size", "A5")
	case printing.PaperSizeReceipt58MM, printing.PaperSizeReceipt80MM:
		// Receipt paper - continuous/variable height
		// Use custom page size with width in mm
		args = append(args,
			"--page-width", fmt.Sprintf("%dmm", width),
			"--page-height", "0", // Auto height for receipt
			"--disable-smart-shrinking",
		)
	case printing.PaperSizeContinuous241:
		// Continuous paper (dot matrix)
		args = append(args,
			"--page-width", fmt.Sprintf("%dmm", width),
			"--page-height", "0",
		)
	default:
		// Fallback to A4
		args = append(args, "--page-size", "A4")
	}

	// Orientation (only for non-receipt paper)
	if !paperSize.IsReceipt() && !paperSize.IsContinuous() {
		if orientation == printing.OrientationLandscape {
			args = append(args, "--orientation", "Landscape")
		} else {
			args = append(args, "--orientation", "Portrait")
		}
	}

	return args
}

// writeTempHTML writes HTML content to a temporary file
func (r *WkhtmltopdfRenderer) writeTempHTML(html, pattern string) (string, error) {
	file, err := os.CreateTemp(r.config.TempDir, pattern)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := file.WriteString(html); err != nil {
		os.Remove(file.Name())
		return "", err
	}

	return file.Name(), nil
}

// Close releases resources (no-op for wkhtmltopdf)
func (r *WkhtmltopdfRenderer) Close() error {
	return nil
}

// estimatePageCount estimates the page count from PDF data
// This is a simple heuristic that counts "/Type /Page" occurrences
func estimatePageCount(pdfData []byte) int {
	count := bytes.Count(pdfData, []byte("/Type /Page"))
	// Each page has one "/Type /Page" but the count also includes "/Type /Pages"
	// So we subtract the parent Pages object occurrences
	parentCount := bytes.Count(pdfData, []byte("/Type /Pages"))
	count = count - parentCount
	return max(count, 1)
}

// Ensure WkhtmltopdfRenderer implements PDFRenderer
var _ PDFRenderer = (*WkhtmltopdfRenderer)(nil)
