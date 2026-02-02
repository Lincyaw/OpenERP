package renderer

import (
	"context"
	"testing"
	"time"
)

func TestDefaultRenderOptions(t *testing.T) {
	opts := DefaultRenderOptions()

	if opts.PageSize != PageSizeA4 {
		t.Errorf("expected PageSize A4, got %s", opts.PageSize)
	}
	if opts.Orientation != OrientationPortrait {
		t.Errorf("expected Orientation portrait, got %s", opts.Orientation)
	}
	if opts.MarginTop != 10 {
		t.Errorf("expected MarginTop 10, got %f", opts.MarginTop)
	}
	if opts.MarginBottom != 10 {
		t.Errorf("expected MarginBottom 10, got %f", opts.MarginBottom)
	}
	if opts.MarginLeft != 10 {
		t.Errorf("expected MarginLeft 10, got %f", opts.MarginLeft)
	}
	if opts.MarginRight != 10 {
		t.Errorf("expected MarginRight 10, got %f", opts.MarginRight)
	}
	if opts.Scale != 1.0 {
		t.Errorf("expected Scale 1.0, got %f", opts.Scale)
	}
	if !opts.PrintBackground {
		t.Error("expected PrintBackground true")
	}
}

func TestDefaultChromedpOptions(t *testing.T) {
	opts := DefaultChromedpOptions()

	if opts.Timeout != 60*time.Second {
		t.Errorf("expected Timeout 60s, got %v", opts.Timeout)
	}
	if !opts.Headless {
		t.Error("expected Headless true")
	}
	if !opts.DisableGPU {
		t.Error("expected DisableGPU true")
	}
	if opts.NoSandbox {
		t.Error("expected NoSandbox false")
	}
}

func TestDefaultWkhtmltopdfOptions(t *testing.T) {
	opts := DefaultWkhtmltopdfOptions()

	if opts.Timeout != 60*time.Second {
		t.Errorf("expected Timeout 60s, got %v", opts.Timeout)
	}
	if !opts.EnableLocalFileAccess {
		t.Error("expected EnableLocalFileAccess true")
	}
	if opts.JavascriptDelay != 200 {
		t.Errorf("expected JavascriptDelay 200, got %d", opts.JavascriptDelay)
	}
}

func TestNewChromedpRenderer(t *testing.T) {
	opts := DefaultChromedpOptions()
	renderer, err := NewChromedpRenderer(opts)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if renderer == nil {
		t.Fatal("expected renderer to be non-nil")
	}

	// Renderer should be created even if Chrome is not available
	if renderer.Name() != string(EngineChromedp) {
		t.Errorf("expected name %s, got %s", EngineChromedp, renderer.Name())
	}

	// Close should not error
	if err := renderer.Close(); err != nil {
		t.Errorf("unexpected error on close: %v", err)
	}
}

func TestNewWkhtmltopdfRenderer(t *testing.T) {
	opts := DefaultWkhtmltopdfOptions()
	renderer, err := NewWkhtmltopdfRenderer(opts)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if renderer == nil {
		t.Fatal("expected renderer to be non-nil")
	}

	// Renderer should be created even if wkhtmltopdf is not available
	if renderer.Name() != string(EngineWkhtmltopdf) {
		t.Errorf("expected name %s, got %s", EngineWkhtmltopdf, renderer.Name())
	}

	// Close should not error
	if err := renderer.Close(); err != nil {
		t.Errorf("unexpected error on close: %v", err)
	}
}

func TestNewManager(t *testing.T) {
	// Test with chromedp (may not be available)
	manager, err := NewManager(EngineChromedp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if manager == nil {
		t.Fatal("expected manager to be non-nil")
	}

	// Close should not error
	if err := manager.Close(); err != nil {
		t.Errorf("unexpected error on close: %v", err)
	}
}

func TestNewManagerWithFallback(t *testing.T) {
	manager, err := NewManager(EngineChromedp, WithFallback(EngineWkhtmltopdf))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if manager == nil {
		t.Fatal("expected manager to be non-nil")
	}

	// Should have at least one engine
	engines := manager.AvailableEngines()
	// Note: engines may be empty if neither Chrome nor wkhtmltopdf is installed
	t.Logf("Available engines: %v", engines)

	if err := manager.Close(); err != nil {
		t.Errorf("unexpected error on close: %v", err)
	}
}

func TestManagerRenderHTMLEmptyInput(t *testing.T) {
	manager, err := NewManager(EngineChromedp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()
	opts := DefaultRenderOptions()

	_, err = manager.RenderHTML(ctx, "", opts)
	if err == nil {
		t.Error("expected error for empty HTML")
	}
}

func TestManagerRenderURLEmptyInput(t *testing.T) {
	manager, err := NewManager(EngineChromedp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()
	opts := DefaultRenderOptions()

	_, err = manager.RenderURL(ctx, "", opts)
	if err == nil {
		t.Error("expected error for empty URL")
	}
}

func TestChromedpBuildPrintOptions(t *testing.T) {
	renderer := &ChromedpRenderer{}

	tests := []struct {
		name          string
		opts          RenderOptions
		wantWidth     float64
		wantHeight    float64
		wantLandscape bool
	}{
		{
			name: "A4 Portrait",
			opts: RenderOptions{
				PageSize:    PageSizeA4,
				Orientation: OrientationPortrait,
			},
			wantWidth:     8.27,
			wantHeight:    11.69,
			wantLandscape: false,
		},
		{
			name: "A4 Landscape",
			opts: RenderOptions{
				PageSize:    PageSizeA4,
				Orientation: OrientationLandscape,
			},
			wantWidth:     8.27,
			wantHeight:    11.69,
			wantLandscape: true,
		},
		{
			name: "Letter Portrait",
			opts: RenderOptions{
				PageSize:    PageSizeLetter,
				Orientation: OrientationPortrait,
			},
			wantWidth:     8.5,
			wantHeight:    11,
			wantLandscape: false,
		},
		{
			name: "A3 Portrait",
			opts: RenderOptions{
				PageSize:    PageSizeA3,
				Orientation: OrientationPortrait,
			},
			wantWidth:     11.69,
			wantHeight:    16.54,
			wantLandscape: false,
		},
		{
			name: "Legal Portrait",
			opts: RenderOptions{
				PageSize:    PageSizeLegal,
				Orientation: OrientationPortrait,
			},
			wantWidth:     8.5,
			wantHeight:    14,
			wantLandscape: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderer.buildPrintOptions(tt.opts)

			if result.paperWidth != tt.wantWidth {
				t.Errorf("paperWidth = %v, want %v", result.paperWidth, tt.wantWidth)
			}
			if result.paperHeight != tt.wantHeight {
				t.Errorf("paperHeight = %v, want %v", result.paperHeight, tt.wantHeight)
			}
			if result.landscape != tt.wantLandscape {
				t.Errorf("landscape = %v, want %v", result.landscape, tt.wantLandscape)
			}
		})
	}
}

func TestChromedpBuildPrintOptionsMargins(t *testing.T) {
	renderer := &ChromedpRenderer{}

	opts := RenderOptions{
		PageSize:     PageSizeA4,
		MarginTop:    25.4, // 1 inch in mm
		MarginBottom: 25.4,
		MarginLeft:   25.4,
		MarginRight:  25.4,
	}

	result := renderer.buildPrintOptions(opts)

	// 25.4mm = 1 inch
	expectedMargin := 1.0
	tolerance := 0.001

	if diff := result.marginTop - expectedMargin; diff > tolerance || diff < -tolerance {
		t.Errorf("marginTop = %v, want %v", result.marginTop, expectedMargin)
	}
	if diff := result.marginBottom - expectedMargin; diff > tolerance || diff < -tolerance {
		t.Errorf("marginBottom = %v, want %v", result.marginBottom, expectedMargin)
	}
	if diff := result.marginLeft - expectedMargin; diff > tolerance || diff < -tolerance {
		t.Errorf("marginLeft = %v, want %v", result.marginLeft, expectedMargin)
	}
	if diff := result.marginRight - expectedMargin; diff > tolerance || diff < -tolerance {
		t.Errorf("marginRight = %v, want %v", result.marginRight, expectedMargin)
	}
}

func TestChromedpBuildPrintOptionsScale(t *testing.T) {
	renderer := &ChromedpRenderer{}

	// Test default scale
	opts := RenderOptions{
		Scale: 0, // Should default to 1.0
	}
	result := renderer.buildPrintOptions(opts)
	if result.scale != 1.0 {
		t.Errorf("scale = %v, want 1.0", result.scale)
	}

	// Test custom scale
	opts.Scale = 0.5
	result = renderer.buildPrintOptions(opts)
	if result.scale != 0.5 {
		t.Errorf("scale = %v, want 0.5", result.scale)
	}
}

func TestWkhtmltopdfBuildArgs(t *testing.T) {
	renderer := &WkhtmltopdfRenderer{
		opts: WkhtmltopdfOptions{
			EnableLocalFileAccess: true,
			JavascriptDelay:       200,
		},
	}

	opts := RenderOptions{
		PageSize:        PageSizeA4,
		Orientation:     OrientationLandscape,
		MarginTop:       10,
		MarginBottom:    10,
		MarginLeft:      10,
		MarginRight:     10,
		PrintBackground: true,
	}

	args := renderer.buildArgs(opts)

	// Check for expected arguments
	hasQuiet := false
	hasPageSize := false
	hasOrientation := false
	hasMargins := false
	hasPrintMedia := false
	hasJsDelay := false
	hasLocalFile := false

	for i, arg := range args {
		switch arg {
		case "--quiet":
			hasQuiet = true
		case "--page-size":
			if i+1 < len(args) && args[i+1] == "A4" {
				hasPageSize = true
			}
		case "--orientation":
			if i+1 < len(args) && args[i+1] == "Landscape" {
				hasOrientation = true
			}
		case "--margin-top":
			hasMargins = true
		case "--print-media-type":
			hasPrintMedia = true
		case "--javascript-delay":
			hasJsDelay = true
		case "--enable-local-file-access":
			hasLocalFile = true
		}
	}

	if !hasQuiet {
		t.Error("expected --quiet argument")
	}
	if !hasPageSize {
		t.Error("expected --page-size A4 argument")
	}
	if !hasOrientation {
		t.Error("expected --orientation Landscape argument")
	}
	if !hasMargins {
		t.Error("expected margin arguments")
	}
	if !hasPrintMedia {
		t.Error("expected --print-media-type argument")
	}
	if !hasJsDelay {
		t.Error("expected --javascript-delay argument")
	}
	if !hasLocalFile {
		t.Error("expected --enable-local-file-access argument")
	}
}

func TestCreateRendererInvalidEngine(t *testing.T) {
	_, err := createRenderer("invalid", &managerConfig{})
	if err == nil {
		t.Error("expected error for invalid engine type")
	}
}
