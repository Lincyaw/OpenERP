package circuit

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProducerChainGuard_DefaultConfig(t *testing.T) {
	guard := NewProducerChainGuard(nil)
	require.NotNil(t, guard)

	cfg := guard.Config()
	assert.Equal(t, 3, cfg.MaxDepth)
	assert.Equal(t, time.Second, cfg.CooldownPeriod)
	assert.Equal(t, 5, cfg.MinPoolSize)
	assert.Equal(t, 10, cfg.RefillBatchSize)
}

func TestNewProducerChainGuard_CustomConfig(t *testing.T) {
	config := &ProducerChainGuardConfig{
		MaxDepth:        5,
		CooldownPeriod:  2 * time.Second,
		MinPoolSize:     10,
		RefillBatchSize: 20,
	}
	guard := NewProducerChainGuard(config)

	cfg := guard.Config()
	assert.Equal(t, 5, cfg.MaxDepth)
	assert.Equal(t, 2*time.Second, cfg.CooldownPeriod)
	assert.Equal(t, 10, cfg.MinPoolSize)
	assert.Equal(t, 20, cfg.RefillBatchSize)
}

func TestNewProducerChainGuard_PartialConfig(t *testing.T) {
	// Only override some values, others should use defaults
	config := &ProducerChainGuardConfig{
		MaxDepth: 7,
	}
	guard := NewProducerChainGuard(config)

	cfg := guard.Config()
	assert.Equal(t, 7, cfg.MaxDepth)
	assert.Equal(t, time.Second, cfg.CooldownPeriod) // default
	assert.Equal(t, 5, cfg.MinPoolSize)              // default
	assert.Equal(t, 10, cfg.RefillBatchSize)         // default
}

func TestProducerChainGuard_Enter_Success(t *testing.T) {
	guard := NewProducerChainGuard(nil)
	ctx := context.Background()

	// Enter first level
	err := guard.Enter(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, guard.CurrentDepth())

	// Enter second level
	err = guard.Enter(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, guard.CurrentDepth())

	// Enter third level (at max depth)
	err = guard.Enter(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, guard.CurrentDepth())
}

func TestProducerChainGuard_Enter_MaxDepthExceeded(t *testing.T) {
	guard := NewProducerChainGuard(&ProducerChainGuardConfig{
		MaxDepth: 3,
	})
	ctx := context.Background()

	// Fill to max depth
	for i := 0; i < 3; i++ {
		err := guard.Enter(ctx)
		require.NoError(t, err)
	}
	assert.Equal(t, 3, guard.CurrentDepth())

	// Attempt to exceed depth
	err := guard.Enter(ctx)
	assert.ErrorIs(t, err, ErrMaxDepthExceeded)
	assert.Equal(t, 3, guard.CurrentDepth()) // Should not have incremented

	// Verify stats
	stats := guard.Stats()
	assert.Equal(t, int64(4), stats.TotalEnterAttempts)
	assert.Equal(t, int64(3), stats.TotalEnterSuccess)
	assert.Equal(t, int64(1), stats.TotalDepthRejections)
}

func TestProducerChainGuard_Exit(t *testing.T) {
	guard := NewProducerChainGuard(nil)
	ctx := context.Background()

	// Enter multiple levels
	for i := 0; i < 3; i++ {
		_ = guard.Enter(ctx)
	}
	assert.Equal(t, 3, guard.CurrentDepth())

	// Exit one level
	guard.Exit()
	assert.Equal(t, 2, guard.CurrentDepth())

	// Exit all levels
	guard.Exit()
	guard.Exit()
	assert.Equal(t, 0, guard.CurrentDepth())

	// Extra exit should not go negative
	guard.Exit()
	assert.Equal(t, 0, guard.CurrentDepth())
}

func TestProducerChainGuard_ContextCancellation(t *testing.T) {
	guard := NewProducerChainGuard(nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := guard.Enter(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestProducerChainGuard_TryRefill_Success(t *testing.T) {
	guard := NewProducerChainGuard(&ProducerChainGuardConfig{
		RefillBatchSize: 15,
	})
	ctx := context.Background()
	semantic := SemanticType("entity.customer.id")

	// First refill should succeed
	batchSize, err := guard.TryRefill(ctx, semantic)
	require.NoError(t, err)
	assert.Equal(t, 15, batchSize)

	stats := guard.Stats()
	assert.Equal(t, int64(1), stats.TotalRefillsTriggered)
}

func TestProducerChainGuard_TryRefill_Cooldown(t *testing.T) {
	// Use a fixed time for testing
	currentTime := time.Now()
	guard := NewProducerChainGuard(&ProducerChainGuardConfig{
		CooldownPeriod: time.Second,
	})
	guard.WithNowFunc(func() time.Time {
		return currentTime
	})

	ctx := context.Background()
	semantic := SemanticType("entity.customer.id")

	// First refill succeeds
	_, err := guard.TryRefill(ctx, semantic)
	require.NoError(t, err)

	// Immediate second refill should fail (in cooldown)
	batchSize, err := guard.TryRefill(ctx, semantic)
	assert.ErrorIs(t, err, ErrCooldownActive)
	assert.Equal(t, 0, batchSize)

	stats := guard.Stats()
	assert.Equal(t, int64(1), stats.TotalRefillsTriggered)
	assert.Equal(t, int64(1), stats.TotalCooldownSkips)
}

func TestProducerChainGuard_TryRefill_AfterCooldown(t *testing.T) {
	currentTime := time.Now()
	guard := NewProducerChainGuard(&ProducerChainGuardConfig{
		CooldownPeriod: time.Second,
	})
	guard.WithNowFunc(func() time.Time {
		return currentTime
	})

	ctx := context.Background()
	semantic := SemanticType("entity.customer.id")

	// First refill
	_, err := guard.TryRefill(ctx, semantic)
	require.NoError(t, err)

	// Advance time past cooldown
	currentTime = currentTime.Add(2 * time.Second)

	// Second refill should now succeed
	batchSize, err := guard.TryRefill(ctx, semantic)
	require.NoError(t, err)
	assert.Equal(t, 10, batchSize)

	stats := guard.Stats()
	assert.Equal(t, int64(2), stats.TotalRefillsTriggered)
}

func TestProducerChainGuard_CanRefill(t *testing.T) {
	currentTime := time.Now()
	guard := NewProducerChainGuard(&ProducerChainGuardConfig{
		CooldownPeriod: time.Second,
	})
	guard.WithNowFunc(func() time.Time {
		return currentTime
	})

	ctx := context.Background()
	semantic := SemanticType("entity.product.id")

	// Before any refill, should be allowed
	assert.True(t, guard.CanRefill(semantic))

	// After refill, should not be allowed
	_, _ = guard.TryRefill(ctx, semantic)
	assert.False(t, guard.CanRefill(semantic))

	// After cooldown, should be allowed again
	currentTime = currentTime.Add(2 * time.Second)
	assert.True(t, guard.CanRefill(semantic))
}

func TestProducerChainGuard_ShouldRefill(t *testing.T) {
	guard := NewProducerChainGuard(&ProducerChainGuardConfig{
		MinPoolSize: 5,
	})

	assert.True(t, guard.ShouldRefill(0))
	assert.True(t, guard.ShouldRefill(4))
	assert.False(t, guard.ShouldRefill(5))
	assert.False(t, guard.ShouldRefill(100))
}

func TestProducerChainGuard_ExecuteWithGuard(t *testing.T) {
	guard := NewProducerChainGuard(&ProducerChainGuardConfig{
		MaxDepth: 2,
	})
	ctx := context.Background()

	executed := false
	err := guard.ExecuteWithGuard(ctx, func(ctx context.Context) error {
		executed = true
		assert.Equal(t, 1, guard.CurrentDepth())
		return nil
	})

	require.NoError(t, err)
	assert.True(t, executed)
	assert.Equal(t, 0, guard.CurrentDepth()) // Should have exited
}

func TestProducerChainGuard_ExecuteWithGuard_Nested(t *testing.T) {
	guard := NewProducerChainGuard(&ProducerChainGuardConfig{
		MaxDepth: 3,
	})
	ctx := context.Background()

	innerExecuted := false
	err := guard.ExecuteWithGuard(ctx, func(ctx context.Context) error {
		return guard.ExecuteWithGuard(ctx, func(ctx context.Context) error {
			innerExecuted = true
			assert.Equal(t, 2, guard.CurrentDepth())
			return nil
		})
	})

	require.NoError(t, err)
	assert.True(t, innerExecuted)
	assert.Equal(t, 0, guard.CurrentDepth())
}

func TestProducerChainGuard_ExecuteWithGuard_MaxDepthReached(t *testing.T) {
	guard := NewProducerChainGuard(&ProducerChainGuardConfig{
		MaxDepth: 2,
	})
	ctx := context.Background()

	var deepestLevel int
	err := guard.ExecuteWithGuard(ctx, func(ctx context.Context) error {
		return guard.ExecuteWithGuard(ctx, func(ctx context.Context) error {
			deepestLevel = 2
			// This should fail - we're at depth 2, trying to go to 3
			innerErr := guard.ExecuteWithGuard(ctx, func(ctx context.Context) error {
				deepestLevel = 3
				return nil
			})
			return innerErr
		})
	})

	assert.ErrorIs(t, err, ErrMaxDepthExceeded)
	assert.Equal(t, 2, deepestLevel) // Third level was not executed
	assert.Equal(t, 0, guard.CurrentDepth())
}

func TestProducerChainGuard_PeakDepth(t *testing.T) {
	guard := NewProducerChainGuard(&ProducerChainGuardConfig{
		MaxDepth: 5,
	})
	ctx := context.Background()

	// Enter 4 levels
	for i := 0; i < 4; i++ {
		_ = guard.Enter(ctx)
	}

	// Exit 2 levels
	guard.Exit()
	guard.Exit()

	// Enter 1 more level
	_ = guard.Enter(ctx)

	stats := guard.Stats()
	assert.Equal(t, int32(4), stats.PeakDepth) // Peak was 4, even though current is 3
}

func TestProducerChainGuard_ResetCooldown(t *testing.T) {
	currentTime := time.Now()
	guard := NewProducerChainGuard(&ProducerChainGuardConfig{
		CooldownPeriod: time.Hour, // Long cooldown
	})
	guard.WithNowFunc(func() time.Time {
		return currentTime
	})

	ctx := context.Background()
	semantic := SemanticType("entity.customer.id")

	// Trigger refill
	_, _ = guard.TryRefill(ctx, semantic)
	assert.False(t, guard.CanRefill(semantic))

	// Reset cooldown for this semantic
	guard.ResetCooldown(semantic)
	assert.True(t, guard.CanRefill(semantic))
}

func TestProducerChainGuard_ResetAllCooldowns(t *testing.T) {
	currentTime := time.Now()
	guard := NewProducerChainGuard(&ProducerChainGuardConfig{
		CooldownPeriod: time.Hour,
	})
	guard.WithNowFunc(func() time.Time {
		return currentTime
	})

	ctx := context.Background()
	semantic1 := SemanticType("entity.customer.id")
	semantic2 := SemanticType("entity.product.id")

	// Trigger refills
	_, _ = guard.TryRefill(ctx, semantic1)
	_, _ = guard.TryRefill(ctx, semantic2)

	assert.False(t, guard.CanRefill(semantic1))
	assert.False(t, guard.CanRefill(semantic2))

	// Reset all
	guard.ResetAllCooldowns()

	assert.True(t, guard.CanRefill(semantic1))
	assert.True(t, guard.CanRefill(semantic2))
}

func TestProducerChainGuard_ResetStats(t *testing.T) {
	guard := NewProducerChainGuard(nil)
	ctx := context.Background()

	// Generate some stats
	_ = guard.Enter(ctx)
	guard.Exit()

	stats := guard.Stats()
	assert.Greater(t, stats.TotalEnterAttempts, int64(0))

	// Reset
	guard.ResetStats()

	stats = guard.Stats()
	assert.Equal(t, int64(0), stats.TotalEnterAttempts)
	assert.Equal(t, int64(0), stats.TotalEnterSuccess)
}

func TestProducerChainGuard_Close(t *testing.T) {
	guard := NewProducerChainGuard(nil)
	ctx := context.Background()

	assert.False(t, guard.IsClosed())

	guard.Close()
	assert.True(t, guard.IsClosed())

	// Operations should fail after close
	err := guard.Enter(ctx)
	assert.ErrorIs(t, err, ErrGuardClosed)

	_, err = guard.TryRefill(ctx, SemanticType("test"))
	assert.ErrorIs(t, err, ErrGuardClosed)

	assert.False(t, guard.CanRefill(SemanticType("test")))
}

func TestProducerChainGuard_Concurrent_Enter_Exit(t *testing.T) {
	guard := NewProducerChainGuard(&ProducerChainGuardConfig{
		MaxDepth: 100, // High limit for concurrency test
	})
	ctx := context.Background()

	var wg sync.WaitGroup
	goroutines := 50
	iterations := 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if err := guard.Enter(ctx); err == nil {
					// Simulate some work
					time.Sleep(time.Microsecond)
					guard.Exit()
				}
			}
		}()
	}

	wg.Wait()

	// After all goroutines complete, depth should be 0
	assert.Equal(t, 0, guard.CurrentDepth())

	// Stats should be consistent
	stats := guard.Stats()
	assert.Equal(t, stats.TotalEnterSuccess, stats.TotalEnterAttempts-stats.TotalDepthRejections)
}

func TestProducerChainGuard_Concurrent_TryRefill(t *testing.T) {
	guard := NewProducerChainGuard(&ProducerChainGuardConfig{
		CooldownPeriod: 10 * time.Millisecond,
	})
	ctx := context.Background()

	var wg sync.WaitGroup
	goroutines := 20
	successCount := atomic.Int32{}

	// All goroutines try to refill the same semantic type
	semantic := SemanticType("entity.customer.id")

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := guard.TryRefill(ctx, semantic); err == nil {
				successCount.Add(1)
			}
		}()
	}

	wg.Wait()

	// Only one should succeed (all others hit cooldown)
	assert.Equal(t, int32(1), successCount.Load())
}

func TestProducerChainGuard_MultipleSemanticTypes(t *testing.T) {
	currentTime := time.Now()
	guard := NewProducerChainGuard(&ProducerChainGuardConfig{
		CooldownPeriod: time.Second,
	})
	guard.WithNowFunc(func() time.Time {
		return currentTime
	})

	ctx := context.Background()

	semantics := []SemanticType{
		"entity.customer.id",
		"entity.product.id",
		"entity.supplier.id",
		"order.sales.id",
	}

	// All should be allowed to refill (independent cooldowns)
	for _, s := range semantics {
		_, err := guard.TryRefill(ctx, s)
		require.NoError(t, err, "Failed for semantic: %s", s)
	}

	// All should be in cooldown
	for _, s := range semantics {
		_, err := guard.TryRefill(ctx, s)
		assert.ErrorIs(t, err, ErrCooldownActive)
	}

	stats := guard.Stats()
	assert.Equal(t, int64(4), stats.TotalRefillsTriggered)
	assert.Equal(t, int64(4), stats.TotalCooldownSkips)
}

// Acceptance test: Recursive calls exceeding 3 levels are rejected
func TestAcceptance_RecursiveCallsExceed3LevelsRejected(t *testing.T) {
	guard := NewProducerChainGuard(nil) // Default maxDepth = 3
	ctx := context.Background()

	var levels []int
	var recurse func(level int) error
	recurse = func(level int) error {
		return guard.ExecuteWithGuard(ctx, func(ctx context.Context) error {
			levels = append(levels, level)
			if level < 5 { // Try to go deeper than allowed
				return recurse(level + 1)
			}
			return nil
		})
	}

	err := recurse(1)

	// Should have stopped at level 3
	assert.ErrorIs(t, err, ErrMaxDepthExceeded)
	assert.Equal(t, []int{1, 2, 3}, levels) // Only 3 levels executed
}

// Acceptance test: Skip refill during cooldown period
func TestAcceptance_SkipRefillDuringCooldown(t *testing.T) {
	cooldown := 100 * time.Millisecond
	guard := NewProducerChainGuard(&ProducerChainGuardConfig{
		CooldownPeriod: cooldown,
	})
	ctx := context.Background()
	semantic := SemanticType("entity.customer.id")

	// First refill succeeds
	_, err := guard.TryRefill(ctx, semantic)
	require.NoError(t, err)

	// Immediate second refill should be skipped
	_, err = guard.TryRefill(ctx, semantic)
	assert.ErrorIs(t, err, ErrCooldownActive)

	// Wait for cooldown to expire
	time.Sleep(cooldown + 10*time.Millisecond)

	// Now refill should succeed
	_, err = guard.TryRefill(ctx, semantic)
	require.NoError(t, err)
}

func BenchmarkProducerChainGuard_EnterExit(b *testing.B) {
	guard := NewProducerChainGuard(&ProducerChainGuardConfig{
		MaxDepth: 100,
	})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = guard.Enter(ctx)
		guard.Exit()
	}
}

func BenchmarkProducerChainGuard_Concurrent(b *testing.B) {
	guard := NewProducerChainGuard(&ProducerChainGuardConfig{
		MaxDepth: 1000,
	})
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = guard.Enter(ctx)
			guard.Exit()
		}
	})
}

func BenchmarkProducerChainGuard_TryRefill(b *testing.B) {
	guard := NewProducerChainGuard(nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		semantic := SemanticType("test")
		guard.ResetCooldown(semantic)
		_, _ = guard.TryRefill(ctx, semantic)
	}
}
