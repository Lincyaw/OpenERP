package handler

import "github.com/erp/backend/internal/interfaces/http/dto"

// APIResponse represents a generic API response for OpenAPI documentation
// @Description Standard API response wrapper with typed data field
type APIResponse[T any] struct {
	Success bool           `json:"success"`
	Data    T              `json:"data,omitempty"`
	Error   *dto.ErrorInfo `json:"error,omitempty"`
	Meta    *dto.Meta      `json:"meta,omitempty"`
}

// ErrorResponse represents an error API response for OpenAPI documentation
// @Description Standard error response
type ErrorResponse struct {
	Success bool           `json:"success" example:"false"`
	Error   *dto.ErrorInfo `json:"error,omitempty"`
}

// SuccessResponse represents a simple success API response for OpenAPI documentation
// @Description Simple success response without data
type SuccessResponse struct {
	Success bool `json:"success" example:"true"`
}

// BalanceData represents balance data in response
// @Description Balance data
type BalanceData struct {
	Balance float64 `json:"balance"`
}

// ContentData represents content data in response
// @Description Content data
type ContentData struct {
	Content string `json:"content"`
}

// CountData represents count data in response
// @Description Count data
type CountData struct {
	Count int64 `json:"count"`
}

// SchedulerStatusData represents scheduler status data
// @Description Scheduler status information
type SchedulerStatusData struct {
	Enabled        bool     `json:"enabled"`
	AvailableTypes []string `json:"available_types"`
}
