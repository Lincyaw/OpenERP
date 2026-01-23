package event

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/erp/backend/internal/domain/shared"
	"go.uber.org/zap"
)

// InMemoryEventBus implements EventBus with in-memory pub/sub
type InMemoryEventBus struct {
	registry *HandlerRegistry
	logger   *zap.Logger
	running  atomic.Bool
	wg       sync.WaitGroup
}

// NewInMemoryEventBus creates a new in-memory event bus
func NewInMemoryEventBus(logger *zap.Logger) *InMemoryEventBus {
	return &InMemoryEventBus{
		registry: NewHandlerRegistry(),
		logger:   logger,
	}
}

// Publish publishes events to all registered handlers synchronously
func (b *InMemoryEventBus) Publish(ctx context.Context, events ...shared.DomainEvent) error {
	for _, event := range events {
		handlers := b.registry.GetHandlers(event.EventType())

		for _, handler := range handlers {
			if err := b.dispatchToHandler(ctx, handler, event); err != nil {
				// Log error but continue with other handlers
				b.logger.Error("handler failed to process event",
					zap.String("event_type", event.EventType()),
					zap.String("event_id", event.EventID().String()),
					zap.Error(err),
				)
			}
		}
	}
	return nil
}

// Subscribe registers a handler for specific event types
func (b *InMemoryEventBus) Subscribe(handler shared.EventHandler, eventTypes ...string) {
	// If handler specifies its own event types, use those
	if len(eventTypes) == 0 {
		eventTypes = handler.EventTypes()
	}
	b.registry.Register(handler, eventTypes...)
	b.logger.Debug("handler subscribed",
		zap.Strings("event_types", eventTypes),
	)
}

// Unsubscribe removes a handler
func (b *InMemoryEventBus) Unsubscribe(handler shared.EventHandler) {
	b.registry.Unregister(handler)
	b.logger.Debug("handler unsubscribed")
}

// Start starts the event bus
func (b *InMemoryEventBus) Start(ctx context.Context) error {
	b.running.Store(true)
	b.logger.Info("event bus started")
	return nil
}

// Stop stops the event bus gracefully
func (b *InMemoryEventBus) Stop(ctx context.Context) error {
	b.running.Store(false)
	b.wg.Wait()
	b.logger.Info("event bus stopped")
	return nil
}

// dispatchToHandler safely dispatches an event to a handler
func (b *InMemoryEventBus) dispatchToHandler(ctx context.Context, handler shared.EventHandler, event shared.DomainEvent) (err error) {
	defer func() {
		if r := recover(); r != nil {
			b.logger.Error("handler panicked",
				zap.String("event_type", event.EventType()),
				zap.Any("panic", r),
			)
		}
	}()

	return handler.Handle(ctx, event)
}

// Ensure InMemoryEventBus implements EventBus
var _ shared.EventBus = (*InMemoryEventBus)(nil)
