package handler

import (
	"time"

	appIdentity "github.com/erp/backend/internal/application/identity"
	"github.com/google/uuid"
)

// ============================================================================
// Batch Operation Request DTOs
// ============================================================================

// BatchSuspendRequest represents a batch suspend request
//
//	@Description Request body for suspending multiple tenants at once
type BatchSuspendRequest struct {
	TenantIDs             []string   `json:"tenant_ids" binding:"required,min=1,max=100,dive,uuid" example:"550e8400-e29b-41d4-a716-446655440001,550e8400-e29b-41d4-a716-446655440002"`
	Reason                string     `json:"reason,omitempty" binding:"omitempty,max=500" example:"Bulk suspension due to payment default"`
	ScheduledReactivateAt *time.Time `json:"scheduled_reactivate_at,omitempty" example:"2024-03-01T00:00:00Z"`
}

// BatchActivateRequest represents a batch activate request
//
//	@Description Request body for activating multiple tenants at once
type BatchActivateRequest struct {
	TenantIDs []string `json:"tenant_ids" binding:"required,min=1,max=100,dive,uuid" example:"550e8400-e29b-41d4-a716-446655440001,550e8400-e29b-41d4-a716-446655440002"`
}

// BatchChangePlanRequest represents a batch plan change request
//
//	@Description Request body for changing subscription plan for multiple tenants
type BatchChangePlanRequest struct {
	TenantIDs []string `json:"tenant_ids" binding:"required,min=1,max=100,dive,uuid" example:"550e8400-e29b-41d4-a716-446655440001,550e8400-e29b-41d4-a716-446655440002"`
	NewPlan   string   `json:"new_plan" binding:"required,oneof=free basic pro enterprise" example:"pro"`
	Reason    string   `json:"reason,omitempty" binding:"omitempty,max=500" example:"Promotional upgrade for Q1 campaign"`
}

// BatchDeleteRequest represents a batch delete request
//
//	@Description Request body for soft-deleting multiple tenants at once
type BatchDeleteRequest struct {
	TenantIDs []string `json:"tenant_ids" binding:"required,min=1,max=100,dive,uuid" example:"550e8400-e29b-41d4-a716-446655440001,550e8400-e29b-41d4-a716-446655440002"`
}

// ============================================================================
// Batch Operation Response DTOs
// ============================================================================

// BatchPreviewResponse represents a batch operation preview
//
//	@Description Preview of batch operation showing which tenants will be affected
type BatchPreviewResponse struct {
	OperationType   string                       `json:"operation_type" example:"suspend"`
	TotalCount      int                          `json:"total_count" example:"10"`
	CanProcessCount int                          `json:"can_process_count" example:"8"`
	WillSkipCount   int                          `json:"will_skip_count" example:"2"`
	AffectedTenants []BatchPreviewTenantResponse `json:"affected_tenants"`
	Warnings        []string                     `json:"warnings,omitempty" example:"2 tenants are already suspended and will be skipped"`
}

// BatchPreviewTenantResponse represents a tenant in the batch preview
//
//	@Description Individual tenant details in batch operation preview
type BatchPreviewTenantResponse struct {
	ID            uuid.UUID `json:"id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Code          string    `json:"code" example:"ACME001"`
	Name          string    `json:"name" example:"Acme Corporation"`
	CurrentStatus string    `json:"current_status" example:"active"`
	CurrentPlan   string    `json:"current_plan,omitempty" example:"pro"`
	CanProcess    bool      `json:"can_process" example:"true"`
	SkipReason    string    `json:"skip_reason,omitempty" example:"Tenant is already suspended"`
}

// BatchOperationResponse represents a batch operation result
//
//	@Description Result of batch operation execution with detailed status
type BatchOperationResponse struct {
	ID              uuid.UUID                    `json:"id" example:"550e8400-e29b-41d4-a716-446655440099"`
	OperationType   string                       `json:"operation_type" example:"suspend"`
	Status          string                       `json:"status" example:"completed"`
	TotalCount      int                          `json:"total_count" example:"10"`
	ProcessedCount  int                          `json:"processed_count" example:"10"`
	SuccessCount    int                          `json:"success_count" example:"8"`
	FailedCount     int                          `json:"failed_count" example:"0"`
	SkippedCount    int                          `json:"skipped_count" example:"2"`
	Progress        float64                      `json:"progress" example:"100.0"`
	Items           []BatchOperationItemResponse `json:"items"`
	CreatedByUserID uuid.UUID                    `json:"created_by_user_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	ConfirmedAt     *time.Time                   `json:"confirmed_at,omitempty" example:"2024-02-01T10:30:00Z"`
	CompletedAt     *time.Time                   `json:"completed_at,omitempty" example:"2024-02-01T10:30:05Z"`
	CreatedAt       time.Time                    `json:"created_at" example:"2024-02-01T10:29:55Z"`
	UpdatedAt       time.Time                    `json:"updated_at" example:"2024-02-01T10:30:05Z"`
}

// BatchOperationItemResponse represents a single item in a batch operation
//
//	@Description Individual tenant result in batch operation
type BatchOperationItemResponse struct {
	ID           uuid.UUID  `json:"id" example:"550e8400-e29b-41d4-a716-446655440088"`
	TenantID     uuid.UUID  `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	TenantCode   string     `json:"tenant_code" example:"ACME001"`
	TenantName   string     `json:"tenant_name" example:"Acme Corporation"`
	Status       string     `json:"status" example:"success"`
	ErrorMessage string     `json:"error_message,omitempty" example:""`
	ProcessedAt  *time.Time `json:"processed_at,omitempty" example:"2024-02-01T10:30:02Z"`
}

// ============================================================================
// Conversion Functions
// ============================================================================

// parseUUIDs parses string UUIDs to uuid.UUID
func parseUUIDs(strs []string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, len(strs))
	for i, s := range strs {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, err
		}
		ids[i] = id
	}
	return ids, nil
}

// toBatchPreviewResponse converts application DTO to HTTP response
func toBatchPreviewResponse(dto *appIdentity.BatchPreviewDTO) BatchPreviewResponse {
	tenants := make([]BatchPreviewTenantResponse, len(dto.AffectedTenants))
	for i, t := range dto.AffectedTenants {
		tenants[i] = BatchPreviewTenantResponse{
			ID:            t.ID,
			Code:          t.Code,
			Name:          t.Name,
			CurrentStatus: t.CurrentStatus,
			CurrentPlan:   t.CurrentPlan,
			CanProcess:    t.CanProcess,
			SkipReason:    t.SkipReason,
		}
	}

	return BatchPreviewResponse{
		OperationType:   dto.OperationType,
		TotalCount:      dto.TotalCount,
		CanProcessCount: dto.CanProcessCount,
		WillSkipCount:   dto.WillSkipCount,
		AffectedTenants: tenants,
		Warnings:        dto.Warnings,
	}
}

// toBatchOperationResponse converts application DTO to HTTP response
func toBatchOperationResponse(dto *appIdentity.BatchOperationDTO) BatchOperationResponse {
	items := make([]BatchOperationItemResponse, len(dto.Items))
	for i, item := range dto.Items {
		items[i] = BatchOperationItemResponse{
			ID:           item.ID,
			TenantID:     item.TenantID,
			TenantCode:   item.TenantCode,
			TenantName:   item.TenantName,
			Status:       item.Status,
			ErrorMessage: item.ErrorMessage,
			ProcessedAt:  item.ProcessedAt,
		}
	}

	return BatchOperationResponse{
		ID:              dto.ID,
		OperationType:   dto.OperationType,
		Status:          dto.Status,
		TotalCount:      dto.TotalCount,
		ProcessedCount:  dto.ProcessedCount,
		SuccessCount:    dto.SuccessCount,
		FailedCount:     dto.FailedCount,
		SkippedCount:    dto.SkippedCount,
		Progress:        dto.Progress,
		Items:           items,
		CreatedByUserID: dto.CreatedByUserID,
		ConfirmedAt:     dto.ConfirmedAt,
		CompletedAt:     dto.CompletedAt,
		CreatedAt:       dto.CreatedAt,
		UpdatedAt:       dto.UpdatedAt,
	}
}
