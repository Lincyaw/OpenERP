package dto

import (
	"time"

	"github.com/erp/backend/internal/domain/featureflag"
	"github.com/google/uuid"
)

// CreateOverrideRequest represents the request to create a flag override
type CreateOverrideRequest struct {
	TargetType string       `json:"target_type" binding:"required,oneof=user tenant"`
	TargetID   uuid.UUID    `json:"target_id" binding:"required"`
	Value      FlagValueDTO `json:"value"`
	Reason     string       `json:"reason,omitempty" binding:"max=500"`
	ExpiresAt  *time.Time   `json:"expires_at,omitempty"`
}

// OverrideResponse represents a flag override in API responses
type OverrideResponse struct {
	ID         uuid.UUID    `json:"id"`
	FlagKey    string       `json:"flag_key"`
	TargetType string       `json:"target_type"`
	TargetID   uuid.UUID    `json:"target_id"`
	Value      FlagValueDTO `json:"value"`
	Reason     string       `json:"reason,omitempty"`
	ExpiresAt  *time.Time   `json:"expires_at,omitempty"`
	IsExpired  bool         `json:"is_expired"`
	CreatedBy  *uuid.UUID   `json:"created_by,omitempty"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"`
}

// ToOverrideResponse converts domain FlagOverride to DTO
func ToOverrideResponse(override *featureflag.FlagOverride) *OverrideResponse {
	if override == nil {
		return nil
	}
	return &OverrideResponse{
		ID:         override.ID,
		FlagKey:    override.GetFlagKey(),
		TargetType: string(override.GetTargetType()),
		TargetID:   override.GetTargetID(),
		Value:      ToFlagValueDTO(override.GetValue()),
		Reason:     override.GetReason(),
		ExpiresAt:  override.GetExpiresAt(),
		IsExpired:  override.IsExpired(),
		CreatedBy:  override.GetCreatedBy(),
		CreatedAt:  override.CreatedAt,
		UpdatedAt:  override.UpdatedAt,
	}
}

// OverrideListFilter represents filter options for listing overrides
type OverrideListFilter struct {
	Page       int     `form:"page,default=1" binding:"min=1"`
	PageSize   int     `form:"page_size,default=20" binding:"min=1,max=100"`
	TargetType *string `form:"target_type,omitempty"`
	Expired    *bool   `form:"expired,omitempty"`
}

// OverrideListResponse represents a paginated list of flag overrides
type OverrideListResponse struct {
	Overrides  []OverrideResponse `json:"overrides"`
	Total      int64              `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalPages int                `json:"total_pages"`
}

// ToOverrideListResponse creates an OverrideListResponse from domain overrides
func ToOverrideListResponse(overrides []featureflag.FlagOverride, total int64, page, pageSize int) *OverrideListResponse {
	if pageSize <= 0 {
		pageSize = 20
	}
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	overrideResponses := make([]OverrideResponse, len(overrides))
	for i := range overrides {
		overrideResponses[i] = *ToOverrideResponse(&overrides[i])
	}

	return &OverrideListResponse{
		Overrides:  overrideResponses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}

// CleanupExpiredOverridesResponse represents the response for cleaning up expired overrides
type CleanupExpiredOverridesResponse struct {
	DeletedCount int64 `json:"deleted_count"`
}

// UpdateOverrideRequest represents the request to update a flag override
type UpdateOverrideRequest struct {
	Value     *FlagValueDTO `json:"value,omitempty"`
	Reason    *string       `json:"reason,omitempty" binding:"omitempty,max=500"`
	ExpiresAt *time.Time    `json:"expires_at,omitempty"`
}
