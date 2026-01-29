package dto

import csvimport "github.com/erp/backend/internal/infrastructure/import"

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
