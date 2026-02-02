// Package service provides the main print service implementation.
package service

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// PrintTaskStatus represents the status of a print task.
type PrintTaskStatus string

// Print task statuses.
const (
	TaskStatusPending    PrintTaskStatus = "pending"
	TaskStatusProcessing PrintTaskStatus = "processing"
	TaskStatusCompleted  PrintTaskStatus = "completed"
	TaskStatusFailed     PrintTaskStatus = "failed"
	TaskStatusRetrying   PrintTaskStatus = "retrying"
)

// PrintTask represents a print task in the queue.
type PrintTask struct {
	ID             string          `json:"id"`
	DocumentType   string          `json:"document_type"`
	DocumentID     string          `json:"document_id"`
	DocumentNumber string          `json:"document_number,omitempty"`
	TemplateID     string          `json:"template_id,omitempty"`
	PrinterName    string          `json:"printer_name,omitempty"`
	Copies         int             `json:"copies"`
	PdfData        []byte          `json:"-"` // Not serialized
	Status         PrintTaskStatus `json:"status"`
	RetryCount     int             `json:"retry_count"`
	ErrorMessage   string          `json:"error_message,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	StartedAt      *time.Time      `json:"started_at,omitempty"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty"`
}

// PrintLog represents a log entry for a print task.
type PrintLog struct {
	TaskID         string          `json:"task_id"`
	DocumentType   string          `json:"document_type"`
	DocumentID     string          `json:"document_id"`
	DocumentNumber string          `json:"document_number,omitempty"`
	PrinterName    string          `json:"printer_name"`
	Copies         int             `json:"copies"`
	Status         PrintTaskStatus `json:"status"`
	ErrorMessage   string          `json:"error_message,omitempty"`
	RetryCount     int             `json:"retry_count"`
	Duration       time.Duration   `json:"duration"`
	Timestamp      time.Time       `json:"timestamp"`
}

// PrintQueueConfig holds configuration for the print queue.
type PrintQueueConfig struct {
	// QueueSize is the buffer size for the print queue channel.
	QueueSize int

	// MaxConcurrent is the maximum number of concurrent print tasks.
	MaxConcurrent int

	// MaxRetries is the maximum number of retry attempts.
	MaxRetries int

	// BaseRetryDelay is the base delay for exponential backoff.
	BaseRetryDelay time.Duration

	// MaxRetryDelay is the maximum delay between retries.
	MaxRetryDelay time.Duration

	// Logger is the logger instance.
	Logger *zap.Logger
}

// DefaultPrintQueueConfig returns the default print queue configuration.
func DefaultPrintQueueConfig() PrintQueueConfig {
	return PrintQueueConfig{
		QueueSize:      100,
		MaxConcurrent:  5,
		MaxRetries:     3,
		BaseRetryDelay: 1 * time.Second,
		MaxRetryDelay:  30 * time.Second,
		Logger:         zap.NewNop(),
	}
}

// PrintHandler is a function that handles printing a task.
type PrintHandler func(ctx context.Context, task *PrintTask) error

// PrintQueue manages a queue of print tasks with concurrency control and retry.
type PrintQueue struct {
	config       PrintQueueConfig
	taskChan     chan *PrintTask
	handler      PrintHandler
	logger       *zap.Logger
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	running      atomic.Bool
	workerSem    chan struct{}
	logs         []PrintLog
	logsMu       sync.RWMutex
	tasksCount   atomic.Int64
	successCount atomic.Int64
	failedCount  atomic.Int64
	closeOnce    sync.Once
}

// maxLogEntries is the maximum number of log entries to keep in memory.
const maxLogEntries = 10000

// NewPrintQueue creates a new print queue.
func NewPrintQueue(config PrintQueueConfig) *PrintQueue {
	if config.Logger == nil {
		config.Logger = zap.NewNop()
	}
	if config.QueueSize <= 0 {
		config.QueueSize = 100
	}
	if config.MaxConcurrent <= 0 {
		config.MaxConcurrent = 5
	}
	if config.MaxRetries < 0 {
		config.MaxRetries = 3
	}
	if config.BaseRetryDelay <= 0 {
		config.BaseRetryDelay = 1 * time.Second
	}
	if config.MaxRetryDelay <= 0 {
		config.MaxRetryDelay = 30 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &PrintQueue{
		config:    config,
		taskChan:  make(chan *PrintTask, config.QueueSize),
		logger:    config.Logger,
		ctx:       ctx,
		cancel:    cancel,
		workerSem: make(chan struct{}, config.MaxConcurrent),
		logs:      make([]PrintLog, 0),
	}
}

// SetHandler sets the print handler function.
// Must be called before Start().
func (q *PrintQueue) SetHandler(handler PrintHandler) error {
	if q.running.Load() {
		return fmt.Errorf("cannot set handler while queue is running")
	}
	q.handler = handler
	return nil
}

// Start starts the print queue workers.
func (q *PrintQueue) Start() error {
	if q.running.Load() {
		return fmt.Errorf("print queue already running")
	}

	if q.handler == nil {
		return fmt.Errorf("print handler not set")
	}

	q.running.Store(true)

	// Start the dispatcher goroutine
	q.wg.Add(1)
	go q.dispatcher()

	q.logger.Info("Print queue started",
		zap.Int("queue_size", q.config.QueueSize),
		zap.Int("max_concurrent", q.config.MaxConcurrent),
		zap.Int("max_retries", q.config.MaxRetries))

	return nil
}

// Stop stops the print queue gracefully.
func (q *PrintQueue) Stop() {
	if !q.running.Load() {
		return
	}

	q.running.Store(false)
	q.cancel()

	// Close the task channel to signal dispatcher to stop (only once)
	q.closeOnce.Do(func() {
		close(q.taskChan)
	})

	// Wait for all workers to finish
	q.wg.Wait()

	q.logger.Info("Print queue stopped",
		zap.Int64("total_tasks", q.tasksCount.Load()),
		zap.Int64("successful", q.successCount.Load()),
		zap.Int64("failed", q.failedCount.Load()))
}

// Enqueue adds a print task to the queue.
// Returns an error if the queue is full or not running.
func (q *PrintQueue) Enqueue(task *PrintTask) error {
	if !q.running.Load() {
		return fmt.Errorf("print queue not running")
	}

	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}

	// Set default values
	if task.Status == "" {
		task.Status = TaskStatusPending
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}
	if task.Copies < 1 {
		task.Copies = 1
	}

	// Try to enqueue without blocking
	select {
	case q.taskChan <- task:
		q.tasksCount.Add(1)
		q.logger.Debug("Task enqueued",
			zap.String("task_id", task.ID),
			zap.String("document_type", task.DocumentType),
			zap.String("document_id", task.DocumentID))
		return nil
	default:
		return fmt.Errorf("print queue is full")
	}
}

// EnqueueBlocking adds a print task to the queue, blocking if full.
// Returns an error if the context is cancelled or queue is stopped.
func (q *PrintQueue) EnqueueBlocking(ctx context.Context, task *PrintTask) error {
	if !q.running.Load() {
		return fmt.Errorf("print queue not running")
	}

	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}

	// Set default values
	if task.Status == "" {
		task.Status = TaskStatusPending
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}
	if task.Copies < 1 {
		task.Copies = 1
	}

	select {
	case q.taskChan <- task:
		q.tasksCount.Add(1)
		q.logger.Debug("Task enqueued (blocking)",
			zap.String("task_id", task.ID),
			zap.String("document_type", task.DocumentType),
			zap.String("document_id", task.DocumentID))
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-q.ctx.Done():
		return fmt.Errorf("print queue stopped")
	}
}

// dispatcher reads tasks from the channel and dispatches them to workers.
func (q *PrintQueue) dispatcher() {
	defer q.wg.Done()

	for task := range q.taskChan {
		// Acquire semaphore slot (limits concurrency)
		select {
		case q.workerSem <- struct{}{}:
			// Got a slot, process the task
			q.wg.Add(1)
			go q.processTask(task)
		case <-q.ctx.Done():
			// Queue is stopping, drain remaining tasks
			q.logger.Warn("Queue stopping, task dropped",
				zap.String("task_id", task.ID))
			return
		}
	}
}

// processTask processes a single print task with retry logic.
func (q *PrintQueue) processTask(task *PrintTask) {
	defer q.wg.Done()
	defer func() { <-q.workerSem }() // Release semaphore slot

	startTime := time.Now()
	task.Status = TaskStatusProcessing
	task.StartedAt = &startTime

	q.logger.Info("Processing print task",
		zap.String("task_id", task.ID),
		zap.String("document_type", task.DocumentType),
		zap.String("document_id", task.DocumentID),
		zap.Int("copies", task.Copies))

	var lastErr error

	// Retry loop with exponential backoff
	for attempt := 0; attempt <= q.config.MaxRetries; attempt++ {
		if attempt > 0 {
			task.Status = TaskStatusRetrying
			task.RetryCount = attempt

			// Calculate delay with exponential backoff
			delay := q.calculateBackoff(attempt)
			q.logger.Info("Retrying print task",
				zap.String("task_id", task.ID),
				zap.Int("attempt", attempt),
				zap.Duration("delay", delay))

			select {
			case <-time.After(delay):
				// Continue with retry
			case <-q.ctx.Done():
				// Queue is stopping
				q.recordLog(task, startTime, fmt.Errorf("queue stopped during retry"))
				return
			}
		}

		// Check context before executing handler
		select {
		case <-q.ctx.Done():
			task.Status = TaskStatusFailed
			task.ErrorMessage = "queue stopped"
			q.recordLog(task, startTime, q.ctx.Err())
			return
		default:
		}

		// Execute the print handler
		err := q.handler(q.ctx, task)
		if err == nil {
			// Success
			completedAt := time.Now()
			task.Status = TaskStatusCompleted
			task.CompletedAt = &completedAt
			task.ErrorMessage = ""
			q.successCount.Add(1)

			q.logger.Info("Print task completed",
				zap.String("task_id", task.ID),
				zap.Duration("duration", time.Since(startTime)))

			q.recordLog(task, startTime, nil)
			return
		}

		lastErr = err
		q.logger.Warn("Print task failed",
			zap.String("task_id", task.ID),
			zap.Int("attempt", attempt+1),
			zap.Int("max_retries", q.config.MaxRetries),
			zap.Error(err))
	}

	// All retries exhausted
	completedAt := time.Now()
	task.Status = TaskStatusFailed
	task.CompletedAt = &completedAt
	task.ErrorMessage = lastErr.Error()
	q.failedCount.Add(1)

	q.logger.Error("Print task failed after all retries",
		zap.String("task_id", task.ID),
		zap.Int("retries", q.config.MaxRetries),
		zap.Error(lastErr))

	q.recordLog(task, startTime, lastErr)
}

// calculateBackoff calculates the delay for exponential backoff.
func (q *PrintQueue) calculateBackoff(attempt int) time.Duration {
	// Exponential backoff: base * 2^attempt
	delay := float64(q.config.BaseRetryDelay) * math.Pow(2, float64(attempt-1))

	// Cap at max delay
	if delay > float64(q.config.MaxRetryDelay) {
		delay = float64(q.config.MaxRetryDelay)
	}

	return time.Duration(delay)
}

// recordLog records a print log entry.
func (q *PrintQueue) recordLog(task *PrintTask, startTime time.Time, err error) {
	log := PrintLog{
		TaskID:         task.ID,
		DocumentType:   task.DocumentType,
		DocumentID:     task.DocumentID,
		DocumentNumber: task.DocumentNumber,
		PrinterName:    task.PrinterName,
		Copies:         task.Copies,
		Status:         task.Status,
		RetryCount:     task.RetryCount,
		Duration:       time.Since(startTime),
		Timestamp:      time.Now(),
	}

	if err != nil {
		log.ErrorMessage = err.Error()
	}

	q.logsMu.Lock()
	q.logs = append(q.logs, log)
	// Limit log entries to prevent unbounded memory growth
	if len(q.logs) > maxLogEntries {
		// Remove oldest entries (keep last maxLogEntries)
		q.logs = q.logs[len(q.logs)-maxLogEntries:]
	}
	q.logsMu.Unlock()

	q.logger.Debug("Print log recorded",
		zap.String("task_id", log.TaskID),
		zap.String("status", string(log.Status)),
		zap.Duration("duration", log.Duration))
}

// GetLogs returns a copy of all print logs.
func (q *PrintQueue) GetLogs() []PrintLog {
	q.logsMu.RLock()
	defer q.logsMu.RUnlock()

	logs := make([]PrintLog, len(q.logs))
	copy(logs, q.logs)
	return logs
}

// GetRecentLogs returns the most recent n print logs.
func (q *PrintQueue) GetRecentLogs(n int) []PrintLog {
	q.logsMu.RLock()
	defer q.logsMu.RUnlock()

	if n <= 0 || len(q.logs) == 0 {
		return []PrintLog{}
	}

	start := len(q.logs) - n
	if start < 0 {
		start = 0
	}

	logs := make([]PrintLog, len(q.logs)-start)
	copy(logs, q.logs[start:])
	return logs
}

// Stats returns queue statistics.
func (q *PrintQueue) Stats() PrintQueueStats {
	return PrintQueueStats{
		TotalTasks:    q.tasksCount.Load(),
		SuccessCount:  q.successCount.Load(),
		FailedCount:   q.failedCount.Load(),
		QueueLength:   len(q.taskChan),
		QueueSize:     q.config.QueueSize,
		MaxConcurrent: q.config.MaxConcurrent,
		IsRunning:     q.running.Load(),
	}
}

// PrintQueueStats holds queue statistics.
type PrintQueueStats struct {
	TotalTasks    int64 `json:"total_tasks"`
	SuccessCount  int64 `json:"success_count"`
	FailedCount   int64 `json:"failed_count"`
	QueueLength   int   `json:"queue_length"`
	QueueSize     int   `json:"queue_size"`
	MaxConcurrent int   `json:"max_concurrent"`
	IsRunning     bool  `json:"is_running"`
}
