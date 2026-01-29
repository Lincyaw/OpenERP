package csvimport

import (
	"errors"
	"fmt"
	"strings"
)

// Import error codes
const (
	// General import errors
	ErrCodeImportUnknown      = "ERR_IMPORT_UNKNOWN"
	ErrCodeImportInvalidFile  = "ERR_IMPORT_INVALID_FILE"
	ErrCodeImportEmptyFile    = "ERR_IMPORT_EMPTY_FILE"
	ErrCodeImportFileTooLarge = "ERR_IMPORT_FILE_TOO_LARGE"

	// Encoding errors
	ErrCodeImportInvalidEncoding     = "ERR_IMPORT_INVALID_ENCODING"
	ErrCodeImportUnsupportedEncoding = "ERR_IMPORT_UNSUPPORTED_ENCODING"

	// CSV parsing errors
	ErrCodeImportCSVParsing    = "ERR_IMPORT_CSV_PARSING"
	ErrCodeImportMissingHeader = "ERR_IMPORT_MISSING_HEADER"
	ErrCodeImportInvalidHeader = "ERR_IMPORT_INVALID_HEADER"
	ErrCodeImportMalformedRow  = "ERR_IMPORT_MALFORMED_ROW"

	// Validation errors
	ErrCodeImportValidation        = "ERR_IMPORT_VALIDATION"
	ErrCodeImportRequiredField     = "ERR_IMPORT_REQUIRED_FIELD"
	ErrCodeImportInvalidType       = "ERR_IMPORT_INVALID_TYPE"
	ErrCodeImportInvalidFormat     = "ERR_IMPORT_INVALID_FORMAT"
	ErrCodeImportInvalidLength     = "ERR_IMPORT_INVALID_LENGTH"
	ErrCodeImportInvalidRange      = "ERR_IMPORT_INVALID_RANGE"
	ErrCodeImportPatternMismatch   = "ERR_IMPORT_PATTERN_MISMATCH"
	ErrCodeImportDuplicateInFile   = "ERR_IMPORT_DUPLICATE_IN_FILE"
	ErrCodeImportDuplicateInDB     = "ERR_IMPORT_DUPLICATE_IN_DB"
	ErrCodeImportReferenceNotFound = "ERR_IMPORT_REFERENCE_NOT_FOUND"
)

// Common import errors
var (
	// ErrEmptyFile is returned when the CSV file is empty
	ErrEmptyFile = errors.New("CSV file is empty")

	// ErrInvalidEncoding is returned when the file encoding cannot be detected
	ErrInvalidEncoding = errors.New("invalid file encoding")

	// ErrMissingHeader is returned when the CSV file has no header row
	ErrMissingHeader = errors.New("CSV file missing header row")

	// ErrNoDataRows is returned when the CSV file has no data rows
	ErrNoDataRows = errors.New("CSV file contains no data rows")

	// ErrFileTooLarge is returned when the file exceeds maximum size
	ErrFileTooLarge = errors.New("file exceeds maximum allowed size")

	// ErrUnsupportedEntityType is returned when entity type is not supported
	ErrUnsupportedEntityType = errors.New("unsupported entity type")
)

// RowError represents an error in a specific row
type RowError struct {
	Row     int    `json:"row"`
	Column  string `json:"column"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

// Error implements the error interface
func (e RowError) Error() string {
	if e.Column != "" {
		return fmt.Sprintf("row %d, column '%s': %s", e.Row, e.Column, e.Message)
	}
	return fmt.Sprintf("row %d: %s", e.Row, e.Message)
}

// NewRowError creates a new RowError
func NewRowError(row int, column, code, message string) RowError {
	return RowError{
		Row:     row,
		Column:  column,
		Code:    code,
		Message: message,
	}
}

// NewRowErrorWithValue creates a new RowError with the invalid value
func NewRowErrorWithValue(row int, column, code, message, value string) RowError {
	return RowError{
		Row:     row,
		Column:  column,
		Code:    code,
		Message: message,
		Value:   value,
	}
}

// ErrorCollection manages a collection of import errors
type ErrorCollection struct {
	errors     []RowError
	maxErrors  int
	totalCount int
}

// NewErrorCollection creates a new ErrorCollection with a maximum error limit
func NewErrorCollection(maxErrors int) *ErrorCollection {
	if maxErrors <= 0 {
		maxErrors = 100 // Default limit
	}
	return &ErrorCollection{
		errors:    make([]RowError, 0, maxErrors),
		maxErrors: maxErrors,
	}
}

// Add adds an error to the collection
func (ec *ErrorCollection) Add(err RowError) {
	ec.totalCount++
	if len(ec.errors) < ec.maxErrors {
		ec.errors = append(ec.errors, err)
	}
}

// AddValidationError adds a validation error for a specific field
func (ec *ErrorCollection) AddValidationError(row int, column, code, message string) {
	ec.Add(NewRowError(row, column, code, message))
}

// AddRequiredError adds a required field error
func (ec *ErrorCollection) AddRequiredError(row int, column string) {
	ec.Add(NewRowError(row, column, ErrCodeImportRequiredField, fmt.Sprintf("field '%s' is required", column)))
}

// AddTypeError adds a type validation error
func (ec *ErrorCollection) AddTypeError(row int, column, expectedType, value string) {
	ec.Add(NewRowErrorWithValue(row, column, ErrCodeImportInvalidType,
		fmt.Sprintf("expected %s", expectedType), value))
}

// AddFormatError adds a format validation error
func (ec *ErrorCollection) AddFormatError(row int, column, expectedFormat, value string) {
	ec.Add(NewRowErrorWithValue(row, column, ErrCodeImportInvalidFormat,
		fmt.Sprintf("invalid format, expected %s", expectedFormat), value))
}

// AddLengthError adds a length validation error
func (ec *ErrorCollection) AddLengthError(row int, column string, minLen, maxLen int) {
	msg := fmt.Sprintf("length must be between %d and %d", minLen, maxLen)
	if minLen == 0 {
		msg = fmt.Sprintf("length must be at most %d", maxLen)
	}
	if maxLen == 0 || maxLen == int(^uint(0)>>1) {
		msg = fmt.Sprintf("length must be at least %d", minLen)
	}
	ec.Add(NewRowError(row, column, ErrCodeImportInvalidLength, msg))
}

// AddRangeError adds a range validation error
func (ec *ErrorCollection) AddRangeError(row int, column string, min, max float64) {
	ec.Add(NewRowError(row, column, ErrCodeImportInvalidRange,
		fmt.Sprintf("value must be between %.2f and %.2f", min, max)))
}

// AddPatternError adds a pattern mismatch error
func (ec *ErrorCollection) AddPatternError(row int, column, pattern, value string) {
	ec.Add(NewRowErrorWithValue(row, column, ErrCodeImportPatternMismatch,
		fmt.Sprintf("value does not match pattern '%s'", pattern), value))
}

// AddDuplicateError adds a duplicate value error
func (ec *ErrorCollection) AddDuplicateError(row int, column, value string, inDB bool) {
	code := ErrCodeImportDuplicateInFile
	msg := fmt.Sprintf("duplicate value '%s' found in file", value)
	if inDB {
		code = ErrCodeImportDuplicateInDB
		msg = fmt.Sprintf("value '%s' already exists in database", value)
	}
	ec.Add(NewRowErrorWithValue(row, column, code, msg, value))
}

// AddReferenceError adds a reference not found error
func (ec *ErrorCollection) AddReferenceError(row int, column, value, refType string) {
	ec.Add(NewRowErrorWithValue(row, column, ErrCodeImportReferenceNotFound,
		fmt.Sprintf("%s '%s' not found", refType, value), value))
}

// Errors returns the collected errors
func (ec *ErrorCollection) Errors() []RowError {
	return ec.errors
}

// Count returns the number of collected errors (up to maxErrors)
func (ec *ErrorCollection) Count() int {
	return len(ec.errors)
}

// TotalCount returns the total number of errors including those not collected
func (ec *ErrorCollection) TotalCount() int {
	return ec.totalCount
}

// HasErrors returns true if there are any errors
func (ec *ErrorCollection) HasErrors() bool {
	return ec.totalCount > 0
}

// IsTruncated returns true if some errors were not collected due to the limit
func (ec *ErrorCollection) IsTruncated() bool {
	return ec.totalCount > ec.maxErrors
}

// Clear clears all errors
func (ec *ErrorCollection) Clear() {
	ec.errors = ec.errors[:0]
	ec.totalCount = 0
}

// ErrorSummary returns a summary of errors by code
func (ec *ErrorCollection) ErrorSummary() map[string]int {
	summary := make(map[string]int)
	for _, err := range ec.errors {
		summary[err.Code]++
	}
	return summary
}

// String returns a string representation of all errors
func (ec *ErrorCollection) String() string {
	if !ec.HasErrors() {
		return "no errors"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d error(s) found", ec.totalCount))
	if ec.IsTruncated() {
		sb.WriteString(fmt.Sprintf(" (showing first %d)", ec.maxErrors))
	}
	sb.WriteString(":\n")

	for _, err := range ec.errors {
		sb.WriteString(fmt.Sprintf("  - %s\n", err.Error()))
	}

	return sb.String()
}

// ValidationResult represents the result of validating a CSV file
type ValidationResult struct {
	ValidationID string           `json:"validation_id"`
	TotalRows    int              `json:"total_rows"`
	ValidRows    int              `json:"valid_rows"`
	ErrorRows    int              `json:"error_rows"`
	Errors       []RowError       `json:"errors,omitempty"`
	Preview      []map[string]any `json:"preview,omitempty"`
	IsTruncated  bool             `json:"is_truncated,omitempty"`
	TotalErrors  int              `json:"total_errors,omitempty"`
}

// NewValidationResult creates a new ValidationResult
func NewValidationResult(validationID string) *ValidationResult {
	return &ValidationResult{
		ValidationID: validationID,
		Errors:       make([]RowError, 0),
		Preview:      make([]map[string]any, 0),
	}
}

// SetCounts sets the row counts
func (vr *ValidationResult) SetCounts(total, valid, errorCount int) {
	vr.TotalRows = total
	vr.ValidRows = valid
	vr.ErrorRows = errorCount
}

// AddPreview adds a preview row
func (vr *ValidationResult) AddPreview(row map[string]any) {
	if len(vr.Preview) < 5 {
		vr.Preview = append(vr.Preview, row)
	}
}

// SetErrors sets the errors from an ErrorCollection
func (vr *ValidationResult) SetErrors(ec *ErrorCollection) {
	vr.Errors = ec.Errors()
	vr.IsTruncated = ec.IsTruncated()
	vr.TotalErrors = ec.TotalCount()
}

// IsValid returns true if there are no errors
func (vr *ValidationResult) IsValid() bool {
	return vr.ErrorRows == 0
}
