package models

import (
	"time"

	"github.com/erp/backend/internal/domain/catalog"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// ProductAttachmentModel is the persistence model for the ProductAttachment domain entity.
// Note: This model uses a custom embedded struct instead of TenantAggregateModel
// because the product_attachments table doesn't have a created_by column (uses uploaded_by instead).
type ProductAttachmentModel struct {
	AggregateModel
	TenantID     uuid.UUID                `gorm:"type:uuid;not null;index"`
	ProductID    uuid.UUID                `gorm:"type:uuid;not null;index"`
	Type         catalog.AttachmentType   `gorm:"type:varchar(20);not null"`
	Status       catalog.AttachmentStatus `gorm:"type:varchar(20);not null;default:'pending'"`
	FileName     string                   `gorm:"column:file_name;type:varchar(255);not null"`
	FileSize     int64                    `gorm:"column:file_size;type:bigint;not null"`
	ContentType  string                   `gorm:"column:content_type;type:varchar(100);not null"`
	StorageKey   string                   `gorm:"column:storage_key;type:varchar(500);not null"`
	ThumbnailKey string                   `gorm:"column:thumbnail_key;type:varchar(500)"`
	SortOrder    int                      `gorm:"column:sort_order;type:integer;not null;default:0"`
	UploadedBy   *uuid.UUID               `gorm:"column:uploaded_by;type:uuid"`
}

// TableName returns the table name for GORM
func (ProductAttachmentModel) TableName() string {
	return "product_attachments"
}

// ToDomain converts the persistence model to a domain ProductAttachment entity.
func (m *ProductAttachmentModel) ToDomain() *catalog.ProductAttachment {
	return &catalog.ProductAttachment{
		TenantAggregateRoot: shared.TenantAggregateRoot{
			BaseAggregateRoot: shared.BaseAggregateRoot{
				BaseEntity: shared.BaseEntity{
					ID:        m.ID,
					CreatedAt: m.CreatedAt,
					UpdatedAt: m.UpdatedAt,
				},
				Version: m.Version,
			},
			TenantID:  m.TenantID,
			CreatedBy: nil, // product_attachments uses uploaded_by instead of created_by
		},
		ProductID:    m.ProductID,
		Type:         m.Type,
		Status:       m.Status,
		FileName:     m.FileName,
		FileSize:     m.FileSize,
		ContentType:  m.ContentType,
		StorageKey:   m.StorageKey,
		ThumbnailKey: m.ThumbnailKey,
		SortOrder:    m.SortOrder,
		UploadedBy:   m.UploadedBy,
	}
}

// FromDomain populates the persistence model from a domain ProductAttachment entity.
func (m *ProductAttachmentModel) FromDomain(a *catalog.ProductAttachment) {
	m.FromDomainAggregateRoot(a.BaseAggregateRoot)
	m.TenantID = a.TenantID
	m.ProductID = a.ProductID
	m.Type = a.Type
	m.Status = a.Status
	m.FileName = a.FileName
	m.FileSize = a.FileSize
	m.ContentType = a.ContentType
	m.StorageKey = a.StorageKey
	m.ThumbnailKey = a.ThumbnailKey
	m.SortOrder = a.SortOrder
	m.UploadedBy = a.UploadedBy
}

// ProductAttachmentModelFromDomain creates a new persistence model from a domain ProductAttachment entity.
func ProductAttachmentModelFromDomain(a *catalog.ProductAttachment) *ProductAttachmentModel {
	m := &ProductAttachmentModel{}
	m.FromDomain(a)
	return m
}

// ToSummary converts the persistence model to a summary for list views.
type ProductAttachmentSummary struct {
	ID           uuid.UUID                `json:"id"`
	ProductID    uuid.UUID                `json:"product_id"`
	Type         catalog.AttachmentType   `json:"type"`
	Status       catalog.AttachmentStatus `json:"status"`
	FileName     string                   `json:"file_name"`
	FileSize     int64                    `json:"file_size"`
	ContentType  string                   `json:"content_type"`
	StorageKey   string                   `json:"storage_key"`
	ThumbnailKey string                   `json:"thumbnail_key,omitempty"`
	SortOrder    int                      `json:"sort_order"`
	CreatedAt    time.Time                `json:"created_at"`
}

// ToSummary converts the model to a summary struct.
func (m *ProductAttachmentModel) ToSummary() ProductAttachmentSummary {
	return ProductAttachmentSummary{
		ID:           m.ID,
		ProductID:    m.ProductID,
		Type:         m.Type,
		Status:       m.Status,
		FileName:     m.FileName,
		FileSize:     m.FileSize,
		ContentType:  m.ContentType,
		StorageKey:   m.StorageKey,
		ThumbnailKey: m.ThumbnailKey,
		SortOrder:    m.SortOrder,
		CreatedAt:    m.CreatedAt,
	}
}
