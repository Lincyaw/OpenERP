package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/erp/backend/internal/application/billing"
	"go.uber.org/zap"
)

// UsageReportingScheduler manages scheduled usage reporting to Stripe
type UsageReportingScheduler struct {
	service   *billing.UsageReportingService
	logger    *zap.Logger
	config    UsageReportingSchedulerConfig
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mu        sync.Mutex
	isRunning bool
}

// UsageReportingSchedulerConfig holds configuration for the usage reporting scheduler
type UsageReportingSchedulerConfig struct {
	// Enabled determines if the scheduler is active
	Enabled bool

	// HourlyReportingEnabled enables hourly usage reporting
	HourlyReportingEnabled bool

	// DailyReportingEnabled enables daily usage reporting (runs at DailyReportingHour)
	DailyReportingEnabled bool

	// DailyReportingHour is the hour (0-23) when daily reporting runs
	DailyReportingHour int

	// RetryInterval is how often to retry failed reports
	RetryInterval time.Duration

	// ReportingTimeout is the maximum time for a reporting run
	ReportingTimeout time.Duration
}

// DefaultUsageReportingSchedulerConfig returns default configuration
func DefaultUsageReportingSchedulerConfig() UsageReportingSchedulerConfig {
	return UsageReportingSchedulerConfig{
		Enabled:                true,
		HourlyReportingEnabled: true,
		DailyReportingEnabled:  true,
		DailyReportingHour:     2, // 2 AM
		RetryInterval:          15 * time.Minute,
		ReportingTimeout:       30 * time.Minute,
	}
}

// NewUsageReportingScheduler creates a new usage reporting scheduler
func NewUsageReportingScheduler(
	service *billing.UsageReportingService,
	logger *zap.Logger,
	config UsageReportingSchedulerConfig,
) *UsageReportingScheduler {
	return &UsageReportingScheduler{
		service: service,
		logger:  logger,
		config:  config,
	}
}

// Start starts the usage reporting scheduler
func (s *UsageReportingScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return nil
	}
	if !s.config.Enabled {
		s.mu.Unlock()
		s.logger.Info("Usage reporting scheduler is disabled")
		return nil
	}
	s.isRunning = true
	s.mu.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	// Start hourly reporting goroutine
	if s.config.HourlyReportingEnabled {
		s.wg.Add(1)
		go s.runHourlyReporting(ctx)
	}

	// Start daily reporting goroutine
	if s.config.DailyReportingEnabled {
		s.wg.Add(1)
		go s.runDailyReporting(ctx)
	}

	// Start retry goroutine
	s.wg.Add(1)
	go s.runRetryLoop(ctx)

	s.logger.Info("Usage reporting scheduler started",
		zap.Bool("hourly_enabled", s.config.HourlyReportingEnabled),
		zap.Bool("daily_enabled", s.config.DailyReportingEnabled),
		zap.Int("daily_hour", s.config.DailyReportingHour),
	)

	return nil
}

// Stop gracefully stops the scheduler
func (s *UsageReportingScheduler) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.isRunning {
		s.mu.Unlock()
		return nil
	}
	s.isRunning = false
	s.mu.Unlock()

	if s.cancel != nil {
		s.cancel()
	}

	// Wait for goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("Usage reporting scheduler stopped gracefully")
		return nil
	case <-ctx.Done():
		s.logger.Warn("Usage reporting scheduler stop timed out")
		return ctx.Err()
	}
}

// runHourlyReporting runs usage reporting every hour
func (s *UsageReportingScheduler) runHourlyReporting(ctx context.Context) {
	defer s.wg.Done()

	// Calculate time until next hour
	now := time.Now()
	nextHour := now.Truncate(time.Hour).Add(time.Hour)
	initialDelay := time.Until(nextHour)

	s.logger.Info("Hourly usage reporting scheduled",
		zap.Time("next_run", nextHour),
		zap.Duration("initial_delay", initialDelay),
	)

	// Wait until the next hour
	select {
	case <-ctx.Done():
		return
	case <-time.After(initialDelay):
	}

	// Create ticker for hourly runs
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	// Run immediately after initial delay
	s.executeReporting(ctx, "hourly")

	for {
		select {
		case <-ctx.Done():
			s.logger.Debug("Hourly reporting loop stopping")
			return
		case <-ticker.C:
			s.executeReporting(ctx, "hourly")
		}
	}
}

// runDailyReporting runs usage reporting once per day at the configured hour
func (s *UsageReportingScheduler) runDailyReporting(ctx context.Context) {
	defer s.wg.Done()

	for {
		// Calculate time until next daily run
		now := time.Now()
		nextRun := time.Date(now.Year(), now.Month(), now.Day(), s.config.DailyReportingHour, 0, 0, 0, now.Location())
		if now.After(nextRun) {
			// Already past today's run time, schedule for tomorrow
			nextRun = nextRun.Add(24 * time.Hour)
		}
		delay := time.Until(nextRun)

		s.logger.Info("Daily usage reporting scheduled",
			zap.Time("next_run", nextRun),
			zap.Duration("delay", delay),
		)

		select {
		case <-ctx.Done():
			s.logger.Debug("Daily reporting loop stopping")
			return
		case <-time.After(delay):
			s.executeReporting(ctx, "daily")
		}
	}
}

// runRetryLoop periodically retries failed usage reports
func (s *UsageReportingScheduler) runRetryLoop(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.RetryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Debug("Retry loop stopping")
			return
		case <-ticker.C:
			s.executeRetry(ctx)
		}
	}
}

// executeReporting executes usage reporting for all tenants
func (s *UsageReportingScheduler) executeReporting(ctx context.Context, reportType string) {
	s.logger.Info("Starting usage reporting",
		zap.String("type", reportType),
		zap.Time("started_at", time.Now()),
	)

	// Create context with timeout
	reportCtx, cancel := context.WithTimeout(ctx, s.config.ReportingTimeout)
	defer cancel()

	startTime := time.Now()
	err := s.service.ReportUsageForAllTenants(reportCtx)
	duration := time.Since(startTime)

	if err != nil {
		s.logger.Error("Usage reporting failed",
			zap.String("type", reportType),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
		return
	}

	s.logger.Info("Usage reporting completed",
		zap.String("type", reportType),
		zap.Duration("duration", duration),
	)
}

// executeRetry retries failed usage reports
func (s *UsageReportingScheduler) executeRetry(ctx context.Context) {
	s.logger.Debug("Starting retry of failed usage reports")

	// Create context with timeout
	retryCtx, cancel := context.WithTimeout(ctx, s.config.ReportingTimeout)
	defer cancel()

	startTime := time.Now()
	err := s.service.RetryFailedReports(retryCtx)
	duration := time.Since(startTime)

	if err != nil {
		s.logger.Error("Retry of failed reports failed",
			zap.Duration("duration", duration),
			zap.Error(err),
		)
		return
	}

	s.logger.Debug("Retry of failed reports completed",
		zap.Duration("duration", duration),
	)
}

// TriggerImmediateReporting triggers an immediate usage reporting run
func (s *UsageReportingScheduler) TriggerImmediateReporting(ctx context.Context) error {
	s.mu.Lock()
	if !s.isRunning {
		s.mu.Unlock()
		return ErrSchedulerNotRunning
	}
	s.wg.Add(1) // Track the goroutine
	s.mu.Unlock()

	s.logger.Info("Triggering immediate usage reporting")

	// Run in a goroutine to not block
	go func() {
		defer s.wg.Done()
		s.executeReporting(ctx, "manual")
	}()

	return nil
}

// IsRunning returns whether the scheduler is running
func (s *UsageReportingScheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isRunning
}
