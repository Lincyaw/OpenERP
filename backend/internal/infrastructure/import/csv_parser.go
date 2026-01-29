package csvimport

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"unicode/utf8"
)

// CSVParser handles parsing of CSV files with encoding detection
type CSVParser struct {
	delimiter  rune
	lazyQuotes bool
	trimSpace  bool
	headerMap  map[string]int
	headers    []string
	currentRow int
	totalRows  int
	reader     *csv.Reader
	bufReader  *bufio.Reader
}

// ParserOption is a functional option for CSVParser configuration
type ParserOption func(*CSVParser)

// WithDelimiter sets the field delimiter (default is comma)
func WithDelimiter(d rune) ParserOption {
	return func(p *CSVParser) {
		p.delimiter = d
	}
}

// WithLazyQuotes enables lazy quote handling
func WithLazyQuotes(lazy bool) ParserOption {
	return func(p *CSVParser) {
		p.lazyQuotes = lazy
	}
}

// WithTrimSpace enables trimming of leading/trailing spaces from fields
func WithTrimSpace(trim bool) ParserOption {
	return func(p *CSVParser) {
		p.trimSpace = trim
	}
}

// NewCSVParser creates a new CSV parser from a reader
func NewCSVParser(r io.Reader, opts ...ParserOption) (*CSVParser, error) {
	parser := &CSVParser{
		delimiter:  ',',
		lazyQuotes: true,
		trimSpace:  true,
		headerMap:  make(map[string]int),
	}

	for _, opt := range opts {
		opt(parser)
	}

	// Wrap in buffered reader for BOM detection
	parser.bufReader = bufio.NewReader(r)

	// Detect and strip UTF-8 BOM
	content, err := parser.bufReader.Peek(3)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// UTF-8 BOM: 0xEF, 0xBB, 0xBF
	if len(content) >= 3 && content[0] == 0xEF && content[1] == 0xBB && content[2] == 0xBF {
		// Discard BOM
		_, _ = parser.bufReader.Discard(3)
	}

	// Validate encoding is UTF-8
	if err := validateUTF8(parser.bufReader); err != nil {
		return nil, err
	}

	// Create CSV reader
	parser.reader = csv.NewReader(parser.bufReader)
	parser.reader.Comma = parser.delimiter
	parser.reader.LazyQuotes = parser.lazyQuotes
	parser.reader.TrimLeadingSpace = parser.trimSpace
	parser.reader.FieldsPerRecord = -1 // Allow variable number of fields

	return parser, nil
}

// validateUTF8 checks that the content is valid UTF-8
func validateUTF8(r *bufio.Reader) error {
	// Peek enough bytes to check for valid UTF-8
	const checkSize = 4096
	content, err := r.Peek(checkSize)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read file for encoding validation: %w", err)
	}

	if len(content) == 0 {
		return ErrEmptyFile
	}

	if !utf8.Valid(content) {
		return ErrInvalidEncoding
	}

	return nil
}

// ParseHeader reads and parses the header row
func (p *CSVParser) ParseHeader() error {
	record, err := p.reader.Read()
	if err == io.EOF {
		return ErrMissingHeader
	}
	if err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	p.headers = make([]string, len(record))
	for i, h := range record {
		header := h
		if p.trimSpace {
			header = trimSpaces(header)
		}
		p.headers[i] = header
		p.headerMap[header] = i
	}

	if len(p.headers) == 0 {
		return ErrMissingHeader
	}

	p.currentRow = 1 // Header is row 1

	return nil
}

// Headers returns the parsed header names
func (p *CSVParser) Headers() []string {
	return p.headers
}

// HeaderMap returns a map of header name to column index
func (p *CSVParser) HeaderMap() map[string]int {
	return p.headerMap
}

// HasHeader checks if a header exists
func (p *CSVParser) HasHeader(name string) bool {
	_, ok := p.headerMap[name]
	return ok
}

// Row represents a parsed CSV row with its data and line number
type Row struct {
	LineNumber int
	Data       map[string]string
	RawFields  []string
}

// Get returns the value for a column by header name
func (r *Row) Get(header string) string {
	return r.Data[header]
}

// GetOrDefault returns the value for a column, or default if not present
func (r *Row) GetOrDefault(header, defaultVal string) string {
	if val, ok := r.Data[header]; ok && val != "" {
		return val
	}
	return defaultVal
}

// IsEmpty returns true if the row has no non-empty values
func (r *Row) IsEmpty() bool {
	for _, v := range r.Data {
		if v != "" {
			return false
		}
	}
	return true
}

// ReadRow reads the next row from the CSV
func (p *CSVParser) ReadRow() (*Row, error) {
	record, err := p.reader.Read()
	if err == io.EOF {
		return nil, io.EOF
	}
	if err != nil {
		p.currentRow++
		return nil, fmt.Errorf("error reading row %d: %w", p.currentRow, err)
	}

	p.currentRow++
	p.totalRows++

	row := &Row{
		LineNumber: p.currentRow,
		Data:       make(map[string]string, len(p.headers)),
		RawFields:  record,
	}

	// Map fields to headers
	for i, header := range p.headers {
		if i < len(record) {
			value := record[i]
			if p.trimSpace {
				value = trimSpaces(value)
			}
			row.Data[header] = value
		} else {
			row.Data[header] = ""
		}
	}

	return row, nil
}

// ReadAllRows reads all remaining rows from the CSV
func (p *CSVParser) ReadAllRows() ([]*Row, error) {
	var rows []*Row

	for {
		row, err := p.ReadRow()
		if err == io.EOF {
			break
		}
		if err != nil {
			return rows, err
		}

		// Skip completely empty rows
		if row.IsEmpty() {
			continue
		}

		rows = append(rows, row)
	}

	return rows, nil
}

// CurrentRow returns the current row number (1-indexed)
func (p *CSVParser) CurrentRow() int {
	return p.currentRow
}

// TotalRows returns the total number of data rows read
func (p *CSVParser) TotalRows() int {
	return p.totalRows
}

// ParseFromBytes creates a parser from a byte slice
func ParseFromBytes(data []byte, opts ...ParserOption) (*CSVParser, error) {
	return NewCSVParser(bytes.NewReader(data), opts...)
}

// trimSpaces trims whitespace from a string
func trimSpaces(s string) string {
	// Trim common whitespace characters
	start := 0
	end := len(s)

	for start < end {
		r, size := utf8.DecodeRuneInString(s[start:])
		if !isWhitespace(r) {
			break
		}
		start += size
	}

	for end > start {
		r, size := utf8.DecodeLastRuneInString(s[:end])
		if !isWhitespace(r) {
			break
		}
		end -= size
	}

	return s[start:end]
}

// isWhitespace checks if a rune is whitespace
func isWhitespace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	}
	return false
}

// ValidateHeaders checks if required headers are present
func (p *CSVParser) ValidateHeaders(required []string) []string {
	var missing []string
	for _, h := range required {
		if !p.HasHeader(h) {
			missing = append(missing, h)
		}
	}
	return missing
}

// GetColumnIndex returns the index of a column by name
func (p *CSVParser) GetColumnIndex(name string) (int, bool) {
	idx, ok := p.headerMap[name]
	return idx, ok
}
