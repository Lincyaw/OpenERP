package executor

import (
	"context"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/example/erp/tools/loadgen/internal/circuit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScheduler(t *testing.T) {
	t.Run("creates scheduler with default config", func(t *testing.T) {
		config := DefaultSchedulerConfig()
		s := NewScheduler(config)

		require.NotNil(t, s)
		assert.Equal(t, 0.8, s.config.ReadWriteRatio)
		assert.NotNil(t, s.endpoints)
		assert.NotNil(t, s.excludeSet)
	})

	t.Run("creates scheduler with custom config", func(t *testing.T) {
		config := SchedulerConfig{
			ReadWriteRatio:    0.5,
			ReadWriteRatioSet: true,
			Weights:           map[string]int{"ep1": 10},
			Exclude:           []string{"ep2"},
			CategoryWeights:   map[string]int{"catalog": 2},
			TagWeights:        map[string]int{"important": 3},
		}
		s := NewScheduler(config)

		require.NotNil(t, s)
		assert.Equal(t, 0.5, s.config.ReadWriteRatio)
		assert.Equal(t, 10, s.config.Weights["ep1"])
		assert.Contains(t, s.excludeSet, "ep2")
	})
}

func TestScheduler_Register(t *testing.T) {
	t.Run("registers endpoint successfully", func(t *testing.T) {
		s := NewScheduler(DefaultSchedulerConfig())

		err := s.Register(&EndpointInfo{
			Name:   "get-products",
			Method: "GET",
			Path:   "/products",
			Weight: 1,
		})

		require.NoError(t, err)
		assert.Equal(t, 1, len(s.endpoints))
	})

	t.Run("rejects nil endpoint", func(t *testing.T) {
		s := NewScheduler(DefaultSchedulerConfig())

		err := s.Register(nil)

		assert.ErrorIs(t, err, ErrNoEndpoints)
	})

	t.Run("rejects endpoint without name", func(t *testing.T) {
		s := NewScheduler(DefaultSchedulerConfig())

		err := s.Register(&EndpointInfo{
			Method: "GET",
			Path:   "/products",
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("rejects negative weight", func(t *testing.T) {
		s := NewScheduler(DefaultSchedulerConfig())

		err := s.Register(&EndpointInfo{
			Name:   "test",
			Method: "GET",
			Path:   "/test",
			Weight: -1,
		})

		assert.ErrorIs(t, err, ErrInvalidWeight)
	})
}

func TestScheduler_RegisterAll(t *testing.T) {
	t.Run("registers multiple endpoints", func(t *testing.T) {
		s := NewScheduler(DefaultSchedulerConfig())

		endpoints := []*EndpointInfo{
			{Name: "ep1", Method: "GET", Path: "/ep1", Weight: 1},
			{Name: "ep2", Method: "POST", Path: "/ep2", Weight: 2},
			{Name: "ep3", Method: "PUT", Path: "/ep3", Weight: 3},
		}

		err := s.RegisterAll(endpoints)

		require.NoError(t, err)
		assert.Equal(t, 3, len(s.endpoints))
	})

	t.Run("rejects empty list", func(t *testing.T) {
		s := NewScheduler(DefaultSchedulerConfig())

		err := s.RegisterAll([]*EndpointInfo{})

		assert.ErrorIs(t, err, ErrNoEndpoints)
	})
}

func TestScheduler_RegisterFromUnits(t *testing.T) {
	t.Run("registers endpoints from circuit units", func(t *testing.T) {
		s := NewScheduler(DefaultSchedulerConfig())

		units := []*circuit.EndpointUnit{
			{Name: "create-product", Method: "POST", Path: "/catalog/products", Weight: 1},
			{Name: "get-product", Method: "GET", Path: "/catalog/products/{id}", Weight: 2},
		}

		err := s.RegisterFromUnits(units)

		require.NoError(t, err)
		assert.Equal(t, 2, len(s.endpoints))

		// Check category inference
		ep, _ := s.GetEndpoint("create-product")
		assert.Equal(t, "catalog", ep.Category)
	})

	t.Run("skips disabled units", func(t *testing.T) {
		s := NewScheduler(DefaultSchedulerConfig())

		units := []*circuit.EndpointUnit{
			{Name: "active", Method: "GET", Path: "/active", Disabled: false},
			{Name: "disabled", Method: "GET", Path: "/disabled", Disabled: true},
		}

		err := s.RegisterFromUnits(units)

		require.NoError(t, err)
		assert.Equal(t, 1, len(s.endpoints))
	})
}

func TestScheduler_Select(t *testing.T) {
	t.Run("selects endpoint from pool", func(t *testing.T) {
		s := NewScheduler(DefaultSchedulerConfig())
		err := s.RegisterAll([]*EndpointInfo{
			{Name: "ep1", Method: "GET", Path: "/ep1", Weight: 1},
			{Name: "ep2", Method: "POST", Path: "/ep2", Weight: 1},
		})
		require.NoError(t, err)

		ep, err := s.Select()

		require.NoError(t, err)
		require.NotNil(t, ep)
		assert.Contains(t, []string{"ep1", "ep2"}, ep.Name)
	})

	t.Run("returns error when no endpoints", func(t *testing.T) {
		s := NewScheduler(DefaultSchedulerConfig())

		_, err := s.Select()

		assert.ErrorIs(t, err, ErrNoEndpoints)
	})

	t.Run("respects read/write ratio", func(t *testing.T) {
		// Use 100% read ratio
		config := SchedulerConfig{
			ReadWriteRatio:    1.0,
			ReadWriteRatioSet: true,
		}
		s := NewScheduler(config)
		err := s.RegisterAll([]*EndpointInfo{
			{Name: "read1", Method: "GET", Path: "/read1", Weight: 1},
			{Name: "write1", Method: "POST", Path: "/write1", Weight: 1},
		})
		require.NoError(t, err)

		// All selections should be reads
		for range 10 {
			ep, err := s.Select()
			require.NoError(t, err)
			assert.Equal(t, "GET", ep.Method)
		}
	})
}

func TestScheduler_SelectByCategory(t *testing.T) {
	t.Run("selects from specific category", func(t *testing.T) {
		s := NewScheduler(DefaultSchedulerConfig())
		err := s.RegisterAll([]*EndpointInfo{
			{Name: "catalog1", Method: "GET", Path: "/catalog/1", Category: "catalog", Weight: 1},
			{Name: "trade1", Method: "GET", Path: "/trade/1", Category: "trade", Weight: 1},
			{Name: "catalog2", Method: "GET", Path: "/catalog/2", Category: "catalog", Weight: 1},
		})
		require.NoError(t, err)

		ep, err := s.SelectByCategory("catalog")

		require.NoError(t, err)
		assert.Equal(t, "catalog", ep.Category)
	})

	t.Run("returns error for empty category", func(t *testing.T) {
		s := NewScheduler(DefaultSchedulerConfig())
		err := s.RegisterAll([]*EndpointInfo{
			{Name: "ep1", Method: "GET", Path: "/ep1", Category: "other", Weight: 1},
		})
		require.NoError(t, err)

		_, err = s.SelectByCategory("nonexistent")

		assert.ErrorIs(t, err, ErrNoEndpoints)
	})
}

func TestScheduler_SelectByTag(t *testing.T) {
	t.Run("selects from specific tag", func(t *testing.T) {
		s := NewScheduler(DefaultSchedulerConfig())
		err := s.RegisterAll([]*EndpointInfo{
			{Name: "ep1", Method: "GET", Path: "/ep1", Tags: []string{"important", "read"}, Weight: 1},
			{Name: "ep2", Method: "POST", Path: "/ep2", Tags: []string{"write"}, Weight: 1},
		})
		require.NoError(t, err)

		ep, err := s.SelectByTag("important")

		require.NoError(t, err)
		assert.Contains(t, ep.Tags, "important")
	})
}

func TestScheduler_Unregister(t *testing.T) {
	t.Run("removes endpoint", func(t *testing.T) {
		s := NewScheduler(DefaultSchedulerConfig())
		err := s.RegisterAll([]*EndpointInfo{
			{Name: "ep1", Method: "GET", Path: "/ep1", Weight: 1},
			{Name: "ep2", Method: "GET", Path: "/ep2", Weight: 1},
		})
		require.NoError(t, err)

		err = s.Unregister("ep1")

		require.NoError(t, err)
		assert.Equal(t, 1, len(s.endpoints))
	})

	t.Run("returns error for nonexistent endpoint", func(t *testing.T) {
		s := NewScheduler(DefaultSchedulerConfig())

		err := s.Unregister("nonexistent")

		assert.ErrorIs(t, err, ErrEndpointNotFound)
	})
}

func TestScheduler_WeightDistribution(t *testing.T) {
	t.Run("distributes selection according to weights", func(t *testing.T) {
		s := NewScheduler(DefaultSchedulerConfig())
		// Register with 1:9 weight ratio
		err := s.RegisterAll([]*EndpointInfo{
			{Name: "low", Method: "GET", Path: "/low", Weight: 1},
			{Name: "high", Method: "GET", Path: "/high", Weight: 9},
		})
		require.NoError(t, err)

		counts := make(map[string]int)
		iterations := 10000

		for range iterations {
			ep, err := s.SelectAny()
			require.NoError(t, err)
			counts[ep.Name]++
		}

		// With 1:9 ratio, "high" should be selected ~90% of the time
		// Allow some tolerance (85-95%)
		highRatio := float64(counts["high"]) / float64(iterations)
		assert.Greater(t, highRatio, 0.85, "high endpoint should be selected ~90% of time")
		assert.Less(t, highRatio, 0.95, "high endpoint should be selected ~90% of time")
	})
}

func TestScheduler_Exclude(t *testing.T) {
	t.Run("excludes endpoints from selection", func(t *testing.T) {
		config := SchedulerConfig{
			Exclude: []string{"excluded"},
		}
		s := NewScheduler(config)
		err := s.RegisterAll([]*EndpointInfo{
			{Name: "included", Method: "GET", Path: "/included", Weight: 1},
			{Name: "excluded", Method: "GET", Path: "/excluded", Weight: 1},
		})
		require.NoError(t, err)

		// Should only select "included"
		for range 100 {
			ep, err := s.Select()
			require.NoError(t, err)
			assert.Equal(t, "included", ep.Name)
		}
	})

	t.Run("can add exclusion at runtime", func(t *testing.T) {
		s := NewScheduler(DefaultSchedulerConfig())
		err := s.RegisterAll([]*EndpointInfo{
			{Name: "ep1", Method: "GET", Path: "/ep1", Weight: 1},
			{Name: "ep2", Method: "GET", Path: "/ep2", Weight: 1},
		})
		require.NoError(t, err)

		s.AddExclude("ep1")

		// Should only select "ep2"
		for range 100 {
			ep, err := s.Select()
			require.NoError(t, err)
			assert.Equal(t, "ep2", ep.Name)
		}
	})

	t.Run("can remove exclusion at runtime", func(t *testing.T) {
		config := SchedulerConfig{
			Exclude: []string{"ep1"},
		}
		s := NewScheduler(config)
		err := s.RegisterAll([]*EndpointInfo{
			{Name: "ep1", Method: "GET", Path: "/ep1", Weight: 100},
			{Name: "ep2", Method: "GET", Path: "/ep2", Weight: 1},
		})
		require.NoError(t, err)

		s.RemoveExclude("ep1")

		// Now ep1 should be selected (has much higher weight)
		counts := make(map[string]int)
		for range 100 {
			ep, err := s.Select()
			require.NoError(t, err)
			counts[ep.Name]++
		}

		// ep1 should dominate with 100:1 weight
		assert.Greater(t, counts["ep1"], counts["ep2"])
	})
}

func TestScheduler_WeightOverrides(t *testing.T) {
	t.Run("applies endpoint weight override", func(t *testing.T) {
		config := SchedulerConfig{
			Weights: map[string]int{"ep1": 100},
		}
		s := NewScheduler(config)
		err := s.RegisterAll([]*EndpointInfo{
			{Name: "ep1", Method: "GET", Path: "/ep1", Weight: 1}, // Base weight 1, override 100
			{Name: "ep2", Method: "GET", Path: "/ep2", Weight: 1},
		})
		require.NoError(t, err)

		counts := make(map[string]int)
		for range 1000 {
			ep, err := s.SelectAny()
			require.NoError(t, err)
			counts[ep.Name]++
		}

		// ep1 should dominate with 100:1 ratio
		assert.Greater(t, counts["ep1"], counts["ep2"]*50)
	})

	t.Run("applies category weight multiplier", func(t *testing.T) {
		config := SchedulerConfig{
			CategoryWeights: map[string]int{"important": 10},
		}
		s := NewScheduler(config)
		err := s.RegisterAll([]*EndpointInfo{
			{Name: "ep1", Method: "GET", Path: "/ep1", Category: "important", Weight: 1},
			{Name: "ep2", Method: "GET", Path: "/ep2", Category: "normal", Weight: 1},
		})
		require.NoError(t, err)

		counts := make(map[string]int)
		for range 1000 {
			ep, err := s.SelectAny()
			require.NoError(t, err)
			counts[ep.Name]++
		}

		// ep1 should be selected ~10x more often
		assert.Greater(t, counts["ep1"], counts["ep2"]*5)
	})

	t.Run("applies tag weight multiplier", func(t *testing.T) {
		config := SchedulerConfig{
			TagWeights: map[string]int{"priority": 10},
		}
		s := NewScheduler(config)
		err := s.RegisterAll([]*EndpointInfo{
			{Name: "ep1", Method: "GET", Path: "/ep1", Tags: []string{"priority"}, Weight: 1},
			{Name: "ep2", Method: "GET", Path: "/ep2", Tags: []string{"normal"}, Weight: 1},
		})
		require.NoError(t, err)

		counts := make(map[string]int)
		for range 1000 {
			ep, err := s.SelectAny()
			require.NoError(t, err)
			counts[ep.Name]++
		}

		// ep1 should be selected ~10x more often
		assert.Greater(t, counts["ep1"], counts["ep2"]*5)
	})

	t.Run("can set weight at runtime", func(t *testing.T) {
		s := NewScheduler(DefaultSchedulerConfig())
		err := s.RegisterAll([]*EndpointInfo{
			{Name: "ep1", Method: "GET", Path: "/ep1", Weight: 1},
			{Name: "ep2", Method: "GET", Path: "/ep2", Weight: 1},
		})
		require.NoError(t, err)

		err = s.SetWeight("ep1", 100)
		require.NoError(t, err)

		w, _ := s.GetEffectiveWeight("ep1")
		assert.Equal(t, 100, w)
	})
}

func TestScheduler_ReadWriteRatio(t *testing.T) {
	t.Run("100% read ratio selects only reads", func(t *testing.T) {
		config := SchedulerConfig{
			ReadWriteRatio:    1.0,
			ReadWriteRatioSet: true,
		}
		s := NewScheduler(config)
		err := s.RegisterAll([]*EndpointInfo{
			{Name: "read", Method: "GET", Path: "/read", Weight: 1},
			{Name: "write", Method: "POST", Path: "/write", Weight: 1},
		})
		require.NoError(t, err)

		for range 100 {
			ep, err := s.Select()
			require.NoError(t, err)
			assert.Equal(t, "GET", ep.Method)
		}
	})

	t.Run("0% read ratio selects only writes", func(t *testing.T) {
		config := SchedulerConfig{
			ReadWriteRatio:    0.0,
			ReadWriteRatioSet: true,
		}
		s := NewScheduler(config)
		err := s.RegisterAll([]*EndpointInfo{
			{Name: "read", Method: "GET", Path: "/read", Weight: 1},
			{Name: "write", Method: "POST", Path: "/write", Weight: 1},
		})
		require.NoError(t, err)

		for range 100 {
			ep, err := s.Select()
			require.NoError(t, err)
			assert.Equal(t, "POST", ep.Method)
		}
	})

	t.Run("falls back to available pool when one is empty", func(t *testing.T) {
		config := SchedulerConfig{
			ReadWriteRatio:    1.0, // 100% read
			ReadWriteRatioSet: true,
		}
		s := NewScheduler(config)
		// Only register write endpoints
		err := s.RegisterAll([]*EndpointInfo{
			{Name: "write", Method: "POST", Path: "/write", Weight: 1},
		})
		require.NoError(t, err)

		// Should still return the write endpoint
		ep, err := s.Select()
		require.NoError(t, err)
		assert.Equal(t, "POST", ep.Method)
	})

	t.Run("can change ratio at runtime", func(t *testing.T) {
		s := NewScheduler(DefaultSchedulerConfig())
		err := s.RegisterAll([]*EndpointInfo{
			{Name: "read", Method: "GET", Path: "/read", Weight: 1},
			{Name: "write", Method: "POST", Path: "/write", Weight: 1},
		})
		require.NoError(t, err)

		// Set to 100% write
		s.SetReadWriteRatio(0.0)

		for range 100 {
			ep, err := s.Select()
			require.NoError(t, err)
			assert.Equal(t, "POST", ep.Method)
		}
	})
}

func TestScheduler_Stats(t *testing.T) {
	t.Run("tracks selection statistics", func(t *testing.T) {
		s := NewScheduler(DefaultSchedulerConfig())
		err := s.RegisterAll([]*EndpointInfo{
			{Name: "ep1", Method: "GET", Path: "/ep1", Weight: 1},
			{Name: "ep2", Method: "POST", Path: "/ep2", Weight: 1},
		})
		require.NoError(t, err)

		for range 100 {
			_, _ = s.Select()
		}

		stats := s.Stats()

		assert.Equal(t, int64(100), stats.TotalSelections)
		assert.Equal(t, int64(100), stats.ReadSelections+stats.WriteSelections)
		assert.Equal(t, 2, stats.TotalEndpoints)
		assert.Equal(t, 2, stats.ActiveEndpoints)
	})

	t.Run("tracks per-endpoint selection counts", func(t *testing.T) {
		s := NewScheduler(DefaultSchedulerConfig())
		err := s.RegisterAll([]*EndpointInfo{
			{Name: "ep1", Method: "GET", Path: "/ep1", Weight: 1},
		})
		require.NoError(t, err)

		for range 50 {
			_, _ = s.SelectAny()
		}

		stats := s.Stats()

		assert.Equal(t, int64(50), stats.SelectionsByEndpoint["ep1"])
	})
}

func TestScheduler_Concurrency(t *testing.T) {
	t.Run("handles concurrent selections safely", func(t *testing.T) {
		s := NewScheduler(DefaultSchedulerConfig())
		err := s.RegisterAll([]*EndpointInfo{
			{Name: "ep1", Method: "GET", Path: "/ep1", Weight: 1},
			{Name: "ep2", Method: "POST", Path: "/ep2", Weight: 1},
		})
		require.NoError(t, err)

		var wg sync.WaitGroup
		numGoroutines := 10
		selectionsPerGoroutine := 1000

		for range numGoroutines {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range selectionsPerGoroutine {
					_, err := s.Select()
					assert.NoError(t, err)
				}
			}()
		}

		wg.Wait()

		stats := s.Stats()
		assert.Equal(t, int64(numGoroutines*selectionsPerGoroutine), stats.TotalSelections)
	})

	t.Run("handles concurrent registration and selection", func(t *testing.T) {
		s := NewScheduler(DefaultSchedulerConfig())

		// Pre-register some endpoints
		_ = s.RegisterAll([]*EndpointInfo{
			{Name: "ep1", Method: "GET", Path: "/ep1", Weight: 1},
		})

		var wg sync.WaitGroup
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Selection goroutines
		for range 5 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-ctx.Done():
						return
					default:
						_, _ = s.Select()
					}
				}
			}()
		}

		// Registration goroutines
		for i := range 3 {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := range 10 {
					select {
					case <-ctx.Done():
						return
					default:
						_ = s.Register(&EndpointInfo{
							Name:   "dynamic-" + string(rune('a'+id)) + "-" + string(rune('0'+j)),
							Method: "GET",
							Path:   "/dynamic",
							Weight: 1,
						})
					}
				}
			}(i)
		}

		wg.Wait()
		// Should complete without panics or data races
	})
}

func TestScheduler_DisabledEndpoints(t *testing.T) {
	t.Run("excludes disabled endpoints from selection", func(t *testing.T) {
		s := NewScheduler(DefaultSchedulerConfig())
		err := s.RegisterAll([]*EndpointInfo{
			{Name: "active", Method: "GET", Path: "/active", Weight: 1, Disabled: false},
			{Name: "disabled", Method: "GET", Path: "/disabled", Weight: 1, Disabled: true},
		})
		require.NoError(t, err)

		for range 100 {
			ep, err := s.Select()
			require.NoError(t, err)
			assert.Equal(t, "active", ep.Name)
		}
	})
}

func TestInferCategoryFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/catalog/products", "catalog"},
		{"/trade/orders/123", "trade"},
		{"/finance/invoices", "finance"},
		{"/api/v1/users", "api"},
		{"users", "users"},
		{"/single", "single"},
		{"", ""},
		{"/", ""},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			result := inferCategoryFromPath(tc.path)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// Benchmark tests
func BenchmarkScheduler_Select(b *testing.B) {
	s := NewScheduler(DefaultSchedulerConfig())
	endpoints := make([]*EndpointInfo, 100)
	for i := range 100 {
		method := "GET"
		if i%4 != 0 {
			method = "POST"
		}
		endpoints[i] = &EndpointInfo{
			Name:   "ep" + string(rune('0'+i/10)) + string(rune('0'+i%10)),
			Method: method,
			Path:   "/ep/" + string(rune('0'+i)),
			Weight: 1 + i%10,
		}
	}
	_ = s.RegisterAll(endpoints)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = s.Select()
		}
	})
}

func BenchmarkScheduler_SelectByCategory(b *testing.B) {
	s := NewScheduler(DefaultSchedulerConfig())
	categories := []string{"catalog", "trade", "finance", "inventory"}
	endpoints := make([]*EndpointInfo, 100)
	for i := range 100 {
		endpoints[i] = &EndpointInfo{
			Name:     "ep" + string(rune('0'+i/10)) + string(rune('0'+i%10)),
			Method:   "GET",
			Path:     "/ep/" + string(rune('0'+i)),
			Weight:   1,
			Category: categories[i%len(categories)],
		}
	}
	_ = s.RegisterAll(endpoints)

	b.ResetTimer()
	for b.Loop() {
		_, _ = s.SelectByCategory("catalog")
	}
}

func TestScheduler_SelectRead(t *testing.T) {
	s := NewScheduler(DefaultSchedulerConfig())
	err := s.RegisterAll([]*EndpointInfo{
		{Name: "get1", Method: "GET", Path: "/get1", Weight: 1},
		{Name: "post1", Method: "POST", Path: "/post1", Weight: 1},
		{Name: "head1", Method: "HEAD", Path: "/head1", Weight: 1},
	})
	require.NoError(t, err)

	for range 100 {
		ep, err := s.SelectRead()
		require.NoError(t, err)
		assert.True(t, ep.Method == "GET" || ep.Method == "HEAD")
	}
}

func TestScheduler_SelectWrite(t *testing.T) {
	s := NewScheduler(DefaultSchedulerConfig())
	err := s.RegisterAll([]*EndpointInfo{
		{Name: "get1", Method: "GET", Path: "/get1", Weight: 1},
		{Name: "post1", Method: "POST", Path: "/post1", Weight: 1},
		{Name: "put1", Method: "PUT", Path: "/put1", Weight: 1},
		{Name: "delete1", Method: "DELETE", Path: "/delete1", Weight: 1},
	})
	require.NoError(t, err)

	for range 100 {
		ep, err := s.SelectWrite()
		require.NoError(t, err)
		assert.Contains(t, []string{"POST", "PUT", "DELETE"}, ep.Method)
	}
}

func TestScheduler_EffectiveWeightCalculation(t *testing.T) {
	t.Run("default weight is 1", func(t *testing.T) {
		s := NewScheduler(DefaultSchedulerConfig())
		_ = s.Register(&EndpointInfo{
			Name:   "ep1",
			Method: "GET",
			Path:   "/ep1",
			Weight: 0, // Will default to 1
		})

		weight, err := s.GetEffectiveWeight("ep1")
		require.NoError(t, err)
		assert.Equal(t, 1, weight)
	})

	t.Run("combines base weight with multipliers", func(t *testing.T) {
		config := SchedulerConfig{
			CategoryWeights: map[string]int{"important": 2},
			TagWeights:      map[string]int{"priority": 3},
		}
		s := NewScheduler(config)
		_ = s.Register(&EndpointInfo{
			Name:     "ep1",
			Method:   "GET",
			Path:     "/ep1",
			Weight:   5,
			Category: "important",
			Tags:     []string{"priority"},
		})

		// 5 (base) * 2 (category) * 3 (tag) = 30
		weight, err := s.GetEffectiveWeight("ep1")
		require.NoError(t, err)
		assert.Equal(t, 30, weight)
	})

	t.Run("tag multiplier uses max of all tags", func(t *testing.T) {
		config := SchedulerConfig{
			TagWeights: map[string]int{
				"low":  2,
				"high": 10,
			},
		}
		s := NewScheduler(config)
		_ = s.Register(&EndpointInfo{
			Name:   "ep1",
			Method: "GET",
			Path:   "/ep1",
			Weight: 1,
			Tags:   []string{"low", "high"},
		})

		// Should use max tag weight (10)
		weight, err := s.GetEffectiveWeight("ep1")
		require.NoError(t, err)
		assert.Equal(t, 10, weight)
	})

	t.Run("handles overflow protection", func(t *testing.T) {
		config := SchedulerConfig{
			Weights:         map[string]int{"ep1": math.MaxInt32},
			CategoryWeights: map[string]int{"big": 10},
		}
		s := NewScheduler(config)
		_ = s.Register(&EndpointInfo{
			Name:     "ep1",
			Method:   "GET",
			Path:     "/ep1",
			Category: "big",
		})

		weight, err := s.GetEffectiveWeight("ep1")
		require.NoError(t, err)
		// Should be capped at maxWeight
		assert.Equal(t, math.MaxInt32, weight)
	})
}
