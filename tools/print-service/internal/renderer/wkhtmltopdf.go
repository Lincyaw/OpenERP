// Package renderer provides PDF rendering capabilities.
package renderer

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// WkhtmltopdfOptions contains options for the wkhtmltopdf renderer.
type WkhtmltopdfOptions struct {
	// Timeout is the maximum time to wait for rendering.
	Timeout time.Duration
	// BinaryPath is the path to the wkhtmltopdf executable.
	// If empty, it will be searched in PATH.
	BinaryPath string
	// TempDir is the directory for temporary files.
	// If empty, the system temp directory is used.
	TempDir string
	// EnableLocalFileAccess allows access to local files.
	EnableLocalFileAccess bool
	// JavascriptDelay is the time to wait for JavaScript execution (ms).
	JavascriptDelay int
}

// DefaultWkhtmltopdfOptions returns default wkhtmltopdf options.
func DefaultWkhtmltopdfOptions() WkhtmltopdfOptions {
	return WkhtmltopdfOptions{
		Timeout:               60 * time.Second,
		EnableLocalFileAccess: true,
		JavascriptDelay:       200,
	}
}

// WkhtmltopdfRenderer renders PDFs using the wkhtmltopdf command-line tool.
type WkhtmltopdfRenderer struct {
	opts       WkhtmltopdfOptions
	binaryPath string
	available  bool
}

// NewWkhtmltopdfRenderer creates a new wkhtmltopdf renderer.
func NewWkhtmltopdfRenderer(opts WkhtmltopdfOptions) (*WkhtmltopdfRenderer, error) {
	r := &WkhtmltopdfRenderer{
		opts:      opts,
		available: false,
	}

	// Find wkhtmltopdf binary
	binaryPath := opts.BinaryPath
	if binaryPath == "" {
		// Try to find in PATH
		paths := []string{
			"wkhtmltopdf",
			"/usr/bin/wkhtmltopdf",
			"/usr/local/bin/wkhtmltopdf",
		}
		for _, p := range paths {
			if path, err := exec.LookPath(p); err == nil {
				binaryPath = path
				break
			}
		}
	}

	if binaryPath == "" {
		return r, nil // Return renderer but mark as unavailable
	}

	// Verify the binary works
	cmd := exec.Command(binaryPath, "--version")
	if err := cmd.Run(); err != nil {
		return r, nil // Return renderer but mark as unavailable
	}

	r.binaryPath = binaryPath
	r.available = true
	return r, nil
}

// RenderHTML renders HTML content to PDF.
func (r *WkhtmltopdfRenderer) RenderHTML(ctx context.Context, html string, opts RenderOptions) ([]byte, error) {
	if !r.available {
		return nil, ErrEngineNotAvailable
	}

	// Create temporary directory for this render
	tempDir := r.opts.TempDir
	if tempDir == "" {
		tempDir = os.TempDir()
	}

	// Create temp HTML file
	htmlFile, err := os.CreateTemp(tempDir, "render-*.html")
	if err != nil {
		return nil, fmt.Errorf("%w: creating temp HTML file: %v", ErrRenderFailed, err)
	}
	defer os.Remove(htmlFile.Name())

	if _, err := htmlFile.WriteString(html); err != nil {
		htmlFile.Close()
		return nil, fmt.Errorf("%w: writing temp HTML file: %v", ErrRenderFailed, err)
	}
	htmlFile.Close()

	// Create temp PDF file
	pdfFile, err := os.CreateTemp(tempDir, "output-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("%w: creating temp PDF file: %v", ErrRenderFailed, err)
	}
	pdfPath := pdfFile.Name()
	pdfFile.Close()
	defer os.Remove(pdfPath)

	// Build command arguments
	args := r.buildArgs(opts)
	args = append(args, htmlFile.Name(), pdfPath)

	// Execute wkhtmltopdf
	if err := r.execute(ctx, args); err != nil {
		return nil, err
	}

	// Read the PDF file
	pdfData, err := os.ReadFile(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("%w: reading PDF file: %v", ErrRenderFailed, err)
	}

	return pdfData, nil
}

// RenderURL renders a URL to PDF.
func (r *WkhtmltopdfRenderer) RenderURL(ctx context.Context, url string, opts RenderOptions) ([]byte, error) {
	if !r.available {
		return nil, ErrEngineNotAvailable
	}

	// Create temporary directory for this render
	tempDir := r.opts.TempDir
	if tempDir == "" {
		tempDir = os.TempDir()
	}

	// Create temp PDF file
	pdfFile, err := os.CreateTemp(tempDir, "output-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("%w: creating temp PDF file: %v", ErrRenderFailed, err)
	}
	pdfPath := pdfFile.Name()
	pdfFile.Close()
	defer os.Remove(pdfPath)

	// Build command arguments
	args := r.buildArgs(opts)
	args = append(args, url, pdfPath)

	// Execute wkhtmltopdf
	if err := r.execute(ctx, args); err != nil {
		return nil, err
	}

	// Read the PDF file
	pdfData, err := os.ReadFile(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("%w: reading PDF file: %v", ErrRenderFailed, err)
	}

	return pdfData, nil
}

// buildArgs builds command-line arguments for wkhtmltopdf.
func (r *WkhtmltopdfRenderer) buildArgs(opts RenderOptions) []string {
	args := []string{
		"--quiet",
	}

	// Page size
	switch opts.PageSize {
	case PageSizeA4:
		args = append(args, "--page-size", "A4")
	case PageSizeA3:
		args = append(args, "--page-size", "A3")
	case PageSizeLetter:
		args = append(args, "--page-size", "Letter")
	case PageSizeLegal:
		args = append(args, "--page-size", "Legal")
	default:
		args = append(args, "--page-size", "A4")
	}

	// Orientation
	if opts.Orientation == OrientationLandscape {
		args = append(args, "--orientation", "Landscape")
	} else {
		args = append(args, "--orientation", "Portrait")
	}

	// Margins (in mm)
	args = append(args, "--margin-top", fmt.Sprintf("%.0fmm", opts.MarginTop))
	args = append(args, "--margin-bottom", fmt.Sprintf("%.0fmm", opts.MarginBottom))
	args = append(args, "--margin-left", fmt.Sprintf("%.0fmm", opts.MarginLeft))
	args = append(args, "--margin-right", fmt.Sprintf("%.0fmm", opts.MarginRight))

	// Print background
	if opts.PrintBackground {
		args = append(args, "--print-media-type")
	} else {
		args = append(args, "--no-background")
	}

	// JavaScript delay
	if r.opts.JavascriptDelay > 0 {
		args = append(args, "--javascript-delay", fmt.Sprintf("%d", r.opts.JavascriptDelay))
	}

	// Enable local file access
	if r.opts.EnableLocalFileAccess {
		args = append(args, "--enable-local-file-access")
	}

	// Header and footer
	if opts.DisplayHeaderFooter {
		if opts.HeaderTemplate != "" {
			// Write header to temp file
			headerFile := filepath.Join(os.TempDir(), "header.html")
			os.WriteFile(headerFile, []byte(opts.HeaderTemplate), 0644)
			args = append(args, "--header-html", headerFile)
		}
		if opts.FooterTemplate != "" {
			// Write footer to temp file
			footerFile := filepath.Join(os.TempDir(), "footer.html")
			os.WriteFile(footerFile, []byte(opts.FooterTemplate), 0644)
			args = append(args, "--footer-html", footerFile)
		}
	}

	return args
}

// execute runs the wkhtmltopdf command.
func (r *WkhtmltopdfRenderer) execute(ctx context.Context, args []string) error {
	// Apply timeout
	timeout := r.opts.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, r.binaryPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return ErrTimeout
		}
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return fmt.Errorf("%w: %s", ErrRenderFailed, errMsg)
		}
		return fmt.Errorf("%w: %v", ErrRenderFailed, err)
	}

	return nil
}

// IsAvailable returns true if wkhtmltopdf is available.
func (r *WkhtmltopdfRenderer) IsAvailable() bool {
	return r.available
}

// Name returns the name of the renderer.
func (r *WkhtmltopdfRenderer) Name() string {
	return string(EngineWkhtmltopdf)
}

// Close releases resources held by the renderer.
func (r *WkhtmltopdfRenderer) Close() error {
	// wkhtmltopdf doesn't hold any persistent resources
	return nil
}
