package featureflag

import (
	"context"
	"sort"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// Evaluator is responsible for evaluating feature flags based on the evaluation context.
// It follows a specific evaluation order:
// 1. Check user-level override (highest priority)
// 2. Check tenant-level override
// 3. Check if flag is enabled (disabled flags return default with reason "disabled")
// 4. Evaluate targeting rules in priority order
// 5. For percentage/variant types, apply consistent hashing
// 6. Return default value (lowest priority)
type Evaluator struct {
	flagRepo     FeatureFlagRepository
	overrideRepo FlagOverrideRepository
}

// NewEvaluator creates a new feature flag evaluator
func NewEvaluator(flagRepo FeatureFlagRepository, overrideRepo FlagOverrideRepository) *Evaluator {
	return &Evaluator{
		flagRepo:     flagRepo,
		overrideRepo: overrideRepo,
	}
}

// Evaluate evaluates a single feature flag for the given context
func (e *Evaluator) Evaluate(ctx context.Context, flagKey string, evalCtx *EvaluationContext) EvaluationResult {
	// Get the feature flag
	flag, err := e.flagRepo.FindByKey(ctx, flagKey)
	if err != nil {
		// Check if it's a "not found" error vs other errors
		if domainErr, ok := err.(*shared.DomainError); ok && domainErr.Code == "NOT_FOUND" {
			return NewFlagNotFoundResult(flagKey)
		}
		// For other errors (database, network, etc.), return error result
		return NewErrorResult(flagKey, err)
	}

	return e.EvaluateFlag(ctx, flag, evalCtx)
}

// EvaluateFlag evaluates a pre-fetched feature flag for the given context
// This is useful when you already have the flag loaded (e.g., from cache)
func (e *Evaluator) EvaluateFlag(ctx context.Context, flag *FeatureFlag, evalCtx *EvaluationContext) EvaluationResult {
	if flag == nil {
		return NewFlagNotFoundResult("")
	}

	flagKey := flag.GetKey()
	flagVersion := flag.GetVersion()

	// Step 1: Check user-level override (highest priority)
	if evalCtx != nil && evalCtx.HasUser() {
		userID, err := uuid.Parse(evalCtx.UserID)
		if err == nil {
			override, err := e.findUserOverride(ctx, flagKey, userID)
			if err == nil && override != nil && override.IsActive() {
				result := NewEvaluationResult(flagKey, override.GetValue(), EvaluationReasonOverrideUser, flagVersion)
				return result
			}
		}
	}

	// Step 2: Check tenant-level override
	if evalCtx != nil && evalCtx.HasTenant() {
		tenantID, err := uuid.Parse(evalCtx.TenantID)
		if err == nil {
			override, err := e.findTenantOverride(ctx, flagKey, tenantID)
			if err == nil && override != nil && override.IsActive() {
				result := NewEvaluationResult(flagKey, override.GetValue(), EvaluationReasonOverrideTenant, flagVersion)
				return result
			}
		}
	}

	// Step 3: Check if flag is enabled
	if !flag.IsEnabled() {
		return NewDisabledResult(flagKey, flag.GetDefaultValue(), flagVersion)
	}

	// Step 4: Check plan restriction
	// If the flag has a required plan, verify the tenant's plan meets the requirement
	if flag.HasPlanRestriction() {
		tenantPlan := ""
		if evalCtx != nil {
			tenantPlan = evalCtx.UserPlan // UserPlan contains the tenant's subscription plan
		}
		if !flag.MeetsPlanRequirement(tenantPlan) {
			return NewPlanRestrictedResult(flagKey, flag.GetRequiredPlan().String(), flagVersion)
		}
	}

	// Step 5: Evaluate targeting rules in priority order
	rules := flag.GetRules()
	if len(rules) > 0 {
		// Sort rules by priority (lower number = higher priority)
		sortedRules := make([]TargetingRule, len(rules))
		copy(sortedRules, rules)
		sort.Slice(sortedRules, func(i, j int) bool {
			return sortedRules[i].Priority < sortedRules[j].Priority
		})

		for _, rule := range sortedRules {
			if e.matchRule(rule, evalCtx) {
				// Rule matches, check percentage if applicable
				if rule.Percentage < 100 {
					userID := ""
					if evalCtx != nil {
						userID = evalCtx.UserID
					}
					if !IsInPercentage(flagKey+":"+rule.RuleID, userID, rule.Percentage) {
						// User not in percentage rollout for this rule, continue to next rule
						continue
					}
				}

				result := NewEvaluationResult(flagKey, rule.GetValue(), EvaluationReasonRuleMatch, flagVersion)
				return result.WithRuleID(rule.RuleID)
			}
		}
	}

	// Step 6: Handle flag type-specific evaluation (percentage rollout, variant selection)
	result := e.evaluateByType(flag, evalCtx)
	return result
}

// EvaluateBatch evaluates multiple feature flags at once
// Returns a map of flag keys to evaluation results
func (e *Evaluator) EvaluateBatch(ctx context.Context, flagKeys []string, evalCtx *EvaluationContext) map[string]EvaluationResult {
	results := make(map[string]EvaluationResult, len(flagKeys))

	for _, key := range flagKeys {
		results[key] = e.Evaluate(ctx, key, evalCtx)
	}

	return results
}

// EvaluateAll evaluates all enabled feature flags.
// WARNING: This method evaluates up to 1000 flags. For systems with more flags,
// use EvaluateBatch with specific flag keys instead.
// Returns a map of flag keys to evaluation results
func (e *Evaluator) EvaluateAll(ctx context.Context, evalCtx *EvaluationContext) (map[string]EvaluationResult, error) {
	filter := shared.Filter{
		Page:     1,
		PageSize: 1000,
	}
	flags, err := e.flagRepo.FindEnabled(ctx, filter)
	if err != nil {
		return nil, err
	}

	results := make(map[string]EvaluationResult, len(flags))
	for i := range flags {
		flag := &flags[i]
		results[flag.GetKey()] = e.EvaluateFlag(ctx, flag, evalCtx)
	}

	return results, nil
}

// findUserOverride finds a user-level override for the flag
func (e *Evaluator) findUserOverride(ctx context.Context, flagKey string, userID uuid.UUID) (*FlagOverride, error) {
	if e.overrideRepo == nil {
		return nil, nil
	}
	return e.overrideRepo.FindByFlagKeyAndTarget(ctx, flagKey, OverrideTargetTypeUser, userID)
}

// findTenantOverride finds a tenant-level override for the flag
func (e *Evaluator) findTenantOverride(ctx context.Context, flagKey string, tenantID uuid.UUID) (*FlagOverride, error) {
	if e.overrideRepo == nil {
		return nil, nil
	}
	return e.overrideRepo.FindByFlagKeyAndTarget(ctx, flagKey, OverrideTargetTypeTenant, tenantID)
}

// matchRule checks if a targeting rule matches the evaluation context
func (e *Evaluator) matchRule(rule TargetingRule, evalCtx *EvaluationContext) bool {
	if !rule.HasConditions() {
		// Rule with no conditions always matches (catch-all rule)
		return true
	}

	// All conditions must match (AND logic)
	return MatchAllConditions(rule.GetConditions(), evalCtx)
}

// evaluateByType handles type-specific evaluation logic
func (e *Evaluator) evaluateByType(flag *FeatureFlag, evalCtx *EvaluationContext) EvaluationResult {
	flagKey := flag.GetKey()
	flagVersion := flag.GetVersion()
	defaultValue := flag.GetDefaultValue()

	switch flag.GetType() {
	case FlagTypeBoolean:
		// For boolean flags, return the default value
		return NewEvaluationResult(flagKey, defaultValue, EvaluationReasonDefault, flagVersion)

	case FlagTypePercentage:
		// For percentage flags, use consistent hashing to determine if user is included
		userID := ""
		if evalCtx != nil {
			userID = evalCtx.UserID
		}

		// Check if the default value's metadata contains a percentage
		percentage := 0
		if percentageVal, ok := defaultValue.GetMetadataValue("percentage"); ok {
			if p, ok := percentageVal.(int); ok {
				percentage = p
			} else if p, ok := percentageVal.(float64); ok {
				percentage = int(p)
			}
		}

		if IsInPercentage(flagKey, userID, percentage) {
			enabledValue := NewBooleanFlagValue(true)
			return NewEvaluationResult(flagKey, enabledValue, EvaluationReasonPercentage, flagVersion)
		}
		disabledValue := NewBooleanFlagValue(false)
		return NewEvaluationResult(flagKey, disabledValue, EvaluationReasonPercentage, flagVersion)

	case FlagTypeVariant:
		// For variant flags, use consistent hashing to select a variant
		userID := ""
		if evalCtx != nil {
			userID = evalCtx.UserID
		}

		// Check if variants are defined in metadata
		if variantsVal, ok := defaultValue.GetMetadataValue("variants"); ok {
			if variants, ok := variantsVal.([]string); ok && len(variants) > 0 {
				selectedVariant := SelectVariant(flagKey, userID, variants)
				variantValue := NewVariantFlagValue(selectedVariant)
				return NewEvaluationResult(flagKey, variantValue, EvaluationReasonDefault, flagVersion)
			}
			// Handle variants as []any (common when parsing JSON)
			if variantsAny, ok := variantsVal.([]any); ok && len(variantsAny) > 0 {
				variants := make([]string, 0, len(variantsAny))
				for _, v := range variantsAny {
					if s, ok := v.(string); ok {
						variants = append(variants, s)
					}
				}
				if len(variants) > 0 {
					selectedVariant := SelectVariant(flagKey, userID, variants)
					variantValue := NewVariantFlagValue(selectedVariant)
					return NewEvaluationResult(flagKey, variantValue, EvaluationReasonDefault, flagVersion)
				}
			}
		}

		// Return default variant
		return NewEvaluationResult(flagKey, defaultValue, EvaluationReasonDefault, flagVersion)

	case FlagTypeUserSegment:
		// User segment type relies entirely on targeting rules
		// If we reach here, no rules matched, return default
		return NewEvaluationResult(flagKey, defaultValue, EvaluationReasonDefault, flagVersion)

	default:
		return NewEvaluationResult(flagKey, defaultValue, EvaluationReasonDefault, flagVersion)
	}
}

// IsEnabled is a convenience method to check if a flag is enabled for the given context
func (e *Evaluator) IsEnabled(ctx context.Context, flagKey string, evalCtx *EvaluationContext) bool {
	result := e.Evaluate(ctx, flagKey, evalCtx)
	return result.IsEnabled()
}

// GetVariant is a convenience method to get the variant value for the given context
func (e *Evaluator) GetVariant(ctx context.Context, flagKey string, evalCtx *EvaluationContext) string {
	result := e.Evaluate(ctx, flagKey, evalCtx)
	return result.Variant
}

// PureEvaluator evaluates flags without repository access (for cached/pre-loaded flags)
// This is useful for high-performance scenarios where flags are already loaded
type PureEvaluator struct{}

// NewPureEvaluator creates a new pure evaluator
func NewPureEvaluator() *PureEvaluator {
	return &PureEvaluator{}
}

// Evaluate evaluates a feature flag with optional overrides
func (e *PureEvaluator) Evaluate(flag *FeatureFlag, evalCtx *EvaluationContext, userOverride, tenantOverride *FlagOverride) EvaluationResult {
	if flag == nil {
		return NewFlagNotFoundResult("")
	}

	flagKey := flag.GetKey()
	flagVersion := flag.GetVersion()

	// Step 1: Check user-level override (highest priority)
	if userOverride != nil && userOverride.IsActive() {
		return NewEvaluationResult(flagKey, userOverride.GetValue(), EvaluationReasonOverrideUser, flagVersion)
	}

	// Step 2: Check tenant-level override
	if tenantOverride != nil && tenantOverride.IsActive() {
		return NewEvaluationResult(flagKey, tenantOverride.GetValue(), EvaluationReasonOverrideTenant, flagVersion)
	}

	// Step 3: Check if flag is enabled
	if !flag.IsEnabled() {
		return NewDisabledResult(flagKey, flag.GetDefaultValue(), flagVersion)
	}

	// Step 4: Check plan restriction
	if flag.HasPlanRestriction() {
		tenantPlan := ""
		if evalCtx != nil {
			tenantPlan = evalCtx.UserPlan
		}
		if !flag.MeetsPlanRequirement(tenantPlan) {
			return NewPlanRestrictedResult(flagKey, flag.GetRequiredPlan().String(), flagVersion)
		}
	}

	// Step 5: Evaluate targeting rules
	rules := flag.GetRules()
	if len(rules) > 0 {
		sortedRules := make([]TargetingRule, len(rules))
		copy(sortedRules, rules)
		sort.Slice(sortedRules, func(i, j int) bool {
			return sortedRules[i].Priority < sortedRules[j].Priority
		})

		for _, rule := range sortedRules {
			if e.matchRule(rule, evalCtx) {
				if rule.Percentage < 100 {
					userID := ""
					if evalCtx != nil {
						userID = evalCtx.UserID
					}
					if !IsInPercentage(flagKey+":"+rule.RuleID, userID, rule.Percentage) {
						continue
					}
				}

				result := NewEvaluationResult(flagKey, rule.GetValue(), EvaluationReasonRuleMatch, flagVersion)
				return result.WithRuleID(rule.RuleID)
			}
		}
	}

	// Step 6: Return default value
	return e.evaluateByType(flag, evalCtx)
}

// matchRule checks if a targeting rule matches the evaluation context
func (e *PureEvaluator) matchRule(rule TargetingRule, evalCtx *EvaluationContext) bool {
	if !rule.HasConditions() {
		return true
	}
	return MatchAllConditions(rule.GetConditions(), evalCtx)
}

// evaluateByType handles type-specific evaluation logic
func (e *PureEvaluator) evaluateByType(flag *FeatureFlag, evalCtx *EvaluationContext) EvaluationResult {
	flagKey := flag.GetKey()
	flagVersion := flag.GetVersion()
	defaultValue := flag.GetDefaultValue()

	switch flag.GetType() {
	case FlagTypeBoolean:
		return NewEvaluationResult(flagKey, defaultValue, EvaluationReasonDefault, flagVersion)

	case FlagTypePercentage:
		userID := ""
		if evalCtx != nil {
			userID = evalCtx.UserID
		}

		percentage := 0
		if percentageVal, ok := defaultValue.GetMetadataValue("percentage"); ok {
			if p, ok := percentageVal.(int); ok {
				percentage = p
			} else if p, ok := percentageVal.(float64); ok {
				percentage = int(p)
			}
		}

		if IsInPercentage(flagKey, userID, percentage) {
			return NewEvaluationResult(flagKey, NewBooleanFlagValue(true), EvaluationReasonPercentage, flagVersion)
		}
		return NewEvaluationResult(flagKey, NewBooleanFlagValue(false), EvaluationReasonPercentage, flagVersion)

	case FlagTypeVariant:
		userID := ""
		if evalCtx != nil {
			userID = evalCtx.UserID
		}

		if variantsVal, ok := defaultValue.GetMetadataValue("variants"); ok {
			if variants, ok := variantsVal.([]string); ok && len(variants) > 0 {
				selectedVariant := SelectVariant(flagKey, userID, variants)
				return NewEvaluationResult(flagKey, NewVariantFlagValue(selectedVariant), EvaluationReasonDefault, flagVersion)
			}
			if variantsAny, ok := variantsVal.([]any); ok && len(variantsAny) > 0 {
				variants := make([]string, 0, len(variantsAny))
				for _, v := range variantsAny {
					if s, ok := v.(string); ok {
						variants = append(variants, s)
					}
				}
				if len(variants) > 0 {
					selectedVariant := SelectVariant(flagKey, userID, variants)
					return NewEvaluationResult(flagKey, NewVariantFlagValue(selectedVariant), EvaluationReasonDefault, flagVersion)
				}
			}
		}

		return NewEvaluationResult(flagKey, defaultValue, EvaluationReasonDefault, flagVersion)

	default:
		return NewEvaluationResult(flagKey, defaultValue, EvaluationReasonDefault, flagVersion)
	}
}
