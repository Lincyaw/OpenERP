package dto

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetHTTPStatus(t *testing.T) {
	tests := []struct {
		code     string
		expected int
	}{
		{ErrCodeUnknown, http.StatusInternalServerError},
		{ErrCodeInternal, http.StatusInternalServerError},
		{ErrCodeValidation, http.StatusBadRequest},
		{ErrCodeValidationRequired, http.StatusBadRequest},
		{ErrCodeUnauthorized, http.StatusUnauthorized},
		{ErrCodeForbidden, http.StatusForbidden},
		{ErrCodeTokenExpired, http.StatusUnauthorized},
		{ErrCodeNotFound, http.StatusNotFound},
		{ErrCodeAlreadyExists, http.StatusConflict},
		{ErrCodeConflict, http.StatusConflict},
		{ErrCodeConcurrencyConflict, http.StatusConflict},
		{ErrCodeInvalidState, http.StatusUnprocessableEntity},
		{ErrCodeBusinessRule, http.StatusUnprocessableEntity},
		{ErrCodeInsufficientStock, http.StatusUnprocessableEntity},
		{ErrCodeInsufficientBalance, http.StatusUnprocessableEntity},
		{ErrCodeBadRequest, http.StatusBadRequest},
		{ErrCodeInvalidInput, http.StatusBadRequest},
		{ErrCodeRateLimited, http.StatusTooManyRequests},
		// Unknown code should return 500
		{"UNKNOWN_CODE", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetHTTPStatus(tt.code))
		})
	}
}

func TestNormalizeErrorCode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Legacy codes should be normalized
		{"NOT_FOUND", ErrCodeNotFound},
		{"ALREADY_EXISTS", ErrCodeAlreadyExists},
		{"INVALID_INPUT", ErrCodeInvalidInput},
		{"INVALID_STATE", ErrCodeInvalidState},
		{"UNAUTHORIZED", ErrCodeUnauthorized},
		{"FORBIDDEN", ErrCodeForbidden},
		{"CONCURRENCY_CONFLICT", ErrCodeConcurrencyConflict},
		{"INSUFFICIENT_STOCK", ErrCodeInsufficientStock},
		{"INSUFFICIENT_BALANCE", ErrCodeInsufficientBalance},
		{"VALIDATION_ERROR", ErrCodeValidation},
		{"BAD_REQUEST", ErrCodeBadRequest},
		{"INTERNAL_ERROR", ErrCodeInternal},
		// New codes should pass through unchanged
		{ErrCodeNotFound, ErrCodeNotFound},
		{ErrCodeValidation, ErrCodeValidation},
		// Unknown codes should pass through unchanged
		{"CUSTOM_ERROR", "CUSTOM_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, NormalizeErrorCode(tt.input))
		})
	}
}

func TestErrorCodeConstants(t *testing.T) {
	// Ensure all error codes are in the HTTP status map
	allCodes := []string{
		ErrCodeUnknown,
		ErrCodeInternal,
		ErrCodeValidation,
		ErrCodeValidationRequired,
		ErrCodeValidationFormat,
		ErrCodeValidationRange,
		ErrCodeValidationLength,
		ErrCodeUnauthorized,
		ErrCodeForbidden,
		ErrCodeTokenExpired,
		ErrCodeTokenInvalid,
		ErrCodeNotFound,
		ErrCodeAlreadyExists,
		ErrCodeConflict,
		ErrCodeConcurrencyConflict,
		ErrCodeInvalidState,
		ErrCodeBusinessRule,
		ErrCodeInsufficientStock,
		ErrCodeInsufficientBalance,
		ErrCodeBadRequest,
		ErrCodeInvalidInput,
		ErrCodeInvalidJSON,
		ErrCodeRateLimited,
		ErrCodeTooManyRequests,
	}

	for _, code := range allCodes {
		t.Run(code, func(t *testing.T) {
			status, ok := ErrorCodeHTTPStatus[code]
			assert.True(t, ok, "Error code %s should be in ErrorCodeHTTPStatus map", code)
			assert.Greater(t, status, 0, "Status code should be positive")
		})
	}
}

func TestErrorCodeFormat(t *testing.T) {
	// All error codes should follow ERR_ prefix convention
	allCodes := []string{
		ErrCodeUnknown,
		ErrCodeInternal,
		ErrCodeValidation,
		ErrCodeUnauthorized,
		ErrCodeNotFound,
		ErrCodeInvalidState,
	}

	for _, code := range allCodes {
		t.Run(code, func(t *testing.T) {
			assert.Contains(t, code, "ERR_", "Error code should start with ERR_")
		})
	}
}

func TestNewErrorResponse(t *testing.T) {
	resp := NewErrorResponse("NOT_FOUND", "Resource not found")

	assert.False(t, resp.Success)
	assert.Nil(t, resp.Data)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, ErrCodeNotFound, resp.Error.Code) // Should be normalized
	assert.Equal(t, "Resource not found", resp.Error.Message)
	assert.NotZero(t, resp.Error.Timestamp)
}

func TestNewErrorResponseWithRequestID(t *testing.T) {
	requestID := "req-123-456"
	resp := NewErrorResponseWithRequestID(ErrCodeNotFound, "Resource not found", requestID)

	assert.False(t, resp.Success)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, ErrCodeNotFound, resp.Error.Code)
	assert.Equal(t, "Resource not found", resp.Error.Message)
	assert.Equal(t, requestID, resp.Error.RequestID)
	assert.NotZero(t, resp.Error.Timestamp)
}

func TestNewValidationErrorResponse(t *testing.T) {
	details := []ValidationDetail{
		{Field: "email", Message: "Invalid email format"},
		{Field: "age", Message: "Must be at least 18"},
	}
	requestID := "req-789"

	resp := NewValidationErrorResponse("Validation failed", requestID, details)

	assert.False(t, resp.Success)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, ErrCodeValidation, resp.Error.Code)
	assert.Equal(t, "Validation failed", resp.Error.Message)
	assert.Equal(t, requestID, resp.Error.RequestID)
	assert.Len(t, resp.Error.Details, 2)
	assert.Equal(t, "email", resp.Error.Details[0].Field)
	assert.Equal(t, "Invalid email format", resp.Error.Details[0].Message)
}

func TestNewErrorResponseWithHelp(t *testing.T) {
	help := "https://docs.example.com/errors/auth"
	resp := NewErrorResponseWithHelp(ErrCodeUnauthorized, "Not authenticated", "req-001", help)

	assert.False(t, resp.Success)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, ErrCodeUnauthorized, resp.Error.Code)
	assert.Equal(t, "Not authenticated", resp.Error.Message)
	assert.Equal(t, help, resp.Error.Help)
}

func TestErrorResponseJSON(t *testing.T) {
	resp := NewErrorResponseWithRequestID(ErrCodeNotFound, "User not found", "req-test-123")

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	// Unmarshal and verify structure
	var decoded Response
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)

	assert.False(t, decoded.Success)
	assert.NotNil(t, decoded.Error)
	assert.Equal(t, ErrCodeNotFound, decoded.Error.Code)
	assert.Equal(t, "User not found", decoded.Error.Message)
	assert.Equal(t, "req-test-123", decoded.Error.RequestID)
}

func TestErrorResponseTimestamp(t *testing.T) {
	before := time.Now()
	resp := NewErrorResponse(ErrCodeInternal, "Server error")
	after := time.Now()

	// Timestamp should be between before and after
	assert.True(t, !resp.Error.Timestamp.Before(before), "Timestamp should not be before call")
	assert.True(t, !resp.Error.Timestamp.After(after), "Timestamp should not be after call")
}

func TestNewSuccessResponse(t *testing.T) {
	data := map[string]string{"name": "test"}
	resp := NewSuccessResponse(data)

	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Data)
	assert.Nil(t, resp.Error)
	assert.Nil(t, resp.Meta)
}

func TestNewSuccessResponseWithMeta(t *testing.T) {
	data := []string{"item1", "item2"}
	resp := NewSuccessResponseWithMeta(data, 100, 1, 10)

	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Data)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Meta)
	assert.Equal(t, int64(100), resp.Meta.Total)
	assert.Equal(t, 1, resp.Meta.Page)
	assert.Equal(t, 10, resp.Meta.PageSize)
	assert.Equal(t, 10, resp.Meta.TotalPages) // 100 / 10 = 10
}

func TestNewSuccessResponseWithMetaPagination(t *testing.T) {
	tests := []struct {
		total         int64
		page          int
		pageSize      int
		expectedPages int
		expectedSize  int // Expected page size after validation
	}{
		{100, 1, 10, 10, 10},
		{101, 1, 10, 11, 10}, // Partial page
		{0, 1, 10, 0, 10},
		{9, 1, 10, 1, 10},
		{10, 1, 10, 1, 10},
		{11, 1, 10, 2, 10},
		// Edge case: zero pageSize should default to 20
		{100, 1, 0, 5, 20},
		{100, 1, -1, 5, 20},
	}

	for _, tt := range tests {
		resp := NewSuccessResponseWithMeta(nil, tt.total, tt.page, tt.pageSize)
		assert.Equal(t, tt.expectedPages, resp.Meta.TotalPages)
		assert.Equal(t, tt.expectedSize, resp.Meta.PageSize)
	}
}
