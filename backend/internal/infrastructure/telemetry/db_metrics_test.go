package telemetry

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestDefaultDBMetricsConfig tests the default configuration values.
func TestDefaultDBMetricsConfig(t *testing.T) {
	cfg := DefaultDBMetricsConfig()

	assert.True(t, cfg.Enabled, "Metrics should be enabled by default")
	assert.Equal(t, 200*time.Millisecond, cfg.SlowQueryThreshold, "Slow query threshold should be 200ms")
	assert.Equal(t, 15*time.Second, cfg.PoolStatsInterval, "Pool stats interval should be 15s")
}

// TestNewDBMetrics tests creating a new DBMetrics instance.
func TestNewDBMetrics(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	defer provider.Shutdown(context.Background())

	meter := provider.Meter("test")
	logger := zap.NewNop()

	t.Run("creates metrics successfully", func(t *testing.T) {
		metrics, err := NewDBMetrics(meter, DefaultDBMetricsConfig(), logger)
		require.NoError(t, err)
		require.NotNil(t, metrics)

		assert.NotNil(t, metrics.poolConnections, "poolConnections should be created")
		assert.NotNil(t, metrics.poolConnectionsMax, "poolConnectionsMax should be created")
		assert.NotNil(t, metrics.queryTotal, "queryTotal should be created")
		assert.NotNil(t, metrics.queryDuration, "queryDuration should be created")
		assert.NotNil(t, metrics.slowQueryTotal, "slowQueryTotal should be created")
	})

	t.Run("applies default config values", func(t *testing.T) {
		metrics, err := NewDBMetrics(meter, DBMetricsConfig{}, logger)
		require.NoError(t, err)
		require.NotNil(t, metrics)

		assert.Equal(t, 200*time.Millisecond, metrics.config.SlowQueryThreshold)
		assert.Equal(t, 15*time.Second, metrics.config.PoolStatsInterval)
	})

	t.Run("uses nop logger when nil", func(t *testing.T) {
		metrics, err := NewDBMetrics(meter, DefaultDBMetricsConfig(), nil)
		require.NoError(t, err)
		require.NotNil(t, metrics)
		require.NotNil(t, metrics.logger)
	})
}

// TestDBMetrics_RecordQuery tests recording query metrics.
func TestDBMetrics_RecordQuery(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	defer provider.Shutdown(context.Background())

	meter := provider.Meter("test")
	ctx := context.Background()

	t.Run("records query count and duration", func(t *testing.T) {
		metrics, err := NewDBMetrics(meter, DBMetricsConfig{
			Enabled:            true,
			SlowQueryThreshold: 200 * time.Millisecond,
		}, zap.NewNop())
		require.NoError(t, err)

		// Record a fast query
		metrics.RecordQuery(ctx, "SELECT", "users", 50*time.Millisecond, nil)

		// Collect metrics
		var rm metricdata.ResourceMetrics
		err = reader.Collect(ctx, &rm)
		require.NoError(t, err)

		// Verify query total was recorded
		found := findMetric(rm, "db_query_total")
		assert.True(t, found, "db_query_total metric should be recorded")

		// Verify query duration was recorded
		found = findMetric(rm, "db_query_duration_seconds")
		assert.True(t, found, "db_query_duration_seconds metric should be recorded")
	})

	t.Run("records slow query when threshold exceeded", func(t *testing.T) {
		// Create new reader for isolated test
		reader := sdkmetric.NewManualReader()
		provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
		defer provider.Shutdown(ctx)

		meter := provider.Meter("test_slow")
		metrics, err := NewDBMetrics(meter, DBMetricsConfig{
			Enabled:            true,
			SlowQueryThreshold: 100 * time.Millisecond,
		}, zap.NewNop())
		require.NoError(t, err)

		// Record a slow query (exceeds 100ms threshold)
		metrics.RecordQuery(ctx, "SELECT", "orders", 250*time.Millisecond, nil)

		// Collect metrics
		var rm metricdata.ResourceMetrics
		err = reader.Collect(ctx, &rm)
		require.NoError(t, err)

		// Verify slow query was recorded
		found := findMetric(rm, "db_slow_query_total")
		assert.True(t, found, "db_slow_query_total metric should be recorded for slow queries")
	})

	t.Run("does not record slow query when under threshold", func(t *testing.T) {
		// Create new reader for isolated test
		reader := sdkmetric.NewManualReader()
		provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
		defer provider.Shutdown(ctx)

		meter := provider.Meter("test_fast")
		metrics, err := NewDBMetrics(meter, DBMetricsConfig{
			Enabled:            true,
			SlowQueryThreshold: 200 * time.Millisecond,
		}, zap.NewNop())
		require.NoError(t, err)

		// Record a fast query (under 200ms threshold)
		metrics.RecordQuery(ctx, "SELECT", "products", 50*time.Millisecond, nil)

		// Collect metrics
		var rm metricdata.ResourceMetrics
		err = reader.Collect(ctx, &rm)
		require.NoError(t, err)

		// Slow query total should exist but have value 0
		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				if m.Name == "db_slow_query_total" {
					sum := m.Data.(metricdata.Sum[int64])
					for _, dp := range sum.DataPoints {
						assert.Equal(t, int64(0), dp.Value, "slow query count should be 0 for fast queries")
					}
				}
			}
		}
	})

	t.Run("normalizes operation to uppercase", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
		defer provider.Shutdown(ctx)

		meter := provider.Meter("test_ops")
		metrics, err := NewDBMetrics(meter, DefaultDBMetricsConfig(), zap.NewNop())
		require.NoError(t, err)

		// Record queries with different case operations
		metrics.RecordQuery(ctx, "select", "users", 10*time.Millisecond, nil)
		metrics.RecordQuery(ctx, "Insert", "users", 10*time.Millisecond, nil)
		metrics.RecordQuery(ctx, "UPDATE", "users", 10*time.Millisecond, nil)

		// All should be recorded successfully (just verify no panic)
		var rm metricdata.ResourceMetrics
		err = reader.Collect(ctx, &rm)
		require.NoError(t, err)

		found := findMetric(rm, "db_query_total")
		assert.True(t, found, "queries should be recorded with normalized operations")
	})

	t.Run("handles empty operation", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
		defer provider.Shutdown(ctx)

		meter := provider.Meter("test_empty_op")
		metrics, err := NewDBMetrics(meter, DefaultDBMetricsConfig(), zap.NewNop())
		require.NoError(t, err)

		// Record query with empty operation
		metrics.RecordQuery(ctx, "", "users", 10*time.Millisecond, nil)

		// Should be recorded as "UNKNOWN"
		var rm metricdata.ResourceMetrics
		err = reader.Collect(ctx, &rm)
		require.NoError(t, err)
	})

	t.Run("handles empty table name for slow queries", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
		defer provider.Shutdown(ctx)

		meter := provider.Meter("test_empty_table")
		metrics, err := NewDBMetrics(meter, DBMetricsConfig{
			Enabled:            true,
			SlowQueryThreshold: 50 * time.Millisecond,
		}, zap.NewNop())
		require.NoError(t, err)

		// Record slow query with empty table
		metrics.RecordQuery(ctx, "SELECT", "", 100*time.Millisecond, nil)

		// Should be recorded with "unknown" table
		var rm metricdata.ResourceMetrics
		err = reader.Collect(ctx, &rm)
		require.NoError(t, err)

		found := findMetric(rm, "db_slow_query_total")
		assert.True(t, found, "slow query should be recorded even with empty table")
	})
}

// TestDBMetrics_PoolStats tests connection pool statistics collection.
func TestDBMetrics_PoolStats(t *testing.T) {
	ctx := context.Background()

	t.Run("collects pool stats periodically", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
		defer provider.Shutdown(ctx)

		meter := provider.Meter("test_pool")

		// Create mock SQL DB
		mockDB, _, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		metrics, err := NewDBMetrics(meter, DBMetricsConfig{
			Enabled:           true,
			PoolStatsInterval: 50 * time.Millisecond, // Short interval for test
		}, zap.NewNop())
		require.NoError(t, err)

		metrics.SetSQLDB(mockDB)

		// Start collection
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		metrics.StartPoolStatsCollection(ctx)

		// Wait for at least one collection
		time.Sleep(100 * time.Millisecond)

		// Stop collection
		metrics.Stop()

		// Collect metrics
		var rm metricdata.ResourceMetrics
		err = reader.Collect(context.Background(), &rm)
		require.NoError(t, err)

		// Verify pool metrics were recorded
		foundMax := findMetric(rm, "db_pool_connections_max")
		foundPool := findMetric(rm, "db_pool_connections")

		assert.True(t, foundMax, "db_pool_connections_max metric should be recorded")
		assert.True(t, foundPool, "db_pool_connections metric should be recorded")
	})

	t.Run("does nothing when sqlDB not set", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
		defer provider.Shutdown(ctx)

		meter := provider.Meter("test_no_db")
		metrics, err := NewDBMetrics(meter, DefaultDBMetricsConfig(), zap.NewNop())
		require.NoError(t, err)

		// Don't set sqlDB
		// Start collection should log warning and return
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		metrics.StartPoolStatsCollection(ctx)
		time.Sleep(50 * time.Millisecond)
		metrics.Stop()

		// Should not panic and no metrics recorded
	})

	t.Run("stops on context cancellation", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
		defer provider.Shutdown(ctx)

		meter := provider.Meter("test_ctx_cancel")

		mockDB, _, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		metrics, err := NewDBMetrics(meter, DBMetricsConfig{
			Enabled:           true,
			PoolStatsInterval: 1 * time.Second,
		}, zap.NewNop())
		require.NoError(t, err)

		metrics.SetSQLDB(mockDB)

		ctx, cancel := context.WithCancel(ctx)

		metrics.StartPoolStatsCollection(ctx)

		// Cancel immediately
		cancel()

		// Should stop gracefully
		metrics.Stop()
	})
}

// TestDBMetrics_Stop tests the Stop function.
func TestDBMetrics_Stop(t *testing.T) {
	ctx := context.Background()

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	defer provider.Shutdown(ctx)

	meter := provider.Meter("test_stop")

	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	metrics, err := NewDBMetrics(meter, DBMetricsConfig{
		Enabled:           true,
		PoolStatsInterval: 100 * time.Millisecond,
	}, zap.NewNop())
	require.NoError(t, err)

	metrics.SetSQLDB(mockDB)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	metrics.StartPoolStatsCollection(ctx)

	// Stop should complete without blocking
	done := make(chan struct{})
	go func() {
		metrics.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() blocked for too long")
	}
}

// TestDBMetrics_StopIdempotent tests that Stop() can be called multiple times safely.
func TestDBMetrics_StopIdempotent(t *testing.T) {
	ctx := context.Background()

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	defer provider.Shutdown(ctx)

	meter := provider.Meter("test_stop_idempotent")

	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	metrics, err := NewDBMetrics(meter, DBMetricsConfig{
		Enabled:           true,
		PoolStatsInterval: 100 * time.Millisecond,
	}, zap.NewNop())
	require.NoError(t, err)

	metrics.SetSQLDB(mockDB)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	metrics.StartPoolStatsCollection(ctx)

	// First stop
	metrics.Stop()

	// Second stop should not panic
	assert.NotPanics(t, func() {
		metrics.Stop()
	})

	// Third stop should also not panic
	assert.NotPanics(t, func() {
		metrics.Stop()
	})
}

// TestDBMetricsPlugin tests the GORM plugin functionality.
func TestDBMetricsPlugin(t *testing.T) {
	ctx := context.Background()

	t.Run("plugin name is correct", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
		defer provider.Shutdown(ctx)

		meter := provider.Meter("test")
		metrics, err := NewDBMetrics(meter, DefaultDBMetricsConfig(), zap.NewNop())
		require.NoError(t, err)

		plugin := NewDBMetricsPlugin(metrics, zap.NewNop())
		assert.Equal(t, "db_metrics", plugin.Name())
	})

	t.Run("initializes with gorm db", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
		defer provider.Shutdown(ctx)

		meter := provider.Meter("test")
		metrics, err := NewDBMetrics(meter, DefaultDBMetricsConfig(), zap.NewNop())
		require.NoError(t, err)

		plugin := NewDBMetricsPlugin(metrics, zap.NewNop())

		// Create mock GORM DB
		mockDB, _, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		gormDB, err := gorm.Open(postgres.New(postgres.Config{
			Conn: mockDB,
		}), &gorm.Config{})
		require.NoError(t, err)

		// Initialize plugin
		err = plugin.Initialize(gormDB)
		require.NoError(t, err)
	})
}

// TestDetectOperationType tests SQL operation type detection.
func TestDetectOperationType(t *testing.T) {
	tests := []struct {
		sql      string
		expected string
	}{
		{"SELECT * FROM users", "SELECT"},
		{"select id from users", "SELECT"},
		{"  SELECT id FROM users", "SELECT"},
		{"INSERT INTO users (name) VALUES ('test')", "INSERT"},
		{"insert into users values (1)", "INSERT"},
		{"UPDATE users SET name = 'test'", "UPDATE"},
		{"update users set name = 'test'", "UPDATE"},
		{"DELETE FROM users WHERE id = 1", "DELETE"},
		{"delete from users", "DELETE"},
		{"CREATE TABLE users", "OTHER"},
		{"DROP TABLE users", "OTHER"},
		{"", "OTHER"},
		{"TRUNCATE TABLE users", "OTHER"},
	}

	for _, tc := range tests {
		t.Run(tc.sql, func(t *testing.T) {
			result := detectOperationType(tc.sql)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestRegisterDBMetrics tests the convenience registration function.
func TestRegisterDBMetrics(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	t.Run("returns nil when disabled", func(t *testing.T) {
		mockDB, _, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		gormDB, err := gorm.Open(postgres.New(postgres.Config{
			Conn: mockDB,
		}), &gorm.Config{})
		require.NoError(t, err)

		metrics, err := RegisterDBMetrics(gormDB, nil, DBMetricsConfig{
			Enabled: false,
		}, logger)

		require.NoError(t, err)
		assert.Nil(t, metrics, "should return nil when disabled")
	})

	t.Run("returns nil when meter provider is nil", func(t *testing.T) {
		mockDB, _, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		gormDB, err := gorm.Open(postgres.New(postgres.Config{
			Conn: mockDB,
		}), &gorm.Config{})
		require.NoError(t, err)

		metrics, err := RegisterDBMetrics(gormDB, nil, DBMetricsConfig{
			Enabled: true,
		}, logger)

		require.NoError(t, err)
		assert.Nil(t, metrics, "should return nil when meter provider is nil")
	})

	t.Run("registers metrics when enabled", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		sdkProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
		defer sdkProvider.Shutdown(ctx)

		// Create our custom MeterProvider wrapper
		mp := &MeterProvider{
			provider: sdkProvider,
			logger:   logger,
			config:   MetricsConfig{Enabled: true},
		}

		mockDB, _, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		gormDB, err := gorm.Open(postgres.New(postgres.Config{
			Conn: mockDB,
		}), &gorm.Config{})
		require.NoError(t, err)

		metrics, err := RegisterDBMetrics(gormDB, mp, DBMetricsConfig{
			Enabled:            true,
			SlowQueryThreshold: 200 * time.Millisecond,
			PoolStatsInterval:  15 * time.Second,
		}, logger)

		require.NoError(t, err)
		require.NotNil(t, metrics)
	})
}

// TestDBMetrics_ConcurrentRecordQuery tests concurrent query recording.
func TestDBMetrics_ConcurrentRecordQuery(t *testing.T) {
	ctx := context.Background()

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	defer provider.Shutdown(ctx)

	meter := provider.Meter("test_concurrent")
	metrics, err := NewDBMetrics(meter, DefaultDBMetricsConfig(), zap.NewNop())
	require.NoError(t, err)

	// Run concurrent recordings
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			operation := []string{"SELECT", "INSERT", "UPDATE", "DELETE"}[i%4]
			table := []string{"users", "orders", "products", "inventory"}[i%4]
			duration := time.Duration(i) * time.Millisecond
			metrics.RecordQuery(ctx, operation, table, duration, nil)
		}(i)
	}
	wg.Wait()

	// Collect and verify metrics were recorded
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)

	found := findMetric(rm, "db_query_total")
	assert.True(t, found, "queries should be recorded concurrently without race conditions")
}

// Helper function to find a metric by name in ResourceMetrics.
func findMetric(rm metricdata.ResourceMetrics, name string) bool {
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				return true
			}
		}
	}
	return false
}

// TestDBMetrics_WithMeter verifies metrics are created with the provided meter.
func TestDBMetrics_WithMeter(t *testing.T) {
	ctx := context.Background()

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	defer provider.Shutdown(ctx)

	meter := provider.Meter("custom.db.meter")
	metrics, err := NewDBMetrics(meter, DefaultDBMetricsConfig(), zap.NewNop())
	require.NoError(t, err)

	// Record a query to ensure meter is used
	metrics.RecordQuery(ctx, "SELECT", "test", 10*time.Millisecond, nil)

	// Collect metrics
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)

	// Verify scope is from our custom meter
	for _, sm := range rm.ScopeMetrics {
		if sm.Scope.Name == "custom.db.meter" {
			assert.True(t, len(sm.Metrics) > 0, "metrics should be registered under our custom meter")
			return
		}
	}
	t.Error("metrics not found under custom meter scope")
}

// Ensure metric.Meter interface is satisfied by checking at compile time.
var _ metric.Meter = (metric.Meter)(nil)
