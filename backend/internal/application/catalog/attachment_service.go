package catalog

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/catalog"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// AllowedContentTypes defines the whitelist of allowed content types for uploads
// This prevents uploading potentially dangerous file types (executables, scripts, etc.)
// SECURITY: SVG files are explicitly NOT allowed due to XSS risk (can contain <script> tags
// and inline event handlers like onload, onerror, etc.)
var AllowedContentTypes = map[string]bool{
	// Images (SVG excluded - XSS risk)
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
	"image/bmp":  true,
	"image/tiff": true,
	// Documents
	"application/pdf":    true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/vnd.ms-excel": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true,
	"application/vnd.ms-powerpoint":                                             true,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
	// Text
	"text/plain": true,
	"text/csv":   true,
	// Archives (for bundled documentation)
	"application/zip": true,
}

// ObjectStorageService defines the interface for object storage operations
// This interface will be implemented by the infrastructure layer (S3, RustFS, etc.)
type ObjectStorageService interface {
	// GenerateUploadURL generates a presigned URL for uploading a file
	// Returns the upload URL and expiration time
	GenerateUploadURL(ctx context.Context, storageKey, contentType string, expiresIn time.Duration) (string, time.Time, error)

	// GenerateDownloadURL generates a presigned URL for downloading a file
	// Returns the download URL and expiration time
	GenerateDownloadURL(ctx context.Context, storageKey string, expiresIn time.Duration) (string, time.Time, error)

	// DeleteObject deletes an object from storage
	DeleteObject(ctx context.Context, storageKey string) error

	// ObjectExists checks if an object exists in storage
	ObjectExists(ctx context.Context, storageKey string) (bool, error)
}

// AttachmentServiceConfig holds configuration for the attachment service
type AttachmentServiceConfig struct {
	// UploadURLExpiry is the duration for which upload URLs are valid
	UploadURLExpiry time.Duration
	// DownloadURLExpiry is the duration for which download URLs are valid
	DownloadURLExpiry time.Duration
	// MaxAttachmentsPerProduct is the maximum number of attachments per product
	MaxAttachmentsPerProduct int
}

// DefaultAttachmentServiceConfig returns the default configuration
func DefaultAttachmentServiceConfig() AttachmentServiceConfig {
	return AttachmentServiceConfig{
		UploadURLExpiry:          15 * time.Minute,
		DownloadURLExpiry:        1 * time.Hour,
		MaxAttachmentsPerProduct: 50,
	}
}

// AttachmentService handles product attachment operations
type AttachmentService struct {
	attachmentRepo catalog.ProductAttachmentRepository
	productRepo    catalog.ProductRepository
	storageService ObjectStorageService
	config         AttachmentServiceConfig
}

// NewAttachmentService creates a new AttachmentService
func NewAttachmentService(
	attachmentRepo catalog.ProductAttachmentRepository,
	productRepo catalog.ProductRepository,
	storageService ObjectStorageService,
) *AttachmentService {
	return &AttachmentService{
		attachmentRepo: attachmentRepo,
		productRepo:    productRepo,
		storageService: storageService,
		config:         DefaultAttachmentServiceConfig(),
	}
}

// SetConfig sets the service configuration
func (s *AttachmentService) SetConfig(config AttachmentServiceConfig) {
	s.config = config
}

// InitiateUpload creates a pending attachment record and returns a presigned upload URL
func (s *AttachmentService) InitiateUpload(
	ctx context.Context,
	tenantID uuid.UUID,
	req InitiateUploadRequest,
	uploadedBy *uuid.UUID,
) (*InitiateUploadResponse, error) {
	// Validate product exists
	_, err := s.productRepo.FindByIDForTenant(ctx, tenantID, req.ProductID)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil, shared.NewDomainError("PRODUCT_NOT_FOUND", "Product not found")
		}
		return nil, err
	}

	// Check attachment limit
	count, err := s.attachmentRepo.CountActiveByProduct(ctx, tenantID, req.ProductID)
	if err != nil {
		return nil, err
	}
	if count >= int64(s.config.MaxAttachmentsPerProduct) {
		return nil, shared.NewDomainError("ATTACHMENT_LIMIT_EXCEEDED",
			fmt.Sprintf("Maximum %d attachments per product allowed", s.config.MaxAttachmentsPerProduct))
	}

	// Validate attachment type
	attachmentType := catalog.AttachmentType(req.Type)
	if !attachmentType.IsValid() {
		return nil, shared.NewDomainError("INVALID_ATTACHMENT_TYPE", "Invalid attachment type")
	}

	// Validate content type against whitelist (CRITICAL: prevents uploading dangerous files)
	if !isAllowedContentType(req.ContentType) {
		return nil, shared.NewDomainError("DISALLOWED_CONTENT_TYPE",
			fmt.Sprintf("Content type '%s' is not allowed. Allowed types: images, PDF, Office documents, and text files.", req.ContentType))
	}

	// Validate content type matches attachment type (images must be image/* content type)
	if attachmentType.IsImage() && !isImageContentType(req.ContentType) {
		return nil, shared.NewDomainError("INVALID_CONTENT_TYPE",
			"Image attachment type requires an image content type")
	}

	// Check if there's already a main image when adding a new main image
	if attachmentType == catalog.AttachmentTypeMainImage {
		existingMain, err := s.attachmentRepo.FindMainImage(ctx, tenantID, req.ProductID)
		if err != nil && !errors.Is(err, shared.ErrNotFound) {
			return nil, err
		}
		if existingMain != nil {
			return nil, shared.NewDomainError("MAIN_IMAGE_EXISTS",
				"A main image already exists. Delete or change the existing main image first.")
		}
	}

	// Generate storage key
	storageKey := s.generateStorageKey(tenantID, req.ProductID, req.FileName)

	// Create the attachment entity
	attachment, err := catalog.NewProductAttachment(
		tenantID,
		req.ProductID,
		attachmentType,
		req.FileName,
		req.FileSize,
		req.ContentType,
		storageKey,
		uploadedBy,
	)
	if err != nil {
		return nil, err
	}

	// Save the attachment in pending status
	if err := s.attachmentRepo.Save(ctx, attachment); err != nil {
		return nil, err
	}

	// Generate presigned upload URL
	uploadURL, expiresAt, err := s.storageService.GenerateUploadURL(
		ctx,
		storageKey,
		req.ContentType,
		s.config.UploadURLExpiry,
	)
	if err != nil {
		// Clean up the attachment record if URL generation fails
		_ = s.attachmentRepo.DeleteForTenant(ctx, tenantID, attachment.ID)
		return nil, shared.NewDomainError("UPLOAD_URL_FAILED", "Failed to generate upload URL")
	}

	return &InitiateUploadResponse{
		AttachmentID: attachment.ID,
		UploadURL:    uploadURL,
		ExpiresAt:    expiresAt,
	}, nil
}

// ConfirmUpload verifies the upload completed and activates the attachment
func (s *AttachmentService) ConfirmUpload(
	ctx context.Context,
	tenantID uuid.UUID,
	attachmentID uuid.UUID,
) (*AttachmentResponse, error) {
	// Find the attachment
	attachment, err := s.attachmentRepo.FindByIDForTenant(ctx, tenantID, attachmentID)
	if err != nil {
		return nil, err
	}

	// Verify the file exists in storage
	exists, err := s.storageService.ObjectExists(ctx, attachment.StorageKey)
	if err != nil {
		return nil, shared.NewDomainError("STORAGE_CHECK_FAILED", "Failed to verify upload")
	}
	if !exists {
		return nil, shared.NewDomainError("UPLOAD_NOT_FOUND",
			"File not found in storage. Please upload the file first.")
	}

	// Confirm the attachment (changes status from pending to active)
	if err := attachment.Confirm(); err != nil {
		return nil, err
	}

	// Get the next sort order if not set
	if attachment.SortOrder == 0 {
		maxOrder, err := s.attachmentRepo.GetMaxSortOrder(ctx, tenantID, attachment.ProductID)
		if err != nil {
			return nil, err
		}
		if err := attachment.SetSortOrder(maxOrder + 1); err != nil {
			return nil, err
		}
	}

	// Save the updated attachment
	if err := s.attachmentRepo.Save(ctx, attachment); err != nil {
		return nil, err
	}

	response := ToAttachmentResponse(attachment)

	// Enrich with download URL
	url, _, err := s.storageService.GenerateDownloadURL(ctx, attachment.StorageKey, s.config.DownloadURLExpiry)
	if err == nil {
		response.URL = url
	}

	return &response, nil
}

// GetByID retrieves an attachment by ID
func (s *AttachmentService) GetByID(
	ctx context.Context,
	tenantID uuid.UUID,
	attachmentID uuid.UUID,
) (*AttachmentResponse, error) {
	attachment, err := s.attachmentRepo.FindByIDForTenant(ctx, tenantID, attachmentID)
	if err != nil {
		return nil, err
	}

	response := ToAttachmentResponse(attachment)
	s.enrichWithURLs(ctx, &response, attachment)

	return &response, nil
}

// GetByProduct retrieves all attachments for a product
func (s *AttachmentService) GetByProduct(
	ctx context.Context,
	tenantID uuid.UUID,
	productID uuid.UUID,
	filter AttachmentListFilter,
) ([]AttachmentListResponse, int64, error) {
	// Validate product exists
	_, err := s.productRepo.FindByIDForTenant(ctx, tenantID, productID)
	if err != nil {
		return nil, 0, err
	}

	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.OrderBy == "" {
		filter.OrderBy = "sort_order"
	}
	if filter.OrderDir == "" {
		filter.OrderDir = "asc"
	}

	// Build domain filter
	domainFilter := shared.Filter{
		Page:     filter.Page,
		PageSize: filter.PageSize,
		OrderBy:  filter.OrderBy,
		OrderDir: filter.OrderDir,
		Search:   filter.Search,
		Filters:  make(map[string]interface{}),
	}

	// Apply specific filters
	if filter.Status != "" {
		domainFilter.Filters["status"] = filter.Status
	} else {
		// By default, only show active attachments
		domainFilter.Filters["status"] = string(catalog.AttachmentStatusActive)
	}
	if filter.Type != "" {
		domainFilter.Filters["type"] = filter.Type
	}

	// Get attachments
	attachments, err := s.attachmentRepo.FindByProduct(ctx, tenantID, productID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	// Get total count
	count, err := s.attachmentRepo.CountActiveByProduct(ctx, tenantID, productID)
	if err != nil {
		return nil, 0, err
	}

	// Convert to responses
	responses := ToAttachmentListResponses(attachments)

	// Enrich with URLs
	for i, a := range attachments {
		s.enrichListWithURLs(ctx, &responses[i], &a)
	}

	return responses, count, nil
}

// GetActiveByProduct retrieves all active attachments for a product
func (s *AttachmentService) GetActiveByProduct(
	ctx context.Context,
	tenantID uuid.UUID,
	productID uuid.UUID,
) ([]AttachmentListResponse, error) {
	attachments, err := s.attachmentRepo.FindActiveByProduct(ctx, tenantID, productID)
	if err != nil {
		return nil, err
	}

	responses := ToAttachmentListResponses(attachments)

	// Enrich with URLs
	for i, a := range attachments {
		s.enrichListWithURLs(ctx, &responses[i], &a)
	}

	return responses, nil
}

// GetMainImage retrieves the main image for a product
func (s *AttachmentService) GetMainImage(
	ctx context.Context,
	tenantID uuid.UUID,
	productID uuid.UUID,
) (*AttachmentResponse, error) {
	attachment, err := s.attachmentRepo.FindMainImage(ctx, tenantID, productID)
	if err != nil {
		return nil, err
	}

	response := ToAttachmentResponse(attachment)
	s.enrichWithURLs(ctx, &response, attachment)

	return &response, nil
}

// Delete marks an attachment as deleted (soft delete)
func (s *AttachmentService) Delete(
	ctx context.Context,
	tenantID uuid.UUID,
	attachmentID uuid.UUID,
) error {
	attachment, err := s.attachmentRepo.FindByIDForTenant(ctx, tenantID, attachmentID)
	if err != nil {
		return err
	}

	// Perform soft delete (changes status to deleted)
	if err := attachment.Delete(); err != nil {
		return err
	}

	// Save the updated attachment
	return s.attachmentRepo.Save(ctx, attachment)
}

// PermanentDelete permanently deletes an attachment and its storage object
func (s *AttachmentService) PermanentDelete(
	ctx context.Context,
	tenantID uuid.UUID,
	attachmentID uuid.UUID,
) error {
	attachment, err := s.attachmentRepo.FindByIDForTenant(ctx, tenantID, attachmentID)
	if err != nil {
		return err
	}

	// Delete from storage (log error but continue - storage object might already be deleted)
	if err := s.storageService.DeleteObject(ctx, attachment.StorageKey); err != nil {
		slog.WarnContext(ctx, "failed to delete attachment from storage",
			"attachment_id", attachment.ID,
			"storage_key", attachment.StorageKey,
			"error", err)
	}

	// Delete thumbnail if exists
	if attachment.ThumbnailKey != "" {
		if err := s.storageService.DeleteObject(ctx, attachment.ThumbnailKey); err != nil {
			slog.WarnContext(ctx, "failed to delete thumbnail from storage",
				"attachment_id", attachment.ID,
				"thumbnail_key", attachment.ThumbnailKey,
				"error", err)
		}
	}

	// Delete from database
	return s.attachmentRepo.DeleteForTenant(ctx, tenantID, attachmentID)
}

// SetAsMainImage sets an attachment as the main product image
func (s *AttachmentService) SetAsMainImage(
	ctx context.Context,
	tenantID uuid.UUID,
	attachmentID uuid.UUID,
) (*AttachmentResponse, error) {
	// Find the attachment to promote
	attachment, err := s.attachmentRepo.FindByIDForTenant(ctx, tenantID, attachmentID)
	if err != nil {
		return nil, err
	}

	// Check if it's an image
	if !attachment.Type.IsImage() {
		return nil, shared.NewDomainError("NOT_AN_IMAGE",
			"Only image attachments can be set as main image")
	}

	// Find current main image (if any)
	currentMain, err := s.attachmentRepo.FindMainImage(ctx, tenantID, attachment.ProductID)
	if err != nil && !errors.Is(err, shared.ErrNotFound) {
		return nil, err
	}

	// Prepare batch of attachments to save (for atomic update)
	attachmentsToSave := []*catalog.ProductAttachment{attachment}

	// Demote current main image to gallery image (if exists and different)
	if currentMain != nil && currentMain.ID != attachmentID {
		if err := currentMain.SetAsGalleryImage(); err != nil {
			return nil, err
		}
		attachmentsToSave = append(attachmentsToSave, currentMain)
	}

	// Promote the new attachment to main image
	if err := attachment.SetAsMainImage(); err != nil {
		return nil, err
	}

	// Save all attachments atomically using batch save
	if err := s.attachmentRepo.SaveBatch(ctx, attachmentsToSave); err != nil {
		return nil, err
	}

	response := ToAttachmentResponse(attachment)
	s.enrichWithURLs(ctx, &response, attachment)

	return &response, nil
}

// ReorderAttachments updates the sort order of attachments
func (s *AttachmentService) ReorderAttachments(
	ctx context.Context,
	tenantID uuid.UUID,
	productID uuid.UUID,
	attachmentIDs []uuid.UUID,
) ([]AttachmentListResponse, error) {
	// Validate product exists
	_, err := s.productRepo.FindByIDForTenant(ctx, tenantID, productID)
	if err != nil {
		return nil, err
	}

	// Get existing attachments
	attachments, err := s.attachmentRepo.FindByIDs(ctx, tenantID, attachmentIDs)
	if err != nil {
		return nil, err
	}

	// Verify all attachments exist and belong to the product
	attachmentMap := make(map[uuid.UUID]*catalog.ProductAttachment)
	for i := range attachments {
		a := &attachments[i]
		if a.ProductID != productID {
			return nil, shared.NewDomainError("INVALID_ATTACHMENT",
				fmt.Sprintf("Attachment %s does not belong to this product", a.ID))
		}
		if a.Status == catalog.AttachmentStatusDeleted {
			return nil, shared.NewDomainError("ATTACHMENT_DELETED",
				fmt.Sprintf("Attachment %s is deleted", a.ID))
		}
		attachmentMap[a.ID] = a
	}

	// Check for missing attachments
	if len(attachmentMap) != len(attachmentIDs) {
		return nil, shared.NewDomainError("ATTACHMENTS_NOT_FOUND",
			"Some attachments were not found")
	}

	// Update sort orders
	updatedAttachments := make([]*catalog.ProductAttachment, len(attachmentIDs))
	for i, id := range attachmentIDs {
		a := attachmentMap[id]
		if err := a.SetSortOrder(i); err != nil {
			return nil, err
		}
		updatedAttachments[i] = a
	}

	// Save all attachments
	if err := s.attachmentRepo.SaveBatch(ctx, updatedAttachments); err != nil {
		return nil, err
	}

	// Convert to responses
	result := make([]catalog.ProductAttachment, len(updatedAttachments))
	for i, a := range updatedAttachments {
		result[i] = *a
	}
	responses := ToAttachmentListResponses(result)

	// Enrich with URLs
	for i, a := range updatedAttachments {
		s.enrichListWithURLs(ctx, &responses[i], a)
	}

	return responses, nil
}

// ============================================================================
// Helper Methods
// ============================================================================

// generateStorageKey generates a unique storage key for a file
func (s *AttachmentService) generateStorageKey(tenantID, productID uuid.UUID, fileName string) string {
	ext := filepath.Ext(fileName)
	uniqueID := uuid.New().String()
	// Format: tenants/{tenantID}/products/{productID}/attachments/{uniqueID}{ext}
	return fmt.Sprintf("tenants/%s/products/%s/attachments/%s%s",
		tenantID.String(),
		productID.String(),
		uniqueID,
		ext,
	)
}

// enrichWithURLs adds download URLs to an attachment response
func (s *AttachmentService) enrichWithURLs(
	ctx context.Context,
	response *AttachmentResponse,
	attachment *catalog.ProductAttachment,
) {
	if attachment.Status != catalog.AttachmentStatusActive {
		return
	}

	url, _, err := s.storageService.GenerateDownloadURL(
		ctx,
		attachment.StorageKey,
		s.config.DownloadURLExpiry,
	)
	if err == nil {
		response.URL = url
	}

	if attachment.ThumbnailKey != "" {
		thumbURL, _, err := s.storageService.GenerateDownloadURL(
			ctx,
			attachment.ThumbnailKey,
			s.config.DownloadURLExpiry,
		)
		if err == nil {
			response.ThumbnailURL = thumbURL
		}
	}
}

// enrichListWithURLs adds download URLs to an attachment list response
func (s *AttachmentService) enrichListWithURLs(
	ctx context.Context,
	response *AttachmentListResponse,
	attachment *catalog.ProductAttachment,
) {
	if attachment.Status != catalog.AttachmentStatusActive {
		return
	}

	url, _, err := s.storageService.GenerateDownloadURL(
		ctx,
		attachment.StorageKey,
		s.config.DownloadURLExpiry,
	)
	if err == nil {
		response.URL = url
	}

	if attachment.ThumbnailKey != "" {
		thumbURL, _, err := s.storageService.GenerateDownloadURL(
			ctx,
			attachment.ThumbnailKey,
			s.config.DownloadURLExpiry,
		)
		if err == nil {
			response.ThumbnailURL = thumbURL
		}
	}
}

// isImageContentType checks if a content type is an image
func isImageContentType(contentType string) bool {
	return strings.HasPrefix(strings.ToLower(contentType), "image/")
}

// isAllowedContentType checks if a content type is in the whitelist
func isAllowedContentType(contentType string) bool {
	return AllowedContentTypes[strings.ToLower(contentType)]
}
