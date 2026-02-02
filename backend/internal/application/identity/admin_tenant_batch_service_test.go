package identity

import (
	"context"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Batch Operation Domain Tests
// ============================================================================

func TestNewBatchOperation(t *testing.T) {
	t.Run("creates batch operation successfully", func(t *testing.T) {
		tenantIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
		payload := identity.BatchOperationPayload{
			SuspendReason: "Test reason",
		}
		userID := uuid.New()

		op, err := identity.NewBatchOperation(
			identity.BatchOperationTypeSuspend,
			tenantIDs,
			payload,
			userID,
		)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, op.ID)
		assert.Equal(t, identity.BatchOperationTypeSuspend, op.OperationType)
		assert.Equal(t, identity.BatchOperationStatusPending, op.Status)
		assert.Equal(t, 3, op.TotalCount)
		assert.Equal(t, 0, op.ProcessedCount)
		assert.Equal(t, 0, op.SuccessCount)
		assert.Equal(t, 0, op.FailedCount)
		assert.Equal(t, 0, op.SkippedCount)
		assert.Len(t, op.Items, 3)
		assert.Equal(t, userID, op.CreatedByUserID)
	})

	t.Run("fails with empty tenant list", func(t *testing.T) {
		op, err := identity.NewBatchOperation(
			identity.BatchOperationTypeSuspend,
			[]uuid.UUID{},
			identity.BatchOperationPayload{},
			uuid.New(),
		)

		assert.Error(t, err)
		assert.Nil(t, op)
		assert.Contains(t, err.Error(), "At least one tenant")
	})

	t.Run("fails when batch size exceeds limit", func(t *testing.T) {
		// Create 101 tenant IDs (over MaxBatchSize of 100)
		tenantIDs := make([]uuid.UUID, 101)
		for i := range tenantIDs {
			tenantIDs[i] = uuid.New()
		}

		op, err := identity.NewBatchOperation(
			identity.BatchOperationTypeSuspend,
			tenantIDs,
			identity.BatchOperationPayload{},
			uuid.New(),
		)

		assert.Error(t, err)
		assert.Nil(t, op)
		assert.Contains(t, err.Error(), "Maximum batch size")
	})
}

func TestBatchOperation_Confirm(t *testing.T) {
	t.Run("confirms pending operation", func(t *testing.T) {
		op, _ := identity.NewBatchOperation(
			identity.BatchOperationTypeSuspend,
			[]uuid.UUID{uuid.New()},
			identity.BatchOperationPayload{},
			uuid.New(),
		)

		err := op.Confirm()

		require.NoError(t, err)
		assert.Equal(t, identity.BatchOperationStatusConfirmed, op.Status)
		assert.NotNil(t, op.ConfirmedAt)
	})

	t.Run("fails when already confirmed", func(t *testing.T) {
		op, _ := identity.NewBatchOperation(
			identity.BatchOperationTypeSuspend,
			[]uuid.UUID{uuid.New()},
			identity.BatchOperationPayload{},
			uuid.New(),
		)
		op.Confirm()

		err := op.Confirm()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "pending to confirm")
	})
}

func TestBatchOperation_StartProcessing(t *testing.T) {
	t.Run("starts processing confirmed operation", func(t *testing.T) {
		op, _ := identity.NewBatchOperation(
			identity.BatchOperationTypeSuspend,
			[]uuid.UUID{uuid.New()},
			identity.BatchOperationPayload{},
			uuid.New(),
		)
		op.Confirm()

		err := op.StartProcessing()

		require.NoError(t, err)
		assert.Equal(t, identity.BatchOperationStatusProcessing, op.Status)
	})

	t.Run("fails when not confirmed", func(t *testing.T) {
		op, _ := identity.NewBatchOperation(
			identity.BatchOperationTypeSuspend,
			[]uuid.UUID{uuid.New()},
			identity.BatchOperationPayload{},
			uuid.New(),
		)

		err := op.StartProcessing()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "confirmed to start")
	})
}

func TestBatchOperation_MarkItemStatus(t *testing.T) {
	t.Run("marks item as success", func(t *testing.T) {
		tenantID := uuid.New()
		op, _ := identity.NewBatchOperation(
			identity.BatchOperationTypeSuspend,
			[]uuid.UUID{tenantID},
			identity.BatchOperationPayload{},
			uuid.New(),
		)

		op.MarkItemSuccess(tenantID)

		assert.Equal(t, 1, op.ProcessedCount)
		assert.Equal(t, 1, op.SuccessCount)
		assert.Equal(t, 0, op.FailedCount)
		assert.Equal(t, identity.BatchItemStatusSuccess, op.Items[0].Status)
	})

	t.Run("marks item as failed", func(t *testing.T) {
		tenantID := uuid.New()
		op, _ := identity.NewBatchOperation(
			identity.BatchOperationTypeSuspend,
			[]uuid.UUID{tenantID},
			identity.BatchOperationPayload{},
			uuid.New(),
		)

		op.MarkItemFailed(tenantID, "Some error")

		assert.Equal(t, 1, op.ProcessedCount)
		assert.Equal(t, 0, op.SuccessCount)
		assert.Equal(t, 1, op.FailedCount)
		assert.Equal(t, identity.BatchItemStatusFailed, op.Items[0].Status)
		assert.Equal(t, "Some error", op.Items[0].ErrorMessage)
	})

	t.Run("marks item as skipped", func(t *testing.T) {
		tenantID := uuid.New()
		op, _ := identity.NewBatchOperation(
			identity.BatchOperationTypeSuspend,
			[]uuid.UUID{tenantID},
			identity.BatchOperationPayload{},
			uuid.New(),
		)

		op.MarkItemSkipped(tenantID, "Already suspended")

		assert.Equal(t, 1, op.ProcessedCount)
		assert.Equal(t, 0, op.SuccessCount)
		assert.Equal(t, 0, op.FailedCount)
		assert.Equal(t, 1, op.SkippedCount)
		assert.Equal(t, identity.BatchItemStatusSkipped, op.Items[0].Status)
	})
}

func TestBatchOperation_Complete(t *testing.T) {
	t.Run("marks as completed when success count > 0", func(t *testing.T) {
		tenantID := uuid.New()
		op, _ := identity.NewBatchOperation(
			identity.BatchOperationTypeSuspend,
			[]uuid.UUID{tenantID},
			identity.BatchOperationPayload{},
			uuid.New(),
		)
		op.Confirm()
		op.StartProcessing()
		op.MarkItemSuccess(tenantID)

		op.Complete()

		assert.Equal(t, identity.BatchOperationStatusCompleted, op.Status)
		assert.NotNil(t, op.CompletedAt)
	})

	t.Run("marks as failed when only failures", func(t *testing.T) {
		tenantID := uuid.New()
		op, _ := identity.NewBatchOperation(
			identity.BatchOperationTypeSuspend,
			[]uuid.UUID{tenantID},
			identity.BatchOperationPayload{},
			uuid.New(),
		)
		op.Confirm()
		op.StartProcessing()
		op.MarkItemFailed(tenantID, "Error")

		op.Complete()

		assert.Equal(t, identity.BatchOperationStatusFailed, op.Status)
	})
}

func TestBatchOperation_Cancel(t *testing.T) {
	t.Run("cancels pending operation", func(t *testing.T) {
		op, _ := identity.NewBatchOperation(
			identity.BatchOperationTypeSuspend,
			[]uuid.UUID{uuid.New()},
			identity.BatchOperationPayload{},
			uuid.New(),
		)

		err := op.Cancel()

		require.NoError(t, err)
		assert.Equal(t, identity.BatchOperationStatusCancelled, op.Status)
	})

	t.Run("fails when processing", func(t *testing.T) {
		op, _ := identity.NewBatchOperation(
			identity.BatchOperationTypeSuspend,
			[]uuid.UUID{uuid.New()},
			identity.BatchOperationPayload{},
			uuid.New(),
		)
		op.Confirm()
		op.StartProcessing()

		err := op.Cancel()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot cancel")
	})
}

func TestBatchOperation_GetProgress(t *testing.T) {
	t.Run("returns 0% when not started", func(t *testing.T) {
		op, _ := identity.NewBatchOperation(
			identity.BatchOperationTypeSuspend,
			[]uuid.UUID{uuid.New(), uuid.New()},
			identity.BatchOperationPayload{},
			uuid.New(),
		)

		assert.Equal(t, 0.0, op.GetProgress())
	})

	t.Run("returns 50% when half processed", func(t *testing.T) {
		tenantID1 := uuid.New()
		tenantID2 := uuid.New()
		op, _ := identity.NewBatchOperation(
			identity.BatchOperationTypeSuspend,
			[]uuid.UUID{tenantID1, tenantID2},
			identity.BatchOperationPayload{},
			uuid.New(),
		)
		op.MarkItemSuccess(tenantID1)

		assert.Equal(t, 50.0, op.GetProgress())
	})

	t.Run("returns 100% when all processed", func(t *testing.T) {
		tenantID := uuid.New()
		op, _ := identity.NewBatchOperation(
			identity.BatchOperationTypeSuspend,
			[]uuid.UUID{tenantID},
			identity.BatchOperationPayload{},
			uuid.New(),
		)
		op.MarkItemSuccess(tenantID)

		assert.Equal(t, 100.0, op.GetProgress())
	})
}

// ============================================================================
// Batch Operation Service Tests
// ============================================================================

func TestAdminTenantService_PreviewBatchSuspend(t *testing.T) {
	t.Run("creates preview for valid tenants", func(t *testing.T) {
		service, adminRepo, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()

		tenant1 := createTestTenant("TENANT1", "Active Tenant", identity.TenantPlanPro)
		tenant2 := createTestTenant("TENANT2", "Another Tenant", identity.TenantPlanBasic)

		adminRepo.On("FindByIDs", ctx, mock.AnythingOfType("[]uuid.UUID")).
			Return([]identity.Tenant{*tenant1, *tenant2}, nil)

		input := BatchSuspendInput{
			TenantIDs: []uuid.UUID{tenant1.ID, tenant2.ID},
			Reason:    "Test reason",
		}

		result, err := service.PreviewBatchSuspend(ctx, input)

		require.NoError(t, err)
		assert.Equal(t, "suspend", result.OperationType)
		assert.Equal(t, 2, result.TotalCount)
		assert.Equal(t, 2, result.CanProcessCount)
		assert.Equal(t, 0, result.WillSkipCount)
		assert.Len(t, result.AffectedTenants, 2)
		adminRepo.AssertExpectations(t)
	})

	t.Run("skips already suspended tenants", func(t *testing.T) {
		service, adminRepo, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()

		tenant := createTestTenant("TENANT1", "Suspended Tenant", identity.TenantPlanPro)
		tenant.SuspendWithReason("Already suspended", nil)

		adminRepo.On("FindByIDs", ctx, mock.AnythingOfType("[]uuid.UUID")).
			Return([]identity.Tenant{*tenant}, nil)

		input := BatchSuspendInput{
			TenantIDs: []uuid.UUID{tenant.ID},
		}

		result, err := service.PreviewBatchSuspend(ctx, input)

		require.NoError(t, err)
		assert.Equal(t, 1, result.WillSkipCount)
		assert.Equal(t, 0, result.CanProcessCount)
		assert.Equal(t, "Already suspended", result.AffectedTenants[0].SkipReason)
	})

	t.Run("fails with empty batch", func(t *testing.T) {
		service, _, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()

		input := BatchSuspendInput{
			TenantIDs: []uuid.UUID{},
		}

		result, err := service.PreviewBatchSuspend(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestAdminTenantService_PreviewBatchActivate(t *testing.T) {
	t.Run("creates preview for suspended tenants", func(t *testing.T) {
		service, adminRepo, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()

		tenant := createTestTenant("TENANT1", "Suspended Tenant", identity.TenantPlanPro)
		tenant.SuspendWithReason("Test", nil)

		adminRepo.On("FindByIDs", ctx, mock.AnythingOfType("[]uuid.UUID")).
			Return([]identity.Tenant{*tenant}, nil)

		input := BatchActivateInput{
			TenantIDs: []uuid.UUID{tenant.ID},
		}

		result, err := service.PreviewBatchActivate(ctx, input)

		require.NoError(t, err)
		assert.Equal(t, "activate", result.OperationType)
		assert.Equal(t, 1, result.CanProcessCount)
	})

	t.Run("skips already active tenants", func(t *testing.T) {
		service, adminRepo, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()

		tenant := createTestTenant("TENANT1", "Active Tenant", identity.TenantPlanPro)

		adminRepo.On("FindByIDs", ctx, mock.AnythingOfType("[]uuid.UUID")).
			Return([]identity.Tenant{*tenant}, nil)

		input := BatchActivateInput{
			TenantIDs: []uuid.UUID{tenant.ID},
		}

		result, err := service.PreviewBatchActivate(ctx, input)

		require.NoError(t, err)
		assert.Equal(t, 1, result.WillSkipCount)
		assert.Equal(t, "Already active", result.AffectedTenants[0].SkipReason)
	})
}

func TestAdminTenantService_PreviewBatchChangePlan(t *testing.T) {
	t.Run("creates preview for plan change", func(t *testing.T) {
		service, adminRepo, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()

		tenant := createTestTenant("TENANT1", "Basic Tenant", identity.TenantPlanBasic)

		adminRepo.On("FindByIDs", ctx, mock.AnythingOfType("[]uuid.UUID")).
			Return([]identity.Tenant{*tenant}, nil)

		input := BatchChangePlanInput{
			TenantIDs: []uuid.UUID{tenant.ID},
			NewPlan:   "pro",
		}

		result, err := service.PreviewBatchChangePlan(ctx, input)

		require.NoError(t, err)
		assert.Equal(t, "change_plan", result.OperationType)
		assert.Equal(t, 1, result.CanProcessCount)
	})

	t.Run("skips tenants already on same plan", func(t *testing.T) {
		service, adminRepo, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()

		tenant := createTestTenant("TENANT1", "Pro Tenant", identity.TenantPlanPro)

		adminRepo.On("FindByIDs", ctx, mock.AnythingOfType("[]uuid.UUID")).
			Return([]identity.Tenant{*tenant}, nil)

		input := BatchChangePlanInput{
			TenantIDs: []uuid.UUID{tenant.ID},
			NewPlan:   "pro",
		}

		result, err := service.PreviewBatchChangePlan(ctx, input)

		require.NoError(t, err)
		assert.Equal(t, 1, result.WillSkipCount)
		assert.Equal(t, "Already on this plan", result.AffectedTenants[0].SkipReason)
	})

	t.Run("fails with invalid plan", func(t *testing.T) {
		service, _, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()

		input := BatchChangePlanInput{
			TenantIDs: []uuid.UUID{uuid.New()},
			NewPlan:   "invalid",
		}

		result, err := service.PreviewBatchChangePlan(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestAdminTenantService_PreviewBatchDelete(t *testing.T) {
	t.Run("creates preview for delete", func(t *testing.T) {
		service, adminRepo, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()

		tenant := createTestTenant("TENANT1", "Active Tenant", identity.TenantPlanPro)

		adminRepo.On("FindByIDs", ctx, mock.AnythingOfType("[]uuid.UUID")).
			Return([]identity.Tenant{*tenant}, nil)

		input := BatchDeleteInput{
			TenantIDs: []uuid.UUID{tenant.ID},
		}

		result, err := service.PreviewBatchDelete(ctx, input)

		require.NoError(t, err)
		assert.Equal(t, "delete", result.OperationType)
		assert.Equal(t, 1, result.CanProcessCount)
	})

	t.Run("skips already deleted tenants", func(t *testing.T) {
		service, adminRepo, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()

		tenant := createTestTenant("TENANT1", "Inactive Tenant", identity.TenantPlanPro)
		tenant.Deactivate()

		adminRepo.On("FindByIDs", ctx, mock.AnythingOfType("[]uuid.UUID")).
			Return([]identity.Tenant{*tenant}, nil)

		input := BatchDeleteInput{
			TenantIDs: []uuid.UUID{tenant.ID},
		}

		result, err := service.PreviewBatchDelete(ctx, input)

		require.NoError(t, err)
		assert.Equal(t, 1, result.WillSkipCount)
		assert.Equal(t, "Already deleted", result.AffectedTenants[0].SkipReason)
	})

	t.Run("skips system tenant", func(t *testing.T) {
		service, adminRepo, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()

		systemTenant := createTestTenant(identity.SystemTenantCode, "System", identity.TenantPlanEnterprise)

		adminRepo.On("FindByIDs", ctx, mock.AnythingOfType("[]uuid.UUID")).
			Return([]identity.Tenant{*systemTenant}, nil)

		input := BatchDeleteInput{
			TenantIDs: []uuid.UUID{systemTenant.ID},
		}

		result, err := service.PreviewBatchDelete(ctx, input)

		require.NoError(t, err)
		assert.Equal(t, 1, result.WillSkipCount)
		assert.Contains(t, result.AffectedTenants[0].SkipReason, "System tenant")
	})
}

func TestAdminTenantService_BatchSuspend(t *testing.T) {
	t.Run("suspends multiple tenants", func(t *testing.T) {
		service, adminRepo, tenantRepo, _, statusHistoryRepo, auditLogRepo := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		tenant1 := createTestTenant("TENANT1", "Tenant One", identity.TenantPlanPro)
		tenant2 := createTestTenant("TENANT2", "Tenant Two", identity.TenantPlanBasic)

		// Mock FindByIDs
		adminRepo.On("FindByIDs", ctx, mock.AnythingOfType("[]uuid.UUID")).
			Return([]identity.Tenant{*tenant1, *tenant2}, nil)

		// Mock individual suspend operations (these are called internally)
		adminRepo.On("FindByID", ctx, tenant1.ID).Return(tenant1, nil)
		adminRepo.On("FindByID", ctx, tenant2.ID).Return(tenant2, nil)
		tenantRepo.On("Save", ctx, mock.AnythingOfType("*identity.Tenant")).Return(nil)
		statusHistoryRepo.On("Create", ctx, mock.AnythingOfType("*identity.TenantStatusHistory")).Return(nil)
		auditLogRepo.On("Create", ctx, mock.AnythingOfType("*identity.AuditLog")).Return(nil)

		input := BatchSuspendInput{
			TenantIDs: []uuid.UUID{tenant1.ID, tenant2.ID},
			Reason:    "Batch test",
		}

		result, err := service.BatchSuspend(ctx, input, auditCtx)

		require.NoError(t, err)
		assert.Equal(t, "suspend", result.OperationType)
		assert.Equal(t, "completed", result.Status)
		assert.Equal(t, 2, result.TotalCount)
		assert.Equal(t, 2, result.ProcessedCount)
		assert.Equal(t, 2, result.SuccessCount)
		assert.Equal(t, 0, result.FailedCount)
		assert.Equal(t, 100.0, result.Progress)
	})

	t.Run("handles partial failures", func(t *testing.T) {
		service, adminRepo, tenantRepo, _, statusHistoryRepo, auditLogRepo := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		tenant1 := createTestTenant("TENANT1", "Tenant One", identity.TenantPlanPro)
		tenant2 := createTestTenant("TENANT2", "Already Suspended", identity.TenantPlanBasic)
		tenant2.SuspendWithReason("Previous", nil)

		adminRepo.On("FindByIDs", ctx, mock.AnythingOfType("[]uuid.UUID")).
			Return([]identity.Tenant{*tenant1, *tenant2}, nil)
		adminRepo.On("FindByID", ctx, tenant1.ID).Return(tenant1, nil)
		tenantRepo.On("Save", ctx, mock.AnythingOfType("*identity.Tenant")).Return(nil)
		statusHistoryRepo.On("Create", ctx, mock.AnythingOfType("*identity.TenantStatusHistory")).Return(nil)
		auditLogRepo.On("Create", ctx, mock.AnythingOfType("*identity.AuditLog")).Return(nil)

		input := BatchSuspendInput{
			TenantIDs: []uuid.UUID{tenant1.ID, tenant2.ID},
			Reason:    "Batch test",
		}

		result, err := service.BatchSuspend(ctx, input, auditCtx)

		require.NoError(t, err)
		assert.Equal(t, "completed", result.Status)
		assert.Equal(t, 1, result.SuccessCount)
		assert.Equal(t, 1, result.SkippedCount)
	})
}

func TestAdminTenantService_BatchActivate(t *testing.T) {
	t.Run("activates multiple tenants", func(t *testing.T) {
		service, adminRepo, tenantRepo, _, statusHistoryRepo, auditLogRepo := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		tenant := createTestTenant("TENANT1", "Suspended Tenant", identity.TenantPlanPro)
		tenant.SuspendWithReason("Test", nil)

		adminRepo.On("FindByIDs", ctx, mock.AnythingOfType("[]uuid.UUID")).
			Return([]identity.Tenant{*tenant}, nil)
		adminRepo.On("FindByID", ctx, tenant.ID).Return(tenant, nil)
		tenantRepo.On("Save", ctx, mock.AnythingOfType("*identity.Tenant")).Return(nil)
		statusHistoryRepo.On("Create", ctx, mock.AnythingOfType("*identity.TenantStatusHistory")).Return(nil)
		auditLogRepo.On("Create", ctx, mock.AnythingOfType("*identity.AuditLog")).Return(nil)

		input := BatchActivateInput{
			TenantIDs: []uuid.UUID{tenant.ID},
		}

		result, err := service.BatchActivate(ctx, input, auditCtx)

		require.NoError(t, err)
		assert.Equal(t, "activate", result.OperationType)
		assert.Equal(t, "completed", result.Status)
		assert.Equal(t, 1, result.SuccessCount)
	})
}

func TestAdminTenantService_BatchChangePlan(t *testing.T) {
	t.Run("changes plan for multiple tenants", func(t *testing.T) {
		service, adminRepo, tenantRepo, subHistoryRepo, _, auditLogRepo := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		tenant := createTestTenant("TENANT1", "Basic Tenant", identity.TenantPlanBasic)

		adminRepo.On("FindByIDs", ctx, mock.AnythingOfType("[]uuid.UUID")).
			Return([]identity.Tenant{*tenant}, nil)
		adminRepo.On("FindByID", ctx, tenant.ID).Return(tenant, nil)
		tenantRepo.On("Save", ctx, mock.AnythingOfType("*identity.Tenant")).Return(nil)
		subHistoryRepo.On("Create", ctx, mock.AnythingOfType("*identity.SubscriptionHistory")).Return(nil)
		auditLogRepo.On("Create", ctx, mock.AnythingOfType("*identity.AuditLog")).Return(nil)

		input := BatchChangePlanInput{
			TenantIDs: []uuid.UUID{tenant.ID},
			NewPlan:   "pro",
			Reason:    "Batch upgrade",
		}

		result, err := service.BatchChangePlan(ctx, input, auditCtx)

		require.NoError(t, err)
		assert.Equal(t, "change_plan", result.OperationType)
		assert.Equal(t, "completed", result.Status)
		assert.Equal(t, 1, result.SuccessCount)
	})

	t.Run("fails with invalid plan", func(t *testing.T) {
		service, _, _, _, _, _ := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		input := BatchChangePlanInput{
			TenantIDs: []uuid.UUID{uuid.New()},
			NewPlan:   "invalid_plan",
		}

		result, err := service.BatchChangePlan(ctx, input, auditCtx)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestAdminTenantService_BatchDelete(t *testing.T) {
	t.Run("deletes multiple tenants", func(t *testing.T) {
		service, adminRepo, tenantRepo, _, _, auditLogRepo := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		tenant := createTestTenant("TENANT1", "Active Tenant", identity.TenantPlanPro)

		adminRepo.On("FindByIDs", ctx, mock.AnythingOfType("[]uuid.UUID")).
			Return([]identity.Tenant{*tenant}, nil)
		adminRepo.On("FindByID", ctx, tenant.ID).Return(tenant, nil)
		tenantRepo.On("Save", ctx, mock.MatchedBy(func(t *identity.Tenant) bool {
			return t.Status == identity.TenantStatusInactive
		})).Return(nil)
		auditLogRepo.On("Create", ctx, mock.AnythingOfType("*identity.AuditLog")).Return(nil)

		input := BatchDeleteInput{
			TenantIDs: []uuid.UUID{tenant.ID},
		}

		result, err := service.BatchDelete(ctx, input, auditCtx)

		require.NoError(t, err)
		assert.Equal(t, "delete", result.OperationType)
		assert.Equal(t, "completed", result.Status)
		assert.Equal(t, 1, result.SuccessCount)
	})
}

func TestDeduplicateUUIDs(t *testing.T) {
	t.Run("removes duplicates", func(t *testing.T) {
		id1 := uuid.New()
		id2 := uuid.New()

		input := []uuid.UUID{id1, id2, id1, id2, id1}
		result := deduplicateUUIDs(input)

		assert.Len(t, result, 2)
		assert.Contains(t, result, id1)
		assert.Contains(t, result, id2)
	})

	t.Run("handles empty slice", func(t *testing.T) {
		result := deduplicateUUIDs([]uuid.UUID{})
		assert.Empty(t, result)
	})

	t.Run("handles single element", func(t *testing.T) {
		id := uuid.New()
		result := deduplicateUUIDs([]uuid.UUID{id})
		assert.Len(t, result, 1)
		assert.Equal(t, id, result[0])
	})
}

func TestToBatchOperationDTO(t *testing.T) {
	t.Run("converts batch operation to DTO", func(t *testing.T) {
		tenantID := uuid.New()
		op, _ := identity.NewBatchOperation(
			identity.BatchOperationTypeSuspend,
			[]uuid.UUID{tenantID},
			identity.BatchOperationPayload{SuspendReason: "Test"},
			uuid.New(),
		)
		op.SetTenantInfo(tenantID, "TENANT1", "Tenant One")
		op.Confirm()
		op.StartProcessing()
		op.MarkItemSuccess(tenantID)
		op.Complete()

		dto := toBatchOperationDTO(op)

		assert.Equal(t, op.ID, dto.ID)
		assert.Equal(t, "suspend", dto.OperationType)
		assert.Equal(t, "completed", dto.Status)
		assert.Equal(t, 1, dto.TotalCount)
		assert.Equal(t, 1, dto.ProcessedCount)
		assert.Equal(t, 1, dto.SuccessCount)
		assert.Equal(t, 100.0, dto.Progress)
		assert.NotNil(t, dto.ConfirmedAt)
		assert.NotNil(t, dto.CompletedAt)
		assert.Len(t, dto.Items, 1)
		assert.Equal(t, "TENANT1", dto.Items[0].TenantCode)
		assert.Equal(t, "Tenant One", dto.Items[0].TenantName)
		assert.Equal(t, "success", dto.Items[0].Status)
	})
}

// Helper for batch tests
func TestBatchPreviewDTO(t *testing.T) {
	t.Run("validates preview structure", func(t *testing.T) {
		preview := BatchPreviewDTO{
			OperationType:   "suspend",
			TotalCount:      5,
			CanProcessCount: 3,
			WillSkipCount:   2,
			AffectedTenants: []BatchPreviewTenantDTO{
				{
					ID:            uuid.New(),
					Code:          "T1",
					Name:          "Tenant 1",
					CurrentStatus: "active",
					CanProcess:    true,
				},
				{
					ID:            uuid.New(),
					Code:          "T2",
					Name:          "Tenant 2",
					CurrentStatus: "suspended",
					CanProcess:    false,
					SkipReason:    "Already suspended",
				},
			},
			Warnings: []string{"Some tenants will be skipped"},
		}

		assert.Equal(t, "suspend", preview.OperationType)
		assert.Equal(t, 5, preview.TotalCount)
		assert.Equal(t, 3, preview.CanProcessCount)
		assert.Equal(t, 2, preview.WillSkipCount)
		assert.Len(t, preview.AffectedTenants, 2)
		assert.True(t, preview.AffectedTenants[0].CanProcess)
		assert.False(t, preview.AffectedTenants[1].CanProcess)
		assert.NotEmpty(t, preview.Warnings)
	})
}

// Scheduled reactivation tests
func TestAdminTenantService_BatchSuspendWithScheduledReactivation(t *testing.T) {
	t.Run("suspends with scheduled reactivation", func(t *testing.T) {
		service, adminRepo, tenantRepo, _, statusHistoryRepo, auditLogRepo := newTestAdminTenantService()
		ctx := context.Background()
		auditCtx := createTestAuditContext()

		tenant := createTestTenant("TENANT1", "Tenant One", identity.TenantPlanPro)
		reactivateAt := time.Now().Add(24 * time.Hour)

		adminRepo.On("FindByIDs", ctx, mock.AnythingOfType("[]uuid.UUID")).
			Return([]identity.Tenant{*tenant}, nil)
		adminRepo.On("FindByID", ctx, tenant.ID).Return(tenant, nil)
		tenantRepo.On("Save", ctx, mock.AnythingOfType("*identity.Tenant")).Return(nil)
		statusHistoryRepo.On("Create", ctx, mock.AnythingOfType("*identity.TenantStatusHistory")).Return(nil)
		auditLogRepo.On("Create", ctx, mock.AnythingOfType("*identity.AuditLog")).Return(nil)

		input := BatchSuspendInput{
			TenantIDs:             []uuid.UUID{tenant.ID},
			Reason:                "Temporary suspension",
			ScheduledReactivateAt: &reactivateAt,
		}

		result, err := service.BatchSuspend(ctx, input, auditCtx)

		require.NoError(t, err)
		assert.Equal(t, 1, result.SuccessCount)
	})
}
