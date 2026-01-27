package dto

import (
	"time"

	"github.com/erp/backend/internal/domain/featureflag"
	"github.com/google/uuid"
)

// FlagValueDTO represents the value of a feature flag in API responses
type FlagValueDTO struct {
	Enabled  bool           `json:"enabled"`
	Variant  string         `json:"variant,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ToFlagValueDTO converts domain FlagValue to DTO
func ToFlagValueDTO(v featureflag.FlagValue) FlagValueDTO {
	return FlagValueDTO{
		Enabled:  v.Enabled,
		Variant:  v.Variant,
		Metadata: v.GetMetadata(),
	}
}

// ToDomain converts FlagValueDTO to domain FlagValue
func (d FlagValueDTO) ToDomain() featureflag.FlagValue {
	return featureflag.NewFlagValueWithMetadata(d.Enabled, d.Variant, d.Metadata)
}

// ConditionDTO represents a targeting condition in API requests/responses
type ConditionDTO struct {
	Attribute string   `json:"attribute" binding:"required"`
	Operator  string   `json:"operator" binding:"required"`
	Values    []string `json:"values" binding:"required,min=1"`
}

// ToConditionDTO converts domain Condition to DTO
func ToConditionDTO(c featureflag.Condition) ConditionDTO {
	return ConditionDTO{
		Attribute: c.GetAttribute(),
		Operator:  string(c.GetOperator()),
		Values:    c.GetValues(),
	}
}

// ToDomain converts ConditionDTO to domain Condition
func (d ConditionDTO) ToDomain() (featureflag.Condition, error) {
	return featureflag.NewCondition(d.Attribute, featureflag.ConditionOperator(d.Operator), d.Values)
}

// TargetingRuleDTO represents a targeting rule in API requests/responses
type TargetingRuleDTO struct {
	RuleID     string         `json:"rule_id" binding:"required"`
	Priority   int            `json:"priority" binding:"min=0"`
	Conditions []ConditionDTO `json:"conditions"`
	Value      FlagValueDTO   `json:"value"`
	Percentage int            `json:"percentage" binding:"min=0,max=100"`
}

// ToTargetingRuleDTO converts domain TargetingRule to DTO
func ToTargetingRuleDTO(r featureflag.TargetingRule) TargetingRuleDTO {
	conditions := make([]ConditionDTO, len(r.GetConditions()))
	for i, c := range r.GetConditions() {
		conditions[i] = ToConditionDTO(c)
	}
	return TargetingRuleDTO{
		RuleID:     r.GetRuleID(),
		Priority:   r.GetPriority(),
		Conditions: conditions,
		Value:      ToFlagValueDTO(r.GetValue()),
		Percentage: r.GetPercentage(),
	}
}

// ToDomain converts TargetingRuleDTO to domain TargetingRule
func (d TargetingRuleDTO) ToDomain() (featureflag.TargetingRule, error) {
	conditions := make([]featureflag.Condition, len(d.Conditions))
	for i, c := range d.Conditions {
		condition, err := c.ToDomain()
		if err != nil {
			return featureflag.TargetingRule{}, err
		}
		conditions[i] = condition
	}

	return featureflag.NewTargetingRuleWithPercentage(
		d.RuleID,
		d.Priority,
		conditions,
		d.Value.ToDomain(),
		d.Percentage,
	)
}

// CreateFlagRequest represents the request to create a new feature flag
type CreateFlagRequest struct {
	Key          string             `json:"key" binding:"required,min=1,max=100"`
	Name         string             `json:"name" binding:"required,min=1,max=200"`
	Description  string             `json:"description,omitempty"`
	Type         string             `json:"type" binding:"required,oneof=boolean percentage variant user_segment"`
	DefaultValue FlagValueDTO       `json:"default_value"`
	Rules        []TargetingRuleDTO `json:"rules,omitempty"`
	Tags         []string           `json:"tags,omitempty"`
}

// UpdateFlagRequest represents the request to update a feature flag
type UpdateFlagRequest struct {
	Name         *string             `json:"name,omitempty"`
	Description  *string             `json:"description,omitempty"`
	DefaultValue *FlagValueDTO       `json:"default_value,omitempty"`
	Rules        *[]TargetingRuleDTO `json:"rules,omitempty"`
	Tags         *[]string           `json:"tags,omitempty"`
}

// FlagResponse represents a feature flag in API responses
type FlagResponse struct {
	ID           uuid.UUID          `json:"id"`
	Key          string             `json:"key"`
	Name         string             `json:"name"`
	Description  string             `json:"description,omitempty"`
	Type         string             `json:"type"`
	Status       string             `json:"status"`
	DefaultValue FlagValueDTO       `json:"default_value"`
	Rules        []TargetingRuleDTO `json:"rules,omitempty"`
	Tags         []string           `json:"tags,omitempty"`
	Version      int                `json:"version"`
	CreatedBy    *uuid.UUID         `json:"created_by,omitempty"`
	UpdatedBy    *uuid.UUID         `json:"updated_by,omitempty"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

// ToFlagResponse converts domain FeatureFlag to FlagResponse
func ToFlagResponse(flag *featureflag.FeatureFlag) *FlagResponse {
	if flag == nil {
		return nil
	}

	rules := make([]TargetingRuleDTO, len(flag.GetRules()))
	for i, r := range flag.GetRules() {
		rules[i] = ToTargetingRuleDTO(r)
	}

	return &FlagResponse{
		ID:           flag.ID,
		Key:          flag.GetKey(),
		Name:         flag.GetName(),
		Description:  flag.GetDescription(),
		Type:         string(flag.GetType()),
		Status:       string(flag.GetStatus()),
		DefaultValue: ToFlagValueDTO(flag.GetDefaultValue()),
		Rules:        rules,
		Tags:         flag.GetTags(),
		Version:      flag.GetVersion(),
		CreatedBy:    flag.GetCreatedBy(),
		UpdatedBy:    flag.GetUpdatedBy(),
		CreatedAt:    flag.CreatedAt,
		UpdatedAt:    flag.UpdatedAt,
	}
}

// FlagListFilter represents filter options for listing flags
type FlagListFilter struct {
	Page     int      `form:"page,default=1" binding:"min=1"`
	PageSize int      `form:"page_size,default=20" binding:"min=1,max=100"`
	Status   *string  `form:"status,omitempty"`
	Type     *string  `form:"type,omitempty"`
	Tags     []string `form:"tags,omitempty"`
	Search   string   `form:"search,omitempty"`
}

// FlagListResponse represents a paginated list of feature flags
type FlagListResponse struct {
	Flags      []FlagResponse `json:"flags"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

// ToFlagListResponse creates a FlagListResponse from domain flags
func ToFlagListResponse(flags []featureflag.FeatureFlag, total int64, page, pageSize int) *FlagListResponse {
	// Prevent division by zero
	if pageSize <= 0 {
		pageSize = 20
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	flagResponses := make([]FlagResponse, len(flags))
	for i := range flags {
		flagResponses[i] = *ToFlagResponse(&flags[i])
	}

	return &FlagListResponse{
		Flags:      flagResponses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}
