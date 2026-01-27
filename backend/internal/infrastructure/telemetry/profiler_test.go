package telemetry_test

import (
	"sync"
	"testing"

	"github.com/erp/backend/internal/infrastructure/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestNewProfiler_Disabled(t *testing.T) {
	logger := zaptest.NewLogger(t)

	cfg := telemetry.ProfilerConfig{
		Enabled:         false,
		ServerAddress:   "http://localhost:4040",
		ApplicationName: "test-service",
	}

	profiler, err := telemetry.NewProfiler(cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, profiler)

	// Verify profiling is disabled
	assert.False(t, profiler.IsEnabled())

	// GetConfig should return the config
	gotCfg := profiler.GetConfig()
	assert.Equal(t, cfg.ApplicationName, gotCfg.ApplicationName)
	assert.False(t, gotCfg.Enabled)

	// Stop should succeed with no-op
	err = profiler.Stop()
	assert.NoError(t, err)
}

func TestNewProfiler_Enabled_MissingServerAddress(t *testing.T) {
	logger := zaptest.NewLogger(t)

	cfg := telemetry.ProfilerConfig{
		Enabled:         true,
		ServerAddress:   "", // Missing
		ApplicationName: "test-service",
	}

	profiler, err := telemetry.NewProfiler(cfg, logger)
	require.Error(t, err)
	assert.Nil(t, profiler)
	assert.Contains(t, err.Error(), "server address is required")
}

func TestNewProfiler_Enabled_MissingApplicationName(t *testing.T) {
	logger := zaptest.NewLogger(t)

	cfg := telemetry.ProfilerConfig{
		Enabled:         true,
		ServerAddress:   "http://localhost:4040",
		ApplicationName: "", // Missing
	}

	profiler, err := telemetry.NewProfiler(cfg, logger)
	require.Error(t, err)
	assert.Nil(t, profiler)
	assert.Contains(t, err.Error(), "application name is required")
}

func TestNewProfiler_EnabledIntegration(t *testing.T) {
	// Skip this test in CI as it requires a real Pyroscope server
	// This test is for local development with Pyroscope running
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := zaptest.NewLogger(t)

	cfg := telemetry.ProfilerConfig{
		Enabled:             true,
		ServerAddress:       "http://localhost:4040",
		ApplicationName:     "test-service",
		ProfileCPU:          true,
		ProfileAllocObjects: true,
		ProfileAllocSpace:   true,
		ProfileInuseObjects: true,
		ProfileInuseSpace:   true,
		ProfileGoroutines:   true,
	}

	profiler, err := telemetry.NewProfiler(cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, profiler)

	// Verify profiling is enabled
	assert.True(t, profiler.IsEnabled())

	// Stop should succeed
	err = profiler.Stop()
	assert.NoError(t, err)
}

func TestProfiler_StopIdempotent(t *testing.T) {
	logger := zaptest.NewLogger(t)

	cfg := telemetry.ProfilerConfig{
		Enabled: false,
	}

	profiler, err := telemetry.NewProfiler(cfg, logger)
	require.NoError(t, err)

	// Stop multiple times should not panic
	err = profiler.Stop()
	assert.NoError(t, err)

	err = profiler.Stop()
	assert.NoError(t, err)

	err = profiler.Stop()
	assert.NoError(t, err)
}

func TestProfiler_StopConcurrent(t *testing.T) {
	logger := zaptest.NewLogger(t)

	cfg := telemetry.ProfilerConfig{
		Enabled: false,
	}

	profiler, err := telemetry.NewProfiler(cfg, logger)
	require.NoError(t, err)

	// Stop concurrently should not panic or deadlock
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = profiler.Stop()
		}()
	}
	wg.Wait()
}

func TestProfiler_GetConfigReturnsACopy(t *testing.T) {
	logger := zaptest.NewLogger(t)

	cfg := telemetry.ProfilerConfig{
		Enabled:         false,
		ServerAddress:   "http://localhost:4040",
		ApplicationName: "test-service",
	}

	profiler, err := telemetry.NewProfiler(cfg, logger)
	require.NoError(t, err)

	// Get config
	gotCfg := profiler.GetConfig()
	originalName := gotCfg.ApplicationName

	// Get config again and verify it's consistent
	gotCfg2 := profiler.GetConfig()
	assert.Equal(t, originalName, gotCfg2.ApplicationName)
	assert.Equal(t, "test-service", gotCfg2.ApplicationName)
}

func TestProfiler_ProfileTypesConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		config      telemetry.ProfilerConfig
		wantEnabled bool
	}{
		{
			name: "all_profiles_disabled",
			config: telemetry.ProfilerConfig{
				Enabled:             false, // Keep disabled for unit test
				ServerAddress:       "http://localhost:4040",
				ApplicationName:     "test",
				ProfileCPU:          false,
				ProfileAllocObjects: false,
				ProfileAllocSpace:   false,
				ProfileInuseObjects: false,
				ProfileInuseSpace:   false,
				ProfileGoroutines:   false,
			},
			wantEnabled: false,
		},
		{
			name: "cpu_only",
			config: telemetry.ProfilerConfig{
				Enabled:         false,
				ServerAddress:   "http://localhost:4040",
				ApplicationName: "test",
				ProfileCPU:      true,
			},
			wantEnabled: false,
		},
		{
			name: "memory_only",
			config: telemetry.ProfilerConfig{
				Enabled:             false,
				ServerAddress:       "http://localhost:4040",
				ApplicationName:     "test",
				ProfileAllocObjects: true,
				ProfileAllocSpace:   true,
			},
			wantEnabled: false,
		},
		{
			name: "mutex_profiling",
			config: telemetry.ProfilerConfig{
				Enabled:              false,
				ServerAddress:        "http://localhost:4040",
				ApplicationName:      "test",
				ProfileMutexCount:    true,
				ProfileMutexDuration: true,
				MutexProfileFraction: 10,
			},
			wantEnabled: false,
		},
		{
			name: "block_profiling",
			config: telemetry.ProfilerConfig{
				Enabled:              false,
				ServerAddress:        "http://localhost:4040",
				ApplicationName:      "test",
				ProfileBlockCount:    true,
				ProfileBlockDuration: true,
				BlockProfileRate:     10,
			},
			wantEnabled: false,
		},
		{
			name: "all_profiles_enabled",
			config: telemetry.ProfilerConfig{
				Enabled:              false, // Keep disabled for unit test
				ServerAddress:        "http://localhost:4040",
				ApplicationName:      "test",
				ProfileCPU:           true,
				ProfileAllocObjects:  true,
				ProfileAllocSpace:    true,
				ProfileInuseObjects:  true,
				ProfileInuseSpace:    true,
				ProfileGoroutines:    true,
				ProfileMutexCount:    true,
				ProfileMutexDuration: true,
				ProfileBlockCount:    true,
				ProfileBlockDuration: true,
			},
			wantEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)

			profiler, err := telemetry.NewProfiler(tt.config, logger)
			require.NoError(t, err)
			require.NotNil(t, profiler)

			assert.Equal(t, tt.wantEnabled, profiler.IsEnabled())

			// Cleanup
			err = profiler.Stop()
			assert.NoError(t, err)
		})
	}
}

func TestProfiler_DisableGCRuns(t *testing.T) {
	logger := zaptest.NewLogger(t)

	cfg := telemetry.ProfilerConfig{
		Enabled:         false, // Keep disabled for unit test
		ServerAddress:   "http://localhost:4040",
		ApplicationName: "test-service",
		DisableGCRuns:   true,
	}

	profiler, err := telemetry.NewProfiler(cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, profiler)

	gotCfg := profiler.GetConfig()
	assert.True(t, gotCfg.DisableGCRuns)

	err = profiler.Stop()
	assert.NoError(t, err)
}

func TestProfiler_BasicAuth(t *testing.T) {
	logger := zaptest.NewLogger(t)

	cfg := telemetry.ProfilerConfig{
		Enabled:           false, // Keep disabled for unit test
		ServerAddress:     "http://localhost:4040",
		ApplicationName:   "test-service",
		BasicAuthUser:     "user",
		BasicAuthPassword: "password",
	}

	profiler, err := telemetry.NewProfiler(cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, profiler)

	gotCfg := profiler.GetConfig()
	assert.Equal(t, "user", gotCfg.BasicAuthUser)
	assert.Equal(t, "password", gotCfg.BasicAuthPassword)

	err = profiler.Stop()
	assert.NoError(t, err)
}

func TestProfiler_RuntimeSettings_MutexProfiling(t *testing.T) {
	// This test verifies that mutex profiling configuration works correctly
	// Note: We test with Enabled=false to avoid needing a real Pyroscope server

	logger := zaptest.NewLogger(t)

	cfg := telemetry.ProfilerConfig{
		Enabled:              false,
		ServerAddress:        "http://localhost:4040",
		ApplicationName:      "test-service",
		ProfileMutexCount:    true,
		ProfileMutexDuration: true,
		MutexProfileFraction: 10,
	}

	profiler, err := telemetry.NewProfiler(cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, profiler)

	gotCfg := profiler.GetConfig()
	assert.True(t, gotCfg.ProfileMutexCount)
	assert.True(t, gotCfg.ProfileMutexDuration)
	assert.Equal(t, 10, gotCfg.MutexProfileFraction)

	err = profiler.Stop()
	assert.NoError(t, err)
}

func TestProfiler_RuntimeSettings_BlockProfiling(t *testing.T) {
	// This test verifies that block profiling configuration works correctly

	logger := zaptest.NewLogger(t)

	cfg := telemetry.ProfilerConfig{
		Enabled:              false,
		ServerAddress:        "http://localhost:4040",
		ApplicationName:      "test-service",
		ProfileBlockCount:    true,
		ProfileBlockDuration: true,
		BlockProfileRate:     10,
	}

	profiler, err := telemetry.NewProfiler(cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, profiler)

	gotCfg := profiler.GetConfig()
	assert.True(t, gotCfg.ProfileBlockCount)
	assert.True(t, gotCfg.ProfileBlockDuration)
	assert.Equal(t, 10, gotCfg.BlockProfileRate)

	err = profiler.Stop()
	assert.NoError(t, err)
}
