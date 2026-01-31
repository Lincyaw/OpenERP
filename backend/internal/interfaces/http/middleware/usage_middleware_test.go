package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/billing"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockUsageRecordRepository is a mock implementation of billing.UsageRecordRepository
type MockUsageRecordRepository struct {
	mock.Mock
	mu           sync.Mutex
	savedRecords []*billing.UsageRecord
}

func NewMockUsageRecordRepository() *MockUsageRecordRepository {
	return &MockUsageRecordRepository{
		savedRecords: make([]*billing.UsageRecord, 0),
	}
}

func (m *MockUsageRecordRepository) Save(ctx context.Context, record *billing.UsageRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.savedRecords = append(m.savedRecords, record)
	args := m.Called(ctx, record)
	return args.Error(0)
}

func (m *MockUsageRecordRepository) SaveBatch(ctx context.Context, records []*billing.UsageRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.savedRecords = append(m.savedRecords, records...)
	args := m.Called(ctx, records)
	return args.Error(0)
}

func (m *MockUsageRecordRepository) FindByID(ctx context.Context, id uuid.UUID) (*billing.UsageRecord, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*billing.UsageRecord), args.Error(1)
}

func (m *MockUsageRecordRepository) FindByTenant(ctx context.Context, tenantID uuid.UUID, filter billing.UsageRecordFilter) ([]*billing.UsageRecord, error) {
	args := m.Called(ctx, tenantID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*billing.UsageRecord), args.Error(1)
}

func (m *MockUsageRecordRepository) FindByTenantAndType(ctx context.Context, tenantID uuid.UUID, usageType billing.UsageType, filter billing.UsageRecordFilter) ([]*billing.UsageRecord, error) {
	args := m.Called(ctx, tenantID, usageType, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*billing.UsageRecord), args.Error(1)
}

func (m *MockUsageRecordRepository) CountByTenant(ctx context.Context, tenantID uuid.UUID, filter billing.UsageRecordFilter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUsageRecordRepository) SumByTenantAndType(ctx context.Context, tenantID uuid.UUID, usageType billing.UsageType, start, end time.Time) (int64, error) {
	args := m.Called(ctx, tenantID, usageType, start, end)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUsageRecordRepository) GetAggregatedUsage(ctx context.Context, tenantID uuid.UUID, usageType billing.UsageType, start, end time.Time, groupBy billing.AggregationPeriod) ([]billing.UsageAggregation, error) {
	args := m.Called(ctx, tenantID, usageType, start, end, groupBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]billing.UsageAggregation), args.Error(1)
}

func (m *MockUsageRecordRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	args := m.Called(ctx, before)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUsageRecordRepository) GetSavedRecords() []*billing.UsageRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*billing.UsageRecord, len(m.savedRecords))
	copy(result, m.savedRecords)
	return result
}

func (m *MockUsageRecordRepository) ClearSavedRecords() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.savedRecords = m.savedRecords[:0]
}

// Test helper functions
func setupUsageTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func createTestTenantID() uuid.UUID {
	return uuid.MustParse("11111111-1111-1111-1111-111111111111")
}

func createTestUserID() uuid.UUID {
	return uuid.MustParse("22222222-2222-2222-2222-222222222222")
}

// TestDefaultUsageTrackerConfig tests the default configuration
func TestDefaultUsageTrackerConfig(t *testing.T) {
	cfg := DefaultUsageTrackerConfig()

	assert.True(t, cfg.Enabled)
	assert.Equal(t, 10000, cfg.BufferSize)
	assert.Equal(t, 100, cfg.BatchSize)
	assert.Equal(t, 5*time.Second, cfg.FlushInterval)
	assert.NotEmpty(t, cfg.SkipPaths)
	assert.Contains(t, cfg.SkipPaths, "/health")
	assert.Contains(t, cfg.SkipPaths, "/metrics")
}

// TestNewUsageTracker tests creating a new usage tracker
func TestNewUsageTracker(t *testing.T) {
	repo := NewMockUsageRecordRepository()
	cfg := DefaultUsageTrackerConfig()
	cfg.Logger = zap.NewNop()

	tracker, err := NewUsageTracker(cfg, repo)

	require.NoError(t, err)
	assert.NotNil(t, tracker)
	assert.False(t, tracker.IsRunning())
}

// TestUsageTrackerStartStop tests starting and stopping the tracker
func TestUsageTrackerStartStop(t *testing.T) {
	repo := NewMockUsageRecordRepository()
	repo.On("SaveBatch", mock.Anything, mock.Anything).Return(nil)

	cfg := DefaultUsageTrackerConfig()
	cfg.Logger = zap.NewNop()
	cfg.FlushInterval = 100 * time.Millisecond

	tracker, err := NewUsageTracker(cfg, repo)
	require.NoError(t, err)

	// Start the tracker
	tracker.Start()
	assert.True(t, tracker.IsRunning())

	// Starting again should be a no-op
	tracker.Start()
	assert.True(t, tracker.IsRunning())

	// Stop the tracker
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = tracker.Stop(ctx)
	assert.NoError(t, err)
	assert.False(t, tracker.IsRunning())

	// Stopping again should be a no-op
	err = tracker.Stop(ctx)
	assert.NoError(t, err)
}

// TestUsageTrackerTrack tests tracking usage records
func TestUsageTrackerTrack(t *testing.T) {
	repo := NewMockUsageRecordRepository()
	repo.On("SaveBatch", mock.Anything, mock.Anything).Return(nil)

	cfg := DefaultUsageTrackerConfig()
	cfg.Logger = zap.NewNop()
	cfg.FlushInterval = 100 * time.Millisecond
	cfg.BatchSize = 5

	tracker, err := NewUsageTracker(cfg, repo)
	require.NoError(t, err)

	tracker.Start()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tracker.Stop(ctx)
	}()

	tenantID := createTestTenantID()

	// Track some records
	for i := 0; i < 10; i++ {
		record, err := billing.NewUsageRecordSimple(tenantID, billing.UsageTypeAPICalls, 1)
		require.NoError(t, err)
		assert.True(t, tracker.Track(record))
	}

	// Wait for batch to be written
	time.Sleep(300 * time.Millisecond)

	// Verify records were saved
	savedRecords := repo.GetSavedRecords()
	assert.GreaterOrEqual(t, len(savedRecords), 5)
}

// TestUsageTrackerBufferFull tests behavior when buffer is full
func TestUsageTrackerBufferFull(t *testing.T) {
	repo := NewMockUsageRecordRepository()
	// Don't set up SaveBatch expectation - we want the buffer to fill up

	cfg := DefaultUsageTrackerConfig()
	cfg.Logger = zap.NewNop()
	cfg.BufferSize = 5
	cfg.FlushInterval = 10 * time.Second // Long interval to prevent flushing
	cfg.BatchSize = 100                  // Large batch size to prevent flushing

	tracker, err := NewUsageTracker(cfg, repo)
	require.NoError(t, err)

	tracker.Start()
	defer func() {
		repo.On("SaveBatch", mock.Anything, mock.Anything).Return(nil)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tracker.Stop(ctx)
	}()

	tenantID := createTestTenantID()

	// Fill the buffer
	for i := 0; i < 5; i++ {
		record, _ := billing.NewUsageRecordSimple(tenantID, billing.UsageTypeAPICalls, 1)
		tracker.Track(record)
	}

	// Next track should fail (buffer full)
	record, _ := billing.NewUsageRecordSimple(tenantID, billing.UsageTypeAPICalls, 1)
	assert.False(t, tracker.Track(record))
}

// TestUsageTrackerStats tests getting tracker statistics
func TestUsageTrackerStats(t *testing.T) {
	repo := NewMockUsageRecordRepository()
	repo.On("SaveBatch", mock.Anything, mock.Anything).Return(nil)

	cfg := DefaultUsageTrackerConfig()
	cfg.Logger = zap.NewNop()
	cfg.BufferSize = 100

	tracker, err := NewUsageTracker(cfg, repo)
	require.NoError(t, err)

	stats := tracker.Stats()
	assert.Equal(t, 0, stats.BufferSize)
	assert.Equal(t, 100, stats.BufferCapacity)
	assert.Equal(t, 0.0, stats.BufferUsage)
	assert.False(t, stats.Running)

	tracker.Start()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tracker.Stop(ctx)
	}()

	stats = tracker.Stats()
	assert.True(t, stats.Running)
}

// TestTrackAPIUsageMiddleware tests the TrackAPIUsage middleware
func TestTrackAPIUsageMiddleware(t *testing.T) {
	repo := NewMockUsageRecordRepository()
	repo.On("SaveBatch", mock.Anything, mock.Anything).Return(nil)

	cfg := DefaultUsageTrackerConfig()
	cfg.Logger = zap.NewNop()
	cfg.FlushInterval = 100 * time.Millisecond
	cfg.BatchSize = 1

	tracker, err := NewUsageTracker(cfg, repo)
	require.NoError(t, err)

	tracker.Start()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tracker.Stop(ctx)
	}()

	router := setupUsageTestRouter()
	tenantID := createTestTenantID()

	// Add tenant middleware simulation
	router.Use(func(c *gin.Context) {
		c.Set(TenantIDKey, tenantID.String())
		c.Next()
	})

	// Add usage tracking middleware
	router.Use(TrackAPIUsage(tracker))

	// Add test endpoint
	router.GET("/api/v1/products", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Wait for batch to be written
	time.Sleep(300 * time.Millisecond)

	// Verify usage was tracked
	savedRecords := repo.GetSavedRecords()
	require.GreaterOrEqual(t, len(savedRecords), 1)

	record := savedRecords[0]
	assert.Equal(t, tenantID, record.TenantID)
	assert.Equal(t, billing.UsageTypeAPICalls, record.UsageType)
	assert.Equal(t, int64(1), record.Quantity)
}

// TestTrackAPIUsageSkipPaths tests that skip paths are not tracked
func TestTrackAPIUsageSkipPaths(t *testing.T) {
	repo := NewMockUsageRecordRepository()
	repo.On("SaveBatch", mock.Anything, mock.Anything).Return(nil)

	cfg := DefaultUsageTrackerConfig()
	cfg.Logger = zap.NewNop()
	cfg.FlushInterval = 100 * time.Millisecond

	tracker, err := NewUsageTracker(cfg, repo)
	require.NoError(t, err)

	tracker.Start()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tracker.Stop(ctx)
	}()

	router := setupUsageTestRouter()
	tenantID := createTestTenantID()

	router.Use(func(c *gin.Context) {
		c.Set(TenantIDKey, tenantID.String())
		c.Next()
	})
	router.Use(TrackAPIUsage(tracker))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})
	router.GET("/metrics", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"metrics": "data"})
	})

	// Make requests to skip paths
	for _, path := range []string{"/health", "/metrics"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Wait a bit
	time.Sleep(200 * time.Millisecond)

	// Verify no usage was tracked
	savedRecords := repo.GetSavedRecords()
	assert.Empty(t, savedRecords)
}

// TestTrackAPIUsageNoTenant tests that requests without tenant are not tracked
func TestTrackAPIUsageNoTenant(t *testing.T) {
	repo := NewMockUsageRecordRepository()
	repo.On("SaveBatch", mock.Anything, mock.Anything).Return(nil)

	cfg := DefaultUsageTrackerConfig()
	cfg.Logger = zap.NewNop()
	cfg.FlushInterval = 100 * time.Millisecond

	tracker, err := NewUsageTracker(cfg, repo)
	require.NoError(t, err)

	tracker.Start()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tracker.Stop(ctx)
	}()

	router := setupUsageTestRouter()
	// No tenant middleware - simulating unauthenticated request
	router.Use(TrackAPIUsage(tracker))

	router.GET("/api/v1/products", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Wait a bit
	time.Sleep(200 * time.Millisecond)

	// Verify no usage was tracked (no tenant)
	savedRecords := repo.GetSavedRecords()
	assert.Empty(t, savedRecords)
}

// TestTrackAPIUsageNilTracker tests that nil tracker returns no-op middleware
func TestTrackAPIUsageNilTracker(t *testing.T) {
	router := setupUsageTestRouter()

	// Should not panic with nil tracker
	router.Use(TrackAPIUsage(nil))

	router.GET("/api/v1/products", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestTrackAPIUsageDisabled tests that disabled tracker returns no-op middleware
func TestTrackAPIUsageDisabled(t *testing.T) {
	repo := NewMockUsageRecordRepository()

	cfg := DefaultUsageTrackerConfig()
	cfg.Enabled = false
	cfg.Logger = zap.NewNop()

	tracker, err := NewUsageTracker(cfg, repo)
	require.NoError(t, err)

	router := setupUsageTestRouter()
	router.Use(TrackAPIUsage(tracker))

	router.GET("/api/v1/products", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestTrackResourceCreationMiddleware tests the TrackResourceCreation middleware
func TestTrackResourceCreationMiddleware(t *testing.T) {
	repo := NewMockUsageRecordRepository()
	repo.On("SaveBatch", mock.Anything, mock.Anything).Return(nil)

	cfg := DefaultUsageTrackerConfig()
	cfg.Logger = zap.NewNop()
	cfg.FlushInterval = 100 * time.Millisecond
	cfg.BatchSize = 1

	tracker, err := NewUsageTracker(cfg, repo)
	require.NoError(t, err)

	tracker.Start()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tracker.Stop(ctx)
	}()

	router := setupUsageTestRouter()
	tenantID := createTestTenantID()

	router.Use(func(c *gin.Context) {
		c.Set(TenantIDKey, tenantID.String())
		c.Next()
	})

	// Add resource creation tracking for sales orders
	router.POST("/api/v1/sales-orders", TrackResourceCreation(tracker, ResourceTypeSalesOrder), func(c *gin.Context) {
		// Simulate successful order creation
		orderID := uuid.New().String()
		SetResourceCreationContext(c, &ResourceCreationContext{
			ResourceType: ResourceTypeSalesOrder,
			ResourceID:   orderID,
			Metadata: map[string]any{
				"customer_id": "cust-123",
			},
		})
		c.JSON(http.StatusCreated, gin.H{"id": orderID})
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sales-orders", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	// Wait for batch to be written
	time.Sleep(300 * time.Millisecond)

	// Verify usage was tracked
	savedRecords := repo.GetSavedRecords()
	require.GreaterOrEqual(t, len(savedRecords), 1)

	record := savedRecords[0]
	assert.Equal(t, tenantID, record.TenantID)
	assert.Equal(t, billing.UsageTypeOrdersCreated, record.UsageType)
	assert.Equal(t, int64(1), record.Quantity)
}

// TestTrackResourceCreationFailedRequest tests that failed requests are not tracked
func TestTrackResourceCreationFailedRequest(t *testing.T) {
	repo := NewMockUsageRecordRepository()
	repo.On("SaveBatch", mock.Anything, mock.Anything).Return(nil)

	cfg := DefaultUsageTrackerConfig()
	cfg.Logger = zap.NewNop()
	cfg.FlushInterval = 100 * time.Millisecond

	tracker, err := NewUsageTracker(cfg, repo)
	require.NoError(t, err)

	tracker.Start()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tracker.Stop(ctx)
	}()

	router := setupUsageTestRouter()
	tenantID := createTestTenantID()

	router.Use(func(c *gin.Context) {
		c.Set(TenantIDKey, tenantID.String())
		c.Next()
	})

	router.POST("/api/v1/sales-orders", TrackResourceCreation(tracker, ResourceTypeSalesOrder), func(c *gin.Context) {
		// Simulate failed order creation
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation failed"})
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sales-orders", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Wait a bit
	time.Sleep(200 * time.Millisecond)

	// Verify no usage was tracked (failed request)
	savedRecords := repo.GetSavedRecords()
	assert.Empty(t, savedRecords)
}

// TestResourceTypeToUsageType tests the ResourceType to UsageType conversion
func TestResourceTypeToUsageType(t *testing.T) {
	tests := []struct {
		resourceType ResourceType
		expected     billing.UsageType
	}{
		{ResourceTypeSalesOrder, billing.UsageTypeOrdersCreated},
		{ResourceTypePurchaseOrder, billing.UsageTypeOrdersCreated},
		{ResourceTypeProduct, billing.UsageTypeProductsSKU},
		{ResourceTypeCustomer, billing.UsageTypeCustomers},
		{ResourceTypeSupplier, billing.UsageTypeSuppliers},
		{ResourceTypeWarehouse, billing.UsageTypeWarehouses},
		{ResourceTypeReport, billing.UsageTypeReportsGenerated},
		{ResourceTypeDataExport, billing.UsageTypeDataExports},
		{ResourceTypeDataImport, billing.UsageTypeDataImportRows},
	}

	for _, tt := range tests {
		t.Run(string(tt.resourceType), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.resourceType.ToUsageType())
		})
	}
}

// TestTrackDataImport tests the TrackDataImport helper middleware
func TestTrackDataImport(t *testing.T) {
	repo := NewMockUsageRecordRepository()
	repo.On("SaveBatch", mock.Anything, mock.Anything).Return(nil)

	cfg := DefaultUsageTrackerConfig()
	cfg.Logger = zap.NewNop()
	cfg.FlushInterval = 100 * time.Millisecond
	cfg.BatchSize = 1

	tracker, err := NewUsageTracker(cfg, repo)
	require.NoError(t, err)

	tracker.Start()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tracker.Stop(ctx)
	}()

	router := setupUsageTestRouter()
	tenantID := createTestTenantID()

	router.Use(func(c *gin.Context) {
		c.Set(TenantIDKey, tenantID.String())
		c.Next()
	})

	router.POST("/api/v1/import", TrackDataImport(tracker), func(c *gin.Context) {
		// Simulate importing 500 rows
		SetResourceCreationContext(c, &ResourceCreationContext{
			ResourceType: ResourceTypeDataImport,
			Quantity:     500,
			Metadata: map[string]any{
				"file_name": "products.csv",
			},
		})
		c.JSON(http.StatusOK, gin.H{"imported": 500})
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/import", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Wait for batch to be written
	time.Sleep(300 * time.Millisecond)

	// Verify usage was tracked with correct quantity
	savedRecords := repo.GetSavedRecords()
	require.GreaterOrEqual(t, len(savedRecords), 1)

	record := savedRecords[0]
	assert.Equal(t, billing.UsageTypeDataImportRows, record.UsageType)
	assert.Equal(t, int64(500), record.Quantity)
}

// TestSetResourceCreationContext tests the helper function
func TestSetResourceCreationContext(t *testing.T) {
	router := setupUsageTestRouter()

	var capturedCtx *ResourceCreationContext

	router.POST("/test", func(c *gin.Context) {
		SetResourceCreationContext(c, &ResourceCreationContext{
			ResourceType: ResourceTypeSalesOrder,
			ResourceID:   "order-123",
			Quantity:     1,
			Metadata: map[string]any{
				"key": "value",
			},
		})

		// Retrieve and verify
		if ctx, exists := c.Get(ResourceCreationContextKey); exists {
			capturedCtx = ctx.(*ResourceCreationContext)
		}

		c.JSON(http.StatusOK, gin.H{})
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.NotNil(t, capturedCtx)
	assert.Equal(t, ResourceTypeSalesOrder, capturedCtx.ResourceType)
	assert.Equal(t, "order-123", capturedCtx.ResourceID)
	assert.Equal(t, int64(1), capturedCtx.Quantity)
	assert.Equal(t, "value", capturedCtx.Metadata["key"])
}

// TestTrackAPIUsageWithUserID tests that user ID is captured when available
func TestTrackAPIUsageWithUserID(t *testing.T) {
	repo := NewMockUsageRecordRepository()
	repo.On("SaveBatch", mock.Anything, mock.Anything).Return(nil)

	cfg := DefaultUsageTrackerConfig()
	cfg.Logger = zap.NewNop()
	cfg.FlushInterval = 100 * time.Millisecond
	cfg.BatchSize = 1

	tracker, err := NewUsageTracker(cfg, repo)
	require.NoError(t, err)

	tracker.Start()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tracker.Stop(ctx)
	}()

	router := setupUsageTestRouter()
	tenantID := createTestTenantID()
	userID := createTestUserID()

	router.Use(func(c *gin.Context) {
		c.Set(TenantIDKey, tenantID.String())
		c.Set(JWTUserIDKey, userID.String())
		c.Next()
	})
	router.Use(TrackAPIUsage(tracker))

	router.GET("/api/v1/products", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Wait for batch to be written
	time.Sleep(300 * time.Millisecond)

	// Verify user ID was captured
	savedRecords := repo.GetSavedRecords()
	require.GreaterOrEqual(t, len(savedRecords), 1)

	record := savedRecords[0]
	require.NotNil(t, record.UserID)
	assert.Equal(t, userID, *record.UserID)
}

// TestUsageTrackerConcurrentTracking tests concurrent tracking
func TestUsageTrackerConcurrentTracking(t *testing.T) {
	repo := NewMockUsageRecordRepository()
	repo.On("SaveBatch", mock.Anything, mock.Anything).Return(nil)

	cfg := DefaultUsageTrackerConfig()
	cfg.Logger = zap.NewNop()
	cfg.FlushInterval = 100 * time.Millisecond
	cfg.BatchSize = 10
	cfg.BufferSize = 1000

	tracker, err := NewUsageTracker(cfg, repo)
	require.NoError(t, err)

	tracker.Start()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tracker.Stop(ctx)
	}()

	tenantID := createTestTenantID()

	// Track records concurrently
	var wg sync.WaitGroup
	var successCount atomic.Int64

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			record, err := billing.NewUsageRecordSimple(tenantID, billing.UsageTypeAPICalls, 1)
			if err != nil {
				return
			}
			if tracker.Track(record) {
				successCount.Add(1)
			}
		}()
	}

	wg.Wait()

	// All records should have been tracked
	assert.Equal(t, int64(100), successCount.Load())

	// Wait for batches to be written
	time.Sleep(500 * time.Millisecond)

	// Verify records were saved
	savedRecords := repo.GetSavedRecords()
	assert.GreaterOrEqual(t, len(savedRecords), 50) // At least half should be saved
}

// TestTrackingOverheadUnder5ms tests that tracking overhead is minimal
func TestTrackingOverheadUnder5ms(t *testing.T) {
	repo := NewMockUsageRecordRepository()
	repo.On("SaveBatch", mock.Anything, mock.Anything).Return(nil)

	cfg := DefaultUsageTrackerConfig()
	cfg.Logger = zap.NewNop()
	cfg.FlushInterval = 1 * time.Second
	cfg.BatchSize = 1000

	tracker, err := NewUsageTracker(cfg, repo)
	require.NoError(t, err)

	tracker.Start()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tracker.Stop(ctx)
	}()

	router := setupUsageTestRouter()
	tenantID := createTestTenantID()

	router.Use(func(c *gin.Context) {
		c.Set(TenantIDKey, tenantID.String())
		c.Next()
	})
	router.Use(TrackAPIUsage(tracker))

	router.GET("/api/v1/products", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Warm up
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	// Measure overhead
	var totalOverhead time.Duration
	iterations := 100

	for i := 0; i < iterations; i++ {
		start := time.Now()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		totalOverhead += time.Since(start)
	}

	avgOverhead := totalOverhead / time.Duration(iterations)

	// Average overhead should be under 5ms (acceptance criteria)
	// Note: This includes the entire request processing, not just tracking
	// In practice, tracking overhead is much smaller
	assert.Less(t, avgOverhead, 5*time.Millisecond,
		"Average request time should be under 5ms, got %v", avgOverhead)
}
