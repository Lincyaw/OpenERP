package event

/*
Event Versioning Strategy Documentation
=======================================

This document describes the event versioning strategy for maintaining backward
compatibility when evolving domain event schemas.

Overview
--------

When domain events are stored (in outbox, event store, or message queues) and
later replayed or processed, their schema may have changed since they were
created. The versioning system ensures that:

1. Old events can still be deserialized and processed
2. Events are automatically upgraded to the latest schema version
3. The upgrade process is transparent to event handlers

Key Components
--------------

1. BaseDomainEvent.Version
   - Every event has a schema_version field (defaults to 1)
   - The version is serialized with the event payload
   - Events without a version field are treated as version 1

2. VersionedEvent Interface
   - Events can implement this interface to expose their version
   - BaseDomainEvent already implements SchemaVersion()

3. EventUpgrader Interface
   - Transforms event payload from one version to the next
   - Must be sequential (v1->v2, v2->v3, etc.)

4. VersionRegistry
   - Manages registered event types and their versions
   - Stores the upgrader chain for each event type

5. VersionedSerializer
   - Drop-in replacement for EventSerializer
   - Automatically upgrades events during deserialization

Usage Examples
--------------

Example 1: Registering a Simple Event (Version 1 Only)
```go
serializer := NewVersionedSerializer(logger)
serializer.Register("OrderCreated", &OrderCreatedEvent{})
```

Example 2: Evolving an Event Schema

Original v1 event:
```go
type OrderCreatedV1 struct {
    shared.BaseDomainEvent
    OrderID    uuid.UUID `json:"order_id"`
    CustomerID uuid.UUID `json:"customer_id"`
}
```

New v2 event (added customer_name field):
```go
type OrderCreatedV2 struct {
    shared.BaseDomainEvent
    OrderID      uuid.UUID `json:"order_id"`
    CustomerID   uuid.UUID `json:"customer_id"`
    CustomerName string    `json:"customer_name"` // New in v2
}
```

Creating the upgrader:
```go
v1ToV2 := NewBaseEventUpgrader(1, 2, func(data map[string]any) (map[string]any, error) {
    // Add default customer_name for v1 events
    data["customer_name"] = "Unknown"
    return data, nil
})
```

Registering the versioned event:
```go
err := serializer.RegisterVersioned(
    "OrderCreated",
    2, // current version
    map[int]shared.DomainEvent{
        1: &OrderCreatedV1{},
        2: &OrderCreatedV2{},
    },
    v1ToV2,
)
```

Example 3: Using CommonUpgraders for Simple Changes
```go
upgraders := CommonUpgraders{}

// Add a new field with default value
addField := upgraders.AddField(1, "customer_name", "Unknown")

// Rename a field
renameField := upgraders.RenameField(2, "customer_id", "client_id")

// Remove a field
removeField := upgraders.RemoveField(3, "deprecated_field")

// Transform a field value
transformField := upgraders.TransformField(4, "amount", func(v any) any {
    // Convert from cents to dollars
    if cents, ok := v.(float64); ok {
        return cents / 100
    }
    return v
})
```

Example 4: Batch Migration
```go
migrator := NewEventMigrator(serializer, logger)

// Analyze current state
analysis, _ := migrator.AnalyzePayloads("OrderCreated", payloads)
fmt.Printf("Need migration: %d events\n", analysis.NeedsMigration)

// Perform migration
result, _ := migrator.MigratePayloads(ctx, "OrderCreated", payloads)
fmt.Printf("Upgraded: %d, Failed: %d\n", result.Upgraded, result.Failed)
```

Best Practices
--------------

1. Version Increment Rules
   - Increment version when adding/removing/renaming fields
   - Increment version when changing field types
   - Keep changes backward-compatible when possible

2. Upgrader Design
   - Each upgrader handles exactly one version transition
   - Upgraders must be deterministic (same input = same output)
   - Handle missing fields gracefully (use defaults)

3. Testing
   - Write tests for each upgrader in isolation
   - Test full upgrade chain from v1 to latest
   - Test with real historical payloads if available

4. Deployment
   - Deploy upgraders before producing events with new version
   - Run batch migration for existing events in storage
   - Monitor migration failures and address them

5. Event Type Naming
   - Use consistent naming (PascalCase, domain-prefixed)
   - Never change event type names (breaks routing)
   - If type name must change, treat as new event type

Schema Change Guidelines
------------------------

Safe Changes (backward compatible):
- Adding optional fields with defaults
- Adding new event types
- Relaxing field constraints

Breaking Changes (require upgrader):
- Renaming fields
- Changing field types
- Removing fields
- Adding required fields

File Organization
-----------------

```
backend/internal/
├── domain/shared/
│   └── event.go          # DomainEvent, VersionedEvent interfaces
└── infrastructure/event/
    ├── versioning.go     # VersionRegistry, EventUpgrader
    ├── versioned_serializer.go
    ├── migration.go      # Migration utilities
    └── event_registry.go # RegisterAllEvents function
```

Migration Workflow
------------------

When evolving an event schema:

1. Create new event struct (vN+1)
2. Create upgrader from vN to vN+1
3. Update RegisterAllEvents to use versioned registration
4. Run batch migration on stored events
5. Deploy and monitor

Example workflow:
```go
// In domain/trade/sales_order_events.go
type SalesOrderCreatedEventV2 struct {
    shared.BaseDomainEvent
    // ... v2 fields
}

// In infrastructure/event/event_registry.go
func RegisterAllEvents(serializer *VersionedSerializer) {
    serializer.RegisterVersioned(
        "SalesOrderCreated",
        2,
        map[int]shared.DomainEvent{
            1: &trade.SalesOrderCreatedEvent{},   // v1
            2: &trade.SalesOrderCreatedEventV2{}, // v2
        },
        trade.SalesOrderCreatedV1ToV2Upgrader(),
    )
}
```

Error Handling
--------------

The system handles errors at different levels:

1. Unknown event type: Error returned, event not processed
2. Missing upgrader: Error returned with specific version gap
3. Upgrade failure: Error returned, original payload preserved
4. JSON parse failure: Falls back to version 1

Monitoring
----------

Use MigrationStats to track:
- Total events migrated per type
- Failed migrations per type
- Average migration duration
- Version distribution
*/
