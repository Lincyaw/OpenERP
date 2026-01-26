package models

import (
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// OutboxEntryModel is the persistence model for domain events stored in the outbox.
// It implements the transactional outbox pattern for reliable event delivery.
type OutboxEntryModel struct {
	ID            uuid.UUID           `gorm:"type:uuid;primaryKey"`
	TenantID      uuid.UUID           `gorm:"type:uuid;not null;index:idx_outbox_tenant_status,priority:1"`
	EventID       uuid.UUID           `gorm:"type:uuid;not null;uniqueIndex"`
	EventType     string              `gorm:"type:varchar(255);not null"`
	AggregateID   uuid.UUID           `gorm:"type:uuid;not null"`
	AggregateType string              `gorm:"type:varchar(255);not null"`
	Payload       []byte              `gorm:"type:jsonb;not null"`
	Status        shared.OutboxStatus `gorm:"type:varchar(20);default:PENDING;index:idx_outbox_tenant_status,priority:2;index:idx_outbox_status_created,priority:1"`
	RetryCount    int                 `gorm:"default:0"`
	MaxRetries    int                 `gorm:"default:5"`
	LastError     string              `gorm:"type:text"`
	NextRetryAt   *time.Time          `gorm:"index:idx_outbox_next_retry"`
	ProcessedAt   *time.Time
	CreatedAt     time.Time `gorm:"not null;default:now();index:idx_outbox_status_created,priority:2"`
	UpdatedAt     time.Time `gorm:"not null;default:now()"`
}

// TableName returns the table name for GORM
func (OutboxEntryModel) TableName() string {
	return "outbox_events"
}

// ToDomain converts the persistence model to a domain OutboxEntry
func (m *OutboxEntryModel) ToDomain() *shared.OutboxEntry {
	return &shared.OutboxEntry{
		ID:            m.ID,
		TenantID:      m.TenantID,
		EventID:       m.EventID,
		EventType:     m.EventType,
		AggregateID:   m.AggregateID,
		AggregateType: m.AggregateType,
		Payload:       m.Payload,
		Status:        m.Status,
		RetryCount:    m.RetryCount,
		MaxRetries:    m.MaxRetries,
		LastError:     m.LastError,
		NextRetryAt:   m.NextRetryAt,
		ProcessedAt:   m.ProcessedAt,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}

// FromDomain populates the persistence model from a domain OutboxEntry
func (m *OutboxEntryModel) FromDomain(e *shared.OutboxEntry) {
	m.ID = e.ID
	m.TenantID = e.TenantID
	m.EventID = e.EventID
	m.EventType = e.EventType
	m.AggregateID = e.AggregateID
	m.AggregateType = e.AggregateType
	m.Payload = e.Payload
	m.Status = e.Status
	m.RetryCount = e.RetryCount
	m.MaxRetries = e.MaxRetries
	m.LastError = e.LastError
	m.NextRetryAt = e.NextRetryAt
	m.ProcessedAt = e.ProcessedAt
	m.CreatedAt = e.CreatedAt
	m.UpdatedAt = e.UpdatedAt
}

// OutboxEntryModelFromDomain creates a new persistence model from a domain OutboxEntry
func OutboxEntryModelFromDomain(e *shared.OutboxEntry) *OutboxEntryModel {
	m := &OutboxEntryModel{}
	m.FromDomain(e)
	return m
}
