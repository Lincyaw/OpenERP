// Package warmup implements the warmup phase for the load generator.
package warmup

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/example/erp/tools/loadgen/internal/circuit"
	"github.com/example/erp/tools/loadgen/internal/pool"
)

// ProducerFunc is a function that executes a producer endpoint and returns produced values.
// The function should execute the API call and return the values extracted from the response.
type ProducerFunc func(ctx context.Context, semantic circuit.SemanticType) ([]any, error)

// LoginFunc is a function that performs authentication and returns a token.
type LoginFunc func(ctx context.Context) (token string, err error)

// ProgressCallback is called to report warmup progress.
type ProgressCallback func(progress Progress)

// Progress represents the current warmup progress.
type Progress struct {
	// Phase is the current phase: "login", "fill", "verify"
	Phase string

	// CurrentType is the semantic type currently being filled.
	CurrentType circuit.SemanticType

	// TypeIndex is the index of the current type in the fill list.
	TypeIndex int

	// TotalTypes is the total number of types to fill.
	TotalTypes int

	// Iteration is the current iteration number (1-based).
	Iteration int

	// TotalIterations is the total number of iterations configured.
	TotalIterations int

	// PoolStatus maps semantic types to their current pool size.
	PoolStatus map[circuit.SemanticType]int

	// MinPoolSize is the minimum required pool size.
	MinPoolSize int

	// Errors is the count of errors encountered so far.
	Errors int

	// StartTime is when the warmup started.
	StartTime time.Time

	// ElapsedTime is the time elapsed since start.
	ElapsedTime time.Duration

	// Message is an optional status message.
	Message string
}

// Result represents the result of the warmup phase.
type Result struct {
	// Success indicates whether warmup completed successfully.
	Success bool

	// Token is the authentication token obtained during warmup.
	Token string

	// PoolSizes maps semantic types to their final pool size.
	PoolSizes map[circuit.SemanticType]int

	// TotalValues is the total number of values in the pool after warmup.
	TotalValues int

	// Errors is the list of errors encountered during warmup.
	Errors []error

	// Duration is the total time taken for warmup.
	Duration time.Duration

	// Iterations is the number of iterations completed.
	Iterations int

	// SkippedTypes are semantic types that could not be filled due to missing producers.
	SkippedTypes []circuit.SemanticType
}

// ExecutorConfig holds configuration for the warmup executor.
type ExecutorConfig struct {
	// Warmup is the warmup configuration.
	Warmup Config

	// Pool is the parameter pool to fill.
	Pool pool.ParameterPool

	// Producers maps semantic types to their producer functions.
	Producers map[circuit.SemanticType]ProducerFunc

	// Login is the login function (optional).
	Login LoginFunc

	// OnProgress is called to report progress (optional).
	OnProgress ProgressCallback

	// Concurrency is the number of concurrent producer calls.
	// Default: 1 (sequential)
	Concurrency int
}

// Executor executes the warmup phase.
type Executor struct {
	config ExecutorConfig

	// Runtime state
	mu        sync.Mutex
	errors    []error
	startTime time.Time
	token     string
	running   atomic.Bool

	// Statistics
	iterationsCompleted atomic.Int32
	valuesProduced      atomic.Int64

	// For testing
	nowFunc func() time.Time
}

// NewExecutor creates a new warmup executor.
func NewExecutor(config ExecutorConfig) (*Executor, error) {
	// Validate configuration
	if config.Pool == nil {
		return nil, fmt.Errorf("%w: pool is required", ErrInvalidConfig)
	}

	if config.Warmup.Iterations > 0 && len(config.Warmup.Fill) > 0 && config.Producers == nil {
		return nil, fmt.Errorf("%w: producers map is required when fill is configured", ErrInvalidConfig)
	}

	// Apply defaults
	config.Warmup.ApplyDefaults()

	if err := config.Warmup.Validate(); err != nil {
		return nil, err
	}

	if config.Concurrency <= 0 {
		config.Concurrency = 1
	}

	return &Executor{
		config:  config,
		nowFunc: time.Now,
	}, nil
}

// Execute runs the warmup phase.
// Returns the result and any fatal error that prevented warmup from completing.
func (e *Executor) Execute(ctx context.Context) (*Result, error) {
	if e.running.Swap(true) {
		return nil, fmt.Errorf("%w: warmup already running", ErrInvalidConfig)
	}
	defer e.running.Store(false)

	e.mu.Lock()
	e.errors = nil
	e.startTime = e.nowFunc()
	e.token = ""
	e.mu.Unlock()
	e.iterationsCompleted.Store(0)
	e.valuesProduced.Store(0)

	// Create timeout context
	timeoutCtx := ctx
	if e.config.Warmup.Timeout > 0 {
		var cancel context.CancelFunc
		timeoutCtx, cancel = context.WithTimeout(ctx, e.config.Warmup.Timeout)
		defer cancel()
	}

	// Phase 1: Login
	if e.config.Login != nil {
		e.reportProgress(Progress{
			Phase:           "login",
			TotalTypes:      len(e.config.Warmup.Fill),
			TotalIterations: e.config.Warmup.Iterations,
			MinPoolSize:     e.config.Warmup.MinPoolSize,
			StartTime:       e.startTime,
			ElapsedTime:     e.nowFunc().Sub(e.startTime),
			Message:         "Authenticating...",
		})

		token, err := e.executeLogin(timeoutCtx)
		if err != nil {
			return e.buildResult(false), err
		}
		e.mu.Lock()
		e.token = token
		e.mu.Unlock()
	}

	// Phase 2: Fill parameter pool
	if len(e.config.Warmup.Fill) > 0 && e.config.Warmup.Iterations > 0 {
		if err := e.executeFill(timeoutCtx); err != nil {
			if !e.config.Warmup.ContinueOnError {
				return e.buildResult(false), err
			}
		}
	}

	// Phase 3: Verify pool status
	e.reportProgress(Progress{
		Phase:           "verify",
		TotalTypes:      len(e.config.Warmup.Fill),
		TotalIterations: e.config.Warmup.Iterations,
		MinPoolSize:     e.config.Warmup.MinPoolSize,
		PoolStatus:      e.getPoolStatus(),
		StartTime:       e.startTime,
		ElapsedTime:     e.nowFunc().Sub(e.startTime),
		Message:         "Verifying pool status...",
	})

	success, skipped := e.verifyPoolStatus()

	result := e.buildResult(success)
	result.SkippedTypes = skipped

	return result, nil
}

// ExecuteWarmupOnly runs only the warmup phase without starting the main test.
// This is useful for pre-populating the parameter pool.
func (e *Executor) ExecuteWarmupOnly(ctx context.Context) (*Result, error) {
	return e.Execute(ctx)
}

// Token returns the authentication token obtained during warmup.
func (e *Executor) Token() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.token
}

// Errors returns the list of errors encountered during warmup.
func (e *Executor) Errors() []error {
	e.mu.Lock()
	defer e.mu.Unlock()
	errs := make([]error, len(e.errors))
	copy(errs, e.errors)
	return errs
}

// executeLogin performs the login phase with retries.
func (e *Executor) executeLogin(ctx context.Context) (string, error) {
	var lastErr error

	for attempt := 0; attempt <= e.config.Warmup.RetryCount; attempt++ {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		if attempt > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(e.config.Warmup.RetryDelay):
			}
		}

		token, err := e.config.Login(ctx)
		if err == nil {
			return token, nil
		}

		lastErr = err
		// Don't record error for retry attempts - only record if all retries fail
	}

	// All retries failed - now record the error
	finalErr := fmt.Errorf("%w: %v", ErrLoginFailed, lastErr)
	e.recordError(finalErr)
	return "", finalErr
}

// executeFill executes the fill phase.
func (e *Executor) executeFill(ctx context.Context) error {
	for iteration := 1; iteration <= e.config.Warmup.Iterations; iteration++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		for typeIdx, semantic := range e.config.Warmup.Fill {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			e.reportProgress(Progress{
				Phase:           "fill",
				CurrentType:     semantic,
				TypeIndex:       typeIdx + 1,
				TotalTypes:      len(e.config.Warmup.Fill),
				Iteration:       iteration,
				TotalIterations: e.config.Warmup.Iterations,
				MinPoolSize:     e.config.Warmup.MinPoolSize,
				PoolStatus:      e.getPoolStatus(),
				StartTime:       e.startTime,
				ElapsedTime:     e.nowFunc().Sub(e.startTime),
				Message:         fmt.Sprintf("Filling %s (%d/%d)", semantic, iteration, e.config.Warmup.Iterations),
			})

			if err := e.executeProducer(ctx, semantic); err != nil {
				if !e.config.Warmup.ContinueOnError {
					return err
				}
			}
		}

		e.iterationsCompleted.Add(1)
	}

	return nil
}

// executeProducer executes a single producer with retries.
func (e *Executor) executeProducer(ctx context.Context, semantic circuit.SemanticType) error {
	producer, ok := e.config.Producers[semantic]
	if !ok {
		err := fmt.Errorf("no producer for semantic type: %s", semantic)
		e.recordError(err)
		return err
	}

	var lastErr error

	for attempt := 0; attempt <= e.config.Warmup.RetryCount; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(e.config.Warmup.RetryDelay):
			}
		}

		values, err := producer(ctx, semantic)
		if err == nil {
			// Add values to pool
			for _, val := range values {
				e.config.Pool.Add(semantic, val, pool.ValueSource{
					Endpoint:      "warmup",
					ResponseField: "",
				})
				e.valuesProduced.Add(1)
			}
			return nil
		}

		lastErr = err
		// Don't record error for retry attempts - only record if all retries fail
	}

	// All retries failed - now record the error
	finalErr := fmt.Errorf("producer %s failed after %d attempts: %w", semantic, e.config.Warmup.RetryCount+1, lastErr)
	e.recordError(finalErr)
	return finalErr
}

// verifyPoolStatus verifies that all required semantic types meet the minimum pool size.
// Returns success status and list of types that didn't meet the requirement.
func (e *Executor) verifyPoolStatus() (bool, []circuit.SemanticType) {
	if e.config.Warmup.MinPoolSize <= 0 {
		return true, nil
	}

	success := true
	var skipped []circuit.SemanticType

	for _, semantic := range e.config.Warmup.Fill {
		size := e.config.Pool.Size(semantic)
		if size < e.config.Warmup.MinPoolSize {
			success = false
			skipped = append(skipped, semantic)

			err := fmt.Errorf("pool for %s has only %d values, minimum required: %d",
				semantic, size, e.config.Warmup.MinPoolSize)
			e.recordError(err)
		}
	}

	return success, skipped
}

// getPoolStatus returns the current pool size for each semantic type in the fill list.
func (e *Executor) getPoolStatus() map[circuit.SemanticType]int {
	status := make(map[circuit.SemanticType]int)

	for _, semantic := range e.config.Warmup.Fill {
		status[semantic] = e.config.Pool.Size(semantic)
	}

	return status
}

// buildResult builds the warmup result.
func (e *Executor) buildResult(success bool) *Result {
	e.mu.Lock()
	defer e.mu.Unlock()

	poolSizes := make(map[circuit.SemanticType]int)
	for _, semantic := range e.config.Warmup.Fill {
		poolSizes[semantic] = e.config.Pool.Size(semantic)
	}

	errs := make([]error, len(e.errors))
	copy(errs, e.errors)

	return &Result{
		Success:     success && len(e.errors) == 0,
		Token:       e.token,
		PoolSizes:   poolSizes,
		TotalValues: e.config.Pool.TotalSize(),
		Errors:      errs,
		Duration:    e.nowFunc().Sub(e.startTime),
		Iterations:  int(e.iterationsCompleted.Load()),
	}
}

// recordError records an error.
func (e *Executor) recordError(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.errors = append(e.errors, err)
}

// reportProgress reports progress to the callback if configured.
func (e *Executor) reportProgress(progress Progress) {
	if e.config.OnProgress == nil {
		return
	}

	e.mu.Lock()
	progress.Errors = len(e.errors)
	e.mu.Unlock()

	e.config.OnProgress(progress)
}

// WithNowFunc sets a custom time function for testing.
// IMPORTANT: This method is NOT thread-safe. It must be called during initialization
// before Execute() is called. Calling it during execution will cause data races.
func (e *Executor) WithNowFunc(fn func() time.Time) *Executor {
	e.nowFunc = fn
	return e
}

// CheckPoolReady checks if the parameter pool meets the minimum requirements.
// This can be called before starting the main load test.
func CheckPoolReady(pool pool.ParameterPool, fill []circuit.SemanticType, minSize int) error {
	if minSize <= 0 {
		return nil
	}

	var errs []error
	for _, semantic := range fill {
		size := pool.Size(semantic)
		if size < minSize {
			errs = append(errs, fmt.Errorf("%s: %d/%d", semantic, size, minSize))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%w: insufficient pool sizes: %v", ErrPoolNotReady, errors.Join(errs...))
	}

	return nil
}
