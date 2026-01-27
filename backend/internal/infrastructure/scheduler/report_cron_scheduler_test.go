package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCronSchedule(t *testing.T) {
	tests := []struct {
		name         string
		cronExpr     string
		expectedHour int
		expectedMin  int
	}{
		{
			name:         "Default 2am",
			cronExpr:     "0 2 * * *",
			expectedHour: 2,
			expectedMin:  0,
		},
		{
			name:         "3:30am",
			cronExpr:     "30 3 * * *",
			expectedHour: 3,
			expectedMin:  30,
		},
		{
			name:         "Midnight",
			cronExpr:     "0 0 * * *",
			expectedHour: 0,
			expectedMin:  0,
		},
		{
			name:         "11pm",
			cronExpr:     "0 23 * * *",
			expectedHour: 23,
			expectedMin:  0,
		},
		{
			name:         "Empty string defaults",
			cronExpr:     "",
			expectedHour: 2,
			expectedMin:  0,
		},
		{
			name:         "Extra whitespace",
			cronExpr:     "  15   4   *   *   *  ",
			expectedHour: 4,
			expectedMin:  15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hour, minute, err := ParseCronSchedule(tt.cronExpr)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedHour, hour, "hour mismatch")
			assert.Equal(t, tt.expectedMin, minute, "minute mismatch")
		})
	}
}

func TestDefaultReportCronSchedulerConfig(t *testing.T) {
	cfg := DefaultReportCronSchedulerConfig()

	assert.True(t, cfg.Enabled)
	assert.Equal(t, 2, cfg.CronHour)
	assert.Equal(t, 0, cfg.CronMinute)
	assert.Equal(t, 3, cfg.MaxConcurrentJobs)
	assert.Equal(t, 30*time.Minute, cfg.JobTimeout)
	assert.Equal(t, 3, cfg.RetryAttempts)
	assert.Equal(t, 5*time.Minute, cfg.RetryDelay)
}

func TestShouldRun(t *testing.T) {
	cfg := DefaultReportCronSchedulerConfig()
	cfg.CronHour = 2
	cfg.CronMinute = 30

	// Create a minimal scheduler for testing shouldRun
	s := &ReportCronScheduler{
		config: cfg,
	}

	tests := []struct {
		name     string
		time     time.Time
		expected bool
	}{
		{
			name:     "Exact match",
			time:     time.Date(2026, 1, 15, 2, 30, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "Wrong hour",
			time:     time.Date(2026, 1, 15, 3, 30, 0, 0, time.UTC),
			expected: false,
		},
		{
			name:     "Wrong minute",
			time:     time.Date(2026, 1, 15, 2, 31, 0, 0, time.UTC),
			expected: false,
		},
		{
			name:     "Midnight vs 2:30",
			time:     time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.shouldRun(tt.time)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateNextRunTime(t *testing.T) {
	cfg := DefaultReportCronSchedulerConfig()
	cfg.CronHour = 2
	cfg.CronMinute = 0

	s := &ReportCronScheduler{
		config: cfg,
	}

	// Test before today's run time
	t.Run("Before run time", func(t *testing.T) {
		// Assume current time is 1:00 AM
		s.calculateNextRunTime()
		assert.NotNil(t, s.nextRunAt)
		// Should be scheduled for today at 2:00 AM
		assert.Equal(t, cfg.CronHour, s.nextRunAt.Hour())
		assert.Equal(t, cfg.CronMinute, s.nextRunAt.Minute())
	})
}

func TestSchedulerJobRecord(t *testing.T) {
	record := SchedulerJobRecord{}
	assert.Equal(t, "report_scheduler_jobs", record.TableName())
}

func TestReportCronScheduler_GetStatus(t *testing.T) {
	cfg := DefaultReportCronSchedulerConfig()
	s := &ReportCronScheduler{
		config:    cfg,
		isRunning: true,
	}

	status := s.GetStatus()

	assert.Equal(t, true, status["enabled"])
	assert.Equal(t, true, status["is_running"])
	assert.Equal(t, cfg.CronHour, status["cron_hour"])
	assert.Equal(t, cfg.CronMinute, status["cron_minute"])
	assert.Equal(t, "Daily", status["cron_schedule"])
	assert.Contains(t, status, "report_types")
}

func TestReportCronScheduler_TriggerManualRun_NotRunning(t *testing.T) {
	cfg := DefaultReportCronSchedulerConfig()
	s := &ReportCronScheduler{
		config:    cfg,
		isRunning: false,
	}

	err := s.TriggerManualRun(context.Background())
	assert.ErrorIs(t, err, ErrSchedulerNotRunning)
}

func TestReportCronScheduler_TriggerTenantAggregation_NotRunning(t *testing.T) {
	cfg := DefaultReportCronSchedulerConfig()
	s := &ReportCronScheduler{
		config:    cfg,
		isRunning: false,
	}

	err := s.TriggerTenantAggregation(context.Background(), [16]byte{}, time.Now(), time.Now())
	assert.ErrorIs(t, err, ErrSchedulerNotRunning)
}

func TestAllReportTypes(t *testing.T) {
	types := AllReportTypes()

	require.Len(t, types, 6)
	assert.Contains(t, types, ReportTypeSalesSummary)
	assert.Contains(t, types, ReportTypeSalesDailyTrend)
	assert.Contains(t, types, ReportTypeInventorySummary)
	assert.Contains(t, types, ReportTypeProfitLossMonthly)
	assert.Contains(t, types, ReportTypeProductRanking)
	assert.Contains(t, types, ReportTypeCustomerRanking)
}
