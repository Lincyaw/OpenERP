package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/erp/backend/internal/application/billing"
	"go.uber.org/zap"
)

// UsageSnapshotScheduler manages scheduled usage snapshot creation and cleanup
type UsageSnapshotScheduler struct {
	service   *billing.UsageSnapshotService
	logger    *zap.Logger
	config    UsageSnapshotSchedulerConfig
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mu        sync.Mutex
	isRunning bool
}

// UsageSnapshotSchedulerConfig holds configuration for the usage snapshot scheduler
type UsageSnapshotSchedulerConfig struct {
	// Enabled determines if the scheduler is active
	Enabled bool

	// SnapshotHour is the hour (0-23) when daily snapshots are created
	SnapshotHour int

	// CleanupEnabled enables automatic cleanup of old snapshots
	CleanupEnabled bool

	// CleanupHour is the hour (0-23) when cleanup runs (should be different from SnapshotHour)
	CleanupHour int

	// SnapshotTimeout is the maximum time for a snapshot run
	SnapshotTimeout time.Duration

	// CleanupTimeout is the maximum time for a cleanup run
	CleanupTimeout time.Duration
}

// DefaultUsageSnapshotSchedulerConfig returns default configuration
func DefaultUsageSnapshotSchedulerConfig() UsageSnapshotSchedulerConfig {
	return UsageSnapshotSchedulerConfig{
		Enabled:         true,
		SnapshotHour:    1, // 1 AM - create snapshots
		CleanupEnabled:  true,
		CleanupHour:     3, // 3 AM - cleanup old data
		SnapshotTimeout: 30 * time.Minute,
		CleanupTimeout:  15 * time.Minute,
	}
}

// NewUsageSnapshotScheduler creates a new usage snapshot scheduler
func NewUsageSnapshotScheduler(
	service *billing.UsageSnapshotService,
	logger *zap.Logger,
	config UsageSnapshotSchedulerConfig,
) *UsageSnapshotScheduler {
	return &UsageSnapshotScheduler{
		service: service,
		logger:  logger,
		config:  config,
	}
}

// Start starts the usage snapshot scheduler
func (s *UsageSnapshotScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return nil
	}
	if !s.config.Enabled {
		s.mu.Unlock()
		s.logger.Info("Usage snapshot scheduler is disabled")
		return nil
	}
	s.isRunning = true
	s.mu.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	// Start daily snapshot goroutine
	s.wg.Add(1)
	go s.runDailySnapshots(ctx)

	// Start cleanup goroutine if enabled
	if s.config.CleanupEnabled {
		s.wg.Add(1)
		go s.runDailyCleanup(ctx)
	}

	s.logger.Info("Usage snapshot scheduler started",
		zap.Int("snapshot_hour", s.config.SnapshotHour),
		zap.Bool("cleanup_enabled", s.config.CleanupEnabled),
		zap.Int("cleanup_hour", s.config.CleanupHour),
	)

	return nil
}

// Stop gracefully stops the scheduler
func (s *UsageSnapshotScheduler) Stop(ctx context.Context) error {
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
		s.logger.Info("Usage snapshot scheduler stopped gracefully")
		return nil
	case <-ctx.Done():
		s.logger.Warn("Usage snapshot scheduler stop timed out")
		return ctx.Err()
	}
}

// runDailySnapshots runs snapshot creation once per day at the configured hour
func (s *UsageSnapshotScheduler) runDailySnapshots(ctx context.Context) {
	defer s.wg.Done()

	for {
		// Calculate time until next daily run
		now := time.Now()
		nextRun := time.Date(now.Year(), now.Month(), now.Day(), s.config.SnapshotHour, 0, 0, 0, now.Location())
		if now.After(nextRun) {
			// Already past today's run time, schedule for tomorrow
			nextRun = nextRun.Add(24 * time.Hour)
		}
		delay := time.Until(nextRun)

		s.logger.Info("Daily usage snapshot scheduled",
			zap.Time("next_run", nextRun),
			zap.Duration("delay", delay),
		)

		select {
		case <-ctx.Done():
			s.logger.Debug("Daily snapshot loop stopping")
			return
		case <-time.After(delay):
			s.executeSnapshots(ctx)
		}
	}
}

// runDailyCleanup runs cleanup once per day at the configured hour
func (s *UsageSnapshotScheduler) runDailyCleanup(ctx context.Context) {
	defer s.wg.Done()

	for {
		// Calculate time until next daily run
		now := time.Now()
		nextRun := time.Date(now.Year(), now.Month(), now.Day(), s.config.CleanupHour, 0, 0, 0, now.Location())
		if now.After(nextRun) {
			// Already past today's run time, schedule for tomorrow
			nextRun = nextRun.Add(24 * time.Hour)
		}
		delay := time.Until(nextRun)

		s.logger.Info("Daily usage history cleanup scheduled",
			zap.Time("next_run", nextRun),
			zap.Duration("delay", delay),
		)

		select {
		case <-ctx.Done():
			s.logger.Debug("Daily cleanup loop stopping")
			return
		case <-time.After(delay):
			s.executeCleanup(ctx)
		}
	}
}

// executeSnapshots executes snapshot creation for all tenants
func (s *UsageSnapshotScheduler) executeSnapshots(ctx context.Context) {
	s.logger.Info("Starting daily usage snapshot creation",
		zap.Time("started_at", time.Now()),
	)

	// Create context with timeout
	snapshotCtx, cancel := context.WithTimeout(ctx, s.config.SnapshotTimeout)
	defer cancel()

	startTime := time.Now()
	result, err := s.service.CreateDailySnapshots(snapshotCtx)
	duration := time.Since(startTime)

	if err != nil {
		s.logger.Error("Daily usage snapshot creation failed",
			zap.Duration("duration", duration),
			zap.Error(err),
		)
		return
	}

	s.logger.Info("Daily usage snapshot creation completed",
		zap.Duration("duration", duration),
		zap.Int("total_tenants", result.TotalTenants),
		zap.Int("successful", result.Successful),
		zap.Int("failed", result.Failed),
	)
}

// executeCleanup executes cleanup of old snapshots
func (s *UsageSnapshotScheduler) executeCleanup(ctx context.Context) {
	s.logger.Info("Starting usage history cleanup",
		zap.Time("started_at", time.Now()),
	)

	// Create context with timeout
	cleanupCtx, cancel := context.WithTimeout(ctx, s.config.CleanupTimeout)
	defer cancel()

	startTime := time.Now()
	deleted, err := s.service.CleanupOldSnapshots(cleanupCtx)
	duration := time.Since(startTime)

	if err != nil {
		s.logger.Error("Usage history cleanup failed",
			zap.Duration("duration", duration),
			zap.Error(err),
		)
		return
	}

	s.logger.Info("Usage history cleanup completed",
		zap.Duration("duration", duration),
		zap.Int64("deleted_count", deleted),
	)
}

// TriggerImmediateSnapshot triggers an immediate snapshot creation run
func (s *UsageSnapshotScheduler) TriggerImmediateSnapshot(ctx context.Context) error {
	s.mu.Lock()
	if !s.isRunning {
		s.mu.Unlock()
		return ErrSchedulerNotRunning
	}
	s.wg.Add(1)
	s.mu.Unlock()

	s.logger.Info("Triggering immediate usage snapshot creation")

	// Run in a goroutine to not block
	go func() {
		defer s.wg.Done()
		s.executeSnapshots(ctx)
	}()

	return nil
}

// TriggerImmediateCleanup triggers an immediate cleanup run
func (s *UsageSnapshotScheduler) TriggerImmediateCleanup(ctx context.Context) error {
	s.mu.Lock()
	if !s.isRunning {
		s.mu.Unlock()
		return ErrSchedulerNotRunning
	}
	s.wg.Add(1)
	s.mu.Unlock()

	s.logger.Info("Triggering immediate usage history cleanup")

	// Run in a goroutine to not block
	go func() {
		defer s.wg.Done()
		s.executeCleanup(ctx)
	}()

	return nil
}

// IsRunning returns whether the scheduler is running
func (s *UsageSnapshotScheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isRunning
}
