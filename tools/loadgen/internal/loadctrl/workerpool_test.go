package loadctrl

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWorkerPool_BasicOperation(t *testing.T) {
	pool := NewWorkerPool(WorkerPoolConfig{
		MinSize:     2,
		MaxSize:     10,
		InitialSize: 5,
	})

	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Stop()

	assert.Equal(t, 5, pool.CurrentSize())
	assert.Equal(t, 2, pool.MinSize())
	assert.Equal(t, 10, pool.MaxSize())
}

func TestWorkerPool_Submit(t *testing.T) {
	pool := NewWorkerPool(WorkerPoolConfig{
		MinSize:     2,
		MaxSize:     10,
		InitialSize: 5,
	})

	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Stop()

	var executed atomic.Int32
	task := func(_ context.Context) error {
		executed.Add(1)
		return nil
	}

	// Submit tasks
	for range 10 {
		assert.True(t, pool.Submit(task))
	}

	// Wait for execution
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(10), executed.Load())

	stats := pool.Stats()
	assert.Equal(t, int64(10), stats.TotalExecuted)
	assert.Equal(t, int64(0), stats.TotalFailed)
}

func TestWorkerPool_AdjustSize(t *testing.T) {
	pool := NewWorkerPool(WorkerPoolConfig{
		MinSize:     2,
		MaxSize:     20,
		InitialSize: 5,
	})

	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Stop()

	// Scale up
	pool.AdjustSize(15)
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 15, pool.CurrentSize())

	// Scale down
	pool.AdjustSize(3)
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 3, pool.CurrentSize())

	// Try to go below min
	pool.AdjustSize(1)
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 2, pool.CurrentSize()) // Should be clamped to min

	// Try to go above max
	pool.AdjustSize(100)
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 20, pool.CurrentSize()) // Should be clamped to max
}

func TestWorkerPool_SubmitWait(t *testing.T) {
	pool := NewWorkerPool(WorkerPoolConfig{
		MinSize:       2,
		MaxSize:       5,
		InitialSize:   2,
		TaskQueueSize: 5,
	})

	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Stop()

	var executed atomic.Int32
	task := func(_ context.Context) error {
		executed.Add(1)
		time.Sleep(10 * time.Millisecond)
		return nil
	}

	// Submit with wait
	submitCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	err := pool.SubmitWait(submitCtx, task)
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(1), executed.Load())
}

func TestWorkerPool_StopCleanly(t *testing.T) {
	pool := NewWorkerPool(WorkerPoolConfig{
		MinSize:     2,
		MaxSize:     10,
		InitialSize: 5,
	})

	ctx := context.Background()
	pool.Start(ctx)

	// Submit some tasks
	var executed atomic.Int32
	for range 5 {
		pool.Submit(func(_ context.Context) error {
			executed.Add(1)
			return nil
		})
	}

	time.Sleep(50 * time.Millisecond)

	// Stop should complete without hanging
	done := make(chan struct{})
	go func() {
		pool.Stop()
		close(done)
	}()

	select {
	case <-done:
		// OK
	case <-time.After(time.Second):
		t.Fatal("Stop() hung")
	}
}

func TestWorkerPool_ConcurrentSubmit(t *testing.T) {
	pool := NewWorkerPool(WorkerPoolConfig{
		MinSize:       5,
		MaxSize:       20,
		InitialSize:   10,
		TaskQueueSize: 100,
	})

	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Stop()

	var executed atomic.Int64
	var wg sync.WaitGroup

	const goroutines = 10
	const tasksPerGoroutine = 50

	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range tasksPerGoroutine {
				pool.Submit(func(_ context.Context) error {
					executed.Add(1)
					return nil
				})
			}
		}()
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	// Most tasks should have been executed
	assert.True(t, executed.Load() > 0)
}

func TestWorkerPool_Stats(t *testing.T) {
	pool := NewWorkerPool(WorkerPoolConfig{
		MinSize:     2,
		MaxSize:     10,
		InitialSize: 5,
	})

	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Stop()

	// Submit some tasks
	for range 10 {
		pool.Submit(func(_ context.Context) error {
			return nil
		})
	}

	time.Sleep(100 * time.Millisecond)

	stats := pool.Stats()
	assert.Equal(t, 5, stats.CurrentSize)
	assert.Equal(t, 2, stats.MinSize)
	assert.Equal(t, 10, stats.MaxSize)
	assert.True(t, stats.TotalExecuted > 0)
}

func TestWorkerPool_NotStarted(t *testing.T) {
	pool := NewWorkerPool(WorkerPoolConfig{
		MinSize:     2,
		MaxSize:     10,
		InitialSize: 5,
	})

	// Submit without starting should return false
	submitted := pool.Submit(func(_ context.Context) error {
		return nil
	})
	assert.False(t, submitted)
}

// TestWorkerPool_Scale10To100_Acceptance verifies the acceptance criteria:
// "从 10 工作者扩展到 100 工作者平滑过渡" (Smooth transition from 10 to 100 workers)
func TestWorkerPool_Scale10To100_Acceptance(t *testing.T) {
	pool := NewWorkerPool(WorkerPoolConfig{
		MinSize:       1,
		MaxSize:       100,
		InitialSize:   10,
		TaskQueueSize: 500,
	})

	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Stop()

	// Verify initial state
	assert.Equal(t, 10, pool.CurrentSize(), "Should start with 10 workers")

	// Scale up from 10 to 100 smoothly
	var executed atomic.Int64
	var wg sync.WaitGroup

	// Start submitting tasks while scaling
	tasksDone := make(chan struct{})
	go func() {
		for range 500 {
			pool.Submit(func(_ context.Context) error {
				executed.Add(1)
				time.Sleep(time.Millisecond)
				return nil
			})
			time.Sleep(time.Millisecond)
		}
		close(tasksDone)
	}()

	// Scale up in steps to verify smooth transition
	scalingSteps := []int{25, 50, 75, 100}
	for _, targetSize := range scalingSteps {
		wg.Add(1)
		go func(target int) {
			defer wg.Done()
			pool.AdjustSize(target)
			time.Sleep(20 * time.Millisecond)
			currentSize := pool.CurrentSize()
			assert.Equal(t, target, currentSize, "Should scale to %d workers", target)
		}(targetSize)
		time.Sleep(50 * time.Millisecond)
	}

	wg.Wait()

	// Verify we reached 100 workers
	assert.Equal(t, 100, pool.CurrentSize(), "Should have 100 workers after scaling")

	// Wait for tasks to complete
	<-tasksDone
	time.Sleep(200 * time.Millisecond)

	// All workers should have processed tasks
	stats := pool.Stats()
	assert.True(t, executed.Load() > 100, "Should have executed many tasks: got %d", executed.Load())
	assert.Equal(t, 100, stats.CurrentSize)
}

// TestWorkerPool_Scale100To10_ScaleDown verifies scaling down from 100 to 10
func TestWorkerPool_Scale100To10_ScaleDown(t *testing.T) {
	pool := NewWorkerPool(WorkerPoolConfig{
		MinSize:       1,
		MaxSize:       100,
		InitialSize:   100,
		TaskQueueSize: 200,
	})

	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Stop()

	// Verify initial state
	assert.Equal(t, 100, pool.CurrentSize(), "Should start with 100 workers")

	// Scale down from 100 to 10
	scalingSteps := []int{75, 50, 25, 10}
	for _, targetSize := range scalingSteps {
		pool.AdjustSize(targetSize)
		time.Sleep(50 * time.Millisecond)
		assert.Equal(t, targetSize, pool.CurrentSize(), "Should scale to %d workers", targetSize)
	}

	// Verify we reached 10 workers
	assert.Equal(t, 10, pool.CurrentSize(), "Should have 10 workers after scaling down")

	// Verify tasks still execute correctly with reduced workers
	var executed atomic.Int32
	for range 20 {
		pool.Submit(func(_ context.Context) error {
			executed.Add(1)
			return nil
		})
	}

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(20), executed.Load(), "All tasks should execute with 10 workers")
}

// TestWorkerPool_Results tests the Results channel
func TestWorkerPool_Results(t *testing.T) {
	pool := NewWorkerPool(WorkerPoolConfig{
		MinSize:       2,
		MaxSize:       5,
		InitialSize:   2,
		TaskQueueSize: 10,
	})

	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Stop()

	// Get results channel
	resultsCh := pool.Results()
	assert.NotNil(t, resultsCh, "Results channel should not be nil")

	// Submit a task
	pool.Submit(func(_ context.Context) error {
		return nil
	})

	// Read result from channel
	select {
	case result := <-resultsCh:
		assert.NoError(t, result.Err, "Task should succeed")
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for result")
	}
}

// TestWorkerPool_FailedTask tests failed task counting
func TestWorkerPool_FailedTask(t *testing.T) {
	pool := NewWorkerPool(WorkerPoolConfig{
		MinSize:     2,
		MaxSize:     5,
		InitialSize: 2,
	})

	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Stop()

	// Submit tasks that fail
	for range 5 {
		pool.Submit(func(_ context.Context) error {
			return assert.AnError
		})
	}

	// Submit tasks that succeed
	for range 5 {
		pool.Submit(func(_ context.Context) error {
			return nil
		})
	}

	time.Sleep(100 * time.Millisecond)

	stats := pool.Stats()
	assert.Equal(t, int64(5), stats.TotalFailed, "Should have 5 failed tasks")
	assert.Equal(t, int64(5), stats.TotalExecuted, "Should have 5 executed tasks")
}

// TestWorkerPool_SubmitWait_Cancelled tests SubmitWait with cancelled context
func TestWorkerPool_SubmitWait_Cancelled(t *testing.T) {
	pool := NewWorkerPool(WorkerPoolConfig{
		MinSize:       1,
		MaxSize:       1,
		InitialSize:   1,
		TaskQueueSize: 1, // Very small queue
	})

	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Stop()

	// Block the only worker with a slow task
	started := make(chan struct{})
	pool.Submit(func(_ context.Context) error {
		close(started)
		time.Sleep(500 * time.Millisecond)
		return nil
	})

	// Wait for worker to pick up the first task
	<-started

	// Fill the queue (size 1)
	pool.Submit(func(_ context.Context) error {
		time.Sleep(500 * time.Millisecond)
		return nil
	})

	// SubmitWait with already cancelled context - queue is full so it will block and check context
	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel()

	err := pool.SubmitWait(cancelledCtx, func(_ context.Context) error {
		return nil
	})
	assert.Error(t, err, "Should return error for cancelled context")
	assert.ErrorIs(t, err, context.Canceled)
}

// TestWorkerPool_SubmitWait_PoolStopped tests SubmitWait when pool is stopped
func TestWorkerPool_SubmitWait_PoolStopped(t *testing.T) {
	pool := NewWorkerPool(WorkerPoolConfig{
		MinSize:     1,
		MaxSize:     2,
		InitialSize: 1,
	})

	ctx := context.Background()
	pool.Start(ctx)
	pool.Stop()

	// SubmitWait after pool is stopped
	err := pool.SubmitWait(ctx, func(_ context.Context) error {
		return nil
	})
	assert.Error(t, err, "Should return error for stopped pool")
}

// TestWorkerPool_AdjustSize_NotRunning tests AdjustSize when pool is not running
func TestWorkerPool_AdjustSize_NotRunning(t *testing.T) {
	pool := NewWorkerPool(WorkerPoolConfig{
		MinSize:     2,
		MaxSize:     10,
		InitialSize: 5,
	})

	// AdjustSize without starting should do nothing
	pool.AdjustSize(8)
	// No panic or error expected
}

// TestWorkerPool_DefaultConfig tests default configuration values
func TestWorkerPool_DefaultConfig(t *testing.T) {
	// Test with all zeros (should use defaults)
	pool := NewWorkerPool(WorkerPoolConfig{})

	assert.Equal(t, 1, pool.MinSize(), "Default MinSize should be 1")
	assert.Equal(t, 100, pool.MaxSize(), "Default MaxSize should be 100")
}

// TestWorkerPool_InvalidConfig tests config validation
func TestWorkerPool_InvalidConfig(t *testing.T) {
	// MaxSize < MinSize should be corrected
	pool := NewWorkerPool(WorkerPoolConfig{
		MinSize:     10,
		MaxSize:     5,
		InitialSize: 7,
	})

	assert.Equal(t, 10, pool.MinSize())
	assert.Equal(t, 10, pool.MaxSize(), "MaxSize should be clamped to MinSize")
}

// TestWorkerPool_RestartAfterStop tests that pool can be restarted
func TestWorkerPool_RestartAfterStop(t *testing.T) {
	pool := NewWorkerPool(WorkerPoolConfig{
		MinSize:     2,
		MaxSize:     10,
		InitialSize: 5,
	})

	ctx := context.Background()

	// First run
	pool.Start(ctx)
	var executed1 atomic.Int32
	pool.Submit(func(_ context.Context) error {
		executed1.Add(1)
		return nil
	})
	time.Sleep(50 * time.Millisecond)
	pool.Stop()

	// Second run
	pool.Start(ctx)
	var executed2 atomic.Int32
	pool.Submit(func(_ context.Context) error {
		executed2.Add(1)
		return nil
	})
	time.Sleep(50 * time.Millisecond)
	pool.Stop()

	assert.Equal(t, int32(1), executed1.Load())
	assert.Equal(t, int32(1), executed2.Load())
}

// TestWorkerPool_ContextCancellation tests worker response to context cancellation
func TestWorkerPool_ContextCancellation(t *testing.T) {
	pool := NewWorkerPool(WorkerPoolConfig{
		MinSize:     2,
		MaxSize:     10,
		InitialSize: 5,
	})

	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)

	var submitted atomic.Int32

	// Submit some tasks
	for range 10 {
		pool.Submit(func(_ context.Context) error {
			submitted.Add(1)
			time.Sleep(100 * time.Millisecond)
			return nil
		})
	}

	// Cancel context
	cancel()

	// Pool should eventually stop (workers exit on context cancellation)
	time.Sleep(200 * time.Millisecond)

	// Stop should complete quickly since workers are already stopping
	done := make(chan struct{})
	go func() {
		pool.Stop()
		close(done)
	}()

	select {
	case <-done:
		// OK
	case <-time.After(time.Second):
		t.Fatal("Stop() hung after context cancellation")
	}
}

// TestWorkerPool_ConcurrentAdjustSize tests concurrent size adjustments
func TestWorkerPool_ConcurrentAdjustSize(t *testing.T) {
	pool := NewWorkerPool(WorkerPoolConfig{
		MinSize:       5,
		MaxSize:       50,
		InitialSize:   25,
		TaskQueueSize: 200,
	})

	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Stop()

	var wg sync.WaitGroup
	const goroutines = 5

	// Concurrently adjust size from multiple goroutines
	for i := range goroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range 10 {
				targetSize := 10 + (id+j)%40 // Vary between 10-49
				pool.AdjustSize(targetSize)
				time.Sleep(5 * time.Millisecond)
			}
		}(i)
	}

	// Also submit tasks concurrently
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 20 {
				pool.Submit(func(_ context.Context) error {
					return nil
				})
			}
		}()
	}

	wg.Wait()

	// Pool should be in a valid state
	currentSize := pool.CurrentSize()
	assert.True(t, currentSize >= 5 && currentSize <= 50,
		"Size should be within bounds: got %d", currentSize)
}

// TestWorkerPool_PendingTasks tests PendingTasks statistic
func TestWorkerPool_PendingTasks(t *testing.T) {
	pool := NewWorkerPool(WorkerPoolConfig{
		MinSize:       1,
		MaxSize:       2,
		InitialSize:   1,
		TaskQueueSize: 20,
	})

	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Stop()

	// Submit slow task to block the worker
	pool.Submit(func(_ context.Context) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})

	// Give worker time to pick up task
	time.Sleep(10 * time.Millisecond)

	// Submit more tasks that will queue
	for range 5 {
		pool.Submit(func(_ context.Context) error {
			return nil
		})
	}

	stats := pool.Stats()
	assert.True(t, stats.PendingTasks > 0, "Should have pending tasks: got %d", stats.PendingTasks)
}
