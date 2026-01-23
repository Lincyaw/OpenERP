package shared

import "context"

// EventHandler handles domain events
type EventHandler interface {
	// Handle processes a domain event
	Handle(ctx context.Context, event DomainEvent) error
	// EventTypes returns the event types this handler is interested in
	// An empty slice means the handler receives all events
	EventTypes() []string
}

// EventPublisher publishes domain events
type EventPublisher interface {
	// Publish publishes one or more domain events
	Publish(ctx context.Context, events ...DomainEvent) error
}

// EventSubscriber subscribes to domain events
type EventSubscriber interface {
	// Subscribe registers a handler for specific event types
	// If no event types are provided, the handler receives all events
	Subscribe(handler EventHandler, eventTypes ...string)
	// Unsubscribe removes a handler from the subscription list
	Unsubscribe(handler EventHandler)
}

// EventBus combines publisher and subscriber capabilities
type EventBus interface {
	EventPublisher
	EventSubscriber
	// Start starts the event bus (e.g., background processing)
	Start(ctx context.Context) error
	// Stop gracefully stops the event bus
	Stop(ctx context.Context) error
}
