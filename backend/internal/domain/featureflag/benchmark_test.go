package featureflag

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// =============================================================================
// Performance Benchmarks for FF-VAL-007 Validation
// Requirements:
// - Single flag evaluation latency < 5ms
// - Batch evaluation of 100 flags < 50ms
// - Cache hit rate > 90%
// - Memory usage with 10k flags reasonable
// =============================================================================

// BenchmarkPureEvaluator_SingleEvaluation benchmarks single flag evaluation
// Requirement: < 5ms per evaluation
func BenchmarkPureEvaluator_SingleEvaluation(b *testing.B) {
	flag, _ := NewFeatureFlag("benchmark-flag", "Benchmark Flag", FlagTypeBoolean, NewBooleanFlagValue(true), nil)
	flag.Enable(nil)
	evalCtx := NewEvaluationContext().WithUser(uuid.New().String()).WithTenant(uuid.New().String())
	evaluator := NewPureEvaluator()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = evaluator.Evaluate(flag, evalCtx, nil, nil)
	}
}

// BenchmarkPureEvaluator_WithRules benchmarks evaluation with targeting rules
func BenchmarkPureEvaluator_WithRules(b *testing.B) {
	flag, _ := NewFeatureFlag("benchmark-rule-flag", "Benchmark Rule Flag", FlagTypeBoolean, NewBooleanFlagValue(true), nil)
	flag.Enable(nil)

	// Add multiple targeting rules
	for i := 0; i < 5; i++ {
		condition, _ := NewCondition("user_role", ConditionOperatorEquals, []string{fmt.Sprintf("role-%d", i)})
		rule, _ := NewTargetingRule(fmt.Sprintf("rule-%d", i), i+1, []Condition{condition}, NewBooleanFlagValue(true))
		flag.AddRule(rule, nil)
	}

	evalCtx := NewEvaluationContext().WithUser(uuid.New().String()).WithUserRole("role-3")
	evaluator := NewPureEvaluator()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = evaluator.Evaluate(flag, evalCtx, nil, nil)
	}
}

// BenchmarkPureEvaluator_WithPercentageRollout benchmarks percentage rollout evaluation
func BenchmarkPureEvaluator_WithPercentageRollout(b *testing.B) {
	// Create a percentage flag with 50% rollout via metadata
	defaultValue := NewBooleanFlagValue(false).WithMetadata("percentage", 50)
	flag, _ := NewFeatureFlag("benchmark-percentage-flag", "Benchmark Percentage Flag", FlagTypePercentage, defaultValue, nil)
	flag.Enable(nil)

	userIDs := make([]string, 100)
	for i := range userIDs {
		userIDs[i] = uuid.New().String()
	}

	evaluator := NewPureEvaluator()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		evalCtx := NewEvaluationContext().WithUser(userIDs[i%100])
		_ = evaluator.Evaluate(flag, evalCtx, nil, nil)
	}
}

// BenchmarkPureEvaluator_WithVariants benchmarks variant selection
func BenchmarkPureEvaluator_WithVariants(b *testing.B) {
	variants := []string{"control", "variant-a", "variant-b", "variant-c"}
	defaultValue := NewVariantFlagValue("control").WithMetadata("variants", variants)
	flag, _ := NewFeatureFlag("benchmark-variant-flag", "Benchmark Variant Flag", FlagTypeVariant, defaultValue, nil)
	flag.Enable(nil)

	userIDs := make([]string, 100)
	for i := range userIDs {
		userIDs[i] = uuid.New().String()
	}

	evaluator := NewPureEvaluator()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		evalCtx := NewEvaluationContext().WithUser(userIDs[i%100])
		_ = evaluator.Evaluate(flag, evalCtx, nil, nil)
	}
}

// BenchmarkCachedEvaluator_CacheHit benchmarks evaluation with cache hit
func BenchmarkCachedEvaluator_CacheHit(b *testing.B) {
	flagRepo := newMockFlagRepo()
	overrideRepo := newMockOverrideRepo()
	cache := newMockCache()

	// Pre-populate cache
	flag, _ := NewFeatureFlag("cached-flag", "Cached Flag", FlagTypeBoolean, NewBooleanFlagValue(true), nil)
	flag.Enable(nil)
	flagRepo.flags["cached-flag"] = flag
	cache.Set(context.Background(), "cached-flag", flag, 5*time.Minute)

	evaluator := NewCachedEvaluator(flagRepo, overrideRepo, cache, WithCachedEvaluatorLogger(zap.NewNop()))
	ctx := context.Background()
	evalCtx := NewEvaluationContext().WithUser(uuid.New().String())

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = evaluator.Evaluate(ctx, "cached-flag", evalCtx)
	}
}

// BenchmarkCachedEvaluator_BatchEvaluation benchmarks batch evaluation
// Requirement: 100 flags < 50ms
func BenchmarkCachedEvaluator_BatchEvaluation_100Flags(b *testing.B) {
	flagRepo := newMockFlagRepo()
	overrideRepo := newMockOverrideRepo()
	cache := newMockCache()

	// Create 100 flags
	flagKeys := make([]string, 100)
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("batch-flag-%d", i)
		flag, _ := NewFeatureFlag(key, fmt.Sprintf("Batch Flag %d", i), FlagTypeBoolean, NewBooleanFlagValue(true), nil)
		flag.Enable(nil)
		flagRepo.flags[key] = flag
		cache.Set(context.Background(), key, flag, 5*time.Minute)
		flagKeys[i] = key
	}

	evaluator := NewCachedEvaluator(flagRepo, overrideRepo, cache, WithCachedEvaluatorLogger(zap.NewNop()))
	ctx := context.Background()
	evalCtx := NewEvaluationContext().WithUser(uuid.New().String())

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = evaluator.EvaluateBatch(ctx, flagKeys, evalCtx)
	}
}

// BenchmarkCachedEvaluator_BatchEvaluation_Parallel benchmarks parallel batch evaluation
func BenchmarkCachedEvaluator_BatchEvaluation_Parallel(b *testing.B) {
	flagRepo := newMockFlagRepo()
	overrideRepo := newMockOverrideRepo()
	cache := newMockCache()

	// Create 100 flags
	flagKeys := make([]string, 100)
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("parallel-flag-%d", i)
		flag, _ := NewFeatureFlag(key, fmt.Sprintf("Parallel Flag %d", i), FlagTypeBoolean, NewBooleanFlagValue(true), nil)
		flag.Enable(nil)
		flagRepo.flags[key] = flag
		cache.Set(context.Background(), key, flag, 5*time.Minute)
		flagKeys[i] = key
	}

	evaluator := NewCachedEvaluator(flagRepo, overrideRepo, cache, WithCachedEvaluatorLogger(zap.NewNop()))
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		evalCtx := NewEvaluationContext().WithUser(uuid.New().String())
		for pb.Next() {
			_ = evaluator.EvaluateBatch(ctx, flagKeys, evalCtx)
		}
	})
}

// BenchmarkIsInPercentage benchmarks the percentage hashing function
func BenchmarkIsInPercentage(b *testing.B) {
	flagKey := "benchmark-percentage-key"
	userID := uuid.New().String()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = IsInPercentage(flagKey, userID, 50)
	}
}

// BenchmarkSelectVariant benchmarks variant selection
func BenchmarkSelectVariant(b *testing.B) {
	flagKey := "benchmark-variant-key"
	userID := uuid.New().String()
	variants := []string{"control", "variant-a", "variant-b", "variant-c"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = SelectVariant(flagKey, userID, variants)
	}
}

// BenchmarkConditionMatch benchmarks condition matching
func BenchmarkConditionMatch(b *testing.B) {
	condition, _ := NewCondition("user_role", ConditionOperatorEquals, []string{"admin"})
	evalCtx := NewEvaluationContext().WithUserRole("admin")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = MatchCondition(condition, evalCtx)
	}
}

// BenchmarkConditionMatch_Contains benchmarks contains operator
func BenchmarkConditionMatch_Contains(b *testing.B) {
	condition, _ := NewCondition("user_plan", ConditionOperatorContains, []string{"pro"})
	evalCtx := NewEvaluationContext().WithUserPlan("professional")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = MatchCondition(condition, evalCtx)
	}
}

// =============================================================================
// Performance Assertion Tests (Non-benchmark tests that verify requirements)
// =============================================================================

// TestPerformance_SingleEvaluationLatency verifies single evaluation < 5ms
func TestPerformance_SingleEvaluationLatency(t *testing.T) {
	flag, _ := NewFeatureFlag("perf-flag", "Perf Flag", FlagTypeBoolean, NewBooleanFlagValue(true), nil)
	flag.Enable(nil)

	// Add some rules to simulate real-world usage
	for i := 0; i < 5; i++ {
		condition, _ := NewCondition("user_role", ConditionOperatorEquals, []string{fmt.Sprintf("role-%d", i)})
		rule, _ := NewTargetingRule(fmt.Sprintf("rule-%d", i), i+1, []Condition{condition}, NewBooleanFlagValue(true))
		flag.AddRule(rule, nil)
	}

	evalCtx := NewEvaluationContext().
		WithUser(uuid.New().String()).
		WithTenant(uuid.New().String()).
		WithUserRole("role-3")
	evaluator := NewPureEvaluator()

	// Warmup
	for i := 0; i < 100; i++ {
		_ = evaluator.Evaluate(flag, evalCtx, nil, nil)
	}

	// Measure
	iterations := 10000
	start := time.Now()
	for i := 0; i < iterations; i++ {
		_ = evaluator.Evaluate(flag, evalCtx, nil, nil)
	}
	elapsed := time.Since(start)
	avgLatency := elapsed / time.Duration(iterations)

	t.Logf("Average single evaluation latency: %v", avgLatency)
	t.Logf("Total time for %d evaluations: %v", iterations, elapsed)

	// Requirement: < 5ms per evaluation (we expect sub-microsecond for pure in-memory evaluation)
	assert.Less(t, avgLatency, 5*time.Millisecond, "Single evaluation should complete in < 5ms")
	// In practice, pure evaluation should be sub-microsecond
	assert.Less(t, avgLatency, 100*time.Microsecond, "Pure evaluation should be sub-100µs")
}

// TestPerformance_BatchEvaluation100Flags verifies batch evaluation < 50ms
func TestPerformance_BatchEvaluation100Flags(t *testing.T) {
	flagRepo := newMockFlagRepo()
	overrideRepo := newMockOverrideRepo()
	cache := newMockCache()

	// Create 100 flags
	flagKeys := make([]string, 100)
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("perf-batch-flag-%d", i)
		flag, _ := NewFeatureFlag(key, fmt.Sprintf("Perf Batch Flag %d", i), FlagTypeBoolean, NewBooleanFlagValue(true), nil)
		flag.Enable(nil)
		flagRepo.flags[key] = flag
		cache.Set(context.Background(), key, flag, 5*time.Minute)
		flagKeys[i] = key
	}

	evaluator := NewCachedEvaluator(flagRepo, overrideRepo, cache, WithCachedEvaluatorLogger(zap.NewNop()))
	ctx := context.Background()
	evalCtx := NewEvaluationContext().WithUser(uuid.New().String())

	// Warmup
	for i := 0; i < 10; i++ {
		_ = evaluator.EvaluateBatch(ctx, flagKeys, evalCtx)
	}

	// Measure
	iterations := 100
	start := time.Now()
	for i := 0; i < iterations; i++ {
		results := evaluator.EvaluateBatch(ctx, flagKeys, evalCtx)
		assert.Len(t, results, 100)
	}
	elapsed := time.Since(start)
	avgLatency := elapsed / time.Duration(iterations)

	t.Logf("Average batch evaluation (100 flags) latency: %v", avgLatency)
	t.Logf("Total time for %d batch evaluations: %v", iterations, elapsed)

	// Requirement: < 50ms for 100 flags
	assert.Less(t, avgLatency, 50*time.Millisecond, "Batch evaluation of 100 flags should complete in < 50ms")
}

// TestPerformance_CacheHitRate verifies cache hit rate > 90%
func TestPerformance_CacheHitRate(t *testing.T) {
	flagRepo := newMockFlagRepo()
	overrideRepo := newMockOverrideRepo()
	cache := newMockCache()

	// Create test flags
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("cache-test-flag-%d", i)
		flag, _ := NewFeatureFlag(key, fmt.Sprintf("Cache Test Flag %d", i), FlagTypeBoolean, NewBooleanFlagValue(true), nil)
		flag.Enable(nil)
		flagRepo.flags[key] = flag
	}

	evaluator := NewCachedEvaluator(flagRepo, overrideRepo, cache, WithCachedEvaluatorLogger(zap.NewNop()))
	ctx := context.Background()
	evalCtx := NewEvaluationContext().WithUser(uuid.New().String())

	// First round: all cache misses (populate cache)
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("cache-test-flag-%d", i)
		_ = evaluator.Evaluate(ctx, key, evalCtx)
	}

	// Second round: all cache hits
	totalRequests := 0
	cacheHits := 0

	for round := 0; round < 10; round++ {
		for i := 0; i < 10; i++ {
			key := fmt.Sprintf("cache-test-flag-%d", i)
			// Check if flag is in cache before evaluation
			cachedFlag, _ := cache.Get(ctx, key)
			if cachedFlag != nil {
				cacheHits++
			}
			_ = evaluator.Evaluate(ctx, key, evalCtx)
			totalRequests++
		}
	}

	hitRate := float64(cacheHits) / float64(totalRequests) * 100

	t.Logf("Cache hit rate: %.2f%% (%d hits / %d requests)", hitRate, cacheHits, totalRequests)

	// Requirement: > 90% cache hit rate
	assert.Greater(t, hitRate, 90.0, "Cache hit rate should be > 90%%")
}

// TestPerformance_MemoryUsage10kFlags verifies memory usage with 10k flags
func TestPerformance_MemoryUsage10kFlags(t *testing.T) {
	// Force GC before measurement
	runtime.GC()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	cache := newMockCache()
	ctx := context.Background()

	// Create 10k flags
	flags := make([]*FeatureFlag, 10000)
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("memory-test-flag-%d", i)
		flag, _ := NewFeatureFlag(key, fmt.Sprintf("Memory Test Flag %d", i), FlagTypeBoolean, NewBooleanFlagValue(true), nil)
		flag.Enable(nil)

		// Add some rules to simulate real-world usage
		if i%10 == 0 {
			condition, _ := NewCondition("user_role", ConditionOperatorEquals, []string{"admin"})
			rule, _ := NewTargetingRule("rule-1", 1, []Condition{condition}, NewBooleanFlagValue(true))
			flag.AddRule(rule, nil)
		}

		flags[i] = flag
		cache.Set(ctx, key, flag, 5*time.Minute)
	}

	// Force GC after creation
	runtime.GC()
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	memUsedMB := float64(memAfter.Alloc-memBefore.Alloc) / 1024 / 1024
	memTotalMB := float64(memAfter.Alloc) / 1024 / 1024
	flagCount := cache.flagCount

	t.Logf("Memory used for 10k flags: %.2f MB", memUsedMB)
	t.Logf("Total memory allocated: %.2f MB", memTotalMB)
	t.Logf("Flags in cache: %d", flagCount)
	t.Logf("Average memory per flag: %.2f KB", memUsedMB*1024/10000)

	// Requirement: Memory usage should be reasonable (< 500MB for 10k flags)
	assert.Less(t, memUsedMB, 500.0, "Memory usage for 10k flags should be < 500MB")

	// Verify all flags are cached
	assert.Equal(t, 10000, flagCount, "All 10k flags should be cached")
}

// TestPerformance_CacheInvalidation verifies cache invalidation mechanism
func TestPerformance_CacheInvalidation(t *testing.T) {
	ctx := context.Background()
	cache := newMockCache()

	// Populate cache
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("invalidation-flag-%d", i)
		flag, _ := NewFeatureFlag(key, fmt.Sprintf("Invalidation Flag %d", i), FlagTypeBoolean, NewBooleanFlagValue(true), nil)
		flag.Enable(nil)
		cache.Set(ctx, key, flag, 5*time.Minute)
	}

	flagCount := cache.flagCount
	require.Equal(t, 100, flagCount, "Should have 100 flags cached")

	// Measure single invalidation
	start := time.Now()
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("invalidation-flag-%d", i)
		err := cache.Delete(ctx, key)
		require.NoError(t, err)
	}
	elapsed := time.Since(start)

	t.Logf("Time to invalidate 100 flags: %v", elapsed)
	t.Logf("Average invalidation latency: %v", elapsed/100)

	flagCount = cache.flagCount
	assert.Equal(t, 0, flagCount, "All flags should be invalidated")

	// Requirement: Invalidation should be fast (< 1ms per flag)
	assert.Less(t, elapsed/100, time.Millisecond, "Single invalidation should be < 1ms")
}

// TestPerformance_InvalidateAll verifies bulk cache invalidation
func TestPerformance_InvalidateAll(t *testing.T) {
	ctx := context.Background()
	cache := newMockCache()

	// Populate cache with 1000 flags
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("bulk-invalidation-flag-%d", i)
		flag, _ := NewFeatureFlag(key, fmt.Sprintf("Bulk Invalidation Flag %d", i), FlagTypeBoolean, NewBooleanFlagValue(true), nil)
		flag.Enable(nil)
		cache.Set(ctx, key, flag, 5*time.Minute)
	}

	flagCount := cache.flagCount
	require.Equal(t, 1000, flagCount, "Should have 1000 flags cached")

	// Measure bulk invalidation
	start := time.Now()
	err := cache.InvalidateAll(ctx)
	require.NoError(t, err)
	elapsed := time.Since(start)

	t.Logf("Time to invalidate all 1000 flags: %v", elapsed)

	flagCount = cache.flagCount
	assert.Equal(t, 0, flagCount, "All flags should be invalidated")

	// Requirement: Bulk invalidation should be fast (< 100ms for 1000 flags)
	assert.Less(t, elapsed, 100*time.Millisecond, "Bulk invalidation should be < 100ms")
}

// TestPerformance_HashConsistency verifies hash function consistency
func TestPerformance_HashConsistency(t *testing.T) {
	flagKey := "consistency-test-flag"
	userID := uuid.New().String()

	// Verify same input produces same output
	results := make([]bool, 1000)
	for i := 0; i < 1000; i++ {
		results[i] = IsInPercentage(flagKey, userID, 50)
	}

	// All results should be the same
	firstResult := results[0]
	for i, result := range results {
		assert.Equal(t, firstResult, result, "Hash should be consistent (iteration %d)", i)
	}
}

// TestPerformance_PercentageDistribution verifies percentage distribution accuracy
func TestPerformance_PercentageDistribution(t *testing.T) {
	flagKey := "distribution-test-flag"
	targetPercentage := 50

	// Test with many users
	included := 0
	totalUsers := 10000

	for i := 0; i < totalUsers; i++ {
		userID := uuid.New().String()
		if IsInPercentage(flagKey, userID, targetPercentage) {
			included++
		}
	}

	actualPercentage := float64(included) / float64(totalUsers) * 100

	t.Logf("Target percentage: %d%%", targetPercentage)
	t.Logf("Actual percentage: %.2f%% (%d/%d)", actualPercentage, included, totalUsers)

	// Verify distribution is within acceptable variance (±5%)
	assert.InDelta(t, float64(targetPercentage), actualPercentage, 5.0,
		"Percentage distribution should be within ±5%% of target")
}
