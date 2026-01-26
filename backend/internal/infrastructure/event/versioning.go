package event

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/erp/backend/internal/domain/shared"
)

// EventUpgrader transforms an event from one schema version to another.
// Each upgrader handles a single version transition (e.g., v1 -> v2).
type EventUpgrader interface {
	// SourceVersion returns the version this upgrader reads from
	SourceVersion() int
	// TargetVersion returns the version this upgrader produces
	TargetVersion() int
	// Upgrade transforms the event payload from source to target version
	// The input is the raw JSON payload, the output is the upgraded payload
	Upgrade(payload []byte) ([]byte, error)
}

// VersionedEventConfig holds configuration for a single event type's versioning
type VersionedEventConfig struct {
	EventType      string                     // e.g., "SalesOrderCreated"
	CurrentVersion int                        // Latest version (e.g., 3)
	Upgraders      map[int]EventUpgrader      // version -> upgrader to next version
	Versions       map[int]shared.DomainEvent // version -> event type instance
}

// VersionRegistry manages versioned event types and their migrations
type VersionRegistry struct {
	mu      sync.RWMutex
	configs map[string]*VersionedEventConfig // eventType -> config
}

// NewVersionRegistry creates a new version registry
func NewVersionRegistry() *VersionRegistry {
	return &VersionRegistry{
		configs: make(map[string]*VersionedEventConfig),
	}
}

// RegisterVersionedEvent registers a versioned event type with its upgraders
// eventType: The event type name (e.g., "SalesOrderCreated")
// currentVersion: The latest schema version
// versions: Map of version number to an instance of the event struct for that version
// upgraders: List of upgraders for migrating between versions
func (r *VersionRegistry) RegisterVersionedEvent(
	eventType string,
	currentVersion int,
	versions map[int]shared.DomainEvent,
	upgraders ...EventUpgrader,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Build upgrader map
	upgraderMap := make(map[int]EventUpgrader)
	for _, u := range upgraders {
		if u.TargetVersion() != u.SourceVersion()+1 {
			return fmt.Errorf("upgrader must be sequential: got %d -> %d", u.SourceVersion(), u.TargetVersion())
		}
		upgraderMap[u.SourceVersion()] = u
	}

	// Validate version chain
	for v := 1; v < currentVersion; v++ {
		if _, ok := upgraderMap[v]; !ok {
			return fmt.Errorf("missing upgrader for version %d -> %d for event type %s", v, v+1, eventType)
		}
	}

	// Validate versions map includes current version
	if _, ok := versions[currentVersion]; !ok {
		return fmt.Errorf("versions map must include current version %d for event type %s", currentVersion, eventType)
	}

	r.configs[eventType] = &VersionedEventConfig{
		EventType:      eventType,
		CurrentVersion: currentVersion,
		Upgraders:      upgraderMap,
		Versions:       versions,
	}

	return nil
}

// RegisterSimpleEvent registers an event type that has only version 1 (no migrations)
// This is a convenience method for events that don't need versioning yet
func (r *VersionRegistry) RegisterSimpleEvent(eventType string, eventInstance shared.DomainEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.configs[eventType] = &VersionedEventConfig{
		EventType:      eventType,
		CurrentVersion: 1,
		Upgraders:      make(map[int]EventUpgrader),
		Versions: map[int]shared.DomainEvent{
			1: eventInstance,
		},
	}
}

// GetConfig returns the versioning config for an event type
func (r *VersionRegistry) GetConfig(eventType string) (*VersionedEventConfig, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	config, ok := r.configs[eventType]
	return config, ok
}

// GetCurrentVersion returns the current (latest) version for an event type
func (r *VersionRegistry) GetCurrentVersion(eventType string) (int, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	config, ok := r.configs[eventType]
	if !ok {
		return 0, false
	}
	return config.CurrentVersion, true
}

// IsRegistered checks if an event type is registered
func (r *VersionRegistry) IsRegistered(eventType string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.configs[eventType]
	return ok
}

// RegisteredTypes returns all registered event types
func (r *VersionRegistry) RegisteredTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.configs))
	for t := range r.configs {
		types = append(types, t)
	}
	return types
}

// UpgradePayload upgrades an event payload from its current version to the latest version
func (r *VersionRegistry) UpgradePayload(eventType string, payload []byte, fromVersion int) ([]byte, int, error) {
	r.mu.RLock()
	config, ok := r.configs[eventType]
	r.mu.RUnlock()

	if !ok {
		return nil, 0, fmt.Errorf("unknown event type: %s", eventType)
	}

	// If already at current version, return as-is
	if fromVersion >= config.CurrentVersion {
		return payload, config.CurrentVersion, nil
	}

	// Apply upgraders sequentially
	currentPayload := payload
	var err error
	for v := fromVersion; v < config.CurrentVersion; v++ {
		upgrader, ok := config.Upgraders[v]
		if !ok {
			return nil, 0, fmt.Errorf("missing upgrader for version %d -> %d for event type %s", v, v+1, eventType)
		}
		currentPayload, err = upgrader.Upgrade(currentPayload)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to upgrade from v%d to v%d: %w", v, v+1, err)
		}
	}

	return currentPayload, config.CurrentVersion, nil
}

// EventVersionInfo extracts version info from raw event JSON
type EventVersionInfo struct {
	SchemaVersion int `json:"schema_version"`
}

// ExtractVersion extracts the schema version from raw event JSON
// Returns 1 if no version field is present (backward compatibility)
func ExtractVersion(payload []byte) int {
	var info EventVersionInfo
	if err := json.Unmarshal(payload, &info); err != nil {
		return 1 // Default to version 1 if parsing fails
	}
	if info.SchemaVersion == 0 {
		return 1 // Default to version 1 if not specified
	}
	return info.SchemaVersion
}

// BaseEventUpgrader provides a base implementation for event upgraders
// that work by unmarshaling to a map, transforming, and marshaling back
type BaseEventUpgrader struct {
	sourceVersion int
	targetVersion int
	transformFunc func(data map[string]any) (map[string]any, error)
}

// NewBaseEventUpgrader creates a new base event upgrader
func NewBaseEventUpgrader(source, target int, transform func(data map[string]any) (map[string]any, error)) *BaseEventUpgrader {
	return &BaseEventUpgrader{
		sourceVersion: source,
		targetVersion: target,
		transformFunc: transform,
	}
}

// SourceVersion returns the source version
func (u *BaseEventUpgrader) SourceVersion() int {
	return u.sourceVersion
}

// TargetVersion returns the target version
func (u *BaseEventUpgrader) TargetVersion() int {
	return u.targetVersion
}

// Upgrade transforms the payload from source to target version
func (u *BaseEventUpgrader) Upgrade(payload []byte) ([]byte, error) {
	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	transformed, err := u.transformFunc(data)
	if err != nil {
		return nil, fmt.Errorf("transform failed: %w", err)
	}

	// Update schema version in the transformed data
	transformed["schema_version"] = u.targetVersion

	result, err := json.Marshal(transformed)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transformed payload: %w", err)
	}

	return result, nil
}

// Ensure BaseEventUpgrader implements EventUpgrader
var _ EventUpgrader = (*BaseEventUpgrader)(nil)
