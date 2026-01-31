package billing

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/billing"
	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ResourceCounter defines the interface for counting tenant resources
type ResourceCounter interface {
	CountUsers(ctx context.Context, tenantID uuid.UUID) (int64, error)
	CountProducts(ctx context.Context, tenantID uuid.UUID) (int64, error)
	CountWarehouses(ctx context.Context, tenantID uuid.UUID) (int64, error)
	CountCustomers(ctx context.Context, tenantID uuid.UUID) (int64, error)
	CountSuppliers(ctx context.Context, tenantID uuid.UUID) (int64, error)
	CountOrders(ctx context.Context, tenantID uuid.UUID) (int64, error)
}

// UsageSnapshotService handles daily usage snapshot creation and data retention
type UsageSnapshotService struct {
	historyRepo     billing.UsageHistoryRepository
	tenantRepo      identity.TenantRepository
	resourceCounter ResourceCounter
	logger          *zap.Logger

	// Configuration
	retentionDays int // Number of days to retain history (default 90)
}

// UsageSnapshotServiceConfig contains configuration for UsageSnapshotService
type UsageSnapshotServiceConfig struct {
	RetentionDays int
}

// DefaultUsageSnapshotServiceConfig returns default configuration
func DefaultUsageSnapshotServiceConfig() UsageSnapshotServiceConfig {
	return UsageSnapshotServiceConfig{
		RetentionDays: 90,
	}
}

// NewUsageSnapshotService creates a new UsageSnapshotService
func NewUsageSnapshotService(
	historyRepo billing.UsageHistoryRepository,
	tenantRepo identity.TenantRepository,
	resourceCounter ResourceCounter,
	logger *zap.Logger,
	config UsageSnapshotServiceConfig,
) *UsageSnapshotService {
	if config.RetentionDays <= 0 {
		config.RetentionDays = 90
	}

	return &UsageSnapshotService{
		historyRepo:     historyRepo,
		tenantRepo:      tenantRepo,
		resourceCounter: resourceCounter,
		logger:          logger,
		retentionDays:   config.RetentionDays,
	}
}

// CreateSnapshotForTenant creates a usage snapshot for a specific tenant
func (s *UsageSnapshotService) CreateSnapshotForTenant(ctx context.Context, tenantID uuid.UUID, snapshotDate time.Time) (*billing.UsageHistory, error) {
	if tenantID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_TENANT", "Tenant ID cannot be empty")
	}

	s.logger.Debug("Creating usage snapshot",
		zap.String("tenant_id", tenantID.String()),
		zap.Time("snapshot_date", snapshotDate))

	// Create new history record
	history, err := billing.NewUsageHistory(tenantID, snapshotDate)
	if err != nil {
		return nil, err
	}

	// Collect all usage counts
	if s.resourceCounter != nil {
		// Count users
		if userCount, err := s.resourceCounter.CountUsers(ctx, tenantID); err == nil {
			history.WithUsersCount(userCount)
		} else {
			s.logger.Warn("Failed to count users", zap.String("tenant_id", tenantID.String()), zap.Error(err))
		}

		// Count products
		if productCount, err := s.resourceCounter.CountProducts(ctx, tenantID); err == nil {
			history.WithProductsCount(productCount)
		} else {
			s.logger.Warn("Failed to count products", zap.String("tenant_id", tenantID.String()), zap.Error(err))
		}

		// Count warehouses
		if warehouseCount, err := s.resourceCounter.CountWarehouses(ctx, tenantID); err == nil {
			history.WithWarehousesCount(warehouseCount)
		} else {
			s.logger.Warn("Failed to count warehouses", zap.String("tenant_id", tenantID.String()), zap.Error(err))
		}

		// Count customers
		if customerCount, err := s.resourceCounter.CountCustomers(ctx, tenantID); err == nil {
			history.WithCustomersCount(customerCount)
		} else {
			s.logger.Warn("Failed to count customers", zap.String("tenant_id", tenantID.String()), zap.Error(err))
		}

		// Count suppliers
		if supplierCount, err := s.resourceCounter.CountSuppliers(ctx, tenantID); err == nil {
			history.WithSuppliersCount(supplierCount)
		} else {
			s.logger.Warn("Failed to count suppliers", zap.String("tenant_id", tenantID.String()), zap.Error(err))
		}

		// Count orders
		if orderCount, err := s.resourceCounter.CountOrders(ctx, tenantID); err == nil {
			history.WithOrdersCount(orderCount)
		} else {
			s.logger.Warn("Failed to count orders", zap.String("tenant_id", tenantID.String()), zap.Error(err))
		}
	}

	// Upsert the snapshot (update if exists for same tenant+date)
	if err := s.historyRepo.Upsert(ctx, history); err != nil {
		s.logger.Error("Failed to save usage snapshot",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err))
		return nil, shared.NewDomainError("SAVE_FAILED", "Failed to save usage snapshot")
	}

	s.logger.Info("Usage snapshot created",
		zap.String("tenant_id", tenantID.String()),
		zap.Time("snapshot_date", snapshotDate),
		zap.Int64("users", history.UsersCount),
		zap.Int64("products", history.ProductsCount),
		zap.Int64("warehouses", history.WarehousesCount))

	return history, nil
}

// CreateDailySnapshots creates usage snapshots for all active tenants
func (s *UsageSnapshotService) CreateDailySnapshots(ctx context.Context) (*DailySnapshotResult, error) {
	snapshotDate := time.Now().UTC()

	s.logger.Info("Starting daily usage snapshot job", zap.Time("snapshot_date", snapshotDate))

	// Get all active tenants
	tenants, err := s.tenantRepo.FindAll(ctx, shared.Filter{})
	if err != nil {
		s.logger.Error("Failed to fetch tenants", zap.Error(err))
		return nil, shared.NewDomainError("FETCH_FAILED", "Failed to fetch tenants")
	}

	result := &DailySnapshotResult{
		SnapshotDate: snapshotDate,
		TotalTenants: len(tenants),
		Successful:   0,
		Failed:       0,
		Errors:       make([]SnapshotError, 0),
	}

	// Create snapshot for each tenant
	for _, tenant := range tenants {
		_, err := s.CreateSnapshotForTenant(ctx, tenant.ID, snapshotDate)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, SnapshotError{
				TenantID: tenant.ID,
				Error:    err.Error(),
			})
			s.logger.Warn("Failed to create snapshot for tenant",
				zap.String("tenant_id", tenant.ID.String()),
				zap.Error(err))
		} else {
			result.Successful++
		}
	}

	s.logger.Info("Daily usage snapshot job completed",
		zap.Int("total", result.TotalTenants),
		zap.Int("successful", result.Successful),
		zap.Int("failed", result.Failed))

	return result, nil
}

// CleanupOldSnapshots removes snapshots older than the retention period
func (s *UsageSnapshotService) CleanupOldSnapshots(ctx context.Context) (int64, error) {
	cutoffDate := time.Now().UTC().AddDate(0, 0, -s.retentionDays)

	s.logger.Info("Starting usage history cleanup",
		zap.Time("cutoff_date", cutoffDate),
		zap.Int("retention_days", s.retentionDays))

	deleted, err := s.historyRepo.DeleteOlderThan(ctx, cutoffDate)
	if err != nil {
		s.logger.Error("Failed to cleanup old snapshots", zap.Error(err))
		return 0, shared.NewDomainError("CLEANUP_FAILED", "Failed to cleanup old snapshots")
	}

	s.logger.Info("Usage history cleanup completed",
		zap.Int64("deleted_count", deleted))

	return deleted, nil
}

// GetUsageHistory retrieves usage history for a tenant within a date range
func (s *UsageSnapshotService) GetUsageHistory(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) ([]*billing.UsageHistory, error) {
	if tenantID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_TENANT", "Tenant ID cannot be empty")
	}

	filter := billing.UsageHistoryFilter{
		StartDate: &startDate,
		EndDate:   &endDate,
		Page:      1,
		PageSize:  1000, // Max 1000 records
	}

	histories, err := s.historyRepo.FindByTenant(ctx, tenantID, filter)
	if err != nil {
		s.logger.Error("Failed to fetch usage history",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err))
		return nil, shared.NewDomainError("FETCH_FAILED", "Failed to fetch usage history")
	}

	return histories, nil
}

// GetLatestSnapshot retrieves the most recent snapshot for a tenant
func (s *UsageSnapshotService) GetLatestSnapshot(ctx context.Context, tenantID uuid.UUID) (*billing.UsageHistory, error) {
	if tenantID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_TENANT", "Tenant ID cannot be empty")
	}

	history, err := s.historyRepo.FindLatestByTenant(ctx, tenantID)
	if err != nil {
		s.logger.Error("Failed to fetch latest snapshot",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err))
		return nil, shared.NewDomainError("FETCH_FAILED", "Failed to fetch latest snapshot")
	}

	return history, nil
}

// DailySnapshotResult contains the result of a daily snapshot job
type DailySnapshotResult struct {
	SnapshotDate time.Time       `json:"snapshot_date"`
	TotalTenants int             `json:"total_tenants"`
	Successful   int             `json:"successful"`
	Failed       int             `json:"failed"`
	Errors       []SnapshotError `json:"errors,omitempty"`
}

// SnapshotError contains error information for a failed snapshot
type SnapshotError struct {
	TenantID uuid.UUID `json:"tenant_id"`
	Error    string    `json:"error"`
}
