package models

import (
	"time"

	"github.com/erp/backend/internal/domain/bulk"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// ImportHistoryModel is the persistence model for the ImportHistory domain entity.
type ImportHistoryModel struct {
	TenantAggregateModel
	EntityType   bulk.ImportEntityType `gorm:"type:import_entity_type;not null"`
	FileName     string                `gorm:"type:varchar(255);not null"`
	FileSize     int64                 `gorm:"not null;default:0"`
	TotalRows    int                   `gorm:"not null;default:0"`
	SuccessRows  int                   `gorm:"not null;default:0"`
	ErrorRows    int                   `gorm:"not null;default:0"`
	SkippedRows  int                   `gorm:"not null;default:0"`
	UpdatedRows  int                   `gorm:"not null;default:0"`
	ConflictMode bulk.ConflictMode     `gorm:"type:varchar(20);not null;default:'skip'"`
	Status       bulk.ImportStatus     `gorm:"type:import_status;not null;default:'pending'"`
	ErrorDetails string                `gorm:"type:jsonb;default:'[]'"`
	ImportedBy   *uuid.UUID            `gorm:"type:uuid;index"`
	StartedAt    *time.Time            `gorm:"type:timestamptz"`
	CompletedAt  *time.Time            `gorm:"type:timestamptz"`
}

// TableName returns the table name for GORM
func (ImportHistoryModel) TableName() string {
	return "import_histories"
}

// ToDomain converts the persistence model to a domain ImportHistory entity.
func (m *ImportHistoryModel) ToDomain() *bulk.ImportHistory {
	history := &bulk.ImportHistory{
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
			CreatedBy: m.CreatedBy,
		},
		EntityType:   m.EntityType,
		FileName:     m.FileName,
		FileSize:     m.FileSize,
		TotalRows:    m.TotalRows,
		SuccessRows:  m.SuccessRows,
		ErrorRows:    m.ErrorRows,
		SkippedRows:  m.SkippedRows,
		UpdatedRows:  m.UpdatedRows,
		ConflictMode: m.ConflictMode,
		Status:       m.Status,
		ImportedBy:   m.ImportedBy,
		StartedAt:    m.StartedAt,
		CompletedAt:  m.CompletedAt,
	}

	// Parse error details JSON
	if m.ErrorDetails != "" {
		_ = history.SetErrorDetailsFromJSON(m.ErrorDetails)
	}

	return history
}

// FromDomain populates the persistence model from a domain ImportHistory entity.
func (m *ImportHistoryModel) FromDomain(h *bulk.ImportHistory) {
	m.FromDomainTenantAggregateRoot(h.TenantAggregateRoot)
	m.EntityType = h.EntityType
	m.FileName = h.FileName
	m.FileSize = h.FileSize
	m.TotalRows = h.TotalRows
	m.SuccessRows = h.SuccessRows
	m.ErrorRows = h.ErrorRows
	m.SkippedRows = h.SkippedRows
	m.UpdatedRows = h.UpdatedRows
	m.ConflictMode = h.ConflictMode
	m.Status = h.Status
	m.ImportedBy = h.ImportedBy
	m.StartedAt = h.StartedAt
	m.CompletedAt = h.CompletedAt

	// Serialize error details to JSON
	if errorJSON, err := h.ErrorDetailsJSON(); err == nil {
		m.ErrorDetails = errorJSON
	} else {
		m.ErrorDetails = "[]"
	}
}

// ImportHistoryModelFromDomain creates a new persistence model from a domain ImportHistory entity.
func ImportHistoryModelFromDomain(h *bulk.ImportHistory) *ImportHistoryModel {
	m := &ImportHistoryModel{}
	m.FromDomain(h)
	return m
}
