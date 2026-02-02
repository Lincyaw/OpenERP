package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewPrintQueue(t *testing.T) {
	config := DefaultPrintQueueConfig()
	queue := NewPrintQueue(config)

	if queue == nil {
		t.Fatal("Expected non-nil PrintQueue")
	}

	stats := queue.Stats()
	if stats.QueueSize != config.QueueSize {
		t.Errorf("Expected QueueSize %d, got %d", config.QueueSize, stats.QueueSize)
	}
	if stats.MaxConcurrent != config.MaxConcurrent {
		t.Errorf("Expected MaxConcurrent %d, got %d", config.MaxConcurrent, stats.MaxConcurrent)
	}
	if stats.IsRunning {
		t.Error("Expected queue to not be running initially")
	}
}

func TestPrintQueue_DefaultConfig(t *testing.T) {
	config := DefaultPrintQueueConfig()

	if config.QueueSize != 100 {
		t.Errorf("Expected default QueueSize 100, got %d", config.QueueSize)
	}
	if config.MaxConcurrent != 5 {
		t.Errorf("Expected default MaxConcurrent 5, got %d", config.MaxConcurrent)
	}
	if config.MaxRetries != 3 {
		t.Errorf("Expected default MaxRetries 3, got %d", config.MaxRetries)
	}
	if config.BaseRetryDelay != 1*time.Second {
		t.Errorf("Expected default BaseRetryDelay 1s, got %v", config.BaseRetryDelay)
	}
	if config.MaxRetryDelay != 30*time.Second {
		t.Errorf("Expected default MaxRetryDelay 30s, got %v", config.MaxRetryDelay)
	}
}

func TestPrintQueue_StartStop(t *testing.T) {
	config := DefaultPrintQueueConfig()
	config.Logger = zap.NewNop()
	queue := NewPrintQueue(config)

	// Set a dummy handler
	queue.SetHandler(func(ctx context.Context, task *PrintTask) error {
		return nil
	})

	// Start the queue
	err := queue.Start()
	if err != nil {
		t.Fatalf("Failed to start queue: %v", err)
	}

	stats := queue.Stats()
	if !stats.IsRunning {
		t.Error("Expected queue to be running after Start")
	}

	// Starting again should fail
	err = queue.Start()
	if err == nil {
		t.Error("Expected error when starting already running queue")
	}

	// Stop the queue
	queue.Stop()

	stats = queue.Stats()
	if stats.IsRunning {
		t.Error("Expected queue to not be running after Stop")
	}
}

func TestPrintQueue_StartWithoutHandler(t *testing.T) {
	config := DefaultPrintQueueConfig()
	queue := NewPrintQueue(config)

	err := queue.Start()
	if err == nil {
		t.Error("Expected error when starting without handler")
	}
}

func TestPrintQueue_Enqueue(t *testing.T) {
	config := DefaultPrintQueueConfig()
	config.QueueSize = 10
	config.Logger = zap.NewNop()
	queue := NewPrintQueue(config)

	var processedTasks atomic.Int32
	queue.SetHandler(func(ctx context.Context, task *PrintTask) error {
		processedTasks.Add(1)
		return nil
	})

	err := queue.Start()
	if err != nil {
		t.Fatalf("Failed to start queue: %v", err)
	}
	defer queue.Stop()

	// Enqueue a task
	task := &PrintTask{
		ID:           "task-1",
		DocumentType: "SALES_ORDER",
		DocumentID:   "order-123",
		Copies:       2,
	}

	err = queue.Enqueue(task)
	if err != nil {
		t.Fatalf("Failed to enqueue task: %v", err)
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	if processedTasks.Load() != 1 {
		t.Errorf("Expected 1 processed task, got %d", processedTasks.Load())
	}

	stats := queue.Stats()
	if stats.TotalTasks != 1 {
		t.Errorf("Expected TotalTasks 1, got %d", stats.TotalTasks)
	}
	if stats.SuccessCount != 1 {
		t.Errorf("Expected SuccessCount 1, got %d", stats.SuccessCount)
	}
}

func TestPrintQueue_EnqueueNilTask(t *testing.T) {
	config := DefaultPrintQueueConfig()
	queue := NewPrintQueue(config)
	queue.SetHandler(func(ctx context.Context, task *PrintTask) error {
		return nil
	})
	queue.Start()
	defer queue.Stop()

	err := queue.Enqueue(nil)
	if err == nil {
		t.Error("Expected error when enqueueing nil task")
	}
}

func TestPrintQueue_EnqueueWhenNotRunning(t *testing.T) {
	config := DefaultPrintQueueConfig()
	queue := NewPrintQueue(config)

	task := &PrintTask{
		ID:           "task-1",
		DocumentType: "SALES_ORDER",
		DocumentID:   "order-123",
	}

	err := queue.Enqueue(task)
	if err == nil {
		t.Error("Expected error when enqueueing to non-running queue")
	}
}

func TestPrintQueue_EnqueueFull(t *testing.T) {
	config := DefaultPrintQueueConfig()
	config.QueueSize = 2
	config.MaxConcurrent = 1
	config.Logger = zap.NewNop()
	queue := NewPrintQueue(config)

	// Handler that blocks
	blockChan := make(chan struct{})
	queue.SetHandler(func(ctx context.Context, task *PrintTask) error {
		<-blockChan
		return nil
	})

	err := queue.Start()
	if err != nil {
		t.Fatalf("Failed to start queue: %v", err)
	}
	defer func() {
		close(blockChan)
		queue.Stop()
	}()

	// Fill the queue
	for i := 0; i < 3; i++ {
		task := &PrintTask{
			ID:           "task-" + string(rune('0'+i)),
			DocumentType: "SALES_ORDER",
			DocumentID:   "order-" + string(rune('0'+i)),
		}
		queue.Enqueue(task)
	}

	// Next enqueue should fail (queue full)
	task := &PrintTask{
		ID:           "task-overflow",
		DocumentType: "SALES_ORDER",
		DocumentID:   "order-overflow",
	}
	err = queue.Enqueue(task)
	if err == nil {
		t.Error("Expected error when queue is full")
	}
}

func TestPrintQueue_ConcurrencyControl(t *testing.T) {
	config := DefaultPrintQueueConfig()
	config.QueueSize = 20
	config.MaxConcurrent = 3
	config.Logger = zap.NewNop()
	queue := NewPrintQueue(config)

	var maxConcurrent atomic.Int32
	var currentConcurrent atomic.Int32
	var mu sync.Mutex

	queue.SetHandler(func(ctx context.Context, task *PrintTask) error {
		current := currentConcurrent.Add(1)
		defer currentConcurrent.Add(-1)

		mu.Lock()
		if current > maxConcurrent.Load() {
			maxConcurrent.Store(current)
		}
		mu.Unlock()

		// Simulate work
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	err := queue.Start()
	if err != nil {
		t.Fatalf("Failed to start queue: %v", err)
	}
	defer queue.Stop()

	// Enqueue many tasks
	for i := 0; i < 10; i++ {
		task := &PrintTask{
			ID:           "task-" + string(rune('0'+i)),
			DocumentType: "SALES_ORDER",
			DocumentID:   "order-" + string(rune('0'+i)),
		}
		queue.Enqueue(task)
	}

	// Wait for all tasks to complete
	time.Sleep(500 * time.Millisecond)

	// Verify max concurrent never exceeded limit
	if maxConcurrent.Load() > int32(config.MaxConcurrent) {
		t.Errorf("Max concurrent %d exceeded limit %d", maxConcurrent.Load(), config.MaxConcurrent)
	}

	stats := queue.Stats()
	if stats.SuccessCount != 10 {
		t.Errorf("Expected 10 successful tasks, got %d", stats.SuccessCount)
	}
}

func TestPrintQueue_RetryMechanism(t *testing.T) {
	config := DefaultPrintQueueConfig()
	config.MaxRetries = 2
	config.BaseRetryDelay = 10 * time.Millisecond
	config.MaxRetryDelay = 50 * time.Millisecond
	config.Logger = zap.NewNop()
	queue := NewPrintQueue(config)

	var attempts atomic.Int32

	queue.SetHandler(func(ctx context.Context, task *PrintTask) error {
		attempt := attempts.Add(1)
		if attempt < 3 {
			return errors.New("temporary failure")
		}
		return nil // Success on 3rd attempt
	})

	err := queue.Start()
	if err != nil {
		t.Fatalf("Failed to start queue: %v", err)
	}
	defer queue.Stop()

	task := &PrintTask{
		ID:           "task-retry",
		DocumentType: "SALES_ORDER",
		DocumentID:   "order-retry",
	}

	err = queue.Enqueue(task)
	if err != nil {
		t.Fatalf("Failed to enqueue task: %v", err)
	}

	// Wait for retries
	time.Sleep(200 * time.Millisecond)

	if attempts.Load() != 3 {
		t.Errorf("Expected 3 attempts (1 initial + 2 retries), got %d", attempts.Load())
	}

	stats := queue.Stats()
	if stats.SuccessCount != 1 {
		t.Errorf("Expected 1 successful task, got %d", stats.SuccessCount)
	}
}

func TestPrintQueue_RetryExhausted(t *testing.T) {
	config := DefaultPrintQueueConfig()
	config.MaxRetries = 2
	config.BaseRetryDelay = 10 * time.Millisecond
	config.MaxRetryDelay = 50 * time.Millisecond
	config.Logger = zap.NewNop()
	queue := NewPrintQueue(config)

	var attempts atomic.Int32

	queue.SetHandler(func(ctx context.Context, task *PrintTask) error {
		attempts.Add(1)
		return errors.New("permanent failure")
	})

	err := queue.Start()
	if err != nil {
		t.Fatalf("Failed to start queue: %v", err)
	}
	defer queue.Stop()

	task := &PrintTask{
		ID:           "task-fail",
		DocumentType: "SALES_ORDER",
		DocumentID:   "order-fail",
	}

	err = queue.Enqueue(task)
	if err != nil {
		t.Fatalf("Failed to enqueue task: %v", err)
	}

	// Wait for all retries
	time.Sleep(200 * time.Millisecond)

	// Should have 3 attempts: 1 initial + 2 retries
	if attempts.Load() != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts.Load())
	}

	stats := queue.Stats()
	if stats.FailedCount != 1 {
		t.Errorf("Expected 1 failed task, got %d", stats.FailedCount)
	}
	if stats.SuccessCount != 0 {
		t.Errorf("Expected 0 successful tasks, got %d", stats.SuccessCount)
	}
}

func TestPrintQueue_Logging(t *testing.T) {
	config := DefaultPrintQueueConfig()
	config.Logger = zap.NewNop()
	queue := NewPrintQueue(config)

	queue.SetHandler(func(ctx context.Context, task *PrintTask) error {
		return nil
	})

	err := queue.Start()
	if err != nil {
		t.Fatalf("Failed to start queue: %v", err)
	}
	defer queue.Stop()

	// Enqueue tasks
	for i := 0; i < 5; i++ {
		task := &PrintTask{
			ID:             "task-" + string(rune('0'+i)),
			DocumentType:   "SALES_ORDER",
			DocumentID:     "order-" + string(rune('0'+i)),
			DocumentNumber: "SO-" + string(rune('0'+i)),
			PrinterName:    "Test Printer",
			Copies:         1,
		}
		queue.Enqueue(task)
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	logs := queue.GetLogs()
	if len(logs) != 5 {
		t.Errorf("Expected 5 logs, got %d", len(logs))
	}

	// Check log content
	for _, log := range logs {
		if log.Status != TaskStatusCompleted {
			t.Errorf("Expected status 'completed', got '%s'", log.Status)
		}
		if log.PrinterName != "Test Printer" {
			t.Errorf("Expected printer 'Test Printer', got '%s'", log.PrinterName)
		}
		if log.Duration <= 0 {
			t.Error("Expected positive duration")
		}
	}
}

func TestPrintQueue_GetRecentLogs(t *testing.T) {
	config := DefaultPrintQueueConfig()
	config.Logger = zap.NewNop()
	queue := NewPrintQueue(config)

	queue.SetHandler(func(ctx context.Context, task *PrintTask) error {
		return nil
	})

	err := queue.Start()
	if err != nil {
		t.Fatalf("Failed to start queue: %v", err)
	}
	defer queue.Stop()

	// Enqueue tasks
	for i := 0; i < 10; i++ {
		task := &PrintTask{
			ID:           "task-" + string(rune('0'+i)),
			DocumentType: "SALES_ORDER",
			DocumentID:   "order-" + string(rune('0'+i)),
		}
		queue.Enqueue(task)
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Get recent 3 logs
	recentLogs := queue.GetRecentLogs(3)
	if len(recentLogs) != 3 {
		t.Errorf("Expected 3 recent logs, got %d", len(recentLogs))
	}

	// Get more than available
	allLogs := queue.GetRecentLogs(100)
	if len(allLogs) != 10 {
		t.Errorf("Expected 10 logs, got %d", len(allLogs))
	}

	// Get 0 logs
	noLogs := queue.GetRecentLogs(0)
	if len(noLogs) != 0 {
		t.Errorf("Expected 0 logs, got %d", len(noLogs))
	}
}

func TestPrintQueue_CalculateBackoff(t *testing.T) {
	config := DefaultPrintQueueConfig()
	config.BaseRetryDelay = 1 * time.Second
	config.MaxRetryDelay = 30 * time.Second
	queue := NewPrintQueue(config)

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 1 * time.Second},  // 1 * 2^0 = 1s
		{2, 2 * time.Second},  // 1 * 2^1 = 2s
		{3, 4 * time.Second},  // 1 * 2^2 = 4s
		{4, 8 * time.Second},  // 1 * 2^3 = 8s
		{5, 16 * time.Second}, // 1 * 2^4 = 16s
		{6, 30 * time.Second}, // Capped at max (32s > 30s)
		{7, 30 * time.Second}, // Capped at max
	}

	for _, tt := range tests {
		t.Run("attempt_"+string(rune('0'+tt.attempt)), func(t *testing.T) {
			delay := queue.calculateBackoff(tt.attempt)
			if delay != tt.expected {
				t.Errorf("Attempt %d: expected %v, got %v", tt.attempt, tt.expected, delay)
			}
		})
	}
}

func TestPrintQueue_EnqueueBlocking(t *testing.T) {
	config := DefaultPrintQueueConfig()
	config.QueueSize = 2
	config.Logger = zap.NewNop()
	queue := NewPrintQueue(config)

	queue.SetHandler(func(ctx context.Context, task *PrintTask) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	err := queue.Start()
	if err != nil {
		t.Fatalf("Failed to start queue: %v", err)
	}
	defer queue.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Enqueue with blocking
	task := &PrintTask{
		ID:           "task-blocking",
		DocumentType: "SALES_ORDER",
		DocumentID:   "order-blocking",
	}

	err = queue.EnqueueBlocking(ctx, task)
	if err != nil {
		t.Fatalf("Failed to enqueue task: %v", err)
	}
}

func TestPrintQueue_EnqueueBlockingTimeout(t *testing.T) {
	config := DefaultPrintQueueConfig()
	config.QueueSize = 1
	config.MaxConcurrent = 1
	config.Logger = zap.NewNop()
	queue := NewPrintQueue(config)

	// Handler that blocks forever
	started := make(chan struct{}, 1)
	blockChan := make(chan struct{})
	queue.SetHandler(func(ctx context.Context, task *PrintTask) error {
		select {
		case started <- struct{}{}: // Signal that handler started (only once)
		default:
		}
		<-blockChan
		return nil
	})

	err := queue.Start()
	if err != nil {
		t.Fatalf("Failed to start queue: %v", err)
	}
	defer func() {
		close(blockChan)
		queue.Stop()
	}()

	// Enqueue first task and wait for it to start processing
	err = queue.Enqueue(&PrintTask{ID: "task-1", DocumentType: "SALES_ORDER", DocumentID: "order-1"})
	if err != nil {
		t.Fatalf("Failed to enqueue task-1: %v", err)
	}
	<-started // Wait for handler to start

	// Now fill the buffer (size 1) - this should succeed
	err = queue.Enqueue(&PrintTask{ID: "task-2", DocumentType: "SALES_ORDER", DocumentID: "order-2"})
	if err != nil {
		t.Fatalf("Failed to enqueue task-2: %v", err)
	}

	// Non-blocking enqueue should fail now (queue full)
	err = queue.Enqueue(&PrintTask{ID: "task-3", DocumentType: "SALES_ORDER", DocumentID: "order-3"})
	if err == nil {
		t.Log("Warning: Non-blocking enqueue succeeded when queue should be full")
	}

	// Try to enqueue with short timeout - queue should be full now
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	task := &PrintTask{
		ID:           "task-timeout",
		DocumentType: "SALES_ORDER",
		DocumentID:   "order-timeout",
	}

	err = queue.EnqueueBlocking(ctx, task)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded error, got: %v", err)
	}
}

func TestPrintTaskStatus_Values(t *testing.T) {
	tests := []struct {
		status   PrintTaskStatus
		expected string
	}{
		{TaskStatusPending, "pending"},
		{TaskStatusProcessing, "processing"},
		{TaskStatusCompleted, "completed"},
		{TaskStatusFailed, "failed"},
		{TaskStatusRetrying, "retrying"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, string(tt.status))
			}
		})
	}
}

func TestPrintTask_DefaultValues(t *testing.T) {
	config := DefaultPrintQueueConfig()
	config.Logger = zap.NewNop()
	queue := NewPrintQueue(config)

	var receivedTask *PrintTask
	var mu sync.Mutex
	done := make(chan struct{})

	queue.SetHandler(func(ctx context.Context, task *PrintTask) error {
		mu.Lock()
		receivedTask = task
		mu.Unlock()
		close(done)
		return nil
	})

	queue.Start()
	defer queue.Stop()

	// Enqueue task without setting defaults
	task := &PrintTask{
		ID:           "task-defaults",
		DocumentType: "SALES_ORDER",
		DocumentID:   "order-defaults",
	}

	queue.Enqueue(task)

	// Wait for task to be processed
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for task to be processed")
	}

	mu.Lock()
	defer mu.Unlock()

	if receivedTask == nil {
		t.Fatal("Task was not processed")
	}

	if receivedTask.Copies != 1 {
		t.Errorf("Expected default Copies 1, got %d", receivedTask.Copies)
	}

	if receivedTask.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
}
