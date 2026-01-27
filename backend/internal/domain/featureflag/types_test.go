package featureflag

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlagType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		flagType FlagType
		want     bool
	}{
		{"boolean is valid", FlagTypeBoolean, true},
		{"percentage is valid", FlagTypePercentage, true},
		{"variant is valid", FlagTypeVariant, true},
		{"user_segment is valid", FlagTypeUserSegment, true},
		{"invalid type", FlagType("invalid"), false},
		{"empty type", FlagType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.flagType.IsValid())
		})
	}
}

func TestFlagType_String(t *testing.T) {
	assert.Equal(t, "boolean", FlagTypeBoolean.String())
	assert.Equal(t, "percentage", FlagTypePercentage.String())
	assert.Equal(t, "variant", FlagTypeVariant.String())
	assert.Equal(t, "user_segment", FlagTypeUserSegment.String())
}

func TestFlagType_Scan(t *testing.T) {
	var ft FlagType

	// Test valid scan
	err := ft.Scan("boolean")
	assert.NoError(t, err)
	assert.Equal(t, FlagTypeBoolean, ft)

	// Test nil scan
	err = ft.Scan(nil)
	assert.NoError(t, err)

	// Test invalid type
	err = ft.Scan(123)
	assert.Error(t, err)

	// Test invalid value
	err = ft.Scan("invalid")
	assert.Error(t, err)
}

func TestFlagType_Value(t *testing.T) {
	val, err := FlagTypeBoolean.Value()
	assert.NoError(t, err)
	assert.Equal(t, "boolean", val)
}

func TestAllFlagTypes(t *testing.T) {
	types := AllFlagTypes()
	assert.Len(t, types, 4)
	assert.Contains(t, types, FlagTypeBoolean)
	assert.Contains(t, types, FlagTypePercentage)
	assert.Contains(t, types, FlagTypeVariant)
	assert.Contains(t, types, FlagTypeUserSegment)
}

func TestFlagStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status FlagStatus
		want   bool
	}{
		{"enabled is valid", FlagStatusEnabled, true},
		{"disabled is valid", FlagStatusDisabled, true},
		{"archived is valid", FlagStatusArchived, true},
		{"invalid status", FlagStatus("invalid"), false},
		{"empty status", FlagStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.IsValid())
		})
	}
}

func TestFlagStatus_String(t *testing.T) {
	assert.Equal(t, "enabled", FlagStatusEnabled.String())
	assert.Equal(t, "disabled", FlagStatusDisabled.String())
	assert.Equal(t, "archived", FlagStatusArchived.String())
}

func TestFlagStatus_Scan(t *testing.T) {
	var fs FlagStatus

	// Test valid scan
	err := fs.Scan("enabled")
	assert.NoError(t, err)
	assert.Equal(t, FlagStatusEnabled, fs)

	// Test nil scan
	err = fs.Scan(nil)
	assert.NoError(t, err)

	// Test invalid type
	err = fs.Scan(123)
	assert.Error(t, err)

	// Test invalid value
	err = fs.Scan("invalid")
	assert.Error(t, err)
}

func TestFlagStatus_Value(t *testing.T) {
	val, err := FlagStatusEnabled.Value()
	assert.NoError(t, err)
	assert.Equal(t, "enabled", val)
}

func TestAllFlagStatuses(t *testing.T) {
	statuses := AllFlagStatuses()
	assert.Len(t, statuses, 3)
	assert.Contains(t, statuses, FlagStatusEnabled)
	assert.Contains(t, statuses, FlagStatusDisabled)
	assert.Contains(t, statuses, FlagStatusArchived)
}

func TestOverrideTargetType_IsValid(t *testing.T) {
	tests := []struct {
		name       string
		targetType OverrideTargetType
		want       bool
	}{
		{"user is valid", OverrideTargetTypeUser, true},
		{"tenant is valid", OverrideTargetTypeTenant, true},
		{"invalid type", OverrideTargetType("invalid"), false},
		{"empty type", OverrideTargetType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.targetType.IsValid())
		})
	}
}

func TestOverrideTargetType_String(t *testing.T) {
	assert.Equal(t, "user", OverrideTargetTypeUser.String())
	assert.Equal(t, "tenant", OverrideTargetTypeTenant.String())
}

func TestOverrideTargetType_Scan(t *testing.T) {
	var ot OverrideTargetType

	// Test valid scan
	err := ot.Scan("user")
	assert.NoError(t, err)
	assert.Equal(t, OverrideTargetTypeUser, ot)

	// Test nil scan
	err = ot.Scan(nil)
	assert.NoError(t, err)

	// Test invalid type
	err = ot.Scan(123)
	assert.Error(t, err)

	// Test invalid value
	err = ot.Scan("invalid")
	assert.Error(t, err)
}

func TestOverrideTargetType_Value(t *testing.T) {
	val, err := OverrideTargetTypeUser.Value()
	assert.NoError(t, err)
	assert.Equal(t, "user", val)
}

func TestAllOverrideTargetTypes(t *testing.T) {
	types := AllOverrideTargetTypes()
	assert.Len(t, types, 2)
	assert.Contains(t, types, OverrideTargetTypeUser)
	assert.Contains(t, types, OverrideTargetTypeTenant)
}

func TestConditionOperator_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		operator ConditionOperator
		want     bool
	}{
		{"equals is valid", ConditionOperatorEquals, true},
		{"not_equals is valid", ConditionOperatorNotEquals, true},
		{"in is valid", ConditionOperatorIn, true},
		{"not_in is valid", ConditionOperatorNotIn, true},
		{"contains is valid", ConditionOperatorContains, true},
		{"greater_than is valid", ConditionOperatorGreaterThan, true},
		{"less_than is valid", ConditionOperatorLessThan, true},
		{"invalid operator", ConditionOperator("invalid"), false},
		{"empty operator", ConditionOperator(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.operator.IsValid())
		})
	}
}

func TestConditionOperator_String(t *testing.T) {
	assert.Equal(t, "equals", ConditionOperatorEquals.String())
	assert.Equal(t, "not_equals", ConditionOperatorNotEquals.String())
	assert.Equal(t, "greater_than", ConditionOperatorGreaterThan.String())
}

func TestConditionOperator_Scan(t *testing.T) {
	var co ConditionOperator

	// Test valid scan
	err := co.Scan("equals")
	assert.NoError(t, err)
	assert.Equal(t, ConditionOperatorEquals, co)

	// Test nil scan
	err = co.Scan(nil)
	assert.NoError(t, err)

	// Test invalid type
	err = co.Scan(123)
	assert.Error(t, err)

	// Test invalid value
	err = co.Scan("invalid")
	assert.Error(t, err)
}

func TestConditionOperator_Value(t *testing.T) {
	val, err := ConditionOperatorEquals.Value()
	assert.NoError(t, err)
	assert.Equal(t, "equals", val)
}

func TestAllConditionOperators(t *testing.T) {
	operators := AllConditionOperators()
	assert.Len(t, operators, 7)
	assert.Contains(t, operators, ConditionOperatorEquals)
	assert.Contains(t, operators, ConditionOperatorNotEquals)
	assert.Contains(t, operators, ConditionOperatorIn)
	assert.Contains(t, operators, ConditionOperatorNotIn)
	assert.Contains(t, operators, ConditionOperatorContains)
	assert.Contains(t, operators, ConditionOperatorGreaterThan)
	assert.Contains(t, operators, ConditionOperatorLessThan)
}

func TestAuditAction_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		action AuditAction
		want   bool
	}{
		{"created is valid", AuditActionCreated, true},
		{"updated is valid", AuditActionUpdated, true},
		{"enabled is valid", AuditActionEnabled, true},
		{"disabled is valid", AuditActionDisabled, true},
		{"archived is valid", AuditActionArchived, true},
		{"override_added is valid", AuditActionOverrideAdded, true},
		{"override_removed is valid", AuditActionOverrideRemoved, true},
		{"invalid action", AuditAction("invalid"), false},
		{"empty action", AuditAction(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.action.IsValid())
		})
	}
}

func TestAuditAction_String(t *testing.T) {
	assert.Equal(t, "created", AuditActionCreated.String())
	assert.Equal(t, "updated", AuditActionUpdated.String())
	assert.Equal(t, "override_added", AuditActionOverrideAdded.String())
}

func TestAllAuditActions(t *testing.T) {
	actions := AllAuditActions()
	assert.Len(t, actions, 7)
	assert.Contains(t, actions, AuditActionCreated)
	assert.Contains(t, actions, AuditActionUpdated)
	assert.Contains(t, actions, AuditActionEnabled)
	assert.Contains(t, actions, AuditActionDisabled)
	assert.Contains(t, actions, AuditActionArchived)
	assert.Contains(t, actions, AuditActionOverrideAdded)
	assert.Contains(t, actions, AuditActionOverrideRemoved)
}

func TestAuditAction_Scan(t *testing.T) {
	var aa AuditAction

	// Test valid scan
	err := aa.Scan("created")
	assert.NoError(t, err)
	assert.Equal(t, AuditActionCreated, aa)

	// Test nil scan
	err = aa.Scan(nil)
	assert.NoError(t, err)

	// Test invalid type
	err = aa.Scan(123)
	assert.Error(t, err)

	// Test invalid value
	err = aa.Scan("invalid")
	assert.Error(t, err)
}

func TestAuditAction_Value(t *testing.T) {
	val, err := AuditActionCreated.Value()
	assert.NoError(t, err)
	assert.Equal(t, "created", val)
}
