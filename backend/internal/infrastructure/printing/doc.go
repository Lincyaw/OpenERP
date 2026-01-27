// Package printing provides infrastructure implementations for PDF generation
// and print job rendering using wkhtmltopdf.
//
// This package contains:
// - PDFRenderer interface for rendering HTML to PDF
// - WkhtmltopdfRenderer implementation using the wkhtmltopdf command-line tool
// - PDFStorage interface for storing and managing generated PDF files
// - FileSystemStorage implementation for local file system storage
//
// Example usage:
//
//	config := &WkhtmltopdfConfig{
//	    BinaryPath:     "/usr/bin/wkhtmltopdf",
//	    DefaultTimeout: 30 * time.Second,
//	}
//	renderer, err := NewWkhtmltopdfRenderer(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	result, err := renderer.Render(ctx, &RenderRequest{
//	    HTML:        "<html>...</html>",
//	    PaperSize:   printing.PaperSizeA4,
//	    Orientation: printing.OrientationPortrait,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Generated PDF: %d bytes\n", len(result.PDFData))
package printing
