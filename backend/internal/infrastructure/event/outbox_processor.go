package event

import (
	"context"
	"sync"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// OutboxProcessorConfig holds configuration for the outbox processor
type OutboxProcessorConfig struct {
	BatchSize        int
	PollInterval     time.Duration
	CleanupEnabled   bool
	CleanupRetention time.Duration
	CleanupInterval  time.Duration
}

// DefaultOutboxProcessorConfig returns default configuration
func DefaultOutboxProcessorConfig() OutboxProcessorConfig {
	return OutboxProcessorConfig{
		BatchSize:        100,
		PollInterval:     5 * time.Second,
		CleanupEnabled:   true,
		CleanupRetention: 7 * 24 * time.Hour, // 7 days
		CleanupInterval:  1 * time.Hour,
	}
}

// OutboxProcessor processes outbox entries in the background
type OutboxProcessor struct {
	repo       shared.OutboxRepository
	eventBus   shared.EventBus
	serializer *EventSerializer
	config     OutboxProcessorConfig
	logger     *zap.Logger

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewOutboxProcessor creates a new outbox processor
func NewOutboxProcessor(
	repo shared.OutboxRepository,
	eventBus shared.EventBus,
	serializer *EventSerializer,
	config OutboxProcessorConfig,
	logger *zap.Logger,
) *OutboxProcessor {
	return &OutboxProcessor{
		repo:       repo,
		eventBus:   eventBus,
		serializer: serializer,
		config:     config,
		logger:     logger,
	}
}

// Start starts the background processing
func (p *OutboxProcessor) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	// Start main processor
	p.wg.Add(1)
	go p.processLoop(ctx)

	// Start cleanup processor if enabled
	if p.config.CleanupEnabled {
		p.wg.Add(1)
		go p.cleanupLoop(ctx)
	}

	p.logger.Info("outbox processor started",
		zap.Int("batch_size", p.config.BatchSize),
		zap.Duration("poll_interval", p.config.PollInterval),
	)

	return nil
}

// Stop gracefully stops the processor
func (p *OutboxProcessor) Stop(ctx context.Context) error {
	if p.cancel != nil {
		p.cancel()
	}

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.logger.Info("outbox processor stopped")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// processLoop is the main processing loop
func (p *OutboxProcessor) processLoop(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.processBatch(ctx)
		}
	}
}

// processBatch processes a batch of pending and retryable entries
func (p *OutboxProcessor) processBatch(ctx context.Context) {
	// Process pending entries
	pending, err := p.repo.FindPending(ctx, p.config.BatchSize)
	if err != nil {
		p.logger.Error("failed to find pending entries", zap.Error(err))
		return
	}

	if len(pending) > 0 {
		p.processEntries(ctx, pending)
	}

	// Process retryable entries
	retryable, err := p.repo.FindRetryable(ctx, time.Now(), p.config.BatchSize)
	if err != nil {
		p.logger.Error("failed to find retryable entries", zap.Error(err))
		return
	}

	if len(retryable) > 0 {
		p.processEntries(ctx, retryable)
	}
}

// processEntries processes a slice of outbox entries
func (p *OutboxProcessor) processEntries(ctx context.Context, entries []*shared.OutboxEntry) {
	ids := make([]uuid.UUID, len(entries))
	for i, e := range entries {
		ids[i] = e.ID
	}

	// Atomically claim entries
	claimed, err := p.repo.MarkProcessing(ctx, ids)
	if err != nil {
		p.logger.Error("failed to mark entries as processing", zap.Error(err))
		return
	}

	for _, entry := range claimed {
		p.processEntry(ctx, entry)
	}
}

// processEntry processes a single outbox entry
func (p *OutboxProcessor) processEntry(ctx context.Context, entry *shared.OutboxEntry) {
	// Deserialize the event
	event, err := p.serializer.Deserialize(entry.EventType, entry.Payload)
	if err != nil {
		p.logger.Error("failed to deserialize event",
			zap.String("event_id", entry.EventID.String()),
			zap.String("event_type", entry.EventType),
			zap.Error(err),
		)
		entry.MarkFailed(err.Error())
		if entry.IsDead() {
			p.logger.Warn("event moved to dead letter queue",
				zap.String("event_id", entry.EventID.String()),
				zap.String("event_type", entry.EventType),
				zap.String("aggregate_type", entry.AggregateType),
				zap.String("aggregate_id", entry.AggregateID.String()),
				zap.Int("retry_count", entry.RetryCount),
				zap.String("last_error", entry.LastError),
			)
		}
		if updateErr := p.repo.Update(ctx, entry); updateErr != nil {
			p.logger.Error("failed to update entry", zap.Error(updateErr))
		}
		return
	}

	// Publish to event bus
	if err := p.eventBus.Publish(ctx, event); err != nil {
		p.logger.Error("failed to publish event",
			zap.String("event_id", entry.EventID.String()),
			zap.String("event_type", entry.EventType),
			zap.Error(err),
		)
		entry.MarkFailed(err.Error())
		if entry.IsDead() {
			p.logger.Warn("event moved to dead letter queue",
				zap.String("event_id", entry.EventID.String()),
				zap.String("event_type", entry.EventType),
				zap.String("aggregate_type", entry.AggregateType),
				zap.String("aggregate_id", entry.AggregateID.String()),
				zap.Int("retry_count", entry.RetryCount),
				zap.String("last_error", entry.LastError),
			)
		}
		if updateErr := p.repo.Update(ctx, entry); updateErr != nil {
			p.logger.Error("failed to update entry", zap.Error(updateErr))
		}
		return
	}

	// Mark as sent
	entry.MarkSent()
	if err := p.repo.Update(ctx, entry); err != nil {
		p.logger.Error("failed to mark entry as sent",
			zap.String("event_id", entry.EventID.String()),
			zap.Error(err),
		)
	} else {
		p.logger.Debug("event processed successfully",
			zap.String("event_id", entry.EventID.String()),
			zap.String("event_type", entry.EventType),
		)
	}
}

// cleanupLoop periodically cleans up old processed entries
func (p *OutboxProcessor) cleanupLoop(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.cleanup(ctx)
		}
	}
}

// cleanup removes old processed entries
func (p *OutboxProcessor) cleanup(ctx context.Context) {
	cutoff := time.Now().Add(-p.config.CleanupRetention)
	deleted, err := p.repo.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		p.logger.Error("failed to cleanup old entries", zap.Error(err))
		return
	}

	if deleted > 0 {
		p.logger.Info("cleaned up old outbox entries",
			zap.Int64("deleted", deleted),
			zap.Time("cutoff", cutoff),
		)
	}
}
