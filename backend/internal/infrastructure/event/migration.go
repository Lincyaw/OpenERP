package event

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// MigrationResult holds the result of a batch migration
type MigrationResult struct {
	EventType      string
	TotalProcessed int
	Upgraded       int
	AlreadyCurrent int
	Failed         int
	FailedPayloads []FailedMigration
	StartedAt      time.Time
	CompletedAt    time.Time
	FromVersion    int
	ToVersion      int
}

// FailedMigration holds information about a failed migration
type FailedMigration struct {
	Payload []byte
	Error   string
	Version int
}

// Duration returns the migration duration
func (r *MigrationResult) Duration() time.Duration {
	return r.CompletedAt.Sub(r.StartedAt)
}

// EventMigrator provides utilities for batch migration of events
type EventMigrator struct {
	serializer *VersionedSerializer
	logger     *zap.Logger
}

// NewEventMigrator creates a new event migrator
func NewEventMigrator(serializer *VersionedSerializer, logger *zap.Logger) *EventMigrator {
	return &EventMigrator{
		serializer: serializer,
		logger:     logger,
	}
}

// MigratePayloads migrates a batch of event payloads to the current version
// This is useful for batch processing events from storage or message queues
func (m *EventMigrator) MigratePayloads(ctx context.Context, eventType string, payloads [][]byte) (*MigrationResult, error) {
	result := &MigrationResult{
		EventType:      eventType,
		StartedAt:      time.Now(),
		FailedPayloads: make([]FailedMigration, 0),
	}

	currentVersion, ok := m.serializer.GetCurrentVersion(eventType)
	if !ok {
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}
	result.ToVersion = currentVersion

	for _, payload := range payloads {
		select {
		case <-ctx.Done():
			result.CompletedAt = time.Now()
			return result, ctx.Err()
		default:
		}

		result.TotalProcessed++
		version := ExtractVersion(payload)

		if result.FromVersion == 0 || version < result.FromVersion {
			result.FromVersion = version
		}

		if version >= currentVersion {
			result.AlreadyCurrent++
			continue
		}

		_, _, err := m.serializer.UpgradePayloadOnly(eventType, payload)
		if err != nil {
			result.Failed++
			result.FailedPayloads = append(result.FailedPayloads, FailedMigration{
				Payload: payload,
				Error:   err.Error(),
				Version: version,
			})
			continue
		}

		result.Upgraded++
	}

	result.CompletedAt = time.Now()
	return result, nil
}

// MigratePayload migrates a single event payload to the current version
// Returns the upgraded payload and its new version
func (m *EventMigrator) MigratePayload(eventType string, payload []byte) ([]byte, int, error) {
	return m.serializer.UpgradePayloadOnly(eventType, payload)
}

// ValidateUpgradeChain validates that all upgraders are correctly chained
// for a given event type. Returns an error if there are gaps in the chain.
func (m *EventMigrator) ValidateUpgradeChain(eventType string) error {
	config, ok := m.serializer.GetVersionRegistry().GetConfig(eventType)
	if !ok {
		return fmt.Errorf("unknown event type: %s", eventType)
	}

	for v := 1; v < config.CurrentVersion; v++ {
		if _, ok := config.Upgraders[v]; !ok {
			return fmt.Errorf("missing upgrader for version %d -> %d", v, v+1)
		}
	}

	return nil
}

// EventVersionAnalysis contains analysis of event versions in a payload set
type EventVersionAnalysis struct {
	EventType      string
	CurrentVersion int
	VersionCounts  map[int]int
	OldestVersion  int
	NewestVersion  int
	TotalEvents    int
	NeedsMigration int
	UpToDate       int
}

// AnalyzePayloads analyzes a batch of event payloads to understand version distribution
func (m *EventMigrator) AnalyzePayloads(eventType string, payloads [][]byte) (*EventVersionAnalysis, error) {
	currentVersion, ok := m.serializer.GetCurrentVersion(eventType)
	if !ok {
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}

	analysis := &EventVersionAnalysis{
		EventType:      eventType,
		CurrentVersion: currentVersion,
		VersionCounts:  make(map[int]int),
		OldestVersion:  -1,
		NewestVersion:  -1,
		TotalEvents:    len(payloads),
	}

	for _, payload := range payloads {
		version := ExtractVersion(payload)
		analysis.VersionCounts[version]++

		if analysis.OldestVersion == -1 || version < analysis.OldestVersion {
			analysis.OldestVersion = version
		}
		if version > analysis.NewestVersion {
			analysis.NewestVersion = version
		}

		if version < currentVersion {
			analysis.NeedsMigration++
		} else {
			analysis.UpToDate++
		}
	}

	return analysis, nil
}

// MigrationPlan represents a plan for migrating events
type MigrationPlan struct {
	EventType        string
	FromVersion      int
	ToVersion        int
	UpgradeSteps     []UpgradeStep
	EstimatedPayload int
}

// UpgradeStep represents a single upgrade step in a migration plan
type UpgradeStep struct {
	FromVersion int
	ToVersion   int
	HasUpgrader bool
}

// CreateMigrationPlan creates a migration plan for a specific event type
func (m *EventMigrator) CreateMigrationPlan(eventType string, fromVersion int) (*MigrationPlan, error) {
	config, ok := m.serializer.GetVersionRegistry().GetConfig(eventType)
	if !ok {
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}

	if fromVersion >= config.CurrentVersion {
		return &MigrationPlan{
			EventType:    eventType,
			FromVersion:  fromVersion,
			ToVersion:    config.CurrentVersion,
			UpgradeSteps: []UpgradeStep{},
		}, nil
	}

	steps := make([]UpgradeStep, 0, config.CurrentVersion-fromVersion)
	for v := fromVersion; v < config.CurrentVersion; v++ {
		_, hasUpgrader := config.Upgraders[v]
		steps = append(steps, UpgradeStep{
			FromVersion: v,
			ToVersion:   v + 1,
			HasUpgrader: hasUpgrader,
		})
	}

	return &MigrationPlan{
		EventType:    eventType,
		FromVersion:  fromVersion,
		ToVersion:    config.CurrentVersion,
		UpgradeSteps: steps,
	}, nil
}

// IsValid checks if the migration plan is valid (all upgraders exist)
func (p *MigrationPlan) IsValid() bool {
	for _, step := range p.UpgradeSteps {
		if !step.HasUpgrader {
			return false
		}
	}
	return true
}

// CommonUpgraders provides factory functions for common upgrade patterns
type CommonUpgraders struct{}

// AddField creates an upgrader that adds a new field with a default value
func (CommonUpgraders) AddField(sourceVersion int, fieldName string, defaultValue any) *BaseEventUpgrader {
	return NewBaseEventUpgrader(sourceVersion, sourceVersion+1, func(data map[string]any) (map[string]any, error) {
		data[fieldName] = defaultValue
		return data, nil
	})
}

// RemoveField creates an upgrader that removes a field
func (CommonUpgraders) RemoveField(sourceVersion int, fieldName string) *BaseEventUpgrader {
	return NewBaseEventUpgrader(sourceVersion, sourceVersion+1, func(data map[string]any) (map[string]any, error) {
		delete(data, fieldName)
		return data, nil
	})
}

// RenameField creates an upgrader that renames a field
func (CommonUpgraders) RenameField(sourceVersion int, oldName, newName string) *BaseEventUpgrader {
	return NewBaseEventUpgrader(sourceVersion, sourceVersion+1, func(data map[string]any) (map[string]any, error) {
		if val, ok := data[oldName]; ok {
			data[newName] = val
			delete(data, oldName)
		}
		return data, nil
	})
}

// TransformField creates an upgrader that transforms a field value
func (CommonUpgraders) TransformField(sourceVersion int, fieldName string, transform func(any) any) *BaseEventUpgrader {
	return NewBaseEventUpgrader(sourceVersion, sourceVersion+1, func(data map[string]any) (map[string]any, error) {
		if val, ok := data[fieldName]; ok {
			data[fieldName] = transform(val)
		}
		return data, nil
	})
}

// SplitField creates an upgrader that splits one field into multiple fields
func (CommonUpgraders) SplitField(sourceVersion int, sourceName string, splitter func(any) map[string]any) *BaseEventUpgrader {
	return NewBaseEventUpgrader(sourceVersion, sourceVersion+1, func(data map[string]any) (map[string]any, error) {
		if val, ok := data[sourceName]; ok {
			newFields := splitter(val)
			for k, v := range newFields {
				data[k] = v
			}
			delete(data, sourceName)
		}
		return data, nil
	})
}

// MergeFields creates an upgrader that merges multiple fields into one
func (CommonUpgraders) MergeFields(sourceVersion int, fieldNames []string, targetName string, merger func(map[string]any) any) *BaseEventUpgrader {
	return NewBaseEventUpgrader(sourceVersion, sourceVersion+1, func(data map[string]any) (map[string]any, error) {
		values := make(map[string]any)
		for _, name := range fieldNames {
			if val, ok := data[name]; ok {
				values[name] = val
				delete(data, name)
			}
		}
		data[targetName] = merger(values)
		return data, nil
	})
}

// SetFieldType creates an upgrader that converts a field to a different type
func (CommonUpgraders) SetFieldType(sourceVersion int, fieldName string, converter func(any) (any, error)) *BaseEventUpgrader {
	return NewBaseEventUpgrader(sourceVersion, sourceVersion+1, func(data map[string]any) (map[string]any, error) {
		if val, ok := data[fieldName]; ok {
			newVal, err := converter(val)
			if err != nil {
				return nil, fmt.Errorf("failed to convert field %s: %w", fieldName, err)
			}
			data[fieldName] = newVal
		}
		return data, nil
	})
}

// WrapInObject creates an upgrader that wraps a field value in a nested object
func (CommonUpgraders) WrapInObject(sourceVersion int, fieldName, wrapperKey string) *BaseEventUpgrader {
	return NewBaseEventUpgrader(sourceVersion, sourceVersion+1, func(data map[string]any) (map[string]any, error) {
		if val, ok := data[fieldName]; ok {
			data[fieldName] = map[string]any{wrapperKey: val}
		}
		return data, nil
	})
}

// UnwrapFromObject creates an upgrader that unwraps a field value from a nested object
func (CommonUpgraders) UnwrapFromObject(sourceVersion int, fieldName, wrapperKey string) *BaseEventUpgrader {
	return NewBaseEventUpgrader(sourceVersion, sourceVersion+1, func(data map[string]any) (map[string]any, error) {
		if val, ok := data[fieldName]; ok {
			if obj, ok := val.(map[string]any); ok {
				if unwrapped, ok := obj[wrapperKey]; ok {
					data[fieldName] = unwrapped
				}
			}
		}
		return data, nil
	})
}

// CopyPayload creates a deep copy of an event payload
func CopyPayload(payload []byte) ([]byte, error) {
	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, err
	}
	return json.Marshal(data)
}

// MigrationStats holds statistics for ongoing migrations
type MigrationStats struct {
	mu    sync.RWMutex
	stats map[string]*EventMigrationStats
}

// EventMigrationStats holds migration stats for a single event type
type EventMigrationStats struct {
	EventType           string
	TotalMigrated       int64
	TotalFailed         int64
	LastMigratedAt      time.Time
	AverageDurationMs   float64
	MigrationsByVersion map[string]int64 // "v1->v2" => count
}

// NewMigrationStats creates a new migration stats tracker
func NewMigrationStats() *MigrationStats {
	return &MigrationStats{
		stats: make(map[string]*EventMigrationStats),
	}
}

// RecordMigration records a migration event
func (s *MigrationStats) RecordMigration(eventType string, fromVersion, toVersion int, durationMs float64, success bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.stats[eventType]; !ok {
		s.stats[eventType] = &EventMigrationStats{
			EventType:           eventType,
			MigrationsByVersion: make(map[string]int64),
		}
	}

	stats := s.stats[eventType]
	if success {
		stats.TotalMigrated++
		stats.LastMigratedAt = time.Now()
		// Update rolling average
		n := float64(stats.TotalMigrated)
		stats.AverageDurationMs = stats.AverageDurationMs*(n-1)/n + durationMs/n
	} else {
		stats.TotalFailed++
	}

	key := fmt.Sprintf("v%d->v%d", fromVersion, toVersion)
	stats.MigrationsByVersion[key]++
}

// GetStats returns stats for an event type
func (s *MigrationStats) GetStats(eventType string) (*EventMigrationStats, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stats, ok := s.stats[eventType]
	if !ok {
		return nil, false
	}
	// Return a copy
	statsCopy := *stats
	statsCopy.MigrationsByVersion = make(map[string]int64)
	for k, v := range stats.MigrationsByVersion {
		statsCopy.MigrationsByVersion[k] = v
	}
	return &statsCopy, true
}
