package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/erp/backend/internal/domain/integration"
)

// ---------------------------------------------------------------------------
// OrderSyncExecutorImpl
// ---------------------------------------------------------------------------

// OrderSyncExecutorImpl implements OrderSyncExecutor interface
type OrderSyncExecutorImpl struct {
	platformRegistry integration.EcommercePlatformRegistry
	configProvider   OrderSyncConfigProvider
	logger           *zap.Logger

	// Callback handlers (optional, for extending functionality)
	onOrderPulled   func(ctx context.Context, tenantID uuid.UUID, order *integration.PlatformOrder) error
	onSyncCompleted func(ctx context.Context, job *OrderSyncJob) error
}

// NewOrderSyncExecutor creates a new order sync executor
func NewOrderSyncExecutor(
	platformRegistry integration.EcommercePlatformRegistry,
	configProvider OrderSyncConfigProvider,
	logger *zap.Logger,
) *OrderSyncExecutorImpl {
	return &OrderSyncExecutorImpl{
		platformRegistry: platformRegistry,
		configProvider:   configProvider,
		logger:           logger,
	}
}

// SetOnOrderPulledCallback sets the callback for when an order is pulled
func (e *OrderSyncExecutorImpl) SetOnOrderPulledCallback(cb func(ctx context.Context, tenantID uuid.UUID, order *integration.PlatformOrder) error) {
	e.onOrderPulled = cb
}

// SetOnSyncCompletedCallback sets the callback for when sync completes
func (e *OrderSyncExecutorImpl) SetOnSyncCompletedCallback(cb func(ctx context.Context, job *OrderSyncJob) error) {
	e.onSyncCompleted = cb
}

// Execute pulls orders from the platform and processes them
func (e *OrderSyncExecutorImpl) Execute(ctx context.Context, job *OrderSyncJob) error {
	// Get platform adapter
	platform, err := e.platformRegistry.GetPlatform(job.PlatformCode)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrOrderSyncPlatformUnavailable, err)
	}

	// Check if platform is enabled for tenant
	enabled, err := platform.IsEnabled(ctx, job.TenantID)
	if err != nil {
		return fmt.Errorf("failed to check platform status: %w", err)
	}
	if !enabled {
		return fmt.Errorf("%w: platform %s not enabled for tenant", ErrOrderSyncPlatformUnavailable, job.PlatformCode)
	}

	e.logger.Info("Starting order sync execution",
		zap.String("job_id", job.ID.String()),
		zap.String("tenant_id", job.TenantID.String()),
		zap.String("platform_code", string(job.PlatformCode)),
		zap.Time("start_time", job.StartTime),
		zap.Time("end_time", job.EndTime),
	)

	// Pull orders with pagination
	totalOrders := 0
	successCount := 0
	failedCount := 0
	skippedCount := 0
	failedOrderIDs := make([]string, 0)

	pageNo := 1
	pageSize := 50 // Reasonable page size for most platforms

	for {
		select {
		case <-ctx.Done():
			return ErrOrderSyncTimeout
		default:
		}

		// Build pull request
		req := &integration.OrderPullRequest{
			TenantID:     job.TenantID,
			PlatformCode: job.PlatformCode,
			StartTime:    job.StartTime,
			EndTime:      job.EndTime,
			PageNo:       pageNo,
			PageSize:     pageSize,
		}

		// Pull orders from platform
		resp, err := platform.PullOrders(ctx, req)
		if err != nil {
			// Check if it's a rate limit error
			if isRateLimitError(err) {
				e.logger.Warn("Rate limited by platform, will retry later",
					zap.String("platform_code", string(job.PlatformCode)),
					zap.Error(err),
				)
				return ErrOrderSyncRateLimited
			}

			// Check if it's a transient error
			if isTransientError(err) {
				return fmt.Errorf("%w: %v", ErrOrderSyncFailed, err)
			}

			// For permanent errors, log and continue if we have some results
			e.logger.Error("Failed to pull orders",
				zap.String("job_id", job.ID.String()),
				zap.Int("page_no", pageNo),
				zap.Error(err),
			)

			if totalOrders > 0 {
				// We have some results, mark as partial success
				break
			}
			return fmt.Errorf("%w: %v", ErrOrderSyncFailed, err)
		}

		totalOrders += len(resp.Orders)

		// Process each order
		for i := range resp.Orders {
			order := &resp.Orders[i]

			// Skip orders that don't need processing (e.g., already synced)
			if e.shouldSkipOrder(ctx, job.TenantID, order) {
				skippedCount++
				continue
			}

			// Process the order via callback if set
			if e.onOrderPulled != nil {
				if err := e.onOrderPulled(ctx, job.TenantID, order); err != nil {
					e.logger.Error("Failed to process order",
						zap.String("platform_order_id", order.PlatformOrderID),
						zap.Error(err),
					)
					failedCount++
					failedOrderIDs = append(failedOrderIDs, order.PlatformOrderID)
					continue
				}
			}

			successCount++
		}

		e.logger.Debug("Processed page of orders",
			zap.String("job_id", job.ID.String()),
			zap.Int("page_no", pageNo),
			zap.Int("orders_in_page", len(resp.Orders)),
			zap.Int("total_so_far", totalOrders),
		)

		// Check if there are more pages
		if !resp.HasMore || len(resp.Orders) == 0 {
			break
		}

		pageNo = resp.NextPageNo
	}

	// Update job with results
	job.Complete(totalOrders, successCount, failedCount, skippedCount)
	job.FailedOrderIDs = failedOrderIDs

	// Update last sync time
	if err := e.configProvider.UpdateLastSyncTime(ctx, job.TenantID, job.PlatformCode, time.Now()); err != nil {
		e.logger.Warn("Failed to update last sync time",
			zap.String("tenant_id", job.TenantID.String()),
			zap.String("platform_code", string(job.PlatformCode)),
			zap.Error(err),
		)
		// Don't fail the job for this
	}

	// Call completion callback if set
	if e.onSyncCompleted != nil {
		if err := e.onSyncCompleted(ctx, job); err != nil {
			e.logger.Warn("Sync completed callback failed",
				zap.String("job_id", job.ID.String()),
				zap.Error(err),
			)
		}
	}

	e.logger.Info("Order sync execution completed",
		zap.String("job_id", job.ID.String()),
		zap.String("status", string(job.Status)),
		zap.Int("total_orders", totalOrders),
		zap.Int("success_count", successCount),
		zap.Int("failed_count", failedCount),
		zap.Int("skipped_count", skippedCount),
	)

	return nil
}

// shouldSkipOrder determines if an order should be skipped
// This can be extended to check if order was already synced
// Parameters ctx and tenantID are reserved for future use (e.g., checking sync records)
func (e *OrderSyncExecutorImpl) shouldSkipOrder(_ context.Context, _ uuid.UUID, order *integration.PlatformOrder) bool {
	// Skip cancelled orders by default
	if order.Status == integration.PlatformOrderStatusCancelled {
		return true
	}

	// Skip closed orders by default
	if order.Status == integration.PlatformOrderStatusClosed {
		return true
	}

	return false
}

// isRateLimitError checks if the error is a rate limit error
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	// Check for known rate limit errors from platform adapters
	return err == integration.ErrPlatformRateLimited
}

// isTransientError checks if the error is transient (retryable)
func isTransientError(err error) bool {
	if err == nil {
		return false
	}
	// Platform unavailable is usually transient
	if err == integration.ErrPlatformUnavailable {
		return true
	}
	// Request failed might be transient
	if err == integration.ErrPlatformRequestFailed {
		return true
	}
	return false
}

// Ensure OrderSyncExecutorImpl implements OrderSyncExecutor
var _ OrderSyncExecutor = (*OrderSyncExecutorImpl)(nil)
