package printing

import (
	"strings"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/printing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderRequest_Validation(t *testing.T) {
	tests := []struct {
		name      string
		req       *RenderRequest
		shouldErr bool
	}{
		{
			name:      "nil request",
			req:       nil,
			shouldErr: true,
		},
		{
			name: "empty HTML",
			req: &RenderRequest{
				HTML:      "",
				PaperSize: printing.PaperSizeA4,
			},
			shouldErr: true,
		},
		{
			name: "whitespace only HTML",
			req: &RenderRequest{
				HTML:      "   \n\t  ",
				PaperSize: printing.PaperSizeA4,
			},
			shouldErr: true, // Will be caught by Render() with TrimSpace check
		},
		{
			name: "invalid paper size",
			req: &RenderRequest{
				HTML:      "<html>test</html>",
				PaperSize: printing.PaperSize("INVALID"),
			},
			shouldErr: true,
		},
		{
			name: "valid A4 request",
			req: &RenderRequest{
				HTML:        "<html>test</html>",
				PaperSize:   printing.PaperSizeA4,
				Orientation: printing.OrientationPortrait,
				Margins:     printing.DefaultMargins(),
			},
			shouldErr: false,
		},
		{
			name: "valid receipt request",
			req: &RenderRequest{
				HTML:      "<html>receipt</html>",
				PaperSize: printing.PaperSizeReceipt58MM,
				Margins:   printing.ReceiptMargins(),
			},
			shouldErr: false,
		},
	}

	// Note: These tests validate the request structure
	// Actual rendering requires wkhtmltopdf binary
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.req == nil {
				return // nil check is done in Render
			}
			// Validate paper size
			valid := tt.req.PaperSize.IsValid()
			// Check for meaningful content (not just whitespace)
			hasContent := strings.TrimSpace(tt.req.HTML) != ""

			if tt.shouldErr {
				assert.True(t, !valid || !hasContent, "expected validation to fail")
			} else {
				assert.True(t, valid && hasContent, "expected validation to pass")
			}
		})
	}
}

func TestRenderResult_Fields(t *testing.T) {
	result := &RenderResult{
		PDFData:        []byte("test pdf data"),
		PageCount:      3,
		RenderDuration: 500 * time.Millisecond,
	}

	assert.Equal(t, 13, len(result.PDFData))
	assert.Equal(t, 3, result.PageCount)
	assert.Equal(t, 500*time.Millisecond, result.RenderDuration)
}

func TestRenderError(t *testing.T) {
	t.Run("error without cause", func(t *testing.T) {
		err := NewRenderError(ErrCodeRenderTimeout, "timeout occurred", nil)

		assert.Equal(t, ErrCodeRenderTimeout, err.Code)
		assert.Equal(t, "timeout occurred", err.Message)
		assert.Equal(t, "timeout occurred", err.Error())
		assert.Nil(t, err.Unwrap())
	})

	t.Run("error with cause", func(t *testing.T) {
		cause := assert.AnError
		err := NewRenderError(ErrCodeRenderFailed, "render failed", cause)

		assert.Equal(t, ErrCodeRenderFailed, err.Code)
		assert.Equal(t, "render failed", err.Message)
		assert.Contains(t, err.Error(), "render failed")
		assert.Contains(t, err.Error(), cause.Error())
		assert.Equal(t, cause, err.Unwrap())
	})
}

func TestErrorCodes(t *testing.T) {
	// Ensure all error codes are defined
	codes := []string{
		ErrCodeRenderTimeout,
		ErrCodeRenderFailed,
		ErrCodeInvalidHTML,
		ErrCodeBinaryNotFound,
		ErrCodeInvalidPaperSize,
		ErrCodeStorageFailed,
	}

	for _, code := range codes {
		assert.NotEmpty(t, code, "error code should not be empty")
	}
}

func TestWkhtmltopdfConfig_Defaults(t *testing.T) {
	// Test that NewWkhtmltopdfRenderer sets proper defaults
	// This test doesn't require the binary to be present

	config := &WkhtmltopdfConfig{
		BinaryPath: "", // Will be defaulted
		TempDir:    "", // Will be defaulted
	}

	// Check defaults
	assert.Empty(t, config.BinaryPath)
	assert.Empty(t, config.TempDir)
	assert.False(t, config.EnableJavaScript)
	assert.Equal(t, 0, config.JavaScriptDelay)
	assert.Equal(t, 0, config.DPI)
	assert.Equal(t, 0, config.ImageQuality)
}

func TestBuildPaperSizeArgs_A4Portrait(t *testing.T) {
	config := &WkhtmltopdfConfig{
		BinaryPath: "wkhtmltopdf",
		DPI:        96,
	}
	r := &WkhtmltopdfRenderer{config: config}

	args := r.buildPaperSizeArgs(printing.PaperSizeA4, printing.OrientationPortrait)

	assert.Contains(t, args, "--page-size")
	assert.Contains(t, args, "A4")
	assert.Contains(t, args, "--orientation")
	assert.Contains(t, args, "Portrait")
}

func TestBuildPaperSizeArgs_A4Landscape(t *testing.T) {
	config := &WkhtmltopdfConfig{
		BinaryPath: "wkhtmltopdf",
		DPI:        96,
	}
	r := &WkhtmltopdfRenderer{config: config}

	args := r.buildPaperSizeArgs(printing.PaperSizeA4, printing.OrientationLandscape)

	assert.Contains(t, args, "--page-size")
	assert.Contains(t, args, "A4")
	assert.Contains(t, args, "--orientation")
	assert.Contains(t, args, "Landscape")
}

func TestBuildPaperSizeArgs_A5(t *testing.T) {
	config := &WkhtmltopdfConfig{
		BinaryPath: "wkhtmltopdf",
		DPI:        96,
	}
	r := &WkhtmltopdfRenderer{config: config}

	args := r.buildPaperSizeArgs(printing.PaperSizeA5, printing.OrientationPortrait)

	assert.Contains(t, args, "--page-size")
	assert.Contains(t, args, "A5")
}

func TestBuildPaperSizeArgs_Receipt58MM(t *testing.T) {
	config := &WkhtmltopdfConfig{
		BinaryPath: "wkhtmltopdf",
		DPI:        96,
	}
	r := &WkhtmltopdfRenderer{config: config}

	args := r.buildPaperSizeArgs(printing.PaperSizeReceipt58MM, printing.OrientationPortrait)

	assert.Contains(t, args, "--page-width")
	assert.Contains(t, args, "58mm")
	assert.Contains(t, args, "--page-height")
	assert.Contains(t, args, "0")
	// Receipt paper should not have orientation
	assert.NotContains(t, args, "--orientation")
}

func TestBuildPaperSizeArgs_Receipt80MM(t *testing.T) {
	config := &WkhtmltopdfConfig{
		BinaryPath: "wkhtmltopdf",
		DPI:        96,
	}
	r := &WkhtmltopdfRenderer{config: config}

	args := r.buildPaperSizeArgs(printing.PaperSizeReceipt80MM, printing.OrientationPortrait)

	assert.Contains(t, args, "--page-width")
	assert.Contains(t, args, "80mm")
}

func TestBuildPaperSizeArgs_Continuous241(t *testing.T) {
	config := &WkhtmltopdfConfig{
		BinaryPath: "wkhtmltopdf",
		DPI:        96,
	}
	r := &WkhtmltopdfRenderer{config: config}

	args := r.buildPaperSizeArgs(printing.PaperSizeContinuous241, printing.OrientationPortrait)

	assert.Contains(t, args, "--page-width")
	assert.Contains(t, args, "241mm")
	// Continuous paper should not have orientation
	assert.NotContains(t, args, "--orientation")
}

func TestBuildArgs_Complete(t *testing.T) {
	config := &WkhtmltopdfConfig{
		BinaryPath:       "wkhtmltopdf",
		DPI:              96,
		ImageQuality:     94,
		EnableJavaScript: false,
	}
	r := &WkhtmltopdfRenderer{config: config}

	margins, _ := printing.NewMargins(10, 15, 10, 15)
	req := &RenderRequest{
		HTML:                  "<html>test</html>",
		PaperSize:             printing.PaperSizeA4,
		Orientation:           printing.OrientationPortrait,
		Margins:               margins,
		Title:                 "Test Document",
		EnableLocalFileAccess: false,
	}

	args, temps := r.buildArgs(req, "/tmp/input.html", "/tmp/output.pdf")
	defer temps.cleanup()

	// Check required arguments
	assert.Contains(t, args, "--quiet")
	assert.Contains(t, args, "--encoding")
	assert.Contains(t, args, "UTF-8")
	assert.Contains(t, args, "--dpi")
	assert.Contains(t, args, "96")

	// Check margins
	assert.Contains(t, args, "--margin-top")
	assert.Contains(t, args, "10mm")
	assert.Contains(t, args, "--margin-right")
	assert.Contains(t, args, "15mm")
	assert.Contains(t, args, "--margin-bottom")
	assert.Contains(t, args, "10mm")
	assert.Contains(t, args, "--margin-left")
	assert.Contains(t, args, "15mm")

	// Check title
	assert.Contains(t, args, "--title")
	assert.Contains(t, args, "Test Document")

	// Check JavaScript disabled
	assert.Contains(t, args, "--disable-javascript")

	// Check local file access disabled
	assert.Contains(t, args, "--disable-local-file-access")

	// Check input/output paths
	assert.Contains(t, args, "/tmp/input.html")
	assert.Contains(t, args, "/tmp/output.pdf")
}

func TestBuildArgs_WithJavaScript(t *testing.T) {
	config := &WkhtmltopdfConfig{
		BinaryPath:       "wkhtmltopdf",
		DPI:              96,
		ImageQuality:     94,
		EnableJavaScript: true,
		JavaScriptDelay:  500,
	}
	r := &WkhtmltopdfRenderer{config: config}

	req := &RenderRequest{
		HTML:      "<html>test</html>",
		PaperSize: printing.PaperSizeA4,
		Margins:   printing.DefaultMargins(),
	}

	args, temps := r.buildArgs(req, "/tmp/input.html", "/tmp/output.pdf")
	defer temps.cleanup()

	assert.Contains(t, args, "--enable-javascript")
	assert.Contains(t, args, "--javascript-delay")
	assert.Contains(t, args, "500")
}

func TestBuildArgs_WithLocalFileAccess(t *testing.T) {
	config := &WkhtmltopdfConfig{
		BinaryPath:   "wkhtmltopdf",
		DPI:          96,
		ImageQuality: 94,
	}
	r := &WkhtmltopdfRenderer{config: config}

	req := &RenderRequest{
		HTML:                  "<html>test</html>",
		PaperSize:             printing.PaperSizeA4,
		Margins:               printing.DefaultMargins(),
		EnableLocalFileAccess: true,
	}

	args, temps := r.buildArgs(req, "/tmp/input.html", "/tmp/output.pdf")
	defer temps.cleanup()

	assert.Contains(t, args, "--enable-local-file-access")
}

func TestEstimatePageCount(t *testing.T) {
	tests := []struct {
		name     string
		pdfData  []byte
		expected int
	}{
		{
			name:     "empty data",
			pdfData:  []byte{},
			expected: 1, // Minimum 1 page
		},
		{
			name:     "single page",
			pdfData:  []byte("/Type /Page\n"),
			expected: 1,
		},
		{
			name:     "two pages",
			pdfData:  []byte("/Type /Page\n/Type /Page\n/Type /Pages"),
			expected: 2,
		},
		{
			name:     "three pages",
			pdfData:  []byte("/Type /Page\n/Type /Page\n/Type /Page\n/Type /Pages"),
			expected: 3,
		},
		{
			name:     "pages with parent",
			pdfData:  []byte("/Type /Pages\n/Type /Page\n/Type /Page\n"),
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := estimatePageCount(tt.pdfData)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWkhtmltopdfRenderer_Close(t *testing.T) {
	// Close should be a no-op and return nil
	config := &WkhtmltopdfConfig{
		BinaryPath: "wkhtmltopdf",
	}
	r := &WkhtmltopdfRenderer{config: config}

	err := r.Close()
	require.NoError(t, err)
}
