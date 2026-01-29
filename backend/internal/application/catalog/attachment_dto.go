package catalog

import (
	"time"

	"github.com/erp/backend/internal/domain/catalog"
	"github.com/google/uuid"
)

// ============================================================================
// Request DTOs
// ============================================================================

// InitiateUploadRequest represents a request to initiate a file upload
type InitiateUploadRequest struct {
	ProductID   uuid.UUID `json:"product_id" binding:"required"`
	Type        string    `json:"type" binding:"required,oneof=main_image gallery_image document other"`
	FileName    string    `json:"file_name" binding:"required,min=1,max=255"`
	FileSize    int64     `json:"file_size" binding:"required,gt=0"`
	ContentType string    `json:"content_type" binding:"required"`
}

// ConfirmUploadRequest represents a request to confirm a file upload
type ConfirmUploadRequest struct {
	AttachmentID uuid.UUID `json:"attachment_id" binding:"required"`
}

// SetMainImageRequest represents a request to set an attachment as main image
type SetMainImageRequest struct {
	AttachmentID uuid.UUID `json:"attachment_id" binding:"required"`
}

// DeleteAttachmentRequest represents a request to delete an attachment
type DeleteAttachmentRequest struct {
	AttachmentID uuid.UUID `json:"attachment_id" binding:"required"`
}

// ReorderAttachmentsRequest represents a request to reorder attachments
type ReorderAttachmentsRequest struct {
	ProductID     uuid.UUID   `json:"product_id" binding:"required"`
	AttachmentIDs []uuid.UUID `json:"attachment_ids" binding:"required,min=1"`
}

// AttachmentListFilter represents filter options for attachment list
type AttachmentListFilter struct {
	Search   string `form:"search"`
	Status   string `form:"status" binding:"omitempty,oneof=pending active deleted"`
	Type     string `form:"type" binding:"omitempty,oneof=main_image gallery_image document other"`
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	OrderBy  string `form:"order_by"`
	OrderDir string `form:"order_dir" binding:"omitempty,oneof=asc desc"`
}

// ============================================================================
// Response DTOs
// ============================================================================

// InitiateUploadResponse represents the response from initiating an upload
type InitiateUploadResponse struct {
	AttachmentID uuid.UUID `json:"attachment_id"`
	UploadURL    string    `json:"upload_url"`
	StorageKey   string    `json:"storage_key"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// AttachmentResponse represents an attachment in API responses
type AttachmentResponse struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	ProductID    uuid.UUID  `json:"product_id"`
	Type         string     `json:"type"`
	Status       string     `json:"status"`
	FileName     string     `json:"file_name"`
	FileSize     int64      `json:"file_size"`
	ContentType  string     `json:"content_type"`
	StorageKey   string     `json:"storage_key"`
	ThumbnailKey string     `json:"thumbnail_key,omitempty"`
	SortOrder    int        `json:"sort_order"`
	UploadedBy   *uuid.UUID `json:"uploaded_by,omitempty"`
	URL          string     `json:"url,omitempty"`
	ThumbnailURL string     `json:"thumbnail_url,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	Version      int        `json:"version"`
}

// AttachmentListResponse represents a list item for attachments
type AttachmentListResponse struct {
	ID           uuid.UUID  `json:"id"`
	ProductID    uuid.UUID  `json:"product_id"`
	Type         string     `json:"type"`
	Status       string     `json:"status"`
	FileName     string     `json:"file_name"`
	FileSize     int64      `json:"file_size"`
	ContentType  string     `json:"content_type"`
	SortOrder    int        `json:"sort_order"`
	UploadedBy   *uuid.UUID `json:"uploaded_by,omitempty"`
	URL          string     `json:"url,omitempty"`
	ThumbnailURL string     `json:"thumbnail_url,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// ============================================================================
// Conversion Functions
// ============================================================================

// ToAttachmentResponse converts a domain ProductAttachment to AttachmentResponse
func ToAttachmentResponse(a *catalog.ProductAttachment) AttachmentResponse {
	return AttachmentResponse{
		ID:           a.ID,
		TenantID:     a.TenantID,
		ProductID:    a.ProductID,
		Type:         string(a.Type),
		Status:       string(a.Status),
		FileName:     a.FileName,
		FileSize:     a.FileSize,
		ContentType:  a.ContentType,
		StorageKey:   a.StorageKey,
		ThumbnailKey: a.ThumbnailKey,
		SortOrder:    a.SortOrder,
		UploadedBy:   a.UploadedBy,
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
		Version:      a.Version,
	}
}

// ToAttachmentListResponse converts a domain ProductAttachment to AttachmentListResponse
func ToAttachmentListResponse(a *catalog.ProductAttachment) AttachmentListResponse {
	return AttachmentListResponse{
		ID:          a.ID,
		ProductID:   a.ProductID,
		Type:        string(a.Type),
		Status:      string(a.Status),
		FileName:    a.FileName,
		FileSize:    a.FileSize,
		ContentType: a.ContentType,
		SortOrder:   a.SortOrder,
		UploadedBy:  a.UploadedBy,
		CreatedAt:   a.CreatedAt,
	}
}

// ToAttachmentListResponses converts a slice of domain ProductAttachments to AttachmentListResponses
func ToAttachmentListResponses(attachments []catalog.ProductAttachment) []AttachmentListResponse {
	responses := make([]AttachmentListResponse, len(attachments))
	for i, a := range attachments {
		responses[i] = ToAttachmentListResponse(&a)
	}
	return responses
}

// ToAttachmentResponses converts a slice of domain ProductAttachments to AttachmentResponses
func ToAttachmentResponses(attachments []catalog.ProductAttachment) []AttachmentResponse {
	responses := make([]AttachmentResponse, len(attachments))
	for i, a := range attachments {
		responses[i] = ToAttachmentResponse(&a)
	}
	return responses
}

// EnrichWithURLs enriches an AttachmentResponse with URLs from the storage service
func (r *AttachmentResponse) EnrichWithURLs(url, thumbnailURL string) {
	r.URL = url
	r.ThumbnailURL = thumbnailURL
}

// EnrichWithURLs enriches an AttachmentListResponse with URLs from the storage service
func (r *AttachmentListResponse) EnrichWithURLs(url, thumbnailURL string) {
	r.URL = url
	r.ThumbnailURL = thumbnailURL
}
