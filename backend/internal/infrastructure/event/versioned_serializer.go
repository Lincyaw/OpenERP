package event

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/erp/backend/internal/domain/shared"
	"go.uber.org/zap"
)

// VersionedSerializer handles JSON serialization/deserialization of domain events
// with automatic version migration support. It extends the basic serializer with
// the ability to deserialize old event versions and automatically upgrade them
// to the current schema version.
type VersionedSerializer struct {
	versionRegistry *VersionRegistry
	logger          *zap.Logger
}

// NewVersionedSerializer creates a new versioned event serializer
func NewVersionedSerializer(logger *zap.Logger) *VersionedSerializer {
	return &VersionedSerializer{
		versionRegistry: NewVersionRegistry(),
		logger:          logger,
	}
}

// Register registers an event type for deserialization (simple version 1 event)
// This is the same interface as the original EventSerializer for backward compatibility
func (s *VersionedSerializer) Register(eventType string, eventInstance shared.DomainEvent) {
	s.versionRegistry.RegisterSimpleEvent(eventType, eventInstance)
}

// RegisterVersioned registers a versioned event type with migration support
// eventType: The event type name
// currentVersion: The current (latest) schema version
// versions: Map of version number to event struct instance for each supported version
// upgraders: Chain of upgraders to migrate from old versions to new
func (s *VersionedSerializer) RegisterVersioned(
	eventType string,
	currentVersion int,
	versions map[int]shared.DomainEvent,
	upgraders ...EventUpgrader,
) error {
	return s.versionRegistry.RegisterVersionedEvent(eventType, currentVersion, versions, upgraders...)
}

// Serialize serializes a domain event to JSON bytes
// The schema_version field is automatically included in the output
func (s *VersionedSerializer) Serialize(event shared.DomainEvent) ([]byte, error) {
	return json.Marshal(event)
}

// SerializeWithVersion serializes a domain event to JSON and ensures version is included
func (s *VersionedSerializer) SerializeWithVersion(event shared.DomainEvent) ([]byte, error) {
	// Serialize normally - BaseDomainEvent already includes version in JSON
	return json.Marshal(event)
}

// Deserialize deserializes JSON bytes to a domain event
// If the event is an older version, it will be automatically upgraded to the current version
func (s *VersionedSerializer) Deserialize(eventType string, data []byte) (shared.DomainEvent, error) {
	config, ok := s.versionRegistry.GetConfig(eventType)
	if !ok {
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}

	// Extract version from payload
	version := ExtractVersion(data)

	// Upgrade payload if needed
	payload := data
	var err error
	if version < config.CurrentVersion {
		s.logVersionUpgrade(eventType, version, config.CurrentVersion)
		payload, _, err = s.versionRegistry.UpgradePayload(eventType, data, version)
		if err != nil {
			return nil, fmt.Errorf("failed to upgrade event: %w", err)
		}
	}

	// Get the event type for current version
	eventInstance, ok := config.Versions[config.CurrentVersion]
	if !ok {
		return nil, fmt.Errorf("no event type registered for version %d of %s", config.CurrentVersion, eventType)
	}

	// Create new instance of the registered type
	t := reflect.TypeOf(eventInstance)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	eventPtr := reflect.New(t).Interface()

	if err := json.Unmarshal(payload, eventPtr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}

	event, ok := eventPtr.(shared.DomainEvent)
	if !ok {
		return nil, fmt.Errorf("deserialized object does not implement DomainEvent")
	}

	return event, nil
}

// DeserializeToVersion deserializes JSON bytes to a specific version of an event
// This is useful for testing or when you need a specific version
func (s *VersionedSerializer) DeserializeToVersion(eventType string, data []byte, targetVersion int) (shared.DomainEvent, error) {
	config, ok := s.versionRegistry.GetConfig(eventType)
	if !ok {
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}

	// Extract version from payload
	version := ExtractVersion(data)

	if version > targetVersion {
		return nil, fmt.Errorf("cannot downgrade event from version %d to %d", version, targetVersion)
	}

	// Upgrade payload if needed (only up to target version)
	payload := data
	if version < targetVersion {
		var err error
		// Temporarily modify config to stop at targetVersion
		for v := version; v < targetVersion; v++ {
			upgrader, ok := config.Upgraders[v]
			if !ok {
				return nil, fmt.Errorf("missing upgrader for version %d -> %d", v, v+1)
			}
			payload, err = upgrader.Upgrade(payload)
			if err != nil {
				return nil, fmt.Errorf("failed to upgrade from v%d to v%d: %w", v, v+1, err)
			}
		}
	}

	// Get the event type for target version
	eventInstance, ok := config.Versions[targetVersion]
	if !ok {
		return nil, fmt.Errorf("no event type registered for version %d of %s", targetVersion, eventType)
	}

	// Create new instance of the registered type
	t := reflect.TypeOf(eventInstance)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	eventPtr := reflect.New(t).Interface()

	if err := json.Unmarshal(payload, eventPtr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}

	event, ok := eventPtr.(shared.DomainEvent)
	if !ok {
		return nil, fmt.Errorf("deserialized object does not implement DomainEvent")
	}

	return event, nil
}

// IsRegistered checks if an event type is registered
func (s *VersionedSerializer) IsRegistered(eventType string) bool {
	return s.versionRegistry.IsRegistered(eventType)
}

// RegisteredTypes returns all registered event types
func (s *VersionedSerializer) RegisteredTypes() []string {
	return s.versionRegistry.RegisteredTypes()
}

// GetCurrentVersion returns the current version for an event type
func (s *VersionedSerializer) GetCurrentVersion(eventType string) (int, bool) {
	return s.versionRegistry.GetCurrentVersion(eventType)
}

// GetVersionRegistry returns the underlying version registry for advanced use cases
func (s *VersionedSerializer) GetVersionRegistry() *VersionRegistry {
	return s.versionRegistry
}

// logVersionUpgrade logs when an event is being upgraded
func (s *VersionedSerializer) logVersionUpgrade(eventType string, from, to int) {
	if s.logger != nil {
		s.logger.Debug("upgrading event version",
			zap.String("event_type", eventType),
			zap.Int("from_version", from),
			zap.Int("to_version", to),
		)
	}
}

// UpgradePayloadOnly upgrades an event payload without deserializing to a struct
// Useful for batch migrations or when you just need the upgraded JSON
func (s *VersionedSerializer) UpgradePayloadOnly(eventType string, data []byte) ([]byte, int, error) {
	version := ExtractVersion(data)
	return s.versionRegistry.UpgradePayload(eventType, data, version)
}

// GetEventVersion extracts the version from an event payload
func (s *VersionedSerializer) GetEventVersion(data []byte) int {
	return ExtractVersion(data)
}
