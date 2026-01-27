package pool

import (
	"context"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"
)

// shard represents a single shard of the sharded parameter pool.
// Each shard contains a map of semantic types to ring buffers.
type shard struct {
	mu      sync.RWMutex
	buffers map[SemanticType]*RingBuffer

	// Per-shard statistics
	hitCount    atomic.Int64
	missCount   atomic.Int64
	addCount    atomic.Int64
	expireCount atomic.Int64
}

// ShardedParameterPool is a high-performance parameter pool that distributes
// values across multiple shards based on semantic type hash to reduce lock contention.
type ShardedParameterPool struct {
	shards    []*shard
	shardMask uint64 // shardCount - 1, used for fast modulo via bitwise AND

	config  PoolConfig
	startAt time.Time

	// Global statistics
	evictionCount atomic.Int64

	// Cleanup
	cleanupTicker *time.Ticker
	cleanupDone   chan struct{}
	closed        atomic.Bool
}

// NewShardedParameterPool creates a new ShardedParameterPool with the given configuration.
// ShardCount must be a power of 2 for optimal performance.
func NewShardedParameterPool(config PoolConfig) *ShardedParameterPool {
	// Ensure shard count is a power of 2
	shardCount := config.ShardCount
	if shardCount <= 0 {
		shardCount = 16
	}
	shardCount = nextPowerOfTwo(shardCount)

	shards := make([]*shard, shardCount)
	for i := range shards {
		shards[i] = &shard{
			buffers: make(map[SemanticType]*RingBuffer),
		}
	}

	pool := &ShardedParameterPool{
		shards:      shards,
		shardMask:   uint64(shardCount - 1),
		config:      config,
		startAt:     time.Now(),
		cleanupDone: make(chan struct{}),
	}

	// Start cleanup goroutine if configured
	if config.CleanupInterval > 0 {
		pool.cleanupTicker = time.NewTicker(config.CleanupInterval)
		go pool.cleanupLoop()
	}

	return pool
}

// nextPowerOfTwo returns the smallest power of 2 >= n.
func nextPowerOfTwo(n int) int {
	if n <= 1 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	return n + 1
}

// getShard returns the shard for the given semantic type.
func (p *ShardedParameterPool) getShard(semanticType SemanticType) *shard {
	h := fnv.New64a()
	h.Write([]byte(semanticType))
	idx := h.Sum64() & p.shardMask
	return p.shards[idx]
}

// getOrCreateBuffer returns the ring buffer for the given semantic type,
// creating it if it doesn't exist.
// Must be called with shard lock held for writing.
func (s *shard) getOrCreateBuffer(semanticType SemanticType, maxValues int, policy EvictionPolicy) *RingBuffer {
	rb, ok := s.buffers[semanticType]
	if !ok {
		if maxValues <= 0 {
			maxValues = 1000
		}
		rb = NewRingBuffer(maxValues, policy)
		s.buffers[semanticType] = rb
	}
	return rb
}

// Add adds a value to the pool.
func (p *ShardedParameterPool) Add(ctx context.Context, value *ParameterValue) (int, error) {
	if p.closed.Load() {
		return 0, ErrPoolClosed
	}

	s := p.getShard(value.SemanticType)

	s.mu.Lock()
	rb := s.getOrCreateBuffer(value.SemanticType, p.config.MaxValuesPerType, p.config.EvictionPolicy)
	evicted := rb.Add(value)
	s.addCount.Add(1)
	s.mu.Unlock()

	if evicted > 0 {
		p.evictionCount.Add(int64(evicted))
	}

	return evicted, nil
}

// Get retrieves a value for the given semantic type.
func (p *ShardedParameterPool) Get(ctx context.Context, semanticType SemanticType) (*ParameterValue, error) {
	if p.closed.Load() {
		return nil, ErrPoolClosed
	}

	s := p.getShard(semanticType)

	s.mu.RLock()
	rb, ok := s.buffers[semanticType]
	s.mu.RUnlock()

	if !ok {
		s.missCount.Add(1)
		return nil, nil
	}

	value := rb.Get()
	if value == nil || value.IsExpired() {
		s.missCount.Add(1)
		return nil, nil
	}

	s.hitCount.Add(1)
	return value, nil
}

// GetRandom retrieves a random value for the given semantic type.
func (p *ShardedParameterPool) GetRandom(ctx context.Context, semanticType SemanticType) (*ParameterValue, error) {
	if p.closed.Load() {
		return nil, ErrPoolClosed
	}

	s := p.getShard(semanticType)

	s.mu.RLock()
	rb, ok := s.buffers[semanticType]
	s.mu.RUnlock()

	if !ok {
		s.missCount.Add(1)
		return nil, nil
	}

	value := rb.GetRandom()
	if value == nil || value.IsExpired() {
		s.missCount.Add(1)
		return nil, nil
	}

	s.hitCount.Add(1)
	return value, nil
}

// GetAll retrieves all values for the given semantic type.
func (p *ShardedParameterPool) GetAll(ctx context.Context, semanticType SemanticType) ([]*ParameterValue, error) {
	if p.closed.Load() {
		return nil, ErrPoolClosed
	}

	s := p.getShard(semanticType)

	s.mu.RLock()
	rb, ok := s.buffers[semanticType]
	s.mu.RUnlock()

	if !ok {
		return nil, nil
	}

	values := rb.GetAll()

	// Filter out expired values
	result := make([]*ParameterValue, 0, len(values))
	for _, v := range values {
		if !v.IsExpired() {
			result = append(result, v)
		}
	}

	return result, nil
}

// Count returns the number of values for the given semantic type.
func (p *ShardedParameterPool) Count(ctx context.Context, semanticType SemanticType) (int, error) {
	if p.closed.Load() {
		return 0, ErrPoolClosed
	}

	s := p.getShard(semanticType)

	s.mu.RLock()
	rb, ok := s.buffers[semanticType]
	s.mu.RUnlock()

	if !ok {
		return 0, nil
	}

	return rb.Count(), nil
}

// Remove removes a specific value from the pool.
func (p *ShardedParameterPool) Remove(ctx context.Context, value *ParameterValue) (bool, error) {
	if p.closed.Load() {
		return false, ErrPoolClosed
	}

	s := p.getShard(value.SemanticType)

	s.mu.Lock()
	rb, ok := s.buffers[value.SemanticType]
	if !ok {
		s.mu.Unlock()
		return false, nil
	}
	removed := rb.Remove(value)
	s.mu.Unlock()

	return removed, nil
}

// Clear removes all values for the given semantic type.
func (p *ShardedParameterPool) Clear(ctx context.Context, semanticType SemanticType) (int, error) {
	if p.closed.Load() {
		return 0, ErrPoolClosed
	}

	s := p.getShard(semanticType)

	s.mu.Lock()
	rb, ok := s.buffers[semanticType]
	if !ok {
		s.mu.Unlock()
		return 0, nil
	}
	cleared := rb.Clear()
	delete(s.buffers, semanticType)
	s.mu.Unlock()

	return cleared, nil
}

// ClearAll removes all values from the pool.
func (p *ShardedParameterPool) ClearAll(ctx context.Context) error {
	if p.closed.Load() {
		return ErrPoolClosed
	}

	for _, s := range p.shards {
		s.mu.Lock()
		for st, rb := range s.buffers {
			rb.Clear()
			delete(s.buffers, st)
		}
		s.mu.Unlock()
	}

	return nil
}

// Cleanup removes expired values from the pool.
func (p *ShardedParameterPool) Cleanup(ctx context.Context) (int, error) {
	if p.closed.Load() {
		return 0, ErrPoolClosed
	}

	total := 0
	for _, s := range p.shards {
		s.mu.Lock()
		for _, rb := range s.buffers {
			removed := rb.RemoveExpired()
			total += removed
			s.expireCount.Add(int64(removed))
		}
		s.mu.Unlock()
	}

	return total, nil
}

// cleanupLoop periodically runs cleanup.
func (p *ShardedParameterPool) cleanupLoop() {
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
func (p *ShardedParameterPool) Stats(ctx context.Context) (Stats, error) {
	if p.closed.Load() {
		return Stats{}, ErrPoolClosed
	}

	stats := Stats{
		ValuesByType:  make(map[SemanticType]int64),
		EvictionCount: p.evictionCount.Load(),
		Uptime:        time.Since(p.startAt),
	}

	for _, s := range p.shards {
		s.mu.RLock()
		stats.HitCount += s.hitCount.Load()
		stats.MissCount += s.missCount.Load()
		stats.AddCount += s.addCount.Load()
		stats.ExpiredCount += s.expireCount.Load()

		for st, rb := range s.buffers {
			count := int64(rb.Count())
			stats.TotalValues += count
			stats.ValuesByType[st] += count
		}
		s.mu.RUnlock()
	}

	return stats, nil
}

// Types returns all semantic types that have values in the pool.
func (p *ShardedParameterPool) Types(ctx context.Context) ([]SemanticType, error) {
	if p.closed.Load() {
		return nil, ErrPoolClosed
	}

	types := make([]SemanticType, 0)
	seen := make(map[SemanticType]bool)

	for _, s := range p.shards {
		s.mu.RLock()
		for st, rb := range s.buffers {
			if rb.Count() > 0 && !seen[st] {
				types = append(types, st)
				seen[st] = true
			}
		}
		s.mu.RUnlock()
	}

	return types, nil
}

// Close releases resources held by the pool.
func (p *ShardedParameterPool) Close() error {
	if p.closed.Swap(true) {
		return ErrPoolClosed
	}

	if p.cleanupTicker != nil {
		p.cleanupTicker.Stop()
		close(p.cleanupDone)
	}

	return nil
}

// ShardCount returns the number of shards.
func (p *ShardedParameterPool) ShardCount() int {
	return len(p.shards)
}

// EvictionCount returns the total number of values that have been evicted.
func (p *ShardedParameterPool) EvictionCount() int64 {
	return p.evictionCount.Load()
}
