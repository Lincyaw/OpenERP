package pool

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestShardedParameterPoolBasicOperations(t *testing.T) {
	config := DefaultPoolConfig()
	config.CleanupInterval = 0 // Disable auto cleanup for tests
	pool := NewShardedParameterPool(config)
	defer pool.Close()

	ctx := context.Background()

	// Test Add
	v := NewParameterValue("customer-123", SemanticTypeCustomerID, 0)
	evicted, err := pool.Add(ctx, v)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if evicted != 0 {
		t.Errorf("Evicted = %d, want 0", evicted)
	}

	// Test Get
	got, err := pool.Get(ctx, SemanticTypeCustomerID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil")
	}
	if got.Value != "customer-123" {
		t.Errorf("Value = %v, want customer-123", got.Value)
	}

	// Test Count
	count, err := pool.Count(ctx, SemanticTypeCustomerID)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Count = %d, want 1", count)
	}
}

func TestShardedParameterPoolMultipleTypes(t *testing.T) {
	config := DefaultPoolConfig()
	config.CleanupInterval = 0
	pool := NewShardedParameterPool(config)
	defer pool.Close()

	ctx := context.Background()

	// Add values of different types
	types := []SemanticType{
		SemanticTypeCustomerID,
		SemanticTypeProductID,
		SemanticTypeSalesOrderID,
		SemanticTypeWarehouseID,
	}

	for _, st := range types {
		v := NewParameterValue("value-"+string(st), st, 0)
		_, err := pool.Add(ctx, v)
		if err != nil {
			t.Fatalf("Add failed for %s: %v", st, err)
		}
	}

	// Verify each type has values
	for _, st := range types {
		count, _ := pool.Count(ctx, st)
		if count != 1 {
			t.Errorf("Count for %s = %d, want 1", st, count)
		}
	}

	// Test Types
	gotTypes, err := pool.Types(ctx)
	if err != nil {
		t.Fatalf("Types failed: %v", err)
	}
	if len(gotTypes) != len(types) {
		t.Errorf("Types count = %d, want %d", len(gotTypes), len(types))
	}
}

func TestShardedParameterPoolGetRandom(t *testing.T) {
	config := DefaultPoolConfig()
	config.CleanupInterval = 0
	pool := NewShardedParameterPool(config)
	defer pool.Close()

	ctx := context.Background()

	// Add multiple values
	for i := range 10 {
		v := NewParameterValue(i, SemanticTypeCustomerID, 0)
		pool.Add(ctx, v)
	}

	// GetRandom should return values
	for range 20 {
		got, err := pool.GetRandom(ctx, SemanticTypeCustomerID)
		if err != nil {
			t.Fatalf("GetRandom failed: %v", err)
		}
		if got == nil {
			t.Error("GetRandom returned nil")
		}
	}
}

func TestShardedParameterPoolGetAll(t *testing.T) {
	config := DefaultPoolConfig()
	config.CleanupInterval = 0
	pool := NewShardedParameterPool(config)
	defer pool.Close()

	ctx := context.Background()

	// Add values
	for i := range 5 {
		v := NewParameterValue(i, SemanticTypeCustomerID, 0)
		pool.Add(ctx, v)
	}

	// GetAll
	all, err := pool.GetAll(ctx, SemanticTypeCustomerID)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(all) != 5 {
		t.Errorf("GetAll returned %d values, want 5", len(all))
	}
}

func TestShardedParameterPoolRemove(t *testing.T) {
	config := DefaultPoolConfig()
	config.CleanupInterval = 0
	pool := NewShardedParameterPool(config)
	defer pool.Close()

	ctx := context.Background()

	v := NewParameterValue("to-remove", SemanticTypeCustomerID, 0)
	pool.Add(ctx, v)

	// Remove
	removed, err := pool.Remove(ctx, v)
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	if !removed {
		t.Error("Remove should return true")
	}

	// Verify removed
	count, _ := pool.Count(ctx, SemanticTypeCustomerID)
	if count != 0 {
		t.Errorf("Count = %d, want 0", count)
	}
}

func TestShardedParameterPoolClear(t *testing.T) {
	config := DefaultPoolConfig()
	config.CleanupInterval = 0
	pool := NewShardedParameterPool(config)
	defer pool.Close()

	ctx := context.Background()

	// Add values
	for i := range 10 {
		v := NewParameterValue(i, SemanticTypeCustomerID, 0)
		pool.Add(ctx, v)
	}

	// Clear
	cleared, err := pool.Clear(ctx, SemanticTypeCustomerID)
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}
	if cleared != 10 {
		t.Errorf("Cleared = %d, want 10", cleared)
	}

	// Verify cleared
	count, _ := pool.Count(ctx, SemanticTypeCustomerID)
	if count != 0 {
		t.Errorf("Count after clear = %d, want 0", count)
	}
}

func TestShardedParameterPoolClearAll(t *testing.T) {
	config := DefaultPoolConfig()
	config.CleanupInterval = 0
	pool := NewShardedParameterPool(config)
	defer pool.Close()

	ctx := context.Background()

	// Add values of different types
	pool.Add(ctx, NewParameterValue("c1", SemanticTypeCustomerID, 0))
	pool.Add(ctx, NewParameterValue("p1", SemanticTypeProductID, 0))

	// ClearAll
	err := pool.ClearAll(ctx)
	if err != nil {
		t.Fatalf("ClearAll failed: %v", err)
	}

	// Verify all cleared
	c1, _ := pool.Count(ctx, SemanticTypeCustomerID)
	c2, _ := pool.Count(ctx, SemanticTypeProductID)
	if c1+c2 != 0 {
		t.Errorf("Total count = %d, want 0", c1+c2)
	}
}

func TestShardedParameterPoolCleanup(t *testing.T) {
	config := DefaultPoolConfig()
	config.CleanupInterval = 0
	pool := NewShardedParameterPool(config)
	defer pool.Close()

	ctx := context.Background()

	// Add expired and non-expired values
	pool.Add(ctx, NewParameterValue("expired", SemanticTypeCustomerID, time.Millisecond))
	pool.Add(ctx, NewParameterValue("valid", SemanticTypeCustomerID, time.Hour))

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Cleanup
	cleaned, err := pool.Cleanup(ctx)
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}
	if cleaned != 1 {
		t.Errorf("Cleaned = %d, want 1", cleaned)
	}

	// Verify only valid value remains
	count, _ := pool.Count(ctx, SemanticTypeCustomerID)
	if count != 1 {
		t.Errorf("Count = %d, want 1", count)
	}
}

func TestShardedParameterPoolStats(t *testing.T) {
	config := DefaultPoolConfig()
	config.CleanupInterval = 0
	pool := NewShardedParameterPool(config)
	defer pool.Close()

	ctx := context.Background()

	// Add some values
	for i := range 5 {
		v := NewParameterValue(i, SemanticTypeCustomerID, 0)
		pool.Add(ctx, v)
	}

	// Get some values
	for range 3 {
		pool.Get(ctx, SemanticTypeCustomerID)
	}

	// Get miss
	pool.Get(ctx, SemanticTypeProductID)

	// Check stats
	stats, err := pool.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}

	if stats.TotalValues != 5 {
		t.Errorf("TotalValues = %d, want 5", stats.TotalValues)
	}

	if stats.AddCount != 5 {
		t.Errorf("AddCount = %d, want 5", stats.AddCount)
	}

	if stats.HitCount != 3 {
		t.Errorf("HitCount = %d, want 3", stats.HitCount)
	}

	if stats.MissCount != 1 {
		t.Errorf("MissCount = %d, want 1", stats.MissCount)
	}
}

func TestShardedParameterPoolEviction(t *testing.T) {
	config := DefaultPoolConfig()
	config.MaxValuesPerType = 3
	config.EvictionPolicy = EvictionFIFO
	config.CleanupInterval = 0
	pool := NewShardedParameterPool(config)
	defer pool.Close()

	ctx := context.Background()

	// Add 5 values (should evict 2)
	for i := range 5 {
		pool.Add(ctx, NewParameterValue(i, SemanticTypeCustomerID, 0))
	}

	// Check count
	count, _ := pool.Count(ctx, SemanticTypeCustomerID)
	if count != 3 {
		t.Errorf("Count = %d, want 3", count)
	}

	// Check eviction count
	if pool.EvictionCount() != 2 {
		t.Errorf("EvictionCount = %d, want 2", pool.EvictionCount())
	}
}

func TestShardedParameterPoolClose(t *testing.T) {
	config := DefaultPoolConfig()
	config.CleanupInterval = time.Millisecond * 10 // Enable cleanup
	pool := NewShardedParameterPool(config)

	ctx := context.Background()

	// Add a value
	pool.Add(ctx, NewParameterValue("test", SemanticTypeCustomerID, 0))

	// Close
	err := pool.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Operations should fail after close
	_, err = pool.Get(ctx, SemanticTypeCustomerID)
	if err != ErrPoolClosed {
		t.Errorf("Expected ErrPoolClosed, got %v", err)
	}

	// Double close should return error
	err = pool.Close()
	if err != ErrPoolClosed {
		t.Errorf("Double close should return ErrPoolClosed, got %v", err)
	}
}

func TestShardedParameterPoolConcurrency(t *testing.T) {
	config := DefaultPoolConfig()
	config.ShardCount = 16
	config.MaxValuesPerType = 100
	config.CleanupInterval = 0
	pool := NewShardedParameterPool(config)
	defer pool.Close()

	ctx := context.Background()

	var wg sync.WaitGroup
	numGoroutines := 100
	numOperations := 100

	// Concurrent adds
	for i := range numGoroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range numOperations {
				v := NewParameterValue(id*1000+j, SemanticTypeCustomerID, 0)
				pool.Add(ctx, v)
			}
		}(i)
	}

	// Concurrent reads
	for range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range numOperations {
				pool.Get(ctx, SemanticTypeCustomerID)
				pool.GetRandom(ctx, SemanticTypeCustomerID)
				pool.Count(ctx, SemanticTypeCustomerID)
			}
		}()
	}

	wg.Wait()

	// Pool should be in consistent state
	stats, _ := pool.Stats(ctx)
	if stats.TotalValues <= 0 {
		t.Error("Pool should have values after concurrent operations")
	}
}

func TestShardedParameterPoolShardCount(t *testing.T) {
	tests := []struct {
		configShards   int
		expectedShards int
	}{
		{0, 16},  // Default
		{1, 1},   // Minimum
		{8, 8},   // Power of 2
		{10, 16}, // Rounds up to power of 2
		{17, 32}, // Rounds up to power of 2
	}

	for _, tt := range tests {
		config := DefaultPoolConfig()
		config.ShardCount = tt.configShards
		config.CleanupInterval = 0
		pool := NewShardedParameterPool(config)

		if pool.ShardCount() != tt.expectedShards {
			t.Errorf("ShardCount(%d) = %d, want %d", tt.configShards, pool.ShardCount(), tt.expectedShards)
		}
		pool.Close()
	}
}

func TestShardedParameterPoolGetMiss(t *testing.T) {
	config := DefaultPoolConfig()
	config.CleanupInterval = 0
	pool := NewShardedParameterPool(config)
	defer pool.Close()

	ctx := context.Background()

	// Get from empty pool
	got, err := pool.Get(ctx, SemanticTypeCustomerID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got != nil {
		t.Error("Get from empty pool should return nil")
	}

	// Verify miss count
	stats, _ := pool.Stats(ctx)
	if stats.MissCount != 1 {
		t.Errorf("MissCount = %d, want 1", stats.MissCount)
	}
}

func TestShardedParameterPoolExpiredValueGet(t *testing.T) {
	config := DefaultPoolConfig()
	config.CleanupInterval = 0
	pool := NewShardedParameterPool(config)
	defer pool.Close()

	ctx := context.Background()

	// Add expired value
	v := NewParameterValue("expired", SemanticTypeCustomerID, time.Nanosecond)
	pool.Add(ctx, v)

	// Wait for expiration
	time.Sleep(time.Millisecond)

	// Get should return nil for expired value
	got, err := pool.Get(ctx, SemanticTypeCustomerID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got != nil {
		t.Error("Get should return nil for expired value")
	}
}

func TestEvictionPolicyString(t *testing.T) {
	tests := []struct {
		policy EvictionPolicy
		want   string
	}{
		{EvictionFIFO, "FIFO"},
		{EvictionLRU, "LRU"},
		{EvictionRandom, "Random"},
		{EvictionPolicy(99), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.policy.String(); got != tt.want {
			t.Errorf("EvictionPolicy(%d).String() = %s, want %s", tt.policy, got, tt.want)
		}
	}
}

func TestParseEvictionPolicy(t *testing.T) {
	tests := []struct {
		input string
		want  EvictionPolicy
	}{
		{"LRU", EvictionLRU},
		{"lru", EvictionLRU},
		{"Random", EvictionRandom},
		{"random", EvictionRandom},
		{"RANDOM", EvictionRandom},
		{"FIFO", EvictionFIFO},
		{"fifo", EvictionFIFO},
		{"unknown", EvictionFIFO}, // Default
		{"", EvictionFIFO},        // Default
	}

	for _, tt := range tests {
		if got := ParseEvictionPolicy(tt.input); got != tt.want {
			t.Errorf("ParseEvictionPolicy(%s) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestStatsHitRate(t *testing.T) {
	tests := []struct {
		hits   int64
		misses int64
		want   float64
	}{
		{0, 0, 0},
		{10, 0, 100},
		{0, 10, 0},
		{50, 50, 50},
		{3, 1, 75},
	}

	for _, tt := range tests {
		stats := Stats{
			HitCount:  tt.hits,
			MissCount: tt.misses,
		}
		if got := stats.HitRate(); got != tt.want {
			t.Errorf("HitRate(hits=%d, misses=%d) = %f, want %f", tt.hits, tt.misses, got, tt.want)
		}
	}
}
