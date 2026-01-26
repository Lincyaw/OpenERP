package scheduler

import "errors"

var (
	// ErrSchedulerNotRunning is returned when trying to submit a job to a stopped scheduler
	ErrSchedulerNotRunning = errors.New("scheduler is not running")

	// ErrJobQueueFull is returned when the job queue is full
	ErrJobQueueFull = errors.New("job queue is full")

	// ErrInvalidReportType is returned for unknown report types
	ErrInvalidReportType = errors.New("invalid report type")

	// ErrJobNotFound is returned when a job is not found
	ErrJobNotFound = errors.New("job not found")

	// ErrReportComputationFailed is returned when report computation fails
	ErrReportComputationFailed = errors.New("report computation failed")

	// ErrInvalidConfig is returned when configuration is invalid
	ErrInvalidConfig = errors.New("invalid scheduler configuration")

	// ---------------------------------------------------------------------------
	// Order Sync Errors
	// ---------------------------------------------------------------------------

	// ErrOrderSyncFailed is returned when order sync fails
	ErrOrderSyncFailed = errors.New("order sync failed")

	// ErrOrderSyncTimeout is returned when order sync times out
	ErrOrderSyncTimeout = errors.New("order sync timed out")

	// ErrOrderSyncPlatformUnavailable is returned when the platform is unavailable
	ErrOrderSyncPlatformUnavailable = errors.New("platform unavailable for order sync")

	// ErrOrderSyncRateLimited is returned when rate limited by platform
	ErrOrderSyncRateLimited = errors.New("order sync rate limited by platform")

	// ErrOrderSyncInvalidTimeRange is returned for invalid time ranges
	ErrOrderSyncInvalidTimeRange = errors.New("invalid order sync time range")

	// ErrOrderSyncNoEnabledPlatforms is returned when no platforms are enabled for sync
	ErrOrderSyncNoEnabledPlatforms = errors.New("no enabled platforms for order sync")

	// ErrOrderSyncConfigNotFound is returned when sync config is not found
	ErrOrderSyncConfigNotFound = errors.New("order sync config not found")

	// ErrOrderSyncAlreadyInProgress is returned when a sync is already running
	ErrOrderSyncAlreadyInProgress = errors.New("order sync already in progress for this tenant/platform")
)
