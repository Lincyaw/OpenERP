package printing

import (
	"strings"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/printing"
	"github.com/stretchr/testify/assert"
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
