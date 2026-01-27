package featureflag

import (
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// Event type constants
const (
	EventTypeFlagCreated     = "FlagCreated"
	EventTypeFlagUpdated     = "FlagUpdated"
	EventTypeFlagEnabled     = "FlagEnabled"
	EventTypeFlagDisabled    = "FlagDisabled"
	EventTypeFlagArchived    = "FlagArchived"
	EventTypeOverrideCreated = "OverrideCreated"
	EventTypeOverrideRemoved = "OverrideRemoved"
	EventTypeOverrideUpdated = "OverrideUpdated"
)

// FlagCreatedEvent is published when a new feature flag is created
type FlagCreatedEvent struct {
	shared.BaseDomainEvent
	FlagID       uuid.UUID  `json:"flag_id"`
	Key          string     `json:"key"`
	Name         string     `json:"name"`
	Type         FlagType   `json:"type"`
	Status       FlagStatus `json:"status"`
	DefaultValue FlagValue  `json:"default_value"`
	CreatedBy    *uuid.UUID `json:"created_by,omitempty"`
}

// NewFlagCreatedEvent creates a new FlagCreatedEvent
func NewFlagCreatedEvent(flag *FeatureFlag) *FlagCreatedEvent {
	return &FlagCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			EventTypeFlagCreated,
			AggregateTypeFeatureFlag,
			flag.ID,
			uuid.Nil, // Feature flags are global, no tenant ID
		),
		FlagID:       flag.ID,
		Key:          flag.Key,
		Name:         flag.Name,
		Type:         flag.Type,
		Status:       flag.Status,
		DefaultValue: flag.DefaultValue,
		CreatedBy:    flag.CreatedBy,
	}
}

// FlagUpdatedEvent is published when a feature flag is updated
type FlagUpdatedEvent struct {
	shared.BaseDomainEvent
	FlagID      uuid.UUID  `json:"flag_id"`
	Key         string     `json:"key"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	UpdatedBy   *uuid.UUID `json:"updated_by,omitempty"`
	Version     int        `json:"version"`
}

// NewFlagUpdatedEvent creates a new FlagUpdatedEvent
func NewFlagUpdatedEvent(flag *FeatureFlag) *FlagUpdatedEvent {
	return &FlagUpdatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			EventTypeFlagUpdated,
			AggregateTypeFeatureFlag,
			flag.ID,
			uuid.Nil,
		),
		FlagID:      flag.ID,
		Key:         flag.Key,
		Name:        flag.Name,
		Description: flag.Description,
		UpdatedBy:   flag.UpdatedBy,
		Version:     flag.Version,
	}
}

// FlagUpdatedEventWithDetails includes more details about the update
type FlagUpdatedEventWithDetails struct {
	shared.BaseDomainEvent
	FlagID    uuid.UUID  `json:"flag_id"`
	Key       string     `json:"key"`
	Name      string     `json:"name"`
	OldValue  FlagValue  `json:"old_value"`
	NewValue  FlagValue  `json:"new_value"`
	UpdatedBy *uuid.UUID `json:"updated_by,omitempty"`
	Version   int        `json:"version"`
}

// NewFlagUpdatedEventWithDetails creates a new FlagUpdatedEventWithDetails
func NewFlagUpdatedEventWithDetails(flag *FeatureFlag, oldValue FlagValue) *FlagUpdatedEventWithDetails {
	return &FlagUpdatedEventWithDetails{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			EventTypeFlagUpdated,
			AggregateTypeFeatureFlag,
			flag.ID,
			uuid.Nil,
		),
		FlagID:    flag.ID,
		Key:       flag.Key,
		Name:      flag.Name,
		OldValue:  oldValue,
		NewValue:  flag.DefaultValue,
		UpdatedBy: flag.UpdatedBy,
		Version:   flag.Version,
	}
}

// FlagEnabledEvent is published when a feature flag is enabled
type FlagEnabledEvent struct {
	shared.BaseDomainEvent
	FlagID    uuid.UUID  `json:"flag_id"`
	Key       string     `json:"key"`
	OldStatus FlagStatus `json:"old_status"`
	NewStatus FlagStatus `json:"new_status"`
	EnabledBy *uuid.UUID `json:"enabled_by,omitempty"`
}

// NewFlagEnabledEvent creates a new FlagEnabledEvent
func NewFlagEnabledEvent(flag *FeatureFlag, oldStatus FlagStatus) *FlagEnabledEvent {
	return &FlagEnabledEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			EventTypeFlagEnabled,
			AggregateTypeFeatureFlag,
			flag.ID,
			uuid.Nil,
		),
		FlagID:    flag.ID,
		Key:       flag.Key,
		OldStatus: oldStatus,
		NewStatus: FlagStatusEnabled,
		EnabledBy: flag.UpdatedBy,
	}
}

// FlagDisabledEvent is published when a feature flag is disabled
type FlagDisabledEvent struct {
	shared.BaseDomainEvent
	FlagID     uuid.UUID  `json:"flag_id"`
	Key        string     `json:"key"`
	OldStatus  FlagStatus `json:"old_status"`
	NewStatus  FlagStatus `json:"new_status"`
	DisabledBy *uuid.UUID `json:"disabled_by,omitempty"`
}

// NewFlagDisabledEvent creates a new FlagDisabledEvent
func NewFlagDisabledEvent(flag *FeatureFlag, oldStatus FlagStatus) *FlagDisabledEvent {
	return &FlagDisabledEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			EventTypeFlagDisabled,
			AggregateTypeFeatureFlag,
			flag.ID,
			uuid.Nil,
		),
		FlagID:     flag.ID,
		Key:        flag.Key,
		OldStatus:  oldStatus,
		NewStatus:  FlagStatusDisabled,
		DisabledBy: flag.UpdatedBy,
	}
}

// FlagArchivedEvent is published when a feature flag is archived
type FlagArchivedEvent struct {
	shared.BaseDomainEvent
	FlagID     uuid.UUID  `json:"flag_id"`
	Key        string     `json:"key"`
	OldStatus  FlagStatus `json:"old_status"`
	NewStatus  FlagStatus `json:"new_status"`
	ArchivedBy *uuid.UUID `json:"archived_by,omitempty"`
}

// NewFlagArchivedEvent creates a new FlagArchivedEvent
func NewFlagArchivedEvent(flag *FeatureFlag, oldStatus FlagStatus) *FlagArchivedEvent {
	return &FlagArchivedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			EventTypeFlagArchived,
			AggregateTypeFeatureFlag,
			flag.ID,
			uuid.Nil,
		),
		FlagID:     flag.ID,
		Key:        flag.Key,
		OldStatus:  oldStatus,
		NewStatus:  FlagStatusArchived,
		ArchivedBy: flag.UpdatedBy,
	}
}

// OverrideCreatedEvent is published when a flag override is created
type OverrideCreatedEvent struct {
	shared.BaseDomainEvent
	OverrideID uuid.UUID          `json:"override_id"`
	FlagKey    string             `json:"flag_key"`
	TargetType OverrideTargetType `json:"target_type"`
	TargetID   uuid.UUID          `json:"target_id"`
	Value      FlagValue          `json:"value"`
	Reason     string             `json:"reason,omitempty"`
	CreatedBy  *uuid.UUID         `json:"created_by,omitempty"`
}

// NewOverrideCreatedEvent creates a new OverrideCreatedEvent
func NewOverrideCreatedEvent(override *FlagOverride, flagID uuid.UUID) *OverrideCreatedEvent {
	return &OverrideCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			EventTypeOverrideCreated,
			AggregateTypeFeatureFlag,
			flagID,
			uuid.Nil,
		),
		OverrideID: override.ID,
		FlagKey:    override.FlagKey,
		TargetType: override.TargetType,
		TargetID:   override.TargetID,
		Value:      override.Value,
		Reason:     override.Reason,
		CreatedBy:  override.CreatedBy,
	}
}

// OverrideRemovedEvent is published when a flag override is removed
type OverrideRemovedEvent struct {
	shared.BaseDomainEvent
	OverrideID uuid.UUID          `json:"override_id"`
	FlagKey    string             `json:"flag_key"`
	TargetType OverrideTargetType `json:"target_type"`
	TargetID   uuid.UUID          `json:"target_id"`
	RemovedBy  *uuid.UUID         `json:"removed_by,omitempty"`
}

// NewOverrideRemovedEvent creates a new OverrideRemovedEvent
func NewOverrideRemovedEvent(override *FlagOverride, flagID uuid.UUID, removedBy *uuid.UUID) *OverrideRemovedEvent {
	return &OverrideRemovedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			EventTypeOverrideRemoved,
			AggregateTypeFeatureFlag,
			flagID,
			uuid.Nil,
		),
		OverrideID: override.ID,
		FlagKey:    override.FlagKey,
		TargetType: override.TargetType,
		TargetID:   override.TargetID,
		RemovedBy:  removedBy,
	}
}

// OverrideUpdatedEvent is published when a flag override is updated
type OverrideUpdatedEvent struct {
	shared.BaseDomainEvent
	OverrideID uuid.UUID          `json:"override_id"`
	FlagKey    string             `json:"flag_key"`
	TargetType OverrideTargetType `json:"target_type"`
	TargetID   uuid.UUID          `json:"target_id"`
	OldValue   FlagValue          `json:"old_value"`
	NewValue   FlagValue          `json:"new_value"`
	UpdatedBy  *uuid.UUID         `json:"updated_by,omitempty"`
}

// NewOverrideUpdatedEvent creates a new OverrideUpdatedEvent
func NewOverrideUpdatedEvent(override *FlagOverride, flagID uuid.UUID, oldValue FlagValue, updatedBy *uuid.UUID) *OverrideUpdatedEvent {
	return &OverrideUpdatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(
			EventTypeOverrideUpdated,
			AggregateTypeFeatureFlag,
			flagID,
			uuid.Nil,
		),
		OverrideID: override.ID,
		FlagKey:    override.FlagKey,
		TargetType: override.TargetType,
		TargetID:   override.TargetID,
		OldValue:   oldValue,
		NewValue:   override.Value,
		UpdatedBy:  updatedBy,
	}
}
