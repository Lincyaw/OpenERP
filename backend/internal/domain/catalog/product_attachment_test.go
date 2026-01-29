package catalog

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttachmentType(t *testing.T) {
	t.Run("IsValid returns true for valid types", func(t *testing.T) {
		validTypes := []AttachmentType{
			AttachmentTypeMainImage,
			AttachmentTypeGalleryImage,
			AttachmentTypeDocument,
			AttachmentTypeOther,
		}
		for _, at := range validTypes {
			assert.True(t, at.IsValid(), "expected %s to be valid", at)
		}
	})

	t.Run("IsValid returns false for invalid types", func(t *testing.T) {
		invalidType := AttachmentType("invalid")
		assert.False(t, invalidType.IsValid())
	})

	t.Run("IsImage returns true for image types", func(t *testing.T) {
		assert.True(t, AttachmentTypeMainImage.IsImage())
		assert.True(t, AttachmentTypeGalleryImage.IsImage())
	})

	t.Run("IsImage returns false for non-image types", func(t *testing.T) {
		assert.False(t, AttachmentTypeDocument.IsImage())
		assert.False(t, AttachmentTypeOther.IsImage())
	})
}

func TestAttachmentStatus(t *testing.T) {
	t.Run("IsValid returns true for valid statuses", func(t *testing.T) {
		validStatuses := []AttachmentStatus{
			AttachmentStatusPending,
			AttachmentStatusActive,
			AttachmentStatusDeleted,
		}
		for _, s := range validStatuses {
			assert.True(t, s.IsValid(), "expected %s to be valid", s)
		}
	})

	t.Run("IsValid returns false for invalid statuses", func(t *testing.T) {
		invalidStatus := AttachmentStatus("invalid")
		assert.False(t, invalidStatus.IsValid())
	})
}

func TestNewProductAttachment(t *testing.T) {
	tenantID := uuid.New()
	productID := uuid.New()
	userID := uuid.New()

	t.Run("creates attachment with valid inputs", func(t *testing.T) {
		attachment, err := NewProductAttachment(
			tenantID,
			productID,
			AttachmentTypeGalleryImage,
			"product_photo.jpg",
			1024*500, // 500KB
			"image/jpeg",
			"products/123/photos/product_photo.jpg",
			&userID,
		)
		require.NoError(t, err)
		require.NotNil(t, attachment)

		assert.Equal(t, tenantID, attachment.TenantID)
		assert.Equal(t, productID, attachment.ProductID)
		assert.Equal(t, AttachmentTypeGalleryImage, attachment.Type)
		assert.Equal(t, AttachmentStatusPending, attachment.Status)
		assert.Equal(t, "product_photo.jpg", attachment.FileName)
		assert.Equal(t, int64(1024*500), attachment.FileSize)
		assert.Equal(t, "image/jpeg", attachment.ContentType)
		assert.Equal(t, "products/123/photos/product_photo.jpg", attachment.StorageKey)
		assert.Equal(t, 0, attachment.SortOrder)
		assert.Equal(t, &userID, attachment.UploadedBy)
		assert.NotEmpty(t, attachment.ID)
		assert.Equal(t, 1, attachment.GetVersion())
	})

	t.Run("allows nil uploadedBy", func(t *testing.T) {
		attachment, err := NewProductAttachment(
			tenantID,
			productID,
			AttachmentTypeDocument,
			"specs.pdf",
			1024,
			"application/pdf",
			"products/123/docs/specs.pdf",
			nil,
		)
		require.NoError(t, err)
		assert.Nil(t, attachment.UploadedBy)
	})

	t.Run("publishes ProductAttachmentCreated event", func(t *testing.T) {
		attachment, err := NewProductAttachment(
			tenantID,
			productID,
			AttachmentTypeMainImage,
			"main.jpg",
			1024,
			"image/jpeg",
			"products/123/main.jpg",
			&userID,
		)
		require.NoError(t, err)

		events := attachment.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeProductAttachmentCreated, events[0].EventType())

		event, ok := events[0].(*ProductAttachmentCreatedEvent)
		require.True(t, ok)
		assert.Equal(t, attachment.ID, event.AttachmentID)
		assert.Equal(t, productID, event.ProductID)
		assert.Equal(t, AttachmentTypeMainImage, event.Type)
		assert.Equal(t, "main.jpg", event.FileName)
		assert.Equal(t, int64(1024), event.FileSize)
		assert.Equal(t, "image/jpeg", event.ContentType)
	})

	t.Run("fails with invalid attachment type", func(t *testing.T) {
		_, err := NewProductAttachment(
			tenantID,
			productID,
			AttachmentType("invalid"),
			"test.jpg",
			1024,
			"image/jpeg",
			"products/123/test.jpg",
			nil,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid attachment type")
	})

	t.Run("fails with zero tenant ID", func(t *testing.T) {
		_, err := NewProductAttachment(
			uuid.Nil,
			productID,
			AttachmentTypeGalleryImage,
			"test.jpg",
			1024,
			"image/jpeg",
			"products/123/test.jpg",
			nil,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Tenant ID cannot be empty")
	})

	t.Run("fails with zero product ID", func(t *testing.T) {
		_, err := NewProductAttachment(
			tenantID,
			uuid.Nil,
			AttachmentTypeGalleryImage,
			"test.jpg",
			1024,
			"image/jpeg",
			"products/123/test.jpg",
			nil,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Product ID cannot be empty")
	})

	t.Run("fails with empty file name", func(t *testing.T) {
		_, err := NewProductAttachment(
			tenantID,
			productID,
			AttachmentTypeGalleryImage,
			"",
			1024,
			"image/jpeg",
			"products/123/test.jpg",
			nil,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "File name cannot be empty")
	})

	t.Run("fails with file name too long", func(t *testing.T) {
		longName := make([]byte, 256)
		for i := range longName {
			longName[i] = 'a'
		}
		_, err := NewProductAttachment(
			tenantID,
			productID,
			AttachmentTypeGalleryImage,
			string(longName),
			1024,
			"image/jpeg",
			"products/123/test.jpg",
			nil,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "File name cannot exceed 255 characters")
	})

	t.Run("fails with zero file size", func(t *testing.T) {
		_, err := NewProductAttachment(
			tenantID,
			productID,
			AttachmentTypeGalleryImage,
			"test.jpg",
			0,
			"image/jpeg",
			"products/123/test.jpg",
			nil,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "File size must be greater than 0")
	})

	t.Run("fails with negative file size", func(t *testing.T) {
		_, err := NewProductAttachment(
			tenantID,
			productID,
			AttachmentTypeGalleryImage,
			"test.jpg",
			-100,
			"image/jpeg",
			"products/123/test.jpg",
			nil,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "File size must be greater than 0")
	})

	t.Run("fails with file size exceeding 100MB", func(t *testing.T) {
		_, err := NewProductAttachment(
			tenantID,
			productID,
			AttachmentTypeGalleryImage,
			"test.jpg",
			101*1024*1024, // 101MB
			"image/jpeg",
			"products/123/test.jpg",
			nil,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "File size cannot exceed 100MB")
	})

	t.Run("accepts maximum file size of 100MB", func(t *testing.T) {
		attachment, err := NewProductAttachment(
			tenantID,
			productID,
			AttachmentTypeGalleryImage,
			"test.jpg",
			100*1024*1024, // exactly 100MB
			"image/jpeg",
			"products/123/test.jpg",
			nil,
		)
		require.NoError(t, err)
		assert.Equal(t, int64(100*1024*1024), attachment.FileSize)
	})

	t.Run("fails with empty content type", func(t *testing.T) {
		_, err := NewProductAttachment(
			tenantID,
			productID,
			AttachmentTypeGalleryImage,
			"test.jpg",
			1024,
			"",
			"products/123/test.jpg",
			nil,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Content type cannot be empty")
	})

	t.Run("fails with content type too long", func(t *testing.T) {
		longType := make([]byte, 101)
		for i := range longType {
			longType[i] = 'a'
		}
		_, err := NewProductAttachment(
			tenantID,
			productID,
			AttachmentTypeGalleryImage,
			"test.jpg",
			1024,
			string(longType),
			"products/123/test.jpg",
			nil,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Content type cannot exceed 100 characters")
	})

	t.Run("fails with empty storage key", func(t *testing.T) {
		_, err := NewProductAttachment(
			tenantID,
			productID,
			AttachmentTypeGalleryImage,
			"test.jpg",
			1024,
			"image/jpeg",
			"",
			nil,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Storage key cannot be empty")
	})

	t.Run("fails with storage key too long", func(t *testing.T) {
		longKey := make([]byte, 501)
		for i := range longKey {
			longKey[i] = 'a'
		}
		_, err := NewProductAttachment(
			tenantID,
			productID,
			AttachmentTypeGalleryImage,
			"test.jpg",
			1024,
			"image/jpeg",
			string(longKey),
			nil,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Storage key cannot exceed 500 characters")
	})
}

func TestProductAttachmentConfirm(t *testing.T) {
	tenantID := uuid.New()
	productID := uuid.New()

	t.Run("confirms pending attachment", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		attachment.ClearDomainEvents()

		err := attachment.Confirm()
		require.NoError(t, err)

		assert.Equal(t, AttachmentStatusActive, attachment.Status)
		assert.True(t, attachment.IsActive())
		assert.False(t, attachment.IsPending())

		events := attachment.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeProductAttachmentConfirmed, events[0].EventType())
	})

	t.Run("publishes confirm event with correct data", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		attachment.ClearDomainEvents()
		_ = attachment.Confirm()

		events := attachment.GetDomainEvents()
		event, ok := events[0].(*ProductAttachmentConfirmedEvent)
		require.True(t, ok)

		assert.Equal(t, attachment.ID, event.AttachmentID)
		assert.Equal(t, productID, event.ProductID)
		assert.Equal(t, attachment.StorageKey, event.StorageKey)
	})

	t.Run("fails if already confirmed", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		_ = attachment.Confirm()
		attachment.ClearDomainEvents()

		err := attachment.Confirm()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already confirmed")
	})

	t.Run("fails if deleted", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		attachment.Status = AttachmentStatusDeleted

		err := attachment.Confirm()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot confirm a deleted attachment")
	})

	t.Run("increments version", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		originalVersion := attachment.GetVersion()

		_ = attachment.Confirm()
		assert.Equal(t, originalVersion+1, attachment.GetVersion())
	})
}

func TestProductAttachmentDelete(t *testing.T) {
	tenantID := uuid.New()
	productID := uuid.New()

	t.Run("deletes pending attachment", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		attachment.ClearDomainEvents()

		err := attachment.Delete()
		require.NoError(t, err)

		assert.Equal(t, AttachmentStatusDeleted, attachment.Status)
		assert.True(t, attachment.IsDeleted())
	})

	t.Run("deletes active attachment", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		_ = attachment.Confirm()
		attachment.ClearDomainEvents()

		err := attachment.Delete()
		require.NoError(t, err)

		assert.Equal(t, AttachmentStatusDeleted, attachment.Status)
	})

	t.Run("publishes delete event with old status", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		_ = attachment.Confirm()
		attachment.ClearDomainEvents()

		_ = attachment.Delete()

		events := attachment.GetDomainEvents()
		require.Len(t, events, 1)

		event, ok := events[0].(*ProductAttachmentDeletedEvent)
		require.True(t, ok)
		assert.Equal(t, attachment.ID, event.AttachmentID)
		assert.Equal(t, productID, event.ProductID)
		assert.Equal(t, AttachmentStatusActive, event.OldStatus)
	})

	t.Run("fails if already deleted", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		_ = attachment.Delete()

		err := attachment.Delete()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already deleted")
	})

	t.Run("increments version", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		originalVersion := attachment.GetVersion()

		_ = attachment.Delete()
		assert.Equal(t, originalVersion+1, attachment.GetVersion())
	})
}

func TestProductAttachmentSetAsMainImage(t *testing.T) {
	tenantID := uuid.New()
	productID := uuid.New()

	t.Run("sets gallery image as main image", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		attachment.Type = AttachmentTypeGalleryImage
		attachment.ClearDomainEvents()

		err := attachment.SetAsMainImage()
		require.NoError(t, err)

		assert.Equal(t, AttachmentTypeMainImage, attachment.Type)
		assert.True(t, attachment.IsMainImage())

		events := attachment.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeProductAttachmentTypeChanged, events[0].EventType())
	})

	t.Run("publishes type changed event", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		attachment.Type = AttachmentTypeGalleryImage
		attachment.ClearDomainEvents()

		_ = attachment.SetAsMainImage()

		events := attachment.GetDomainEvents()
		event, ok := events[0].(*ProductAttachmentTypeChangedEvent)
		require.True(t, ok)
		assert.Equal(t, AttachmentTypeGalleryImage, event.OldType)
		assert.Equal(t, AttachmentTypeMainImage, event.NewType)
	})

	t.Run("fails for other type", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		attachment.Type = AttachmentTypeOther

		err := attachment.SetAsMainImage()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Only image attachments")
	})

	t.Run("fails if already main image", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		attachment.Type = AttachmentTypeMainImage

		err := attachment.SetAsMainImage()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already the main image")
	})

	t.Run("fails for document type", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		attachment.Type = AttachmentTypeDocument

		err := attachment.SetAsMainImage()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Only image attachments")
	})

	t.Run("fails if deleted", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		attachment.Type = AttachmentTypeGalleryImage
		attachment.Status = AttachmentStatusDeleted

		err := attachment.SetAsMainImage()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot update a deleted attachment")
	})

	t.Run("increments version", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		attachment.Type = AttachmentTypeGalleryImage
		originalVersion := attachment.GetVersion()

		_ = attachment.SetAsMainImage()
		assert.Equal(t, originalVersion+1, attachment.GetVersion())
	})
}

func TestProductAttachmentSetAsGalleryImage(t *testing.T) {
	tenantID := uuid.New()
	productID := uuid.New()

	t.Run("demotes main image to gallery", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		attachment.Type = AttachmentTypeMainImage
		attachment.ClearDomainEvents()

		err := attachment.SetAsGalleryImage()
		require.NoError(t, err)

		assert.Equal(t, AttachmentTypeGalleryImage, attachment.Type)
		assert.False(t, attachment.IsMainImage())
		assert.True(t, attachment.IsImage())
	})

	t.Run("fails if already gallery image", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		attachment.Type = AttachmentTypeGalleryImage

		err := attachment.SetAsGalleryImage()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already a gallery image")
	})

	t.Run("fails for non-image types", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		attachment.Type = AttachmentTypeDocument

		err := attachment.SetAsGalleryImage()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Only image attachments")
	})

	t.Run("fails if deleted", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		attachment.Type = AttachmentTypeMainImage
		attachment.Status = AttachmentStatusDeleted

		err := attachment.SetAsGalleryImage()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot update a deleted attachment")
	})
}

func TestProductAttachmentSetSortOrder(t *testing.T) {
	tenantID := uuid.New()
	productID := uuid.New()

	t.Run("sets sort order", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)

		err := attachment.SetSortOrder(5)
		require.NoError(t, err)
		assert.Equal(t, 5, attachment.SortOrder)
	})

	t.Run("allows zero sort order", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		attachment.SortOrder = 10

		err := attachment.SetSortOrder(0)
		require.NoError(t, err)
		assert.Equal(t, 0, attachment.SortOrder)
	})

	t.Run("fails with negative sort order", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)

		err := attachment.SetSortOrder(-1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Sort order cannot be negative")
	})

	t.Run("fails if deleted", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		attachment.Status = AttachmentStatusDeleted

		err := attachment.SetSortOrder(5)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot update a deleted attachment")
	})

	t.Run("increments version", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		originalVersion := attachment.GetVersion()

		_ = attachment.SetSortOrder(10)
		assert.Equal(t, originalVersion+1, attachment.GetVersion())
	})
}

func TestProductAttachmentSetThumbnailKey(t *testing.T) {
	tenantID := uuid.New()
	productID := uuid.New()

	t.Run("sets thumbnail key for image", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		attachment.Type = AttachmentTypeGalleryImage

		err := attachment.SetThumbnailKey("products/123/thumbs/photo_thumb.jpg")
		require.NoError(t, err)
		assert.Equal(t, "products/123/thumbs/photo_thumb.jpg", attachment.ThumbnailKey)
	})

	t.Run("fails for non-image types", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		attachment.Type = AttachmentTypeDocument

		err := attachment.SetThumbnailKey("products/123/thumbs/doc_thumb.jpg")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Only image attachments can have thumbnails")
	})

	t.Run("fails if deleted", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		attachment.Type = AttachmentTypeGalleryImage
		attachment.Status = AttachmentStatusDeleted

		err := attachment.SetThumbnailKey("products/123/thumbs/photo_thumb.jpg")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot update a deleted attachment")
	})

	t.Run("increments version", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		attachment.Type = AttachmentTypeGalleryImage
		originalVersion := attachment.GetVersion()

		_ = attachment.SetThumbnailKey("products/123/thumbs/photo_thumb.jpg")
		assert.Equal(t, originalVersion+1, attachment.GetVersion())
	})
}

func TestProductAttachmentStatusChecks(t *testing.T) {
	tenantID := uuid.New()
	productID := uuid.New()

	t.Run("IsPending", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		assert.True(t, attachment.IsPending())
		assert.False(t, attachment.IsActive())
		assert.False(t, attachment.IsDeleted())
	})

	t.Run("IsActive", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		_ = attachment.Confirm()
		assert.False(t, attachment.IsPending())
		assert.True(t, attachment.IsActive())
		assert.False(t, attachment.IsDeleted())
	})

	t.Run("IsDeleted", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		_ = attachment.Delete()
		assert.False(t, attachment.IsPending())
		assert.False(t, attachment.IsActive())
		assert.True(t, attachment.IsDeleted())
	})
}

func TestProductAttachmentTypeChecks(t *testing.T) {
	tenantID := uuid.New()
	productID := uuid.New()

	t.Run("IsMainImage", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		attachment.Type = AttachmentTypeMainImage
		assert.True(t, attachment.IsMainImage())
		assert.True(t, attachment.IsImage())
	})

	t.Run("IsImage for gallery", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		attachment.Type = AttachmentTypeGalleryImage
		assert.False(t, attachment.IsMainImage())
		assert.True(t, attachment.IsImage())
	})

	t.Run("IsImage for document", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		attachment.Type = AttachmentTypeDocument
		assert.False(t, attachment.IsMainImage())
		assert.False(t, attachment.IsImage())
	})
}

func TestProductAttachmentEvents(t *testing.T) {
	tenantID := uuid.New()
	productID := uuid.New()
	attachment := createTestAttachment(t, tenantID, productID)

	t.Run("ProductAttachmentCreatedEvent has correct fields", func(t *testing.T) {
		events := attachment.GetDomainEvents()
		require.Len(t, events, 1)

		event, ok := events[0].(*ProductAttachmentCreatedEvent)
		require.True(t, ok)

		assert.Equal(t, attachment.ID, event.AttachmentID)
		assert.Equal(t, productID, event.ProductID)
		assert.Equal(t, attachment.Type, event.Type)
		assert.Equal(t, attachment.FileName, event.FileName)
		assert.Equal(t, attachment.FileSize, event.FileSize)
		assert.Equal(t, attachment.ContentType, event.ContentType)
		assert.Equal(t, attachment.StorageKey, event.StorageKey)
		assert.Equal(t, tenantID, event.TenantID())
		assert.Equal(t, EventTypeProductAttachmentCreated, event.EventType())
		assert.Equal(t, AggregateTypeProductAttachment, event.AggregateType())
	})

	t.Run("ProductAttachmentDeletedEvent includes thumbnail key", func(t *testing.T) {
		attachment := createTestAttachment(t, tenantID, productID)
		attachment.Type = AttachmentTypeGalleryImage
		attachment.ThumbnailKey = "products/123/thumbs/photo_thumb.jpg"
		_ = attachment.Confirm()
		attachment.ClearDomainEvents()

		_ = attachment.Delete()

		events := attachment.GetDomainEvents()
		event, ok := events[0].(*ProductAttachmentDeletedEvent)
		require.True(t, ok)

		assert.Equal(t, "products/123/thumbs/photo_thumb.jpg", event.ThumbnailKey)
		assert.Equal(t, attachment.StorageKey, event.StorageKey)
	})
}

// Helper function to create test attachment
func createTestAttachment(t *testing.T, tenantID, productID uuid.UUID) *ProductAttachment {
	t.Helper()
	attachment, err := NewProductAttachment(
		tenantID,
		productID,
		AttachmentTypeGalleryImage,
		"test_photo.jpg",
		1024*100, // 100KB
		"image/jpeg",
		"products/test/photos/test_photo.jpg",
		nil,
	)
	require.NoError(t, err)
	return attachment
}

func TestValidateAttachmentType(t *testing.T) {
	tests := []struct {
		name        string
		input       AttachmentType
		wantErr     bool
		errContains string
	}{
		{"valid main_image", AttachmentTypeMainImage, false, ""},
		{"valid gallery_image", AttachmentTypeGalleryImage, false, ""},
		{"valid document", AttachmentTypeDocument, false, ""},
		{"valid other", AttachmentTypeOther, false, ""},
		{"invalid type", AttachmentType("invalid"), true, "Invalid attachment type"},
		{"empty type", AttachmentType(""), true, "Invalid attachment type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAttachmentType(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateFileName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid filename", "photo.jpg", false},
		{"valid with spaces", "product photo.jpg", false},
		{"valid with special chars", "photo-2024_01.jpg", false},
		{"valid max length", string(make([]byte, 255)), false},
		{"empty", "", true},
		{"too long", string(make([]byte, 256)), true},
		{"contains forward slash", "path/to/file.jpg", true},
		{"contains backslash", "path\\to\\file.jpg", true},
		{"contains null byte", "file\x00.jpg", true},
		{"contains control char", "file\x01.jpg", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFileName(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateFileSize(t *testing.T) {
	tests := []struct {
		name    string
		input   int64
		wantErr bool
	}{
		{"valid small", 1024, false},
		{"valid medium", 1024 * 1024, false},
		{"valid max", 100 * 1024 * 1024, false},
		{"zero", 0, true},
		{"negative", -1, true},
		{"too large", 101 * 1024 * 1024, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFileSize(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateContentType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid jpeg", "image/jpeg", false},
		{"valid png", "image/png", false},
		{"valid pdf", "application/pdf", false},
		{"valid with charset", "text/html; charset=utf-8", false},
		{"empty", "", true},
		{"too long", string(make([]byte, 101)), true},
		{"no slash", "imagejpeg", true},
		{"starts with slash", "/jpeg", true},
		{"ends with slash", "image/", true},
		{"just slash", "/", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateContentType(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateStorageKey(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "test.jpg", false},
		{"valid path", "products/123/photos/photo.jpg", false},
		{"valid max length", string(make([]byte, 500)), false},
		{"empty", "", true},
		{"too long", string(make([]byte, 501)), true},
		{"path traversal", "../../../etc/passwd", true},
		{"path traversal mid", "products/../../../etc/passwd", true},
		{"double dots", "products/..photo.jpg", true},
		{"absolute path", "/products/123/test.jpg", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStorageKey(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
