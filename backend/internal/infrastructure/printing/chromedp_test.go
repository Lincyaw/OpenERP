package printing

import (
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/printing"
	"github.com/stretchr/testify/assert"
)

func TestChromedpConfig_Defaults(t *testing.T) {
	config := &ChromedpConfig{}

	// Check initial state (zeros/false)
	assert.Equal(t, time.Duration(0), config.DefaultTimeout)
	assert.Empty(t, config.RemoteURL)
	assert.False(t, config.Headless)
	assert.False(t, config.DisableGPU)
	assert.False(t, config.NoSandbox)
	assert.Equal(t, 0.0, config.Scale)
	assert.False(t, config.PrintBackground)
}

func TestBuildPrintParams_A4Portrait(t *testing.T) {
	config := &ChromedpConfig{
		Scale: 1.0,
	}
	r := &ChromedpRenderer{config: config}

	req := &RenderRequest{
		HTML:        "<html>test</html>",
		PaperSize:   printing.PaperSizeA4,
		Orientation: printing.OrientationPortrait,
		Margins:     printing.DefaultMargins(),
	}

	params := r.buildPrintParams(req)

	// A4 is 210mm x 297mm
	assert.InDelta(t, mmToInches(210), params.paperWidth, 0.01)
	assert.InDelta(t, mmToInches(297), params.paperHeight, 0.01)
	assert.False(t, params.landscape)
	assert.True(t, params.printBackground)
}

func TestBuildPrintParams_A4Landscape(t *testing.T) {
	config := &ChromedpConfig{
		Scale: 1.0,
	}
	r := &ChromedpRenderer{config: config}

	req := &RenderRequest{
		HTML:        "<html>test</html>",
		PaperSize:   printing.PaperSizeA4,
		Orientation: printing.OrientationLandscape,
		Margins:     printing.DefaultMargins(),
	}

	params := r.buildPrintParams(req)

	assert.True(t, params.landscape)
}

func TestBuildPrintParams_A5(t *testing.T) {
	config := &ChromedpConfig{
		Scale: 1.0,
	}
	r := &ChromedpRenderer{config: config}

	req := &RenderRequest{
		HTML:      "<html>test</html>",
		PaperSize: printing.PaperSizeA5,
		Margins:   printing.DefaultMargins(),
	}

	params := r.buildPrintParams(req)

	// A5 is 148mm x 210mm
	assert.InDelta(t, mmToInches(148), params.paperWidth, 0.01)
	assert.InDelta(t, mmToInches(210), params.paperHeight, 0.01)
}

func TestBuildPrintParams_Receipt58MM(t *testing.T) {
	config := &ChromedpConfig{
		Scale: 1.0,
	}
	r := &ChromedpRenderer{config: config}

	req := &RenderRequest{
		HTML:      "<html>receipt</html>",
		PaperSize: printing.PaperSizeReceipt58MM,
		Margins:   printing.ReceiptMargins(),
	}

	params := r.buildPrintParams(req)

	// Receipt 58mm width
	assert.InDelta(t, mmToInches(58), params.paperWidth, 0.01)
	// Should have tall height for continuous paper
	assert.Greater(t, params.paperHeight, mmToInches(1000))
}

func TestBuildPrintParams_Receipt80MM(t *testing.T) {
	config := &ChromedpConfig{
		Scale: 1.0,
	}
	r := &ChromedpRenderer{config: config}

	req := &RenderRequest{
		HTML:      "<html>receipt</html>",
		PaperSize: printing.PaperSizeReceipt80MM,
		Margins:   printing.ReceiptMargins(),
	}

	params := r.buildPrintParams(req)

	// Receipt 80mm width
	assert.InDelta(t, mmToInches(80), params.paperWidth, 0.01)
}

func TestBuildPrintParams_Continuous241(t *testing.T) {
	config := &ChromedpConfig{
		Scale: 1.0,
	}
	r := &ChromedpRenderer{config: config}

	req := &RenderRequest{
		HTML:      "<html>continuous</html>",
		PaperSize: printing.PaperSizeContinuous241,
		Margins:   printing.DefaultMargins(),
	}

	params := r.buildPrintParams(req)

	// Continuous 241mm width
	assert.InDelta(t, mmToInches(241), params.paperWidth, 0.01)
	// Should have tall height for continuous paper
	assert.Greater(t, params.paperHeight, mmToInches(1000))
}

func TestBuildPrintParams_WithMargins(t *testing.T) {
	config := &ChromedpConfig{
		Scale: 1.0,
	}
	r := &ChromedpRenderer{config: config}

	margins, _ := printing.NewMargins(10, 15, 20, 25)
	req := &RenderRequest{
		HTML:      "<html>test</html>",
		PaperSize: printing.PaperSizeA4,
		Margins:   margins,
	}

	params := r.buildPrintParams(req)

	assert.InDelta(t, mmToInches(10), params.marginTop, 0.001)
	assert.InDelta(t, mmToInches(15), params.marginRight, 0.001)
	assert.InDelta(t, mmToInches(20), params.marginBottom, 0.001)
	assert.InDelta(t, mmToInches(25), params.marginLeft, 0.001)
}

func TestBuildPrintParams_WithHeaderFooter(t *testing.T) {
	config := &ChromedpConfig{
		Scale: 1.0,
	}
	r := &ChromedpRenderer{config: config}

	req := &RenderRequest{
		HTML:       "<html>test</html>",
		PaperSize:  printing.PaperSizeA4,
		Margins:    printing.DefaultMargins(),
		HeaderHTML: "<div>Header</div>",
		FooterHTML: "<div>Footer</div>",
	}

	params := r.buildPrintParams(req)

	assert.True(t, params.displayHeaderFooter)
	assert.Equal(t, "<div>Header</div>", params.headerTemplate)
	assert.Equal(t, "<div>Footer</div>", params.footerTemplate)
	// Should have minimum margins for header/footer
	assert.GreaterOrEqual(t, params.marginTop, mmToInches(10))
	assert.GreaterOrEqual(t, params.marginBottom, mmToInches(10))
}

func TestBuildCompleteHTML_WithDoctype(t *testing.T) {
	config := &ChromedpConfig{}
	r := &ChromedpRenderer{config: config}

	html := "<!DOCTYPE html><html><head></head><body>test</body></html>"
	req := &RenderRequest{
		HTML: html,
	}

	result := r.buildCompleteHTML(req)

	// Should return as-is since it has DOCTYPE
	assert.Equal(t, html, result)
}

func TestBuildCompleteHTML_WithHtmlTag(t *testing.T) {
	config := &ChromedpConfig{}
	r := &ChromedpRenderer{config: config}

	html := "<html><head></head><body>test</body></html>"
	req := &RenderRequest{
		HTML: html,
	}

	result := r.buildCompleteHTML(req)

	// Should return as-is since it has <html> tag
	assert.Equal(t, html, result)
}

func TestBuildCompleteHTML_FragmentOnly(t *testing.T) {
	config := &ChromedpConfig{}
	r := &ChromedpRenderer{config: config}

	req := &RenderRequest{
		HTML:  "<div>Hello World</div>",
		Title: "Test Document",
	}

	result := r.buildCompleteHTML(req)

	assert.Contains(t, result, "<!DOCTYPE html>")
	assert.Contains(t, result, "<html>")
	assert.Contains(t, result, "<head>")
	assert.Contains(t, result, "<meta charset=\"UTF-8\">")
	assert.Contains(t, result, "<title>Test Document</title>")
	assert.Contains(t, result, "<body>")
	assert.Contains(t, result, "<div>Hello World</div>")
	assert.Contains(t, result, "</body></html>")
}

func TestMmToInches(t *testing.T) {
	tests := []struct {
		mm       float64
		expected float64
	}{
		{0, 0},
		{25.4, 1.0},
		{50.8, 2.0},
		{210, 8.2677},  // A4 width
		{297, 11.6929}, // A4 height
	}

	for _, tt := range tests {
		result := mmToInches(tt.mm)
		assert.InDelta(t, tt.expected, result, 0.001)
	}
}

func TestInchesToMM(t *testing.T) {
	tests := []struct {
		inches   float64
		expected float64
	}{
		{0, 0},
		{1.0, 25.4},
		{2.0, 50.8},
		{8.5, 215.9},  // Letter width
		{11.0, 279.4}, // Letter height
	}

	for _, tt := range tests {
		result := inchesToMM(tt.inches)
		assert.InDelta(t, tt.expected, result, 0.001)
	}
}

func TestDataURLToBytes(t *testing.T) {
	t.Run("valid data URL", func(t *testing.T) {
		// "Hello" in base64
		dataURL := "data:text/plain;base64,SGVsbG8="
		data, err := dataURLToBytes(dataURL)

		assert.NoError(t, err)
		assert.Equal(t, []byte("Hello"), data)
	})

	t.Run("invalid format - no comma", func(t *testing.T) {
		dataURL := "data:text/plain;base64SGVsbG8="
		_, err := dataURLToBytes(dataURL)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid data URL format")
	})
}

func TestChromedpRenderer_Close(t *testing.T) {
	// Test that Close doesn't panic with nil allocCancel
	r := &ChromedpRenderer{
		config: &ChromedpConfig{},
	}

	err := r.Close()
	assert.NoError(t, err)
}
