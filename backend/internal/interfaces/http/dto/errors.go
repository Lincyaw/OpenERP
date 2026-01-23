package dto

import "net/http"

// Error code constants organized by category
// Format: ERR_<CATEGORY>_<DESCRIPTION>

// General error codes
const (
	// ErrCodeUnknown is used when the error type is unknown
	ErrCodeUnknown = "ERR_UNKNOWN"
	// ErrCodeInternal is used for internal server errors
	ErrCodeInternal = "ERR_INTERNAL"
)

// Validation error codes
const (
	// ErrCodeValidation is the base code for validation errors
	ErrCodeValidation = "ERR_VALIDATION"
	// ErrCodeValidationRequired is used when a required field is missing
	ErrCodeValidationRequired = "ERR_VALIDATION_REQUIRED"
	// ErrCodeValidationFormat is used when a field has invalid format
	ErrCodeValidationFormat = "ERR_VALIDATION_FORMAT"
	// ErrCodeValidationRange is used when a value is out of range
	ErrCodeValidationRange = "ERR_VALIDATION_RANGE"
	// ErrCodeValidationLength is used when a field length is invalid
	ErrCodeValidationLength = "ERR_VALIDATION_LENGTH"
)

// Authentication error codes
const (
	// ErrCodeUnauthorized is used when authentication is required but missing/invalid
	ErrCodeUnauthorized = "ERR_UNAUTHORIZED"
	// ErrCodeForbidden is used when the user lacks permission
	ErrCodeForbidden = "ERR_FORBIDDEN"
	// ErrCodeTokenExpired is used when the auth token has expired
	ErrCodeTokenExpired = "ERR_TOKEN_EXPIRED"
	// ErrCodeTokenInvalid is used when the auth token is invalid
	ErrCodeTokenInvalid = "ERR_TOKEN_INVALID"
)

// Resource error codes
const (
	// ErrCodeNotFound is used when a resource is not found
	ErrCodeNotFound = "ERR_NOT_FOUND"
	// ErrCodeAlreadyExists is used when trying to create a duplicate resource
	ErrCodeAlreadyExists = "ERR_ALREADY_EXISTS"
	// ErrCodeConflict is used for general resource conflicts
	ErrCodeConflict = "ERR_CONFLICT"
	// ErrCodeConcurrencyConflict is used when optimistic locking fails
	ErrCodeConcurrencyConflict = "ERR_CONCURRENCY_CONFLICT"
)

// Business rule error codes
const (
	// ErrCodeInvalidState is used when an operation is invalid for current state
	ErrCodeInvalidState = "ERR_INVALID_STATE"
	// ErrCodeBusinessRule is used for generic business rule violations
	ErrCodeBusinessRule = "ERR_BUSINESS_RULE"
	// ErrCodeInsufficientStock is used when stock is insufficient
	ErrCodeInsufficientStock = "ERR_INSUFFICIENT_STOCK"
	// ErrCodeInsufficientBalance is used when balance is insufficient
	ErrCodeInsufficientBalance = "ERR_INSUFFICIENT_BALANCE"
)

// Input error codes
const (
	// ErrCodeBadRequest is used for malformed requests
	ErrCodeBadRequest = "ERR_BAD_REQUEST"
	// ErrCodeInvalidInput is used for invalid input data
	ErrCodeInvalidInput = "ERR_INVALID_INPUT"
	// ErrCodeInvalidJSON is used when JSON parsing fails
	ErrCodeInvalidJSON = "ERR_INVALID_JSON"
)

// Rate limiting error codes
const (
	// ErrCodeRateLimited is used when rate limit is exceeded
	ErrCodeRateLimited = "ERR_RATE_LIMITED"
	// ErrCodeTooManyRequests is an alias for rate limiting
	ErrCodeTooManyRequests = "ERR_TOO_MANY_REQUESTS"
)

// ErrorCodeHTTPStatus maps error codes to HTTP status codes
var ErrorCodeHTTPStatus = map[string]int{
	// General errors
	ErrCodeUnknown:  http.StatusInternalServerError,
	ErrCodeInternal: http.StatusInternalServerError,

	// Validation errors -> 400 Bad Request
	ErrCodeValidation:         http.StatusBadRequest,
	ErrCodeValidationRequired: http.StatusBadRequest,
	ErrCodeValidationFormat:   http.StatusBadRequest,
	ErrCodeValidationRange:    http.StatusBadRequest,
	ErrCodeValidationLength:   http.StatusBadRequest,

	// Auth errors
	ErrCodeUnauthorized: http.StatusUnauthorized,
	ErrCodeForbidden:    http.StatusForbidden,
	ErrCodeTokenExpired: http.StatusUnauthorized,
	ErrCodeTokenInvalid: http.StatusUnauthorized,

	// Resource errors
	ErrCodeNotFound:            http.StatusNotFound,
	ErrCodeAlreadyExists:       http.StatusConflict,
	ErrCodeConflict:            http.StatusConflict,
	ErrCodeConcurrencyConflict: http.StatusConflict,

	// Business rule errors -> 422 Unprocessable Entity
	ErrCodeInvalidState:        http.StatusUnprocessableEntity,
	ErrCodeBusinessRule:        http.StatusUnprocessableEntity,
	ErrCodeInsufficientStock:   http.StatusUnprocessableEntity,
	ErrCodeInsufficientBalance: http.StatusUnprocessableEntity,

	// Input errors -> 400 Bad Request
	ErrCodeBadRequest:   http.StatusBadRequest,
	ErrCodeInvalidInput: http.StatusBadRequest,
	ErrCodeInvalidJSON:  http.StatusBadRequest,

	// Rate limiting -> 429 Too Many Requests
	ErrCodeRateLimited:     http.StatusTooManyRequests,
	ErrCodeTooManyRequests: http.StatusTooManyRequests,
}

// GetHTTPStatus returns the HTTP status code for an error code
// Returns 500 Internal Server Error if the error code is not found
func GetHTTPStatus(code string) int {
	if status, ok := ErrorCodeHTTPStatus[code]; ok {
		return status
	}
	return http.StatusInternalServerError
}

// LegacyErrorCodeMapping maps old error codes to new standardized codes
// This is for backward compatibility with existing domain errors
var LegacyErrorCodeMapping = map[string]string{
	"NOT_FOUND":            ErrCodeNotFound,
	"ALREADY_EXISTS":       ErrCodeAlreadyExists,
	"INVALID_INPUT":        ErrCodeInvalidInput,
	"INVALID_STATE":        ErrCodeInvalidState,
	"UNAUTHORIZED":         ErrCodeUnauthorized,
	"FORBIDDEN":            ErrCodeForbidden,
	"CONCURRENCY_CONFLICT": ErrCodeConcurrencyConflict,
	"INSUFFICIENT_STOCK":   ErrCodeInsufficientStock,
	"INSUFFICIENT_BALANCE": ErrCodeInsufficientBalance,
	"VALIDATION_ERROR":     ErrCodeValidation,
	"BAD_REQUEST":          ErrCodeBadRequest,
	"INTERNAL_ERROR":       ErrCodeInternal,
}

// NormalizeErrorCode converts a legacy error code to the standardized format
// If the code is already in the new format or unknown, returns it as-is
func NormalizeErrorCode(code string) string {
	if newCode, ok := LegacyErrorCodeMapping[code]; ok {
		return newCode
	}
	return code
}
