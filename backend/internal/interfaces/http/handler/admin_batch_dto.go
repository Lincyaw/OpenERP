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
type BatchSuspendRequest struct {
	TenantIDs             []string   `json:"tenant_ids" binding:"required,min=1,max=100,dive,uuid"`
	Reason                string     `json:"reason,omitempty" binding:"omitempty,max=500"`
	ScheduledReactivateAt *time.Time `json:"scheduled_reactivate_at,omitempty"`
}

// BatchActivateRequest represents a batch activate request
type BatchActivateRequest struct {
	TenantIDs []string `json:"tenant_ids" binding:"required,min=1,max=100,dive,uuid"`
}

// BatchChangePlanRequest represents a batch plan change request
type BatchChangePlanRequest struct {
	TenantIDs []string `json:"tenant_ids" binding:"required,min=1,max=100,dive,uuid"`
	NewPlan   string   `json:"new_plan" binding:"required,oneof=free basic pro enterprise"`
	Reason    string   `json:"reason,omitempty" binding:"omitempty,max=500"`
}

// BatchDeleteRequest represents a batch delete request
type BatchDeleteRequest struct {
	TenantIDs []string `json:"tenant_ids" binding:"required,min=1,max=100,dive,uuid"`
}

// ============================================================================
// Batch Operation Response DTOs
// ============================================================================

// BatchPreviewResponse represents a batch operation preview
type BatchPreviewResponse struct {
	OperationType   string                       `json:"operation_type"`
	TotalCount      int                          `json:"total_count"`
	CanProcessCount int                          `json:"can_process_count"`
	WillSkipCount   int                          `json:"will_skip_count"`
	AffectedTenants []BatchPreviewTenantResponse `json:"affected_tenants"`
	Warnings        []string                     `json:"warnings,omitempty"`
}

// BatchPreviewTenantResponse represents a tenant in the batch preview
type BatchPreviewTenantResponse struct {
	ID            uuid.UUID `json:"id"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	CurrentStatus string    `json:"current_status"`
	CurrentPlan   string    `json:"current_plan,omitempty"`
	CanProcess    bool      `json:"can_process"`
	SkipReason    string    `json:"skip_reason,omitempty"`
}

// BatchOperationResponse represents a batch operation result
type BatchOperationResponse struct {
	ID              uuid.UUID                    `json:"id"`
	OperationType   string                       `json:"operation_type"`
	Status          string                       `json:"status"`
	TotalCount      int                          `json:"total_count"`
	ProcessedCount  int                          `json:"processed_count"`
	SuccessCount    int                          `json:"success_count"`
	FailedCount     int                          `json:"failed_count"`
	SkippedCount    int                          `json:"skipped_count"`
	Progress        float64                      `json:"progress"`
	Items           []BatchOperationItemResponse `json:"items"`
	CreatedByUserID uuid.UUID                    `json:"created_by_user_id"`
	ConfirmedAt     *time.Time                   `json:"confirmed_at,omitempty"`
	CompletedAt     *time.Time                   `json:"completed_at,omitempty"`
	CreatedAt       time.Time                    `json:"created_at"`
	UpdatedAt       time.Time                    `json:"updated_at"`
}

// BatchOperationItemResponse represents a single item in a batch operation
type BatchOperationItemResponse struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	TenantCode   string     `json:"tenant_code"`
	TenantName   string     `json:"tenant_name"`
	Status       string     `json:"status"`
	ErrorMessage string     `json:"error_message,omitempty"`
	ProcessedAt  *time.Time `json:"processed_at,omitempty"`
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
