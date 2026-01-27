// Package pool provides the parameter pool implementation for the load generator.
// The parameter pool stores and retrieves values by semantic type, enabling
// automatic connection of API endpoint inputs and outputs.
package pool

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/example/erp/tools/loadgen/internal/circuit"
)

// Errors returned by the pool package.
var (
	// ErrPoolClosed is returned when operations are attempted on a closed pool.
	ErrPoolClosed = errors.New("pool: pool is closed")
	// ErrNoValue is returned when no value is available for the requested type.
	ErrNoValue = errors.New("pool: no value available")
	// ErrTypeNotFound is returned when the semantic type is not found.
	ErrTypeNotFound = errors.New("pool: semantic type not found")
)

// Value represents a stored parameter value with metadata.
type Value struct {
	// Data is the actual parameter value (typically a string or number).
	Data any

	// SemanticType is the semantic classification of this value.
	SemanticType circuit.SemanticType

	// Source describes where this value came from.
	Source ValueSource

	// CreatedAt is when this value was added to the pool.
	CreatedAt time.Time

	// ExpiresAt is when this value expires (zero means no expiration).
	ExpiresAt time.Time

	// UsageCount tracks how many times this value has been retrieved.
	UsageCount int64
}

// ValueSource describes the origin of a parameter value.
type ValueSource struct {
	// Endpoint is the API endpoint that produced this value.
	Endpoint string

	// ResponseField is the JSONPath to the field in the response.
	ResponseField string

	// RequestID is the ID of the request that produced this value.
	RequestID string
}

// PoolConfig holds configuration for the parameter pool.
type PoolConfig struct {
	// MaxValuesPerType is the maximum number of values per semantic type.
	// Default: 10000
	MaxValuesPerType int

	// DefaultTTL is the default time-to-live for values.
	// Default: 30m (zero means no expiration)
	DefaultTTL time.Duration

	// EvictionPolicy determines how values are evicted when at capacity.
	// Supported: "lru", "fifo", "random"
	// Default: "fifo"
	EvictionPolicy string

	// ShardCount is the number of shards for concurrent access.
	// Default: 32
	ShardCount int

	// CleanupInterval is how often to clean up expired values.
	// Default: 1m
	CleanupInterval time.Duration
}

// DefaultPoolConfig returns the default pool configuration.
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxValuesPerType: 10000,
		DefaultTTL:       30 * time.Minute,
		EvictionPolicy:   "fifo",
		ShardCount:       32,
		CleanupInterval:  time.Minute,
	}
}

// Stats holds statistics about the parameter pool.
type Stats struct {
	// TotalValues is the total number of values across all types.
	TotalValues int64

	// ValuesByType is the count of values per semantic type.
	ValuesByType map[circuit.SemanticType]int

	// TotalAdds is the total number of Add operations.
	TotalAdds int64

	// TotalGets is the total number of Get operations.
	TotalGets int64

	// TotalHits is the number of successful Get operations.
	TotalHits int64

	// TotalMisses is the number of Get operations that found no value.
	TotalMisses int64

	// TotalEvictions is the number of values evicted.
	TotalEvictions int64

	// TotalExpirations is the number of values expired.
	TotalExpirations int64

	// HitRate is the cache hit rate (TotalHits / TotalGets).
	HitRate float64
}

// ParameterPool is the interface for parameter storage and retrieval.
type ParameterPool interface {
	// Add adds a value to the pool for the given semantic type.
	Add(semantic circuit.SemanticType, value any, source ValueSource)

	// AddWithTTL adds a value with a custom TTL.
	AddWithTTL(semantic circuit.SemanticType, value any, source ValueSource, ttl time.Duration)

	// Get retrieves a random value for the given semantic type.
	// Returns ErrNoValue if no value is available.
	Get(semantic circuit.SemanticType) (*Value, error)

	// GetAll retrieves all values for the given semantic type.
	GetAll(semantic circuit.SemanticType) []Value

	// Size returns the number of values for the given semantic type.
	Size(semantic circuit.SemanticType) int

	// TotalSize returns the total number of values across all types.
	TotalSize() int

	// Types returns all semantic types that have values in the pool.
	Types() []circuit.SemanticType

	// Clear removes all values for the given semantic type.
	// If semantic is nil, clears all values.
	Clear(semantic *circuit.SemanticType)

	// Cleanup removes expired values.
	Cleanup()

	// Stats returns pool statistics.
	Stats() Stats

	// Close closes the pool and releases resources.
	Close()

	// IsClosed returns true if the pool has been closed.
	IsClosed() bool
}

// ShardedPool is a sharded implementation of ParameterPool for high concurrency.
type ShardedPool struct {
	shards      []*shard
	shardMask   uint32
	config      PoolConfig
	closed      atomic.Bool
	cleanupDone chan struct{}

	// Statistics (atomic counters)
	totalAdds        atomic.Int64
	totalGets        atomic.Int64
	totalHits        atomic.Int64
	totalMisses      atomic.Int64
	totalEvictions   atomic.Int64
	totalExpirations atomic.Int64

	// nowFunc for testing
	nowFunc func() time.Time
}

// shard is a single partition of the pool.
type shard struct {
	mu    sync.RWMutex
	pools map[circuit.SemanticType]*ringBuffer
}

// ringBuffer is a circular buffer for storing values.
type ringBuffer struct {
	values   []Value
	head     int
	tail     int
	size     int
	capacity int
}

// NewShardedPool creates a new sharded parameter pool.
func NewShardedPool(config *PoolConfig) *ShardedPool {
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
		if config.ShardCount > 0 {
			cfg.ShardCount = config.ShardCount
		}
		if config.CleanupInterval > 0 {
			cfg.CleanupInterval = config.CleanupInterval
		}
	}

	// Ensure shard count is a power of 2 for efficient hashing
	shardCount := nextPowerOf2(cfg.ShardCount)
	cfg.ShardCount = shardCount

	shards := make([]*shard, shardCount)
	for i := range shards {
		shards[i] = &shard{
			pools: make(map[circuit.SemanticType]*ringBuffer),
		}
	}

	pool := &ShardedPool{
		shards:      shards,
		shardMask:   uint32(shardCount - 1),
		config:      cfg,
		cleanupDone: make(chan struct{}),
		nowFunc:     time.Now,
	}

	// Start cleanup goroutine
	go pool.cleanupLoop()

	return pool
}

// getShard returns the shard for the given semantic type.
func (p *ShardedPool) getShard(semantic circuit.SemanticType) *shard {
	h := fnv32(string(semantic))
	return p.shards[h&p.shardMask]
}

// Add implements ParameterPool.Add.
func (p *ShardedPool) Add(semantic circuit.SemanticType, value any, source ValueSource) {
	p.AddWithTTL(semantic, value, source, p.config.DefaultTTL)
}

// AddWithTTL implements ParameterPool.AddWithTTL.
func (p *ShardedPool) AddWithTTL(semantic circuit.SemanticType, value any, source ValueSource, ttl time.Duration) {
	if p.closed.Load() {
		return
	}

	now := p.nowFunc()
	val := Value{
		Data:         value,
		SemanticType: semantic,
		Source:       source,
		CreatedAt:    now,
	}

	if ttl > 0 {
		val.ExpiresAt = now.Add(ttl)
	}

	s := p.getShard(semantic)
	s.mu.Lock()
	defer s.mu.Unlock()

	rb, ok := s.pools[semantic]
	if !ok {
		rb = newRingBuffer(p.config.MaxValuesPerType)
		s.pools[semantic] = rb
	}

	evicted := rb.add(val)
	if evicted {
		p.totalEvictions.Add(1)
	}

	p.totalAdds.Add(1)
}

// Get implements ParameterPool.Get.
func (p *ShardedPool) Get(semantic circuit.SemanticType) (*Value, error) {
	if p.closed.Load() {
		return nil, ErrPoolClosed
	}

	p.totalGets.Add(1)

	s := p.getShard(semantic)
	s.mu.RLock()
	defer s.mu.RUnlock()

	rb, ok := s.pools[semantic]
	if !ok || rb.size == 0 {
		p.totalMisses.Add(1)
		return nil, ErrNoValue
	}

	// Get a random value
	val := rb.getRandom(p.nowFunc())
	if val == nil {
		p.totalMisses.Add(1)
		return nil, ErrNoValue
	}

	p.totalHits.Add(1)
	return val, nil
}

// GetAll implements ParameterPool.GetAll.
func (p *ShardedPool) GetAll(semantic circuit.SemanticType) []Value {
	if p.closed.Load() {
		return nil
	}

	s := p.getShard(semantic)
	s.mu.RLock()
	defer s.mu.RUnlock()

	rb, ok := s.pools[semantic]
	if !ok {
		return nil
	}

	return rb.getAll(p.nowFunc())
}

// Size implements ParameterPool.Size.
func (p *ShardedPool) Size(semantic circuit.SemanticType) int {
	if p.closed.Load() {
		return 0
	}

	s := p.getShard(semantic)
	s.mu.RLock()
	defer s.mu.RUnlock()

	rb, ok := s.pools[semantic]
	if !ok {
		return 0
	}

	return rb.validCount(p.nowFunc())
}

// TotalSize implements ParameterPool.TotalSize.
func (p *ShardedPool) TotalSize() int {
	if p.closed.Load() {
		return 0
	}

	var total int
	now := p.nowFunc()

	for _, s := range p.shards {
		s.mu.RLock()
		for _, rb := range s.pools {
			total += rb.validCount(now)
		}
		s.mu.RUnlock()
	}

	return total
}

// Types implements ParameterPool.Types.
func (p *ShardedPool) Types() []circuit.SemanticType {
	if p.closed.Load() {
		return nil
	}

	typeSet := make(map[circuit.SemanticType]struct{})

	for _, s := range p.shards {
		s.mu.RLock()
		for st := range s.pools {
			typeSet[st] = struct{}{}
		}
		s.mu.RUnlock()
	}

	types := make([]circuit.SemanticType, 0, len(typeSet))
	for st := range typeSet {
		types = append(types, st)
	}

	return types
}

// Clear implements ParameterPool.Clear.
func (p *ShardedPool) Clear(semantic *circuit.SemanticType) {
	if p.closed.Load() {
		return
	}

	if semantic == nil {
		// Clear all
		for _, s := range p.shards {
			s.mu.Lock()
			s.pools = make(map[circuit.SemanticType]*ringBuffer)
			s.mu.Unlock()
		}
	} else {
		// Clear specific type
		s := p.getShard(*semantic)
		s.mu.Lock()
		delete(s.pools, *semantic)
		s.mu.Unlock()
	}
}

// Cleanup implements ParameterPool.Cleanup.
func (p *ShardedPool) Cleanup() {
	if p.closed.Load() {
		return
	}

	now := p.nowFunc()
	var expired int64

	for _, s := range p.shards {
		s.mu.Lock()
		for _, rb := range s.pools {
			expired += int64(rb.removeExpired(now))
		}
		s.mu.Unlock()
	}

	p.totalExpirations.Add(expired)
}

// Stats implements ParameterPool.Stats.
func (p *ShardedPool) Stats() Stats {
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

	now := p.nowFunc()
	for _, s := range p.shards {
		s.mu.RLock()
		for st, rb := range s.pools {
			count := rb.validCount(now)
			stats.ValuesByType[st] = count
			stats.TotalValues += int64(count)
		}
		s.mu.RUnlock()
	}

	return stats
}

// Close implements ParameterPool.Close.
func (p *ShardedPool) Close() {
	if p.closed.Swap(true) {
		return // Already closed
	}

	close(p.cleanupDone)

	// Clear all data
	for _, s := range p.shards {
		s.mu.Lock()
		s.pools = nil
		s.mu.Unlock()
	}
}

// IsClosed implements ParameterPool.IsClosed.
func (p *ShardedPool) IsClosed() bool {
	return p.closed.Load()
}

// cleanupLoop periodically cleans up expired values.
func (p *ShardedPool) cleanupLoop() {
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
// IMPORTANT: This method is NOT thread-safe. It must be called during initialization
// before any concurrent operations begin. Calling it after the pool is in use
// will cause data races.
func (p *ShardedPool) WithNowFunc(fn func() time.Time) *ShardedPool {
	p.nowFunc = fn
	return p
}

// --- Ring Buffer Implementation ---

func newRingBuffer(capacity int) *ringBuffer {
	if capacity <= 0 {
		capacity = 1000
	}
	return &ringBuffer{
		values:   make([]Value, capacity),
		capacity: capacity,
	}
}

// add adds a value to the ring buffer, returning true if a value was evicted.
func (rb *ringBuffer) add(val Value) bool {
	evicted := rb.size == rb.capacity

	rb.values[rb.tail] = val
	rb.tail = (rb.tail + 1) % rb.capacity

	if rb.size < rb.capacity {
		rb.size++
	} else {
		// Move head forward (FIFO eviction)
		rb.head = (rb.head + 1) % rb.capacity
	}

	return evicted
}

// getRandom returns a random non-expired value from the buffer.
// Note: This method does not increment UsageCount to avoid data races under RLock.
// UsageCount tracking should be done by the caller if needed.
func (rb *ringBuffer) getRandom(now time.Time) *Value {
	if rb.size == 0 {
		return nil
	}

	// Simple approach: try from a random starting point
	// For better performance, we could maintain a separate list of valid indices
	start := int(now.UnixNano()) % rb.size
	for i := 0; i < rb.size; i++ {
		idx := (rb.head + (start+i)%rb.size) % rb.capacity
		val := &rb.values[idx]
		if val.ExpiresAt.IsZero() || val.ExpiresAt.After(now) {
			// Don't mutate UsageCount here to avoid data race under RLock
			return val
		}
	}

	return nil
}

// getAll returns all non-expired values.
func (rb *ringBuffer) getAll(now time.Time) []Value {
	if rb.size == 0 {
		return nil
	}

	result := make([]Value, 0, rb.size)
	for i := 0; i < rb.size; i++ {
		idx := (rb.head + i) % rb.capacity
		val := rb.values[idx]
		if val.ExpiresAt.IsZero() || val.ExpiresAt.After(now) {
			result = append(result, val)
		}
	}

	return result
}

// validCount returns the number of non-expired values.
func (rb *ringBuffer) validCount(now time.Time) int {
	if rb.size == 0 {
		return 0
	}

	count := 0
	for i := 0; i < rb.size; i++ {
		idx := (rb.head + i) % rb.capacity
		val := &rb.values[idx]
		if val.ExpiresAt.IsZero() || val.ExpiresAt.After(now) {
			count++
		}
	}

	return count
}

// removeExpired removes expired values and returns the count removed.
func (rb *ringBuffer) removeExpired(now time.Time) int {
	// For ring buffer, we don't actually remove individual items.
	// Instead, we compact the buffer by moving valid items forward.
	if rb.size == 0 {
		return 0
	}

	validValues := make([]Value, 0, rb.size)
	removed := 0

	for i := 0; i < rb.size; i++ {
		idx := (rb.head + i) % rb.capacity
		val := rb.values[idx]
		if val.ExpiresAt.IsZero() || val.ExpiresAt.After(now) {
			validValues = append(validValues, val)
		} else {
			removed++
		}
	}

	// Rebuild the buffer with only valid values
	rb.head = 0
	rb.size = len(validValues)
	rb.tail = rb.size % rb.capacity

	copy(rb.values, validValues)

	// Clear unused slots to allow garbage collection of any pointers in Value
	for i := rb.size; i < rb.capacity; i++ {
		rb.values[i] = Value{}
	}

	return removed
}

// --- Helper Functions ---

// fnv32 computes a simple 32-bit FNV-1a hash.
func fnv32(s string) uint32 {
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)

	h := uint32(offset32)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= prime32
	}
	return h
}

// nextPowerOf2 returns the smallest power of 2 >= n.
func nextPowerOf2(n int) int {
	if n <= 0 {
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
