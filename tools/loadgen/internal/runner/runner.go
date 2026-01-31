// Package runner provides the main load test runner that orchestrates all components.
package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/example/erp/tools/loadgen/internal/circuit"
	"github.com/example/erp/tools/loadgen/internal/client"
	"github.com/example/erp/tools/loadgen/internal/config"
	"github.com/example/erp/tools/loadgen/internal/loadctrl"
	"github.com/example/erp/tools/loadgen/internal/pool"
)

// Runner is the main load test runner that orchestrates all components.
type Runner struct {
	cfg        *config.Config
	httpClient *client.Client
	rawClient  *http.Client
	pool       *pool.ShardedPool
	metrics    loadctrl.MetricsCollector
	controller *loadctrl.LoadController
	workerPool *loadctrl.WorkerPool

	// State
	running   atomic.Bool
	startTime time.Time
	stopCh    chan struct{}
	wg        sync.WaitGroup

	// Statistics
	stats Stats
}

// Stats holds overall test statistics.
type Stats struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	TotalLatency    int64 // nanoseconds
}

// New creates a new load test runner.
func New(cfg *config.Config) (*Runner, error) {
	// Create HTTP client
	httpClient, err := client.NewClient(cfg.Target, &cfg.Auth, nil)
	if err != nil {
		return nil, fmt.Errorf("creating HTTP client: %w", err)
	}

	// Create raw HTTP client for direct requests
	rawClient := &http.Client{
		Timeout: cfg.Target.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:        200,
			MaxIdleConnsPerHost: 50,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	// Create parameter pool
	poolCfg := &pool.PoolConfig{
		MaxValuesPerType: 10000,
		DefaultTTL:       30 * time.Minute,
		ShardCount:       32,
		CleanupInterval:  time.Minute,
	}
	paramPool := pool.NewShardedPool(poolCfg)

	// Create metrics collector
	metricsCollector := loadctrl.NewMetricsCollector()

	// Create traffic shaper
	shaper, err := createTrafficShaper(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating traffic shaper: %w", err)
	}

	// Create rate limiter
	rateLimiterCfg := loadctrl.RateLimiterConfig{
		Type:      "token_bucket",
		QPS:       cfg.TrafficShaper.BaseQPS,
		BurstSize: 10,
	}
	if cfg.RateLimiter.QPS > 0 {
		rateLimiterCfg.QPS = cfg.RateLimiter.QPS
	}
	if cfg.RateLimiter.BurstSize > 0 {
		rateLimiterCfg.BurstSize = cfg.RateLimiter.BurstSize
	}
	rateLimiter, err := loadctrl.NewRateLimiter(rateLimiterCfg)
	if err != nil {
		return nil, fmt.Errorf("creating rate limiter: %w", err)
	}

	// Create worker pool
	workerPoolCfg := loadctrl.WorkerPoolConfig{
		MinSize:       cfg.WorkerPool.MinSize,
		MaxSize:       cfg.WorkerPool.MaxSize,
		InitialSize:   cfg.WorkerPool.InitialSize,
		TaskQueueSize: cfg.WorkerPool.TaskQueueSize,
	}
	if workerPoolCfg.MinSize == 0 {
		workerPoolCfg.MinSize = 5
	}
	if workerPoolCfg.MaxSize == 0 {
		workerPoolCfg.MaxSize = 100
	}
	if workerPoolCfg.InitialSize == 0 {
		workerPoolCfg.InitialSize = 10
	}
	if workerPoolCfg.TaskQueueSize == 0 {
		workerPoolCfg.TaskQueueSize = 200
	}
	workerPool := loadctrl.NewWorkerPool(workerPoolCfg)

	// Create load controller
	controllerCfg := loadctrl.LoadControllerConfig{
		AdjustInterval: 100 * time.Millisecond,
	}
	controller := loadctrl.NewLoadController(rateLimiter, shaper, workerPool, metricsCollector, controllerCfg)

	return &Runner{
		cfg:        cfg,
		httpClient: httpClient,
		rawClient:  rawClient,
		pool:       paramPool,
		metrics:    metricsCollector,
		controller: controller,
		workerPool: workerPool,
		stopCh:     make(chan struct{}),
	}, nil
}

// Run executes the load test.
func (r *Runner) Run(ctx context.Context) error {
	if r.running.Swap(true) {
		return fmt.Errorf("runner is already running")
	}
	defer r.running.Store(false)

	r.startTime = time.Now()

	// Handle interrupt signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	// Create cancellable context
	ctx, cancel := context.WithTimeout(ctx, r.cfg.Duration)
	defer cancel()

	// Print banner
	r.printBanner()

	// Phase 1: Authentication
	fmt.Println("\n[Phase 1] Authentication...")
	if err := r.authenticate(ctx); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	fmt.Println("  ✓ Authenticated successfully")

	// Phase 2: Warmup
	fmt.Println("\n[Phase 2] Warmup...")
	if err := r.warmup(ctx); err != nil {
		fmt.Printf("  ⚠ Warmup completed with errors: %v\n", err)
	} else {
		fmt.Println("  ✓ Warmup complete")
	}
	r.printPoolStatus()

	// Phase 3: Load test
	fmt.Println("\n[Phase 3] Running load test...")
	fmt.Printf("  Duration: %v\n", r.cfg.Duration)
	fmt.Printf("  Target QPS: %.1f\n", r.cfg.TrafficShaper.BaseQPS)
	fmt.Println()

	// Start components
	r.controller.Start(ctx)

	// Run the main load generation loop
	r.wg.Add(1)
	go r.runLoadLoop(ctx)

	// Progress reporting
	r.wg.Add(1)
	go r.runProgressReporter(ctx)

	// Wait for completion or interrupt
	select {
	case <-ctx.Done():
		fmt.Println("\n  Test duration reached")
	case sig := <-sigCh:
		fmt.Printf("\n  Received signal: %v, stopping...\n", sig)
		cancel()
	}

	// Stop and cleanup
	close(r.stopCh)
	r.controller.Stop()
	r.wg.Wait()

	// Print final report
	r.printFinalReport()

	return nil
}

// authenticate performs the login flow.
func (r *Runner) authenticate(ctx context.Context) error {
	authMgr := r.httpClient.GetAuthManager()
	if authMgr == nil {
		return nil // No auth configured
	}
	// Auth was already performed during client creation
	return nil
}

// warmup fills the parameter pool with initial values.
func (r *Runner) warmup(ctx context.Context) error {
	if r.cfg.Warmup.Iterations == 0 {
		return nil
	}

	fillTypes := r.cfg.Warmup.Fill
	if len(fillTypes) == 0 {
		return nil
	}

	fmt.Printf("  Filling %d semantic types with %d iterations each\n", len(fillTypes), r.cfg.Warmup.Iterations)

	producers := r.findProducers()

	for _, semanticType := range fillTypes {
		producer, ok := producers[circuit.SemanticType(semanticType)]
		if !ok {
			fmt.Printf("    ⚠ No producer found for %s\n", semanticType)
			continue
		}

		fmt.Printf("    Filling %s via %s... ", semanticType, producer.Name)

		successCount := 0
		for i := 0; i < r.cfg.Warmup.Iterations; i++ {
			values, err := r.executeProducer(ctx, producer)
			if err != nil {
				continue
			}
			successCount += len(values)
		}
		fmt.Printf("%d values\n", successCount)
	}

	return nil
}

// findProducers returns a map of semantic types to their producer endpoints.
func (r *Runner) findProducers() map[circuit.SemanticType]*config.EndpointConfig {
	producers := make(map[circuit.SemanticType]*config.EndpointConfig)

	for i := range r.cfg.Endpoints {
		ep := &r.cfg.Endpoints[i]
		for _, prod := range ep.Produces {
			producers[circuit.SemanticType(prod.SemanticType)] = ep
		}
	}

	return producers
}

// executeProducer executes a producer endpoint and extracts values.
func (r *Runner) executeProducer(ctx context.Context, ep *config.EndpointConfig) ([]any, error) {
	req, err := r.buildRequest(ctx, ep)
	if err != nil {
		return nil, err
	}

	if authMgr := r.httpClient.GetAuthManager(); authMgr != nil && isAuthRequired(ep) {
		if err := authMgr.Authenticate(req); err != nil {
			return nil, fmt.Errorf("auth failed: %w", err)
		}
	}

	resp, err := r.rawClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("producer returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var values []any
	for _, prod := range ep.Produces {
		extracted := extractJSONPath(body, prod.JSONPath, prod.Multiple)
		for _, v := range extracted {
			r.pool.Add(
				circuit.SemanticType(prod.SemanticType),
				v,
				pool.ValueSource{
					Endpoint:      ep.Name,
					ResponseField: prod.JSONPath,
				},
			)
			values = append(values, v)
		}
	}

	return values, nil
}

// buildRequest builds an HTTP request for an endpoint.
func (r *Runner) buildRequest(ctx context.Context, ep *config.EndpointConfig) (*http.Request, error) {
	path := ep.Path

	// Replace path parameters
	for paramName, paramCfg := range ep.PathParams {
		var value string
		if paramCfg.SemanticType != "" {
			poolValue, err := r.pool.Get(circuit.SemanticType(paramCfg.SemanticType))
			if err == nil {
				value = fmt.Sprintf("%v", poolValue.Data)
			}
		}
		if value == "" && paramCfg.Value != "" {
			value = paramCfg.Value
		}
		if value == "" {
			value = fmt.Sprintf("gen-%d", time.Now().UnixNano()%100000)
		}
		path = strings.ReplaceAll(path, "{"+paramName+"}", value)
	}

	// Build query string
	var queryParts []string
	for paramName, paramCfg := range ep.QueryParams {
		var value string
		if paramCfg.SemanticType != "" {
			poolValue, err := r.pool.Get(circuit.SemanticType(paramCfg.SemanticType))
			if err == nil {
				value = fmt.Sprintf("%v", poolValue.Data)
			}
		}
		if value == "" && paramCfg.Value != "" {
			value = paramCfg.Value
		}
		if value != "" {
			queryParts = append(queryParts, fmt.Sprintf("%s=%s", paramName, value))
		}
	}

	fullURL := r.cfg.Target.BaseURL + path
	if len(queryParts) > 0 {
		fullURL += "?" + strings.Join(queryParts, "&")
	}

	var bodyReader io.Reader
	if ep.Body != "" {
		body := r.expandTemplate(ep.Body)
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, ep.Method, fullURL, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	for k, v := range r.cfg.Target.Headers {
		req.Header.Set(k, v)
	}
	for k, v := range ep.Headers {
		req.Header.Set(k, v)
	}

	return req, nil
}

// expandTemplate expands template placeholders in a string.
func (r *Runner) expandTemplate(template string) string {
	result := template

	for {
		start := strings.Index(result, "{{.")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "}}")
		if end == -1 {
			break
		}
		end += start

		placeholder := result[start+3 : end]
		var value string

		poolValue, err := r.pool.Get(circuit.SemanticType(placeholder))
		if err == nil {
			value = fmt.Sprintf("%v", poolValue.Data)
		} else {
			value = fmt.Sprintf("gen-%d", time.Now().UnixNano()%100000)
		}

		result = result[:start] + value + result[end+2:]
	}

	return result
}

// runLoadLoop is the main load generation loop.
func (r *Runner) runLoadLoop(ctx context.Context) {
	defer r.wg.Done()

	endpoints := r.cfg.GetEnabledEndpoints()
	if len(endpoints) == 0 {
		return
	}

	// Build weighted list
	var weightedEndpoints []config.EndpointConfig
	for _, ep := range endpoints {
		weight := ep.Weight
		if weight <= 0 {
			weight = 1
		}
		for i := 0; i < weight; i++ {
			weightedEndpoints = append(weightedEndpoints, ep)
		}
	}

	epIdx := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopCh:
			return
		default:
			// Wait for rate limiter
			if err := r.controller.Acquire(ctx); err != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}

			// Select an endpoint
			ep := weightedEndpoints[epIdx%len(weightedEndpoints)]
			epIdx++

			// Submit task to worker pool
			epCopy := ep
			task := func(taskCtx context.Context) error {
				r.executeEndpoint(taskCtx, &epCopy)
				return nil
			}

			if !r.workerPool.Submit(task) {
				// Queue full, execute synchronously
				r.executeEndpoint(ctx, &epCopy)
			}
		}
	}
}

// executeEndpoint executes a single endpoint request.
func (r *Runner) executeEndpoint(ctx context.Context, ep *config.EndpointConfig) {
	startTime := time.Now()

	req, err := r.buildRequest(ctx, ep)
	if err != nil {
		atomic.AddInt64(&r.stats.TotalRequests, 1)
		atomic.AddInt64(&r.stats.FailedRequests, 1)
		r.metrics.RecordLatency(time.Since(startTime))
		r.metrics.RecordError()
		return
	}

	if authMgr := r.httpClient.GetAuthManager(); authMgr != nil && isAuthRequired(ep) {
		if err := authMgr.Authenticate(req); err != nil {
			atomic.AddInt64(&r.stats.TotalRequests, 1)
			atomic.AddInt64(&r.stats.FailedRequests, 1)
			r.metrics.RecordLatency(time.Since(startTime))
			r.metrics.RecordError()
			return
		}
	}

	resp, err := r.rawClient.Do(req)
	latency := time.Since(startTime)

	atomic.AddInt64(&r.stats.TotalRequests, 1)
	atomic.AddInt64(&r.stats.TotalLatency, int64(latency))
	r.metrics.RecordLatency(latency)

	if err != nil {
		atomic.AddInt64(&r.stats.FailedRequests, 1)
		r.metrics.RecordError()
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	success := resp.StatusCode >= 200 && resp.StatusCode < 400

	if success {
		atomic.AddInt64(&r.stats.SuccessRequests, 1)
		r.metrics.RecordSuccess()

		if len(ep.Produces) > 0 {
			for _, prod := range ep.Produces {
				extracted := extractJSONPath(body, prod.JSONPath, prod.Multiple)
				for _, v := range extracted {
					r.pool.Add(
						circuit.SemanticType(prod.SemanticType),
						v,
						pool.ValueSource{
							Endpoint:      ep.Name,
							ResponseField: prod.JSONPath,
						},
					)
				}
			}
		}
	} else {
		atomic.AddInt64(&r.stats.FailedRequests, 1)
		r.metrics.RecordError()
	}
}

// runProgressReporter reports progress periodically.
func (r *Runner) runProgressReporter(ctx context.Context) {
	defer r.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.printProgress()
		}
	}
}

// printBanner prints the test banner.
func (r *Runner) printBanner() {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Printf("║  Load Generator: %-42s ║\n", truncate(r.cfg.Name, 42))
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Target:    %-48s ║\n", truncate(r.cfg.Target.BaseURL, 48))
	fmt.Printf("║  Duration:  %-48s ║\n", r.cfg.Duration)
	fmt.Printf("║  Endpoints: %-48d ║\n", len(r.cfg.GetEnabledEndpoints()))
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
}

// printPoolStatus prints the current pool status.
func (r *Runner) printPoolStatus() {
	stats := r.pool.Stats()
	fmt.Printf("  Pool status: %d total values across %d types\n",
		stats.TotalValues, len(stats.ValuesByType))

	if r.cfg.Output.Verbose {
		for st, count := range stats.ValuesByType {
			fmt.Printf("    - %s: %d\n", st, count)
		}
	}
}

// printProgress prints current progress.
func (r *Runner) printProgress() {
	elapsed := time.Since(r.startTime)
	total := atomic.LoadInt64(&r.stats.TotalRequests)
	success := atomic.LoadInt64(&r.stats.SuccessRequests)

	successRate := float64(0)
	if total > 0 {
		successRate = float64(success) / float64(total) * 100
	}

	qps := float64(total) / elapsed.Seconds()

	stats := r.metrics.GetStats()

	fmt.Printf("  [%s] Requests: %d | QPS: %.1f | Success: %.1f%% | P95: %s\n",
		elapsed.Round(time.Second), total, qps, successRate, stats.P95Latency)
}

// printFinalReport prints the final test report.
func (r *Runner) printFinalReport() {
	elapsed := time.Since(r.startTime)
	total := atomic.LoadInt64(&r.stats.TotalRequests)
	success := atomic.LoadInt64(&r.stats.SuccessRequests)
	failed := atomic.LoadInt64(&r.stats.FailedRequests)
	totalLatency := atomic.LoadInt64(&r.stats.TotalLatency)

	var avgLatency time.Duration
	if total > 0 {
		avgLatency = time.Duration(totalLatency / total)
	}

	successRate := float64(0)
	if total > 0 {
		successRate = float64(success) / float64(total) * 100
	}

	qps := float64(total) / elapsed.Seconds()

	stats := r.metrics.GetStats()

	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    LOAD TEST RESULTS                       ║")
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Duration:       %-42s ║\n", elapsed.Round(time.Second))
	fmt.Printf("║  Total Requests: %-42d ║\n", total)
	fmt.Printf("║  Successful:     %-42d ║\n", success)
	fmt.Printf("║  Failed:         %-42d ║\n", failed)
	fmt.Printf("║  QPS:            %-42.2f ║\n", qps)
	fmt.Printf("║  Success Rate:   %-41.2f%% ║\n", successRate)
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Avg Latency:    %-42s ║\n", avgLatency)
	fmt.Printf("║  P50 Latency:    %-42s ║\n", stats.P50Latency)
	fmt.Printf("║  P95 Latency:    %-42s ║\n", stats.P95Latency)
	fmt.Printf("║  P99 Latency:    %-42s ║\n", stats.P99Latency)
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
}

// createTrafficShaper creates a traffic shaper from config.
func createTrafficShaper(cfg *config.Config) (loadctrl.TrafficShaper, error) {
	shaperCfg := loadctrl.ShaperConfig{
		Type:    cfg.TrafficShaper.Type,
		BaseQPS: cfg.TrafficShaper.BaseQPS,
	}

	if shaperCfg.Type == "" {
		shaperCfg.Type = "step" // Default to step if configured
	}

	switch shaperCfg.Type {
	case "constant":
		shaperCfg.Type = "step"
		shaperCfg.Step = &loadctrl.StepConfig{
			Steps: []loadctrl.StepLevel{{QPS: shaperCfg.BaseQPS, Duration: cfg.Duration}},
		}
	case "step":
		if cfg.TrafficShaper.Step != nil && len(cfg.TrafficShaper.Step.Steps) > 0 {
			steps := make([]loadctrl.StepLevel, len(cfg.TrafficShaper.Step.Steps))
			for i, s := range cfg.TrafficShaper.Step.Steps {
				steps[i] = loadctrl.StepLevel{
					QPS:          s.QPS,
					Duration:     s.Duration,
					RampDuration: s.RampDuration,
				}
			}
			shaperCfg.Step = &loadctrl.StepConfig{
				Steps: steps,
				Loop:  cfg.TrafficShaper.Step.Loop,
			}
		} else {
			shaperCfg.Step = &loadctrl.StepConfig{
				Steps: []loadctrl.StepLevel{{QPS: shaperCfg.BaseQPS, Duration: cfg.Duration}},
			}
		}
	case "sine":
		shaperCfg.Amplitude = cfg.TrafficShaper.Amplitude
		shaperCfg.Period = cfg.TrafficShaper.Period
		if shaperCfg.Period == 0 {
			shaperCfg.Period = time.Minute
		}
	case "spike":
		shaperCfg.Spike = &loadctrl.SpikeConfig{
			SpikeQPS:      cfg.TrafficShaper.Spike.SpikeQPS,
			SpikeDuration: cfg.TrafficShaper.Spike.SpikeDuration,
			SpikeInterval: cfg.TrafficShaper.Spike.SpikeInterval,
		}
	}

	return loadctrl.NewTrafficShaper(shaperCfg)
}

// extractJSONPath extracts values from JSON using a simplified JSONPath.
func extractJSONPath(body []byte, path string, multiple bool) []any {
	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil
	}

	path = strings.TrimPrefix(path, "$.")
	path = strings.TrimPrefix(path, "$")

	parts := strings.Split(path, ".")
	current := data

	for i, part := range parts {
		if current == nil {
			return nil
		}

		if strings.HasSuffix(part, "[*]") {
			fieldName := strings.TrimSuffix(part, "[*]")

			if fieldName != "" {
				if m, ok := current.(map[string]any); ok {
					current = m[fieldName]
				} else {
					return nil
				}
			}

			arr, ok := current.([]any)
			if !ok {
				return nil
			}

			if i == len(parts)-1 {
				return arr
			}

			remainingPath := strings.Join(parts[i+1:], ".")
			var results []any
			for _, item := range arr {
				itemJSON, _ := json.Marshal(item)
				extracted := extractJSONPath(itemJSON, remainingPath, false)
				results = append(results, extracted...)
			}
			return results
		}

		if m, ok := current.(map[string]any); ok {
			current = m[part]
		} else if arr, ok := current.([]any); ok {
			var idx int
			if _, err := fmt.Sscanf(part, "[%d]", &idx); err == nil {
				if idx < len(arr) {
					current = arr[idx]
				} else {
					return nil
				}
			} else {
				return nil
			}
		} else {
			return nil
		}
	}

	if current == nil {
		return nil
	}

	if multiple {
		if arr, ok := current.([]any); ok {
			return arr
		}
	}

	return []any{current}
}

// isAuthRequired checks if an endpoint requires authentication.
func isAuthRequired(ep *config.EndpointConfig) bool {
	if ep.RequiresAuth != nil {
		return *ep.RequiresAuth
	}
	return true // Default to requiring auth
}

// truncate truncates a string to max length.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
