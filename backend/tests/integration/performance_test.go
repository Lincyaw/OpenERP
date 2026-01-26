// Package integration provides performance and stress tests for the ERP backend.
// This file implements P8-002: Performance stress testing with concurrent and load testing.
//
// Test Categories:
// 1. Concurrent Testing - Multiple goroutines hitting APIs simultaneously
// 2. Load Testing - Sustained load over time with metrics collection
// 3. Bottleneck Identification - Measuring response times to identify slow endpoints
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http/httptest"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	catalogapp "github.com/erp/backend/internal/application/catalog"
	inventoryapp "github.com/erp/backend/internal/application/inventory"
	partnerapp "github.com/erp/backend/internal/application/partner"
	"github.com/erp/backend/internal/domain/catalog"
	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/erp/backend/internal/infrastructure/persistence"
	infrastrategy "github.com/erp/backend/internal/infrastructure/strategy"
	"github.com/erp/backend/internal/interfaces/http/handler"
	"github.com/erp/backend/internal/interfaces/http/router"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Performance Test Configuration ====================

// PerformanceConfig defines the configuration for performance tests
type PerformanceConfig struct {
	// ConcurrentUsers is the number of concurrent goroutines
	ConcurrentUsers int
	// RequestsPerUser is the number of requests each user makes
	RequestsPerUser int
	// TestDuration is the duration for sustained load tests
	TestDuration time.Duration
	// TargetRPS is the target requests per second for load tests
	TargetRPS int
	// MaxResponseTime is the acceptable maximum response time
	MaxResponseTime time.Duration
	// P95ResponseTime is the acceptable 95th percentile response time
	P95ResponseTime time.Duration
	// P99ResponseTime is the acceptable 99th percentile response time
	P99ResponseTime time.Duration
}

// DefaultPerformanceConfig returns default configuration for integration tests
func DefaultPerformanceConfig() PerformanceConfig {
	return PerformanceConfig{
		ConcurrentUsers: 10,
		RequestsPerUser: 50,
		TestDuration:    10 * time.Second,
		TargetRPS:       100,
		MaxResponseTime: 500 * time.Millisecond,
		P95ResponseTime: 200 * time.Millisecond,
		P99ResponseTime: 400 * time.Millisecond,
	}
}

// PerformanceMetrics collects and reports performance metrics
type PerformanceMetrics struct {
	mu              sync.Mutex
	responseTimes   []time.Duration
	successCount    int64
	errorCount      int64
	statusCodes     map[int]int64
	endpointMetrics map[string]*EndpointMetric
	startTime       time.Time
	endTime         time.Time
}

// EndpointMetric tracks metrics for a specific endpoint
type EndpointMetric struct {
	Name          string
	Count         int64
	TotalDuration time.Duration
	MinDuration   time.Duration
	MaxDuration   time.Duration
	ErrorCount    int64
}

// NewPerformanceMetrics creates a new metrics collector
func NewPerformanceMetrics() *PerformanceMetrics {
	return &PerformanceMetrics{
		responseTimes:   make([]time.Duration, 0, 10000),
		statusCodes:     make(map[int]int64),
		endpointMetrics: make(map[string]*EndpointMetric),
		startTime:       time.Now(),
	}
}

// RecordRequest records a request's metrics
func (m *PerformanceMetrics) RecordRequest(endpoint string, duration time.Duration, statusCode int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.responseTimes = append(m.responseTimes, duration)
	m.statusCodes[statusCode]++

	if err != nil || statusCode >= 400 {
		atomic.AddInt64(&m.errorCount, 1)
	} else {
		atomic.AddInt64(&m.successCount, 1)
	}

	// Update endpoint-specific metrics
	em, exists := m.endpointMetrics[endpoint]
	if !exists {
		em = &EndpointMetric{
			Name:        endpoint,
			MinDuration: duration,
			MaxDuration: duration,
		}
		m.endpointMetrics[endpoint] = em
	}

	em.Count++
	em.TotalDuration += duration
	if duration < em.MinDuration {
		em.MinDuration = duration
	}
	if duration > em.MaxDuration {
		em.MaxDuration = duration
	}
	if statusCode >= 400 || err != nil {
		em.ErrorCount++
	}
}

// Finish marks the end of the test
func (m *PerformanceMetrics) Finish() {
	m.endTime = time.Now()
}

// GetReport generates a performance report
func (m *PerformanceMetrics) GetReport() PerformanceReport {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.endTime.IsZero() {
		m.endTime = time.Now()
	}

	duration := m.endTime.Sub(m.startTime)
	totalRequests := int64(len(m.responseTimes))

	// Sort response times for percentile calculation
	sortedTimes := make([]time.Duration, len(m.responseTimes))
	copy(sortedTimes, m.responseTimes)
	sort.Slice(sortedTimes, func(i, j int) bool {
		return sortedTimes[i] < sortedTimes[j]
	})

	report := PerformanceReport{
		TotalRequests:  totalRequests,
		SuccessCount:   m.successCount,
		ErrorCount:     m.errorCount,
		Duration:       duration,
		StatusCodes:    m.statusCodes,
		EndpointReport: make(map[string]EndpointReport),
	}

	if totalRequests > 0 {
		report.RequestsPerSecond = float64(totalRequests) / duration.Seconds()
		report.AvgResponseTime = m.calculateAvg(sortedTimes)
		report.MinResponseTime = sortedTimes[0]
		report.MaxResponseTime = sortedTimes[len(sortedTimes)-1]
		report.P50ResponseTime = m.percentile(sortedTimes, 50)
		report.P90ResponseTime = m.percentile(sortedTimes, 90)
		report.P95ResponseTime = m.percentile(sortedTimes, 95)
		report.P99ResponseTime = m.percentile(sortedTimes, 99)
		report.ErrorRate = float64(m.errorCount) / float64(totalRequests) * 100
	}

	// Generate endpoint reports
	for name, em := range m.endpointMetrics {
		report.EndpointReport[name] = EndpointReport{
			Name:            name,
			Count:           em.Count,
			AvgResponseTime: time.Duration(int64(em.TotalDuration) / em.Count),
			MinResponseTime: em.MinDuration,
			MaxResponseTime: em.MaxDuration,
			ErrorCount:      em.ErrorCount,
			ErrorRate:       float64(em.ErrorCount) / float64(em.Count) * 100,
		}
	}

	return report
}

func (m *PerformanceMetrics) calculateAvg(times []time.Duration) time.Duration {
	if len(times) == 0 {
		return 0
	}
	var total time.Duration
	for _, t := range times {
		total += t
	}
	return time.Duration(int64(total) / int64(len(times)))
}

func (m *PerformanceMetrics) percentile(sortedTimes []time.Duration, p float64) time.Duration {
	if len(sortedTimes) == 0 {
		return 0
	}
	index := int(math.Ceil(p/100*float64(len(sortedTimes)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sortedTimes) {
		index = len(sortedTimes) - 1
	}
	return sortedTimes[index]
}

// PerformanceReport contains the final performance metrics
type PerformanceReport struct {
	TotalRequests     int64
	SuccessCount      int64
	ErrorCount        int64
	Duration          time.Duration
	RequestsPerSecond float64
	AvgResponseTime   time.Duration
	MinResponseTime   time.Duration
	MaxResponseTime   time.Duration
	P50ResponseTime   time.Duration
	P90ResponseTime   time.Duration
	P95ResponseTime   time.Duration
	P99ResponseTime   time.Duration
	ErrorRate         float64
	StatusCodes       map[int]int64
	EndpointReport    map[string]EndpointReport
}

// EndpointReport contains metrics for a specific endpoint
type EndpointReport struct {
	Name            string
	Count           int64
	AvgResponseTime time.Duration
	MinResponseTime time.Duration
	MaxResponseTime time.Duration
	ErrorCount      int64
	ErrorRate       float64
}

// PrintReport prints the performance report to the test log
func (r PerformanceReport) PrintReport(t *testing.T) {
	t.Logf("\n==================== PERFORMANCE REPORT ====================")
	t.Logf("Test Duration:       %v", r.Duration)
	t.Logf("Total Requests:      %d", r.TotalRequests)
	t.Logf("Successful:          %d (%.2f%%)", r.SuccessCount, float64(r.SuccessCount)/float64(r.TotalRequests)*100)
	t.Logf("Failed:              %d (%.2f%%)", r.ErrorCount, r.ErrorRate)
	t.Logf("Requests/Second:     %.2f", r.RequestsPerSecond)
	t.Logf("")
	t.Logf("Response Times:")
	t.Logf("  Average:           %v", r.AvgResponseTime)
	t.Logf("  Min:               %v", r.MinResponseTime)
	t.Logf("  Max:               %v", r.MaxResponseTime)
	t.Logf("  P50:               %v", r.P50ResponseTime)
	t.Logf("  P90:               %v", r.P90ResponseTime)
	t.Logf("  P95:               %v", r.P95ResponseTime)
	t.Logf("  P99:               %v", r.P99ResponseTime)
	t.Logf("")
	t.Logf("Status Codes:")
	for code, count := range r.StatusCodes {
		t.Logf("  %d: %d", code, count)
	}
	t.Logf("")
	t.Logf("Endpoint Performance (sorted by avg response time):")

	// Sort endpoints by average response time (slowest first)
	endpoints := make([]EndpointReport, 0, len(r.EndpointReport))
	for _, ep := range r.EndpointReport {
		endpoints = append(endpoints, ep)
	}
	sort.Slice(endpoints, func(i, j int) bool {
		return endpoints[i].AvgResponseTime > endpoints[j].AvgResponseTime
	})

	for _, ep := range endpoints {
		t.Logf("  %s:", ep.Name)
		t.Logf("    Count: %d, Avg: %v, Min: %v, Max: %v, Errors: %d (%.2f%%)",
			ep.Count, ep.AvgResponseTime, ep.MinResponseTime, ep.MaxResponseTime, ep.ErrorCount, ep.ErrorRate)
	}
	t.Logf("=============================================================\n")
}

// ==================== Performance Test Server Setup ====================

// PerformanceTestServer wraps the test database and HTTP server for performance testing
type PerformanceTestServer struct {
	DB               *TestDB
	Engine           *gin.Engine
	TenantID         uuid.UUID
	ProductIDs       []uuid.UUID
	CustomerIDs      []uuid.UUID
	WarehouseIDs     []uuid.UUID
	InventoryItemIDs []uuid.UUID
}

// NewPerformanceTestServer creates a new test server with pre-populated data
func NewPerformanceTestServer(t *testing.T) *PerformanceTestServer {
	t.Helper()

	gin.SetMode(gin.TestMode)
	testDB := NewTestDB(t)

	// Initialize repositories
	productRepo := persistence.NewGormProductRepository(testDB.DB)
	categoryRepo := persistence.NewGormCategoryRepository(testDB.DB)
	customerRepo := persistence.NewGormCustomerRepository(testDB.DB)
	supplierRepo := persistence.NewGormSupplierRepository(testDB.DB)
	warehouseRepo := persistence.NewGormWarehouseRepository(testDB.DB)
	inventoryRepo := persistence.NewGormInventoryItemRepository(testDB.DB)
	batchRepo := persistence.NewGormStockBatchRepository(testDB.DB)
	lockRepo := persistence.NewGormStockLockRepository(testDB.DB)
	transactionRepo := persistence.NewGormInventoryTransactionRepository(testDB.DB)

	// Initialize strategy registry
	strategyRegistry := infrastrategy.NewStrategyRegistry()

	// Initialize services
	productService := catalogapp.NewProductService(productRepo, categoryRepo, strategyRegistry)
	customerService := partnerapp.NewCustomerService(customerRepo)
	supplierService := partnerapp.NewSupplierService(supplierRepo)
	warehouseService := partnerapp.NewWarehouseService(warehouseRepo)
	inventoryService := inventoryapp.NewInventoryService(inventoryRepo, batchRepo, lockRepo, transactionRepo)

	// Initialize handlers
	productHandler := handler.NewProductHandler(productService)
	customerHandler := handler.NewCustomerHandler(customerService)
	supplierHandler := handler.NewSupplierHandler(supplierService)
	warehouseHandler := handler.NewWarehouseHandler(warehouseService)
	inventoryHandler := handler.NewInventoryHandler(inventoryService)

	// Setup engine
	engine := gin.New()

	// Setup routes
	r := router.NewRouter(engine, router.WithAPIVersion("v1"))

	// Register catalog routes
	catalogRoutes := router.NewDomainGroup("catalog", "/catalog")
	catalogRoutes.POST("/products", productHandler.Create)
	catalogRoutes.GET("/products", productHandler.List)
	catalogRoutes.GET("/products/stats/count", productHandler.CountByStatus)
	catalogRoutes.GET("/products/:id", productHandler.GetByID)
	catalogRoutes.GET("/products/code/:code", productHandler.GetByCode)
	catalogRoutes.PUT("/products/:id", productHandler.Update)
	catalogRoutes.DELETE("/products/:id", productHandler.Delete)
	r.Register(catalogRoutes)

	// Register partner routes
	partnerRoutes := router.NewDomainGroup("partner", "/partner")
	partnerRoutes.POST("/customers", customerHandler.Create)
	partnerRoutes.GET("/customers", customerHandler.List)
	partnerRoutes.GET("/customers/:id", customerHandler.GetByID)
	partnerRoutes.PUT("/customers/:id", customerHandler.Update)
	partnerRoutes.DELETE("/customers/:id", customerHandler.Delete)
	partnerRoutes.POST("/suppliers", supplierHandler.Create)
	partnerRoutes.GET("/suppliers", supplierHandler.List)
	partnerRoutes.GET("/suppliers/:id", supplierHandler.GetByID)
	partnerRoutes.PUT("/suppliers/:id", supplierHandler.Update)
	partnerRoutes.DELETE("/suppliers/:id", supplierHandler.Delete)
	partnerRoutes.POST("/warehouses", warehouseHandler.Create)
	partnerRoutes.GET("/warehouses", warehouseHandler.List)
	partnerRoutes.GET("/warehouses/:id", warehouseHandler.GetByID)
	partnerRoutes.PUT("/warehouses/:id", warehouseHandler.Update)
	partnerRoutes.DELETE("/warehouses/:id", warehouseHandler.Delete)
	r.Register(partnerRoutes)

	// Register inventory routes
	inventoryRoutes := router.NewDomainGroup("inventory", "/inventory")
	inventoryRoutes.GET("/items", inventoryHandler.List)
	inventoryRoutes.GET("/items/:id", inventoryHandler.GetByID)
	inventoryRoutes.GET("/items/warehouse/:warehouse_id/product/:product_id", inventoryHandler.GetByWarehouseAndProduct)
	r.Register(inventoryRoutes)

	r.Setup()

	// Create test tenant and data
	tenantID := uuid.New()
	testDB.CreateTestTenantWithUUID(tenantID)

	server := &PerformanceTestServer{
		DB:               testDB,
		Engine:           engine,
		TenantID:         tenantID,
		ProductIDs:       make([]uuid.UUID, 0),
		CustomerIDs:      make([]uuid.UUID, 0),
		WarehouseIDs:     make([]uuid.UUID, 0),
		InventoryItemIDs: make([]uuid.UUID, 0),
	}

	// Pre-populate test data
	server.populateTestData(t)

	return server
}

// populateTestData creates test data for performance testing
func (s *PerformanceTestServer) populateTestData(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	// Create products
	productRepo := persistence.NewGormProductRepository(s.DB.DB)
	for i := 0; i < 100; i++ {
		product, err := catalog.NewProduct(
			s.TenantID,
			fmt.Sprintf("PERF-PROD-%04d", i),
			fmt.Sprintf("Performance Test Product %d", i),
			"pcs",
		)
		require.NoError(t, err)
		err = productRepo.Save(ctx, product)
		require.NoError(t, err)
		s.ProductIDs = append(s.ProductIDs, product.ID)
	}

	// Create customers
	customerRepo := persistence.NewGormCustomerRepository(s.DB.DB)
	for i := 0; i < 50; i++ {
		customer, err := partner.NewCustomer(
			s.TenantID,
			fmt.Sprintf("PERF-CUST-%04d", i),
			fmt.Sprintf("Performance Test Customer %d", i),
			partner.CustomerTypeIndividual,
		)
		require.NoError(t, err)
		err = customerRepo.Save(ctx, customer)
		require.NoError(t, err)
		s.CustomerIDs = append(s.CustomerIDs, customer.ID)
	}

	// Create warehouses
	warehouseRepo := persistence.NewGormWarehouseRepository(s.DB.DB)
	for i := 0; i < 5; i++ {
		warehouse, err := partner.NewWarehouse(
			s.TenantID,
			fmt.Sprintf("PERF-WH-%04d", i),
			fmt.Sprintf("Performance Test Warehouse %d", i),
			partner.WarehouseTypePhysical,
		)
		require.NoError(t, err)
		err = warehouseRepo.Save(ctx, warehouse)
		require.NoError(t, err)
		s.WarehouseIDs = append(s.WarehouseIDs, warehouse.ID)
	}

	// Create inventory items
	inventoryRepo := persistence.NewGormInventoryItemRepository(s.DB.DB)
	for _, whID := range s.WarehouseIDs {
		for _, prodID := range s.ProductIDs[:20] { // First 20 products in each warehouse
			item, err := inventory.NewInventoryItem(s.TenantID, whID, prodID)
			require.NoError(t, err)
			// Add some initial stock
			unitCost := valueobject.NewMoneyCNY(decimal.NewFromFloat(10.00))
			err = item.IncreaseStock(decimal.NewFromFloat(1000), unitCost, nil)
			require.NoError(t, err)
			err = inventoryRepo.Save(ctx, item)
			require.NoError(t, err)
			s.InventoryItemIDs = append(s.InventoryItemIDs, item.ID)
		}
	}
}

// Request makes an HTTP request to the test server and returns timing information
func (s *PerformanceTestServer) Request(method, path string, body interface{}) (int, time.Duration, error) {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return 0, 0, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", s.TenantID.String())

	w := httptest.NewRecorder()

	start := time.Now()
	s.Engine.ServeHTTP(w, req)
	duration := time.Since(start)

	return w.Code, duration, nil
}

// ==================== Concurrent Testing ====================

func TestPerformance_ConcurrentProductList(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	server := NewPerformanceTestServer(t)
	config := DefaultPerformanceConfig()
	metrics := NewPerformanceMetrics()

	var wg sync.WaitGroup
	endpoint := "/api/v1/catalog/products"

	t.Run("concurrent_product_list_requests", func(t *testing.T) {
		for i := 0; i < config.ConcurrentUsers; i++ {
			wg.Add(1)
			go func(userID int) {
				defer wg.Done()
				for j := 0; j < config.RequestsPerUser; j++ {
					code, duration, err := server.Request("GET", endpoint+"?page=1&page_size=20", nil)
					metrics.RecordRequest("GET "+endpoint, duration, code, err)
				}
			}(i)
		}

		wg.Wait()
		metrics.Finish()

		report := metrics.GetReport()
		report.PrintReport(t)

		// Assertions
		assert.Zero(t, report.ErrorCount, "Should have no errors")
		assert.LessOrEqual(t, report.P95ResponseTime, config.P95ResponseTime,
			"P95 response time should be under %v", config.P95ResponseTime)
	})
}

func TestPerformance_ConcurrentProductGetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	server := NewPerformanceTestServer(t)
	config := DefaultPerformanceConfig()
	metrics := NewPerformanceMetrics()

	var wg sync.WaitGroup

	t.Run("concurrent_product_get_by_id", func(t *testing.T) {
		for i := 0; i < config.ConcurrentUsers; i++ {
			wg.Add(1)
			go func(userID int) {
				defer wg.Done()
				for j := 0; j < config.RequestsPerUser; j++ {
					// Round-robin through product IDs
					productID := server.ProductIDs[j%len(server.ProductIDs)]
					endpoint := fmt.Sprintf("/api/v1/catalog/products/%s", productID)
					code, duration, err := server.Request("GET", endpoint, nil)
					metrics.RecordRequest("GET /products/:id", duration, code, err)
				}
			}(i)
		}

		wg.Wait()
		metrics.Finish()

		report := metrics.GetReport()
		report.PrintReport(t)

		assert.Zero(t, report.ErrorCount, "Should have no errors")
		assert.LessOrEqual(t, report.P95ResponseTime, config.P95ResponseTime,
			"P95 response time should be under %v", config.P95ResponseTime)
	})
}

func TestPerformance_ConcurrentMixedOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	server := NewPerformanceTestServer(t)
	config := DefaultPerformanceConfig()
	metrics := NewPerformanceMetrics()

	var wg sync.WaitGroup
	var productCounter int64

	t.Run("concurrent_mixed_crud_operations", func(t *testing.T) {
		for i := 0; i < config.ConcurrentUsers; i++ {
			wg.Add(1)
			go func(userID int) {
				defer wg.Done()
				for j := 0; j < config.RequestsPerUser; j++ {
					// Mix of operations: 50% reads, 30% creates, 20% updates
					operation := j % 10
					switch {
					case operation < 5: // 50% - List products
						endpoint := "/api/v1/catalog/products?page=1&page_size=20"
						code, duration, err := server.Request("GET", endpoint, nil)
						metrics.RecordRequest("GET /products (list)", duration, code, err)

					case operation < 8: // 30% - Create product
						counter := atomic.AddInt64(&productCounter, 1)
						body := map[string]interface{}{
							"code":      fmt.Sprintf("PERF-NEW-%d-%d-%d", userID, j, counter),
							"name":      fmt.Sprintf("Performance New Product %d", counter),
							"unit":      "pcs",
							"tenant_id": server.TenantID.String(),
						}
						code, duration, err := server.Request("POST", "/api/v1/catalog/products", body)
						metrics.RecordRequest("POST /products (create)", duration, code, err)

					default: // 20% - Get product by ID
						productID := server.ProductIDs[j%len(server.ProductIDs)]
						endpoint := fmt.Sprintf("/api/v1/catalog/products/%s", productID)
						code, duration, err := server.Request("GET", endpoint, nil)
						metrics.RecordRequest("GET /products/:id", duration, code, err)
					}
				}
			}(i)
		}

		wg.Wait()
		metrics.Finish()

		report := metrics.GetReport()
		report.PrintReport(t)

		// Assertions - allow some errors for concurrent creates (potential code conflicts)
		assert.LessOrEqual(t, report.ErrorRate, 5.0, "Error rate should be under 5%%")
		assert.LessOrEqual(t, report.P95ResponseTime, config.P95ResponseTime*2, // Allow 2x for writes
			"P95 response time should be under %v", config.P95ResponseTime*2)
	})
}

func TestPerformance_ConcurrentInventoryQueries(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	server := NewPerformanceTestServer(t)
	config := DefaultPerformanceConfig()
	metrics := NewPerformanceMetrics()

	var wg sync.WaitGroup

	t.Run("concurrent_inventory_queries", func(t *testing.T) {
		for i := 0; i < config.ConcurrentUsers; i++ {
			wg.Add(1)
			go func(userID int) {
				defer wg.Done()
				for j := 0; j < config.RequestsPerUser; j++ {
					operation := j % 2
					switch operation {
					case 0: // List inventory items
						endpoint := "/api/v1/inventory/items?page=1&page_size=20"
						code, duration, err := server.Request("GET", endpoint, nil)
						metrics.RecordRequest("GET /inventory/items (list)", duration, code, err)

					case 1: // Get inventory by ID
						itemID := server.InventoryItemIDs[j%len(server.InventoryItemIDs)]
						endpoint := fmt.Sprintf("/api/v1/inventory/items/%s", itemID)
						code, duration, err := server.Request("GET", endpoint, nil)
						metrics.RecordRequest("GET /inventory/items/:id", duration, code, err)
					}
				}
			}(i)
		}

		wg.Wait()
		metrics.Finish()

		report := metrics.GetReport()
		report.PrintReport(t)

		assert.Zero(t, report.ErrorCount, "Should have no errors")
		assert.LessOrEqual(t, report.P95ResponseTime, config.P95ResponseTime,
			"P95 response time should be under %v", config.P95ResponseTime)
	})
}

func TestPerformance_ConcurrentCustomerOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	server := NewPerformanceTestServer(t)
	config := DefaultPerformanceConfig()
	metrics := NewPerformanceMetrics()

	var wg sync.WaitGroup

	t.Run("concurrent_customer_operations", func(t *testing.T) {
		for i := 0; i < config.ConcurrentUsers; i++ {
			wg.Add(1)
			go func(userID int) {
				defer wg.Done()
				for j := 0; j < config.RequestsPerUser; j++ {
					operation := j % 2
					switch operation {
					case 0: // List customers
						endpoint := "/api/v1/partner/customers?page=1&page_size=20"
						code, duration, err := server.Request("GET", endpoint, nil)
						metrics.RecordRequest("GET /customers (list)", duration, code, err)

					case 1: // Get customer by ID
						customerID := server.CustomerIDs[j%len(server.CustomerIDs)]
						endpoint := fmt.Sprintf("/api/v1/partner/customers/%s", customerID)
						code, duration, err := server.Request("GET", endpoint, nil)
						metrics.RecordRequest("GET /customers/:id", duration, code, err)
					}
				}
			}(i)
		}

		wg.Wait()
		metrics.Finish()

		report := metrics.GetReport()
		report.PrintReport(t)

		assert.Zero(t, report.ErrorCount, "Should have no errors")
		assert.LessOrEqual(t, report.P95ResponseTime, config.P95ResponseTime,
			"P95 response time should be under %v", config.P95ResponseTime)
	})
}

// ==================== Load Testing ====================

func TestPerformance_SustainedLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	server := NewPerformanceTestServer(t)
	config := DefaultPerformanceConfig()
	metrics := NewPerformanceMetrics()

	t.Run("sustained_load_test", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), config.TestDuration)
		defer cancel()

		var wg sync.WaitGroup
		requestsPerSecond := config.TargetRPS
		ticker := time.NewTicker(time.Second / time.Duration(requestsPerSecond))
		defer ticker.Stop()

		// Start worker goroutines
		requestChan := make(chan struct{}, requestsPerSecond*2)

		for i := 0; i < 20; i++ { // 20 worker goroutines
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				requestCounter := 0
				for {
					select {
					case <-ctx.Done():
						return
					case <-requestChan:
						requestCounter++
						operation := requestCounter % 4
						switch operation {
						case 0:
							endpoint := "/api/v1/catalog/products?page=1&page_size=20"
							code, duration, err := server.Request("GET", endpoint, nil)
							metrics.RecordRequest("GET /products", duration, code, err)
						case 1:
							productID := server.ProductIDs[requestCounter%len(server.ProductIDs)]
							endpoint := fmt.Sprintf("/api/v1/catalog/products/%s", productID)
							code, duration, err := server.Request("GET", endpoint, nil)
							metrics.RecordRequest("GET /products/:id", duration, code, err)
						case 2:
							endpoint := "/api/v1/partner/customers?page=1&page_size=20"
							code, duration, err := server.Request("GET", endpoint, nil)
							metrics.RecordRequest("GET /customers", duration, code, err)
						case 3:
							endpoint := "/api/v1/inventory/items?page=1&page_size=20"
							code, duration, err := server.Request("GET", endpoint, nil)
							metrics.RecordRequest("GET /inventory", duration, code, err)
						}
					}
				}
			}(i)
		}

		// Send requests at target rate
		go func() {
			for {
				select {
				case <-ctx.Done():
					close(requestChan)
					return
				case <-ticker.C:
					select {
					case requestChan <- struct{}{}:
					default:
						// Channel full, skip this tick
					}
				}
			}
		}()

		wg.Wait()
		metrics.Finish()

		report := metrics.GetReport()
		report.PrintReport(t)

		// Assertions
		assert.LessOrEqual(t, report.ErrorRate, 1.0, "Error rate should be under 1%%")
		assert.GreaterOrEqual(t, report.RequestsPerSecond, float64(config.TargetRPS)*0.8,
			"Should achieve at least 80%% of target RPS")
		assert.LessOrEqual(t, report.P95ResponseTime, config.P95ResponseTime,
			"P95 response time should be under %v", config.P95ResponseTime)
	})
}

// ==================== Bottleneck Identification ====================

func TestPerformance_EndpointComparison(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	server := NewPerformanceTestServer(t)
	metrics := NewPerformanceMetrics()

	endpoints := []struct {
		name   string
		method string
		path   string
		body   interface{}
	}{
		{"Product List", "GET", "/api/v1/catalog/products?page=1&page_size=20", nil},
		{"Product List Large", "GET", "/api/v1/catalog/products?page=1&page_size=100", nil},
		{"Product Get", "GET", fmt.Sprintf("/api/v1/catalog/products/%s", server.ProductIDs[0]), nil},
		{"Product Stats", "GET", "/api/v1/catalog/products/stats/count", nil},
		{"Customer List", "GET", "/api/v1/partner/customers?page=1&page_size=20", nil},
		{"Customer Get", "GET", fmt.Sprintf("/api/v1/partner/customers/%s", server.CustomerIDs[0]), nil},
		{"Warehouse List", "GET", "/api/v1/partner/warehouses?page=1&page_size=20", nil},
		{"Inventory List", "GET", "/api/v1/inventory/items?page=1&page_size=20", nil},
		{"Inventory Get", "GET", fmt.Sprintf("/api/v1/inventory/items/%s", server.InventoryItemIDs[0]), nil},
	}

	t.Run("endpoint_comparison", func(t *testing.T) {
		iterations := 100

		for _, ep := range endpoints {
			for i := 0; i < iterations; i++ {
				code, duration, err := server.Request(ep.method, ep.path, ep.body)
				metrics.RecordRequest(ep.name, duration, code, err)
			}
		}

		metrics.Finish()
		report := metrics.GetReport()
		report.PrintReport(t)

		// Identify potential bottlenecks (endpoints with high avg response time)
		t.Log("\n==================== BOTTLENECK ANALYSIS ====================")
		for name, ep := range report.EndpointReport {
			if ep.AvgResponseTime > 50*time.Millisecond {
				t.Logf("POTENTIAL BOTTLENECK: %s (avg: %v)", name, ep.AvgResponseTime)
			}
		}
		t.Log("=============================================================\n")
	})
}

func TestPerformance_DatabaseConnectionPool(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	server := NewPerformanceTestServer(t)
	metrics := NewPerformanceMetrics()

	t.Run("connection_pool_stress", func(t *testing.T) {
		// Stress test with many concurrent connections
		var wg sync.WaitGroup
		concurrency := 50
		requestsPerGoroutine := 20

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < requestsPerGoroutine; j++ {
					endpoint := "/api/v1/catalog/products?page=1&page_size=10"
					code, duration, err := server.Request("GET", endpoint, nil)
					metrics.RecordRequest("connection_pool_test", duration, code, err)
				}
			}(i)
		}

		wg.Wait()
		metrics.Finish()

		report := metrics.GetReport()
		report.PrintReport(t)

		// The connection pool should handle the load without errors
		assert.Zero(t, report.ErrorCount, "Should have no connection errors")
		assert.LessOrEqual(t, report.P99ResponseTime, 1*time.Second,
			"P99 should be under 1s even under connection pressure")
	})
}

func TestPerformance_LargeResultSet(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	server := NewPerformanceTestServer(t)
	metrics := NewPerformanceMetrics()

	t.Run("large_result_set_pagination", func(t *testing.T) {
		pageSizes := []int{10, 20, 50, 100}
		iterations := 20

		for _, pageSize := range pageSizes {
			for i := 0; i < iterations; i++ {
				endpoint := fmt.Sprintf("/api/v1/catalog/products?page=1&page_size=%d", pageSize)
				code, duration, err := server.Request("GET", endpoint, nil)
				name := fmt.Sprintf("products_page_size_%d", pageSize)
				metrics.RecordRequest(name, duration, code, err)
			}
		}

		metrics.Finish()
		report := metrics.GetReport()
		report.PrintReport(t)

		// Analyze scaling of response time with page size
		t.Log("\n==================== PAGE SIZE ANALYSIS ====================")
		for _, pageSize := range pageSizes {
			name := fmt.Sprintf("products_page_size_%d", pageSize)
			if ep, ok := report.EndpointReport[name]; ok {
				t.Logf("Page size %d: avg %v, max %v", pageSize, ep.AvgResponseTime, ep.MaxResponseTime)
			}
		}
		t.Log("=============================================================\n")
	})
}

// ==================== Concurrent Write Operations ====================

func TestPerformance_ConcurrentWrites(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	server := NewPerformanceTestServer(t)
	config := DefaultPerformanceConfig()
	metrics := NewPerformanceMetrics()

	var wg sync.WaitGroup
	var productCounter int64

	t.Run("concurrent_product_creation", func(t *testing.T) {
		for i := 0; i < config.ConcurrentUsers; i++ {
			wg.Add(1)
			go func(userID int) {
				defer wg.Done()
				for j := 0; j < 10; j++ { // 10 creates per user
					counter := atomic.AddInt64(&productCounter, 1)
					body := map[string]interface{}{
						"code":      fmt.Sprintf("PERF-WRITE-%d-%d", userID, counter),
						"name":      fmt.Sprintf("Performance Write Product %d", counter),
						"unit":      "pcs",
						"tenant_id": server.TenantID.String(),
					}
					code, duration, err := server.Request("POST", "/api/v1/catalog/products", body)
					metrics.RecordRequest("POST /products", duration, code, err)
				}
			}(i)
		}

		wg.Wait()
		metrics.Finish()

		report := metrics.GetReport()
		report.PrintReport(t)

		// Writes are slower, allow longer response times
		assert.LessOrEqual(t, report.ErrorRate, 5.0, "Error rate should be under 5%%")
		assert.LessOrEqual(t, report.P95ResponseTime, config.P95ResponseTime*3,
			"P95 response time should be under %v for writes", config.P95ResponseTime*3)
	})
}

// ==================== Summary Test ====================

func TestPerformance_Summary(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	server := NewPerformanceTestServer(t)
	metrics := NewPerformanceMetrics()

	t.Run("comprehensive_performance_summary", func(t *testing.T) {
		var wg sync.WaitGroup
		config := DefaultPerformanceConfig()

		// Run mixed workload for comprehensive metrics
		for i := 0; i < config.ConcurrentUsers; i++ {
			wg.Add(1)
			go func(userID int) {
				defer wg.Done()
				for j := 0; j < config.RequestsPerUser; j++ {
					op := j % 10
					switch {
					case op < 3: // 30% product reads
						endpoint := "/api/v1/catalog/products?page=1&page_size=20"
						code, duration, err := server.Request("GET", endpoint, nil)
						metrics.RecordRequest("Product List", duration, code, err)
					case op < 5: // 20% product detail
						productID := server.ProductIDs[j%len(server.ProductIDs)]
						endpoint := fmt.Sprintf("/api/v1/catalog/products/%s", productID)
						code, duration, err := server.Request("GET", endpoint, nil)
						metrics.RecordRequest("Product Detail", duration, code, err)
					case op < 7: // 20% customer reads
						endpoint := "/api/v1/partner/customers?page=1&page_size=20"
						code, duration, err := server.Request("GET", endpoint, nil)
						metrics.RecordRequest("Customer List", duration, code, err)
					case op < 9: // 20% inventory queries
						endpoint := "/api/v1/inventory/items?page=1&page_size=20"
						code, duration, err := server.Request("GET", endpoint, nil)
						metrics.RecordRequest("Inventory List", duration, code, err)
					default: // 10% warehouse reads
						endpoint := "/api/v1/partner/warehouses?page=1&page_size=20"
						code, duration, err := server.Request("GET", endpoint, nil)
						metrics.RecordRequest("Warehouse List", duration, code, err)
					}
				}
			}(i)
		}

		wg.Wait()
		metrics.Finish()

		report := metrics.GetReport()
		report.PrintReport(t)

		// Final assertions
		t.Log("\n==================== FINAL ASSESSMENT ====================")
		t.Logf("Total Requests: %d", report.TotalRequests)
		t.Logf("Throughput: %.2f req/s", report.RequestsPerSecond)
		t.Logf("Error Rate: %.2f%%", report.ErrorRate)
		t.Logf("P95 Response Time: %v", report.P95ResponseTime)
		t.Logf("P99 Response Time: %v", report.P99ResponseTime)

		// Performance grade
		grade := "A"
		if report.ErrorRate > 0.1 {
			grade = "B"
		}
		if report.P95ResponseTime > 100*time.Millisecond {
			grade = "C"
		}
		if report.P95ResponseTime > 200*time.Millisecond {
			grade = "D"
		}
		if report.ErrorRate > 1.0 || report.P95ResponseTime > 500*time.Millisecond {
			grade = "F"
		}
		t.Logf("Performance Grade: %s", grade)
		t.Log("=============================================================\n")

		// Assertions
		assert.LessOrEqual(t, report.ErrorRate, 1.0, "Error rate should be under 1%%")
		assert.LessOrEqual(t, report.P95ResponseTime, config.P95ResponseTime,
			"P95 response time should be under %v", config.P95ResponseTime)
	})
}
