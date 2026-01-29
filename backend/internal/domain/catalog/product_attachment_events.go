package catalog

import (
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// Aggregate type constant for ProductAttachment
const AggregateTypeProductAttachment = "ProductAttachment"

// Event type constants for ProductAttachment
const (
	EventTypeProductAttachmentCreated     = "ProductAttachmentCreated"
	EventTypeProductAttachmentConfirmed   = "ProductAttachmentConfirmed"
	EventTypeProductAttachmentDeleted     = "ProductAttachmentDeleted"
	EventTypeProductAttachmentTypeChanged = "ProductAttachmentTypeChanged"
)

// ProductAttachmentCreatedEvent is published when a new attachment is created
type ProductAttachmentCreatedEvent struct {
	shared.BaseDomainEvent
	AttachmentID uuid.UUID      `json:"attachment_id"`
	ProductID    uuid.UUID      `json:"product_id"`
	Type         AttachmentType `json:"type"`
	FileName     string         `json:"file_name"`
	FileSize     int64          `json:"file_size"`
	ContentType  string         `json:"content_type"`
	StorageKey   string         `json:"storage_key"`
}

// NewProductAttachmentCreatedEvent creates a new ProductAttachmentCreatedEvent
func NewProductAttachmentCreatedEvent(attachment *ProductAttachment) *ProductAttachmentCreatedEvent {
	return &ProductAttachmentCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			EventTypeProductAttachmentCreated,
			AggregateTypeProductAttachment,
			attachment.ID,
			attachment.TenantID,
		),
		AttachmentID: attachment.ID,
		ProductID:    attachment.ProductID,
		Type:         attachment.Type,
		FileName:     attachment.FileName,
		FileSize:     attachment.FileSize,
		ContentType:  attachment.ContentType,
		StorageKey:   attachment.StorageKey,
	}
}

// ProductAttachmentConfirmedEvent is published when an attachment upload is confirmed
type ProductAttachmentConfirmedEvent struct {
	shared.BaseDomainEvent
	AttachmentID uuid.UUID      `json:"attachment_id"`
	ProductID    uuid.UUID      `json:"product_id"`
	Type         AttachmentType `json:"type"`
	StorageKey   string         `json:"storage_key"`
}

// NewProductAttachmentConfirmedEvent creates a new ProductAttachmentConfirmedEvent
func NewProductAttachmentConfirmedEvent(attachment *ProductAttachment) *ProductAttachmentConfirmedEvent {
	return &ProductAttachmentConfirmedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			EventTypeProductAttachmentConfirmed,
			AggregateTypeProductAttachment,
			attachment.ID,
			attachment.TenantID,
		),
		AttachmentID: attachment.ID,
		ProductID:    attachment.ProductID,
		Type:         attachment.Type,
		StorageKey:   attachment.StorageKey,
	}
}

// ProductAttachmentDeletedEvent is published when an attachment is deleted
type ProductAttachmentDeletedEvent struct {
	shared.BaseDomainEvent
	AttachmentID uuid.UUID        `json:"attachment_id"`
	ProductID    uuid.UUID        `json:"product_id"`
	StorageKey   string           `json:"storage_key"`
	ThumbnailKey string           `json:"thumbnail_key,omitempty"`
	OldStatus    AttachmentStatus `json:"old_status"`
}

// NewProductAttachmentDeletedEvent creates a new ProductAttachmentDeletedEvent
func NewProductAttachmentDeletedEvent(attachment *ProductAttachment, oldStatus AttachmentStatus) *ProductAttachmentDeletedEvent {
	return &ProductAttachmentDeletedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			EventTypeProductAttachmentDeleted,
			AggregateTypeProductAttachment,
			attachment.ID,
			attachment.TenantID,
		),
		AttachmentID: attachment.ID,
		ProductID:    attachment.ProductID,
		StorageKey:   attachment.StorageKey,
		ThumbnailKey: attachment.ThumbnailKey,
		OldStatus:    oldStatus,
	}
}

// ProductAttachmentTypeChangedEvent is published when an attachment's type changes
// (e.g., when setting as main image or demoting from main image)
type ProductAttachmentTypeChangedEvent struct {
	shared.BaseDomainEvent
	AttachmentID uuid.UUID      `json:"attachment_id"`
	ProductID    uuid.UUID      `json:"product_id"`
	OldType      AttachmentType `json:"old_type"`
	NewType      AttachmentType `json:"new_type"`
}

// NewProductAttachmentTypeChangedEvent creates a new ProductAttachmentTypeChangedEvent
func NewProductAttachmentTypeChangedEvent(attachment *ProductAttachment, oldType AttachmentType) *ProductAttachmentTypeChangedEvent {
	return &ProductAttachmentTypeChangedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			EventTypeProductAttachmentTypeChanged,
			AggregateTypeProductAttachment,
			attachment.ID,
			attachment.TenantID,
		),
		AttachmentID: attachment.ID,
		ProductID:    attachment.ProductID,
		OldType:      oldType,
		NewType:      attachment.Type,
	}
}
