package scheduler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// cronTickerInterval is the interval at which the cron scheduler checks for execution
const cronTickerInterval = 1 * time.Minute

// ReportCronSchedulerConfig holds configuration for the cron-based report scheduler
type ReportCronSchedulerConfig struct {
	// Enabled indicates if the cron scheduler is enabled
	Enabled bool
	// CronHour is the hour (0-23) to run the daily aggregation
	CronHour int
	// CronMinute is the minute (0-59) to run the daily aggregation
	CronMinute int
	// DailyCronSchedule is the cron expression (parsed to extract hour/minute)
	DailyCronSchedule string
	// JobTimeout is the maximum time a single report job can run
	JobTimeout time.Duration
	// MaxConcurrentJobs is the maximum number of concurrent report jobs
	MaxConcurrentJobs int
	// RetryAttempts is the number of retry attempts for failed jobs
	RetryAttempts int
	// RetryDelay is the delay between retries
	RetryDelay time.Duration
}

// DefaultReportCronSchedulerConfig returns default cron scheduler configuration
// Defaults to running at 2:00 AM daily
func DefaultReportCronSchedulerConfig() ReportCronSchedulerConfig {
	return ReportCronSchedulerConfig{
		Enabled:           true,
		CronHour:          2, // 2 AM
		CronMinute:        0, // 0 minutes
		DailyCronSchedule: "0 2 * * *",
		JobTimeout:        30 * time.Minute,
		MaxConcurrentJobs: 3,
		RetryAttempts:     3,
		RetryDelay:        5 * time.Minute,
	}
}

// ParseCronSchedule parses a cron expression "minute hour * * *" to extract hour and minute
// Returns defaults (2:00) if parsing fails or expression is empty
func ParseCronSchedule(cronExpr string) (hour, minute int, err error) {
	// Default values
	hour = 2
	minute = 0

	if cronExpr == "" {
		return hour, minute, nil
	}

	// Use strings.Fields for simple whitespace splitting
	parts := strings.Fields(cronExpr)

	if len(parts) < 2 {
		return hour, minute, nil
	}

	// Parse minute
	if parts[0] != "*" {
		if val, parseErr := parseIntOrDefault(parts[0], 0); parseErr == nil {
			minute = val
		}
	}

	// Parse hour
	if parts[1] != "*" {
		if val, parseErr := parseIntOrDefault(parts[1], 2); parseErr == nil {
			hour = val
		}
	}

	// Validate ranges
	if minute < 0 || minute > 59 {
		return 2, 0, fmt.Errorf("minute must be 0-59, got %d", minute)
	}
	if hour < 0 || hour > 23 {
		return 2, 0, fmt.Errorf("hour must be 0-23, got %d", hour)
	}

	return hour, minute, nil
}

// parseIntOrDefault parses an int string or returns default
func parseIntOrDefault(s string, defaultVal int) (int, error) {
	if s == "" || s == "*" {
		return defaultVal, nil
	}
	var val int
	for _, c := range s {
		if c < '0' || c > '9' {
			return defaultVal, ErrInvalidConfig
		}
		val = val*10 + int(c-'0')
	}
	return val, nil
}

// SchedulerJobRecord represents a record of a scheduled job execution
type SchedulerJobRecord struct {
	ID          uuid.UUID  `gorm:"column:id;type:uuid;primaryKey"`
	TenantID    *uuid.UUID `gorm:"column:tenant_id;type:uuid"`
	ReportType  string     `gorm:"column:report_type;size:50;not null"`
	Status      string     `gorm:"column:last_run_status;size:20"`
	Error       string     `gorm:"column:last_error;type:text"`
	StartedAt   *time.Time `gorm:"column:last_run_at"`
	CompletedAt *time.Time `gorm:"column:completed_at"`
	NextRunAt   *time.Time `gorm:"column:next_run_at"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at"`
}

// TableName returns the table name for GORM
func (SchedulerJobRecord) TableName() string {
	return "report_scheduler_jobs"
}

// SchedulerJobRepository handles persistence of scheduler job records
type SchedulerJobRepository struct {
	db *gorm.DB
}

// NewSchedulerJobRepository creates a new SchedulerJobRepository
func NewSchedulerJobRepository(db *gorm.DB) *SchedulerJobRepository {
	return &SchedulerJobRepository{db: db}
}

// RecordJobStart records the start of a job execution
func (r *SchedulerJobRepository) RecordJobStart(ctx context.Context, tenantID *uuid.UUID, reportType string) (uuid.UUID, error) {
	now := time.Now()
	record := &SchedulerJobRecord{
		ID:         uuid.New(),
		TenantID:   tenantID,
		ReportType: reportType,
		Status:     string(JobStatusRunning),
		StartedAt:  &now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := r.db.WithContext(ctx).Create(record).Error; err != nil {
		return uuid.Nil, err
	}
	return record.ID, nil
}

// RecordJobComplete records the completion of a job
func (r *SchedulerJobRepository) RecordJobComplete(ctx context.Context, jobID uuid.UUID, success bool, errMsg string) error {
	now := time.Now()
	status := string(JobStatusSuccess)
	if !success {
		status = string(JobStatusFailed)
	}
	return r.db.WithContext(ctx).
		Model(&SchedulerJobRecord{}).
		Where("id = ?", jobID).
		Updates(map[string]any{
			"last_run_status": status,
			"last_error":      errMsg,
			"completed_at":    now,
			"updated_at":      now,
		}).Error
}

// GetLastJobStatus gets the last job status for a report type
func (r *SchedulerJobRepository) GetLastJobStatus(ctx context.Context, tenantID *uuid.UUID, reportType string) (*SchedulerJobRecord, error) {
	var record SchedulerJobRecord
	query := r.db.WithContext(ctx).Where("report_type = ?", reportType)
	if tenantID != nil {
		query = query.Where("tenant_id = ?", *tenantID)
	} else {
		query = query.Where("tenant_id IS NULL")
	}
	if err := query.Order("last_run_at DESC").First(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

// ReportCronScheduler implements cron-based scheduling for daily report aggregation
type ReportCronScheduler struct {
	config     ReportCronSchedulerConfig
	executor   JobExecutor
	tenantRepo identity.TenantRepository
	jobRepo    *SchedulerJobRepository
	logger     *zap.Logger
	scheduler  *Scheduler

	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mu        sync.Mutex
	isRunning bool

	// Last execution tracking
	lastRunAt *time.Time
	nextRunAt *time.Time
}

// NewReportCronScheduler creates a new cron-based report scheduler
func NewReportCronScheduler(
	config ReportCronSchedulerConfig,
	executor JobExecutor,
	tenantRepo identity.TenantRepository,
	jobRepo *SchedulerJobRepository,
	logger *zap.Logger,
) *ReportCronScheduler {
	schedulerConfig := SchedulerConfig{
		Enabled:           config.Enabled,
		MaxConcurrentJobs: config.MaxConcurrentJobs,
		JobTimeout:        config.JobTimeout,
		RetryAttempts:     config.RetryAttempts,
		RetryDelay:        config.RetryDelay,
	}
	scheduler := NewScheduler(schedulerConfig, executor, logger)

	return &ReportCronScheduler{
		config:     config,
		executor:   executor,
		tenantRepo: tenantRepo,
		jobRepo:    jobRepo,
		logger:     logger,
		scheduler:  scheduler,
	}
}

// Start starts the cron scheduler
func (s *ReportCronScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return nil
	}
	s.isRunning = true
	s.mu.Unlock()

	// Start the underlying job scheduler
	if err := s.scheduler.Start(ctx); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	// Calculate next run time
	s.calculateNextRunTime()

	// Start the cron ticker
	s.wg.Add(1)
	go s.cronLoop(ctx)

	s.logger.Info("Report cron scheduler started",
		zap.Int("cron_hour", s.config.CronHour),
		zap.Int("cron_minute", s.config.CronMinute),
		zap.Timep("next_run_at", s.nextRunAt),
	)

	return nil
}

// Stop stops the cron scheduler
func (s *ReportCronScheduler) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.isRunning {
		s.mu.Unlock()
		return nil
	}
	s.isRunning = false
	s.mu.Unlock()

	// Cancel the cron loop
	if s.cancel != nil {
		s.cancel()
	}

	// Wait for cron loop to finish
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Now stop the underlying scheduler
		if err := s.scheduler.Stop(ctx); err != nil {
			s.logger.Warn("Error stopping underlying scheduler", zap.Error(err))
		}
		s.logger.Info("Report cron scheduler stopped")
		return nil
	case <-ctx.Done():
		s.logger.Warn("Report cron scheduler stop timed out")
		return ctx.Err()
	}
}

// cronLoop runs the main cron loop
func (s *ReportCronScheduler) cronLoop(ctx context.Context) {
	defer s.wg.Done()

	// Use a ticker that checks every minute for cron execution
	ticker := time.NewTicker(cronTickerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			if s.shouldRun(now) {
				s.runDailyAggregation(ctx)
				s.calculateNextRunTime()
			}
		}
	}
}

// shouldRun checks if the cron should run at the given time
func (s *ReportCronScheduler) shouldRun(now time.Time) bool {
	return now.Hour() == s.config.CronHour && now.Minute() == s.config.CronMinute
}

// calculateNextRunTime calculates the next run time
func (s *ReportCronScheduler) calculateNextRunTime() {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), s.config.CronHour, s.config.CronMinute, 0, 0, now.Location())

	// If we've already passed today's run time, schedule for tomorrow
	if now.After(next) {
		next = next.AddDate(0, 0, 1)
	}

	s.mu.Lock()
	s.nextRunAt = &next
	s.mu.Unlock()
}

// runDailyAggregation runs the daily aggregation for all active tenants
func (s *ReportCronScheduler) runDailyAggregation(ctx context.Context) {
	s.logger.Info("Starting daily report aggregation")

	now := time.Now()
	s.mu.Lock()
	s.lastRunAt = &now
	s.mu.Unlock()

	// Get all active tenants
	tenants, err := s.tenantRepo.FindActive(ctx, shared.Filter{})
	if err != nil {
		s.logger.Error("Failed to fetch active tenants for report aggregation", zap.Error(err))
		return
	}

	s.logger.Info("Scheduling report aggregation for tenants", zap.Int("tenant_count", len(tenants)))

	// Calculate period (yesterday)
	yesterday := now.AddDate(0, 0, -1)
	periodStart := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.Local)
	periodEnd := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 23, 59, 59, 999999999, time.Local)

	// Schedule jobs for each tenant
	for _, tenant := range tenants {
		tenantID := tenant.ID
		for _, reportType := range AllReportTypes() {
			// Record job start
			var jobID uuid.UUID
			if s.jobRepo != nil {
				var recordErr error
				jobID, recordErr = s.jobRepo.RecordJobStart(ctx, &tenantID, string(reportType))
				if recordErr != nil {
					s.logger.Warn("Failed to record job start",
						zap.String("tenant_id", tenantID.String()),
						zap.String("report_type", string(reportType)),
						zap.Error(recordErr),
					)
				}
			}

			// Create and submit job
			job := NewJob(&tenantID, reportType, periodStart, periodEnd, s.config.RetryAttempts)
			if err := s.scheduler.SubmitJob(job); err != nil {
				s.logger.Error("Failed to submit report job",
					zap.String("tenant_id", tenantID.String()),
					zap.String("report_type", string(reportType)),
					zap.Error(err),
				)
				// Record failure
				if s.jobRepo != nil && jobID != uuid.Nil {
					_ = s.jobRepo.RecordJobComplete(ctx, jobID, false, err.Error())
				}
				continue
			}

			s.logger.Debug("Scheduled report job",
				zap.String("tenant_id", tenantID.String()),
				zap.String("report_type", string(reportType)),
			)
		}
	}

	s.logger.Info("Daily report aggregation jobs scheduled",
		zap.Int("tenant_count", len(tenants)),
		zap.Int("report_types", len(AllReportTypes())),
	)
}

// TriggerManualRun triggers a manual run of the daily aggregation
// Note: Uses background context to avoid premature cancellation when HTTP request completes
func (s *ReportCronScheduler) TriggerManualRun(ctx context.Context) error {
	s.mu.Lock()
	if !s.isRunning {
		s.mu.Unlock()
		return ErrSchedulerNotRunning
	}
	s.mu.Unlock()

	// Use background context to prevent premature cancellation when HTTP request completes
	go s.runDailyAggregation(context.Background())
	return nil
}

// TriggerTenantAggregation triggers aggregation for a specific tenant
func (s *ReportCronScheduler) TriggerTenantAggregation(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) error {
	s.mu.Lock()
	if !s.isRunning {
		s.mu.Unlock()
		return ErrSchedulerNotRunning
	}
	s.mu.Unlock()

	for _, reportType := range AllReportTypes() {
		job := NewJob(&tenantID, reportType, startDate, endDate, s.config.RetryAttempts)
		if err := s.scheduler.SubmitJob(job); err != nil {
			return err
		}
	}
	return nil
}

// GetStatus returns the current status of the cron scheduler
func (s *ReportCronScheduler) GetStatus() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	return map[string]any{
		"enabled":       s.config.Enabled,
		"is_running":    s.isRunning,
		"cron_hour":     s.config.CronHour,
		"cron_minute":   s.config.CronMinute,
		"cron_schedule": "Daily",
		"last_run_at":   s.lastRunAt,
		"next_run_at":   s.nextRunAt,
		"report_types":  AllReportTypes(),
	}
}

// GetNextRunAt returns when the next scheduled run will occur
func (s *ReportCronScheduler) GetNextRunAt() *time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.nextRunAt
}

// GetLastRunAt returns when the last run occurred
func (s *ReportCronScheduler) GetLastRunAt() *time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastRunAt
}
