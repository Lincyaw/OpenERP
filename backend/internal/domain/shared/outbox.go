package shared

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// OutboxStatus represents the status of an outbox entry
type OutboxStatus string

const (
	OutboxStatusPending    OutboxStatus = "PENDING"
	OutboxStatusProcessing OutboxStatus = "PROCESSING"
	OutboxStatusSent       OutboxStatus = "SENT"
	OutboxStatusFailed     OutboxStatus = "FAILED"
	OutboxStatusDead       OutboxStatus = "DEAD"
)

// Default retry configuration
const (
	DefaultMaxRetries  = 5
	DefaultBaseBackoff = time.Second
)

// OutboxEntry represents an event stored in the outbox for reliable delivery
type OutboxEntry struct {
	ID            uuid.UUID    `gorm:"type:uuid;primaryKey"`
	TenantID      uuid.UUID    `gorm:"type:uuid;not null;index:idx_outbox_tenant_status,priority:1"`
	EventID       uuid.UUID    `gorm:"type:uuid;not null;uniqueIndex"`
	EventType     string       `gorm:"type:varchar(255);not null"`
	AggregateID   uuid.UUID    `gorm:"type:uuid;not null"`
	AggregateType string       `gorm:"type:varchar(255);not null"`
	Payload       []byte       `gorm:"type:jsonb;not null"`
	Status        OutboxStatus `gorm:"type:varchar(20);default:PENDING;index:idx_outbox_tenant_status,priority:2;index:idx_outbox_status_created,priority:1"`
	RetryCount    int          `gorm:"default:0"`
	MaxRetries    int          `gorm:"default:5"`
	LastError     string       `gorm:"type:text"`
	NextRetryAt   *time.Time   `gorm:"index:idx_outbox_next_retry"`
	ProcessedAt   *time.Time
	CreatedAt     time.Time `gorm:"not null;default:now();index:idx_outbox_status_created,priority:2"`
	UpdatedAt     time.Time `gorm:"not null;default:now()"`
}

// TableName returns the table name for GORM
func (OutboxEntry) TableName() string {
	return "outbox_events"
}

// NewOutboxEntry creates a new outbox entry for a domain event
func NewOutboxEntry(tenantID uuid.UUID, event DomainEvent, payload []byte) *OutboxEntry {
	return &OutboxEntry{
		ID:            uuid.New(),
		TenantID:      tenantID,
		EventID:       event.EventID(),
		EventType:     event.EventType(),
		AggregateID:   event.AggregateID(),
		AggregateType: event.AggregateType(),
		Payload:       payload,
		Status:        OutboxStatusPending,
		RetryCount:    0,
		MaxRetries:    DefaultMaxRetries,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// CanRetry returns true if the entry can be retried
func (e *OutboxEntry) CanRetry() bool {
	return e.Status == OutboxStatusFailed && e.RetryCount < e.MaxRetries
}

// MarkProcessing marks the entry as being processed
func (e *OutboxEntry) MarkProcessing() error {
	if e.Status != OutboxStatusPending && e.Status != OutboxStatusFailed {
		return errors.New("can only mark pending or failed entries as processing")
	}
	e.Status = OutboxStatusProcessing
	e.UpdatedAt = time.Now()
	return nil
}

// MarkSent marks the entry as successfully sent
func (e *OutboxEntry) MarkSent() {
	now := time.Now()
	e.Status = OutboxStatusSent
	e.ProcessedAt = &now
	e.UpdatedAt = now
}

// MarkFailed marks the entry as failed with error and calculates next retry time
func (e *OutboxEntry) MarkFailed(errMsg string) {
	e.RetryCount++
	e.LastError = errMsg
	e.UpdatedAt = time.Now()

	if e.RetryCount >= e.MaxRetries {
		e.Status = OutboxStatusDead
	} else {
		e.Status = OutboxStatusFailed
		// Exponential backoff: 1s, 2s, 4s, 8s, 16s, ...
		backoff := DefaultBaseBackoff * time.Duration(1<<uint(e.RetryCount-1))
		nextRetry := time.Now().Add(backoff)
		e.NextRetryAt = &nextRetry
	}
}

// OutboxRepository defines the interface for outbox persistence
type OutboxRepository interface {
	// Save persists one or more outbox entries
	Save(ctx context.Context, entries ...*OutboxEntry) error
	// FindPending retrieves pending entries up to the specified limit
	FindPending(ctx context.Context, limit int) ([]*OutboxEntry, error)
	// FindRetryable retrieves failed entries that are due for retry
	FindRetryable(ctx context.Context, before time.Time, limit int) ([]*OutboxEntry, error)
	// MarkProcessing atomically marks entries as processing and returns them
	MarkProcessing(ctx context.Context, ids []uuid.UUID) ([]*OutboxEntry, error)
	// Update updates an existing outbox entry
	Update(ctx context.Context, entry *OutboxEntry) error
	// DeleteOlderThan deletes entries older than the specified time
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}
