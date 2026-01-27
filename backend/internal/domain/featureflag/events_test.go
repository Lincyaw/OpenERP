package featureflag

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewFlagCreatedEvent(t *testing.T) {
	userID := uuid.New()
	flag, _ := NewFeatureFlag("test_flag", "Test Flag", FlagTypeBoolean, NewBooleanFlagValue(true), &userID)

	events := flag.GetDomainEvents()
	assert.Len(t, events, 1)

	event, ok := events[0].(*FlagCreatedEvent)
	assert.True(t, ok)
	assert.Equal(t, EventTypeFlagCreated, event.EventType())
	assert.Equal(t, AggregateTypeFeatureFlag, event.AggregateType())
	assert.Equal(t, flag.ID, event.FlagID)
	assert.Equal(t, "test_flag", event.Key)
	assert.Equal(t, "Test Flag", event.Name)
	assert.Equal(t, FlagTypeBoolean, event.Type)
	assert.Equal(t, FlagStatusDisabled, event.Status)
	assert.True(t, event.DefaultValue.Enabled)
	assert.Equal(t, &userID, event.CreatedBy)
}

func TestNewFlagUpdatedEvent(t *testing.T) {
	flag, _ := NewFeatureFlag("test_flag", "Test Flag", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
	flag.ClearDomainEvents()

	userID := uuid.New()
	_ = flag.Update("Updated Name", "Updated Description", &userID)

	events := flag.GetDomainEvents()
	assert.Len(t, events, 1)

	event, ok := events[0].(*FlagUpdatedEvent)
	assert.True(t, ok)
	assert.Equal(t, EventTypeFlagUpdated, event.EventType())
	assert.Equal(t, flag.ID, event.FlagID)
	assert.Equal(t, "test_flag", event.Key)
	assert.Equal(t, "Updated Name", event.Name)
	assert.Equal(t, "Updated Description", event.Description)
	assert.Equal(t, &userID, event.UpdatedBy)
	assert.Equal(t, flag.Version, event.Version)
}

func TestNewFlagUpdatedEventWithDetails(t *testing.T) {
	flag, _ := NewFeatureFlag("test_flag", "Test Flag", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
	flag.ClearDomainEvents()

	userID := uuid.New()
	_ = flag.SetDefault(NewBooleanFlagValue(true), &userID)

	events := flag.GetDomainEvents()
	assert.Len(t, events, 1)

	event, ok := events[0].(*FlagUpdatedEventWithDetails)
	assert.True(t, ok)
	assert.Equal(t, EventTypeFlagUpdated, event.EventType())
	assert.Equal(t, flag.ID, event.FlagID)
	assert.False(t, event.OldValue.Enabled)
	assert.True(t, event.NewValue.Enabled)
	assert.Equal(t, &userID, event.UpdatedBy)
}

func TestNewFlagEnabledEvent(t *testing.T) {
	flag, _ := NewFeatureFlag("test_flag", "Test Flag", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
	flag.ClearDomainEvents()

	userID := uuid.New()
	_ = flag.Enable(&userID)

	events := flag.GetDomainEvents()
	assert.Len(t, events, 1)

	event, ok := events[0].(*FlagEnabledEvent)
	assert.True(t, ok)
	assert.Equal(t, EventTypeFlagEnabled, event.EventType())
	assert.Equal(t, flag.ID, event.FlagID)
	assert.Equal(t, "test_flag", event.Key)
	assert.Equal(t, FlagStatusDisabled, event.OldStatus)
	assert.Equal(t, FlagStatusEnabled, event.NewStatus)
	assert.Equal(t, &userID, event.EnabledBy)
}

func TestNewFlagDisabledEvent(t *testing.T) {
	flag, _ := NewFeatureFlag("test_flag", "Test Flag", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
	_ = flag.Enable(nil)
	flag.ClearDomainEvents()

	userID := uuid.New()
	_ = flag.Disable(&userID)

	events := flag.GetDomainEvents()
	assert.Len(t, events, 1)

	event, ok := events[0].(*FlagDisabledEvent)
	assert.True(t, ok)
	assert.Equal(t, EventTypeFlagDisabled, event.EventType())
	assert.Equal(t, flag.ID, event.FlagID)
	assert.Equal(t, "test_flag", event.Key)
	assert.Equal(t, FlagStatusEnabled, event.OldStatus)
	assert.Equal(t, FlagStatusDisabled, event.NewStatus)
	assert.Equal(t, &userID, event.DisabledBy)
}

func TestNewFlagArchivedEvent(t *testing.T) {
	flag, _ := NewFeatureFlag("test_flag", "Test Flag", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
	flag.ClearDomainEvents()

	userID := uuid.New()
	_ = flag.Archive(&userID)

	events := flag.GetDomainEvents()
	assert.Len(t, events, 1)

	event, ok := events[0].(*FlagArchivedEvent)
	assert.True(t, ok)
	assert.Equal(t, EventTypeFlagArchived, event.EventType())
	assert.Equal(t, flag.ID, event.FlagID)
	assert.Equal(t, "test_flag", event.Key)
	assert.Equal(t, FlagStatusDisabled, event.OldStatus)
	assert.Equal(t, FlagStatusArchived, event.NewStatus)
	assert.Equal(t, &userID, event.ArchivedBy)
}

func TestNewOverrideCreatedEvent(t *testing.T) {
	userID := uuid.New()
	targetID := uuid.New()
	flagID := uuid.New()

	override, _ := NewFlagOverride("test_flag", OverrideTargetTypeUser, targetID, NewBooleanFlagValue(true), "Testing", nil, &userID)

	event := NewOverrideCreatedEvent(override, flagID)
	assert.Equal(t, EventTypeOverrideCreated, event.EventType())
	assert.Equal(t, AggregateTypeFeatureFlag, event.AggregateType())
	assert.Equal(t, flagID, event.AggregateID())
	assert.Equal(t, override.ID, event.OverrideID)
	assert.Equal(t, "test_flag", event.FlagKey)
	assert.Equal(t, OverrideTargetTypeUser, event.TargetType)
	assert.Equal(t, targetID, event.TargetID)
	assert.True(t, event.Value.Enabled)
	assert.Equal(t, "Testing", event.Reason)
	assert.Equal(t, &userID, event.CreatedBy)
}

func TestNewOverrideRemovedEvent(t *testing.T) {
	targetID := uuid.New()
	flagID := uuid.New()
	userID := uuid.New()

	override, _ := NewFlagOverride("test_flag", OverrideTargetTypeTenant, targetID, NewBooleanFlagValue(false), "", nil, nil)

	event := NewOverrideRemovedEvent(override, flagID, &userID)
	assert.Equal(t, EventTypeOverrideRemoved, event.EventType())
	assert.Equal(t, flagID, event.AggregateID())
	assert.Equal(t, override.ID, event.OverrideID)
	assert.Equal(t, "test_flag", event.FlagKey)
	assert.Equal(t, OverrideTargetTypeTenant, event.TargetType)
	assert.Equal(t, targetID, event.TargetID)
	assert.Equal(t, &userID, event.RemovedBy)
}

func TestNewOverrideUpdatedEvent(t *testing.T) {
	targetID := uuid.New()
	flagID := uuid.New()
	userID := uuid.New()

	override, _ := NewFlagOverride("test_flag", OverrideTargetTypeUser, targetID, NewBooleanFlagValue(true), "", nil, nil)
	oldValue := NewBooleanFlagValue(false)

	event := NewOverrideUpdatedEvent(override, flagID, oldValue, &userID)
	assert.Equal(t, EventTypeOverrideUpdated, event.EventType())
	assert.Equal(t, flagID, event.AggregateID())
	assert.Equal(t, override.ID, event.OverrideID)
	assert.Equal(t, "test_flag", event.FlagKey)
	assert.Equal(t, OverrideTargetTypeUser, event.TargetType)
	assert.Equal(t, targetID, event.TargetID)
	assert.False(t, event.OldValue.Enabled)
	assert.True(t, event.NewValue.Enabled)
	assert.Equal(t, &userID, event.UpdatedBy)
}
