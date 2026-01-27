package pool

import (
	"sync"
	"testing"
	"time"

	"github.com/example/erp/tools/loadgen/internal/circuit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultPoolConfig(t *testing.T) {
	cfg := DefaultPoolConfig()

	assert.Equal(t, 10000, cfg.MaxValuesPerType)
	assert.Equal(t, 30*time.Minute, cfg.DefaultTTL)
	assert.Equal(t, "fifo", cfg.EvictionPolicy)
	assert.Equal(t, 32, cfg.ShardCount)
	assert.Equal(t, time.Minute, cfg.CleanupInterval)
}

func TestNewShardedPool(t *testing.T) {
	t.Run("with nil config uses defaults", func(t *testing.T) {
		pool := NewShardedPool(nil)
		defer pool.Close()

		assert.NotNil(t, pool)
		assert.Equal(t, 32, len(pool.shards))
	})

	t.Run("with custom config", func(t *testing.T) {
		pool := NewShardedPool(&PoolConfig{
			MaxValuesPerType: 100,
			DefaultTTL:       time.Minute,
			ShardCount:       8,
		})
		defer pool.Close()

		assert.NotNil(t, pool)
		assert.Equal(t, 8, len(pool.shards))
		assert.Equal(t, 100, pool.config.MaxValuesPerType)
		assert.Equal(t, time.Minute, pool.config.DefaultTTL)
	})

	t.Run("shard count rounds up to power of 2", func(t *testing.T) {
		pool := NewShardedPool(&PoolConfig{
			ShardCount: 5, // Should become 8
		})
		defer pool.Close()

		assert.Equal(t, 8, len(pool.shards))
	})
}

func TestShardedPool_Add_Get(t *testing.T) {
	t.Run("add and get single value", func(t *testing.T) {
		pool := NewShardedPool(nil)
		defer pool.Close()

		source := ValueSource{Endpoint: "/customers", ResponseField: "$.id"}
		pool.Add("entity.customer.id", "cust-123", source)

		val, err := pool.Get("entity.customer.id")
		require.NoError(t, err)
		assert.Equal(t, "cust-123", val.Data)
		assert.Equal(t, circuit.SemanticType("entity.customer.id"), val.SemanticType)
	})

	t.Run("get from empty pool returns error", func(t *testing.T) {
		pool := NewShardedPool(nil)
		defer pool.Close()

		val, err := pool.Get("entity.customer.id")
		assert.ErrorIs(t, err, ErrNoValue)
		assert.Nil(t, val)
	})

	t.Run("add multiple values", func(t *testing.T) {
		pool := NewShardedPool(nil)
		defer pool.Close()

		source := ValueSource{Endpoint: "/customers", ResponseField: "$.id"}
		pool.Add("entity.customer.id", "cust-1", source)
		pool.Add("entity.customer.id", "cust-2", source)
		pool.Add("entity.customer.id", "cust-3", source)

		assert.Equal(t, 3, pool.Size("entity.customer.id"))
	})

	t.Run("add to closed pool does nothing", func(t *testing.T) {
		pool := NewShardedPool(nil)
		pool.Close()

		source := ValueSource{}
		pool.Add("entity.customer.id", "cust-1", source)

		assert.Equal(t, 0, pool.Size("entity.customer.id"))
	})

	t.Run("get from closed pool returns error", func(t *testing.T) {
		pool := NewShardedPool(nil)
		pool.Close()

		val, err := pool.Get("entity.customer.id")
		assert.ErrorIs(t, err, ErrPoolClosed)
		assert.Nil(t, val)
	})
}

func TestShardedPool_AddWithTTL(t *testing.T) {
	pool := NewShardedPool(&PoolConfig{
		DefaultTTL:      time.Hour, // Long default
		CleanupInterval: time.Hour, // Disable auto cleanup for test
	})
	defer pool.Close()

	now := time.Now()
	pool.nowFunc = func() time.Time { return now }

	source := ValueSource{}
	pool.AddWithTTL("entity.customer.id", "cust-1", source, 100*time.Millisecond)

	// Value should exist initially
	val, err := pool.Get("entity.customer.id")
	require.NoError(t, err)
	assert.Equal(t, "cust-1", val.Data)

	// Advance time past TTL
	pool.nowFunc = func() time.Time { return now.Add(200 * time.Millisecond) }

	// Value should be expired
	val, err = pool.Get("entity.customer.id")
	assert.ErrorIs(t, err, ErrNoValue)
	assert.Nil(t, val)
}

func TestShardedPool_GetAll(t *testing.T) {
	pool := NewShardedPool(nil)
	defer pool.Close()

	source := ValueSource{}
	pool.Add("entity.customer.id", "cust-1", source)
	pool.Add("entity.customer.id", "cust-2", source)
	pool.Add("entity.customer.id", "cust-3", source)

	values := pool.GetAll("entity.customer.id")
	assert.Len(t, values, 3)

	// Verify all values are present
	dataSet := make(map[string]bool)
	for _, v := range values {
		dataSet[v.Data.(string)] = true
	}
	assert.True(t, dataSet["cust-1"])
	assert.True(t, dataSet["cust-2"])
	assert.True(t, dataSet["cust-3"])
}

func TestShardedPool_Size(t *testing.T) {
	pool := NewShardedPool(nil)
	defer pool.Close()

	assert.Equal(t, 0, pool.Size("entity.customer.id"))

	source := ValueSource{}
	pool.Add("entity.customer.id", "cust-1", source)
	assert.Equal(t, 1, pool.Size("entity.customer.id"))

	pool.Add("entity.customer.id", "cust-2", source)
	assert.Equal(t, 2, pool.Size("entity.customer.id"))
}

func TestShardedPool_TotalSize(t *testing.T) {
	pool := NewShardedPool(nil)
	defer pool.Close()

	assert.Equal(t, 0, pool.TotalSize())

	source := ValueSource{}
	pool.Add("entity.customer.id", "cust-1", source)
	pool.Add("entity.customer.id", "cust-2", source)
	pool.Add("entity.product.id", "prod-1", source)

	assert.Equal(t, 3, pool.TotalSize())
}

func TestShardedPool_Types(t *testing.T) {
	pool := NewShardedPool(nil)
	defer pool.Close()

	source := ValueSource{}
	pool.Add("entity.customer.id", "cust-1", source)
	pool.Add("entity.product.id", "prod-1", source)
	pool.Add("entity.warehouse.id", "wh-1", source)

	types := pool.Types()
	assert.Len(t, types, 3)

	typeSet := make(map[circuit.SemanticType]bool)
	for _, t := range types {
		typeSet[t] = true
	}
	assert.True(t, typeSet["entity.customer.id"])
	assert.True(t, typeSet["entity.product.id"])
	assert.True(t, typeSet["entity.warehouse.id"])
}

func TestShardedPool_Clear(t *testing.T) {
	t.Run("clear specific type", func(t *testing.T) {
		pool := NewShardedPool(nil)
		defer pool.Close()

		source := ValueSource{}
		pool.Add("entity.customer.id", "cust-1", source)
		pool.Add("entity.product.id", "prod-1", source)

		semantic := circuit.SemanticType("entity.customer.id")
		pool.Clear(&semantic)

		assert.Equal(t, 0, pool.Size("entity.customer.id"))
		assert.Equal(t, 1, pool.Size("entity.product.id"))
	})

	t.Run("clear all", func(t *testing.T) {
		pool := NewShardedPool(nil)
		defer pool.Close()

		source := ValueSource{}
		pool.Add("entity.customer.id", "cust-1", source)
		pool.Add("entity.product.id", "prod-1", source)

		pool.Clear(nil)

		assert.Equal(t, 0, pool.TotalSize())
	})
}

func TestShardedPool_Cleanup(t *testing.T) {
	pool := NewShardedPool(&PoolConfig{
		DefaultTTL:      time.Hour,
		CleanupInterval: time.Hour, // Disable auto cleanup
	})
	defer pool.Close()

	now := time.Now()
	pool.nowFunc = func() time.Time { return now }

	source := ValueSource{}
	pool.AddWithTTL("entity.customer.id", "cust-1", source, 100*time.Millisecond)
	pool.AddWithTTL("entity.customer.id", "cust-2", source, time.Hour)

	// Both values exist
	assert.Equal(t, 2, pool.Size("entity.customer.id"))

	// Advance time
	pool.nowFunc = func() time.Time { return now.Add(200 * time.Millisecond) }

	// Manual cleanup
	pool.Cleanup()

	// Only non-expired value should remain
	assert.Equal(t, 1, pool.Size("entity.customer.id"))
}

func TestShardedPool_Stats(t *testing.T) {
	pool := NewShardedPool(nil)
	defer pool.Close()

	source := ValueSource{}
	pool.Add("entity.customer.id", "cust-1", source)
	pool.Add("entity.customer.id", "cust-2", source)
	pool.Add("entity.product.id", "prod-1", source)

	// Some gets
	_, _ = pool.Get("entity.customer.id")
	_, _ = pool.Get("entity.customer.id")
	_, _ = pool.Get("nonexistent")

	stats := pool.Stats()

	assert.Equal(t, int64(3), stats.TotalValues)
	assert.Equal(t, int64(3), stats.TotalAdds)
	assert.Equal(t, int64(3), stats.TotalGets)
	assert.Equal(t, int64(2), stats.TotalHits)
	assert.Equal(t, int64(1), stats.TotalMisses)
	assert.Equal(t, 2, stats.ValuesByType["entity.customer.id"])
	assert.Equal(t, 1, stats.ValuesByType["entity.product.id"])
	assert.InDelta(t, 0.666, stats.HitRate, 0.01)
}

func TestShardedPool_Eviction(t *testing.T) {
	pool := NewShardedPool(&PoolConfig{
		MaxValuesPerType: 3,
		CleanupInterval:  time.Hour,
	})
	defer pool.Close()

	source := ValueSource{}
	pool.Add("entity.customer.id", "cust-1", source)
	pool.Add("entity.customer.id", "cust-2", source)
	pool.Add("entity.customer.id", "cust-3", source)
	pool.Add("entity.customer.id", "cust-4", source) // Should evict cust-1

	assert.Equal(t, 3, pool.Size("entity.customer.id"))

	values := pool.GetAll("entity.customer.id")
	dataSet := make(map[string]bool)
	for _, v := range values {
		dataSet[v.Data.(string)] = true
	}

	// cust-1 should be evicted (FIFO)
	assert.False(t, dataSet["cust-1"])
	assert.True(t, dataSet["cust-2"])
	assert.True(t, dataSet["cust-3"])
	assert.True(t, dataSet["cust-4"])

	// Check eviction stats
	stats := pool.Stats()
	assert.Equal(t, int64(1), stats.TotalEvictions)
}

func TestShardedPool_Concurrent(t *testing.T) {
	pool := NewShardedPool(nil)
	defer pool.Close()

	const numGoroutines = 100
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			source := ValueSource{}
			for j := 0; j < numOperations; j++ {
				semantic := circuit.SemanticType("entity.customer.id")
				pool.Add(semantic, id*numOperations+j, source)
				_, _ = pool.Get(semantic)
				_ = pool.Size(semantic)
			}
		}(i)
	}

	wg.Wait()

	// Should not panic and pool should have values
	assert.Greater(t, pool.TotalSize(), 0)
}

func TestShardedPool_Close(t *testing.T) {
	pool := NewShardedPool(nil)

	source := ValueSource{}
	pool.Add("entity.customer.id", "cust-1", source)

	assert.False(t, pool.IsClosed())

	pool.Close()

	assert.True(t, pool.IsClosed())

	// Operations on closed pool
	pool.Add("entity.customer.id", "cust-2", source)
	assert.Equal(t, 0, pool.Size("entity.customer.id"))

	_, err := pool.Get("entity.customer.id")
	assert.ErrorIs(t, err, ErrPoolClosed)

	// Double close should be safe
	pool.Close()
}

func TestRingBuffer(t *testing.T) {
	t.Run("basic operations", func(t *testing.T) {
		rb := newRingBuffer(5)

		val1 := Value{Data: "a"}
		val2 := Value{Data: "b"}
		val3 := Value{Data: "c"}

		evicted := rb.add(val1)
		assert.False(t, evicted)
		assert.Equal(t, 1, rb.size)

		evicted = rb.add(val2)
		assert.False(t, evicted)
		assert.Equal(t, 2, rb.size)

		evicted = rb.add(val3)
		assert.False(t, evicted)
		assert.Equal(t, 3, rb.size)
	})

	t.Run("eviction at capacity", func(t *testing.T) {
		rb := newRingBuffer(3)

		rb.add(Value{Data: "a"})
		rb.add(Value{Data: "b"})
		rb.add(Value{Data: "c"})

		evicted := rb.add(Value{Data: "d"})
		assert.True(t, evicted)
		assert.Equal(t, 3, rb.size)
	})

	t.Run("get random", func(t *testing.T) {
		rb := newRingBuffer(5)
		now := time.Now()

		rb.add(Value{Data: "a"})
		rb.add(Value{Data: "b"})

		val := rb.getRandom(now)
		assert.NotNil(t, val)
		assert.Contains(t, []string{"a", "b"}, val.Data)
	})

	t.Run("get all", func(t *testing.T) {
		rb := newRingBuffer(5)
		now := time.Now()

		rb.add(Value{Data: "a"})
		rb.add(Value{Data: "b"})
		rb.add(Value{Data: "c"})

		values := rb.getAll(now)
		assert.Len(t, values, 3)
	})

	t.Run("valid count with expiration", func(t *testing.T) {
		rb := newRingBuffer(5)
		now := time.Now()

		rb.add(Value{Data: "a", ExpiresAt: now.Add(-time.Hour)}) // Expired
		rb.add(Value{Data: "b", ExpiresAt: now.Add(time.Hour)})  // Valid
		rb.add(Value{Data: "c"})                                 // No expiration (valid)

		count := rb.validCount(now)
		assert.Equal(t, 2, count)
	})

	t.Run("remove expired", func(t *testing.T) {
		rb := newRingBuffer(5)
		now := time.Now()

		rb.add(Value{Data: "a", ExpiresAt: now.Add(-time.Hour)}) // Expired
		rb.add(Value{Data: "b", ExpiresAt: now.Add(-time.Hour)}) // Expired
		rb.add(Value{Data: "c", ExpiresAt: now.Add(time.Hour)})  // Valid

		removed := rb.removeExpired(now)
		assert.Equal(t, 2, removed)
		assert.Equal(t, 1, rb.size)
	})
}

func TestNextPowerOf2(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{0, 1},
		{1, 1},
		{2, 2},
		{3, 4},
		{4, 4},
		{5, 8},
		{7, 8},
		{8, 8},
		{9, 16},
		{16, 16},
		{17, 32},
	}

	for _, tt := range tests {
		result := nextPowerOf2(tt.input)
		assert.Equal(t, tt.expected, result, "nextPowerOf2(%d)", tt.input)
	}
}

func TestFnv32(t *testing.T) {
	// Verify consistent hashing
	h1 := fnv32("entity.customer.id")
	h2 := fnv32("entity.customer.id")
	assert.Equal(t, h1, h2)

	// Different strings should have different hashes (with high probability)
	h3 := fnv32("entity.product.id")
	assert.NotEqual(t, h1, h3)
}
