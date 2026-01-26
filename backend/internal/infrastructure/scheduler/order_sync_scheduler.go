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
// Order Sync Job Types
// ---------------------------------------------------------------------------

// OrderSyncJobStatus represents the status of an order sync job
type OrderSyncJobStatus string

const (
	OrderSyncJobStatusPending   OrderSyncJobStatus = "PENDING"
	OrderSyncJobStatusRunning   OrderSyncJobStatus = "RUNNING"
	OrderSyncJobStatusSuccess   OrderSyncJobStatus = "SUCCESS"
	OrderSyncJobStatusPartial   OrderSyncJobStatus = "PARTIAL"
	OrderSyncJobStatusFailed    OrderSyncJobStatus = "FAILED"
	OrderSyncJobStatusCancelled OrderSyncJobStatus = "CANCELLED"
)

// OrderSyncJob represents a scheduled order sync job
type OrderSyncJob struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	PlatformCode integration.PlatformCode
	StartTime    time.Time
	EndTime      time.Time
	Status       OrderSyncJobStatus
	Error        string
	StartedAt    *time.Time
	CompletedAt  *time.Time
	RetryCount   int
	MaxRetries   int
	NextRetryAt  *time.Time

	// Sync results
	TotalOrders    int
	SuccessCount   int
	FailedCount    int
	SkippedCount   int
	FailedOrderIDs []string
}

// NewOrderSyncJob creates a new order sync job
func NewOrderSyncJob(tenantID uuid.UUID, platformCode integration.PlatformCode, startTime, endTime time.Time, maxRetries int) *OrderSyncJob {
	return &OrderSyncJob{
		ID:           uuid.New(),
		TenantID:     tenantID,
		PlatformCode: platformCode,
		StartTime:    startTime,
		EndTime:      endTime,
		Status:       OrderSyncJobStatusPending,
		MaxRetries:   maxRetries,
	}
}

// Start marks the job as running
func (j *OrderSyncJob) Start() {
	now := time.Now()
	j.Status = OrderSyncJobStatusRunning
	j.StartedAt = &now
	j.Error = ""
}

// Complete marks the job as successful
func (j *OrderSyncJob) Complete(totalOrders, successCount, failedCount, skippedCount int) {
	now := time.Now()
	j.TotalOrders = totalOrders
	j.SuccessCount = successCount
	j.FailedCount = failedCount
	j.SkippedCount = skippedCount
	j.CompletedAt = &now

	if failedCount == 0 {
		j.Status = OrderSyncJobStatusSuccess
	} else if successCount > 0 {
		j.Status = OrderSyncJobStatusPartial
	} else {
		j.Status = OrderSyncJobStatusFailed
	}
}

// Fail marks the job as failed
func (j *OrderSyncJob) Fail(err string) {
	now := time.Now()
	j.Status = OrderSyncJobStatusFailed
	j.CompletedAt = &now
	j.Error = err
}

// ShouldRetry returns true if the job should be retried
func (j *OrderSyncJob) ShouldRetry() bool {
	return j.Status == OrderSyncJobStatusFailed && j.RetryCount < j.MaxRetries
}

// ScheduleRetry schedules the job for retry with exponential backoff
func (j *OrderSyncJob) ScheduleRetry(baseDelay time.Duration) {
	j.RetryCount++
	j.Status = OrderSyncJobStatusPending
	// Exponential backoff: baseDelay * 2^(retryCount-1)
	delay := baseDelay * time.Duration(1<<(j.RetryCount-1))
	if delay > 30*time.Minute {
		delay = 30 * time.Minute // Cap at 30 minutes
	}
	nextRetry := time.Now().Add(delay)
	j.NextRetryAt = &nextRetry
	j.Error = ""
}

// ---------------------------------------------------------------------------
// OrderSyncExecutor Interface
// ---------------------------------------------------------------------------

// OrderSyncExecutor executes order sync jobs
type OrderSyncExecutor interface {
	// Execute pulls orders from platform and processes them
	Execute(ctx context.Context, job *OrderSyncJob) error
}

// ---------------------------------------------------------------------------
// OrderSyncSchedulerConfig
// ---------------------------------------------------------------------------

// OrderSyncSchedulerConfig holds configuration for order sync scheduler
type OrderSyncSchedulerConfig struct {
	// Enabled indicates if the scheduler is enabled
	Enabled bool
	// MaxConcurrentJobs is the maximum number of concurrent sync jobs
	MaxConcurrentJobs int
	// JobTimeout is the maximum time a job can run
	JobTimeout time.Duration
	// RetryAttempts is the number of retry attempts for failed jobs
	RetryAttempts int
	// RetryDelay is the base delay between retries (with exponential backoff)
	RetryDelay time.Duration
	// DefaultSyncInterval is the default sync interval for tenants
	DefaultSyncInterval time.Duration
	// MinSyncInterval is the minimum allowed sync interval
	MinSyncInterval time.Duration
	// MaxSyncInterval is the maximum allowed sync interval
	MaxSyncInterval time.Duration
	// LookbackDuration is how far back to look for orders on first sync
	LookbackDuration time.Duration
}

// DefaultOrderSyncSchedulerConfig returns default configuration
func DefaultOrderSyncSchedulerConfig() OrderSyncSchedulerConfig {
	return OrderSyncSchedulerConfig{
		Enabled:             true,
		MaxConcurrentJobs:   5,
		JobTimeout:          15 * time.Minute,
		RetryAttempts:       3,
		RetryDelay:          1 * time.Minute,
		DefaultSyncInterval: 15 * time.Minute,
		MinSyncInterval:     5 * time.Minute,
		MaxSyncInterval:     60 * time.Minute,
		LookbackDuration:    24 * time.Hour,
	}
}

// Validate validates the configuration
func (c *OrderSyncSchedulerConfig) Validate() error {
	if c.MaxConcurrentJobs <= 0 {
		return ErrInvalidConfig
	}
	if c.JobTimeout <= 0 {
		return ErrInvalidConfig
	}
	if c.RetryAttempts < 0 {
		return ErrInvalidConfig
	}
	if c.MinSyncInterval <= 0 {
		return ErrInvalidConfig
	}
	if c.MaxSyncInterval < c.MinSyncInterval {
		return ErrInvalidConfig
	}
	if c.DefaultSyncInterval < c.MinSyncInterval || c.DefaultSyncInterval > c.MaxSyncInterval {
		return ErrInvalidConfig
	}
	return nil
}

// ---------------------------------------------------------------------------
// OrderSyncScheduler
// ---------------------------------------------------------------------------

// OrderSyncScheduler manages scheduled order sync jobs
type OrderSyncScheduler struct {
	config   OrderSyncSchedulerConfig
	executor OrderSyncExecutor
	logger   *zap.Logger

	jobs      chan *OrderSyncJob
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mu        sync.Mutex
	isRunning bool

	// Job history for monitoring (in-memory, limited size)
	historyMu  sync.RWMutex
	history    []*OrderSyncJob
	maxHistory int
}

// NewOrderSyncScheduler creates a new order sync scheduler
func NewOrderSyncScheduler(config OrderSyncSchedulerConfig, executor OrderSyncExecutor, logger *zap.Logger) (*OrderSyncScheduler, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &OrderSyncScheduler{
		config:     config,
		executor:   executor,
		logger:     logger,
		jobs:       make(chan *OrderSyncJob, 100),
		history:    make([]*OrderSyncJob, 0, 100),
		maxHistory: 100,
	}, nil
}

// Start starts the scheduler
func (s *OrderSyncScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return nil
	}
	s.isRunning = true
	s.mu.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	// Start worker pool
	for i := 0; i < s.config.MaxConcurrentJobs; i++ {
		s.wg.Add(1)
		go s.worker(ctx, i)
	}

	s.logger.Info("Order sync scheduler started",
		zap.Int("workers", s.config.MaxConcurrentJobs),
		zap.Duration("job_timeout", s.config.JobTimeout),
	)

	return nil
}

// Stop gracefully stops the scheduler
func (s *OrderSyncScheduler) Stop(ctx context.Context) error {
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

	// Close job channel
	close(s.jobs)

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("Order sync scheduler stopped gracefully")
		return nil
	case <-ctx.Done():
		s.logger.Warn("Order sync scheduler stop timed out")
		return ctx.Err()
	}
}

// SubmitJob submits a job for execution
func (s *OrderSyncScheduler) SubmitJob(job *OrderSyncJob) error {
	s.mu.Lock()
	if !s.isRunning {
		s.mu.Unlock()
		return ErrSchedulerNotRunning
	}
	s.mu.Unlock()

	select {
	case s.jobs <- job:
		s.logger.Debug("Order sync job submitted",
			zap.String("job_id", job.ID.String()),
			zap.String("tenant_id", job.TenantID.String()),
			zap.String("platform_code", string(job.PlatformCode)),
		)
		return nil
	default:
		return ErrJobQueueFull
	}
}

// worker processes jobs from the queue
func (s *OrderSyncScheduler) worker(ctx context.Context, workerID int) {
	defer s.wg.Done()

	s.logger.Debug("Order sync worker started", zap.Int("worker_id", workerID))

	for {
		select {
		case <-ctx.Done():
			s.logger.Debug("Order sync worker stopping", zap.Int("worker_id", workerID))
			return
		case job, ok := <-s.jobs:
			if !ok {
				s.logger.Debug("Order sync job channel closed", zap.Int("worker_id", workerID))
				return
			}
			s.processJob(ctx, job, workerID)
		}
	}
}

// processJob executes a single job
func (s *OrderSyncScheduler) processJob(ctx context.Context, job *OrderSyncJob, workerID int) {
	// Check if job is ready to run (for retries)
	if job.NextRetryAt != nil && time.Now().Before(*job.NextRetryAt) {
		// Re-queue the job
		select {
		case s.jobs <- job:
		default:
			s.logger.Warn("Failed to re-queue order sync job for retry",
				zap.String("job_id", job.ID.String()),
			)
		}
		return
	}

	job.Start()
	s.logger.Info("Processing order sync job",
		zap.Int("worker_id", workerID),
		zap.String("job_id", job.ID.String()),
		zap.String("tenant_id", job.TenantID.String()),
		zap.String("platform_code", string(job.PlatformCode)),
		zap.Time("start_time", job.StartTime),
		zap.Time("end_time", job.EndTime),
	)

	// Create context with timeout
	jobCtx, cancel := context.WithTimeout(ctx, s.config.JobTimeout)
	defer cancel()

	// Execute the job
	err := s.executor.Execute(jobCtx, job)
	if err != nil {
		job.Fail(err.Error())
		s.logger.Error("Order sync job failed",
			zap.Int("worker_id", workerID),
			zap.String("job_id", job.ID.String()),
			zap.String("tenant_id", job.TenantID.String()),
			zap.String("platform_code", string(job.PlatformCode)),
			zap.Error(err),
		)

		// Check if should retry
		if job.ShouldRetry() {
			job.ScheduleRetry(s.config.RetryDelay)
			s.logger.Info("Order sync job scheduled for retry",
				zap.String("job_id", job.ID.String()),
				zap.Int("retry_count", job.RetryCount),
				zap.Int("max_retries", job.MaxRetries),
				zap.Time("next_retry_at", *job.NextRetryAt),
			)
			// Re-submit job
			select {
			case s.jobs <- job:
			default:
				s.logger.Warn("Failed to re-queue order sync job for retry",
					zap.String("job_id", job.ID.String()),
				)
			}
		}

		// Add to history
		s.addToHistory(job)
		return
	}

	s.logger.Info("Order sync job completed",
		zap.Int("worker_id", workerID),
		zap.String("job_id", job.ID.String()),
		zap.String("tenant_id", job.TenantID.String()),
		zap.String("platform_code", string(job.PlatformCode)),
		zap.String("status", string(job.Status)),
		zap.Int("total_orders", job.TotalOrders),
		zap.Int("success_count", job.SuccessCount),
		zap.Int("failed_count", job.FailedCount),
		zap.Int("skipped_count", job.SkippedCount),
	)

	// Add to history
	s.addToHistory(job)
}

// addToHistory adds a completed job to history
func (s *OrderSyncScheduler) addToHistory(job *OrderSyncJob) {
	s.historyMu.Lock()
	defer s.historyMu.Unlock()

	// Add to front
	s.history = append([]*OrderSyncJob{job}, s.history...)

	// Trim if over limit
	if len(s.history) > s.maxHistory {
		s.history = s.history[:s.maxHistory]
	}
}

// GetJobHistory returns recent job history
func (s *OrderSyncScheduler) GetJobHistory(limit int) []*OrderSyncJob {
	s.historyMu.RLock()
	defer s.historyMu.RUnlock()

	if limit <= 0 || limit > len(s.history) {
		limit = len(s.history)
	}

	result := make([]*OrderSyncJob, limit)
	copy(result, s.history[:limit])
	return result
}

// GetJobHistoryByTenant returns job history for a specific tenant
func (s *OrderSyncScheduler) GetJobHistoryByTenant(tenantID uuid.UUID, limit int) []*OrderSyncJob {
	s.historyMu.RLock()
	defer s.historyMu.RUnlock()

	result := make([]*OrderSyncJob, 0, limit)
	for _, job := range s.history {
		if job.TenantID == tenantID {
			result = append(result, job)
			if len(result) >= limit {
				break
			}
		}
	}
	return result
}

// ScheduleSync schedules an order sync job for a tenant and platform
func (s *OrderSyncScheduler) ScheduleSync(tenantID uuid.UUID, platformCode integration.PlatformCode, startTime, endTime time.Time) error {
	job := NewOrderSyncJob(tenantID, platformCode, startTime, endTime, s.config.RetryAttempts)
	return s.SubmitJob(job)
}

// ScheduleSyncWithDefaults schedules a sync job using default time range (last sync interval)
func (s *OrderSyncScheduler) ScheduleSyncWithDefaults(tenantID uuid.UUID, platformCode integration.PlatformCode) error {
	now := time.Now()
	// Use default lookback for safety (ensures no gaps)
	startTime := now.Add(-s.config.LookbackDuration)
	return s.ScheduleSync(tenantID, platformCode, startTime, now)
}
