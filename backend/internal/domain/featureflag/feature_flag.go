package featureflag

import (
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// keyRegex validates flag keys: must start with lowercase letter,
// followed by lowercase letters, numbers, underscores, hyphens, or dots
var keyRegex = regexp.MustCompile(`^[a-z][a-z0-9_.-]*$`)

// Aggregate type constant
const AggregateTypeFeatureFlag = "FeatureFlag"

// FeatureFlag is the aggregate root for feature flag management.
// It represents a configurable feature flag that can be used to control
// feature rollouts, A/B testing, and user targeting.
//
// DESIGN DECISION: Feature flags are GLOBAL (not tenant-scoped).
// Unlike other aggregates that use TenantAggregateRoot, FeatureFlag uses
// BaseAggregateRoot because feature flags control application behavior
// across the entire system. Tenant-specific behavior is achieved through
// FlagOverride entities, which allow per-tenant or per-user overrides.
type FeatureFlag struct {
	shared.BaseAggregateRoot
	Key          string          `json:"key"`
	Name         string          `json:"name"`
	Description  string          `json:"description,omitempty"`
	Type         FlagType        `json:"type"`
	Status       FlagStatus      `json:"status"`
	DefaultValue FlagValue       `json:"default_value"`
	Rules        []TargetingRule `json:"rules,omitempty"`
	Tags         []string        `json:"tags,omitempty"`
	CreatedBy    *uuid.UUID      `json:"created_by,omitempty"`
	UpdatedBy    *uuid.UUID      `json:"updated_by,omitempty"`
}

// NewFeatureFlag creates a new feature flag
func NewFeatureFlag(
	key string,
	name string,
	flagType FlagType,
	defaultValue FlagValue,
	createdBy *uuid.UUID,
) (*FeatureFlag, error) {
	if err := ValidateKey(key); err != nil {
		return nil, err
	}
	if err := validateName(name); err != nil {
		return nil, err
	}
	if !flagType.IsValid() {
		return nil, shared.NewDomainError("INVALID_FLAG_TYPE", "Invalid flag type")
	}

	flag := &FeatureFlag{
		BaseAggregateRoot: shared.NewBaseAggregateRoot(),
		Key:               strings.ToLower(key),
		Name:              name,
		Type:              flagType,
		Status:            FlagStatusDisabled,
		DefaultValue:      defaultValue,
		Rules:             make([]TargetingRule, 0),
		Tags:              make([]string, 0),
		CreatedBy:         createdBy,
		UpdatedBy:         createdBy,
	}

	flag.AddDomainEvent(NewFlagCreatedEvent(flag))

	return flag, nil
}

// NewBooleanFlag creates a new boolean feature flag
func NewBooleanFlag(key, name string, defaultEnabled bool, createdBy *uuid.UUID) (*FeatureFlag, error) {
	return NewFeatureFlag(key, name, FlagTypeBoolean, NewBooleanFlagValue(defaultEnabled), createdBy)
}

// NewPercentageFlag creates a new percentage rollout flag
func NewPercentageFlag(key, name string, createdBy *uuid.UUID) (*FeatureFlag, error) {
	return NewFeatureFlag(key, name, FlagTypePercentage, NewBooleanFlagValue(false), createdBy)
}

// NewVariantFlag creates a new variant (A/B testing) flag
func NewVariantFlag(key, name, defaultVariant string, createdBy *uuid.UUID) (*FeatureFlag, error) {
	return NewFeatureFlag(key, name, FlagTypeVariant, NewVariantFlagValue(defaultVariant), createdBy)
}

// GetKey returns the flag key
func (f *FeatureFlag) GetKey() string {
	return f.Key
}

// GetName returns the flag name
func (f *FeatureFlag) GetName() string {
	return f.Name
}

// GetDescription returns the flag description
func (f *FeatureFlag) GetDescription() string {
	return f.Description
}

// GetType returns the flag type
func (f *FeatureFlag) GetType() FlagType {
	return f.Type
}

// GetStatus returns the flag status
func (f *FeatureFlag) GetStatus() FlagStatus {
	return f.Status
}

// GetDefaultValue returns the default value
func (f *FeatureFlag) GetDefaultValue() FlagValue {
	return f.DefaultValue
}

// GetRules returns a copy of the targeting rules
func (f *FeatureFlag) GetRules() []TargetingRule {
	if f.Rules == nil {
		return []TargetingRule{}
	}
	result := make([]TargetingRule, len(f.Rules))
	copy(result, f.Rules)
	return result
}

// GetTags returns a copy of the tags
func (f *FeatureFlag) GetTags() []string {
	if f.Tags == nil {
		return []string{}
	}
	result := make([]string, len(f.Tags))
	copy(result, f.Tags)
	return result
}

// GetCreatedBy returns the creator user ID
func (f *FeatureFlag) GetCreatedBy() *uuid.UUID {
	return f.CreatedBy
}

// GetUpdatedBy returns the last updater user ID
func (f *FeatureFlag) GetUpdatedBy() *uuid.UUID {
	return f.UpdatedBy
}

// IsEnabled returns true if the flag is enabled
func (f *FeatureFlag) IsEnabled() bool {
	return f.Status == FlagStatusEnabled
}

// IsDisabled returns true if the flag is disabled
func (f *FeatureFlag) IsDisabled() bool {
	return f.Status == FlagStatusDisabled
}

// IsArchived returns true if the flag is archived
func (f *FeatureFlag) IsArchived() bool {
	return f.Status == FlagStatusArchived
}

// IsBooleanType returns true if the flag is a boolean type
func (f *FeatureFlag) IsBooleanType() bool {
	return f.Type == FlagTypeBoolean
}

// IsPercentageType returns true if the flag is a percentage type
func (f *FeatureFlag) IsPercentageType() bool {
	return f.Type == FlagTypePercentage
}

// IsVariantType returns true if the flag is a variant type
func (f *FeatureFlag) IsVariantType() bool {
	return f.Type == FlagTypeVariant
}

// IsUserSegmentType returns true if the flag is a user segment type
func (f *FeatureFlag) IsUserSegmentType() bool {
	return f.Type == FlagTypeUserSegment
}

// HasRules returns true if the flag has any targeting rules
func (f *FeatureFlag) HasRules() bool {
	return len(f.Rules) > 0
}

// HasTags returns true if the flag has any tags
func (f *FeatureFlag) HasTags() bool {
	return len(f.Tags) > 0
}

// HasTag returns true if the flag has the specified tag
func (f *FeatureFlag) HasTag(tag string) bool {
	for _, t := range f.Tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}

// Enable enables the feature flag
func (f *FeatureFlag) Enable(updatedBy *uuid.UUID) error {
	if f.Status == FlagStatusEnabled {
		return shared.NewDomainError("ALREADY_ENABLED", "Flag is already enabled")
	}
	if f.Status == FlagStatusArchived {
		return shared.NewDomainError("CANNOT_ENABLE", "Cannot enable an archived flag")
	}

	oldStatus := f.Status
	f.Status = FlagStatusEnabled
	f.UpdatedBy = updatedBy
	f.UpdatedAt = time.Now()
	f.IncrementVersion()

	f.AddDomainEvent(NewFlagEnabledEvent(f, oldStatus))

	return nil
}

// Disable disables the feature flag
func (f *FeatureFlag) Disable(updatedBy *uuid.UUID) error {
	if f.Status == FlagStatusDisabled {
		return shared.NewDomainError("ALREADY_DISABLED", "Flag is already disabled")
	}
	if f.Status == FlagStatusArchived {
		return shared.NewDomainError("CANNOT_DISABLE", "Cannot disable an archived flag")
	}

	oldStatus := f.Status
	f.Status = FlagStatusDisabled
	f.UpdatedBy = updatedBy
	f.UpdatedAt = time.Now()
	f.IncrementVersion()

	f.AddDomainEvent(NewFlagDisabledEvent(f, oldStatus))

	return nil
}

// Archive archives the feature flag
// An archived flag cannot be enabled or disabled
func (f *FeatureFlag) Archive(updatedBy *uuid.UUID) error {
	if f.Status == FlagStatusArchived {
		return shared.NewDomainError("ALREADY_ARCHIVED", "Flag is already archived")
	}

	oldStatus := f.Status
	f.Status = FlagStatusArchived
	f.UpdatedBy = updatedBy
	f.UpdatedAt = time.Now()
	f.IncrementVersion()

	f.AddDomainEvent(NewFlagArchivedEvent(f, oldStatus))

	return nil
}

// Update updates the flag's basic information
func (f *FeatureFlag) Update(name, description string, updatedBy *uuid.UUID) error {
	if f.Status == FlagStatusArchived {
		return shared.NewDomainError("CANNOT_UPDATE", "Cannot update an archived flag")
	}
	if err := validateName(name); err != nil {
		return err
	}

	f.Name = name
	f.Description = description
	f.UpdatedBy = updatedBy
	f.UpdatedAt = time.Now()
	f.IncrementVersion()

	f.AddDomainEvent(NewFlagUpdatedEvent(f))

	return nil
}

// SetDefault sets the default value of the flag
func (f *FeatureFlag) SetDefault(value FlagValue, updatedBy *uuid.UUID) error {
	if f.Status == FlagStatusArchived {
		return shared.NewDomainError("CANNOT_UPDATE", "Cannot update an archived flag")
	}

	oldValue := f.DefaultValue
	f.DefaultValue = value
	f.UpdatedBy = updatedBy
	f.UpdatedAt = time.Now()
	f.IncrementVersion()

	f.AddDomainEvent(NewFlagUpdatedEventWithDetails(f, oldValue))

	return nil
}

// AddRule adds a new targeting rule to the flag
func (f *FeatureFlag) AddRule(rule TargetingRule, updatedBy *uuid.UUID) error {
	if f.Status == FlagStatusArchived {
		return shared.NewDomainError("CANNOT_UPDATE", "Cannot update an archived flag")
	}
	if err := rule.Validate(); err != nil {
		return err
	}

	// Check for duplicate rule ID
	for _, r := range f.Rules {
		if r.RuleID == rule.RuleID {
			return shared.NewDomainError("DUPLICATE_RULE_ID", "Rule ID already exists")
		}
	}

	f.Rules = append(f.Rules, rule)
	f.sortRulesByPriority()
	f.UpdatedBy = updatedBy
	f.UpdatedAt = time.Now()
	f.IncrementVersion()

	f.AddDomainEvent(NewFlagUpdatedEvent(f))

	return nil
}

// RemoveRule removes a targeting rule by its ID
func (f *FeatureFlag) RemoveRule(ruleID string, updatedBy *uuid.UUID) error {
	if f.Status == FlagStatusArchived {
		return shared.NewDomainError("CANNOT_UPDATE", "Cannot update an archived flag")
	}

	found := false
	newRules := make([]TargetingRule, 0, len(f.Rules))
	for _, r := range f.Rules {
		if r.RuleID == ruleID {
			found = true
			continue
		}
		newRules = append(newRules, r)
	}

	if !found {
		return shared.NewDomainError("RULE_NOT_FOUND", "Rule not found")
	}

	f.Rules = newRules
	f.UpdatedBy = updatedBy
	f.UpdatedAt = time.Now()
	f.IncrementVersion()

	f.AddDomainEvent(NewFlagUpdatedEvent(f))

	return nil
}

// UpdateRule updates an existing targeting rule
func (f *FeatureFlag) UpdateRule(rule TargetingRule, updatedBy *uuid.UUID) error {
	if f.Status == FlagStatusArchived {
		return shared.NewDomainError("CANNOT_UPDATE", "Cannot update an archived flag")
	}
	if err := rule.Validate(); err != nil {
		return err
	}

	found := false
	for i, r := range f.Rules {
		if r.RuleID == rule.RuleID {
			f.Rules[i] = rule
			found = true
			break
		}
	}

	if !found {
		return shared.NewDomainError("RULE_NOT_FOUND", "Rule not found")
	}

	f.sortRulesByPriority()
	f.UpdatedBy = updatedBy
	f.UpdatedAt = time.Now()
	f.IncrementVersion()

	f.AddDomainEvent(NewFlagUpdatedEvent(f))

	return nil
}

// ClearRules removes all targeting rules
func (f *FeatureFlag) ClearRules(updatedBy *uuid.UUID) error {
	if f.Status == FlagStatusArchived {
		return shared.NewDomainError("CANNOT_UPDATE", "Cannot update an archived flag")
	}

	f.Rules = make([]TargetingRule, 0)
	f.UpdatedBy = updatedBy
	f.UpdatedAt = time.Now()
	f.IncrementVersion()

	f.AddDomainEvent(NewFlagUpdatedEvent(f))

	return nil
}

// SetTags sets the tags for the flag
func (f *FeatureFlag) SetTags(tags []string, updatedBy *uuid.UUID) error {
	if f.Status == FlagStatusArchived {
		return shared.NewDomainError("CANNOT_UPDATE", "Cannot update an archived flag")
	}

	// Normalize tags: lowercase and deduplicate
	tagSet := make(map[string]struct{})
	normalizedTags := make([]string, 0, len(tags))
	for _, tag := range tags {
		normalized := strings.ToLower(strings.TrimSpace(tag))
		if normalized == "" {
			continue
		}
		if _, exists := tagSet[normalized]; !exists {
			tagSet[normalized] = struct{}{}
			normalizedTags = append(normalizedTags, normalized)
		}
	}

	f.Tags = normalizedTags
	f.UpdatedBy = updatedBy
	f.UpdatedAt = time.Now()
	f.IncrementVersion()

	return nil
}

// AddTag adds a tag to the flag
func (f *FeatureFlag) AddTag(tag string, updatedBy *uuid.UUID) error {
	if f.Status == FlagStatusArchived {
		return shared.NewDomainError("CANNOT_UPDATE", "Cannot update an archived flag")
	}

	normalized := strings.ToLower(strings.TrimSpace(tag))
	if normalized == "" {
		return shared.NewDomainError("INVALID_TAG", "Tag cannot be empty")
	}

	if f.HasTag(normalized) {
		return nil // Tag already exists, no-op
	}

	f.Tags = append(f.Tags, normalized)
	f.UpdatedBy = updatedBy
	f.UpdatedAt = time.Now()
	f.IncrementVersion()

	return nil
}

// RemoveTag removes a tag from the flag
func (f *FeatureFlag) RemoveTag(tag string, updatedBy *uuid.UUID) error {
	if f.Status == FlagStatusArchived {
		return shared.NewDomainError("CANNOT_UPDATE", "Cannot update an archived flag")
	}

	normalized := strings.ToLower(strings.TrimSpace(tag))
	newTags := make([]string, 0, len(f.Tags))
	for _, t := range f.Tags {
		if t != normalized {
			newTags = append(newTags, t)
		}
	}

	f.Tags = newTags
	f.UpdatedBy = updatedBy
	f.UpdatedAt = time.Now()
	f.IncrementVersion()

	return nil
}

// GetRuleByID returns a rule by its ID
func (f *FeatureFlag) GetRuleByID(ruleID string) *TargetingRule {
	for i := range f.Rules {
		if f.Rules[i].RuleID == ruleID {
			return &f.Rules[i]
		}
	}
	return nil
}

// sortRulesByPriority sorts rules by priority (lower number = higher priority)
func (f *FeatureFlag) sortRulesByPriority() {
	sort.Slice(f.Rules, func(i, j int) bool {
		return f.Rules[i].Priority < f.Rules[j].Priority
	})
}

// ValidateKey validates a feature flag key
func ValidateKey(key string) error {
	if key == "" {
		return shared.NewDomainError("INVALID_KEY", "Flag key cannot be empty")
	}
	if len(key) > 100 {
		return shared.NewDomainError("INVALID_KEY", "Flag key cannot exceed 100 characters")
	}
	if !keyRegex.MatchString(strings.ToLower(key)) {
		return shared.NewDomainError("INVALID_KEY", "Flag key must start with a letter and contain only lowercase letters, numbers, underscores, hyphens, and dots")
	}
	return nil
}

// ValidateRules validates all targeting rules
func (f *FeatureFlag) ValidateRules() error {
	ruleIDs := make(map[string]struct{})
	for _, rule := range f.Rules {
		if err := rule.Validate(); err != nil {
			return err
		}
		if _, exists := ruleIDs[rule.RuleID]; exists {
			return shared.NewDomainError("DUPLICATE_RULE_ID", "Duplicate rule ID: "+rule.RuleID)
		}
		ruleIDs[rule.RuleID] = struct{}{}
	}
	return nil
}

// validateName validates the flag name
func validateName(name string) error {
	if name == "" {
		return shared.NewDomainError("INVALID_NAME", "Flag name cannot be empty")
	}
	if len(name) > 200 {
		return shared.NewDomainError("INVALID_NAME", "Flag name cannot exceed 200 characters")
	}
	return nil
}
