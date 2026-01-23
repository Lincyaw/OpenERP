package dto

import "time"

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// ErrorInfo represents error details
type ErrorInfo struct {
	Code    string `json:"code"`
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
func NewSuccessResponse(data interface{}) Response {
	return Response{
		Success: true,
		Data:    data,
	}
}

// NewSuccessResponseWithMeta creates a success response with pagination meta
func NewSuccessResponseWithMeta(data interface{}, total int64, page, pageSize int) Response {
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

// NewErrorResponse creates an error response
func NewErrorResponse(code, message string) Response {
	return Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
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
