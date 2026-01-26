package scheduler

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/erp/backend/internal/domain/integration"
)

// ---------------------------------------------------------------------------
// Test Helpers
// ---------------------------------------------------------------------------

func newTestLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

// ---------------------------------------------------------------------------
// OrderSyncJob Tests
// ---------------------------------------------------------------------------

func TestNewOrderSyncJob(t *testing.T) {
	tenantID := uuid.New()
	platformCode := integration.PlatformCodeTaobao
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	job := NewOrderSyncJob(tenantID, platformCode, startTime, endTime, 3)

	assert.NotEqual(t, uuid.Nil, job.ID)
	assert.Equal(t, tenantID, job.TenantID)
	assert.Equal(t, platformCode, job.PlatformCode)
	assert.Equal(t, startTime, job.StartTime)
	assert.Equal(t, endTime, job.EndTime)
	assert.Equal(t, OrderSyncJobStatusPending, job.Status)
	assert.Equal(t, 3, job.MaxRetries)
	assert.Nil(t, job.StartedAt)
	assert.Nil(t, job.CompletedAt)
}

func TestOrderSyncJob_Start(t *testing.T) {
	job := NewOrderSyncJob(uuid.New(), integration.PlatformCodeTaobao, time.Now(), time.Now(), 3)
	job.Error = "previous error"

	job.Start()

	assert.Equal(t, OrderSyncJobStatusRunning, job.Status)
	assert.NotNil(t, job.StartedAt)
	assert.Empty(t, job.Error)
}

func TestOrderSyncJob_Complete_AllSuccess(t *testing.T) {
	job := NewOrderSyncJob(uuid.New(), integration.PlatformCodeTaobao, time.Now(), time.Now(), 3)
	job.Start()

	job.Complete(100, 100, 0, 0)

	assert.Equal(t, OrderSyncJobStatusSuccess, job.Status)
	assert.NotNil(t, job.CompletedAt)
	assert.Equal(t, 100, job.TotalOrders)
	assert.Equal(t, 100, job.SuccessCount)
	assert.Equal(t, 0, job.FailedCount)
}

func TestOrderSyncJob_Complete_Partial(t *testing.T) {
	job := NewOrderSyncJob(uuid.New(), integration.PlatformCodeTaobao, time.Now(), time.Now(), 3)
	job.Start()

	job.Complete(100, 80, 20, 0)

	assert.Equal(t, OrderSyncJobStatusPartial, job.Status)
	assert.Equal(t, 80, job.SuccessCount)
	assert.Equal(t, 20, job.FailedCount)
}

func TestOrderSyncJob_Complete_AllFailed(t *testing.T) {
	job := NewOrderSyncJob(uuid.New(), integration.PlatformCodeTaobao, time.Now(), time.Now(), 3)
	job.Start()

	job.Complete(100, 0, 100, 0)

	assert.Equal(t, OrderSyncJobStatusFailed, job.Status)
}

func TestOrderSyncJob_Fail(t *testing.T) {
	job := NewOrderSyncJob(uuid.New(), integration.PlatformCodeTaobao, time.Now(), time.Now(), 3)
	job.Start()

	job.Fail("connection timeout")

	assert.Equal(t, OrderSyncJobStatusFailed, job.Status)
	assert.NotNil(t, job.CompletedAt)
	assert.Equal(t, "connection timeout", job.Error)
}

func TestOrderSyncJob_ShouldRetry(t *testing.T) {
	tests := []struct {
		name       string
		status     OrderSyncJobStatus
		retryCount int
		maxRetries int
		expected   bool
	}{
		{"Failed with retries available", OrderSyncJobStatusFailed, 0, 3, true},
		{"Failed max retries reached", OrderSyncJobStatusFailed, 3, 3, false},
		{"Success should not retry", OrderSyncJobStatusSuccess, 0, 3, false},
		{"Partial should not retry", OrderSyncJobStatusPartial, 0, 3, false},
		{"Running should not retry", OrderSyncJobStatusRunning, 0, 3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &OrderSyncJob{
				Status:     tt.status,
				RetryCount: tt.retryCount,
				MaxRetries: tt.maxRetries,
			}
			assert.Equal(t, tt.expected, job.ShouldRetry())
		})
	}
}

func TestOrderSyncJob_ScheduleRetry_ExponentialBackoff(t *testing.T) {
	job := NewOrderSyncJob(uuid.New(), integration.PlatformCodeTaobao, time.Now(), time.Now(), 5)
	job.Status = OrderSyncJobStatusFailed
	baseDelay := time.Minute

	// First retry: 1 minute
	job.ScheduleRetry(baseDelay)
	assert.Equal(t, 1, job.RetryCount)
	assert.Equal(t, OrderSyncJobStatusPending, job.Status)
	assert.NotNil(t, job.NextRetryAt)
	firstDelay := time.Until(*job.NextRetryAt)
	assert.True(t, firstDelay > 50*time.Second && firstDelay <= time.Minute+time.Second)

	// Second retry: 2 minutes
	job.Status = OrderSyncJobStatusFailed
	job.ScheduleRetry(baseDelay)
	assert.Equal(t, 2, job.RetryCount)
	secondDelay := time.Until(*job.NextRetryAt)
	assert.True(t, secondDelay > 110*time.Second && secondDelay <= 2*time.Minute+time.Second)

	// Third retry: 4 minutes
	job.Status = OrderSyncJobStatusFailed
	job.ScheduleRetry(baseDelay)
	assert.Equal(t, 3, job.RetryCount)
	thirdDelay := time.Until(*job.NextRetryAt)
	assert.True(t, thirdDelay > 230*time.Second && thirdDelay <= 4*time.Minute+time.Second)
}

// ---------------------------------------------------------------------------
// OrderSyncSchedulerConfig Tests
// ---------------------------------------------------------------------------

func TestOrderSyncSchedulerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  OrderSyncSchedulerConfig
		wantErr bool
	}{
		{
			name:    "Valid default config",
			config:  DefaultOrderSyncSchedulerConfig(),
			wantErr: false,
		},
		{
			name: "Invalid max concurrent jobs",
			config: OrderSyncSchedulerConfig{
				MaxConcurrentJobs:   0,
				JobTimeout:          time.Minute,
				MinSyncInterval:     time.Minute,
				MaxSyncInterval:     time.Hour,
				DefaultSyncInterval: 15 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "Invalid job timeout",
			config: OrderSyncSchedulerConfig{
				MaxConcurrentJobs:   3,
				JobTimeout:          0,
				MinSyncInterval:     time.Minute,
				MaxSyncInterval:     time.Hour,
				DefaultSyncInterval: 15 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "Invalid sync interval range",
			config: OrderSyncSchedulerConfig{
				MaxConcurrentJobs:   3,
				JobTimeout:          time.Minute,
				MinSyncInterval:     time.Hour, // Min > Max
				MaxSyncInterval:     time.Minute,
				DefaultSyncInterval: 15 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "Default interval outside range",
			config: OrderSyncSchedulerConfig{
				MaxConcurrentJobs:   3,
				JobTimeout:          time.Minute,
				MinSyncInterval:     10 * time.Minute,
				MaxSyncInterval:     time.Hour,
				DefaultSyncInterval: 5 * time.Minute, // Less than min
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// OrderSyncScheduler Tests
// ---------------------------------------------------------------------------

// mockOrderSyncExecutor implements OrderSyncExecutor for testing
type mockOrderSyncExecutor struct {
	executeFunc func(ctx context.Context, job *OrderSyncJob) error
	execCount   int32
}

func (m *mockOrderSyncExecutor) Execute(ctx context.Context, job *OrderSyncJob) error {
	atomic.AddInt32(&m.execCount, 1)
	if m.executeFunc != nil {
		return m.executeFunc(ctx, job)
	}
	job.Complete(10, 10, 0, 0)
	return nil
}

func TestNewOrderSyncScheduler(t *testing.T) {
	config := DefaultOrderSyncSchedulerConfig()
	executor := &mockOrderSyncExecutor{}
	logger := newTestLogger()

	scheduler, err := NewOrderSyncScheduler(config, executor, logger)

	require.NoError(t, err)
	assert.NotNil(t, scheduler)
}

func TestNewOrderSyncScheduler_InvalidConfig(t *testing.T) {
	config := OrderSyncSchedulerConfig{MaxConcurrentJobs: 0}
	executor := &mockOrderSyncExecutor{}
	logger := newTestLogger()

	scheduler, err := NewOrderSyncScheduler(config, executor, logger)

	assert.Error(t, err)
	assert.Nil(t, scheduler)
}

func TestOrderSyncScheduler_StartStop(t *testing.T) {
	config := DefaultOrderSyncSchedulerConfig()
	executor := &mockOrderSyncExecutor{}
	logger := newTestLogger()

	scheduler, err := NewOrderSyncScheduler(config, executor, logger)
	require.NoError(t, err)

	ctx := context.Background()

	// Start scheduler
	err = scheduler.Start(ctx)
	require.NoError(t, err)

	// Start again should be idempotent
	err = scheduler.Start(ctx)
	require.NoError(t, err)

	// Stop scheduler
	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = scheduler.Stop(stopCtx)
	require.NoError(t, err)

	// Stop again should be idempotent
	err = scheduler.Stop(stopCtx)
	require.NoError(t, err)
}

func TestOrderSyncScheduler_SubmitJob_NotRunning(t *testing.T) {
	config := DefaultOrderSyncSchedulerConfig()
	executor := &mockOrderSyncExecutor{}
	logger := newTestLogger()

	scheduler, err := NewOrderSyncScheduler(config, executor, logger)
	require.NoError(t, err)

	job := NewOrderSyncJob(uuid.New(), integration.PlatformCodeTaobao, time.Now(), time.Now(), 3)
	err = scheduler.SubmitJob(job)

	assert.Equal(t, ErrSchedulerNotRunning, err)
}

func TestOrderSyncScheduler_SubmitJob_Success(t *testing.T) {
	config := DefaultOrderSyncSchedulerConfig()
	executor := &mockOrderSyncExecutor{}
	logger := newTestLogger()

	scheduler, err := NewOrderSyncScheduler(config, executor, logger)
	require.NoError(t, err)

	ctx := context.Background()
	err = scheduler.Start(ctx)
	require.NoError(t, err)

	job := NewOrderSyncJob(uuid.New(), integration.PlatformCodeTaobao, time.Now(), time.Now(), 3)
	err = scheduler.SubmitJob(job)
	require.NoError(t, err)

	// Wait for job to be processed
	time.Sleep(100 * time.Millisecond)

	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = scheduler.Stop(stopCtx)
	require.NoError(t, err)

	// Check executor was called
	assert.Equal(t, int32(1), atomic.LoadInt32(&executor.execCount))
}

func TestOrderSyncScheduler_JobRetry(t *testing.T) {
	config := DefaultOrderSyncSchedulerConfig()
	config.RetryDelay = 10 * time.Millisecond // Short delay for test
	config.JobTimeout = time.Minute

	callCount := int32(0)
	executor := &mockOrderSyncExecutor{
		executeFunc: func(ctx context.Context, job *OrderSyncJob) error {
			count := atomic.AddInt32(&callCount, 1)
			if count < 3 {
				return errors.New("temporary failure")
			}
			job.Complete(10, 10, 0, 0)
			return nil
		},
	}
	logger := newTestLogger()

	scheduler, err := NewOrderSyncScheduler(config, executor, logger)
	require.NoError(t, err)

	ctx := context.Background()
	err = scheduler.Start(ctx)
	require.NoError(t, err)

	job := NewOrderSyncJob(uuid.New(), integration.PlatformCodeTaobao, time.Now(), time.Now(), 5)
	err = scheduler.SubmitJob(job)
	require.NoError(t, err)

	// Wait for retries
	time.Sleep(500 * time.Millisecond)

	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = scheduler.Stop(stopCtx)
	require.NoError(t, err)

	// Should have been called 3 times (2 failures + 1 success)
	assert.GreaterOrEqual(t, atomic.LoadInt32(&callCount), int32(3))
}

func TestOrderSyncScheduler_ScheduleSync(t *testing.T) {
	config := DefaultOrderSyncSchedulerConfig()
	executor := &mockOrderSyncExecutor{}
	logger := newTestLogger()

	scheduler, err := NewOrderSyncScheduler(config, executor, logger)
	require.NoError(t, err)

	ctx := context.Background()
	err = scheduler.Start(ctx)
	require.NoError(t, err)

	tenantID := uuid.New()
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	err = scheduler.ScheduleSync(tenantID, integration.PlatformCodeTaobao, startTime, endTime)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = scheduler.Stop(stopCtx)
	require.NoError(t, err)

	assert.Equal(t, int32(1), atomic.LoadInt32(&executor.execCount))
}

func TestOrderSyncScheduler_GetJobHistory(t *testing.T) {
	config := DefaultOrderSyncSchedulerConfig()
	executor := &mockOrderSyncExecutor{}
	logger := newTestLogger()

	scheduler, err := NewOrderSyncScheduler(config, executor, logger)
	require.NoError(t, err)

	ctx := context.Background()
	err = scheduler.Start(ctx)
	require.NoError(t, err)

	// Submit multiple jobs
	for i := 0; i < 5; i++ {
		job := NewOrderSyncJob(uuid.New(), integration.PlatformCodeTaobao, time.Now(), time.Now(), 3)
		err = scheduler.SubmitJob(job)
		require.NoError(t, err)
	}

	// Wait for jobs to complete
	time.Sleep(200 * time.Millisecond)

	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = scheduler.Stop(stopCtx)
	require.NoError(t, err)

	// Get history
	history := scheduler.GetJobHistory(10)
	assert.Len(t, history, 5)

	// Get limited history
	limitedHistory := scheduler.GetJobHistory(3)
	assert.Len(t, limitedHistory, 3)
}

func TestOrderSyncScheduler_GetJobHistoryByTenant(t *testing.T) {
	config := DefaultOrderSyncSchedulerConfig()
	executor := &mockOrderSyncExecutor{}
	logger := newTestLogger()

	scheduler, err := NewOrderSyncScheduler(config, executor, logger)
	require.NoError(t, err)

	ctx := context.Background()
	err = scheduler.Start(ctx)
	require.NoError(t, err)

	tenantA := uuid.New()
	tenantB := uuid.New()

	// Submit jobs for tenant A
	for i := 0; i < 3; i++ {
		job := NewOrderSyncJob(tenantA, integration.PlatformCodeTaobao, time.Now(), time.Now(), 3)
		err = scheduler.SubmitJob(job)
		require.NoError(t, err)
	}

	// Submit jobs for tenant B
	for i := 0; i < 2; i++ {
		job := NewOrderSyncJob(tenantB, integration.PlatformCodeDouyin, time.Now(), time.Now(), 3)
		err = scheduler.SubmitJob(job)
		require.NoError(t, err)
	}

	time.Sleep(200 * time.Millisecond)

	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = scheduler.Stop(stopCtx)
	require.NoError(t, err)

	// Get history by tenant
	historyA := scheduler.GetJobHistoryByTenant(tenantA, 10)
	assert.Len(t, historyA, 3)

	historyB := scheduler.GetJobHistoryByTenant(tenantB, 10)
	assert.Len(t, historyB, 2)
}

// ---------------------------------------------------------------------------
// OrderSyncCronTrigger Tests
// ---------------------------------------------------------------------------

// mockOrderSyncConfigProvider implements OrderSyncConfigProvider for testing
type mockOrderSyncConfigProvider struct {
	configs       []integration.OrderSyncConfig
	lastSyncTimes map[string]*time.Time
}

func (m *mockOrderSyncConfigProvider) GetEnabledConfigs(ctx context.Context) ([]integration.OrderSyncConfig, error) {
	result := make([]integration.OrderSyncConfig, 0)
	for _, cfg := range m.configs {
		if cfg.IsEnabled {
			result = append(result, cfg)
		}
	}
	return result, nil
}

func (m *mockOrderSyncConfigProvider) GetConfigByTenantAndPlatform(ctx context.Context, tenantID uuid.UUID, platformCode integration.PlatformCode) (*integration.OrderSyncConfig, error) {
	for i := range m.configs {
		if m.configs[i].TenantID == tenantID && m.configs[i].PlatformCode == platformCode {
			return &m.configs[i], nil
		}
	}
	return nil, ErrOrderSyncConfigNotFound
}

func (m *mockOrderSyncConfigProvider) GetLastSyncTime(ctx context.Context, tenantID uuid.UUID, platformCode integration.PlatformCode) (*time.Time, error) {
	key := tenantID.String() + ":" + string(platformCode)
	if t, ok := m.lastSyncTimes[key]; ok {
		return t, nil
	}
	return nil, nil
}

func (m *mockOrderSyncConfigProvider) UpdateLastSyncTime(ctx context.Context, tenantID uuid.UUID, platformCode integration.PlatformCode, syncTime time.Time) error {
	if m.lastSyncTimes == nil {
		m.lastSyncTimes = make(map[string]*time.Time)
	}
	key := tenantID.String() + ":" + string(platformCode)
	m.lastSyncTimes[key] = &syncTime
	return nil
}

func TestNewOrderSyncCronTrigger(t *testing.T) {
	config := DefaultOrderSyncCronTriggerConfig()
	executor := &mockOrderSyncExecutor{}
	logger := newTestLogger()

	schedulerConfig := DefaultOrderSyncSchedulerConfig()
	scheduler, err := NewOrderSyncScheduler(schedulerConfig, executor, logger)
	require.NoError(t, err)

	configProvider := &mockOrderSyncConfigProvider{}

	trigger := NewOrderSyncCronTrigger(config, scheduler, configProvider, logger)

	assert.NotNil(t, trigger)
}

func TestOrderSyncCronTrigger_StartStop(t *testing.T) {
	config := DefaultOrderSyncCronTriggerConfig()
	executor := &mockOrderSyncExecutor{}
	logger := newTestLogger()

	schedulerConfig := DefaultOrderSyncSchedulerConfig()
	scheduler, err := NewOrderSyncScheduler(schedulerConfig, executor, logger)
	require.NoError(t, err)

	configProvider := &mockOrderSyncConfigProvider{}
	trigger := NewOrderSyncCronTrigger(config, scheduler, configProvider, logger)

	ctx := context.Background()

	// Start scheduler first
	err = scheduler.Start(ctx)
	require.NoError(t, err)

	// Start trigger
	err = trigger.Start(ctx)
	require.NoError(t, err)

	// Start again should be idempotent
	err = trigger.Start(ctx)
	require.NoError(t, err)

	// Stop trigger
	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = trigger.Stop(stopCtx)
	require.NoError(t, err)

	// Stop scheduler
	err = scheduler.Stop(stopCtx)
	require.NoError(t, err)
}

func TestOrderSyncCronTrigger_TriggerManualSync(t *testing.T) {
	config := DefaultOrderSyncCronTriggerConfig()
	executor := &mockOrderSyncExecutor{}
	logger := newTestLogger()

	schedulerConfig := DefaultOrderSyncSchedulerConfig()
	scheduler, err := NewOrderSyncScheduler(schedulerConfig, executor, logger)
	require.NoError(t, err)

	configProvider := &mockOrderSyncConfigProvider{}
	trigger := NewOrderSyncCronTrigger(config, scheduler, configProvider, logger)

	ctx := context.Background()
	err = scheduler.Start(ctx)
	require.NoError(t, err)

	tenantID := uuid.New()
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	err = trigger.TriggerManualSync(ctx, tenantID, integration.PlatformCodeTaobao, startTime, endTime)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = scheduler.Stop(stopCtx)
	require.NoError(t, err)

	assert.Equal(t, int32(1), atomic.LoadInt32(&executor.execCount))
}

func TestOrderSyncCronTrigger_TriggerManualSync_InvalidTimeRange(t *testing.T) {
	config := DefaultOrderSyncCronTriggerConfig()
	executor := &mockOrderSyncExecutor{}
	logger := newTestLogger()

	schedulerConfig := DefaultOrderSyncSchedulerConfig()
	scheduler, err := NewOrderSyncScheduler(schedulerConfig, executor, logger)
	require.NoError(t, err)

	configProvider := &mockOrderSyncConfigProvider{}
	trigger := NewOrderSyncCronTrigger(config, scheduler, configProvider, logger)

	ctx := context.Background()
	err = scheduler.Start(ctx)
	require.NoError(t, err)
	defer func() {
		stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		scheduler.Stop(stopCtx)
	}()

	tenantID := uuid.New()

	// Start time after end time
	err = trigger.TriggerManualSync(ctx, tenantID, integration.PlatformCodeTaobao, time.Now(), time.Now().Add(-1*time.Hour))
	assert.Equal(t, ErrOrderSyncInvalidTimeRange, err)

	// Time range too large (> 7 days)
	err = trigger.TriggerManualSync(ctx, tenantID, integration.PlatformCodeTaobao, time.Now().Add(-8*24*time.Hour), time.Now())
	assert.Equal(t, ErrOrderSyncInvalidTimeRange, err)
}

func TestOrderSyncCronTrigger_TriggerManualSyncForAllPlatforms(t *testing.T) {
	config := DefaultOrderSyncCronTriggerConfig()
	executor := &mockOrderSyncExecutor{}
	logger := newTestLogger()

	schedulerConfig := DefaultOrderSyncSchedulerConfig()
	scheduler, err := NewOrderSyncScheduler(schedulerConfig, executor, logger)
	require.NoError(t, err)

	tenantID := uuid.New()
	configProvider := &mockOrderSyncConfigProvider{
		configs: []integration.OrderSyncConfig{
			{
				TenantID:            tenantID,
				PlatformCode:        integration.PlatformCodeTaobao,
				IsEnabled:           true,
				SyncIntervalMinutes: 15,
			},
			{
				TenantID:            tenantID,
				PlatformCode:        integration.PlatformCodeDouyin,
				IsEnabled:           true,
				SyncIntervalMinutes: 15,
			},
		},
	}

	trigger := NewOrderSyncCronTrigger(config, scheduler, configProvider, logger)

	ctx := context.Background()
	err = scheduler.Start(ctx)
	require.NoError(t, err)

	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	err = trigger.TriggerManualSyncForAllPlatforms(ctx, tenantID, startTime, endTime)
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = scheduler.Stop(stopCtx)
	require.NoError(t, err)

	// Should have executed 2 jobs (one per platform)
	assert.Equal(t, int32(2), atomic.LoadInt32(&executor.execCount))
}

func TestOrderSyncCronTrigger_TriggerManualSyncForAllPlatforms_NoPlatforms(t *testing.T) {
	config := DefaultOrderSyncCronTriggerConfig()
	executor := &mockOrderSyncExecutor{}
	logger := newTestLogger()

	schedulerConfig := DefaultOrderSyncSchedulerConfig()
	scheduler, err := NewOrderSyncScheduler(schedulerConfig, executor, logger)
	require.NoError(t, err)

	configProvider := &mockOrderSyncConfigProvider{
		configs: []integration.OrderSyncConfig{}, // No configs
	}

	trigger := NewOrderSyncCronTrigger(config, scheduler, configProvider, logger)

	ctx := context.Background()
	err = scheduler.Start(ctx)
	require.NoError(t, err)
	defer func() {
		stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		scheduler.Stop(stopCtx)
	}()

	tenantID := uuid.New()
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	err = trigger.TriggerManualSyncForAllPlatforms(ctx, tenantID, startTime, endTime)
	assert.Equal(t, ErrOrderSyncNoEnabledPlatforms, err)
}

func TestOrderSyncCronTrigger_GetSchedulerStats(t *testing.T) {
	config := DefaultOrderSyncCronTriggerConfig()
	executor := &mockOrderSyncExecutor{}
	logger := newTestLogger()

	schedulerConfig := DefaultOrderSyncSchedulerConfig()
	scheduler, err := NewOrderSyncScheduler(schedulerConfig, executor, logger)
	require.NoError(t, err)

	configProvider := &mockOrderSyncConfigProvider{}
	trigger := NewOrderSyncCronTrigger(config, scheduler, configProvider, logger)

	stats := trigger.GetSchedulerStats()

	assert.Contains(t, stats, "is_running")
	assert.Contains(t, stats, "check_interval")
	assert.Contains(t, stats, "tracked_configs")
	assert.Contains(t, stats, "last_scheduled")
}

// ---------------------------------------------------------------------------
// Error Tests
// ---------------------------------------------------------------------------

func TestErrors(t *testing.T) {
	// Ensure all error variables are defined
	assert.NotNil(t, ErrSchedulerNotRunning)
	assert.NotNil(t, ErrJobQueueFull)
	assert.NotNil(t, ErrInvalidConfig)
	assert.NotNil(t, ErrOrderSyncFailed)
	assert.NotNil(t, ErrOrderSyncTimeout)
	assert.NotNil(t, ErrOrderSyncPlatformUnavailable)
	assert.NotNil(t, ErrOrderSyncRateLimited)
	assert.NotNil(t, ErrOrderSyncInvalidTimeRange)
	assert.NotNil(t, ErrOrderSyncNoEnabledPlatforms)
	assert.NotNil(t, ErrOrderSyncConfigNotFound)
	assert.NotNil(t, ErrOrderSyncAlreadyInProgress)
}
