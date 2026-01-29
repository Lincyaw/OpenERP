package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSelector(t *testing.T) {
	tests := []struct {
		name           string
		config         *Config
		expectedCount  int
		expectedWeight int64
	}{
		{
			name:           "nil config",
			config:         nil,
			expectedCount:  0,
			expectedWeight: 0,
		},
		{
			name: "empty workflows",
			config: &Config{
				Workflows: map[string]Definition{},
			},
			expectedCount:  0,
			expectedWeight: 0,
		},
		{
			name: "single workflow",
			config: &Config{
				Workflows: map[string]Definition{
					"workflow1": {
						Weight: 5,
						Steps:  []Step{{Endpoint: "GET /api/test"}},
					},
				},
			},
			expectedCount:  1,
			expectedWeight: 5,
		},
		{
			name: "multiple workflows",
			config: &Config{
				Workflows: map[string]Definition{
					"workflow1": {Weight: 5, Steps: []Step{{Endpoint: "GET /test"}}},
					"workflow2": {Weight: 10, Steps: []Step{{Endpoint: "GET /test"}}},
					"workflow3": {Weight: 3, Steps: []Step{{Endpoint: "GET /test"}}},
				},
			},
			expectedCount:  3,
			expectedWeight: 18,
		},
		{
			name: "excludes disabled workflows",
			config: &Config{
				Workflows: map[string]Definition{
					"enabled":  {Weight: 10, Steps: []Step{{Endpoint: "GET /test"}}},
					"disabled": {Weight: 100, Disabled: true, Steps: []Step{{Endpoint: "GET /test"}}},
				},
			},
			expectedCount:  1,
			expectedWeight: 10,
		},
		{
			name: "default weight for zero weight",
			config: &Config{
				Workflows: map[string]Definition{
					"workflow1": {Weight: 0, Steps: []Step{{Endpoint: "GET /test"}}}, // Should default to 1
				},
			},
			expectedCount:  1,
			expectedWeight: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := NewSelector(tt.config)
			assert.NotNil(t, selector)
			assert.Equal(t, tt.expectedCount, selector.Count())
			assert.Equal(t, tt.expectedWeight, selector.TotalWeight())
		})
	}
}

func TestSelector_Select(t *testing.T) {
	// Test with no workflows
	t.Run("no workflows returns false", func(t *testing.T) {
		selector := NewSelector(nil)
		name, def, ok := selector.Select()
		assert.False(t, ok)
		assert.Empty(t, name)
		assert.Empty(t, def.Name)
	})

	// Test with single workflow (always selects it)
	t.Run("single workflow always selected", func(t *testing.T) {
		config := &Config{
			Workflows: map[string]Definition{
				"only_workflow": {
					Weight: 1,
					Steps:  []Step{{Endpoint: "GET /test"}},
				},
			},
		}
		selector := NewSelector(config)

		for i := 0; i < 10; i++ {
			name, def, ok := selector.Select()
			assert.True(t, ok)
			assert.Equal(t, "only_workflow", name)
			assert.NotEmpty(t, def.Steps)
		}
	})

	// Test with multiple workflows (weighted selection)
	t.Run("multiple workflows weighted selection", func(t *testing.T) {
		config := &Config{
			Workflows: map[string]Definition{
				"heavy":  {Weight: 100, Steps: []Step{{Endpoint: "GET /test"}}},
				"light":  {Weight: 1, Steps: []Step{{Endpoint: "GET /test"}}},
				"medium": {Weight: 10, Steps: []Step{{Endpoint: "GET /test"}}},
			},
		}
		selector := NewSelector(config)

		// Run many selections and verify heavy is selected most often
		counts := make(map[string]int)
		iterations := 1000

		for i := 0; i < iterations; i++ {
			name, _, ok := selector.Select()
			assert.True(t, ok)
			counts[name]++
		}

		// Heavy (weight 100) should be selected most often
		assert.Greater(t, counts["heavy"], counts["medium"])
		assert.Greater(t, counts["medium"], counts["light"])

		// Heavy should be roughly 100x more likely than light
		// Allow some variance due to randomness
		assert.Greater(t, float64(counts["heavy"])/float64(counts["light"]+1), 10.0)
	})
}

func TestSelector_SelectByName(t *testing.T) {
	config := &Config{
		Workflows: map[string]Definition{
			"workflow1": {Name: "workflow1", Weight: 5, Steps: []Step{{Endpoint: "GET /test1"}}},
			"workflow2": {Name: "workflow2", Weight: 10, Steps: []Step{{Endpoint: "GET /test2"}}},
		},
	}
	selector := NewSelector(config)

	// Test existing workflow
	def, ok := selector.SelectByName("workflow1")
	assert.True(t, ok)
	assert.Equal(t, "workflow1", def.Name)

	// Test non-existing workflow
	def, ok = selector.SelectByName("nonexistent")
	assert.False(t, ok)
	assert.Empty(t, def.Name)
}

func TestSelector_GetAll(t *testing.T) {
	config := &Config{
		Workflows: map[string]Definition{
			"workflow1": {Name: "workflow1", Steps: []Step{{Endpoint: "GET /test"}}},
			"workflow2": {Name: "workflow2", Steps: []Step{{Endpoint: "GET /test"}}},
			"disabled":  {Name: "disabled", Disabled: true, Steps: []Step{{Endpoint: "GET /test"}}},
		},
	}
	selector := NewSelector(config)

	all := selector.GetAll()
	assert.Len(t, all, 2)
	assert.Contains(t, all, "workflow1")
	assert.Contains(t, all, "workflow2")
	assert.NotContains(t, all, "disabled")
}

func TestSelector_Names(t *testing.T) {
	config := &Config{
		Workflows: map[string]Definition{
			"alpha": {Steps: []Step{{Endpoint: "GET /test"}}},
			"beta":  {Steps: []Step{{Endpoint: "GET /test"}}},
			"gamma": {Steps: []Step{{Endpoint: "GET /test"}}},
			"delta": {Disabled: true, Steps: []Step{{Endpoint: "GET /test"}}},
		},
	}
	selector := NewSelector(config)

	names := selector.Names()
	assert.Len(t, names, 3)
	assert.Contains(t, names, "alpha")
	assert.Contains(t, names, "beta")
	assert.Contains(t, names, "gamma")
	assert.NotContains(t, names, "delta")
}

func TestSelector_GetWeight(t *testing.T) {
	config := &Config{
		Workflows: map[string]Definition{
			"heavy": {Weight: 100, Steps: []Step{{Endpoint: "GET /test"}}},
			"light": {Weight: 1, Steps: []Step{{Endpoint: "GET /test"}}},
		},
	}
	selector := NewSelector(config)

	weight, ok := selector.GetWeight("heavy")
	assert.True(t, ok)
	assert.Equal(t, 100, weight)

	weight, ok = selector.GetWeight("light")
	assert.True(t, ok)
	assert.Equal(t, 1, weight)

	weight, ok = selector.GetWeight("nonexistent")
	assert.False(t, ok)
	assert.Equal(t, 0, weight)
}

func TestSelector_Update(t *testing.T) {
	// Start with initial config
	initialConfig := &Config{
		Workflows: map[string]Definition{
			"workflow1": {Weight: 5, Steps: []Step{{Endpoint: "GET /test"}}},
		},
	}
	selector := NewSelector(initialConfig)

	assert.Equal(t, 1, selector.Count())
	assert.Equal(t, int64(5), selector.TotalWeight())

	// Update with new workflows
	newWorkflows := map[string]Definition{
		"workflow1": {Weight: 10, Steps: []Step{{Endpoint: "GET /test"}}},
		"workflow2": {Weight: 20, Steps: []Step{{Endpoint: "GET /test"}}},
		"workflow3": {Weight: 30, Steps: []Step{{Endpoint: "GET /test"}}},
	}
	selector.Update(newWorkflows)

	assert.Equal(t, 3, selector.Count())
	assert.Equal(t, int64(60), selector.TotalWeight())

	// Verify new workflows are selectable
	_, ok := selector.SelectByName("workflow2")
	assert.True(t, ok)

	_, ok = selector.SelectByName("workflow3")
	assert.True(t, ok)
}

func TestSelector_DeterministicOrdering(t *testing.T) {
	// Workflow names should be sorted deterministically for consistent behavior
	config := &Config{
		Workflows: map[string]Definition{
			"zebra":  {Weight: 1, Steps: []Step{{Endpoint: "GET /test"}}},
			"apple":  {Weight: 1, Steps: []Step{{Endpoint: "GET /test"}}},
			"mango":  {Weight: 1, Steps: []Step{{Endpoint: "GET /test"}}},
			"banana": {Weight: 1, Steps: []Step{{Endpoint: "GET /test"}}},
		},
	}

	// Create multiple selectors and verify they all have same ordering
	for i := 0; i < 5; i++ {
		selector := NewSelector(config)
		names := selector.Names()

		// Names should be sorted alphabetically
		require.Len(t, names, 4)
		assert.Equal(t, "apple", names[0])
		assert.Equal(t, "banana", names[1])
		assert.Equal(t, "mango", names[2])
		assert.Equal(t, "zebra", names[3])
	}
}

func TestSelector_ConcurrentAccess(t *testing.T) {
	config := &Config{
		Workflows: map[string]Definition{
			"workflow1": {Weight: 10, Steps: []Step{{Endpoint: "GET /test"}}},
			"workflow2": {Weight: 20, Steps: []Step{{Endpoint: "GET /test"}}},
		},
	}
	selector := NewSelector(config)

	// Run concurrent selections
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				name, _, ok := selector.Select()
				assert.True(t, ok)
				assert.NotEmpty(t, name)
			}
			done <- true
		}()
	}

	// Run concurrent reads
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = selector.Count()
				_ = selector.TotalWeight()
				_ = selector.Names()
				_ = selector.GetAll()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 15; i++ {
		<-done
	}
}

func TestSelector_AppliesDefaults(t *testing.T) {
	config := &Config{
		Workflows: map[string]Definition{
			"test_workflow": {
				// No name set, should be set from key
				// No weight set, should default to 1
				Steps: []Step{{Endpoint: "GET /test"}},
			},
		},
	}

	selector := NewSelector(config)
	def, ok := selector.SelectByName("test_workflow")

	assert.True(t, ok)
	assert.Equal(t, "test_workflow", def.Name)
	assert.Equal(t, 1, def.Weight)
}
