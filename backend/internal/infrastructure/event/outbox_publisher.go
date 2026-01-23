package event

import (
	"context"

	"github.com/erp/backend/internal/domain/shared"
	"gorm.io/gorm"
)

// OutboxPublisher publishes domain events to the outbox within a transaction
type OutboxPublisher struct {
	serializer *EventSerializer
}

// NewOutboxPublisher creates a new outbox publisher
func NewOutboxPublisher(serializer *EventSerializer) *OutboxPublisher {
	return &OutboxPublisher{
		serializer: serializer,
	}
}

// PublishWithTx publishes events to the outbox within the provided transaction
// This ensures events are persisted atomically with the aggregate changes
func (p *OutboxPublisher) PublishWithTx(ctx context.Context, tx *gorm.DB, events ...shared.DomainEvent) error {
	if len(events) == 0 {
		return nil
	}

	entries := make([]*shared.OutboxEntry, 0, len(events))
	for _, event := range events {
		payload, err := p.serializer.Serialize(event)
		if err != nil {
			return err
		}

		entry := shared.NewOutboxEntry(event.TenantID(), event, payload)
		entries = append(entries, entry)
	}

	repo := NewGormOutboxRepository(tx)
	return repo.Save(ctx, entries...)
}
