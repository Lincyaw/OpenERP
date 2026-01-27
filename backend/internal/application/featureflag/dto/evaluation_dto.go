package dto

import (
	"time"

	"github.com/erp/backend/internal/domain/featureflag"
)

// EvaluationContextDTO represents the context for evaluating a feature flag
type EvaluationContextDTO struct {
	TenantID       string         `json:"tenant_id,omitempty"`
	UserID         string         `json:"user_id,omitempty"`
	UserRole       string         `json:"user_role,omitempty"`
	UserPlan       string         `json:"user_plan,omitempty"`
	UserAttributes map[string]any `json:"user_attributes,omitempty"`
	RequestID      string         `json:"request_id,omitempty"`
	Environment    string         `json:"environment,omitempty"`
}

// ToDomain converts EvaluationContextDTO to domain EvaluationContext
func (d EvaluationContextDTO) ToDomain() *featureflag.EvaluationContext {
	ctx := featureflag.NewEvaluationContext()
	if d.TenantID != "" {
		ctx = ctx.WithTenant(d.TenantID)
	}
	if d.UserID != "" {
		ctx = ctx.WithUser(d.UserID)
	}
	if d.UserRole != "" {
		ctx = ctx.WithUserRole(d.UserRole)
	}
	if d.UserPlan != "" {
		ctx = ctx.WithUserPlan(d.UserPlan)
	}
	if d.RequestID != "" {
		ctx = ctx.WithRequestID(d.RequestID)
	}
	if d.Environment != "" {
		ctx = ctx.WithEnvironment(d.Environment)
	}
	for key, value := range d.UserAttributes {
		ctx = ctx.WithAttribute(key, value)
	}
	return ctx
}

// EvaluateFlagRequest represents the request to evaluate a feature flag
type EvaluateFlagRequest struct {
	Key     string               `json:"key" binding:"required"`
	Context EvaluationContextDTO `json:"context"`
}

// EvaluateFlagResponse represents the result of evaluating a feature flag
type EvaluateFlagResponse struct {
	Key         string       `json:"key"`
	Enabled     bool         `json:"enabled"`
	Variant     string       `json:"variant,omitempty"`
	Value       FlagValueDTO `json:"value"`
	Reason      string       `json:"reason"`
	RuleID      string       `json:"rule_id,omitempty"`
	FlagVersion int          `json:"flag_version"`
	EvaluatedAt time.Time    `json:"evaluated_at"`
}

// ToEvaluateFlagResponse converts domain EvaluationResult to DTO
func ToEvaluateFlagResponse(result featureflag.EvaluationResult) *EvaluateFlagResponse {
	return &EvaluateFlagResponse{
		Key:         result.Key,
		Enabled:     result.Enabled,
		Variant:     result.Variant,
		Value:       ToFlagValueDTO(result.Value),
		Reason:      string(result.Reason),
		RuleID:      result.RuleID,
		FlagVersion: result.FlagVersion,
		EvaluatedAt: result.EvaluatedAt,
	}
}

// BatchEvaluateRequest represents the request to evaluate multiple flags
type BatchEvaluateRequest struct {
	Keys    []string             `json:"keys" binding:"required,min=1,max=100"`
	Context EvaluationContextDTO `json:"context"`
}

// BatchEvaluateResponse represents the result of evaluating multiple flags
type BatchEvaluateResponse struct {
	Results map[string]*EvaluateFlagResponse `json:"results"`
}

// ToBatchEvaluateResponse converts domain batch results to DTO
func ToBatchEvaluateResponse(results map[string]featureflag.EvaluationResult) *BatchEvaluateResponse {
	response := &BatchEvaluateResponse{
		Results: make(map[string]*EvaluateFlagResponse, len(results)),
	}
	for key, result := range results {
		response.Results[key] = ToEvaluateFlagResponse(result)
	}
	return response
}

// GetClientConfigRequest represents the request to get all flags for a client
type GetClientConfigRequest struct {
	Context EvaluationContextDTO `json:"context"`
}

// FlagClientConfig represents a simplified flag value for client SDKs
type FlagClientConfig struct {
	Enabled bool   `json:"enabled"`
	Variant string `json:"variant,omitempty"`
}

// GetClientConfigResponse represents all flag values for a client
type GetClientConfigResponse struct {
	Flags       map[string]FlagClientConfig `json:"flags"`
	EvaluatedAt time.Time                   `json:"evaluated_at"`
}

// ToGetClientConfigResponse converts domain batch results to client config DTO
func ToGetClientConfigResponse(results map[string]featureflag.EvaluationResult) *GetClientConfigResponse {
	response := &GetClientConfigResponse{
		Flags:       make(map[string]FlagClientConfig, len(results)),
		EvaluatedAt: time.Now(),
	}
	for key, result := range results {
		response.Flags[key] = FlagClientConfig{
			Enabled: result.Enabled,
			Variant: result.Variant,
		}
	}
	return response
}
