package dto

import (
	"time"

	"github.com/erp/backend/internal/domain/bulk"
	csvimport "github.com/erp/backend/internal/infrastructure/import"
)

// ProductImportValidateRequest represents the request for product import validation
type ProductImportValidateRequest struct {
	EntityType string `form:"entity_type" binding:"required"`
}

// ProductImportRequest represents the request to import products
type ProductImportRequest struct {
	ValidationID string `json:"validation_id" binding:"required,uuid"`
	ConflictMode string `json:"conflict_mode" binding:"required,oneof=skip update fail"`
}

// ProductImportResponse represents the response from product import
// @Description Response from product bulk import operation
type ProductImportResponse struct {
	TotalRows    int                  `json:"total_rows" example:"100"`
	ImportedRows int                  `json:"imported_rows" example:"95"`
	UpdatedRows  int                  `json:"updated_rows" example:"3"`
	SkippedRows  int                  `json:"skipped_rows" example:"2"`
	ErrorRows    int                  `json:"error_rows" example:"0"`
	Errors       []csvimport.RowError `json:"errors,omitempty"`
	IsTruncated  bool                 `json:"is_truncated,omitempty" example:"false"`
	TotalErrors  int                  `json:"total_errors,omitempty" example:"0"`
}

// ProductImportValidateResponse represents the response from product import validation
// @Description Response from product CSV validation
type ProductImportValidateResponse struct {
	ValidationID string               `json:"validation_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TotalRows    int                  `json:"total_rows" example:"100"`
	ValidRows    int                  `json:"valid_rows" example:"98"`
	ErrorRows    int                  `json:"error_rows" example:"2"`
	Errors       []csvimport.RowError `json:"errors,omitempty"`
	Preview      []map[string]any     `json:"preview,omitempty"`
	Warnings     []string             `json:"warnings,omitempty"`
	IsTruncated  bool                 `json:"is_truncated,omitempty"`
	TotalErrors  int                  `json:"total_errors,omitempty"`
}

// ==================== Import History DTOs ====================

// ImportHistoryResponse represents a single import history record
// @Description Response for import history record
type ImportHistoryResponse struct {
	ID           string                   `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	EntityType   string                   `json:"entity_type" example:"products"`
	FileName     string                   `json:"file_name" example:"products_2024.csv"`
	FileSize     int64                    `json:"file_size" example:"10240"`
	TotalRows    int                      `json:"total_rows" example:"100"`
	SuccessRows  int                      `json:"success_rows" example:"95"`
	ErrorRows    int                      `json:"error_rows" example:"3"`
	SkippedRows  int                      `json:"skipped_rows" example:"2"`
	UpdatedRows  int                      `json:"updated_rows" example:"0"`
	ConflictMode string                   `json:"conflict_mode" example:"skip"`
	Status       string                   `json:"status" example:"completed"`
	ErrorDetails []bulk.ImportErrorDetail `json:"error_details,omitempty"`
	ImportedBy   string                   `json:"imported_by,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	StartedAt    *time.Time               `json:"started_at,omitempty" example:"2024-01-15T10:30:00Z"`
	CompletedAt  *time.Time               `json:"completed_at,omitempty" example:"2024-01-15T10:31:00Z"`
	CreatedAt    time.Time                `json:"created_at" example:"2024-01-15T10:29:00Z"`
	SuccessRate  float64                  `json:"success_rate" example:"95.0"`
}

// ImportHistoryListResponse represents a paginated list of import histories
// @Description Paginated list of import history records
type ImportHistoryListResponse struct {
	Items      []ImportHistoryResponse `json:"items"`
	TotalCount int64                   `json:"total_count" example:"50"`
	Page       int                     `json:"page" example:"1"`
	PageSize   int                     `json:"page_size" example:"20"`
	TotalPages int                     `json:"total_pages" example:"3"`
}

// ImportHistoryListRequest represents the query parameters for listing import histories
type ImportHistoryListRequest struct {
	EntityType  string `form:"entity_type" binding:"omitempty,oneof=products customers suppliers inventory categories"`
	Status      string `form:"status" binding:"omitempty,oneof=pending processing completed failed cancelled"`
	StartedFrom string `form:"started_from" binding:"omitempty,datetime=2006-01-02"`
	StartedTo   string `form:"started_to" binding:"omitempty,datetime=2006-01-02"`
	Page        int    `form:"page,default=1" binding:"omitempty,min=1"`
	PageSize    int    `form:"page_size,default=20" binding:"omitempty,min=1,max=100"`
}

// NewImportHistoryResponse converts a domain ImportHistory to a response DTO
func NewImportHistoryResponse(h *bulk.ImportHistory) ImportHistoryResponse {
	response := ImportHistoryResponse{
		ID:           h.ID.String(),
		EntityType:   string(h.EntityType),
		FileName:     h.FileName,
		FileSize:     h.FileSize,
		TotalRows:    h.TotalRows,
		SuccessRows:  h.SuccessRows,
		ErrorRows:    h.ErrorRows,
		SkippedRows:  h.SkippedRows,
		UpdatedRows:  h.UpdatedRows,
		ConflictMode: string(h.ConflictMode),
		Status:       string(h.Status),
		ErrorDetails: h.ErrorDetails,
		StartedAt:    h.StartedAt,
		CompletedAt:  h.CompletedAt,
		CreatedAt:    h.CreatedAt,
		SuccessRate:  h.SuccessRate(),
	}

	if h.ImportedBy != nil {
		response.ImportedBy = h.ImportedBy.String()
	}

	return response
}

// NewImportHistoryListResponse converts a list of import histories to a response DTO
func NewImportHistoryListResponse(result *bulk.ImportHistoryListResult) ImportHistoryListResponse {
	items := make([]ImportHistoryResponse, len(result.Items))
	for i, h := range result.Items {
		items[i] = NewImportHistoryResponse(h)
	}

	totalPages := 0
	if result.PageSize > 0 {
		totalPages = int((result.TotalCount + int64(result.PageSize) - 1) / int64(result.PageSize))
	}

	return ImportHistoryListResponse{
		Items:      items,
		TotalCount: result.TotalCount,
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalPages: totalPages,
	}
}
