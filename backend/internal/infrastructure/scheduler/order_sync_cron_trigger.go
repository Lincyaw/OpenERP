package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/erp/backend/internal/domain/integration"
)

// ---------------------------------------------------------------------------
// TenantPlatformConfig Provider
// ---------------------------------------------------------------------------

// OrderSyncConfigProvider provides sync configuration for tenants
type OrderSyncConfigProvider interface {
	// GetEnabledConfigs returns all enabled sync configs
	GetEnabledConfigs(ctx context.Context) ([]integration.OrderSyncConfig, error)

	// GetConfigByTenantAndPlatform returns sync config for a specific tenant and platform
	GetConfigByTenantAndPlatform(ctx context.Context, tenantID uuid.UUID, platformCode integration.PlatformCode) (*integration.OrderSyncConfig, error)

	// GetLastSyncTime returns the last successful sync time for a tenant/platform
	GetLastSyncTime(ctx context.Context, tenantID uuid.UUID, platformCode integration.PlatformCode) (*time.Time, error)

	// UpdateLastSyncTime updates the last sync time after successful sync
	UpdateLastSyncTime(ctx context.Context, tenantID uuid.UUID, platformCode integration.PlatformCode, syncTime time.Time) error
}

// ---------------------------------------------------------------------------
// OrderSyncCronTriggerConfig
// ---------------------------------------------------------------------------

// OrderSyncCronTriggerConfig holds configuration for the order sync cron trigger
type OrderSyncCronTriggerConfig struct {
	// CheckInterval is how often to check for sync jobs to schedule
	CheckInterval time.Duration

	// DefaultSyncIntervalMinutes is the default sync interval if not configured per-tenant
	DefaultSyncIntervalMinutes int

	// LookbackMinutes is how many minutes to look back when syncing
	// Used as a buffer to ensure no orders are missed
	LookbackMinutes int
}

// DefaultOrderSyncCronTriggerConfig returns default configuration
func DefaultOrderSyncCronTriggerConfig() OrderSyncCronTriggerConfig {
	return OrderSyncCronTriggerConfig{
		CheckInterval:              time.Minute,
		DefaultSyncIntervalMinutes: 15,
		LookbackMinutes:            5,
	}
}

// ---------------------------------------------------------------------------
// OrderSyncCronTrigger
// ---------------------------------------------------------------------------

// OrderSyncCronTrigger triggers order sync jobs based on configured schedules
type OrderSyncCronTrigger struct {
	config         OrderSyncCronTriggerConfig
	scheduler      *OrderSyncScheduler
	configProvider OrderSyncConfigProvider
	logger         *zap.Logger

	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mu        sync.Mutex
	isRunning bool

	// Track last scheduled time per tenant/platform to avoid duplicate scheduling
	lastScheduledMu sync.RWMutex
	lastScheduled   map[string]time.Time
}

// NewOrderSyncCronTrigger creates a new order sync cron trigger
func NewOrderSyncCronTrigger(
	config OrderSyncCronTriggerConfig,
	scheduler *OrderSyncScheduler,
	configProvider OrderSyncConfigProvider,
	logger *zap.Logger,
) *OrderSyncCronTrigger {
	return &OrderSyncCronTrigger{
		config:         config,
		scheduler:      scheduler,
		configProvider: configProvider,
		logger:         logger,
		lastScheduled:  make(map[string]time.Time),
	}
}

// Start starts the cron trigger
func (c *OrderSyncCronTrigger) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.isRunning {
		c.mu.Unlock()
		return nil
	}
	c.isRunning = true
	c.mu.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	c.wg.Add(1)
	go c.runLoop(ctx)

	c.logger.Info("Order sync cron trigger started",
		zap.Duration("check_interval", c.config.CheckInterval),
		zap.Int("default_sync_interval_minutes", c.config.DefaultSyncIntervalMinutes),
	)

	return nil
}

// Stop stops the cron trigger
func (c *OrderSyncCronTrigger) Stop(ctx context.Context) error {
	c.mu.Lock()
	if !c.isRunning {
		c.mu.Unlock()
		return nil
	}
	c.isRunning = false
	c.mu.Unlock()

	if c.cancel != nil {
		c.cancel()
	}

	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		c.logger.Info("Order sync cron trigger stopped")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// runLoop periodically checks and triggers sync jobs
func (c *OrderSyncCronTrigger) runLoop(ctx context.Context) {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.CheckInterval)
	defer ticker.Stop()

	// Run immediately on start
	c.checkAndSchedule(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.checkAndSchedule(ctx)
		}
	}
}

// checkAndSchedule checks all enabled sync configs and schedules jobs as needed
func (c *OrderSyncCronTrigger) checkAndSchedule(ctx context.Context) {
	// Get all enabled sync configs
	configs, err := c.configProvider.GetEnabledConfigs(ctx)
	if err != nil {
		c.logger.Error("Failed to get enabled sync configs", zap.Error(err))
		return
	}

	if len(configs) == 0 {
		c.logger.Debug("No enabled sync configs found")
		return
	}

	c.logger.Debug("Checking sync schedules",
		zap.Int("config_count", len(configs)),
	)

	now := time.Now()

	for _, cfg := range configs {
		if !cfg.IsEnabled {
			continue
		}

		// Check if it's time to sync
		shouldSync, startTime, endTime := c.shouldScheduleSync(ctx, cfg, now)
		if !shouldSync {
			continue
		}

		// Schedule the sync job
		c.logger.Info("Scheduling order sync job",
			zap.String("tenant_id", cfg.TenantID.String()),
			zap.String("platform_code", string(cfg.PlatformCode)),
			zap.Time("start_time", startTime),
			zap.Time("end_time", endTime),
		)

		if err := c.scheduler.ScheduleSync(cfg.TenantID, cfg.PlatformCode, startTime, endTime); err != nil {
			c.logger.Error("Failed to schedule sync job",
				zap.String("tenant_id", cfg.TenantID.String()),
				zap.String("platform_code", string(cfg.PlatformCode)),
				zap.Error(err),
			)
			continue
		}

		// Update last scheduled time
		c.updateLastScheduled(cfg.TenantID, cfg.PlatformCode, now)
	}
}

// shouldScheduleSync determines if a sync should be scheduled for this config
func (c *OrderSyncCronTrigger) shouldScheduleSync(
	ctx context.Context,
	cfg integration.OrderSyncConfig,
	now time.Time,
) (bool, time.Time, time.Time) {
	key := c.makeKey(cfg.TenantID, cfg.PlatformCode)

	// Get configured interval (or use default)
	intervalMinutes := cfg.SyncIntervalMinutes
	if intervalMinutes <= 0 {
		intervalMinutes = c.config.DefaultSyncIntervalMinutes
	}
	interval := time.Duration(intervalMinutes) * time.Minute

	// Check if we've scheduled recently (within interval)
	c.lastScheduledMu.RLock()
	lastScheduled, exists := c.lastScheduled[key]
	c.lastScheduledMu.RUnlock()

	if exists && now.Sub(lastScheduled) < interval {
		return false, time.Time{}, time.Time{}
	}

	// Get last sync time from provider
	lastSyncTime, err := c.configProvider.GetLastSyncTime(ctx, cfg.TenantID, cfg.PlatformCode)
	if err != nil {
		c.logger.Warn("Failed to get last sync time, using default lookback",
			zap.String("tenant_id", cfg.TenantID.String()),
			zap.String("platform_code", string(cfg.PlatformCode)),
			zap.Error(err),
		)
	}

	// Calculate time range for sync
	var startTime time.Time
	if lastSyncTime != nil {
		// Start from last sync time minus lookback buffer
		startTime = lastSyncTime.Add(-time.Duration(c.config.LookbackMinutes) * time.Minute)
	} else {
		// First sync: look back 24 hours by default
		startTime = now.Add(-24 * time.Hour)
	}

	endTime := now

	return true, startTime, endTime
}

// makeKey creates a unique key for tenant/platform combination
func (c *OrderSyncCronTrigger) makeKey(tenantID uuid.UUID, platformCode integration.PlatformCode) string {
	return tenantID.String() + ":" + string(platformCode)
}

// updateLastScheduled updates the last scheduled time for a tenant/platform
func (c *OrderSyncCronTrigger) updateLastScheduled(tenantID uuid.UUID, platformCode integration.PlatformCode, t time.Time) {
	key := c.makeKey(tenantID, platformCode)
	c.lastScheduledMu.Lock()
	c.lastScheduled[key] = t
	c.lastScheduledMu.Unlock()
}

// TriggerManualSync triggers an immediate sync for a specific tenant/platform
func (c *OrderSyncCronTrigger) TriggerManualSync(
	ctx context.Context,
	tenantID uuid.UUID,
	platformCode integration.PlatformCode,
	startTime, endTime time.Time,
) error {
	// Validate time range
	if startTime.After(endTime) {
		return ErrOrderSyncInvalidTimeRange
	}
	if endTime.Sub(startTime) > 7*24*time.Hour {
		return ErrOrderSyncInvalidTimeRange // Max 7 days per sync
	}

	c.logger.Info("Manual order sync triggered",
		zap.String("tenant_id", tenantID.String()),
		zap.String("platform_code", string(platformCode)),
		zap.Time("start_time", startTime),
		zap.Time("end_time", endTime),
	)

	return c.scheduler.ScheduleSync(tenantID, platformCode, startTime, endTime)
}

// TriggerManualSyncForAllPlatforms triggers sync for all enabled platforms of a tenant
func (c *OrderSyncCronTrigger) TriggerManualSyncForAllPlatforms(
	ctx context.Context,
	tenantID uuid.UUID,
	startTime, endTime time.Time,
) error {
	// Validate time range
	if startTime.After(endTime) {
		return ErrOrderSyncInvalidTimeRange
	}

	// Get enabled configs for tenant
	configs, err := c.configProvider.GetEnabledConfigs(ctx)
	if err != nil {
		return err
	}

	// Filter by tenant and schedule
	scheduled := 0
	for _, cfg := range configs {
		if cfg.TenantID != tenantID || !cfg.IsEnabled {
			continue
		}

		if err := c.scheduler.ScheduleSync(cfg.TenantID, cfg.PlatformCode, startTime, endTime); err != nil {
			c.logger.Error("Failed to schedule sync for platform",
				zap.String("platform_code", string(cfg.PlatformCode)),
				zap.Error(err),
			)
			continue
		}
		scheduled++
	}

	if scheduled == 0 {
		return ErrOrderSyncNoEnabledPlatforms
	}

	c.logger.Info("Manual sync scheduled for all platforms",
		zap.String("tenant_id", tenantID.String()),
		zap.Int("platforms_scheduled", scheduled),
	)

	return nil
}

// GetSchedulerStats returns statistics about the scheduler
func (c *OrderSyncCronTrigger) GetSchedulerStats() map[string]interface{} {
	c.lastScheduledMu.RLock()
	defer c.lastScheduledMu.RUnlock()

	stats := make(map[string]interface{})
	stats["is_running"] = c.isRunning
	stats["check_interval"] = c.config.CheckInterval.String()
	stats["tracked_configs"] = len(c.lastScheduled)

	lastScheduledTimes := make(map[string]string)
	for key, t := range c.lastScheduled {
		lastScheduledTimes[key] = t.Format(time.RFC3339)
	}
	stats["last_scheduled"] = lastScheduledTimes

	return stats
}
