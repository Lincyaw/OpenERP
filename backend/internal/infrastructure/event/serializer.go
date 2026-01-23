package event

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"github.com/erp/backend/internal/domain/shared"
)

// EventSerializer handles JSON serialization/deserialization of domain events
type EventSerializer struct {
	mu       sync.RWMutex
	registry map[string]reflect.Type // eventType -> Go type
}

// NewEventSerializer creates a new event serializer
func NewEventSerializer() *EventSerializer {
	return &EventSerializer{
		registry: make(map[string]reflect.Type),
	}
}

// Register registers an event type for deserialization
// The eventType should match what EventType() returns on the event
func (s *EventSerializer) Register(eventType string, eventInstance shared.DomainEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := reflect.TypeOf(eventInstance)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	s.registry[eventType] = t
}

// Serialize serializes a domain event to JSON bytes
func (s *EventSerializer) Serialize(event shared.DomainEvent) ([]byte, error) {
	return json.Marshal(event)
}

// Deserialize deserializes JSON bytes to a domain event
func (s *EventSerializer) Deserialize(eventType string, data []byte) (shared.DomainEvent, error) {
	s.mu.RLock()
	t, ok := s.registry[eventType]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}

	// Create new instance of the registered type
	eventPtr := reflect.New(t).Interface()

	if err := json.Unmarshal(data, eventPtr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}

	event, ok := eventPtr.(shared.DomainEvent)
	if !ok {
		return nil, fmt.Errorf("deserialized object does not implement DomainEvent")
	}

	return event, nil
}

// IsRegistered checks if an event type is registered
func (s *EventSerializer) IsRegistered(eventType string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.registry[eventType]
	return ok
}

// RegisteredTypes returns all registered event types
func (s *EventSerializer) RegisteredTypes() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	types := make([]string, 0, len(s.registry))
	for t := range s.registry {
		types = append(types, t)
	}
	return types
}
