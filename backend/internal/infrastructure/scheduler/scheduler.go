package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// JobStatus represents the status of a scheduled job
type JobStatus string

const (
	JobStatusPending JobStatus = "PENDING"
	JobStatusRunning JobStatus = "RUNNING"
	JobStatusSuccess JobStatus = "SUCCESS"
	JobStatusFailed  JobStatus = "FAILED"
)

// ReportType represents the type of report to generate
type ReportType string

const (
	ReportTypeSalesSummary      ReportType = "SALES_SUMMARY"
	ReportTypeSalesDailyTrend   ReportType = "SALES_DAILY_TREND"
	ReportTypeInventorySummary  ReportType = "INVENTORY_SUMMARY"
	ReportTypeProfitLossMonthly ReportType = "PNL_MONTHLY"
	ReportTypeProductRanking    ReportType = "PRODUCT_RANKING"
	ReportTypeCustomerRanking   ReportType = "CUSTOMER_RANKING"
)

// AllReportTypes returns all available report types
func AllReportTypes() []ReportType {
	return []ReportType{
		ReportTypeSalesSummary,
		ReportTypeSalesDailyTrend,
		ReportTypeInventorySummary,
		ReportTypeProfitLossMonthly,
		ReportTypeProductRanking,
		ReportTypeCustomerRanking,
	}
}

// Job represents a scheduled report job
type Job struct {
	ID          uuid.UUID
	TenantID    *uuid.UUID // nil means all tenants
	ReportType  ReportType
	PeriodStart time.Time
	PeriodEnd   time.Time
	Status      JobStatus
	Error       string
	StartedAt   *time.Time
	CompletedAt *time.Time
	RetryCount  int
	MaxRetries  int
	NextRetryAt *time.Time
}

// NewJob creates a new job instance
func NewJob(tenantID *uuid.UUID, reportType ReportType, periodStart, periodEnd time.Time, maxRetries int) *Job {
	return &Job{
		ID:          uuid.New(),
		TenantID:    tenantID,
		ReportType:  reportType,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		Status:      JobStatusPending,
		MaxRetries:  maxRetries,
	}
}

// Start marks the job as running
func (j *Job) Start() {
	now := time.Now()
	j.Status = JobStatusRunning
	j.StartedAt = &now
	j.Error = ""
}

// Complete marks the job as successful
func (j *Job) Complete() {
	now := time.Now()
	j.Status = JobStatusSuccess
	j.CompletedAt = &now
}

// Fail marks the job as failed
func (j *Job) Fail(err string) {
	now := time.Now()
	j.Status = JobStatusFailed
	j.CompletedAt = &now
	j.Error = err
}

// ShouldRetry returns true if the job should be retried
func (j *Job) ShouldRetry() bool {
	return j.Status == JobStatusFailed && j.RetryCount < j.MaxRetries
}

// ScheduleRetry schedules the job for retry
func (j *Job) ScheduleRetry(delay time.Duration) {
	j.RetryCount++
	j.Status = JobStatusPending
	nextRetry := time.Now().Add(delay)
	j.NextRetryAt = &nextRetry
	j.Error = ""
}

// JobExecutor is the interface for executing report jobs
type JobExecutor interface {
	Execute(ctx context.Context, job *Job) error
}

// SchedulerConfig holds scheduler configuration
type SchedulerConfig struct {
	Enabled           bool
	MaxConcurrentJobs int
	JobTimeout        time.Duration
	RetryAttempts     int
	RetryDelay        time.Duration
}

// DefaultSchedulerConfig returns default scheduler configuration
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		Enabled:           true,
		MaxConcurrentJobs: 3,
		JobTimeout:        30 * time.Minute,
		RetryAttempts:     3,
		RetryDelay:        5 * time.Minute,
	}
}

// Scheduler manages scheduled report jobs
type Scheduler struct {
	config   SchedulerConfig
	executor JobExecutor
	logger   *zap.Logger

	jobs      chan *Job
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mu        sync.Mutex
	isRunning bool
}

// NewScheduler creates a new scheduler instance
func NewScheduler(config SchedulerConfig, executor JobExecutor, logger *zap.Logger) *Scheduler {
	return &Scheduler{
		config:   config,
		executor: executor,
		logger:   logger,
		jobs:     make(chan *Job, 100),
	}
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) error {
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

	s.logger.Info("Report scheduler started",
		zap.Int("workers", s.config.MaxConcurrentJobs),
		zap.Duration("job_timeout", s.config.JobTimeout),
	)

	return nil
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop(ctx context.Context) error {
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
		s.logger.Info("Report scheduler stopped gracefully")
		return nil
	case <-ctx.Done():
		s.logger.Warn("Report scheduler stop timed out")
		return ctx.Err()
	}
}

// SubmitJob submits a job for execution
func (s *Scheduler) SubmitJob(job *Job) error {
	s.mu.Lock()
	if !s.isRunning {
		s.mu.Unlock()
		return ErrSchedulerNotRunning
	}
	s.mu.Unlock()

	select {
	case s.jobs <- job:
		s.logger.Debug("Job submitted",
			zap.String("job_id", job.ID.String()),
			zap.String("report_type", string(job.ReportType)),
		)
		return nil
	default:
		return ErrJobQueueFull
	}
}

// worker processes jobs from the queue
func (s *Scheduler) worker(ctx context.Context, workerID int) {
	defer s.wg.Done()

	s.logger.Debug("Worker started", zap.Int("worker_id", workerID))

	for {
		select {
		case <-ctx.Done():
			s.logger.Debug("Worker stopping", zap.Int("worker_id", workerID))
			return
		case job, ok := <-s.jobs:
			if !ok {
				s.logger.Debug("Job channel closed", zap.Int("worker_id", workerID))
				return
			}
			s.processJob(ctx, job, workerID)
		}
	}
}

// processJob executes a single job
func (s *Scheduler) processJob(ctx context.Context, job *Job, workerID int) {
	// Check if job is ready to run (for retries)
	if job.NextRetryAt != nil && time.Now().Before(*job.NextRetryAt) {
		// Re-queue the job
		select {
		case s.jobs <- job:
		default:
			s.logger.Warn("Failed to re-queue job for retry",
				zap.String("job_id", job.ID.String()),
			)
		}
		return
	}

	job.Start()
	s.logger.Info("Processing job",
		zap.Int("worker_id", workerID),
		zap.String("job_id", job.ID.String()),
		zap.String("report_type", string(job.ReportType)),
	)

	// Create context with timeout
	jobCtx, cancel := context.WithTimeout(ctx, s.config.JobTimeout)
	defer cancel()

	// Execute the job
	err := s.executor.Execute(jobCtx, job)
	if err != nil {
		job.Fail(err.Error())
		s.logger.Error("Job failed",
			zap.Int("worker_id", workerID),
			zap.String("job_id", job.ID.String()),
			zap.String("report_type", string(job.ReportType)),
			zap.Error(err),
		)

		// Check if should retry
		if job.ShouldRetry() {
			job.ScheduleRetry(s.config.RetryDelay)
			s.logger.Info("Job scheduled for retry",
				zap.String("job_id", job.ID.String()),
				zap.Int("retry_count", job.RetryCount),
				zap.Int("max_retries", job.MaxRetries),
			)
			// Re-submit job
			select {
			case s.jobs <- job:
			default:
				s.logger.Warn("Failed to re-queue job for retry",
					zap.String("job_id", job.ID.String()),
				)
			}
		}
		return
	}

	job.Complete()
	s.logger.Info("Job completed successfully",
		zap.Int("worker_id", workerID),
		zap.String("job_id", job.ID.String()),
		zap.String("report_type", string(job.ReportType)),
	)
}

// ScheduleDailyReports schedules all report types for a tenant
func (s *Scheduler) ScheduleDailyReports(tenantID *uuid.UUID) error {
	now := time.Now()

	// Calculate period (yesterday)
	yesterday := now.AddDate(0, 0, -1)
	periodStart := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.Local)
	periodEnd := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 23, 59, 59, 999999999, time.Local)

	for _, reportType := range AllReportTypes() {
		job := NewJob(tenantID, reportType, periodStart, periodEnd, s.config.RetryAttempts)
		if err := s.SubmitJob(job); err != nil {
			return err
		}
	}

	return nil
}

// ScheduleReport schedules a specific report type
func (s *Scheduler) ScheduleReport(tenantID *uuid.UUID, reportType ReportType, periodStart, periodEnd time.Time) error {
	job := NewJob(tenantID, reportType, periodStart, periodEnd, s.config.RetryAttempts)
	return s.SubmitJob(job)
}
