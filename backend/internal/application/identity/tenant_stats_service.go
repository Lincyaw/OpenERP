package identity

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// TenantStatsService handles tenant usage statistics operations for super admins.
// This service provides statistics that bypass tenant isolation.
// IMPORTANT: All methods must be protected by SuperAdminMiddleware.
type TenantStatsService struct {
	statsRepo    identity.TenantStatsRepository
	adminRepo    identity.AdminTenantRepository
	redisClient  *redis.Client
	logger       *zap.Logger
	cacheTTL     time.Duration
	cacheEnabled bool
}

// TenantStatsServiceConfig holds configuration for TenantStatsService
type TenantStatsServiceConfig struct {
	CacheTTL     time.Duration
	CacheEnabled bool
}

// DefaultTenantStatsServiceConfig returns default configuration
func DefaultTenantStatsServiceConfig() TenantStatsServiceConfig {
	return TenantStatsServiceConfig{
		CacheTTL:     5 * time.Minute,
		CacheEnabled: true,
	}
}

// NewTenantStatsService creates a new tenant stats service
func NewTenantStatsService(
	statsRepo identity.TenantStatsRepository,
	adminRepo identity.AdminTenantRepository,
	redisClient *redis.Client,
	logger *zap.Logger,
	config TenantStatsServiceConfig,
) *TenantStatsService {
	return &TenantStatsService{
		statsRepo:    statsRepo,
		adminRepo:    adminRepo,
		redisClient:  redisClient,
		logger:       logger,
		cacheTTL:     config.CacheTTL,
		cacheEnabled: config.CacheEnabled && redisClient != nil,
	}
}

// Cache key prefixes
const (
	cacheKeyTenantStats   = "admin:stats:tenant:"
	cacheKeyPlatformStats = "admin:stats:platform"
	cacheKeyGrowthTrend   = "admin:stats:growth:"
)

// ============================================================================
// DTOs
// ============================================================================

// TenantUsageStatsDTO represents usage statistics for a single tenant
type TenantUsageStatsDTO struct {
	TenantID       uuid.UUID `json:"tenant_id"`
	TenantName     string    `json:"tenant_name"`
	TenantCode     string    `json:"tenant_code"`
	Plan           string    `json:"plan"`
	Status         string    `json:"status"`
	UserCount      int64     `json:"user_count"`
	ProductCount   int64     `json:"product_count"`
	OrderCount     int64     `json:"order_count"`
	WarehouseCount int64     `json:"warehouse_count"`
	StorageUsage   int64     `json:"storage_usage"`
	APICallCount   int64     `json:"api_call_count"`
	// Quota information
	MaxUsers      int   `json:"max_users"`
	MaxProducts   int   `json:"max_products"`
	MaxWarehouses int   `json:"max_warehouses"`
	LastUpdated   int64 `json:"last_updated"` // Unix timestamp
}

// PlatformStatsDTO represents platform-wide statistics
type PlatformStatsDTO struct {
	TotalTenants      int64            `json:"total_tenants"`
	ActiveTenants     int64            `json:"active_tenants"`
	TrialTenants      int64            `json:"trial_tenants"`
	SuspendedTenants  int64            `json:"suspended_tenants"`
	InactiveTenants   int64            `json:"inactive_tenants"`
	TenantsByPlan     map[string]int64 `json:"tenants_by_plan"`
	TotalUsers        int64            `json:"total_users"`
	TotalProducts     int64            `json:"total_products"`
	TotalOrders       int64            `json:"total_orders"`
	TotalStorageUsage int64            `json:"total_storage_usage"`
	LastUpdated       int64            `json:"last_updated"` // Unix timestamp
}

// TenantGrowthPointDTO represents a single data point in growth trend
type TenantGrowthPointDTO struct {
	Date         string `json:"date"` // ISO 8601 format
	TotalTenants int64  `json:"total_tenants"`
	NewTenants   int64  `json:"new_tenants"`
	ChurnedCount int64  `json:"churned_count"`
}

// TenantGrowthTrendDTO represents tenant growth over time
type TenantGrowthTrendDTO struct {
	Period     string                 `json:"period"`
	StartDate  string                 `json:"start_date"`
	EndDate    string                 `json:"end_date"`
	DataPoints []TenantGrowthPointDTO `json:"data_points"`
}

// TenantGrowthInput contains input for growth trend queries
type TenantGrowthInput struct {
	Period string `json:"period"` // "daily", "weekly", "monthly"
	Days   int    `json:"days"`   // Number of days to look back
}

// ============================================================================
// Service Methods
// ============================================================================

// GetTenantStats retrieves usage statistics for a specific tenant
func (s *TenantStatsService) GetTenantStats(ctx context.Context, tenantID uuid.UUID) (*TenantUsageStatsDTO, error) {
	s.logger.Debug("Getting tenant stats",
		zap.String("tenant_id", tenantID.String()))

	// Try to get from cache first
	if s.cacheEnabled {
		cached, err := s.getTenantStatsFromCache(ctx, tenantID)
		if err == nil && cached != nil {
			s.logger.Debug("Tenant stats cache hit",
				zap.String("tenant_id", tenantID.String()))
			return cached, nil
		}
	}

	// Verify tenant exists and get tenant info
	tenant, err := s.adminRepo.FindByID(ctx, tenantID)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("TENANT_NOT_FOUND", "Tenant not found")
		}
		s.logger.Error("Failed to find tenant", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find tenant")
	}

	// Get usage stats from repository
	stats, err := s.statsRepo.GetTenantUsageStats(ctx, tenantID)
	if err != nil {
		s.logger.Error("Failed to get tenant usage stats", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to get tenant statistics")
	}

	// Build DTO with tenant info
	dto := &TenantUsageStatsDTO{
		TenantID:       tenant.ID,
		TenantName:     tenant.Name,
		TenantCode:     tenant.Code,
		Plan:           string(tenant.Plan),
		Status:         string(tenant.Status),
		UserCount:      stats.UserCount,
		ProductCount:   stats.ProductCount,
		OrderCount:     stats.OrderCount,
		WarehouseCount: stats.WarehouseCount,
		StorageUsage:   stats.StorageUsage,
		APICallCount:   stats.APICallCount,
		MaxUsers:       tenant.Config.MaxUsers,
		MaxProducts:    tenant.Config.MaxProducts,
		MaxWarehouses:  tenant.Config.MaxWarehouses,
		LastUpdated:    time.Now().Unix(),
	}

	// Cache the result
	if s.cacheEnabled {
		if err := s.cacheTenantStats(ctx, tenantID, dto); err != nil {
			s.logger.Warn("Failed to cache tenant stats", zap.Error(err))
		}
	}

	return dto, nil
}

// GetPlatformStats retrieves platform-wide statistics
func (s *TenantStatsService) GetPlatformStats(ctx context.Context) (*PlatformStatsDTO, error) {
	s.logger.Debug("Getting platform stats")

	// Try to get from cache first
	if s.cacheEnabled {
		cached, err := s.getPlatformStatsFromCache(ctx)
		if err == nil && cached != nil {
			s.logger.Debug("Platform stats cache hit")
			return cached, nil
		}
	}

	// Get platform stats from repository
	stats, err := s.statsRepo.GetPlatformStats(ctx)
	if err != nil {
		s.logger.Error("Failed to get platform stats", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to get platform statistics")
	}

	// Build DTO
	dto := &PlatformStatsDTO{
		TotalTenants:      stats.TotalTenants,
		ActiveTenants:     stats.ActiveTenants,
		TrialTenants:      stats.TrialTenants,
		SuspendedTenants:  stats.SuspendedTenants,
		InactiveTenants:   stats.InactiveTenants,
		TenantsByPlan:     stats.TenantsByPlan,
		TotalUsers:        stats.TotalUsers,
		TotalProducts:     stats.TotalProducts,
		TotalOrders:       stats.TotalOrders,
		TotalStorageUsage: stats.TotalStorageUsage,
		LastUpdated:       time.Now().Unix(),
	}

	// Cache the result
	if s.cacheEnabled {
		if err := s.cachePlatformStats(ctx, dto); err != nil {
			s.logger.Warn("Failed to cache platform stats", zap.Error(err))
		}
	}

	return dto, nil
}

// GetTenantGrowth retrieves tenant growth trend data
func (s *TenantStatsService) GetTenantGrowth(ctx context.Context, input TenantGrowthInput) (*TenantGrowthTrendDTO, error) {
	s.logger.Debug("Getting tenant growth trend",
		zap.String("period", input.Period),
		zap.Int("days", input.Days))

	// Validate input
	if err := s.validateGrowthInput(input); err != nil {
		return nil, err
	}

	// Try to get from cache first
	cacheKey := fmt.Sprintf("%s%s:%d", cacheKeyGrowthTrend, input.Period, input.Days)
	if s.cacheEnabled {
		cached, err := s.getGrowthTrendFromCache(ctx, cacheKey)
		if err == nil && cached != nil {
			s.logger.Debug("Growth trend cache hit",
				zap.String("period", input.Period))
			return cached, nil
		}
	}

	// Get growth trend from repository
	trend, err := s.statsRepo.GetTenantGrowthTrend(ctx, input.Period, input.Days)
	if err != nil {
		s.logger.Error("Failed to get tenant growth trend", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to get growth trend")
	}

	// Convert to DTO
	dto := s.toGrowthTrendDTO(trend)

	// Cache the result
	if s.cacheEnabled {
		if err := s.cacheGrowthTrend(ctx, cacheKey, dto); err != nil {
			s.logger.Warn("Failed to cache growth trend", zap.Error(err))
		}
	}

	return dto, nil
}

// InvalidateCache invalidates all stats caches
func (s *TenantStatsService) InvalidateCache(ctx context.Context) error {
	if !s.cacheEnabled {
		return nil
	}

	s.logger.Info("Invalidating all stats caches")

	// Delete platform stats cache
	if err := s.redisClient.Del(ctx, cacheKeyPlatformStats).Err(); err != nil {
		s.logger.Warn("Failed to delete platform stats cache", zap.Error(err))
	}

	// Delete all tenant stats caches (using pattern)
	keys, err := s.redisClient.Keys(ctx, cacheKeyTenantStats+"*").Result()
	if err != nil {
		s.logger.Warn("Failed to get tenant stats cache keys", zap.Error(err))
	} else if len(keys) > 0 {
		if err := s.redisClient.Del(ctx, keys...).Err(); err != nil {
			s.logger.Warn("Failed to delete tenant stats caches", zap.Error(err))
		}
	}

	// Delete all growth trend caches
	keys, err = s.redisClient.Keys(ctx, cacheKeyGrowthTrend+"*").Result()
	if err != nil {
		s.logger.Warn("Failed to get growth trend cache keys", zap.Error(err))
	} else if len(keys) > 0 {
		if err := s.redisClient.Del(ctx, keys...).Err(); err != nil {
			s.logger.Warn("Failed to delete growth trend caches", zap.Error(err))
		}
	}

	return nil
}

// InvalidateTenantCache invalidates cache for a specific tenant
func (s *TenantStatsService) InvalidateTenantCache(ctx context.Context, tenantID uuid.UUID) error {
	if !s.cacheEnabled {
		return nil
	}

	s.logger.Debug("Invalidating tenant stats cache",
		zap.String("tenant_id", tenantID.String()))

	key := cacheKeyTenantStats + tenantID.String()
	if err := s.redisClient.Del(ctx, key).Err(); err != nil {
		s.logger.Warn("Failed to delete tenant stats cache", zap.Error(err))
		return err
	}

	// Also invalidate platform stats since tenant data changed
	if err := s.redisClient.Del(ctx, cacheKeyPlatformStats).Err(); err != nil {
		s.logger.Warn("Failed to delete platform stats cache", zap.Error(err))
	}

	return nil
}

// ============================================================================
// Validation Helpers
// ============================================================================

func (s *TenantStatsService) validateGrowthInput(input TenantGrowthInput) error {
	// Validate period
	switch input.Period {
	case "daily", "weekly", "monthly":
		// Valid periods
	default:
		return shared.NewDomainError("INVALID_PERIOD", "Period must be 'daily', 'weekly', or 'monthly'")
	}

	// Validate days
	if input.Days < 1 {
		return shared.NewDomainError("INVALID_DAYS", "Days must be at least 1")
	}
	if input.Days > 365 {
		return shared.NewDomainError("INVALID_DAYS", "Days cannot exceed 365")
	}

	return nil
}

// ============================================================================
// Cache Helpers
// ============================================================================

func (s *TenantStatsService) getTenantStatsFromCache(ctx context.Context, tenantID uuid.UUID) (*TenantUsageStatsDTO, error) {
	key := cacheKeyTenantStats + tenantID.String()
	data, err := s.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var dto TenantUsageStatsDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, err
	}

	return &dto, nil
}

func (s *TenantStatsService) cacheTenantStats(ctx context.Context, tenantID uuid.UUID, dto *TenantUsageStatsDTO) error {
	key := cacheKeyTenantStats + tenantID.String()
	data, err := json.Marshal(dto)
	if err != nil {
		return err
	}

	return s.redisClient.Set(ctx, key, data, s.cacheTTL).Err()
}

func (s *TenantStatsService) getPlatformStatsFromCache(ctx context.Context) (*PlatformStatsDTO, error) {
	data, err := s.redisClient.Get(ctx, cacheKeyPlatformStats).Bytes()
	if err != nil {
		return nil, err
	}

	var dto PlatformStatsDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, err
	}

	return &dto, nil
}

func (s *TenantStatsService) cachePlatformStats(ctx context.Context, dto *PlatformStatsDTO) error {
	data, err := json.Marshal(dto)
	if err != nil {
		return err
	}

	return s.redisClient.Set(ctx, cacheKeyPlatformStats, data, s.cacheTTL).Err()
}

func (s *TenantStatsService) getGrowthTrendFromCache(ctx context.Context, key string) (*TenantGrowthTrendDTO, error) {
	data, err := s.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var dto TenantGrowthTrendDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, err
	}

	return &dto, nil
}

func (s *TenantStatsService) cacheGrowthTrend(ctx context.Context, key string, dto *TenantGrowthTrendDTO) error {
	data, err := json.Marshal(dto)
	if err != nil {
		return err
	}

	return s.redisClient.Set(ctx, key, data, s.cacheTTL).Err()
}

// ============================================================================
// Conversion Helpers
// ============================================================================

func (s *TenantStatsService) toGrowthTrendDTO(trend *identity.TenantGrowthTrend) *TenantGrowthTrendDTO {
	dataPoints := make([]TenantGrowthPointDTO, len(trend.DataPoints))
	for i, point := range trend.DataPoints {
		dataPoints[i] = TenantGrowthPointDTO{
			Date:         point.Date.Format("2006-01-02"),
			TotalTenants: point.TotalTenants,
			NewTenants:   point.NewTenants,
			ChurnedCount: point.ChurnedCount,
		}
	}

	return &TenantGrowthTrendDTO{
		Period:     trend.Period,
		StartDate:  trend.StartDate.Format("2006-01-02"),
		EndDate:    trend.EndDate.Format("2006-01-02"),
		DataPoints: dataPoints,
	}
}
