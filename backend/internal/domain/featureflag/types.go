package featureflag

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

// FlagType represents the type of feature flag
type FlagType string

const (
	// FlagTypeBoolean is a simple on/off flag
	FlagTypeBoolean FlagType = "boolean"
	// FlagTypePercentage is a percentage rollout flag (0-100%)
	FlagTypePercentage FlagType = "percentage"
	// FlagTypeVariant is a flag with multiple variants (A/B testing)
	FlagTypeVariant FlagType = "variant"
	// FlagTypeUserSegment is a flag targeting specific user segments
	FlagTypeUserSegment FlagType = "user_segment"
)

// AllFlagTypes returns all valid flag types
func AllFlagTypes() []FlagType {
	return []FlagType{
		FlagTypeBoolean,
		FlagTypePercentage,
		FlagTypeVariant,
		FlagTypeUserSegment,
	}
}

// IsValid checks if the flag type is valid
func (t FlagType) IsValid() bool {
	switch t {
	case FlagTypeBoolean, FlagTypePercentage, FlagTypeVariant, FlagTypeUserSegment:
		return true
	default:
		return false
	}
}

// String returns the string representation of the flag type
func (t FlagType) String() string {
	return string(t)
}

// Scan implements the sql.Scanner interface
func (t *FlagType) Scan(value any) error {
	if value == nil {
		return nil
	}
	s, ok := value.(string)
	if !ok {
		return fmt.Errorf("featureflag: cannot scan type %T into FlagType", value)
	}
	*t = FlagType(strings.ToLower(s))
	if !t.IsValid() {
		return fmt.Errorf("featureflag: invalid flag type: %s", s)
	}
	return nil
}

// Value implements the driver.Valuer interface
func (t FlagType) Value() (driver.Value, error) {
	return string(t), nil
}

// FlagStatus represents the status of a feature flag
type FlagStatus string

const (
	// FlagStatusEnabled means the flag is active and being evaluated
	FlagStatusEnabled FlagStatus = "enabled"
	// FlagStatusDisabled means the flag is inactive and returns default value
	FlagStatusDisabled FlagStatus = "disabled"
	// FlagStatusArchived means the flag is archived and should not be used
	FlagStatusArchived FlagStatus = "archived"
)

// AllFlagStatuses returns all valid flag statuses
func AllFlagStatuses() []FlagStatus {
	return []FlagStatus{
		FlagStatusEnabled,
		FlagStatusDisabled,
		FlagStatusArchived,
	}
}

// IsValid checks if the flag status is valid
func (s FlagStatus) IsValid() bool {
	switch s {
	case FlagStatusEnabled, FlagStatusDisabled, FlagStatusArchived:
		return true
	default:
		return false
	}
}

// String returns the string representation of the flag status
func (s FlagStatus) String() string {
	return string(s)
}

// Scan implements the sql.Scanner interface
func (s *FlagStatus) Scan(value any) error {
	if value == nil {
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("featureflag: cannot scan type %T into FlagStatus", value)
	}
	*s = FlagStatus(strings.ToLower(str))
	if !s.IsValid() {
		return fmt.Errorf("featureflag: invalid flag status: %s", str)
	}
	return nil
}

// Value implements the driver.Valuer interface
func (s FlagStatus) Value() (driver.Value, error) {
	return string(s), nil
}

// OverrideTargetType represents the type of override target
type OverrideTargetType string

const (
	// OverrideTargetTypeUser targets a specific user
	OverrideTargetTypeUser OverrideTargetType = "user"
	// OverrideTargetTypeTenant targets a specific tenant
	OverrideTargetTypeTenant OverrideTargetType = "tenant"
)

// AllOverrideTargetTypes returns all valid override target types
func AllOverrideTargetTypes() []OverrideTargetType {
	return []OverrideTargetType{
		OverrideTargetTypeUser,
		OverrideTargetTypeTenant,
	}
}

// IsValid checks if the override target type is valid
func (t OverrideTargetType) IsValid() bool {
	switch t {
	case OverrideTargetTypeUser, OverrideTargetTypeTenant:
		return true
	default:
		return false
	}
}

// String returns the string representation of the override target type
func (t OverrideTargetType) String() string {
	return string(t)
}

// Scan implements the sql.Scanner interface
func (t *OverrideTargetType) Scan(value any) error {
	if value == nil {
		return nil
	}
	s, ok := value.(string)
	if !ok {
		return fmt.Errorf("featureflag: cannot scan type %T into OverrideTargetType", value)
	}
	*t = OverrideTargetType(strings.ToLower(s))
	if !t.IsValid() {
		return fmt.Errorf("featureflag: invalid override target type: %s", s)
	}
	return nil
}

// Value implements the driver.Valuer interface
func (t OverrideTargetType) Value() (driver.Value, error) {
	return string(t), nil
}

// ConditionOperator represents operators for targeting conditions
type ConditionOperator string

const (
	// ConditionOperatorEquals checks for equality
	ConditionOperatorEquals ConditionOperator = "equals"
	// ConditionOperatorNotEquals checks for inequality
	ConditionOperatorNotEquals ConditionOperator = "not_equals"
	// ConditionOperatorIn checks if value is in a list
	ConditionOperatorIn ConditionOperator = "in"
	// ConditionOperatorNotIn checks if value is not in a list
	ConditionOperatorNotIn ConditionOperator = "not_in"
	// ConditionOperatorContains checks if value contains substring
	ConditionOperatorContains ConditionOperator = "contains"
	// ConditionOperatorGreaterThan checks if value is greater than
	ConditionOperatorGreaterThan ConditionOperator = "greater_than"
	// ConditionOperatorLessThan checks if value is less than
	ConditionOperatorLessThan ConditionOperator = "less_than"
)

// AllConditionOperators returns all valid condition operators
func AllConditionOperators() []ConditionOperator {
	return []ConditionOperator{
		ConditionOperatorEquals,
		ConditionOperatorNotEquals,
		ConditionOperatorIn,
		ConditionOperatorNotIn,
		ConditionOperatorContains,
		ConditionOperatorGreaterThan,
		ConditionOperatorLessThan,
	}
}

// IsValid checks if the condition operator is valid
func (o ConditionOperator) IsValid() bool {
	switch o {
	case ConditionOperatorEquals, ConditionOperatorNotEquals, ConditionOperatorIn,
		ConditionOperatorNotIn, ConditionOperatorContains, ConditionOperatorGreaterThan,
		ConditionOperatorLessThan:
		return true
	default:
		return false
	}
}

// String returns the string representation of the condition operator
func (o ConditionOperator) String() string {
	return string(o)
}

// Scan implements the sql.Scanner interface
func (o *ConditionOperator) Scan(value any) error {
	if value == nil {
		return nil
	}
	s, ok := value.(string)
	if !ok {
		return fmt.Errorf("featureflag: cannot scan type %T into ConditionOperator", value)
	}
	*o = ConditionOperator(strings.ToLower(s))
	if !o.IsValid() {
		return fmt.Errorf("featureflag: invalid condition operator: %s", s)
	}
	return nil
}

// Value implements the driver.Valuer interface
func (o ConditionOperator) Value() (driver.Value, error) {
	return string(o), nil
}

// AuditAction represents actions that can be audited
type AuditAction string

const (
	// AuditActionCreated indicates a flag was created
	AuditActionCreated AuditAction = "created"
	// AuditActionUpdated indicates a flag was updated
	AuditActionUpdated AuditAction = "updated"
	// AuditActionEnabled indicates a flag was enabled
	AuditActionEnabled AuditAction = "enabled"
	// AuditActionDisabled indicates a flag was disabled
	AuditActionDisabled AuditAction = "disabled"
	// AuditActionArchived indicates a flag was archived
	AuditActionArchived AuditAction = "archived"
	// AuditActionOverrideAdded indicates an override was added
	AuditActionOverrideAdded AuditAction = "override_added"
	// AuditActionOverrideRemoved indicates an override was removed
	AuditActionOverrideRemoved AuditAction = "override_removed"
)

// AllAuditActions returns all valid audit actions
func AllAuditActions() []AuditAction {
	return []AuditAction{
		AuditActionCreated,
		AuditActionUpdated,
		AuditActionEnabled,
		AuditActionDisabled,
		AuditActionArchived,
		AuditActionOverrideAdded,
		AuditActionOverrideRemoved,
	}
}

// IsValid checks if the audit action is valid
func (a AuditAction) IsValid() bool {
	switch a {
	case AuditActionCreated, AuditActionUpdated, AuditActionEnabled,
		AuditActionDisabled, AuditActionArchived, AuditActionOverrideAdded,
		AuditActionOverrideRemoved:
		return true
	default:
		return false
	}
}

// String returns the string representation of the audit action
func (a AuditAction) String() string {
	return string(a)
}

// Scan implements the sql.Scanner interface
func (a *AuditAction) Scan(value any) error {
	if value == nil {
		return nil
	}
	s, ok := value.(string)
	if !ok {
		return fmt.Errorf("featureflag: cannot scan type %T into AuditAction", value)
	}
	*a = AuditAction(strings.ToLower(s))
	if !a.IsValid() {
		return fmt.Errorf("featureflag: invalid audit action: %s", s)
	}
	return nil
}

// Value implements the driver.Valuer interface
func (a AuditAction) Value() (driver.Value, error) {
	return string(a), nil
}
