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
)
