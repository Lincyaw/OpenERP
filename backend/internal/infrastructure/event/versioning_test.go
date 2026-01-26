package event

import (
	"context"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Test event types for versioning tests

// TestEventV1 is version 1 of the test event
type TestEventV1 struct {
	shared.BaseDomainEvent
	Name string `json:"name"`
}

// TestEventV2 adds email field
type TestEventV2 struct {
	shared.BaseDomainEvent
	Name  string `json:"name"`
	Email string `json:"email"`
}

// TestEventV3 adds age and renames email to contact_email
type TestEventV3 struct {
	shared.BaseDomainEvent
	Name         string `json:"name"`
	ContactEmail string `json:"contact_email"`
	Age          int    `json:"age"`
}

func newTestEventV1() *TestEventV1 {
	return &TestEventV1{
		BaseDomainEvent: shared.NewVersionedBaseDomainEvent("TestEvent", "TestAggregate", uuid.New(), uuid.New(), 1),
		Name:            "John Doe",
	}
}

func newTestEventV2() *TestEventV2 {
	return &TestEventV2{
		BaseDomainEvent: shared.NewVersionedBaseDomainEvent("TestEvent", "TestAggregate", uuid.New(), uuid.New(), 2),
		Name:            "John Doe",
		Email:           "john@example.com",
	}
}

func newTestEventV3() *TestEventV3 {
	return &TestEventV3{
		BaseDomainEvent: shared.NewVersionedBaseDomainEvent("TestEvent", "TestAggregate", uuid.New(), uuid.New(), 3),
		Name:            "John Doe",
		ContactEmail:    "john@example.com",
		Age:             30,
	}
}

// Test upgraders
func testEventV1ToV2Upgrader() EventUpgrader {
	return NewBaseEventUpgrader(1, 2, func(data map[string]any) (map[string]any, error) {
		data["email"] = "unknown@example.com"
		return data, nil
	})
}

func testEventV2ToV3Upgrader() EventUpgrader {
	return NewBaseEventUpgrader(2, 3, func(data map[string]any) (map[string]any, error) {
		// Rename email to contact_email
		if email, ok := data["email"]; ok {
			data["contact_email"] = email
			delete(data, "email")
		}
		// Add age with default
		data["age"] = 0
		return data, nil
	})
}

func TestVersionRegistry_RegisterSimpleEvent(t *testing.T) {
	registry := NewVersionRegistry()

	registry.RegisterSimpleEvent("TestEvent", &TestEventV1{})

	assert.True(t, registry.IsRegistered("TestEvent"))

	config, ok := registry.GetConfig("TestEvent")
	require.True(t, ok)
	assert.Equal(t, 1, config.CurrentVersion)
	assert.Empty(t, config.Upgraders)
}

func TestVersionRegistry_RegisterVersionedEvent(t *testing.T) {
	registry := NewVersionRegistry()

	err := registry.RegisterVersionedEvent(
		"TestEvent",
		3,
		map[int]shared.DomainEvent{
			1: &TestEventV1{},
			2: &TestEventV2{},
			3: &TestEventV3{},
		},
		testEventV1ToV2Upgrader(),
		testEventV2ToV3Upgrader(),
	)

	require.NoError(t, err)
	assert.True(t, registry.IsRegistered("TestEvent"))

	version, ok := registry.GetCurrentVersion("TestEvent")
	require.True(t, ok)
	assert.Equal(t, 3, version)
}

func TestVersionRegistry_RegisterVersionedEvent_MissingUpgrader(t *testing.T) {
	registry := NewVersionRegistry()

	err := registry.RegisterVersionedEvent(
		"TestEvent",
		3,
		map[int]shared.DomainEvent{
			1: &TestEventV1{},
			2: &TestEventV2{},
			3: &TestEventV3{},
		},
		testEventV1ToV2Upgrader(), // Missing v2->v3
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing upgrader for version 2 -> 3")
}

func TestVersionRegistry_RegisterVersionedEvent_NonSequentialUpgrader(t *testing.T) {
	registry := NewVersionRegistry()

	// Create an invalid upgrader that skips versions
	badUpgrader := NewBaseEventUpgrader(1, 3, func(data map[string]any) (map[string]any, error) {
		return data, nil
	})

	err := registry.RegisterVersionedEvent(
		"TestEvent",
		3,
		map[int]shared.DomainEvent{
			1: &TestEventV1{},
			2: &TestEventV2{},
			3: &TestEventV3{},
		},
		badUpgrader,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "upgrader must be sequential")
}

func TestVersionRegistry_UpgradePayload(t *testing.T) {
	registry := NewVersionRegistry()

	err := registry.RegisterVersionedEvent(
		"TestEvent",
		3,
		map[int]shared.DomainEvent{
			1: &TestEventV1{},
			2: &TestEventV2{},
			3: &TestEventV3{},
		},
		testEventV1ToV2Upgrader(),
		testEventV2ToV3Upgrader(),
	)
	require.NoError(t, err)

	// Create v1 payload
	v1Event := newTestEventV1()
	v1Serializer := NewEventSerializer()
	v1Data, err := v1Serializer.Serialize(v1Event)
	require.NoError(t, err)

	// Upgrade from v1 to v3
	upgraded, version, err := registry.UpgradePayload("TestEvent", v1Data, 1)
	require.NoError(t, err)
	assert.Equal(t, 3, version)

	// Verify the upgraded payload contains v3 fields
	assert.Contains(t, string(upgraded), "contact_email")
	assert.Contains(t, string(upgraded), "age")
	assert.NotContains(t, string(upgraded), `"email":`)
}

func TestVersionRegistry_UpgradePayload_AlreadyCurrent(t *testing.T) {
	registry := NewVersionRegistry()
	registry.RegisterSimpleEvent("TestEvent", &TestEventV1{})

	payload := []byte(`{"schema_version": 1, "name": "test"}`)

	upgraded, version, err := registry.UpgradePayload("TestEvent", payload, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, version)
	assert.Equal(t, payload, upgraded)
}

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		name     string
		payload  string
		expected int
	}{
		{"with version", `{"schema_version": 2, "name": "test"}`, 2},
		{"without version", `{"name": "test"}`, 1},
		{"version zero", `{"schema_version": 0, "name": "test"}`, 1},
		{"invalid json", `invalid`, 1},
		{"empty", `{}`, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version := ExtractVersion([]byte(tt.payload))
			assert.Equal(t, tt.expected, version)
		})
	}
}

func TestBaseEventUpgrader(t *testing.T) {
	upgrader := NewBaseEventUpgrader(1, 2, func(data map[string]any) (map[string]any, error) {
		data["new_field"] = "added"
		return data, nil
	})

	assert.Equal(t, 1, upgrader.SourceVersion())
	assert.Equal(t, 2, upgrader.TargetVersion())

	input := []byte(`{"schema_version": 1, "existing": "value"}`)
	output, err := upgrader.Upgrade(input)
	require.NoError(t, err)

	assert.Contains(t, string(output), `"new_field":"added"`)
	assert.Contains(t, string(output), `"schema_version":2`)
}

func TestVersionedSerializer_Register_Backward_Compatible(t *testing.T) {
	logger := zap.NewNop()
	serializer := NewVersionedSerializer(logger)

	// Simple registration should work just like EventSerializer
	serializer.Register("TestEvent", &TestEventV1{})

	assert.True(t, serializer.IsRegistered("TestEvent"))

	version, ok := serializer.GetCurrentVersion("TestEvent")
	require.True(t, ok)
	assert.Equal(t, 1, version)
}

func TestVersionedSerializer_Serialize(t *testing.T) {
	logger := zap.NewNop()
	serializer := NewVersionedSerializer(logger)

	event := newTestEventV3()
	data, err := serializer.Serialize(event)
	require.NoError(t, err)

	assert.Contains(t, string(data), `"schema_version":3`)
	assert.Contains(t, string(data), `"name":"John Doe"`)
}

func TestVersionedSerializer_Deserialize_CurrentVersion(t *testing.T) {
	logger := zap.NewNop()
	serializer := NewVersionedSerializer(logger)

	err := serializer.RegisterVersioned(
		"TestEvent",
		3,
		map[int]shared.DomainEvent{
			1: &TestEventV1{},
			2: &TestEventV2{},
			3: &TestEventV3{},
		},
		testEventV1ToV2Upgrader(),
		testEventV2ToV3Upgrader(),
	)
	require.NoError(t, err)

	// Serialize v3 event
	original := newTestEventV3()
	data, err := serializer.Serialize(original)
	require.NoError(t, err)

	// Deserialize - should get v3 back
	deserialized, err := serializer.Deserialize("TestEvent", data)
	require.NoError(t, err)

	event, ok := deserialized.(*TestEventV3)
	require.True(t, ok)
	assert.Equal(t, original.Name, event.Name)
	assert.Equal(t, original.ContactEmail, event.ContactEmail)
	assert.Equal(t, original.Age, event.Age)
}

func TestVersionedSerializer_Deserialize_FromV2ToLatest(t *testing.T) {
	logger := zap.NewNop()
	serializer := NewVersionedSerializer(logger)

	err := serializer.RegisterVersioned(
		"TestEvent",
		3,
		map[int]shared.DomainEvent{
			1: &TestEventV1{},
			2: &TestEventV2{},
			3: &TestEventV3{},
		},
		testEventV1ToV2Upgrader(),
		testEventV2ToV3Upgrader(),
	)
	require.NoError(t, err)

	// Use the v2 event helper
	v2Event := newTestEventV2()
	data, err := serializer.Serialize(v2Event)
	require.NoError(t, err)

	// Deserialize - should upgrade from v2 to v3
	deserialized, err := serializer.Deserialize("TestEvent", data)
	require.NoError(t, err)

	event, ok := deserialized.(*TestEventV3)
	require.True(t, ok)
	assert.Equal(t, v2Event.Name, event.Name)
	assert.Equal(t, v2Event.Email, event.ContactEmail) // email becomes contact_email in v3
	assert.Equal(t, 0, event.Age)                      // Age is new in v3 with default 0
}

func TestVersionedSerializer_Deserialize_WithUpgrade(t *testing.T) {
	logger := zap.NewNop()
	serializer := NewVersionedSerializer(logger)

	err := serializer.RegisterVersioned(
		"TestEvent",
		3,
		map[int]shared.DomainEvent{
			1: &TestEventV1{},
			2: &TestEventV2{},
			3: &TestEventV3{},
		},
		testEventV1ToV2Upgrader(),
		testEventV2ToV3Upgrader(),
	)
	require.NoError(t, err)

	// Create v1 payload manually (simulating old stored event)
	v1Payload := []byte(`{
		"id": "00000000-0000-0000-0000-000000000001",
		"type": "TestEvent",
		"timestamp": "2024-01-01T00:00:00Z",
		"aggregate_id": "00000000-0000-0000-0000-000000000002",
		"aggregate_type": "TestAggregate",
		"tenant_id": "00000000-0000-0000-0000-000000000003",
		"schema_version": 1,
		"name": "Legacy User"
	}`)

	// Deserialize - should upgrade to v3
	deserialized, err := serializer.Deserialize("TestEvent", v1Payload)
	require.NoError(t, err)

	event, ok := deserialized.(*TestEventV3)
	require.True(t, ok)
	assert.Equal(t, "Legacy User", event.Name)
	assert.Equal(t, "unknown@example.com", event.ContactEmail) // From v1->v2 upgrade
	assert.Equal(t, 0, event.Age)                              // From v2->v3 upgrade
}

func TestVersionedSerializer_Deserialize_NoVersionField(t *testing.T) {
	logger := zap.NewNop()
	serializer := NewVersionedSerializer(logger)

	err := serializer.RegisterVersioned(
		"TestEvent",
		2,
		map[int]shared.DomainEvent{
			1: &TestEventV1{},
			2: &TestEventV2{},
		},
		testEventV1ToV2Upgrader(),
	)
	require.NoError(t, err)

	// Payload without version field (treated as v1)
	payload := []byte(`{
		"id": "00000000-0000-0000-0000-000000000001",
		"type": "TestEvent",
		"timestamp": "2024-01-01T00:00:00Z",
		"aggregate_id": "00000000-0000-0000-0000-000000000002",
		"aggregate_type": "TestAggregate",
		"tenant_id": "00000000-0000-0000-0000-000000000003",
		"name": "No Version User"
	}`)

	deserialized, err := serializer.Deserialize("TestEvent", payload)
	require.NoError(t, err)

	event, ok := deserialized.(*TestEventV2)
	require.True(t, ok)
	assert.Equal(t, "No Version User", event.Name)
	assert.Equal(t, "unknown@example.com", event.Email)
}

func TestVersionedSerializer_Deserialize_UnknownType(t *testing.T) {
	logger := zap.NewNop()
	serializer := NewVersionedSerializer(logger)

	_, err := serializer.Deserialize("UnknownEvent", []byte(`{}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown event type")
}

func TestVersionedSerializer_DeserializeToVersion(t *testing.T) {
	logger := zap.NewNop()
	serializer := NewVersionedSerializer(logger)

	err := serializer.RegisterVersioned(
		"TestEvent",
		3,
		map[int]shared.DomainEvent{
			1: &TestEventV1{},
			2: &TestEventV2{},
			3: &TestEventV3{},
		},
		testEventV1ToV2Upgrader(),
		testEventV2ToV3Upgrader(),
	)
	require.NoError(t, err)

	// Create v1 payload
	v1Payload := []byte(`{
		"id": "00000000-0000-0000-0000-000000000001",
		"type": "TestEvent",
		"timestamp": "2024-01-01T00:00:00Z",
		"aggregate_id": "00000000-0000-0000-0000-000000000002",
		"aggregate_type": "TestAggregate",
		"tenant_id": "00000000-0000-0000-0000-000000000003",
		"schema_version": 1,
		"name": "Test User"
	}`)

	// Deserialize to v2 (not current v3)
	deserialized, err := serializer.DeserializeToVersion("TestEvent", v1Payload, 2)
	require.NoError(t, err)

	event, ok := deserialized.(*TestEventV2)
	require.True(t, ok)
	assert.Equal(t, "Test User", event.Name)
	assert.Equal(t, "unknown@example.com", event.Email) // From v1->v2 upgrade
}

func TestVersionedSerializer_DeserializeToVersion_CannotDowngrade(t *testing.T) {
	logger := zap.NewNop()
	serializer := NewVersionedSerializer(logger)

	err := serializer.RegisterVersioned(
		"TestEvent",
		3,
		map[int]shared.DomainEvent{
			1: &TestEventV1{},
			2: &TestEventV2{},
			3: &TestEventV3{},
		},
		testEventV1ToV2Upgrader(),
		testEventV2ToV3Upgrader(),
	)
	require.NoError(t, err)

	// Create v3 payload
	v3Payload := []byte(`{
		"schema_version": 3,
		"name": "Test User"
	}`)

	// Try to deserialize v3 payload to v1 - should fail
	_, err = serializer.DeserializeToVersion("TestEvent", v3Payload, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot downgrade")
}

func TestVersionedSerializer_DeserializeToVersion_UnknownType(t *testing.T) {
	logger := zap.NewNop()
	serializer := NewVersionedSerializer(logger)

	_, err := serializer.DeserializeToVersion("UnknownEvent", []byte(`{}`), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown event type")
}

func TestVersionedSerializer_RegisteredTypes(t *testing.T) {
	logger := zap.NewNop()
	serializer := NewVersionedSerializer(logger)

	serializer.Register("Event1", &TestEventV1{})
	serializer.Register("Event2", &TestEventV1{})

	types := serializer.RegisteredTypes()
	assert.Len(t, types, 2)
	assert.Contains(t, types, "Event1")
	assert.Contains(t, types, "Event2")
}

func TestCommonUpgraders_AddField(t *testing.T) {
	upgraders := CommonUpgraders{}
	u := upgraders.AddField(1, "new_field", "default_value")

	input := []byte(`{"schema_version": 1, "existing": "value"}`)
	output, err := u.Upgrade(input)
	require.NoError(t, err)

	assert.Contains(t, string(output), `"new_field":"default_value"`)
}

func TestCommonUpgraders_RemoveField(t *testing.T) {
	upgraders := CommonUpgraders{}
	u := upgraders.RemoveField(1, "old_field")

	input := []byte(`{"schema_version": 1, "old_field": "remove_me", "keep": "value"}`)
	output, err := u.Upgrade(input)
	require.NoError(t, err)

	assert.NotContains(t, string(output), "old_field")
	assert.Contains(t, string(output), `"keep":"value"`)
}

func TestCommonUpgraders_RenameField(t *testing.T) {
	upgraders := CommonUpgraders{}
	u := upgraders.RenameField(1, "old_name", "new_name")

	input := []byte(`{"schema_version": 1, "old_name": "value"}`)
	output, err := u.Upgrade(input)
	require.NoError(t, err)

	assert.NotContains(t, string(output), "old_name")
	assert.Contains(t, string(output), `"new_name":"value"`)
}

func TestCommonUpgraders_TransformField(t *testing.T) {
	upgraders := CommonUpgraders{}
	u := upgraders.TransformField(1, "amount", func(v any) any {
		if num, ok := v.(float64); ok {
			return num * 100 // Convert to cents
		}
		return v
	})

	input := []byte(`{"schema_version": 1, "amount": 10.5}`)
	output, err := u.Upgrade(input)
	require.NoError(t, err)

	assert.Contains(t, string(output), `"amount":1050`)
}

func TestCommonUpgraders_WrapInObject(t *testing.T) {
	upgraders := CommonUpgraders{}
	u := upgraders.WrapInObject(1, "value", "amount")

	input := []byte(`{"schema_version": 1, "value": 100}`)
	output, err := u.Upgrade(input)
	require.NoError(t, err)

	assert.Contains(t, string(output), `"value":{"amount":100}`)
}

func TestCommonUpgraders_UnwrapFromObject(t *testing.T) {
	upgraders := CommonUpgraders{}
	u := upgraders.UnwrapFromObject(1, "value", "amount")

	input := []byte(`{"schema_version": 1, "value": {"amount": 100, "other": "x"}}`)
	output, err := u.Upgrade(input)
	require.NoError(t, err)

	assert.Contains(t, string(output), `"value":100`)
}

func TestEventMigrator_MigratePayloads(t *testing.T) {
	logger := zap.NewNop()
	serializer := NewVersionedSerializer(logger)

	err := serializer.RegisterVersioned(
		"TestEvent",
		2,
		map[int]shared.DomainEvent{
			1: &TestEventV1{},
			2: &TestEventV2{},
		},
		testEventV1ToV2Upgrader(),
	)
	require.NoError(t, err)

	migrator := NewEventMigrator(serializer, logger)

	payloads := [][]byte{
		[]byte(`{"schema_version": 1, "name": "User 1"}`),
		[]byte(`{"schema_version": 1, "name": "User 2"}`),
		[]byte(`{"schema_version": 2, "name": "User 3", "email": "u3@test.com"}`),
	}

	ctx := context.Background()
	result, err := migrator.MigratePayloads(ctx, "TestEvent", payloads)
	require.NoError(t, err)

	assert.Equal(t, 3, result.TotalProcessed)
	assert.Equal(t, 2, result.Upgraded)
	assert.Equal(t, 1, result.AlreadyCurrent)
	assert.Equal(t, 0, result.Failed)
	assert.Equal(t, 2, result.ToVersion)
}

func TestEventMigrator_MigratePayloads_WithCancellation(t *testing.T) {
	logger := zap.NewNop()
	serializer := NewVersionedSerializer(logger)
	serializer.Register("TestEvent", &TestEventV1{})

	migrator := NewEventMigrator(serializer, logger)

	payloads := make([][]byte, 100)
	for i := range payloads {
		payloads[i] = []byte(`{"schema_version": 1, "name": "test"}`)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := migrator.MigratePayloads(ctx, "TestEvent", payloads)
	assert.Error(t, err)
	assert.True(t, result.TotalProcessed < 100)
}

func TestEventMigrator_AnalyzePayloads(t *testing.T) {
	logger := zap.NewNop()
	serializer := NewVersionedSerializer(logger)

	err := serializer.RegisterVersioned(
		"TestEvent",
		3,
		map[int]shared.DomainEvent{
			1: &TestEventV1{},
			2: &TestEventV2{},
			3: &TestEventV3{},
		},
		testEventV1ToV2Upgrader(),
		testEventV2ToV3Upgrader(),
	)
	require.NoError(t, err)

	migrator := NewEventMigrator(serializer, logger)

	payloads := [][]byte{
		[]byte(`{"schema_version": 1}`),
		[]byte(`{"schema_version": 1}`),
		[]byte(`{"schema_version": 2}`),
		[]byte(`{"schema_version": 3}`),
	}

	analysis, err := migrator.AnalyzePayloads("TestEvent", payloads)
	require.NoError(t, err)

	assert.Equal(t, "TestEvent", analysis.EventType)
	assert.Equal(t, 3, analysis.CurrentVersion)
	assert.Equal(t, 4, analysis.TotalEvents)
	assert.Equal(t, 3, analysis.NeedsMigration)
	assert.Equal(t, 1, analysis.UpToDate)
	assert.Equal(t, 1, analysis.OldestVersion)
	assert.Equal(t, 3, analysis.NewestVersion)
	assert.Equal(t, 2, analysis.VersionCounts[1])
	assert.Equal(t, 1, analysis.VersionCounts[2])
	assert.Equal(t, 1, analysis.VersionCounts[3])
}

func TestEventMigrator_ValidateUpgradeChain(t *testing.T) {
	logger := zap.NewNop()
	serializer := NewVersionedSerializer(logger)

	err := serializer.RegisterVersioned(
		"TestEvent",
		3,
		map[int]shared.DomainEvent{
			1: &TestEventV1{},
			2: &TestEventV2{},
			3: &TestEventV3{},
		},
		testEventV1ToV2Upgrader(),
		testEventV2ToV3Upgrader(),
	)
	require.NoError(t, err)

	migrator := NewEventMigrator(serializer, logger)

	err = migrator.ValidateUpgradeChain("TestEvent")
	assert.NoError(t, err)

	err = migrator.ValidateUpgradeChain("UnknownEvent")
	assert.Error(t, err)
}

func TestEventMigrator_CreateMigrationPlan(t *testing.T) {
	logger := zap.NewNop()
	serializer := NewVersionedSerializer(logger)

	err := serializer.RegisterVersioned(
		"TestEvent",
		3,
		map[int]shared.DomainEvent{
			1: &TestEventV1{},
			2: &TestEventV2{},
			3: &TestEventV3{},
		},
		testEventV1ToV2Upgrader(),
		testEventV2ToV3Upgrader(),
	)
	require.NoError(t, err)

	migrator := NewEventMigrator(serializer, logger)

	plan, err := migrator.CreateMigrationPlan("TestEvent", 1)
	require.NoError(t, err)

	assert.Equal(t, "TestEvent", plan.EventType)
	assert.Equal(t, 1, plan.FromVersion)
	assert.Equal(t, 3, plan.ToVersion)
	assert.Len(t, plan.UpgradeSteps, 2)
	assert.True(t, plan.IsValid())

	// Already at current version
	plan, err = migrator.CreateMigrationPlan("TestEvent", 3)
	require.NoError(t, err)
	assert.Empty(t, plan.UpgradeSteps)
}

func TestMigrationStats(t *testing.T) {
	stats := NewMigrationStats()

	// Record some migrations
	stats.RecordMigration("TestEvent", 1, 2, 10.5, true)
	stats.RecordMigration("TestEvent", 1, 2, 5.5, true)
	stats.RecordMigration("TestEvent", 2, 3, 3.0, true)
	stats.RecordMigration("TestEvent", 1, 2, 0, false)

	eventStats, ok := stats.GetStats("TestEvent")
	require.True(t, ok)

	assert.Equal(t, "TestEvent", eventStats.EventType)
	assert.Equal(t, int64(3), eventStats.TotalMigrated)
	assert.Equal(t, int64(1), eventStats.TotalFailed)
	assert.True(t, eventStats.AverageDurationMs > 0)
	assert.Equal(t, int64(3), eventStats.MigrationsByVersion["v1->v2"])
	assert.Equal(t, int64(1), eventStats.MigrationsByVersion["v2->v3"])

	// Unknown event
	_, ok = stats.GetStats("UnknownEvent")
	assert.False(t, ok)
}

func TestMigrationResult_Duration(t *testing.T) {
	result := &MigrationResult{
		StartedAt:   time.Now().Add(-5 * time.Second),
		CompletedAt: time.Now(),
	}

	duration := result.Duration()
	assert.True(t, duration >= 4*time.Second)
	assert.True(t, duration <= 6*time.Second)
}

func TestCopyPayload(t *testing.T) {
	original := []byte(`{"key": "value", "nested": {"a": 1}}`)

	copied, err := CopyPayload(original)
	require.NoError(t, err)

	// Content should be equivalent (though order may differ)
	assert.Contains(t, string(copied), `"key":"value"`)
	assert.Contains(t, string(copied), `"nested"`)

	// Verify it's actually a copy by modifying original
	original[0] = 'X'
	assert.NotEqual(t, original[0], copied[0], "copied should not be affected by changes to original")
}

func TestBaseDomainEvent_SchemaVersion(t *testing.T) {
	// Test default version
	base := shared.NewBaseDomainEvent("Test", "Agg", uuid.New(), uuid.New())
	assert.Equal(t, 1, base.SchemaVersion())

	// Test explicit version
	base = shared.NewVersionedBaseDomainEvent("Test", "Agg", uuid.New(), uuid.New(), 3)
	assert.Equal(t, 3, base.SchemaVersion())

	// Test zero version falls back to 1
	base = shared.BaseDomainEvent{Version: 0}
	assert.Equal(t, 1, base.SchemaVersion())

	// Test negative version defaults to 1
	base = shared.NewVersionedBaseDomainEvent("Test", "Agg", uuid.New(), uuid.New(), -5)
	assert.Equal(t, 1, base.SchemaVersion())

	// Test zero version through NewVersionedBaseDomainEvent defaults to 1
	base = shared.NewVersionedBaseDomainEvent("Test", "Agg", uuid.New(), uuid.New(), 0)
	assert.Equal(t, 1, base.SchemaVersion())
}
