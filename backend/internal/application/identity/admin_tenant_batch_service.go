package identity

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ============================================================================
// Batch Operation Input/Output DTOs
// ============================================================================

// BatchSuspendInput contains input for batch suspend operation
type BatchSuspendInput struct {
	TenantIDs             []uuid.UUID `json:"tenant_ids"`
	Reason                string      `json:"reason,omitempty"`
	ScheduledReactivateAt *time.Time  `json:"scheduled_reactivate_at,omitempty"`
}

// BatchActivateInput contains input for batch activate operation
type BatchActivateInput struct {
	TenantIDs []uuid.UUID `json:"tenant_ids"`
}

// BatchChangePlanInput contains input for batch plan change operation
type BatchChangePlanInput struct {
	TenantIDs []uuid.UUID `json:"tenant_ids"`
	NewPlan   string      `json:"new_plan"`
	Reason    string      `json:"reason,omitempty"`
}

// BatchDeleteInput contains input for batch delete operation
type BatchDeleteInput struct {
	TenantIDs []uuid.UUID `json:"tenant_ids"`
}

// BatchOperationDTO represents a batch operation for API responses
type BatchOperationDTO struct {
	ID              uuid.UUID               `json:"id"`
	OperationType   string                  `json:"operation_type"`
	Status          string                  `json:"status"`
	TotalCount      int                     `json:"total_count"`
	ProcessedCount  int                     `json:"processed_count"`
	SuccessCount    int                     `json:"success_count"`
	FailedCount     int                     `json:"failed_count"`
	SkippedCount    int                     `json:"skipped_count"`
	Progress        float64                 `json:"progress"`
	Items           []BatchOperationItemDTO `json:"items"`
	CreatedByUserID uuid.UUID               `json:"created_by_user_id"`
	ConfirmedAt     *time.Time              `json:"confirmed_at,omitempty"`
	CompletedAt     *time.Time              `json:"completed_at,omitempty"`
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
}

// BatchOperationItemDTO represents a single item in a batch operation
type BatchOperationItemDTO struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	TenantCode   string     `json:"tenant_code"`
	TenantName   string     `json:"tenant_name"`
	Status       string     `json:"status"`
	ErrorMessage string     `json:"error_message,omitempty"`
	ProcessedAt  *time.Time `json:"processed_at,omitempty"`
}

// BatchPreviewDTO represents a preview of a batch operation
type BatchPreviewDTO struct {
	OperationType   string                  `json:"operation_type"`
	TotalCount      int                     `json:"total_count"`
	CanProcessCount int                     `json:"can_process_count"`
	WillSkipCount   int                     `json:"will_skip_count"`
	AffectedTenants []BatchPreviewTenantDTO `json:"affected_tenants"`
	Warnings        []string                `json:"warnings,omitempty"`
}

// BatchPreviewTenantDTO represents a tenant in the batch preview
type BatchPreviewTenantDTO struct {
	ID            uuid.UUID `json:"id"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	CurrentStatus string    `json:"current_status"`
	CurrentPlan   string    `json:"current_plan,omitempty"`
	CanProcess    bool      `json:"can_process"`
	SkipReason    string    `json:"skip_reason,omitempty"`
}

// BatchOperationFilterInput contains filter options for batch operation queries
type BatchOperationFilterInput struct {
	Page          int     `json:"page"`
	PageSize      int     `json:"page_size"`
	OperationType *string `json:"operation_type,omitempty"`
	Status        *string `json:"status,omitempty"`
}

// BatchOperationListResult represents paginated batch operation results
type BatchOperationListResult struct {
	Operations []BatchOperationDTO `json:"operations"`
	Total      int64               `json:"total"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"page_size"`
	TotalPages int                 `json:"total_pages"`
}

// ============================================================================
// Batch Operation Methods
// ============================================================================

// PreviewBatchSuspend creates a preview for batch suspend operation
func (s *AdminTenantService) PreviewBatchSuspend(ctx context.Context, input BatchSuspendInput) (*BatchPreviewDTO, error) {
	return s.previewBatchOperation(ctx, identity.BatchOperationTypeSuspend, input.TenantIDs, nil)
}

// PreviewBatchActivate creates a preview for batch activate operation
func (s *AdminTenantService) PreviewBatchActivate(ctx context.Context, input BatchActivateInput) (*BatchPreviewDTO, error) {
	return s.previewBatchOperation(ctx, identity.BatchOperationTypeActivate, input.TenantIDs, nil)
}

// PreviewBatchChangePlan creates a preview for batch plan change operation
func (s *AdminTenantService) PreviewBatchChangePlan(ctx context.Context, input BatchChangePlanInput) (*BatchPreviewDTO, error) {
	// Validate the new plan
	if err := s.validatePlan(input.NewPlan); err != nil {
		return nil, err
	}
	return s.previewBatchOperation(ctx, identity.BatchOperationTypeChangePlan, input.TenantIDs, &input.NewPlan)
}

// PreviewBatchDelete creates a preview for batch delete operation
func (s *AdminTenantService) PreviewBatchDelete(ctx context.Context, input BatchDeleteInput) (*BatchPreviewDTO, error) {
	return s.previewBatchOperation(ctx, identity.BatchOperationTypeDelete, input.TenantIDs, nil)
}

// previewBatchOperation creates a preview for any batch operation
func (s *AdminTenantService) previewBatchOperation(
	ctx context.Context,
	opType identity.BatchOperationType,
	tenantIDs []uuid.UUID,
	newPlan *string,
) (*BatchPreviewDTO, error) {
	s.logger.Info("Creating batch operation preview",
		zap.String("operation_type", string(opType)),
		zap.Int("tenant_count", len(tenantIDs)))

	// Validate batch size
	if len(tenantIDs) == 0 {
		return nil, shared.NewDomainError("EMPTY_BATCH", "At least one tenant must be specified")
	}
	if len(tenantIDs) > identity.MaxBatchSize {
		return nil, shared.NewDomainError("BATCH_SIZE_EXCEEDED",
			"Maximum batch size is 100 tenants")
	}

	// Deduplicate tenant IDs
	uniqueIDs := deduplicateUUIDs(tenantIDs)

	// Get tenant details
	tenants, err := s.adminRepo.FindByIDs(ctx, uniqueIDs)
	if err != nil {
		s.logger.Error("Failed to find tenants for batch preview", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to load tenant data")
	}

	// Build a map for quick lookup
	tenantMap := make(map[uuid.UUID]*identity.Tenant)
	for i := range tenants {
		tenantMap[tenants[i].ID] = &tenants[i]
	}

	// Check each tenant and build preview
	preview := &BatchPreviewDTO{
		OperationType:   string(opType),
		TotalCount:      len(uniqueIDs),
		AffectedTenants: make([]BatchPreviewTenantDTO, 0, len(uniqueIDs)),
		Warnings:        []string{},
	}

	for _, id := range uniqueIDs {
		tenant, found := tenantMap[id]
		if !found {
			preview.AffectedTenants = append(preview.AffectedTenants, BatchPreviewTenantDTO{
				ID:         id,
				CanProcess: false,
				SkipReason: "Tenant not found",
			})
			preview.WillSkipCount++
			continue
		}

		canProcess, skipReason := s.canProcessTenant(tenant, opType, newPlan)
		preview.AffectedTenants = append(preview.AffectedTenants, BatchPreviewTenantDTO{
			ID:            tenant.ID,
			Code:          tenant.Code,
			Name:          tenant.Name,
			CurrentStatus: string(tenant.Status),
			CurrentPlan:   string(tenant.Plan),
			CanProcess:    canProcess,
			SkipReason:    skipReason,
		})

		if canProcess {
			preview.CanProcessCount++
		} else {
			preview.WillSkipCount++
		}
	}

	// Add warnings
	if preview.WillSkipCount > 0 {
		preview.Warnings = append(preview.Warnings,
			"Some tenants will be skipped due to their current state")
	}

	return preview, nil
}

// canProcessTenant checks if a tenant can be processed for the given operation
func (s *AdminTenantService) canProcessTenant(
	tenant *identity.Tenant,
	opType identity.BatchOperationType,
	newPlan *string,
) (bool, string) {
	// System tenant cannot be modified
	if tenant.Code == identity.SystemTenantCode {
		return false, "System tenant cannot be modified"
	}

	switch opType {
	case identity.BatchOperationTypeSuspend:
		if tenant.Status == identity.TenantStatusSuspended {
			return false, "Already suspended"
		}
		if tenant.Status == identity.TenantStatusInactive {
			return false, "Cannot suspend inactive tenant"
		}
		return true, ""

	case identity.BatchOperationTypeActivate:
		if tenant.Status == identity.TenantStatusActive {
			return false, "Already active"
		}
		return true, ""

	case identity.BatchOperationTypeChangePlan:
		if newPlan == nil {
			return false, "New plan not specified"
		}
		currentPlan := identity.TenantPlan(*newPlan)
		if currentPlan.IsSamePlan(tenant.Plan) {
			return false, "Already on this plan"
		}
		return true, ""

	case identity.BatchOperationTypeDelete:
		if tenant.Status == identity.TenantStatusInactive {
			return false, "Already deleted"
		}
		return true, ""

	default:
		return false, "Unknown operation type"
	}
}

// BatchSuspend executes batch suspend operation synchronously
func (s *AdminTenantService) BatchSuspend(ctx context.Context, input BatchSuspendInput, auditCtx AuditContext) (*BatchOperationDTO, error) {
	s.logger.Info("Executing batch suspend",
		zap.Int("tenant_count", len(input.TenantIDs)),
		zap.String("admin_user_id", auditCtx.AdminUserID.String()))

	// Create batch operation
	payload := identity.BatchOperationPayload{
		SuspendReason:         input.Reason,
		ScheduledReactivateAt: input.ScheduledReactivateAt,
	}

	op, err := identity.NewBatchOperation(
		identity.BatchOperationTypeSuspend,
		deduplicateUUIDs(input.TenantIDs),
		payload,
		auditCtx.AdminUserID,
	)
	if err != nil {
		return nil, err
	}

	// Auto-confirm for immediate execution
	if err := op.Confirm(); err != nil {
		return nil, err
	}
	if err := op.StartProcessing(); err != nil {
		return nil, err
	}

	// Process each tenant
	tenants, err := s.adminRepo.FindByIDs(ctx, deduplicateUUIDs(input.TenantIDs))
	if err != nil {
		s.logger.Error("Failed to load tenants for batch suspend", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to load tenant data")
	}

	tenantMap := make(map[uuid.UUID]*identity.Tenant)
	for i := range tenants {
		tenantMap[tenants[i].ID] = &tenants[i]
		op.SetTenantInfo(tenants[i].ID, tenants[i].Code, tenants[i].Name)
	}

	for i := range op.Items {
		tenantID := op.Items[i].TenantID
		tenant, found := tenantMap[tenantID]

		if !found {
			op.MarkItemSkipped(tenantID, "Tenant not found")
			continue
		}

		canProcess, skipReason := s.canProcessTenant(tenant, identity.BatchOperationTypeSuspend, nil)
		if !canProcess {
			op.MarkItemSkipped(tenantID, skipReason)
			continue
		}

		// Execute suspend
		suspendInput := SuspendTenantInput{
			TenantID:              tenantID,
			Reason:                input.Reason,
			ScheduledReactivateAt: input.ScheduledReactivateAt,
		}

		_, err := s.SuspendTenantWithReason(ctx, suspendInput, auditCtx)
		if err != nil {
			s.logger.Warn("Failed to suspend tenant in batch",
				zap.String("tenant_id", tenantID.String()),
				zap.Error(err))
			op.MarkItemFailed(tenantID, err.Error())
		} else {
			op.MarkItemSuccess(tenantID)
		}
	}

	op.Complete()
	return toBatchOperationDTO(op), nil
}

// BatchActivate executes batch activate operation synchronously
func (s *AdminTenantService) BatchActivate(ctx context.Context, input BatchActivateInput, auditCtx AuditContext) (*BatchOperationDTO, error) {
	s.logger.Info("Executing batch activate",
		zap.Int("tenant_count", len(input.TenantIDs)),
		zap.String("admin_user_id", auditCtx.AdminUserID.String()))

	// Create batch operation
	op, err := identity.NewBatchOperation(
		identity.BatchOperationTypeActivate,
		deduplicateUUIDs(input.TenantIDs),
		identity.BatchOperationPayload{},
		auditCtx.AdminUserID,
	)
	if err != nil {
		return nil, err
	}

	// Auto-confirm for immediate execution
	if err := op.Confirm(); err != nil {
		return nil, err
	}
	if err := op.StartProcessing(); err != nil {
		return nil, err
	}

	// Process each tenant
	tenants, err := s.adminRepo.FindByIDs(ctx, deduplicateUUIDs(input.TenantIDs))
	if err != nil {
		s.logger.Error("Failed to load tenants for batch activate", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to load tenant data")
	}

	tenantMap := make(map[uuid.UUID]*identity.Tenant)
	for i := range tenants {
		tenantMap[tenants[i].ID] = &tenants[i]
		op.SetTenantInfo(tenants[i].ID, tenants[i].Code, tenants[i].Name)
	}

	for i := range op.Items {
		tenantID := op.Items[i].TenantID
		tenant, found := tenantMap[tenantID]

		if !found {
			op.MarkItemSkipped(tenantID, "Tenant not found")
			continue
		}

		canProcess, skipReason := s.canProcessTenant(tenant, identity.BatchOperationTypeActivate, nil)
		if !canProcess {
			op.MarkItemSkipped(tenantID, skipReason)
			continue
		}

		// Execute activate
		_, err := s.ActivateTenant(ctx, tenantID, auditCtx)
		if err != nil {
			s.logger.Warn("Failed to activate tenant in batch",
				zap.String("tenant_id", tenantID.String()),
				zap.Error(err))
			op.MarkItemFailed(tenantID, err.Error())
		} else {
			op.MarkItemSuccess(tenantID)
		}
	}

	op.Complete()
	return toBatchOperationDTO(op), nil
}

// BatchChangePlan executes batch plan change operation synchronously
func (s *AdminTenantService) BatchChangePlan(ctx context.Context, input BatchChangePlanInput, auditCtx AuditContext) (*BatchOperationDTO, error) {
	s.logger.Info("Executing batch change plan",
		zap.Int("tenant_count", len(input.TenantIDs)),
		zap.String("new_plan", input.NewPlan),
		zap.String("admin_user_id", auditCtx.AdminUserID.String()))

	// Validate the new plan
	if err := s.validatePlan(input.NewPlan); err != nil {
		return nil, err
	}

	// Create batch operation
	payload := identity.BatchOperationPayload{
		NewPlan:      input.NewPlan,
		ChangeReason: input.Reason,
	}

	op, err := identity.NewBatchOperation(
		identity.BatchOperationTypeChangePlan,
		deduplicateUUIDs(input.TenantIDs),
		payload,
		auditCtx.AdminUserID,
	)
	if err != nil {
		return nil, err
	}

	// Auto-confirm for immediate execution
	if err := op.Confirm(); err != nil {
		return nil, err
	}
	if err := op.StartProcessing(); err != nil {
		return nil, err
	}

	// Process each tenant
	tenants, err := s.adminRepo.FindByIDs(ctx, deduplicateUUIDs(input.TenantIDs))
	if err != nil {
		s.logger.Error("Failed to load tenants for batch change plan", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to load tenant data")
	}

	tenantMap := make(map[uuid.UUID]*identity.Tenant)
	for i := range tenants {
		tenantMap[tenants[i].ID] = &tenants[i]
		op.SetTenantInfo(tenants[i].ID, tenants[i].Code, tenants[i].Name)
	}

	for i := range op.Items {
		tenantID := op.Items[i].TenantID
		tenant, found := tenantMap[tenantID]

		if !found {
			op.MarkItemSkipped(tenantID, "Tenant not found")
			continue
		}

		canProcess, skipReason := s.canProcessTenant(tenant, identity.BatchOperationTypeChangePlan, &input.NewPlan)
		if !canProcess {
			op.MarkItemSkipped(tenantID, skipReason)
			continue
		}

		// Execute change plan
		changePlanInput := ChangePlanInput{
			TenantID: tenantID,
			NewPlan:  input.NewPlan,
			Reason:   input.Reason,
		}

		_, err := s.ChangePlan(ctx, changePlanInput, auditCtx)
		if err != nil {
			s.logger.Warn("Failed to change plan in batch",
				zap.String("tenant_id", tenantID.String()),
				zap.Error(err))
			op.MarkItemFailed(tenantID, err.Error())
		} else {
			op.MarkItemSuccess(tenantID)
		}
	}

	op.Complete()
	return toBatchOperationDTO(op), nil
}

// BatchDelete executes batch delete operation synchronously
func (s *AdminTenantService) BatchDelete(ctx context.Context, input BatchDeleteInput, auditCtx AuditContext) (*BatchOperationDTO, error) {
	s.logger.Info("Executing batch delete",
		zap.Int("tenant_count", len(input.TenantIDs)),
		zap.String("admin_user_id", auditCtx.AdminUserID.String()))

	// Create batch operation
	op, err := identity.NewBatchOperation(
		identity.BatchOperationTypeDelete,
		deduplicateUUIDs(input.TenantIDs),
		identity.BatchOperationPayload{},
		auditCtx.AdminUserID,
	)
	if err != nil {
		return nil, err
	}

	// Auto-confirm for immediate execution
	if err := op.Confirm(); err != nil {
		return nil, err
	}
	if err := op.StartProcessing(); err != nil {
		return nil, err
	}

	// Process each tenant
	tenants, err := s.adminRepo.FindByIDs(ctx, deduplicateUUIDs(input.TenantIDs))
	if err != nil {
		s.logger.Error("Failed to load tenants for batch delete", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to load tenant data")
	}

	tenantMap := make(map[uuid.UUID]*identity.Tenant)
	for i := range tenants {
		tenantMap[tenants[i].ID] = &tenants[i]
		op.SetTenantInfo(tenants[i].ID, tenants[i].Code, tenants[i].Name)
	}

	for i := range op.Items {
		tenantID := op.Items[i].TenantID
		tenant, found := tenantMap[tenantID]

		if !found {
			op.MarkItemSkipped(tenantID, "Tenant not found")
			continue
		}

		canProcess, skipReason := s.canProcessTenant(tenant, identity.BatchOperationTypeDelete, nil)
		if !canProcess {
			op.MarkItemSkipped(tenantID, skipReason)
			continue
		}

		// Execute delete
		err := s.DeleteTenant(ctx, tenantID, auditCtx)
		if err != nil {
			s.logger.Warn("Failed to delete tenant in batch",
				zap.String("tenant_id", tenantID.String()),
				zap.Error(err))
			op.MarkItemFailed(tenantID, err.Error())
		} else {
			op.MarkItemSuccess(tenantID)
		}
	}

	op.Complete()
	return toBatchOperationDTO(op), nil
}

// ============================================================================
// Helper Functions
// ============================================================================

// deduplicateUUIDs removes duplicate UUIDs from a slice
func deduplicateUUIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]bool)
	result := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	return result
}

// toBatchOperationDTO converts domain BatchOperation to DTO
func toBatchOperationDTO(op *identity.BatchOperation) *BatchOperationDTO {
	items := make([]BatchOperationItemDTO, len(op.Items))
	for i, item := range op.Items {
		items[i] = BatchOperationItemDTO{
			ID:           item.ID,
			TenantID:     item.TenantID,
			TenantCode:   item.TenantCode,
			TenantName:   item.TenantName,
			Status:       string(item.Status),
			ErrorMessage: item.ErrorMessage,
			ProcessedAt:  item.ProcessedAt,
		}
	}

	return &BatchOperationDTO{
		ID:              op.ID,
		OperationType:   string(op.OperationType),
		Status:          string(op.Status),
		TotalCount:      op.TotalCount,
		ProcessedCount:  op.ProcessedCount,
		SuccessCount:    op.SuccessCount,
		FailedCount:     op.FailedCount,
		SkippedCount:    op.SkippedCount,
		Progress:        op.GetProgress(),
		Items:           items,
		CreatedByUserID: op.CreatedByUserID,
		ConfirmedAt:     op.ConfirmedAt,
		CompletedAt:     op.CompletedAt,
		CreatedAt:       op.CreatedAt,
		UpdatedAt:       op.UpdatedAt,
	}
}
