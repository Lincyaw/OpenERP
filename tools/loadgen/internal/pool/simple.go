// Package pool provides the parameter pool implementation for the load generator.
// This file implements SimpleParameterPool, a basic non-sharded parameter pool.
package pool

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/example/erp/tools/loadgen/internal/circuit"
)

// SimpleParameterPool is a basic implementation of ParameterPool.
// It uses a single mutex for all operations, making it simpler but less
// performant under high concurrency compared to ShardedPool.
// Use this for testing, low-concurrency scenarios, or as a reference implementation.
type SimpleParameterPool struct {
	mu     sync.RWMutex
	pools  map[circuit.SemanticType][]Value
	config PoolConfig
	closed atomic.Bool

	// Statistics (atomic counters)
	totalAdds        atomic.Int64
	totalGets        atomic.Int64
	totalHits        atomic.Int64
	totalMisses      atomic.Int64
	totalEvictions   atomic.Int64
	totalExpirations atomic.Int64

	// nowFunc for testing - protected by nowFuncMu
	nowFuncMu sync.RWMutex
	nowFunc   func() time.Time

	// cleanupDone signals the cleanup goroutine to stop
	cleanupDone chan struct{}

	// rng for random selection - protected by rngMu
	rngMu sync.Mutex
	rng   *rand.Rand
}

// NewSimpleParameterPool creates a new simple parameter pool.
func NewSimpleParameterPool(config *PoolConfig) *SimpleParameterPool {
	cfg := DefaultPoolConfig()
	if config != nil {
		if config.MaxValuesPerType > 0 {
			cfg.MaxValuesPerType = config.MaxValuesPerType
		}
		if config.DefaultTTL > 0 {
			cfg.DefaultTTL = config.DefaultTTL
		}
		if config.EvictionPolicy != "" {
			cfg.EvictionPolicy = config.EvictionPolicy
		}
		if config.CleanupInterval > 0 {
			cfg.CleanupInterval = config.CleanupInterval
		}
	}

	pool := &SimpleParameterPool{
		pools:       make(map[circuit.SemanticType][]Value),
		config:      cfg,
		cleanupDone: make(chan struct{}),
		nowFunc:     time.Now,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	// Start cleanup goroutine
	go pool.cleanupLoop()

	return pool
}

// Add implements ParameterPool.Add.
func (p *SimpleParameterPool) Add(semantic circuit.SemanticType, value any, source ValueSource) {
	p.AddWithTTL(semantic, value, source, p.config.DefaultTTL)
}

// AddWithTTL implements ParameterPool.AddWithTTL.
func (p *SimpleParameterPool) AddWithTTL(semantic circuit.SemanticType, value any, source ValueSource, ttl time.Duration) {
	if p.closed.Load() {
		return
	}

	now := p.getNow()
	val := Value{
		Data:         value,
		SemanticType: semantic,
		Source:       source,
		CreatedAt:    now,
	}

	if ttl > 0 {
		val.ExpiresAt = now.Add(ttl)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	values := p.pools[semantic]

	// Check if we need to evict
	if len(values) >= p.config.MaxValuesPerType {
		// FIFO eviction: remove the oldest (first) element
		values = values[1:]
		p.totalEvictions.Add(1)
	}

	p.pools[semantic] = append(values, val)
	p.totalAdds.Add(1)
}

// Get implements ParameterPool.Get.
// Returns a random non-expired value for the given semantic type.
func (p *SimpleParameterPool) Get(semantic circuit.SemanticType) (*Value, error) {
	if p.closed.Load() {
		return nil, ErrPoolClosed
	}

	p.totalGets.Add(1)

	p.mu.RLock()
	defer p.mu.RUnlock()

	values, ok := p.pools[semantic]
	if !ok || len(values) == 0 {
		p.totalMisses.Add(1)
		return nil, ErrNoValue
	}

	// Collect valid (non-expired) values
	now := p.getNow()
	validIndices := make([]int, 0, len(values))
	for i := range values {
		if values[i].ExpiresAt.IsZero() || values[i].ExpiresAt.After(now) {
			validIndices = append(validIndices, i)
		}
	}

	if len(validIndices) == 0 {
		p.totalMisses.Add(1)
		return nil, ErrNoValue
	}

	// Select a random valid value
	p.rngMu.Lock()
	idx := validIndices[p.rng.Intn(len(validIndices))]
	p.rngMu.Unlock()
	val := values[idx]

	p.totalHits.Add(1)
	return &val, nil
}

// GetAll implements ParameterPool.GetAll.
func (p *SimpleParameterPool) GetAll(semantic circuit.SemanticType) []Value {
	if p.closed.Load() {
		return nil
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	values, ok := p.pools[semantic]
	if !ok {
		return nil
	}

	// Filter out expired values
	now := p.getNow()
	result := make([]Value, 0, len(values))
	for _, v := range values {
		if v.ExpiresAt.IsZero() || v.ExpiresAt.After(now) {
			result = append(result, v)
		}
	}

	return result
}

// Size implements ParameterPool.Size.
func (p *SimpleParameterPool) Size(semantic circuit.SemanticType) int {
	if p.closed.Load() {
		return 0
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	values, ok := p.pools[semantic]
	if !ok {
		return 0
	}

	// Count non-expired values
	now := p.getNow()
	count := 0
	for _, v := range values {
		if v.ExpiresAt.IsZero() || v.ExpiresAt.After(now) {
			count++
		}
	}

	return count
}

// TotalSize implements ParameterPool.TotalSize.
func (p *SimpleParameterPool) TotalSize() int {
	if p.closed.Load() {
		return 0
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	now := p.getNow()
	total := 0
	for _, values := range p.pools {
		for _, v := range values {
			if v.ExpiresAt.IsZero() || v.ExpiresAt.After(now) {
				total++
			}
		}
	}

	return total
}

// Types implements ParameterPool.Types.
func (p *SimpleParameterPool) Types() []circuit.SemanticType {
	if p.closed.Load() {
		return nil
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	types := make([]circuit.SemanticType, 0, len(p.pools))
	for st := range p.pools {
		types = append(types, st)
	}

	return types
}

// Clear implements ParameterPool.Clear.
func (p *SimpleParameterPool) Clear(semantic *circuit.SemanticType) {
	if p.closed.Load() {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if semantic == nil {
		// Clear all
		p.pools = make(map[circuit.SemanticType][]Value)
	} else {
		// Clear specific type
		delete(p.pools, *semantic)
	}
}

// Cleanup implements ParameterPool.Cleanup.
func (p *SimpleParameterPool) Cleanup() {
	if p.closed.Load() {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	now := p.getNow()
	var totalExpired int64

	for semantic, values := range p.pools {
		validValues := make([]Value, 0, len(values))
		for _, v := range values {
			if v.ExpiresAt.IsZero() || v.ExpiresAt.After(now) {
				validValues = append(validValues, v)
			} else {
				totalExpired++
			}
		}
		if len(validValues) == 0 {
			delete(p.pools, semantic)
		} else {
			p.pools[semantic] = validValues
		}
	}

	p.totalExpirations.Add(totalExpired)
}

// Stats implements ParameterPool.Stats.
func (p *SimpleParameterPool) Stats() Stats {
	stats := Stats{
		ValuesByType:     make(map[circuit.SemanticType]int),
		TotalAdds:        p.totalAdds.Load(),
		TotalGets:        p.totalGets.Load(),
		TotalHits:        p.totalHits.Load(),
		TotalMisses:      p.totalMisses.Load(),
		TotalEvictions:   p.totalEvictions.Load(),
		TotalExpirations: p.totalExpirations.Load(),
	}

	if stats.TotalGets > 0 {
		stats.HitRate = float64(stats.TotalHits) / float64(stats.TotalGets)
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	now := p.getNow()
	for semantic, values := range p.pools {
		count := 0
		for _, v := range values {
			if v.ExpiresAt.IsZero() || v.ExpiresAt.After(now) {
				count++
			}
		}
		if count > 0 {
			stats.ValuesByType[semantic] = count
			stats.TotalValues += int64(count)
		}
	}

	return stats
}

// Close implements ParameterPool.Close.
func (p *SimpleParameterPool) Close() {
	if p.closed.Swap(true) {
		return // Already closed
	}

	close(p.cleanupDone)

	p.mu.Lock()
	p.pools = nil
	p.mu.Unlock()
}

// IsClosed implements ParameterPool.IsClosed.
func (p *SimpleParameterPool) IsClosed() bool {
	return p.closed.Load()
}

// cleanupLoop periodically cleans up expired values.
func (p *SimpleParameterPool) cleanupLoop() {
	ticker := time.NewTicker(p.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.cleanupDone:
			return
		case <-ticker.C:
			p.Cleanup()
		}
	}
}

// WithNowFunc sets a custom time function for testing.
// This method is thread-safe and can be called at any time.
func (p *SimpleParameterPool) WithNowFunc(fn func() time.Time) *SimpleParameterPool {
	p.nowFuncMu.Lock()
	p.nowFunc = fn
	p.nowFuncMu.Unlock()
	return p
}

// getNow returns the current time using the configured time function.
func (p *SimpleParameterPool) getNow() time.Time {
	p.nowFuncMu.RLock()
	fn := p.nowFunc
	p.nowFuncMu.RUnlock()
	return fn()
}

// Ensure SimpleParameterPool implements ParameterPool interface
var _ ParameterPool = (*SimpleParameterPool)(nil)
