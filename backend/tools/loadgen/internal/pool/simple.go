package pool

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// SimpleParameterPool is a basic thread-safe parameter pool implementation.
// It uses a single lock for all operations, making it simpler but less
// performant under high concurrency compared to ShardedParameterPool.
type SimpleParameterPool struct {
	mu      sync.RWMutex
	values  map[SemanticType][]*ParameterValue
	config  PoolConfig
	startAt time.Time

	// Statistics
	hitCount      atomic.Int64
	missCount     atomic.Int64
	addCount      atomic.Int64
	evictionCount atomic.Int64
	expireCount   atomic.Int64

	// Cleanup
	cleanupTicker *time.Ticker
	cleanupDone   chan struct{}
	closed        atomic.Bool

	// Random source
	rng *rand.Rand
}

// NewSimpleParameterPool creates a new SimpleParameterPool with the given configuration.
func NewSimpleParameterPool(config PoolConfig) *SimpleParameterPool {
	pool := &SimpleParameterPool{
		values:      make(map[SemanticType][]*ParameterValue),
		config:      config,
		startAt:     time.Now(),
		cleanupDone: make(chan struct{}),
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	// Start cleanup goroutine if configured
	if config.CleanupInterval > 0 {
		pool.cleanupTicker = time.NewTicker(config.CleanupInterval)
		go pool.cleanupLoop()
	}

	return pool
}

// Add adds a value to the pool.
func (p *SimpleParameterPool) Add(ctx context.Context, value *ParameterValue) (int, error) {
	if p.closed.Load() {
		return 0, ErrPoolClosed
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.addCount.Add(1)
	evicted := 0

	values := p.values[value.SemanticType]

	// Check if we need to evict
	if p.config.MaxValuesPerType > 0 && len(values) >= p.config.MaxValuesPerType {
		evicted = p.evictOne(value.SemanticType)
	}

	p.values[value.SemanticType] = append(p.values[value.SemanticType], value)

	return evicted, nil
}

// evictOne removes one value according to the eviction policy.
// Must be called with lock held.
func (p *SimpleParameterPool) evictOne(semanticType SemanticType) int {
	values := p.values[semanticType]
	if len(values) == 0 {
		return 0
	}

	var evictIdx int

	switch p.config.EvictionPolicy {
	case EvictionFIFO:
		evictIdx = 0 // Evict the oldest (first)

	case EvictionLRU:
		// Find the least recently accessed
		evictIdx = 0
		oldestAccess := values[0].LastAccessedAt()
		for i, v := range values {
			if v.LastAccessedAt().Before(oldestAccess) {
				oldestAccess = v.LastAccessedAt()
				evictIdx = i
			}
		}

	case EvictionRandom:
		evictIdx = p.rng.Intn(len(values))
	}

	// Remove the value at evictIdx
	p.values[semanticType] = append(values[:evictIdx], values[evictIdx+1:]...)
	p.evictionCount.Add(1)

	return 1
}

// Get retrieves a value for the given semantic type.
func (p *SimpleParameterPool) Get(ctx context.Context, semanticType SemanticType) (*ParameterValue, error) {
	if p.closed.Load() {
		return nil, ErrPoolClosed
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	values := p.values[semanticType]
	if len(values) == 0 {
		p.missCount.Add(1)
		return nil, nil
	}

	// Return the first non-expired value
	for _, v := range values {
		if !v.IsExpired() {
			v.Touch()
			p.hitCount.Add(1)
			return v, nil
		}
	}

	p.missCount.Add(1)
	return nil, nil
}

// GetRandom retrieves a random value for the given semantic type.
func (p *SimpleParameterPool) GetRandom(ctx context.Context, semanticType SemanticType) (*ParameterValue, error) {
	if p.closed.Load() {
		return nil, ErrPoolClosed
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	values := p.values[semanticType]
	if len(values) == 0 {
		p.missCount.Add(1)
		return nil, nil
	}

	// Collect non-expired values
	validValues := make([]*ParameterValue, 0, len(values))
	for _, v := range values {
		if !v.IsExpired() {
			validValues = append(validValues, v)
		}
	}

	if len(validValues) == 0 {
		p.missCount.Add(1)
		return nil, nil
	}

	// Return a random value
	v := validValues[p.rng.Intn(len(validValues))]
	v.Touch()
	p.hitCount.Add(1)
	return v, nil
}

// GetAll retrieves all values for the given semantic type.
func (p *SimpleParameterPool) GetAll(ctx context.Context, semanticType SemanticType) ([]*ParameterValue, error) {
	if p.closed.Load() {
		return nil, ErrPoolClosed
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	values := p.values[semanticType]
	result := make([]*ParameterValue, 0, len(values))

	for _, v := range values {
		if !v.IsExpired() {
			result = append(result, v)
		}
	}

	return result, nil
}

// Count returns the number of values for the given semantic type.
func (p *SimpleParameterPool) Count(ctx context.Context, semanticType SemanticType) (int, error) {
	if p.closed.Load() {
		return 0, ErrPoolClosed
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	return len(p.values[semanticType]), nil
}

// Remove removes a specific value from the pool.
func (p *SimpleParameterPool) Remove(ctx context.Context, value *ParameterValue) (bool, error) {
	if p.closed.Load() {
		return false, ErrPoolClosed
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	values := p.values[value.SemanticType]
	for i, v := range values {
		if v == value {
			p.values[value.SemanticType] = append(values[:i], values[i+1:]...)
			return true, nil
		}
	}

	return false, nil
}

// Clear removes all values for the given semantic type.
func (p *SimpleParameterPool) Clear(ctx context.Context, semanticType SemanticType) (int, error) {
	if p.closed.Load() {
		return 0, ErrPoolClosed
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	count := len(p.values[semanticType])
	delete(p.values, semanticType)
	return count, nil
}

// ClearAll removes all values from the pool.
func (p *SimpleParameterPool) ClearAll(ctx context.Context) error {
	if p.closed.Load() {
		return ErrPoolClosed
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.values = make(map[SemanticType][]*ParameterValue)
	return nil
}

// Cleanup removes expired values from the pool.
func (p *SimpleParameterPool) Cleanup(ctx context.Context) (int, error) {
	if p.closed.Load() {
		return 0, ErrPoolClosed
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	total := 0
	for st, values := range p.values {
		newValues := make([]*ParameterValue, 0, len(values))
		for _, v := range values {
			if !v.IsExpired() {
				newValues = append(newValues, v)
			} else {
				total++
			}
		}
		if len(newValues) != len(values) {
			p.values[st] = newValues
		}
	}

	p.expireCount.Add(int64(total))
	return total, nil
}

// cleanupLoop periodically runs cleanup.
func (p *SimpleParameterPool) cleanupLoop() {
	for {
		select {
		case <-p.cleanupTicker.C:
			_, _ = p.Cleanup(context.Background())
		case <-p.cleanupDone:
			return
		}
	}
}

// Stats returns statistics about the pool.
func (p *SimpleParameterPool) Stats(ctx context.Context) (Stats, error) {
	if p.closed.Load() {
		return Stats{}, ErrPoolClosed
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := Stats{
		ValuesByType:  make(map[SemanticType]int64),
		HitCount:      p.hitCount.Load(),
		MissCount:     p.missCount.Load(),
		EvictionCount: p.evictionCount.Load(),
		ExpiredCount:  p.expireCount.Load(),
		AddCount:      p.addCount.Load(),
		Uptime:        time.Since(p.startAt),
	}

	for st, values := range p.values {
		count := int64(len(values))
		stats.TotalValues += count
		stats.ValuesByType[st] = count
	}

	return stats, nil
}

// Types returns all semantic types that have values in the pool.
func (p *SimpleParameterPool) Types(ctx context.Context) ([]SemanticType, error) {
	if p.closed.Load() {
		return nil, ErrPoolClosed
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	types := make([]SemanticType, 0, len(p.values))
	for st, values := range p.values {
		if len(values) > 0 {
			types = append(types, st)
		}
	}

	return types, nil
}

// Close releases resources held by the pool.
func (p *SimpleParameterPool) Close() error {
	if p.closed.Swap(true) {
		return ErrPoolClosed
	}

	if p.cleanupTicker != nil {
		p.cleanupTicker.Stop()
		close(p.cleanupDone)
	}

	return nil
}

// EvictionCount returns the total number of values that have been evicted.
func (p *SimpleParameterPool) EvictionCount() int64 {
	return p.evictionCount.Load()
}
