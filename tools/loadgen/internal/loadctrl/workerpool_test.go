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
