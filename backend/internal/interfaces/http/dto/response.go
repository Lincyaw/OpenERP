package dto

import "time"

// Response represents a standard API response
type Response struct {
	Success bool       `json:"success"`
	Data    any        `json:"data,omitempty"`
	Error   *ErrorInfo `json:"error,omitempty"`
	Meta    *Meta      `json:"meta,omitempty"`
}

// ErrorInfo represents error details in API responses
type ErrorInfo struct {
	Code      string             `json:"code"`
	Message   string             `json:"message"`
	RequestID string             `json:"request_id,omitempty"`
	Timestamp time.Time          `json:"timestamp,omitempty"`
	Details   []ValidationDetail `json:"details,omitempty"`
	Help      string             `json:"help,omitempty"`
}

// ValidationDetail represents a single validation error detail
type ValidationDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Meta represents pagination metadata
type Meta struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
}

// NewSuccessResponse creates a success response
func NewSuccessResponse(data any) Response {
	return Response{
		Success: true,
		Data:    data,
	}
}

// NewSuccessResponseWithMeta creates a success response with pagination meta
func NewSuccessResponseWithMeta(data any, total int64, page, pageSize int) Response {
	if pageSize <= 0 {
		pageSize = 20 // Default page size to prevent division by zero
	}
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	return Response{
		Success: true,
		Data:    data,
		Meta: &Meta{
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
	}
}

// NewErrorResponse creates an error response with code and message
func NewErrorResponse(code, message string) Response {
	return Response{
		Success: false,
		Error: &ErrorInfo{
			Code:      NormalizeErrorCode(code),
			Message:   message,
			Timestamp: time.Now(),
		},
	}
}

// NewErrorResponseWithRequestID creates an error response with request ID
func NewErrorResponseWithRequestID(code, message, requestID string) Response {
	return Response{
		Success: false,
		Error: &ErrorInfo{
			Code:      NormalizeErrorCode(code),
			Message:   message,
			RequestID: requestID,
			Timestamp: time.Now(),
		},
	}
}

// NewValidationErrorResponse creates a validation error response with field details
func NewValidationErrorResponse(message string, requestID string, details []ValidationDetail) Response {
	return Response{
		Success: false,
		Error: &ErrorInfo{
			Code:      ErrCodeValidation,
			Message:   message,
			RequestID: requestID,
			Timestamp: time.Now(),
			Details:   details,
		},
	}
}

// NewErrorResponseWithDetails creates an error response with additional details
func NewErrorResponseWithDetails(code, message, requestID string, details []ValidationDetail) Response {
	return Response{
		Success: false,
		Error: &ErrorInfo{
			Code:      NormalizeErrorCode(code),
			Message:   message,
			RequestID: requestID,
			Timestamp: time.Now(),
			Details:   details,
		},
	}
}

// NewErrorResponseWithHelp creates an error response with a help URL
func NewErrorResponseWithHelp(code, message, requestID, help string) Response {
	return Response{
		Success: false,
		Error: &ErrorInfo{
			Code:      NormalizeErrorCode(code),
			Message:   message,
			RequestID: requestID,
			Timestamp: time.Now(),
			Help:      help,
		},
	}
}

// ListRequest represents common list/pagination request parameters
type ListRequest struct {
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size" binding:"min=1,max=100"`
	OrderBy  string `form:"order_by"`
	OrderDir string `form:"order_dir" binding:"omitempty,oneof=asc desc"`
	Search   string `form:"search"`
}

// DefaultListRequest returns a list request with defaults
func DefaultListRequest() ListRequest {
	return ListRequest{
		Page:     1,
		PageSize: 20,
		OrderBy:  "created_at",
		OrderDir: "desc",
	}
}

// IDRequest represents a request with an ID path parameter
type IDRequest struct {
	ID string `uri:"id" binding:"required,uuid"`
}

// TimestampResponse represents timestamps in response
type TimestampResponse struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MessageResponse represents a simple message response
type MessageResponse struct {
	Message string `json:"message"`
}
