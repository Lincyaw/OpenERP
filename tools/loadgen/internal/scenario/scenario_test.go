package scenario

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/example/erp/tools/loadgen/internal/config"
	"github.com/example/erp/tools/loadgen/internal/loadctrl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefinition_Validate(t *testing.T) {
	tests := []struct {
		name    string
		def     Definition
		wantErr bool
	}{
		{
			name: "valid minimal definition",
			def: Definition{
				Name: "test_scenario",
			},
			wantErr: false,
		},
		{
			name: "valid full definition",
			def: Definition{
				Name:             "full_scenario",
				Description:      "A comprehensive test scenario",
				Duration:         10 * time.Minute,
				FocusEndpoints:   []string{"endpoint1", "endpoint2"},
				DisableEndpoints: []string{"endpoint3"},
				EndpointWeights:  map[string]int{"endpoint1": 10, "endpoint2": 20},
				Tags:             []string{"stress", "test"},
			},
			wantErr: false,
		},
		{
			name:    "missing name",
			def:     Definition{},
			wantErr: true,
		},
		{
			name: "negative duration",
			def: Definition{
				Name:     "test",
				Duration: -1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "negative endpoint weight",
			def: Definition{
				Name:            "test",
				EndpointWeights: map[string]int{"ep1": -1},
			},
			wantErr: true,
		},
		{
			name: "valid with traffic shaper",
			def: Definition{
				Name: "with_shaper",
				TrafficShaper: &loadctrl.ShaperConfig{
					Type:    "spike",
					BaseQPS: 100,
					Spike: &loadctrl.SpikeConfig{
						SpikeQPS:      500,
						SpikeDuration: 30 * time.Second,
						SpikeInterval: 2 * time.Minute,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.def.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefinition_HasOverrides(t *testing.T) {
	t.Run("HasDurationOverride", func(t *testing.T) {
		def := Definition{Name: "test"}
		assert.False(t, def.HasDurationOverride())

		def.Duration = 5 * time.Minute
		assert.True(t, def.HasDurationOverride())
	})

	t.Run("HasTrafficOverride", func(t *testing.T) {
		def := Definition{Name: "test"}
		assert.False(t, def.HasTrafficOverride())

		def.TrafficShaper = &loadctrl.ShaperConfig{Type: "spike"}
		assert.True(t, def.HasTrafficOverride())
	})

	t.Run("HasFocusEndpoints", func(t *testing.T) {
		def := Definition{Name: "test"}
		assert.False(t, def.HasFocusEndpoints())

		def.FocusEndpoints = []string{"ep1"}
		assert.True(t, def.HasFocusEndpoints())
	})
}

func TestDefinition_EndpointFiltering(t *testing.T) {
	def := Definition{
		Name:             "test",
		FocusEndpoints:   []string{"ep1", "ep2"},
		DisableEndpoints: []string{"ep3"},
		EndpointWeights:  map[string]int{"ep1": 50},
	}

	t.Run("IsEndpointFocused", func(t *testing.T) {
		assert.True(t, def.IsEndpointFocused("ep1"))
		assert.True(t, def.IsEndpointFocused("ep2"))
		assert.False(t, def.IsEndpointFocused("ep4"))
	})

	t.Run("IsEndpointFocused_NoFocusList", func(t *testing.T) {
		defNoFocus := Definition{Name: "test"}
		assert.True(t, defNoFocus.IsEndpointFocused("any_endpoint"))
	})

	t.Run("IsEndpointDisabled", func(t *testing.T) {
		assert.False(t, def.IsEndpointDisabled("ep1"))
		assert.True(t, def.IsEndpointDisabled("ep3"))
	})

	t.Run("GetEndpointWeight", func(t *testing.T) {
		assert.Equal(t, 50, def.GetEndpointWeight("ep1"))
		assert.Equal(t, -1, def.GetEndpointWeight("ep2")) // No override
	})
}

func TestRegistry(t *testing.T) {
	t.Run("basic operations", func(t *testing.T) {
		registry := NewRegistry()

		// Register a scenario
		def := &Definition{Name: "test_scenario", Description: "Test"}
		err := registry.Register(def)
		require.NoError(t, err)

		// Get the scenario
		retrieved, err := registry.Get("test_scenario")
		require.NoError(t, err)
		assert.Equal(t, "test_scenario", retrieved.Name)
		assert.Equal(t, "Test", retrieved.Description)

		// List scenarios
		names := registry.List()
		assert.Contains(t, names, "test_scenario")

		// Count
		assert.Equal(t, 1, registry.Count())
	})

	t.Run("not found", func(t *testing.T) {
		registry := NewRegistry()
		_, err := registry.Get("nonexistent")
		assert.ErrorIs(t, err, ErrScenarioNotFound)
	})

	t.Run("register invalid", func(t *testing.T) {
		registry := NewRegistry()
		def := &Definition{} // Missing name
		err := registry.Register(def)
		assert.ErrorIs(t, err, ErrInvalidScenario)
	})

	t.Run("all scenarios", func(t *testing.T) {
		registry := NewRegistry()
		_ = registry.Register(&Definition{Name: "scenario1"})
		_ = registry.Register(&Definition{Name: "scenario2"})

		all := registry.All()
		assert.Len(t, all, 2)
	})
}

func TestLoadFromFile(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	t.Run("load valid scenario", func(t *testing.T) {
		content := `
name: "test_scenario"
description: "A test scenario"
duration: 5m
focusEndpoints:
  - "endpoint1"
  - "endpoint2"
tags:
  - "test"
`
		path := filepath.Join(tmpDir, "valid.yaml")
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)

		def, err := LoadFromFile(path)
		require.NoError(t, err)
		assert.Equal(t, "test_scenario", def.Name)
		assert.Equal(t, "A test scenario", def.Description)
		assert.Equal(t, 5*time.Minute, def.Duration)
		assert.Equal(t, []string{"endpoint1", "endpoint2"}, def.FocusEndpoints)
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := LoadFromFile(filepath.Join(tmpDir, "nonexistent.yaml"))
		assert.ErrorIs(t, err, ErrScenarioNotFound)
	})

	t.Run("invalid yaml", func(t *testing.T) {
		path := filepath.Join(tmpDir, "invalid.yaml")
		err := os.WriteFile(path, []byte("invalid: yaml: content:"), 0644)
		require.NoError(t, err)

		_, err = LoadFromFile(path)
		assert.Error(t, err)
	})

	t.Run("validation fails", func(t *testing.T) {
		content := `description: "Missing name"`
		path := filepath.Join(tmpDir, "no_name.yaml")
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)

		_, err = LoadFromFile(path)
		assert.ErrorIs(t, err, ErrInvalidScenario)
	})
}

func TestLoadMultipleFromFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("load multiple scenarios", func(t *testing.T) {
		content := `
version: "1.0"
scenarios:
  - name: "scenario1"
    description: "First scenario"
  - name: "scenario2"
    description: "Second scenario"
`
		path := filepath.Join(tmpDir, "multiple.yaml")
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)

		defs, err := LoadMultipleFromFile(path)
		require.NoError(t, err)
		assert.Len(t, defs, 2)
		assert.Equal(t, "scenario1", defs[0].Name)
		assert.Equal(t, "scenario2", defs[1].Name)
	})

	t.Run("fallback to single format", func(t *testing.T) {
		content := `
name: "single_scenario"
description: "A single scenario file"
`
		path := filepath.Join(tmpDir, "single.yaml")
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)

		defs, err := LoadMultipleFromFile(path)
		require.NoError(t, err)
		assert.Len(t, defs, 1)
		assert.Equal(t, "single_scenario", defs[0].Name)
	})
}

func TestRegistry_LoadFromDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create scenario files
	scenario1 := `
name: "scenario1"
description: "First scenario"
`
	scenario2 := `
name: "scenario2"
description: "Second scenario"
`
	err := os.WriteFile(filepath.Join(tmpDir, "scenario1.yaml"), []byte(scenario1), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "scenario2.yml"), []byte(scenario2), 0644)
	require.NoError(t, err)

	// Create a non-yaml file that should be ignored
	err = os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("ignore me"), 0644)
	require.NoError(t, err)

	t.Run("load from directory", func(t *testing.T) {
		registry := NewRegistry()
		registry.SetDirectory(tmpDir)

		err := registry.LoadFromDirectory()
		require.NoError(t, err)

		assert.Equal(t, 2, registry.Count())

		def1, err := registry.Get("scenario1")
		require.NoError(t, err)
		assert.Equal(t, "First scenario", def1.Description)

		def2, err := registry.Get("scenario2")
		require.NoError(t, err)
		assert.Equal(t, "Second scenario", def2.Description)
	})

	t.Run("empty directory", func(t *testing.T) {
		emptyDir := filepath.Join(tmpDir, "empty")
		err := os.Mkdir(emptyDir, 0755)
		require.NoError(t, err)

		registry := NewRegistry()
		registry.SetDirectory(emptyDir)

		err = registry.LoadFromDirectory()
		require.NoError(t, err)
		assert.Equal(t, 0, registry.Count())
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		registry := NewRegistry()
		registry.SetDirectory(filepath.Join(tmpDir, "nonexistent"))

		err := registry.LoadFromDirectory()
		require.NoError(t, err) // Should not error, just skip
		assert.Equal(t, 0, registry.Count())
	})

	t.Run("no directory set", func(t *testing.T) {
		registry := NewRegistry()
		err := registry.LoadFromDirectory()
		require.NoError(t, err)
	})
}

func TestRegistry_LoadFromConfig(t *testing.T) {
	registry := NewRegistry()

	scenarios := []InlineScenario{
		{
			Name:        "browse_catalog",
			Description: "Browse products",
			Endpoints:   []string{"catalog.products.list", "catalog.products.get"},
			Weight:      30,
			Sequential:  false,
		},
		{
			Name:        "create_order",
			Description: "Create an order",
			Endpoints:   []string{"trade.sales_orders.create"},
			Weight:      10,
			Sequential:  true,
		},
	}

	err := registry.LoadFromConfig(scenarios)
	require.NoError(t, err)

	assert.Equal(t, 2, registry.Count())

	def1, err := registry.Get("browse_catalog")
	require.NoError(t, err)
	assert.Equal(t, []string{"catalog.products.list", "catalog.products.get"}, def1.FocusEndpoints)

	def2, err := registry.Get("create_order")
	require.NoError(t, err)
	assert.Contains(t, def2.Tags, "sequential")
}

func TestRunner(t *testing.T) {
	baseConfig := &config.Config{
		Name:     "Test Config",
		Duration: 5 * time.Minute,
		TrafficShaper: loadctrl.ShaperConfig{
			Type:    "spike",
			BaseQPS: 100,
			Spike: &loadctrl.SpikeConfig{
				SpikeQPS:      300,
				SpikeDuration: 10 * time.Second,
				SpikeInterval: 1 * time.Minute,
			},
		},
		Endpoints: []config.EndpointConfig{
			{Name: "ep1", Method: "GET", Path: "/ep1", Weight: 10},
			{Name: "ep2", Method: "GET", Path: "/ep2", Weight: 20},
			{Name: "ep3", Method: "POST", Path: "/ep3", Weight: 5},
		},
	}

	t.Run("apply duration override", func(t *testing.T) {
		def := &Definition{
			Name:     "test",
			Duration: 10 * time.Minute,
		}

		runner := NewRunner(def, baseConfig)
		applied, err := runner.ApplyOverrides()
		require.NoError(t, err)

		assert.Equal(t, 10*time.Minute, applied.Duration)
		assert.Contains(t, runner.Stats().OverridesApplied, "duration")
	})

	t.Run("apply traffic shaper override", func(t *testing.T) {
		def := &Definition{
			Name: "test",
			TrafficShaper: &loadctrl.ShaperConfig{
				Type:    "spike",
				BaseQPS: 50,
				Spike: &loadctrl.SpikeConfig{
					SpikeQPS:      200,
					SpikeDuration: 15 * time.Second,
					SpikeInterval: 1 * time.Minute,
				},
			},
		}

		runner := NewRunner(def, baseConfig)
		applied, err := runner.ApplyOverrides()
		require.NoError(t, err)

		assert.Equal(t, "spike", applied.TrafficShaper.Type)
		assert.Equal(t, float64(50), applied.TrafficShaper.BaseQPS)
		assert.Contains(t, runner.Stats().OverridesApplied, "trafficShaper")
	})

	t.Run("apply focus endpoints", func(t *testing.T) {
		def := &Definition{
			Name:           "test",
			FocusEndpoints: []string{"ep1", "ep2"},
		}

		runner := NewRunner(def, baseConfig)
		applied, err := runner.ApplyOverrides()
		require.NoError(t, err)

		// ep1 and ep2 should be enabled, ep3 should be disabled
		assert.False(t, applied.Endpoints[0].Disabled) // ep1
		assert.False(t, applied.Endpoints[1].Disabled) // ep2
		assert.True(t, applied.Endpoints[2].Disabled)  // ep3

		stats := runner.Stats()
		assert.Equal(t, 2, stats.EndpointsActive)
		assert.Equal(t, 1, stats.EndpointsDisabled)
	})

	t.Run("apply disable endpoints", func(t *testing.T) {
		def := &Definition{
			Name:             "test",
			DisableEndpoints: []string{"ep3"},
		}

		runner := NewRunner(def, baseConfig)
		applied, err := runner.ApplyOverrides()
		require.NoError(t, err)

		assert.False(t, applied.Endpoints[0].Disabled) // ep1
		assert.False(t, applied.Endpoints[1].Disabled) // ep2
		assert.True(t, applied.Endpoints[2].Disabled)  // ep3
	})

	t.Run("apply endpoint weight override", func(t *testing.T) {
		def := &Definition{
			Name:            "test",
			EndpointWeights: map[string]int{"ep1": 100},
		}

		runner := NewRunner(def, baseConfig)
		applied, err := runner.ApplyOverrides()
		require.NoError(t, err)

		assert.Equal(t, 100, applied.Endpoints[0].Weight) // ep1 overridden
		assert.Equal(t, 20, applied.Endpoints[1].Weight)  // ep2 unchanged
	})

	t.Run("apply warmup override", func(t *testing.T) {
		disabled := false
		def := &Definition{
			Name: "test",
			Warmup: &WarmupOverride{
				Enabled: &disabled,
			},
		}

		runner := NewRunner(def, baseConfig)
		applied, err := runner.ApplyOverrides()
		require.NoError(t, err)

		assert.Equal(t, 0, applied.Warmup.Iterations) // Disabled sets to 0
		assert.Contains(t, runner.Stats().OverridesApplied, "warmup")
	})

	t.Run("apply warmup iterations override", func(t *testing.T) {
		def := &Definition{
			Name: "test",
			Warmup: &WarmupOverride{
				Iterations: 5,
				Timeout:    1 * time.Minute,
			},
		}

		runner := NewRunner(def, baseConfig)
		applied, err := runner.ApplyOverrides()
		require.NoError(t, err)

		assert.Equal(t, 5, applied.Warmup.Iterations)
		assert.Equal(t, 1*time.Minute, applied.Warmup.Timeout)
		assert.Contains(t, runner.Stats().OverridesApplied, "warmup")
	})

	t.Run("apply assertion override", func(t *testing.T) {
		maxError := 5.0
		def := &Definition{
			Name: "test",
			Assertions: &AssertionOverride{
				MaxErrorRate:  &maxError,
				MaxP95Latency: 2 * time.Second,
			},
		}

		runner := NewRunner(def, baseConfig)
		applied, err := runner.ApplyOverrides()
		require.NoError(t, err)

		require.NotNil(t, applied.Assertions.Global)
		assert.Equal(t, &maxError, applied.Assertions.Global.MaxErrorRate)
		assert.Equal(t, 2*time.Second, applied.Assertions.Global.MaxP95Latency)
	})

	t.Run("config name includes scenario", func(t *testing.T) {
		def := &Definition{Name: "my_scenario"}

		runner := NewRunner(def, baseConfig)
		applied, err := runner.ApplyOverrides()
		require.NoError(t, err)

		assert.Contains(t, applied.Name, "my_scenario")
		assert.Contains(t, applied.Name, "Scenario")
	})

	t.Run("base config not modified", func(t *testing.T) {
		def := &Definition{
			Name:     "test",
			Duration: 1 * time.Hour,
		}

		runner := NewRunner(def, baseConfig)
		_, err := runner.ApplyOverrides()
		require.NoError(t, err)

		// Base config should still have original duration
		assert.Equal(t, 5*time.Minute, baseConfig.Duration)
	})
}

func TestRunner_Lifecycle(t *testing.T) {
	baseConfig := &config.Config{
		Name:     "Test",
		Duration: 1 * time.Minute,
		Endpoints: []config.EndpointConfig{
			{Name: "ep1", Method: "GET", Path: "/ep1", Weight: 10},
		},
	}

	def := &Definition{Name: "test"}
	runner := NewRunner(def, baseConfig)

	t.Run("initial state", func(t *testing.T) {
		assert.False(t, runner.IsRunning())
		assert.False(t, runner.IsCompleted())
	})

	t.Run("start and complete", func(t *testing.T) {
		ctx := context.Background()
		_, cancel := runner.Start(ctx)
		defer cancel()

		assert.True(t, runner.IsRunning())
		assert.False(t, runner.IsCompleted())

		// Simulate some work
		time.Sleep(10 * time.Millisecond)

		runner.Complete()

		assert.False(t, runner.IsRunning())
		assert.True(t, runner.IsCompleted())

		stats := runner.Stats()
		assert.True(t, stats.Duration >= 10*time.Millisecond)
		assert.False(t, stats.EndTime.IsZero())
	})

	t.Run("cancel", func(t *testing.T) {
		def2 := &Definition{Name: "test2"}
		runner2 := NewRunner(def2, baseConfig)

		ctx := context.Background()
		childCtx, _ := runner2.Start(ctx)

		runner2.Cancel()

		// Context should be canceled
		select {
		case <-childCtx.Done():
			// Expected
		default:
			t.Error("Expected context to be canceled")
		}

		assert.False(t, runner2.IsRunning())
	})

	t.Run("stats during run", func(t *testing.T) {
		def3 := &Definition{Name: "test3"}
		runner3 := NewRunner(def3, baseConfig)
		_, _ = runner3.ApplyOverrides()

		ctx := context.Background()
		_, cancel := runner3.Start(ctx)
		defer cancel()

		// Stats should show running duration
		time.Sleep(5 * time.Millisecond)
		stats := runner3.Stats()
		assert.True(t, stats.Duration > 0)
	})
}

func TestFlashSaleScenario(t *testing.T) {
	// Test loading the actual flash_sale scenario
	// This test verifies the acceptance criteria

	// Find the project root
	cwd, err := os.Getwd()
	require.NoError(t, err)

	scenarioPath := filepath.Join(cwd, "..", "..", "..", "configs", "scenarios", "flash_sale.yaml")

	// Check if file exists (may not exist in test environment)
	if _, err := os.Stat(scenarioPath); os.IsNotExist(err) {
		t.Skip("flash_sale.yaml not found, skipping integration test")
	}

	def, err := LoadFromFile(scenarioPath)
	require.NoError(t, err)

	// Verify flash_sale scenario configuration
	assert.Equal(t, "flash_sale", def.Name)
	assert.True(t, def.HasDurationOverride())
	assert.True(t, def.HasTrafficOverride())
	assert.True(t, def.HasFocusEndpoints())

	// Verify traffic shaper is spike type
	assert.NotNil(t, def.TrafficShaper)
	assert.Equal(t, "spike", def.TrafficShaper.Type)

	// Verify focus endpoints include catalog and order endpoints
	assert.Contains(t, def.FocusEndpoints, "catalog.products.list")
	assert.Contains(t, def.FocusEndpoints, "trade.sales_orders.create")

	// Verify warmup is disabled
	require.NotNil(t, def.Warmup)
	require.NotNil(t, def.Warmup.Enabled)
	assert.False(t, *def.Warmup.Enabled)

	// Verify assertions are set
	assert.NotNil(t, def.Assertions)
	assert.NotNil(t, def.Assertions.MaxErrorRate)
}
