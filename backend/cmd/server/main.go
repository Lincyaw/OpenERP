package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	catalogapp "github.com/erp/backend/internal/application/catalog"
	inventoryapp "github.com/erp/backend/internal/application/inventory"
	partnerapp "github.com/erp/backend/internal/application/partner"
	tradeapp "github.com/erp/backend/internal/application/trade"
	"github.com/erp/backend/internal/infrastructure/config"
	"github.com/erp/backend/internal/infrastructure/logger"
	"github.com/erp/backend/internal/infrastructure/persistence"
	"github.com/erp/backend/internal/interfaces/http/handler"
	"github.com/erp/backend/internal/interfaces/http/middleware"
	"github.com/erp/backend/internal/interfaces/http/router"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	_ "github.com/erp/backend/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           ERP Backend API
// @version         1.0
// @description     进销存系统后端 API - 基于 DDD 设计的库存管理系统
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    https://github.com/erp/backend
// @contact.email  support@erp.example.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Bearer token authentication. Format: "Bearer {token}"

// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic("Failed to load configuration: " + err.Error())
	}

	// Initialize logger
	log, err := logger.New(&logger.Config{
		Level:      cfg.Log.Level,
		Format:     cfg.Log.Format,
		Output:     cfg.Log.Output,
		TimeFormat: "2006-01-02T15:04:05.000Z07:00",
	})
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	defer func() {
		_ = logger.Sync(log)
	}()

	log.Info("Starting ERP Backend",
		zap.String("app", cfg.App.Name),
		zap.String("env", cfg.App.Env),
		zap.String("port", cfg.App.Port),
	)

	// Create GORM logger backed by zap
	gormLogLevel := logger.MapGormLogLevel(cfg.Log.Level)
	gormLog := logger.NewGormLogger(log, gormLogLevel)

	// Initialize database connection with custom logger
	db, err := persistence.NewDatabaseWithCustomLogger(&cfg.Database, gormLog)
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("Error closing database", zap.Error(err))
		}
	}()
	log.Info("Database connected successfully")

	// Initialize repositories
	productRepo := persistence.NewGormProductRepository(db.DB)
	categoryRepo := persistence.NewGormCategoryRepository(db.DB)
	customerRepo := persistence.NewGormCustomerRepository(db.DB)
	supplierRepo := persistence.NewGormSupplierRepository(db.DB)
	warehouseRepo := persistence.NewGormWarehouseRepository(db.DB)
	inventoryItemRepo := persistence.NewGormInventoryItemRepository(db.DB)
	stockBatchRepo := persistence.NewGormStockBatchRepository(db.DB)
	stockLockRepo := persistence.NewGormStockLockRepository(db.DB)
	inventoryTxRepo := persistence.NewGormInventoryTransactionRepository(db.DB)
	salesOrderRepo := persistence.NewGormSalesOrderRepository(db.DB)
	purchaseOrderRepo := persistence.NewGormPurchaseOrderRepository(db.DB)

	// Initialize application services
	productService := catalogapp.NewProductService(productRepo, categoryRepo)
	customerService := partnerapp.NewCustomerService(customerRepo)
	supplierService := partnerapp.NewSupplierService(supplierRepo)
	warehouseService := partnerapp.NewWarehouseService(warehouseRepo)
	inventoryService := inventoryapp.NewInventoryService(inventoryItemRepo, stockBatchRepo, stockLockRepo, inventoryTxRepo)
	salesOrderService := tradeapp.NewSalesOrderService(salesOrderRepo)
	purchaseOrderService := tradeapp.NewPurchaseOrderService(purchaseOrderRepo)

	// Initialize HTTP handlers
	productHandler := handler.NewProductHandler(productService)
	customerHandler := handler.NewCustomerHandler(customerService)
	supplierHandler := handler.NewSupplierHandler(supplierService)
	warehouseHandler := handler.NewWarehouseHandler(warehouseService)
	inventoryHandler := handler.NewInventoryHandler(inventoryService)
	salesOrderHandler := handler.NewSalesOrderHandler(salesOrderService)
	purchaseOrderHandler := handler.NewPurchaseOrderHandler(purchaseOrderService)

	// Set Gin mode based on environment
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Setup validation
	middleware.SetupValidator()

	// Initialize router with custom middleware
	engine := gin.New()

	// Configure trusted proxies
	if len(cfg.HTTP.TrustedProxies) > 0 {
		if err := engine.SetTrustedProxies(cfg.HTTP.TrustedProxies); err != nil {
			log.Warn("Failed to set trusted proxies", zap.Error(err))
		}
	}

	// Apply middleware stack in order:
	// 1. RequestID - Generate/propagate request ID
	// 2. Recovery - Catch panics
	// 3. Logger - Log requests
	// 4. Security - Add security headers
	// 5. CORS - Handle cross-origin requests
	// 6. BodyLimit - Limit request body size
	// 7. RateLimit - Apply rate limiting (if enabled)
	engine.Use(middleware.RequestID())
	engine.Use(logger.Recovery(log))
	engine.Use(logger.GinMiddleware(log))
	engine.Use(middleware.Secure())

	// Configure CORS from config
	corsConfig := middleware.CORSConfig{
		AllowOrigins:     cfg.HTTP.CORSAllowOrigins,
		AllowMethods:     cfg.HTTP.CORSAllowMethods,
		AllowHeaders:     cfg.HTTP.CORSAllowHeaders,
		ExposeHeaders:    []string{"X-Request-ID", "X-RateLimit-Limit", "X-RateLimit-Remaining"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	engine.Use(middleware.CORSWithConfig(corsConfig))

	// Body size limit
	engine.Use(middleware.BodyLimit(cfg.HTTP.MaxBodySize))

	// Rate limiting (if enabled)
	if cfg.HTTP.RateLimitEnabled {
		rateLimiter := middleware.NewRateLimiter(cfg.HTTP.RateLimitRequests, cfg.HTTP.RateLimitWindow)
		engine.Use(middleware.RateLimit(rateLimiter))
		log.Info("Rate limiting enabled",
			zap.Int("requests", cfg.HTTP.RateLimitRequests),
			zap.Duration("window", cfg.HTTP.RateLimitWindow),
		)
	}

	// Health check endpoint (outside API versioning)
	engine.GET("/health", healthHandler(db, log))

	// Swagger documentation endpoint
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Setup API routes using router
	r := router.NewRouter(engine, router.WithAPIVersion("v1"))

	// Register domain route groups
	// These will be populated as domain APIs are implemented

	// Catalog domain (products, categories)
	catalogRoutes := router.NewDomainGroup("catalog", "/catalog")
	catalogRoutes.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "catalog service ready"})
	})
	// Product routes
	catalogRoutes.POST("/products", productHandler.Create)
	catalogRoutes.GET("/products", productHandler.List)
	catalogRoutes.GET("/products/stats/count", productHandler.CountByStatus)
	catalogRoutes.GET("/products/:id", productHandler.GetByID)
	catalogRoutes.GET("/products/code/:code", productHandler.GetByCode)
	catalogRoutes.PUT("/products/:id", productHandler.Update)
	catalogRoutes.PUT("/products/:id/code", productHandler.UpdateCode)
	catalogRoutes.DELETE("/products/:id", productHandler.Delete)
	catalogRoutes.POST("/products/:id/activate", productHandler.Activate)
	catalogRoutes.POST("/products/:id/deactivate", productHandler.Deactivate)
	catalogRoutes.POST("/products/:id/discontinue", productHandler.Discontinue)
	// Products by category
	catalogRoutes.GET("/categories/:category_id/products", productHandler.GetByCategory)

	// Partner domain (customers, suppliers, warehouses)
	partnerRoutes := router.NewDomainGroup("partner", "/partner")
	partnerRoutes.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "partner service ready"})
	})

	// Customer routes
	partnerRoutes.POST("/customers", customerHandler.Create)
	partnerRoutes.GET("/customers", customerHandler.List)
	partnerRoutes.GET("/customers/stats/count", customerHandler.CountByStatus)
	partnerRoutes.GET("/customers/:id", customerHandler.GetByID)
	partnerRoutes.GET("/customers/code/:code", customerHandler.GetByCode)
	partnerRoutes.PUT("/customers/:id", customerHandler.Update)
	partnerRoutes.PUT("/customers/:id/code", customerHandler.UpdateCode)
	partnerRoutes.DELETE("/customers/:id", customerHandler.Delete)
	partnerRoutes.POST("/customers/:id/activate", customerHandler.Activate)
	partnerRoutes.POST("/customers/:id/deactivate", customerHandler.Deactivate)
	partnerRoutes.POST("/customers/:id/suspend", customerHandler.Suspend)
	partnerRoutes.POST("/customers/:id/balance/add", customerHandler.AddBalance)
	partnerRoutes.POST("/customers/:id/balance/deduct", customerHandler.DeductBalance)
	partnerRoutes.PUT("/customers/:id/level", customerHandler.SetLevel)

	// Supplier routes
	partnerRoutes.POST("/suppliers", supplierHandler.Create)
	partnerRoutes.GET("/suppliers", supplierHandler.List)
	partnerRoutes.GET("/suppliers/stats/count", supplierHandler.CountByStatus)
	partnerRoutes.GET("/suppliers/:id", supplierHandler.GetByID)
	partnerRoutes.GET("/suppliers/code/:code", supplierHandler.GetByCode)
	partnerRoutes.PUT("/suppliers/:id", supplierHandler.Update)
	partnerRoutes.PUT("/suppliers/:id/code", supplierHandler.UpdateCode)
	partnerRoutes.DELETE("/suppliers/:id", supplierHandler.Delete)
	partnerRoutes.POST("/suppliers/:id/activate", supplierHandler.Activate)
	partnerRoutes.POST("/suppliers/:id/deactivate", supplierHandler.Deactivate)
	partnerRoutes.POST("/suppliers/:id/block", supplierHandler.Block)
	partnerRoutes.PUT("/suppliers/:id/rating", supplierHandler.SetRating)
	partnerRoutes.PUT("/suppliers/:id/payment-terms", supplierHandler.SetPaymentTerms)

	// Warehouse routes
	partnerRoutes.POST("/warehouses", warehouseHandler.Create)
	partnerRoutes.GET("/warehouses", warehouseHandler.List)
	partnerRoutes.GET("/warehouses/stats/count", warehouseHandler.CountByStatus)
	partnerRoutes.GET("/warehouses/default", warehouseHandler.GetDefault)
	partnerRoutes.GET("/warehouses/:id", warehouseHandler.GetByID)
	partnerRoutes.GET("/warehouses/code/:code", warehouseHandler.GetByCode)
	partnerRoutes.PUT("/warehouses/:id", warehouseHandler.Update)
	partnerRoutes.PUT("/warehouses/:id/code", warehouseHandler.UpdateCode)
	partnerRoutes.DELETE("/warehouses/:id", warehouseHandler.Delete)
	partnerRoutes.POST("/warehouses/:id/enable", warehouseHandler.Enable)
	partnerRoutes.POST("/warehouses/:id/disable", warehouseHandler.Disable)
	partnerRoutes.POST("/warehouses/:id/set-default", warehouseHandler.SetDefault)

	// Inventory domain
	inventoryRoutes := router.NewDomainGroup("inventory", "/inventory")
	inventoryRoutes.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "inventory service ready"})
	})

	// Inventory item query routes
	inventoryRoutes.GET("/items", inventoryHandler.List)
	inventoryRoutes.GET("/items/lookup", inventoryHandler.GetByWarehouseAndProduct)
	inventoryRoutes.GET("/items/alerts/low-stock", inventoryHandler.ListBelowMinimum)
	inventoryRoutes.GET("/items/:id", inventoryHandler.GetByID)
	inventoryRoutes.GET("/items/:id/transactions", inventoryHandler.ListTransactionsByItem)

	// Inventory by warehouse/product
	inventoryRoutes.GET("/warehouses/:warehouse_id/items", inventoryHandler.ListByWarehouse)
	inventoryRoutes.GET("/products/:product_id/items", inventoryHandler.ListByProduct)

	// Stock operations
	inventoryRoutes.POST("/availability/check", inventoryHandler.CheckAvailability)
	inventoryRoutes.POST("/stock/increase", inventoryHandler.IncreaseStock)
	inventoryRoutes.POST("/stock/lock", inventoryHandler.LockStock)
	inventoryRoutes.POST("/stock/unlock", inventoryHandler.UnlockStock)
	inventoryRoutes.POST("/stock/deduct", inventoryHandler.DeductStock)
	inventoryRoutes.POST("/stock/adjust", inventoryHandler.AdjustStock)
	inventoryRoutes.PUT("/thresholds", inventoryHandler.SetThresholds)

	// Lock management
	inventoryRoutes.GET("/locks", inventoryHandler.GetActiveLocks)
	inventoryRoutes.GET("/locks/:id", inventoryHandler.GetLockByID)

	// Transaction audit
	inventoryRoutes.GET("/transactions", inventoryHandler.ListTransactions)
	inventoryRoutes.GET("/transactions/:id", inventoryHandler.GetTransactionByID)

	// Trade domain (sales orders, purchase orders)
	tradeRoutes := router.NewDomainGroup("trade", "/trade")
	tradeRoutes.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "trade service ready"})
	})

	// Sales Order routes
	tradeRoutes.POST("/sales-orders", salesOrderHandler.Create)
	tradeRoutes.GET("/sales-orders", salesOrderHandler.List)
	tradeRoutes.GET("/sales-orders/stats/summary", salesOrderHandler.GetStatusSummary)
	tradeRoutes.GET("/sales-orders/number/:order_number", salesOrderHandler.GetByOrderNumber)
	tradeRoutes.GET("/sales-orders/:id", salesOrderHandler.GetByID)
	tradeRoutes.PUT("/sales-orders/:id", salesOrderHandler.Update)
	tradeRoutes.DELETE("/sales-orders/:id", salesOrderHandler.Delete)
	tradeRoutes.POST("/sales-orders/:id/items", salesOrderHandler.AddItem)
	tradeRoutes.PUT("/sales-orders/:id/items/:item_id", salesOrderHandler.UpdateItem)
	tradeRoutes.DELETE("/sales-orders/:id/items/:item_id", salesOrderHandler.RemoveItem)
	tradeRoutes.POST("/sales-orders/:id/confirm", salesOrderHandler.Confirm)
	tradeRoutes.POST("/sales-orders/:id/ship", salesOrderHandler.Ship)
	tradeRoutes.POST("/sales-orders/:id/complete", salesOrderHandler.Complete)
	tradeRoutes.POST("/sales-orders/:id/cancel", salesOrderHandler.Cancel)

	// Purchase Order routes
	tradeRoutes.POST("/purchase-orders", purchaseOrderHandler.Create)
	tradeRoutes.GET("/purchase-orders", purchaseOrderHandler.List)
	tradeRoutes.GET("/purchase-orders/stats/summary", purchaseOrderHandler.GetStatusSummary)
	tradeRoutes.GET("/purchase-orders/pending-receipt", purchaseOrderHandler.ListPendingReceipt)
	tradeRoutes.GET("/purchase-orders/number/:order_number", purchaseOrderHandler.GetByOrderNumber)
	tradeRoutes.GET("/purchase-orders/:id", purchaseOrderHandler.GetByID)
	tradeRoutes.GET("/purchase-orders/:id/receivable-items", purchaseOrderHandler.GetReceivableItems)
	tradeRoutes.PUT("/purchase-orders/:id", purchaseOrderHandler.Update)
	tradeRoutes.DELETE("/purchase-orders/:id", purchaseOrderHandler.Delete)
	tradeRoutes.POST("/purchase-orders/:id/items", purchaseOrderHandler.AddItem)
	tradeRoutes.PUT("/purchase-orders/:id/items/:item_id", purchaseOrderHandler.UpdateItem)
	tradeRoutes.DELETE("/purchase-orders/:id/items/:item_id", purchaseOrderHandler.RemoveItem)
	tradeRoutes.POST("/purchase-orders/:id/confirm", purchaseOrderHandler.Confirm)
	tradeRoutes.POST("/purchase-orders/:id/receive", purchaseOrderHandler.Receive)
	tradeRoutes.POST("/purchase-orders/:id/cancel", purchaseOrderHandler.Cancel)

	// Finance domain
	financeRoutes := router.NewDomainGroup("finance", "/finance")
	financeRoutes.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "finance service ready"})
	})

	// Register all domain groups
	r.Register(catalogRoutes).
		Register(partnerRoutes).
		Register(inventoryRoutes).
		Register(tradeRoutes).
		Register(financeRoutes)

	// Register system routes with swagger-documented handlers
	systemHandler := handler.NewSystemHandler()
	systemRoutes := router.NewDomainGroup("system", "/system")
	systemRoutes.GET("/info", systemHandler.GetSystemInfo)
	systemRoutes.GET("/ping", systemHandler.Ping)
	r.Register(systemRoutes)

	// Setup routes
	r.Setup()

	// Also keep a simple ping at root API level for basic health checks
	engine.GET("/api/v1/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	// Create HTTP server with config
	srv := &http.Server{
		Addr:           ":" + cfg.App.Port,
		Handler:        engine,
		ReadTimeout:    cfg.HTTP.ReadTimeout,
		WriteTimeout:   cfg.HTTP.WriteTimeout,
		IdleTimeout:    cfg.HTTP.IdleTimeout,
		MaxHeaderBytes: cfg.HTTP.MaxHeaderBytes,
	}

	// Start server in goroutine
	go func() {
		log.Info("Server starting", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown", zap.Error(err))
	}

	log.Info("Server exited gracefully")
}

// healthHandler returns a handler for health check endpoints
func healthHandler(db *persistence.Database, _ *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqLog := logger.GetGinLogger(c)
		if err := db.Ping(); err != nil {
			reqLog.Warn("Health check failed", zap.Error(err))
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":   "unhealthy",
				"time":     time.Now().Format(time.RFC3339),
				"database": "error",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status":   "healthy",
			"time":     time.Now().Format(time.RFC3339),
			"database": "ok",
		})
	}
}
