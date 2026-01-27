// Package loadctrl provides load control components including traffic shaping
// and rate limiting for the load generator.
package loadctrl

import (
	"context"
	"sync"
	"sync/atomic"
)

// WorkerPool manages a dynamic pool of workers for executing tasks.
// It supports resizing the pool at runtime to match target QPS.
//
// Thread Safety: Safe for concurrent use.
type WorkerPool struct {
	minSize     int
	maxSize     int
	currentSize atomic.Int32

	// Worker coordination
	workers   []*worker
	workersMu sync.RWMutex
	taskCh    chan Task
	resultCh  chan TaskResult
	stopCh    chan struct{}
	ctx       context.Context // Stored context from Start
	wg        sync.WaitGroup
	isRunning atomic.Bool

	// Statistics
	totalExecuted atomic.Int64
	totalFailed   atomic.Int64
}

// Task represents a unit of work to be executed by a worker.
type Task func(ctx context.Context) error

// TaskResult represents the result of executing a task.
type TaskResult struct {
	Err error
}

// WorkerPoolConfig holds configuration for creating a worker pool.
type WorkerPoolConfig struct {
	// MinSize is the minimum number of workers (default: 1).
	MinSize int `yaml:"minSize" json:"minSize"`
	// MaxSize is the maximum number of workers (default: 100).
	MaxSize int `yaml:"maxSize" json:"maxSize"`
	// InitialSize is the starting number of workers (default: MinSize).
	InitialSize int `yaml:"initialSize,omitempty" json:"initialSize,omitempty"`
	// TaskQueueSize is the size of the task queue (default: MaxSize * 2).
	TaskQueueSize int `yaml:"taskQueueSize,omitempty" json:"taskQueueSize,omitempty"`
}

// WorkerPoolStats contains statistics about the worker pool.
type WorkerPoolStats struct {
	// CurrentSize is the current number of active workers.
	CurrentSize int
	// MinSize is the configured minimum number of workers.
	MinSize int
	// MaxSize is the configured maximum number of workers.
	MaxSize int
	// TotalExecuted is the total number of successfully executed tasks.
	TotalExecuted int64
	// TotalFailed is the total number of failed tasks.
	TotalFailed int64
	// PendingTasks is the number of tasks waiting in the queue.
	PendingTasks int
}

// worker represents a single worker goroutine.
type worker struct {
	id      int
	pool    *WorkerPool
	stopCh  chan struct{}
	stopped atomic.Bool
}

// NewWorkerPool creates a new worker pool with the given configuration.
func NewWorkerPool(config WorkerPoolConfig) *WorkerPool {
	if config.MinSize <= 0 {
		config.MinSize = 1
	}
	if config.MaxSize <= 0 {
		config.MaxSize = 100
	}
	if config.MaxSize < config.MinSize {
		config.MaxSize = config.MinSize
	}
	if config.InitialSize <= 0 {
		config.InitialSize = config.MinSize
	}
	if config.InitialSize > config.MaxSize {
		config.InitialSize = config.MaxSize
	}
	if config.TaskQueueSize <= 0 {
		config.TaskQueueSize = config.MaxSize * 2
	}

	pool := &WorkerPool{
		minSize:  config.MinSize,
		maxSize:  config.MaxSize,
		workers:  make([]*worker, 0, config.MaxSize),
		taskCh:   make(chan Task, config.TaskQueueSize),
		resultCh: make(chan TaskResult, config.TaskQueueSize),
		stopCh:   make(chan struct{}),
	}

	pool.currentSize.Store(int32(config.InitialSize))

	return pool
}

// Start starts the worker pool with the configured initial size.
func (p *WorkerPool) Start(ctx context.Context) {
	if p.isRunning.Swap(true) {
		return // Already running
	}

	// Store context and recreate stopCh for reusability
	p.ctx = ctx
	p.stopCh = make(chan struct{})

	initialSize := int(p.currentSize.Load())
	p.workersMu.Lock()
	// Clear workers slice for restart support (keep capacity)
	p.workers = p.workers[:0]
	for i := range initialSize {
		w := p.createWorker(i)
		p.workers = append(p.workers, w)
		p.wg.Add(1)
		go w.run(ctx)
	}
	p.workersMu.Unlock()
}

// Stop stops all workers and waits for them to finish.
func (p *WorkerPool) Stop() {
	if !p.isRunning.Swap(false) {
		return // Not running
	}

	close(p.stopCh)

	// Stop all workers
	p.workersMu.Lock()
	for _, w := range p.workers {
		w.stop()
	}
	p.workersMu.Unlock()

	p.wg.Wait()
}

// Submit submits a task to the worker pool.
// Returns false if the pool is stopped or the task queue is full.
func (p *WorkerPool) Submit(task Task) bool {
	if !p.isRunning.Load() {
		return false
	}

	select {
	case p.taskCh <- task:
		return true
	default:
		return false
	}
}

// SubmitWait submits a task and blocks until it can be queued.
// Returns an error if the context is cancelled or the pool is stopped.
func (p *WorkerPool) SubmitWait(ctx context.Context, task Task) error {
	if !p.isRunning.Load() {
		return context.Canceled
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-p.stopCh:
		return context.Canceled
	case p.taskCh <- task:
		return nil
	}
}

// AdjustSize adjusts the number of workers to the target size.
// The size is clamped to [MinSize, MaxSize].
func (p *WorkerPool) AdjustSize(targetSize int) {
	if !p.isRunning.Load() {
		return
	}

	// Clamp to valid range
	if targetSize < p.minSize {
		targetSize = p.minSize
	}
	if targetSize > p.maxSize {
		targetSize = p.maxSize
	}

	p.workersMu.Lock()
	// Use actual slice length inside lock for thread safety
	currentSize := len(p.workers)
	if targetSize == currentSize {
		p.workersMu.Unlock()
		return
	}

	var workersToStop []*worker

	if targetSize > currentSize {
		// Scale up: add workers using stored context
		ctx := p.ctx
		if ctx == nil {
			ctx = context.Background()
		}
		for i := currentSize; i < targetSize; i++ {
			w := p.createWorker(i)
			p.workers = append(p.workers, w)
			p.wg.Add(1)
			go w.run(ctx)
		}
	} else {
		// Scale down: collect workers to stop
		workersToStop = make([]*worker, 0, currentSize-targetSize)
		for len(p.workers) > targetSize {
			idx := len(p.workers) - 1
			workersToStop = append(workersToStop, p.workers[idx])
			p.workers = p.workers[:idx]
		}
	}

	p.currentSize.Store(int32(targetSize))
	p.workersMu.Unlock()

	// Stop workers outside the lock to prevent potential deadlock
	for _, w := range workersToStop {
		w.stop()
	}
}

// CurrentSize returns the current number of workers.
func (p *WorkerPool) CurrentSize() int {
	return int(p.currentSize.Load())
}

// MinSize returns the minimum number of workers.
func (p *WorkerPool) MinSize() int {
	return p.minSize
}

// MaxSize returns the maximum number of workers.
func (p *WorkerPool) MaxSize() int {
	return p.maxSize
}

// Stats returns statistics about the worker pool.
func (p *WorkerPool) Stats() WorkerPoolStats {
	return WorkerPoolStats{
		CurrentSize:   int(p.currentSize.Load()),
		MinSize:       p.minSize,
		MaxSize:       p.maxSize,
		TotalExecuted: p.totalExecuted.Load(),
		TotalFailed:   p.totalFailed.Load(),
		PendingTasks:  len(p.taskCh),
	}
}

// Results returns a channel that receives task results.
func (p *WorkerPool) Results() <-chan TaskResult {
	return p.resultCh
}

// createWorker creates a new worker instance.
func (p *WorkerPool) createWorker(id int) *worker {
	return &worker{
		id:     id,
		pool:   p,
		stopCh: make(chan struct{}),
	}
}

// run is the main worker loop.
func (w *worker) run(ctx context.Context) {
	defer w.pool.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.pool.stopCh:
			return
		case <-w.stopCh:
			return
		case task, ok := <-w.pool.taskCh:
			if !ok {
				return
			}
			if w.stopped.Load() {
				return
			}

			// Execute the task
			err := task(ctx)

			if err != nil {
				w.pool.totalFailed.Add(1)
			} else {
				w.pool.totalExecuted.Add(1)
			}

			// Send result (non-blocking)
			select {
			case w.pool.resultCh <- TaskResult{Err: err}:
			default:
			}
		}
	}
}

// stop signals the worker to stop.
func (w *worker) stop() {
	if w.stopped.Swap(true) {
		return
	}
	close(w.stopCh)
}
