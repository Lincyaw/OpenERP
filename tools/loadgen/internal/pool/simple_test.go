package pool

import (
	"sync"
	"testing"
	"time"

	"github.com/example/erp/tools/loadgen/internal/circuit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSimpleParameterPool(t *testing.T) {
	t.Run("with nil config uses defaults", func(t *testing.T) {
		pool := NewSimpleParameterPool(nil)
		defer pool.Close()

		assert.NotNil(t, pool)
		assert.Equal(t, 10000, pool.config.MaxValuesPerType)
		assert.Equal(t, 30*time.Minute, pool.config.DefaultTTL)
	})

	t.Run("with custom config", func(t *testing.T) {
		pool := NewSimpleParameterPool(&PoolConfig{
			MaxValuesPerType: 100,
			DefaultTTL:       time.Minute,
		})
		defer pool.Close()

		assert.NotNil(t, pool)
		assert.Equal(t, 100, pool.config.MaxValuesPerType)
		assert.Equal(t, time.Minute, pool.config.DefaultTTL)
	})
}

func TestSimpleParameterPool_Add_Get(t *testing.T) {
	t.Run("add and get single value", func(t *testing.T) {
		pool := NewSimpleParameterPool(nil)
		defer pool.Close()

		source := ValueSource{Endpoint: "/customers", ResponseField: "$.id"}
		pool.Add("entity.customer.id", "cust-123", source)

		val, err := pool.Get("entity.customer.id")
		require.NoError(t, err)
		assert.Equal(t, "cust-123", val.Data)
		assert.Equal(t, circuit.SemanticType("entity.customer.id"), val.SemanticType)
	})

	t.Run("get from empty pool returns error", func(t *testing.T) {
		pool := NewSimpleParameterPool(nil)
		defer pool.Close()

		val, err := pool.Get("entity.customer.id")
		assert.ErrorIs(t, err, ErrNoValue)
		assert.Nil(t, val)
	})

	t.Run("add multiple values", func(t *testing.T) {
		pool := NewSimpleParameterPool(nil)
		defer pool.Close()

		source := ValueSource{Endpoint: "/customers", ResponseField: "$.id"}
		pool.Add("entity.customer.id", "cust-1", source)
		pool.Add("entity.customer.id", "cust-2", source)
		pool.Add("entity.customer.id", "cust-3", source)

		assert.Equal(t, 3, pool.Size("entity.customer.id"))
	})

	t.Run("add to closed pool does nothing", func(t *testing.T) {
		pool := NewSimpleParameterPool(nil)
		pool.Close()

		source := ValueSource{}
		pool.Add("entity.customer.id", "cust-1", source)

		assert.Equal(t, 0, pool.Size("entity.customer.id"))
	})

	t.Run("get from closed pool returns error", func(t *testing.T) {
		pool := NewSimpleParameterPool(nil)
		pool.Close()

		val, err := pool.Get("entity.customer.id")
		assert.ErrorIs(t, err, ErrPoolClosed)
		assert.Nil(t, val)
	})
}

func TestSimpleParameterPool_AddWithTTL(t *testing.T) {
	pool := NewSimpleParameterPool(&PoolConfig{
		DefaultTTL:      time.Hour, // Long default
		CleanupInterval: time.Hour, // Disable auto cleanup for test
	})
	defer pool.Close()

	now := time.Now()
	pool.WithNowFunc(func() time.Time { return now })

	source := ValueSource{}
	pool.AddWithTTL("entity.customer.id", "cust-1", source, 100*time.Millisecond)

	// Value should exist initially
	val, err := pool.Get("entity.customer.id")
	require.NoError(t, err)
	assert.Equal(t, "cust-1", val.Data)

	// Advance time past TTL
	pool.WithNowFunc(func() time.Time { return now.Add(200 * time.Millisecond) })

	// Value should be expired
	val, err = pool.Get("entity.customer.id")
	assert.ErrorIs(t, err, ErrNoValue)
	assert.Nil(t, val)
}

func TestSimpleParameterPool_GetAll(t *testing.T) {
	pool := NewSimpleParameterPool(nil)
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

func TestSimpleParameterPool_Size(t *testing.T) {
	pool := NewSimpleParameterPool(nil)
	defer pool.Close()

	assert.Equal(t, 0, pool.Size("entity.customer.id"))

	source := ValueSource{}
	pool.Add("entity.customer.id", "cust-1", source)
	assert.Equal(t, 1, pool.Size("entity.customer.id"))

	pool.Add("entity.customer.id", "cust-2", source)
	assert.Equal(t, 2, pool.Size("entity.customer.id"))
}

func TestSimpleParameterPool_TotalSize(t *testing.T) {
	pool := NewSimpleParameterPool(nil)
	defer pool.Close()

	assert.Equal(t, 0, pool.TotalSize())

	source := ValueSource{}
	pool.Add("entity.customer.id", "cust-1", source)
	pool.Add("entity.customer.id", "cust-2", source)
	pool.Add("entity.product.id", "prod-1", source)

	assert.Equal(t, 3, pool.TotalSize())
}

func TestSimpleParameterPool_Types(t *testing.T) {
	pool := NewSimpleParameterPool(nil)
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

func TestSimpleParameterPool_Clear(t *testing.T) {
	t.Run("clear specific type", func(t *testing.T) {
		pool := NewSimpleParameterPool(nil)
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
		pool := NewSimpleParameterPool(nil)
		defer pool.Close()

		source := ValueSource{}
		pool.Add("entity.customer.id", "cust-1", source)
		pool.Add("entity.product.id", "prod-1", source)

		pool.Clear(nil)

		assert.Equal(t, 0, pool.TotalSize())
	})
}

func TestSimpleParameterPool_Cleanup(t *testing.T) {
	pool := NewSimpleParameterPool(&PoolConfig{
		DefaultTTL:      time.Hour,
		CleanupInterval: time.Hour, // Disable auto cleanup
	})
	defer pool.Close()

	now := time.Now()
	pool.WithNowFunc(func() time.Time { return now })

	source := ValueSource{}
	pool.AddWithTTL("entity.customer.id", "cust-1", source, 100*time.Millisecond)
	pool.AddWithTTL("entity.customer.id", "cust-2", source, time.Hour)

	// Both values exist
	assert.Equal(t, 2, pool.Size("entity.customer.id"))

	// Advance time
	pool.WithNowFunc(func() time.Time { return now.Add(200 * time.Millisecond) })

	// Manual cleanup
	pool.Cleanup()

	// Only non-expired value should remain
	assert.Equal(t, 1, pool.Size("entity.customer.id"))

	// Check expiration stats
	stats := pool.Stats()
	assert.Equal(t, int64(1), stats.TotalExpirations)
}

func TestSimpleParameterPool_Stats(t *testing.T) {
	pool := NewSimpleParameterPool(nil)
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

func TestSimpleParameterPool_Eviction(t *testing.T) {
	pool := NewSimpleParameterPool(&PoolConfig{
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

func TestSimpleParameterPool_Concurrent(t *testing.T) {
	pool := NewSimpleParameterPool(nil)
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

func TestSimpleParameterPool_ConcurrentReadWrite(t *testing.T) {
	pool := NewSimpleParameterPool(&PoolConfig{
		MaxValuesPerType: 1000,
		CleanupInterval:  time.Hour,
	})
	defer pool.Close()

	const numWriters = 10
	const numReaders = 20
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numWriters + numReaders)

	// Writers
	for i := 0; i < numWriters; i++ {
		go func(id int) {
			defer wg.Done()
			source := ValueSource{Endpoint: "/test"}
			for j := 0; j < numOperations; j++ {
				pool.Add("entity.customer.id", id*numOperations+j, source)
				pool.Add("entity.product.id", id*numOperations+j, source)
			}
		}(i)
	}

	// Readers
	for i := 0; i < numReaders; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_, _ = pool.Get("entity.customer.id")
				_, _ = pool.Get("entity.product.id")
				_ = pool.GetAll("entity.customer.id")
				_ = pool.Size("entity.customer.id")
				_ = pool.TotalSize()
				_ = pool.Types()
				_ = pool.Stats()
			}
		}()
	}

	wg.Wait()

	// Verify pool is in consistent state
	stats := pool.Stats()
	assert.Greater(t, stats.TotalAdds, int64(0))
	assert.Greater(t, stats.TotalGets, int64(0))
}

func TestSimpleParameterPool_ConcurrentCleanup(t *testing.T) {
	pool := NewSimpleParameterPool(&PoolConfig{
		DefaultTTL:      50 * time.Millisecond,
		CleanupInterval: time.Hour, // Manual cleanup
	})
	defer pool.Close()

	const numGoroutines = 10
	const numOperations = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines + 1)

	// Writers adding values with short TTL
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			source := ValueSource{}
			for j := 0; j < numOperations; j++ {
				pool.AddWithTTL("entity.customer.id", id*numOperations+j, source, 10*time.Millisecond)
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	// Cleanup goroutine
	go func() {
		defer wg.Done()
		for i := 0; i < numOperations; i++ {
			pool.Cleanup()
			time.Sleep(5 * time.Millisecond)
		}
	}()

	wg.Wait()

	// Should not panic
	stats := pool.Stats()
	assert.GreaterOrEqual(t, stats.TotalExpirations, int64(0))
}

func TestSimpleParameterPool_Close(t *testing.T) {
	pool := NewSimpleParameterPool(nil)

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

func TestSimpleParameterPool_GetRandom(t *testing.T) {
	pool := NewSimpleParameterPool(nil)
	defer pool.Close()

	source := ValueSource{}
	// Add many values
	for i := 0; i < 100; i++ {
		pool.Add("entity.customer.id", i, source)
	}

	// Get multiple times and verify we get different values (randomness)
	seen := make(map[int]bool)
	for i := 0; i < 50; i++ {
		val, err := pool.Get("entity.customer.id")
		require.NoError(t, err)
		seen[val.Data.(int)] = true
	}

	// With 100 values and 50 gets, we should see multiple different values
	// (probability of getting same value 50 times is astronomically low)
	assert.Greater(t, len(seen), 1, "Get should return random values")
}

func TestSimpleParameterPool_MetadataStorage(t *testing.T) {
	pool := NewSimpleParameterPool(nil)
	defer pool.Close()

	source := ValueSource{
		Endpoint:      "/api/v1/customers",
		ResponseField: "$.data.id",
		RequestID:     "req-abc-123",
	}

	pool.Add("entity.customer.id", "cust-456", source)

	val, err := pool.Get("entity.customer.id")
	require.NoError(t, err)

	// Verify metadata is stored correctly
	assert.Equal(t, "/api/v1/customers", val.Source.Endpoint)
	assert.Equal(t, "$.data.id", val.Source.ResponseField)
	assert.Equal(t, "req-abc-123", val.Source.RequestID)
	assert.False(t, val.CreatedAt.IsZero())
}

func TestSimpleParameterPool_ImplementsInterface(t *testing.T) {
	// Compile-time check that SimpleParameterPool implements ParameterPool
	var _ ParameterPool = (*SimpleParameterPool)(nil)

	// Runtime check
	pool := NewSimpleParameterPool(nil)
	defer pool.Close()

	var pp ParameterPool = pool
	assert.NotNil(t, pp)
}
