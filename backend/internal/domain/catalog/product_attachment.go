package catalog

import (
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// MaxAttachmentFileSize is the maximum allowed file size (100MB)
const MaxAttachmentFileSize = 100 * 1024 * 1024

// AttachmentType represents the type of product attachment
type AttachmentType string

const (
	AttachmentTypeMainImage    AttachmentType = "main_image"
	AttachmentTypeGalleryImage AttachmentType = "gallery_image"
	AttachmentTypeDocument     AttachmentType = "document"
	AttachmentTypeOther        AttachmentType = "other"
)

// IsValid checks if the attachment type is valid
func (t AttachmentType) IsValid() bool {
	switch t {
	case AttachmentTypeMainImage, AttachmentTypeGalleryImage,
		AttachmentTypeDocument, AttachmentTypeOther:
		return true
	default:
		return false
	}
}

// IsImage returns true if the attachment type is an image type
func (t AttachmentType) IsImage() bool {
	return t == AttachmentTypeMainImage || t == AttachmentTypeGalleryImage
}

// AttachmentStatus represents the status of a product attachment
type AttachmentStatus string

const (
	AttachmentStatusPending AttachmentStatus = "pending"
	AttachmentStatusActive  AttachmentStatus = "active"
	AttachmentStatusDeleted AttachmentStatus = "deleted"
)

// IsValid checks if the attachment status is valid
func (s AttachmentStatus) IsValid() bool {
	switch s {
	case AttachmentStatusPending, AttachmentStatusActive, AttachmentStatusDeleted:
		return true
	default:
		return false
	}
}

// ProductAttachment represents a file attachment associated with a product
// This is a sub-entity under Product aggregate
type ProductAttachment struct {
	shared.TenantAggregateRoot
	ProductID    uuid.UUID        // Reference to the product
	Type         AttachmentType   // Type of attachment
	Status       AttachmentStatus // Status of the attachment
	FileName     string           // Original file name
	FileSize     int64            // File size in bytes
	ContentType  string           // MIME type (e.g., "image/jpeg", "application/pdf")
	StorageKey   string           // Key/path in storage (S3/RustFS)
	ThumbnailKey string           // Key for thumbnail (for images)
	SortOrder    int              // Display order (0-based)
	UploadedBy   *uuid.UUID       // User who uploaded the file
}

// NewProductAttachment creates a new product attachment in pending status
func NewProductAttachment(
	tenantID uuid.UUID,
	productID uuid.UUID,
	attachmentType AttachmentType,
	fileName string,
	fileSize int64,
	contentType string,
	storageKey string,
	uploadedBy *uuid.UUID,
) (*ProductAttachment, error) {
	// Validate IDs first
	if tenantID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_TENANT_ID", "Tenant ID cannot be empty")
	}
	if productID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_PRODUCT_ID", "Product ID cannot be empty")
	}
	if err := validateAttachmentType(attachmentType); err != nil {
		return nil, err
	}
	if err := validateFileName(fileName); err != nil {
		return nil, err
	}
	if err := validateFileSize(fileSize); err != nil {
		return nil, err
	}
	if err := validateContentType(contentType); err != nil {
		return nil, err
	}
	if err := validateStorageKey(storageKey); err != nil {
		return nil, err
	}

	attachment := &ProductAttachment{
		TenantAggregateRoot: shared.NewTenantAggregateRoot(tenantID),
		ProductID:           productID,
		Type:                attachmentType,
		Status:              AttachmentStatusPending,
		FileName:            fileName,
		FileSize:            fileSize,
		ContentType:         contentType,
		StorageKey:          storageKey,
		SortOrder:           0,
		UploadedBy:          uploadedBy,
	}

	attachment.AddDomainEvent(NewProductAttachmentCreatedEvent(attachment))

	return attachment, nil
}

// Confirm confirms the upload and activates the attachment
// This should be called after the file is successfully uploaded to storage
func (a *ProductAttachment) Confirm() error {
	if a.Status == AttachmentStatusActive {
		return shared.NewDomainError("ALREADY_CONFIRMED", "Attachment is already confirmed")
	}
	if a.Status == AttachmentStatusDeleted {
		return shared.NewDomainError("CANNOT_CONFIRM_DELETED", "Cannot confirm a deleted attachment")
	}

	a.Status = AttachmentStatusActive
	a.UpdatedAt = time.Now()
	a.IncrementVersion()

	a.AddDomainEvent(NewProductAttachmentConfirmedEvent(a))

	return nil
}

// Delete marks the attachment as deleted (soft delete)
func (a *ProductAttachment) Delete() error {
	if a.Status == AttachmentStatusDeleted {
		return shared.NewDomainError("ALREADY_DELETED", "Attachment is already deleted")
	}

	oldStatus := a.Status
	a.Status = AttachmentStatusDeleted
	a.UpdatedAt = time.Now()
	a.IncrementVersion()

	a.AddDomainEvent(NewProductAttachmentDeletedEvent(a, oldStatus))

	return nil
}

// SetAsMainImage changes the attachment type to main_image
// Only image attachments can be set as main image
func (a *ProductAttachment) SetAsMainImage() error {
	if a.Status == AttachmentStatusDeleted {
		return shared.NewDomainError("CANNOT_UPDATE_DELETED", "Cannot update a deleted attachment")
	}
	if !a.Type.IsImage() {
		return shared.NewDomainError("NOT_AN_IMAGE", "Only image attachments can be set as main image")
	}
	if a.Type == AttachmentTypeMainImage {
		return shared.NewDomainError("ALREADY_MAIN_IMAGE", "Attachment is already the main image")
	}

	oldType := a.Type
	a.Type = AttachmentTypeMainImage
	a.UpdatedAt = time.Now()
	a.IncrementVersion()

	a.AddDomainEvent(NewProductAttachmentTypeChangedEvent(a, oldType))

	return nil
}

// SetAsGalleryImage changes the attachment type to gallery_image
// Used when a main_image is demoted or type correction is needed
func (a *ProductAttachment) SetAsGalleryImage() error {
	if a.Status == AttachmentStatusDeleted {
		return shared.NewDomainError("CANNOT_UPDATE_DELETED", "Cannot update a deleted attachment")
	}
	if !a.Type.IsImage() {
		return shared.NewDomainError("NOT_AN_IMAGE", "Only image attachments can be set as gallery image")
	}
	if a.Type == AttachmentTypeGalleryImage {
		return shared.NewDomainError("ALREADY_GALLERY_IMAGE", "Attachment is already a gallery image")
	}

	oldType := a.Type
	a.Type = AttachmentTypeGalleryImage
	a.UpdatedAt = time.Now()
	a.IncrementVersion()

	a.AddDomainEvent(NewProductAttachmentTypeChangedEvent(a, oldType))

	return nil
}

// SetSortOrder sets the display order of the attachment
func (a *ProductAttachment) SetSortOrder(order int) error {
	if a.Status == AttachmentStatusDeleted {
		return shared.NewDomainError("CANNOT_UPDATE_DELETED", "Cannot update a deleted attachment")
	}
	if order < 0 {
		return shared.NewDomainError("INVALID_SORT_ORDER", "Sort order cannot be negative")
	}

	a.SortOrder = order
	a.UpdatedAt = time.Now()
	a.IncrementVersion()

	return nil
}

// SetThumbnailKey sets the storage key for the thumbnail
func (a *ProductAttachment) SetThumbnailKey(key string) error {
	if a.Status == AttachmentStatusDeleted {
		return shared.NewDomainError("CANNOT_UPDATE_DELETED", "Cannot update a deleted attachment")
	}
	if !a.Type.IsImage() {
		return shared.NewDomainError("NOT_AN_IMAGE", "Only image attachments can have thumbnails")
	}

	a.ThumbnailKey = key
	a.UpdatedAt = time.Now()
	a.IncrementVersion()

	return nil
}

// IsPending returns true if the attachment is pending confirmation
func (a *ProductAttachment) IsPending() bool {
	return a.Status == AttachmentStatusPending
}

// IsActive returns true if the attachment is active
func (a *ProductAttachment) IsActive() bool {
	return a.Status == AttachmentStatusActive
}

// IsDeleted returns true if the attachment is deleted
func (a *ProductAttachment) IsDeleted() bool {
	return a.Status == AttachmentStatusDeleted
}

// IsMainImage returns true if this is the main product image
func (a *ProductAttachment) IsMainImage() bool {
	return a.Type == AttachmentTypeMainImage
}

// IsImage returns true if this attachment is any type of image
func (a *ProductAttachment) IsImage() bool {
	return a.Type.IsImage()
}

// validation functions

func validateAttachmentType(t AttachmentType) error {
	if !t.IsValid() {
		return shared.NewDomainError("INVALID_ATTACHMENT_TYPE", "Invalid attachment type")
	}
	return nil
}

func validateFileName(name string) error {
	if name == "" {
		return shared.NewDomainError("INVALID_FILE_NAME", "File name cannot be empty")
	}
	if len(name) > 255 {
		return shared.NewDomainError("INVALID_FILE_NAME", "File name cannot exceed 255 characters")
	}
	// Check for dangerous characters (control characters)
	for _, r := range name {
		if r < 32 || r == 127 {
			return shared.NewDomainError("INVALID_FILE_NAME", "File name contains invalid characters")
		}
	}
	// Prevent path separators in filename
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return shared.NewDomainError("INVALID_FILE_NAME", "File name cannot contain path separators")
	}
	return nil
}

func validateFileSize(size int64) error {
	if size <= 0 {
		return shared.NewDomainError("INVALID_FILE_SIZE", "File size must be greater than 0")
	}
	if size > MaxAttachmentFileSize {
		return shared.NewDomainError("FILE_TOO_LARGE", "File size cannot exceed 100MB")
	}
	return nil
}

func validateContentType(contentType string) error {
	if contentType == "" {
		return shared.NewDomainError("INVALID_CONTENT_TYPE", "Content type cannot be empty")
	}
	if len(contentType) > 100 {
		return shared.NewDomainError("INVALID_CONTENT_TYPE", "Content type cannot exceed 100 characters")
	}
	// Basic MIME type format validation: must contain type/subtype
	if !strings.Contains(contentType, "/") {
		return shared.NewDomainError("INVALID_CONTENT_TYPE", "Content type must be in type/subtype format")
	}
	// Check for obviously invalid patterns
	if strings.HasPrefix(contentType, "/") || strings.HasSuffix(contentType, "/") {
		return shared.NewDomainError("INVALID_CONTENT_TYPE", "Content type must be in type/subtype format")
	}
	return nil
}

func validateStorageKey(key string) error {
	if key == "" {
		return shared.NewDomainError("INVALID_STORAGE_KEY", "Storage key cannot be empty")
	}
	if len(key) > 500 {
		return shared.NewDomainError("INVALID_STORAGE_KEY", "Storage key cannot exceed 500 characters")
	}
	// Prevent path traversal attacks
	if strings.Contains(key, "..") {
		return shared.NewDomainError("INVALID_STORAGE_KEY", "Storage key cannot contain path traversal sequences")
	}
	// Prevent absolute paths (must be relative within bucket)
	if strings.HasPrefix(key, "/") {
		return shared.NewDomainError("INVALID_STORAGE_KEY", "Storage key must be a relative path")
	}
	return nil
}
