package printer

import (
	"testing"
	"time"
)

func TestDefaultPrintOptions(t *testing.T) {
	opts := DefaultPrintOptions()

	if opts.Copies != 1 {
		t.Errorf("expected Copies 1, got %d", opts.Copies)
	}
	if opts.Media != MediaA4 {
		t.Errorf("expected Media A4, got %s", opts.Media)
	}
	if opts.Quality != QualityNormal {
		t.Errorf("expected Quality Normal, got %d", opts.Quality)
	}
	if opts.Orientation != OrientationPortrait {
		t.Errorf("expected Orientation Portrait, got %d", opts.Orientation)
	}
	if opts.Sides != SidesOneSided {
		t.Errorf("expected Sides OneSided, got %s", opts.Sides)
	}
	if opts.ColorMode != "color" {
		t.Errorf("expected ColorMode color, got %s", opts.ColorMode)
	}
}

func TestPrinterStateString(t *testing.T) {
	tests := []struct {
		state    PrinterState
		expected string
	}{
		{PrinterStateIdle, "idle"},
		{PrinterStateProcessing, "processing"},
		{PrinterStateStopped, "stopped"},
		{PrinterState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("PrinterState.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestJobStateString(t *testing.T) {
	tests := []struct {
		state    JobState
		expected string
	}{
		{JobStatePending, "pending"},
		{JobStateHeld, "held"},
		{JobStateProcessing, "processing"},
		{JobStateStopped, "stopped"},
		{JobStateCanceled, "canceled"},
		{JobStateAborted, "aborted"},
		{JobStateCompleted, "completed"},
		{JobState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("JobState.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParsePageRanges(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []int
	}{
		{
			name:     "single page",
			input:    "5",
			expected: []int{5, 5},
		},
		{
			name:     "page range",
			input:    "1-5",
			expected: []int{1, 5},
		},
		{
			name:     "multiple ranges",
			input:    "1-3,5,7-9",
			expected: []int{1, 3, 5, 5, 7, 9},
		},
		{
			name:     "with spaces",
			input:    "1 - 3, 5, 7 - 9",
			expected: []int{1, 3, 5, 5, 7, 9},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "invalid range",
			input:    "abc",
			expected: nil,
		},
		{
			name:     "invalid page number",
			input:    "0",
			expected: nil,
		},
		{
			name:     "reversed range",
			input:    "5-1",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePageRanges(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("parsePageRanges(%q) = %v, want %v", tt.input, result, tt.expected)
				return
			}

			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("parsePageRanges(%q)[%d] = %v, want %v", tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestNewIPPBackendInvalidURL(t *testing.T) {
	_, err := NewIPPBackend(IPPConfig{
		ServerURL: "",
	})
	if err == nil {
		t.Error("expected error for empty server URL")
	}
}

func TestNewIPPBackendInvalidURLFormat(t *testing.T) {
	_, err := NewIPPBackend(IPPConfig{
		ServerURL: "://invalid",
	})
	if err == nil {
		t.Error("expected error for invalid URL format")
	}
}

func TestNewIPPBackendUnavailable(t *testing.T) {
	// Use a non-existent server
	backend, err := NewIPPBackend(IPPConfig{
		ServerURL: "http://localhost:99999",
	})

	// Should not return error, but backend should be unavailable
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if backend == nil {
		t.Fatal("expected backend to be non-nil")
	}
	if backend.IsAvailable() {
		t.Error("expected backend to be unavailable")
	}
	if backend.Protocol() != ProtocolIPP {
		t.Errorf("expected protocol IPP, got %s", backend.Protocol())
	}
}

func TestNewCUPSBackend(t *testing.T) {
	backend, err := NewCUPSBackend(CUPSConfig{})

	// Should not return error even if CUPS is not available
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if backend == nil {
		t.Fatal("expected backend to be non-nil")
	}
	if backend.Protocol() != ProtocolCUPS {
		t.Errorf("expected protocol CUPS, got %s", backend.Protocol())
	}

	// Close should not error
	if err := backend.Close(); err != nil {
		t.Errorf("unexpected error on close: %v", err)
	}
}

func TestCUPSBuildOptions(t *testing.T) {
	backend := &CUPSBackend{}

	opts := PrintOptions{
		Media:       MediaA4,
		Quality:     QualityHigh,
		Orientation: OrientationLandscape,
		Sides:       SidesTwoSidedLongEdge,
		ColorMode:   "monochrome",
		PageRanges:  "1-5",
	}

	options := backend.buildOptions(opts)

	// Check for expected options
	hasMedia := false
	hasQuality := false
	hasOrientation := false
	hasSides := false
	hasColorMode := false
	hasPageRanges := false

	for _, opt := range options {
		switch {
		case opt == "media=A4":
			hasMedia = true
		case opt == "print-quality=high":
			hasQuality = true
		case opt == "orientation-requested=landscape":
			hasOrientation = true
		case opt == "sides=two-sided-long-edge":
			hasSides = true
		case opt == "print-color-mode=monochrome":
			hasColorMode = true
		case opt == "page-ranges=1-5":
			hasPageRanges = true
		}
	}

	if !hasMedia {
		t.Error("expected media=A4 option")
	}
	if !hasQuality {
		t.Error("expected print-quality=high option")
	}
	if !hasOrientation {
		t.Error("expected orientation-requested=landscape option")
	}
	if !hasSides {
		t.Error("expected sides=two-sided-long-edge option")
	}
	if !hasColorMode {
		t.Error("expected print-color-mode=monochrome option")
	}
	if !hasPageRanges {
		t.Error("expected page-ranges=1-5 option")
	}
}

func TestCUPSParseJobID(t *testing.T) {
	backend := &CUPSBackend{}

	tests := []struct {
		name     string
		output   string
		expected int
	}{
		{
			name:     "standard output",
			output:   "request id is HP_LaserJet-123 (1 file(s))",
			expected: 123,
		},
		{
			name:     "printer with hyphen",
			output:   "request id is HP-LaserJet-Pro-456 (1 file(s))",
			expected: 456,
		},
		{
			name:     "no match",
			output:   "some other output",
			expected: 0,
		},
		{
			name:     "empty output",
			output:   "",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := backend.parseJobID(tt.output)
			if result != tt.expected {
				t.Errorf("parseJobID(%q) = %d, want %d", tt.output, result, tt.expected)
			}
		})
	}
}

func TestNewManagerUnsupportedProtocol(t *testing.T) {
	_, err := NewManager(ManagerConfig{
		Protocol: Protocol("unsupported"),
	})
	if err == nil {
		t.Error("expected error for unsupported protocol")
	}
}

func TestNewManagerRawProtocol(t *testing.T) {
	_, err := NewManager(ManagerConfig{
		Protocol: ProtocolRaw,
	})
	if err == nil {
		t.Error("expected error for raw protocol (not implemented)")
	}
}

func TestNewManagerWindowsProtocol(t *testing.T) {
	_, err := NewManager(ManagerConfig{
		Protocol: ProtocolWindows,
	})
	if err == nil {
		t.Error("expected error for windows protocol (not implemented)")
	}
}

func TestPrintJobStatus(t *testing.T) {
	now := time.Now()
	status := PrintJobStatus{
		JobID:        123,
		State:        JobStateProcessing,
		StateMessage: "Printing page 2 of 5",
		PrinterName:  "HP_LaserJet",
		JobName:      "Test Document",
		CreatedAt:    now,
		PagesPrinted: 2,
		TotalPages:   5,
	}

	if status.JobID != 123 {
		t.Errorf("expected JobID 123, got %d", status.JobID)
	}
	if status.State != JobStateProcessing {
		t.Errorf("expected State Processing, got %v", status.State)
	}
	if status.PrinterName != "HP_LaserJet" {
		t.Errorf("expected PrinterName HP_LaserJet, got %s", status.PrinterName)
	}
}

func TestPrinterInfo(t *testing.T) {
	info := PrinterInfo{
		Name:                "HP_LaserJet",
		Description:         "HP LaserJet Pro",
		Location:            "Office 101",
		URI:                 "ipp://localhost:631/printers/HP_LaserJet",
		IsDefault:           true,
		State:               PrinterStateIdle,
		MakeAndModel:        "HP LaserJet Pro M404dn",
		SupportedMediaSizes: []MediaSize{MediaA4, MediaLetter},
		SupportsDuplex:      true,
		SupportsColor:       false,
	}

	if info.Name != "HP_LaserJet" {
		t.Errorf("expected Name HP_LaserJet, got %s", info.Name)
	}
	if !info.IsDefault {
		t.Error("expected IsDefault true")
	}
	if info.State != PrinterStateIdle {
		t.Errorf("expected State Idle, got %v", info.State)
	}
	if !info.SupportsDuplex {
		t.Error("expected SupportsDuplex true")
	}
	if info.SupportsColor {
		t.Error("expected SupportsColor false")
	}
}

func TestMediaSizeConstants(t *testing.T) {
	if MediaA4 != "iso_a4_210x297mm" {
		t.Errorf("unexpected MediaA4 value: %s", MediaA4)
	}
	if MediaA3 != "iso_a3_297x420mm" {
		t.Errorf("unexpected MediaA3 value: %s", MediaA3)
	}
	if MediaLetter != "na_letter_8.5x11in" {
		t.Errorf("unexpected MediaLetter value: %s", MediaLetter)
	}
	if MediaLegal != "na_legal_8.5x14in" {
		t.Errorf("unexpected MediaLegal value: %s", MediaLegal)
	}
}

func TestSidesConstants(t *testing.T) {
	if SidesOneSided != "one-sided" {
		t.Errorf("unexpected SidesOneSided value: %s", SidesOneSided)
	}
	if SidesTwoSidedLongEdge != "two-sided-long-edge" {
		t.Errorf("unexpected SidesTwoSidedLongEdge value: %s", SidesTwoSidedLongEdge)
	}
	if SidesTwoSidedShortEdge != "two-sided-short-edge" {
		t.Errorf("unexpected SidesTwoSidedShortEdge value: %s", SidesTwoSidedShortEdge)
	}
}

func TestQualityConstants(t *testing.T) {
	if QualityDraft != 3 {
		t.Errorf("unexpected QualityDraft value: %d", QualityDraft)
	}
	if QualityNormal != 4 {
		t.Errorf("unexpected QualityNormal value: %d", QualityNormal)
	}
	if QualityHigh != 5 {
		t.Errorf("unexpected QualityHigh value: %d", QualityHigh)
	}
}

func TestOrientationConstants(t *testing.T) {
	if OrientationPortrait != 3 {
		t.Errorf("unexpected OrientationPortrait value: %d", OrientationPortrait)
	}
	if OrientationLandscape != 4 {
		t.Errorf("unexpected OrientationLandscape value: %d", OrientationLandscape)
	}
}
