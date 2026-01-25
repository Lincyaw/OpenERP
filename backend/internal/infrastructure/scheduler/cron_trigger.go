package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// TenantProvider provides a list of tenants for scheduling
type TenantProvider interface {
	GetAllActiveTenantIDs(ctx context.Context) ([]uuid.UUID, error)
}

// CronTriggerConfig holds configuration for the cron trigger
type CronTriggerConfig struct {
	// DailyReportTime is the time to run daily reports (hour:minute in 24h format)
	DailyReportHour   int
	DailyReportMinute int

	// CheckInterval is how often to check if it's time to run
	CheckInterval time.Duration
}

// DefaultCronTriggerConfig returns default cron trigger configuration
func DefaultCronTriggerConfig() CronTriggerConfig {
	return CronTriggerConfig{
		DailyReportHour:   2, // 2am
		DailyReportMinute: 0,
		CheckInterval:     time.Minute,
	}
}

// CronTrigger triggers scheduled report generation
type CronTrigger struct {
	config         CronTriggerConfig
	scheduler      *Scheduler
	tenantProvider TenantProvider
	logger         *zap.Logger

	cancel      context.CancelFunc
	wg          sync.WaitGroup
	mu          sync.Mutex
	isRunning   bool
	lastRunDate string // Track which date we last ran for
}

// NewCronTrigger creates a new cron trigger
func NewCronTrigger(
	config CronTriggerConfig,
	scheduler *Scheduler,
	tenantProvider TenantProvider,
	logger *zap.Logger,
) *CronTrigger {
	return &CronTrigger{
		config:         config,
		scheduler:      scheduler,
		tenantProvider: tenantProvider,
		logger:         logger,
	}
}

// Start starts the cron trigger
func (c *CronTrigger) Start(ctx context.Context) error {
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

	c.logger.Info("Cron trigger started",
		zap.Int("daily_hour", c.config.DailyReportHour),
		zap.Int("daily_minute", c.config.DailyReportMinute),
		zap.Duration("check_interval", c.config.CheckInterval),
	)

	return nil
}

// Stop stops the cron trigger
func (c *CronTrigger) Stop(ctx context.Context) error {
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
		c.logger.Info("Cron trigger stopped")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// runLoop checks periodically if it's time to run scheduled reports
func (c *CronTrigger) runLoop(ctx context.Context) {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.checkAndTrigger(ctx)
		}
	}
}

// checkAndTrigger checks if it's time to run and triggers reports
func (c *CronTrigger) checkAndTrigger(ctx context.Context) {
	now := time.Now()
	currentDate := now.Format("2006-01-02")

	// Skip if we already ran today
	c.mu.Lock()
	if c.lastRunDate == currentDate {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()

	// Check if it's the right time
	if now.Hour() != c.config.DailyReportHour || now.Minute() != c.config.DailyReportMinute {
		return
	}

	// It's time to run!
	c.mu.Lock()
	c.lastRunDate = currentDate
	c.mu.Unlock()

	c.logger.Info("Triggering daily report generation")
	c.triggerDailyReports(ctx)
}

// triggerDailyReports triggers report generation for all tenants
func (c *CronTrigger) triggerDailyReports(ctx context.Context) {
	// Get all active tenants
	tenantIDs, err := c.tenantProvider.GetAllActiveTenantIDs(ctx)
	if err != nil {
		c.logger.Error("Failed to get tenant IDs for daily reports", zap.Error(err))
		return
	}

	c.logger.Info("Scheduling daily reports for tenants",
		zap.Int("tenant_count", len(tenantIDs)),
	)

	// Schedule reports for each tenant
	for _, tenantID := range tenantIDs {
		tid := tenantID // Capture for closure
		if err := c.scheduler.ScheduleDailyReports(&tid); err != nil {
			c.logger.Error("Failed to schedule daily reports for tenant",
				zap.String("tenant_id", tenantID.String()),
				zap.Error(err),
			)
		}
	}
}

// TriggerManualRefresh allows manual triggering of reports
func (c *CronTrigger) TriggerManualRefresh(ctx context.Context, tenantID *uuid.UUID, reportType *ReportType, periodStart, periodEnd time.Time) error {
	if reportType != nil {
		// Trigger specific report type
		return c.scheduler.ScheduleReport(tenantID, *reportType, periodStart, periodEnd)
	}

	// Trigger all report types
	for _, rt := range AllReportTypes() {
		if err := c.scheduler.ScheduleReport(tenantID, rt, periodStart, periodEnd); err != nil {
			return err
		}
	}
	return nil
}
