package selector

import (
	"math"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewWeightedSelector tests the creation of a weighted selector.
func TestNewWeightedSelector(t *testing.T) {
	tests := []struct {
		name    string
		config  WeightedSelectorConfig
		wantErr bool
	}{
		{
			name: "valid config with defaults",
			config: WeightedSelectorConfig{
				ReadWriteRatio: 0.8,
			},
			wantErr: false,
		},
		{
			name: "valid config with all fields",
			config: WeightedSelectorConfig{
				ReadWriteRatio: 0.7,
				Categories:     map[string]int{"catalog": 10, "trade": 5},
				CategoryRatios: map[string]float64{"catalog": 0.9},
				Weights:        map[string]int{"list-products": 20},
				Operations:     map[OperationType]int{OpGET: 10, OpPOST: 1},
			},
			wantErr: false,
		},
		{
			name: "invalid read/write ratio - negative",
			config: WeightedSelectorConfig{
				ReadWriteRatio: -0.1,
			},
			wantErr: true,
		},
		{
			name: "invalid read/write ratio - greater than 1",
			config: WeightedSelectorConfig{
				ReadWriteRatio: 1.5,
			},
			wantErr: true,
		},
		{
			name: "invalid category weight - negative",
			config: WeightedSelectorConfig{
				ReadWriteRatio: 0.8,
				Categories:     map[string]int{"catalog": -1},
			},
			wantErr: true,
		},
		{
			name: "invalid endpoint weight - negative",
			config: WeightedSelectorConfig{
				ReadWriteRatio: 0.8,
				Weights:        map[string]int{"endpoint": -1},
			},
			wantErr: true,
		},
		{
			name: "invalid operation weight - negative",
			config: WeightedSelectorConfig{
				ReadWriteRatio: 0.8,
				Operations:     map[OperationType]int{OpGET: -1},
			},
			wantErr: true,
		},
		{
			name: "invalid category ratio - negative",
			config: WeightedSelectorConfig{
				ReadWriteRatio: 0.8,
				CategoryRatios: map[string]float64{"catalog": -0.1},
			},
			wantErr: true,
		},
		{
			name: "invalid category ratio - greater than 1",
			config: WeightedSelectorConfig{
				ReadWriteRatio: 0.8,
				CategoryRatios: map[string]float64{"catalog": 1.5},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector, err := NewWeightedSelector(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, selector)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, selector)
			}
		})
	}
}

// TestEndpointRegistration tests registering and unregistering endpoints.
func TestEndpointRegistration(t *testing.T) {
	selector, err := NewWeightedSelector(WeightedSelectorConfig{ReadWriteRatio: 0.8})
	require.NoError(t, err)

	t.Run("register single endpoint", func(t *testing.T) {
		ep := &Endpoint{
			Name:       "get-products",
			Path:       "/products",
			Method:     "GET",
			Category:   "catalog",
			BaseWeight: 10,
		}
		err := selector.Register(ep)
		assert.NoError(t, err)

		retrieved, err := selector.GetEndpoint("get-products")
		assert.NoError(t, err)
		assert.Equal(t, ep.Name, retrieved.Name)
	})

	t.Run("register nil endpoint", func(t *testing.T) {
		err := selector.Register(nil)
		assert.Error(t, err)
	})

	t.Run("register endpoint without name", func(t *testing.T) {
		err := selector.Register(&Endpoint{Path: "/test", Method: "GET"})
		assert.Error(t, err)
	})

	t.Run("register endpoint with negative weight", func(t *testing.T) {
		err := selector.Register(&Endpoint{Name: "test", Path: "/test", Method: "GET", BaseWeight: -1})
		assert.Error(t, err)
	})

	t.Run("register multiple endpoints", func(t *testing.T) {
		endpoints := []*Endpoint{
			{Name: "create-product", Path: "/products", Method: "POST", Category: "catalog", BaseWeight: 5},
			{Name: "update-product", Path: "/products/{id}", Method: "PUT", Category: "catalog", BaseWeight: 3},
			{Name: "delete-product", Path: "/products/{id}", Method: "DELETE", Category: "catalog", BaseWeight: 1},
		}
		err := selector.RegisterAll(endpoints)
		assert.NoError(t, err)

		stats := selector.GetStats()
		assert.Equal(t, 4, stats.TotalEndpoints) // 1 from before + 3 new
	})

	t.Run("register empty list", func(t *testing.T) {
		err := selector.RegisterAll([]*Endpoint{})
		assert.Error(t, err)
	})

	t.Run("register duplicate endpoint overwrites", func(t *testing.T) {
		dupSelector, err := NewWeightedSelector(WeightedSelectorConfig{ReadWriteRatio: 0.8, ReadWriteRatioSet: true})
		require.NoError(t, err)

		ep1 := &Endpoint{Name: "test", Path: "/old", Method: "GET", BaseWeight: 1}
		ep2 := &Endpoint{Name: "test", Path: "/new", Method: "POST", BaseWeight: 5}

		require.NoError(t, dupSelector.Register(ep1))
		require.NoError(t, dupSelector.Register(ep2))

		retrieved, err := dupSelector.GetEndpoint("test")
		require.NoError(t, err)
		assert.Equal(t, "/new", retrieved.Path)
		assert.Equal(t, "POST", retrieved.Method)
		assert.Equal(t, 5, retrieved.BaseWeight)

		// Verify stats reflect single endpoint
		stats := dupSelector.GetStats()
		assert.Equal(t, 1, stats.TotalEndpoints)
	})

	t.Run("unregister endpoint", func(t *testing.T) {
		err := selector.Unregister("get-products")
		assert.NoError(t, err)

		_, err = selector.GetEndpoint("get-products")
		assert.Error(t, err)
	})

	t.Run("unregister non-existent endpoint", func(t *testing.T) {
		err := selector.Unregister("non-existent")
		assert.Error(t, err)
	})
}

// TestOperationType tests operation type classification.
func TestOperationType(t *testing.T) {
	tests := []struct {
		op      OperationType
		isRead  bool
		isWrite bool
	}{
		{OpGET, true, false},
		{OpPOST, false, true},
		{OpPUT, false, true},
		{OpDELETE, false, true},
		{OpPATCH, false, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.op), func(t *testing.T) {
			assert.Equal(t, tt.isRead, tt.op.IsRead())
			assert.Equal(t, tt.isWrite, tt.op.IsWrite())
		})
	}
}

// TestEndpointOperation tests endpoint operation type detection.
func TestEndpointOperation(t *testing.T) {
	tests := []struct {
		method string
		want   OperationType
	}{
		{"GET", OpGET},
		{"POST", OpPOST},
		{"PUT", OpPUT},
		{"DELETE", OpDELETE},
		{"PATCH", OpPATCH},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			ep := &Endpoint{Method: tt.method}
			assert.Equal(t, tt.want, ep.Operation())
		})
	}
}

// TestSelectNoEndpoints tests selection when no endpoints are registered.
func TestSelectNoEndpoints(t *testing.T) {
	selector, err := NewWeightedSelector(WeightedSelectorConfig{ReadWriteRatio: 0.8})
	require.NoError(t, err)

	_, err = selector.Select()
	assert.ErrorIs(t, err, ErrNoEndpoints)

	_, err = selector.SelectRead()
	assert.ErrorIs(t, err, ErrNoEndpoints)

	_, err = selector.SelectWrite()
	assert.ErrorIs(t, err, ErrNoEndpoints)
}

// TestReadWriteRatioDistribution verifies the read/write ratio distribution.
// Acceptance: 1000 selections should have < 5% error from configured ratio.
func TestReadWriteRatioDistribution(t *testing.T) {
	tests := []struct {
		name        string
		config      WeightedSelectorConfig
		iterations  int
		maxErrorPct float64
	}{
		{"80% read ratio", WeightedSelectorConfig{ReadWriteRatio: 0.8, ReadWriteRatioSet: true}, 1000, 5.0},
		{"50% read ratio", WeightedSelectorConfig{ReadWriteRatio: 0.5, ReadWriteRatioSet: true}, 1000, 5.0},
		{"20% read ratio", WeightedSelectorConfig{ReadWriteRatio: 0.2, ReadWriteRatioSet: true}, 1000, 5.0},
		{"90% read ratio", WeightedSelectorConfig{ReadWriteRatio: 0.9, ReadWriteRatioSet: true}, 1000, 5.0},
		{"100% read ratio", WeightedSelectorConfig{ReadWriteRatio: 1.0, ReadWriteRatioSet: true}, 1000, 0.0}, // Should be exactly 100% read
		{"0% read ratio", NewConfigWithRatio(0.0), 1000, 0.0},                                                // Should be exactly 0% read (100% write)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector, err := NewWeightedSelector(tt.config)
			require.NoError(t, err)

			// Register both read and write endpoints
			endpoints := []*Endpoint{
				{Name: "read-1", Path: "/read1", Method: "GET", BaseWeight: 1},
				{Name: "read-2", Path: "/read2", Method: "GET", BaseWeight: 1},
				{Name: "write-1", Path: "/write1", Method: "POST", BaseWeight: 1},
				{Name: "write-2", Path: "/write2", Method: "PUT", BaseWeight: 1},
			}
			require.NoError(t, selector.RegisterAll(endpoints))

			// Perform selections
			readCount := 0
			for i := 0; i < tt.iterations; i++ {
				ep, err := selector.Select()
				require.NoError(t, err)
				if ep.Operation().IsRead() {
					readCount++
				}
			}

			// Calculate actual ratio and error
			actualRatio := float64(readCount) / float64(tt.iterations)
			expectedRatio := tt.config.ReadWriteRatio
			errorPct := math.Abs(actualRatio-expectedRatio) * 100

			t.Logf("Expected ratio: %.2f, Actual ratio: %.4f (reads: %d/%d), Error: %.2f%%",
				expectedRatio, actualRatio, readCount, tt.iterations, errorPct)

			assert.LessOrEqual(t, errorPct, tt.maxErrorPct,
				"Read/write ratio error %.2f%% exceeds maximum allowed %.2f%%",
				errorPct, tt.maxErrorPct)
		})
	}
}

// TestEndpointWeightDistribution verifies that endpoint weights affect selection.
// Acceptance: 1000 selections should have < 5% error from expected distribution.
func TestEndpointWeightDistribution(t *testing.T) {
	selector, err := NewWeightedSelector(WeightedSelectorConfig{
		ReadWriteRatio: 1.0, // All reads to simplify test
	})
	require.NoError(t, err)

	// Register endpoints with different weights
	endpoints := []*Endpoint{
		{Name: "heavy", Path: "/heavy", Method: "GET", BaseWeight: 50},   // 50% expected
		{Name: "medium", Path: "/medium", Method: "GET", BaseWeight: 30}, // 30% expected
		{Name: "light", Path: "/light", Method: "GET", BaseWeight: 20},   // 20% expected
	}
	require.NoError(t, selector.RegisterAll(endpoints))

	// Perform selections
	iterations := 1000
	counts := make(map[string]int)
	for i := 0; i < iterations; i++ {
		ep, err := selector.Select()
		require.NoError(t, err)
		counts[ep.Name]++
	}

	// Verify distribution
	totalWeight := 100
	expectedPcts := map[string]float64{
		"heavy":  50.0,
		"medium": 30.0,
		"light":  20.0,
	}

	maxErrorPct := 5.0
	for name, expected := range expectedPcts {
		actual := (float64(counts[name]) / float64(iterations)) * 100
		errorPct := math.Abs(actual - expected)

		t.Logf("Endpoint %s: expected %.1f%%, actual %.2f%% (%d/%d), error: %.2f%%",
			name, expected, actual, counts[name], iterations, errorPct)

		assert.LessOrEqual(t, errorPct, maxErrorPct,
			"Endpoint %s distribution error %.2f%% exceeds maximum %.2f%% (totalWeight=%d)",
			name, errorPct, maxErrorPct, totalWeight)
	}
}

// TestCategoryWeightDistribution verifies category weight multipliers.
func TestCategoryWeightDistribution(t *testing.T) {
	selector, err := NewWeightedSelector(WeightedSelectorConfig{
		ReadWriteRatio: 1.0, // All reads
		Categories: map[string]int{
			"catalog": 3, // 3x weight
			"trade":   1, // 1x weight
		},
	})
	require.NoError(t, err)

	// Register endpoints with same base weight but different categories
	endpoints := []*Endpoint{
		{Name: "catalog-1", Path: "/catalog1", Method: "GET", Category: "catalog", BaseWeight: 1},
		{Name: "catalog-2", Path: "/catalog2", Method: "GET", Category: "catalog", BaseWeight: 1},
		{Name: "trade-1", Path: "/trade1", Method: "GET", Category: "trade", BaseWeight: 1},
		{Name: "trade-2", Path: "/trade2", Method: "GET", Category: "trade", BaseWeight: 1},
	}
	require.NoError(t, selector.RegisterAll(endpoints))

	// Perform selections
	iterations := 1000
	categoryCounts := make(map[string]int)
	for i := 0; i < iterations; i++ {
		ep, err := selector.Select()
		require.NoError(t, err)
		categoryCounts[ep.Category]++
	}

	// Expected: catalog should be ~75% (3/(3+3+1+1) = 6/8), trade ~25% (2/8)
	// Actually: catalog endpoints have weight 3 each (2 endpoints = 6 total)
	// trade endpoints have weight 1 each (2 endpoints = 2 total)
	// Total = 8, catalog = 6/8 = 75%, trade = 2/8 = 25%
	expectedCatalogPct := 75.0
	actualCatalogPct := (float64(categoryCounts["catalog"]) / float64(iterations)) * 100

	errorPct := math.Abs(actualCatalogPct - expectedCatalogPct)

	t.Logf("Category distribution - Catalog: %.2f%% (expected 75%%), Trade: %.2f%% (expected 25%%)",
		actualCatalogPct, (float64(categoryCounts["trade"])/float64(iterations))*100)

	assert.LessOrEqual(t, errorPct, 5.0,
		"Category distribution error %.2f%% exceeds 5%%", errorPct)
}

// TestEndpointSpecificWeightOverride verifies endpoint-specific weight overrides.
func TestEndpointSpecificWeightOverride(t *testing.T) {
	selector, err := NewWeightedSelector(WeightedSelectorConfig{
		ReadWriteRatio: 1.0,
		Weights: map[string]int{
			"override-me": 9, // Override to 9 (should be 90% of selections)
		},
	})
	require.NoError(t, err)

	endpoints := []*Endpoint{
		{Name: "override-me", Path: "/override", Method: "GET", BaseWeight: 1}, // Overridden to 9
		{Name: "normal", Path: "/normal", Method: "GET", BaseWeight: 1},        // Stays at 1
	}
	require.NoError(t, selector.RegisterAll(endpoints))

	// Verify effective weights
	w1, _ := selector.GetEffectiveWeight("override-me")
	w2, _ := selector.GetEffectiveWeight("normal")
	assert.Equal(t, 9, w1)
	assert.Equal(t, 1, w2)

	// Perform selections
	iterations := 1000
	counts := make(map[string]int)
	for i := 0; i < iterations; i++ {
		ep, err := selector.Select()
		require.NoError(t, err)
		counts[ep.Name]++
	}

	// Expected: override-me ~90%, normal ~10%
	overridePct := (float64(counts["override-me"]) / float64(iterations)) * 100
	errorPct := math.Abs(overridePct - 90.0)

	t.Logf("Override endpoint: %.2f%% (expected 90%%)", overridePct)

	assert.LessOrEqual(t, errorPct, 5.0,
		"Endpoint override distribution error %.2f%% exceeds 5%%", errorPct)
}

// TestOperationTypeWeightDistribution verifies operation type weight multipliers.
func TestOperationTypeWeightDistribution(t *testing.T) {
	selector, err := NewWeightedSelector(WeightedSelectorConfig{
		ReadWriteRatio: 0.0, // All writes
		Operations: map[OperationType]int{
			OpPOST:   3, // 3x weight
			OpPUT:    2, // 2x weight
			OpDELETE: 1, // 1x weight
		},
	})
	require.NoError(t, err)

	endpoints := []*Endpoint{
		{Name: "create", Path: "/create", Method: "POST", BaseWeight: 1},   // effective: 3
		{Name: "update", Path: "/update", Method: "PUT", BaseWeight: 1},    // effective: 2
		{Name: "delete", Path: "/delete", Method: "DELETE", BaseWeight: 1}, // effective: 1
	}
	require.NoError(t, selector.RegisterAll(endpoints))

	// Perform selections
	iterations := 1000
	counts := make(map[OperationType]int)
	for i := 0; i < iterations; i++ {
		ep, err := selector.Select()
		require.NoError(t, err)
		counts[ep.Operation()]++
	}

	// Expected: POST ~50% (3/6), PUT ~33% (2/6), DELETE ~17% (1/6)
	totalWeight := 6.0
	expectedPcts := map[OperationType]float64{
		OpPOST:   (3.0 / totalWeight) * 100,
		OpPUT:    (2.0 / totalWeight) * 100,
		OpDELETE: (1.0 / totalWeight) * 100,
	}

	for op, expected := range expectedPcts {
		actual := (float64(counts[op]) / float64(iterations)) * 100
		errorPct := math.Abs(actual - expected)

		t.Logf("Operation %s: expected %.2f%%, actual %.2f%%, error: %.2f%%",
			op, expected, actual, errorPct)

		assert.LessOrEqual(t, errorPct, 5.0,
			"Operation %s distribution error %.2f%% exceeds 5%%", op, errorPct)
	}
}

// TestCategoryRatioOverride verifies that category ratios override global ratio.
func TestCategoryRatioOverride(t *testing.T) {
	selector, err := NewWeightedSelector(WeightedSelectorConfig{
		ReadWriteRatio: 0.5, // Global: 50% read
		CategoryRatios: map[string]float64{
			"heavy-read": 0.9, // 90% read for this category
		},
	})
	require.NoError(t, err)

	endpoints := []*Endpoint{
		{Name: "hr-read", Path: "/hr-read", Method: "GET", Category: "heavy-read", BaseWeight: 1},
		{Name: "hr-write", Path: "/hr-write", Method: "POST", Category: "heavy-read", BaseWeight: 1},
	}
	require.NoError(t, selector.RegisterAll(endpoints))

	// Perform selections by category
	iterations := 1000
	readCount := 0
	for i := 0; i < iterations; i++ {
		ep, err := selector.SelectByCategory("heavy-read")
		require.NoError(t, err)
		if ep.Operation().IsRead() {
			readCount++
		}
	}

	actualRatio := float64(readCount) / float64(iterations)
	expectedRatio := 0.9
	errorPct := math.Abs(actualRatio-expectedRatio) * 100

	t.Logf("Category 'heavy-read' ratio: expected 90%% read, actual %.2f%% read",
		actualRatio*100)

	assert.LessOrEqual(t, errorPct, 5.0,
		"Category ratio override error %.2f%% exceeds 5%%", errorPct)
}

// TestSelectByOperation verifies selection by specific operation type.
func TestSelectByOperation(t *testing.T) {
	selector, err := NewWeightedSelector(WeightedSelectorConfig{ReadWriteRatio: 0.5})
	require.NoError(t, err)

	endpoints := []*Endpoint{
		{Name: "get-1", Path: "/get1", Method: "GET", BaseWeight: 2},
		{Name: "get-2", Path: "/get2", Method: "GET", BaseWeight: 1},
		{Name: "post-1", Path: "/post1", Method: "POST", BaseWeight: 1},
	}
	require.NoError(t, selector.RegisterAll(endpoints))

	// Select only GET operations
	iterations := 100
	for i := 0; i < iterations; i++ {
		ep, err := selector.SelectByOperation(OpGET)
		require.NoError(t, err)
		assert.Equal(t, "GET", ep.Method)
	}

	// Select only POST operations
	for i := 0; i < iterations; i++ {
		ep, err := selector.SelectByOperation(OpPOST)
		require.NoError(t, err)
		assert.Equal(t, "POST", ep.Method)
	}

	// Select non-existent operation
	_, err = selector.SelectByOperation(OpDELETE)
	assert.ErrorIs(t, err, ErrNoEndpoints)
}

// TestSelectReadWrite verifies SelectRead and SelectWrite methods.
func TestSelectReadWrite(t *testing.T) {
	selector, err := NewWeightedSelector(WeightedSelectorConfig{ReadWriteRatio: 0.5})
	require.NoError(t, err)

	endpoints := []*Endpoint{
		{Name: "get", Path: "/get", Method: "GET", BaseWeight: 1},
		{Name: "post", Path: "/post", Method: "POST", BaseWeight: 1},
	}
	require.NoError(t, selector.RegisterAll(endpoints))

	// SelectRead should always return read endpoint
	for i := 0; i < 10; i++ {
		ep, err := selector.SelectRead()
		require.NoError(t, err)
		assert.True(t, ep.Operation().IsRead())
	}

	// SelectWrite should always return write endpoint
	for i := 0; i < 10; i++ {
		ep, err := selector.SelectWrite()
		require.NoError(t, err)
		assert.True(t, ep.Operation().IsWrite())
	}
}

// TestOnlyReadEndpoints verifies behavior when only read endpoints exist.
func TestOnlyReadEndpoints(t *testing.T) {
	selector, err := NewWeightedSelector(WeightedSelectorConfig{ReadWriteRatio: 0.5}) // 50% write expected
	require.NoError(t, err)

	endpoints := []*Endpoint{
		{Name: "read-1", Path: "/read1", Method: "GET", BaseWeight: 1},
		{Name: "read-2", Path: "/read2", Method: "GET", BaseWeight: 1},
	}
	require.NoError(t, selector.RegisterAll(endpoints))

	// Should always return read even with 50% write ratio
	for i := 0; i < 100; i++ {
		ep, err := selector.Select()
		require.NoError(t, err)
		assert.True(t, ep.Operation().IsRead())
	}
}

// TestOnlyWriteEndpoints verifies behavior when only write endpoints exist.
func TestOnlyWriteEndpoints(t *testing.T) {
	selector, err := NewWeightedSelector(WeightedSelectorConfig{ReadWriteRatio: 0.8}) // 80% read expected
	require.NoError(t, err)

	endpoints := []*Endpoint{
		{Name: "write-1", Path: "/write1", Method: "POST", BaseWeight: 1},
		{Name: "write-2", Path: "/write2", Method: "PUT", BaseWeight: 1},
	}
	require.NoError(t, selector.RegisterAll(endpoints))

	// Should always return write even with 80% read ratio
	for i := 0; i < 100; i++ {
		ep, err := selector.Select()
		require.NoError(t, err)
		assert.True(t, ep.Operation().IsWrite())
	}
}

// TestGetStats verifies statistics gathering.
func TestGetStats(t *testing.T) {
	selector, err := NewWeightedSelector(WeightedSelectorConfig{ReadWriteRatio: 0.8})
	require.NoError(t, err)

	endpoints := []*Endpoint{
		{Name: "get-1", Path: "/get1", Method: "GET", Category: "catalog", BaseWeight: 5},
		{Name: "get-2", Path: "/get2", Method: "GET", Category: "trade", BaseWeight: 3},
		{Name: "post-1", Path: "/post1", Method: "POST", Category: "catalog", BaseWeight: 2},
		{Name: "put-1", Path: "/put1", Method: "PUT", Category: "trade", BaseWeight: 1},
	}
	require.NoError(t, selector.RegisterAll(endpoints))

	stats := selector.GetStats()

	assert.Equal(t, 4, stats.TotalEndpoints)
	assert.Equal(t, 2, stats.ReadEndpoints)
	assert.Equal(t, 2, stats.WriteEndpoints)
	assert.Equal(t, 8, stats.TotalReadWeight)  // 5 + 3
	assert.Equal(t, 3, stats.TotalWriteWeight) // 2 + 1
	assert.Equal(t, 2, stats.Categories["catalog"])
	assert.Equal(t, 2, stats.Categories["trade"])
	assert.Equal(t, 2, stats.Operations[OpGET])
	assert.Equal(t, 1, stats.Operations[OpPOST])
	assert.Equal(t, 1, stats.Operations[OpPUT])
}

// TestUpdateConfig verifies configuration updates.
func TestUpdateConfig(t *testing.T) {
	selector, err := NewWeightedSelector(WeightedSelectorConfig{ReadWriteRatio: 0.5})
	require.NoError(t, err)

	endpoints := []*Endpoint{
		{Name: "read", Path: "/read", Method: "GET", BaseWeight: 1},
		{Name: "write", Path: "/write", Method: "POST", BaseWeight: 1},
	}
	require.NoError(t, selector.RegisterAll(endpoints))

	// Update to 100% read
	err = selector.UpdateConfig(WeightedSelectorConfig{ReadWriteRatio: 1.0})
	require.NoError(t, err)

	// Verify all selections are reads
	for i := 0; i < 100; i++ {
		ep, err := selector.Select()
		require.NoError(t, err)
		assert.True(t, ep.Operation().IsRead())
	}

	// Invalid config should fail
	err = selector.UpdateConfig(WeightedSelectorConfig{ReadWriteRatio: 2.0})
	assert.Error(t, err)
}

// TestGetConfig verifies config retrieval and immutability.
func TestGetConfig(t *testing.T) {
	original := WeightedSelectorConfig{
		ReadWriteRatio: 0.7,
		Categories:     map[string]int{"cat1": 5},
		CategoryRatios: map[string]float64{"cat1": 0.9},
		Weights:        map[string]int{"ep1": 10},
		Operations:     map[OperationType]int{OpGET: 3},
	}

	selector, err := NewWeightedSelector(original)
	require.NoError(t, err)

	retrieved := selector.GetConfig()

	// Verify values
	assert.Equal(t, 0.7, retrieved.ReadWriteRatio)
	assert.Equal(t, 5, retrieved.Categories["cat1"])
	assert.Equal(t, 0.9, retrieved.CategoryRatios["cat1"])
	assert.Equal(t, 10, retrieved.Weights["ep1"])
	assert.Equal(t, 3, retrieved.Operations[OpGET])

	// Modify retrieved config - should not affect selector
	retrieved.Categories["cat1"] = 100

	afterMod := selector.GetConfig()
	assert.Equal(t, 5, afterMod.Categories["cat1"]) // Still 5
}

// TestConcurrentAccess verifies thread safety.
func TestConcurrentAccess(t *testing.T) {
	selector, err := NewWeightedSelector(WeightedSelectorConfig{ReadWriteRatio: 0.5})
	require.NoError(t, err)

	endpoints := []*Endpoint{
		{Name: "read-1", Path: "/read1", Method: "GET", BaseWeight: 1},
		{Name: "write-1", Path: "/write1", Method: "POST", BaseWeight: 1},
	}
	require.NoError(t, selector.RegisterAll(endpoints))

	var wg sync.WaitGroup
	var errors atomic.Int32
	goroutines := 10
	iterations := 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_, err := selector.Select()
				if err != nil {
					errors.Add(1)
				}
			}
		}()
	}

	wg.Wait()
	assert.Equal(t, int32(0), errors.Load())
}

// TestConcurrentRegistrationAndSelection verifies concurrent registration and selection.
func TestConcurrentRegistrationAndSelection(t *testing.T) {
	selector, err := NewWeightedSelector(WeightedSelectorConfig{ReadWriteRatio: 0.5})
	require.NoError(t, err)

	// Pre-register some endpoints
	endpoints := []*Endpoint{
		{Name: "initial-1", Path: "/initial1", Method: "GET", BaseWeight: 1},
		{Name: "initial-2", Path: "/initial2", Method: "POST", BaseWeight: 1},
	}
	require.NoError(t, selector.RegisterAll(endpoints))

	var wg sync.WaitGroup
	var errors atomic.Int32

	// Concurrent selections
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, err := selector.Select()
				if err != nil {
					errors.Add(1)
				}
			}
		}()
	}

	// Concurrent registrations
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ep := &Endpoint{
				Name:       "dynamic-" + string(rune('a'+idx)),
				Path:       "/dynamic" + string(rune('a'+idx)),
				Method:     "GET",
				BaseWeight: 1,
			}
			_ = selector.Register(ep) // Ignore duplicates
		}(i)
	}

	wg.Wait()
	// Selection errors are acceptable during concurrent registration
	t.Logf("Selection errors during concurrent registration: %d", errors.Load())
}

// TestZeroWeightEndpoints verifies that zero-weight endpoints are excluded.
func TestZeroWeightEndpoints(t *testing.T) {
	selector, err := NewWeightedSelector(WeightedSelectorConfig{
		ReadWriteRatio: 1.0,
		Weights: map[string]int{
			"excluded": 0, // Override to 0 weight
		},
	})
	require.NoError(t, err)

	endpoints := []*Endpoint{
		{Name: "included", Path: "/included", Method: "GET", BaseWeight: 1},
		{Name: "excluded", Path: "/excluded", Method: "GET", BaseWeight: 1}, // Overridden to 0
	}
	require.NoError(t, selector.RegisterAll(endpoints))

	// Verify excluded endpoint is never selected
	for i := 0; i < 100; i++ {
		ep, err := selector.Select()
		require.NoError(t, err)
		assert.Equal(t, "included", ep.Name)
	}
}

// TestEffectiveWeightCalculation verifies combined weight calculation.
func TestEffectiveWeightCalculation(t *testing.T) {
	selector, err := NewWeightedSelector(WeightedSelectorConfig{
		ReadWriteRatio: 1.0,
		Categories:     map[string]int{"priority": 2},     // 2x multiplier
		Operations:     map[OperationType]int{OpGET: 3},   // 3x multiplier
		Weights:        map[string]int{"overridden": 100}, // Override base weight
	})
	require.NoError(t, err)

	endpoints := []*Endpoint{
		// Base 1, Category 2x, Operation 3x = 1 * 2 * 3 = 6
		{Name: "combined", Path: "/combined", Method: "GET", Category: "priority", BaseWeight: 1},
		// Base overridden to 100, no category multiplier, Operation 3x = 100 * 1 * 3 = 300
		{Name: "overridden", Path: "/overridden", Method: "GET", BaseWeight: 1},
		// Base 1, no multipliers = 1
		{Name: "plain", Path: "/plain", Method: "GET", BaseWeight: 1},
	}
	require.NoError(t, selector.RegisterAll(endpoints))

	w1, _ := selector.GetEffectiveWeight("combined")
	w2, _ := selector.GetEffectiveWeight("overridden")
	w3, _ := selector.GetEffectiveWeight("plain")

	assert.Equal(t, 6, w1)   // 1 * 2 * 3
	assert.Equal(t, 300, w2) // 100 * 1 * 3
	assert.Equal(t, 3, w3)   // 1 * 1 * 3 (operation multiplier still applies)
}

// TestEffectiveWeightOverflowProtection verifies overflow protection in weight calculation.
func TestEffectiveWeightOverflowProtection(t *testing.T) {
	selector, err := NewWeightedSelector(WeightedSelectorConfig{
		ReadWriteRatio:    1.0,
		ReadWriteRatioSet: true,
		Categories:        map[string]int{"huge": 1000000},    // 1M multiplier
		Operations:        map[OperationType]int{OpGET: 1000}, // 1K multiplier
	})
	require.NoError(t, err)

	// Base 10000 * Category 1000000 * Operation 1000 = 10^13 (would overflow int32)
	endpoints := []*Endpoint{
		{Name: "overflow", Path: "/overflow", Method: "GET", Category: "huge", BaseWeight: 10000},
	}
	require.NoError(t, selector.RegisterAll(endpoints))

	w, err := selector.GetEffectiveWeight("overflow")
	require.NoError(t, err)

	// Should be capped at max int32 (2^31 - 1 = 2147483647)
	assert.Equal(t, 1<<31-1, w)
}

// TestConfigApplyDefaults verifies default configuration values.
func TestConfigApplyDefaults(t *testing.T) {
	config := WeightedSelectorConfig{}
	config.ApplyDefaults()

	assert.Equal(t, 0.8, config.ReadWriteRatio)
	assert.NotNil(t, config.Categories)
	assert.NotNil(t, config.CategoryRatios)
	assert.NotNil(t, config.Weights)
	assert.NotNil(t, config.Operations)
}

// TestHighVolumeDistribution tests distribution accuracy with high volume.
func TestHighVolumeDistribution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping high-volume test in short mode")
	}

	selector, err := NewWeightedSelector(WeightedSelectorConfig{ReadWriteRatio: 0.8})
	require.NoError(t, err)

	endpoints := []*Endpoint{
		{Name: "read", Path: "/read", Method: "GET", BaseWeight: 1},
		{Name: "write", Path: "/write", Method: "POST", BaseWeight: 1},
	}
	require.NoError(t, selector.RegisterAll(endpoints))

	iterations := 10000
	readCount := 0
	for i := 0; i < iterations; i++ {
		ep, err := selector.Select()
		require.NoError(t, err)
		if ep.Operation().IsRead() {
			readCount++
		}
	}

	actualRatio := float64(readCount) / float64(iterations)
	errorPct := math.Abs(actualRatio-0.8) * 100

	t.Logf("High-volume test (10000 iterations): %.4f read ratio, %.2f%% error", actualRatio, errorPct)

	// Higher volume should have even lower error
	assert.LessOrEqual(t, errorPct, 2.0,
		"High-volume distribution error %.2f%% exceeds 2%%", errorPct)
}

// Benchmarks

func BenchmarkSelect(b *testing.B) {
	selector, _ := NewWeightedSelector(WeightedSelectorConfig{ReadWriteRatio: 0.8})
	endpoints := make([]*Endpoint, 20)
	for i := 0; i < 20; i++ {
		method := "GET"
		if i%4 != 0 {
			method = []string{"POST", "PUT", "DELETE"}[i%3]
		}
		endpoints[i] = &Endpoint{
			Name:       "endpoint-" + string(rune('a'+i)),
			Path:       "/path" + string(rune('a'+i)),
			Method:     method,
			Category:   []string{"cat1", "cat2", "cat3"}[i%3],
			BaseWeight: (i % 5) + 1,
		}
	}
	_ = selector.RegisterAll(endpoints)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = selector.Select()
	}
}

func BenchmarkSelectByCategory(b *testing.B) {
	selector, _ := NewWeightedSelector(WeightedSelectorConfig{
		ReadWriteRatio: 0.8,
		Categories:     map[string]int{"catalog": 5, "trade": 3},
	})
	endpoints := []*Endpoint{
		{Name: "ep1", Path: "/ep1", Method: "GET", Category: "catalog", BaseWeight: 1},
		{Name: "ep2", Path: "/ep2", Method: "POST", Category: "catalog", BaseWeight: 1},
		{Name: "ep3", Path: "/ep3", Method: "GET", Category: "trade", BaseWeight: 1},
	}
	_ = selector.RegisterAll(endpoints)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = selector.SelectByCategory("catalog")
	}
}

func BenchmarkConcurrentSelect(b *testing.B) {
	selector, _ := NewWeightedSelector(WeightedSelectorConfig{ReadWriteRatio: 0.8})
	endpoints := []*Endpoint{
		{Name: "read", Path: "/read", Method: "GET", BaseWeight: 1},
		{Name: "write", Path: "/write", Method: "POST", BaseWeight: 1},
	}
	_ = selector.RegisterAll(endpoints)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = selector.Select()
		}
	})
}
