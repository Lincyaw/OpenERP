package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	catalogapp "github.com/erp/backend/internal/application/catalog"
	financeapp "github.com/erp/backend/internal/application/finance"
	identityapp "github.com/erp/backend/internal/application/identity"
	inventoryapp "github.com/erp/backend/internal/application/inventory"
	partnerapp "github.com/erp/backend/internal/application/partner"
	reportapp "github.com/erp/backend/internal/application/report"
	tradeapp "github.com/erp/backend/internal/application/trade"
	"github.com/erp/backend/internal/infrastructure/auth"
	"github.com/erp/backend/internal/infrastructure/config"
	"github.com/erp/backend/internal/infrastructure/event"
	"github.com/erp/backend/internal/infrastructure/logger"
	"github.com/erp/backend/internal/infrastructure/persistence"
	"github.com/erp/backend/internal/infrastructure/scheduler"
	"github.com/erp/backend/internal/interfaces/http/handler"
	"github.com/erp/backend/internal/interfaces/http/middleware"
	"github.com/erp/backend/internal/interfaces/http/router"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	_ "github.com/erp/backend/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

//	@title			ERP Backend API
//	@version		1.0
//	@description	进销存系统后端 API - 基于 DDD 设计的库存管理系统
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.url	https://github.com/erp/backend
//	@contact.email	support@erp.example.com

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

//	@host		localhost:8080
//	@BasePath	/api/v1

//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Bearer token authentication. Format: "Bearer {token}"

//	@externalDocs.description	OpenAPI
//	@externalDocs.url			https://swagger.io/resources/open-api/

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
	productUnitRepo := persistence.NewGormProductUnitRepository(db.DB)
	categoryRepo := persistence.NewGormCategoryRepository(db.DB)
	customerRepo := persistence.NewGormCustomerRepository(db.DB)
	supplierRepo := persistence.NewGormSupplierRepository(db.DB)
	warehouseRepo := persistence.NewGormWarehouseRepository(db.DB)
	balanceTransactionRepo := persistence.NewGormBalanceTransactionRepository(db.DB)
	inventoryItemRepo := persistence.NewGormInventoryItemRepository(db.DB)
	stockBatchRepo := persistence.NewGormStockBatchRepository(db.DB)
	stockLockRepo := persistence.NewGormStockLockRepository(db.DB)
	inventoryTxRepo := persistence.NewGormInventoryTransactionRepository(db.DB)
	salesOrderRepo := persistence.NewGormSalesOrderRepository(db.DB)
	purchaseOrderRepo := persistence.NewGormPurchaseOrderRepository(db.DB)
	salesReturnRepo := persistence.NewGormSalesReturnRepository(db.DB)
	purchaseReturnRepo := persistence.NewGormPurchaseReturnRepository(db.DB)
	stockTakingRepo := persistence.NewGormStockTakingRepository(db.DB)
	userRepo := persistence.NewGormUserRepository(db.DB)
	roleRepo := persistence.NewGormRoleRepository(db.DB)
	tenantRepo := persistence.NewGormTenantRepository(db.DB)
	salesReportRepo := persistence.NewGormSalesReportRepository(db.DB)
	inventoryReportRepo := persistence.NewGormInventoryReportRepository(db.DB)
	financeReportRepo := persistence.NewGormFinanceReportRepository(db.DB)
	reportCacheRepo := reportapp.NewGormReportCacheRepository(db.DB)
	receiptVoucherRepo := persistence.NewGormReceiptVoucherRepository(db.DB)
	paymentVoucherRepo := persistence.NewGormPaymentVoucherRepository(db.DB)
	accountReceivableRepo := persistence.NewGormAccountReceivableRepository(db.DB)
	accountPayableRepo := persistence.NewGormAccountPayableRepository(db.DB)
	expenseRecordRepo := persistence.NewGormExpenseRecordRepository(db.DB)
	otherIncomeRecordRepo := persistence.NewGormOtherIncomeRecordRepository(db.DB)
	outboxRepo := event.NewGormOutboxRepository(db.DB)

	// Initialize event serializer and register all event types
	eventSerializer := event.NewEventSerializer()
	event.RegisterAllEvents(eventSerializer)

	// Create outbox publisher for transactional event saving
	outboxPublisher := event.NewOutboxPublisher(eventSerializer)

	// Inject outbox publisher into repositories that need transactional event publishing
	salesOrderRepo.SetOutboxEventSaver(outboxPublisher)
	purchaseOrderRepo.SetOutboxEventSaver(outboxPublisher)

	// Initialize application services
	productService := catalogapp.NewProductService(productRepo, categoryRepo)
	productUnitService := catalogapp.NewProductUnitService(productRepo, productUnitRepo)
	categoryService := catalogapp.NewCategoryService(categoryRepo, productRepo)
	customerService := partnerapp.NewCustomerService(customerRepo)
	supplierService := partnerapp.NewSupplierService(supplierRepo)
	warehouseService := partnerapp.NewWarehouseService(warehouseRepo)
	balanceTransactionService := partnerapp.NewBalanceTransactionService(balanceTransactionRepo, customerRepo)
	inventoryService := inventoryapp.NewInventoryService(inventoryItemRepo, stockBatchRepo, stockLockRepo, inventoryTxRepo)
	salesOrderService := tradeapp.NewSalesOrderService(salesOrderRepo)
	purchaseOrderService := tradeapp.NewPurchaseOrderService(purchaseOrderRepo)
	salesReturnService := tradeapp.NewSalesReturnService(salesReturnRepo, salesOrderRepo)
	purchaseReturnService := tradeapp.NewPurchaseReturnService(purchaseReturnRepo, purchaseOrderRepo)
	stockTakingService := inventoryapp.NewStockTakingService(stockTakingRepo, nil) // eventBus will be set later

	// Identity services (auth, user, role, tenant)
	jwtService := auth.NewJWTService(cfg.JWT)
	authService := identityapp.NewAuthService(userRepo, roleRepo, jwtService, identityapp.DefaultAuthServiceConfig(), log)
	userService := identityapp.NewUserService(userRepo, roleRepo, log)
	roleService := identityapp.NewRoleService(roleRepo, userRepo, log)
	tenantService := identityapp.NewTenantService(tenantRepo, log)

	// Report services
	reportService := reportapp.NewReportService(salesReportRepo, inventoryReportRepo, financeReportRepo)
	reportAggregationService := reportapp.NewReportAggregationService(
		salesReportRepo, inventoryReportRepo, financeReportRepo, reportCacheRepo, log,
	)

	// Payment callback service (for external payment gateway notifications)
	// Note: Payment gateways (WeChat, Alipay) are registered at runtime via config
	paymentCallbackService := financeapp.NewPaymentCallbackService(financeapp.PaymentCallbackServiceConfig{
		Gateways:           nil, // Gateways will be registered via RegisterGateway()
		ReceiptVoucherRepo: receiptVoucherRepo,
		ReceivableRepo:     accountReceivableRepo,
		EventPublisher:     nil, // Will be set after event bus init
		Logger:             log,
	})

	// Expense and income service
	expenseIncomeService := financeapp.NewExpenseIncomeService(expenseRecordRepo, otherIncomeRecordRepo)

	// Finance core service (receivables, payables, vouchers)
	financeService := financeapp.NewFinanceService(
		accountReceivableRepo,
		accountPayableRepo,
		receiptVoucherRepo,
		paymentVoucherRepo,
	)

	// Initialize event bus and handlers
	eventBus := event.NewInMemoryEventBus(log)

	// Register event handlers for cross-context integration
	// Purchase order receiving -> inventory increase
	purchaseOrderReceivedHandler := tradeapp.NewPurchaseOrderReceivedHandler(inventoryService, log)
	eventBus.Subscribe(purchaseOrderReceivedHandler)

	// Sales order confirmed -> stock locking
	salesOrderConfirmedHandler := tradeapp.NewSalesOrderConfirmedHandler(inventoryService, log)
	eventBus.Subscribe(salesOrderConfirmedHandler)

	// Sales order shipped -> stock deduction
	salesOrderShippedHandler := tradeapp.NewSalesOrderShippedHandler(inventoryService, log)
	eventBus.Subscribe(salesOrderShippedHandler)

	// Sales order cancelled -> stock unlock
	salesOrderCancelledHandler := tradeapp.NewSalesOrderCancelledHandler(inventoryService, log)
	eventBus.Subscribe(salesOrderCancelledHandler)

	// Sales return completed -> inventory restoration
	salesReturnCompletedHandler := tradeapp.NewSalesReturnCompletedHandler(inventoryService, log)
	eventBus.Subscribe(salesReturnCompletedHandler)

	// Purchase return shipped -> inventory deduction
	purchaseReturnShippedHandler := tradeapp.NewPurchaseReturnShippedHandler(inventoryService, log)
	eventBus.Subscribe(purchaseReturnShippedHandler)

	log.Info("Event handlers registered",
		zap.Strings("purchase_order_received_events", purchaseOrderReceivedHandler.EventTypes()),
		zap.Strings("sales_order_confirmed_events", salesOrderConfirmedHandler.EventTypes()),
		zap.Strings("sales_order_shipped_events", salesOrderShippedHandler.EventTypes()),
		zap.Strings("sales_order_cancelled_events", salesOrderCancelledHandler.EventTypes()),
		zap.Strings("sales_return_completed_events", salesReturnCompletedHandler.EventTypes()),
		zap.Strings("purchase_return_shipped_events", purchaseReturnShippedHandler.EventTypes()),
	)

	// Start event bus
	if err := eventBus.Start(context.Background()); err != nil {
		log.Fatal("Failed to start event bus", zap.Error(err))
	}
	defer func() {
		if err := eventBus.Stop(context.Background()); err != nil {
			log.Error("Error stopping event bus", zap.Error(err))
		}
	}()

	// Initialize and start outbox processor for guaranteed event delivery
	// The outbox processor reads events from the outbox_events table and publishes them to the event bus
	outboxProcessorConfig := event.DefaultOutboxProcessorConfig()
	outboxProcessor := event.NewOutboxProcessor(outboxRepo, eventBus, eventSerializer, outboxProcessorConfig, log)
	if err := outboxProcessor.Start(context.Background()); err != nil {
		log.Fatal("Failed to start outbox processor", zap.Error(err))
	}
	defer func() {
		if err := outboxProcessor.Stop(context.Background()); err != nil {
			log.Error("Error stopping outbox processor", zap.Error(err))
		}
	}()
	log.Info("Outbox processor started",
		zap.Int("batch_size", outboxProcessorConfig.BatchSize),
		zap.Duration("poll_interval", outboxProcessorConfig.PollInterval),
	)

	// Inject event bus into services that publish events
	purchaseOrderService.SetEventPublisher(eventBus)
	salesOrderService.SetEventPublisher(eventBus)
	salesReturnService.SetEventPublisher(eventBus)
	purchaseReturnService.SetEventPublisher(eventBus)

	// Initialize report scheduler (if enabled)
	if cfg.Scheduler.Enabled {
		schedulerConfig := scheduler.SchedulerConfig{
			Enabled:           cfg.Scheduler.Enabled,
			MaxConcurrentJobs: cfg.Scheduler.MaxConcurrentJobs,
			JobTimeout:        cfg.Scheduler.JobTimeout,
			RetryAttempts:     cfg.Scheduler.RetryAttempts,
			RetryDelay:        cfg.Scheduler.RetryDelay,
		}
		reportScheduler := scheduler.NewScheduler(schedulerConfig, reportAggregationService, log)
		if err := reportScheduler.Start(context.Background()); err != nil {
			log.Fatal("Failed to start report scheduler", zap.Error(err))
		}
		defer func() {
			if err := reportScheduler.Stop(context.Background()); err != nil {
				log.Error("Error stopping report scheduler", zap.Error(err))
			}
		}()
		log.Info("Report scheduler started",
			zap.Int("max_concurrent_jobs", cfg.Scheduler.MaxConcurrentJobs),
			zap.Duration("job_timeout", cfg.Scheduler.JobTimeout),
		)
	}

	// Initialize HTTP handlers
	productHandler := handler.NewProductHandler(productService)
	productUnitHandler := handler.NewProductUnitHandler(productUnitService)
	categoryHandler := handler.NewCategoryHandler(categoryService)
	customerHandler := handler.NewCustomerHandler(customerService)
	supplierHandler := handler.NewSupplierHandler(supplierService)
	warehouseHandler := handler.NewWarehouseHandler(warehouseService)
	balanceTransactionHandler := handler.NewBalanceTransactionHandler(balanceTransactionService)
	inventoryHandler := handler.NewInventoryHandler(inventoryService)
	salesOrderHandler := handler.NewSalesOrderHandler(salesOrderService)
	purchaseOrderHandler := handler.NewPurchaseOrderHandler(purchaseOrderService)
	salesReturnHandler := handler.NewSalesReturnHandler(salesReturnService)
	purchaseReturnHandler := handler.NewPurchaseReturnHandler(purchaseReturnService)
	stockTakingHandler := handler.NewStockTakingHandler(stockTakingService)
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	roleHandler := handler.NewRoleHandler(roleService)
	tenantHandler := handler.NewTenantHandler(tenantService)
	reportHandler := handler.NewReportHandler(reportService)
	reportHandler.SetAggregationService(reportAggregationService)
	paymentCallbackHandler := handler.NewPaymentCallbackHandler(paymentCallbackService)
	expenseIncomeHandler := handler.NewExpenseIncomeHandler(expenseIncomeService)
	financeHandler := handler.NewFinanceHandler(financeService)

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

	// Payment gateway callback endpoints (no authentication required)
	// These endpoints are called directly by external payment gateways
	paymentCallbackGroup := engine.Group("/api/v1/payment/callback")
	paymentCallbackGroup.POST("/wechat", paymentCallbackHandler.HandleWechatPaymentCallback)
	paymentCallbackGroup.POST("/wechat/refund", paymentCallbackHandler.HandleWechatRefundCallback)
	paymentCallbackGroup.POST("/alipay", paymentCallbackHandler.HandleAlipayPaymentCallback)
	paymentCallbackGroup.POST("/alipay/refund", paymentCallbackHandler.HandleAlipayRefundCallback)

	// Setup API routes using router
	r := router.NewRouter(engine, router.WithAPIVersion("v1"))

	// Apply JWT authentication middleware to API routes
	// Configure skip paths for public endpoints
	jwtConfig := middleware.JWTMiddlewareConfig{
		JWTService: jwtService,
		SkipPaths: []string{
			"/api/v1/auth/login",
			"/api/v1/auth/refresh",
			"/api/v1/ping",
			"/api/v1/system/ping",
			"/api/v1/system/info",
		},
		SkipPathPrefixes: []string{
			"/api/v1/payment/callback",
		},
		Logger: log,
	}
	r.Use(middleware.JWTAuthMiddlewareWithConfig(jwtConfig))

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
	// Product unit routes
	catalogRoutes.POST("/products/:id/units", productUnitHandler.Create)
	catalogRoutes.GET("/products/:id/units", productUnitHandler.List)
	catalogRoutes.GET("/products/:id/units/convert", productUnitHandler.Convert)
	catalogRoutes.GET("/products/:id/units/default-purchase", productUnitHandler.GetDefaultPurchaseUnit)
	catalogRoutes.GET("/products/:id/units/default-sales", productUnitHandler.GetDefaultSalesUnit)
	catalogRoutes.GET("/products/:id/units/:unit_id", productUnitHandler.GetByID)
	catalogRoutes.PUT("/products/:id/units/:unit_id", productUnitHandler.Update)
	catalogRoutes.DELETE("/products/:id/units/:unit_id", productUnitHandler.Delete)
	// Products by category
	catalogRoutes.GET("/categories/:id/products", productHandler.GetByCategory)

	// Category routes
	catalogRoutes.POST("/categories", categoryHandler.Create)
	catalogRoutes.GET("/categories", categoryHandler.List)
	catalogRoutes.GET("/categories/tree", categoryHandler.GetTree)
	catalogRoutes.GET("/categories/roots", categoryHandler.GetRoots)
	catalogRoutes.GET("/categories/:id", categoryHandler.GetByID)
	catalogRoutes.GET("/categories/:id/children", categoryHandler.GetChildren)
	catalogRoutes.PUT("/categories/:id", categoryHandler.Update)
	catalogRoutes.POST("/categories/:id/move", categoryHandler.Move)
	catalogRoutes.POST("/categories/:id/activate", categoryHandler.Activate)
	catalogRoutes.POST("/categories/:id/deactivate", categoryHandler.Deactivate)
	catalogRoutes.DELETE("/categories/:id", categoryHandler.Delete)

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

	// Balance transaction routes (customer balance with transaction records)
	partnerRoutes.POST("/customers/:id/balance/recharge", balanceTransactionHandler.Recharge)
	partnerRoutes.POST("/customers/:id/balance/adjust", balanceTransactionHandler.Adjust)
	partnerRoutes.GET("/customers/:id/balance", balanceTransactionHandler.GetBalance)
	partnerRoutes.GET("/customers/:id/balance/summary", balanceTransactionHandler.GetBalanceSummary)
	partnerRoutes.GET("/customers/:id/balance/transactions", balanceTransactionHandler.ListTransactions)
	partnerRoutes.GET("/balance/transactions/:id", balanceTransactionHandler.GetTransaction)

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

	// Stock Taking routes
	inventoryRoutes.POST("/stock-takings", stockTakingHandler.Create)
	inventoryRoutes.GET("/stock-takings", stockTakingHandler.List)
	inventoryRoutes.GET("/stock-takings/pending-approval", stockTakingHandler.ListPendingApproval)
	inventoryRoutes.GET("/stock-takings/by-number/:taking_number", stockTakingHandler.GetByTakingNumber)
	inventoryRoutes.GET("/stock-takings/:id", stockTakingHandler.GetByID)
	inventoryRoutes.GET("/stock-takings/:id/progress", stockTakingHandler.GetProgress)
	inventoryRoutes.PUT("/stock-takings/:id", stockTakingHandler.Update)
	inventoryRoutes.DELETE("/stock-takings/:id", stockTakingHandler.Delete)
	inventoryRoutes.POST("/stock-takings/:id/items", stockTakingHandler.AddItem)
	inventoryRoutes.POST("/stock-takings/:id/items/bulk", stockTakingHandler.AddItems)
	inventoryRoutes.DELETE("/stock-takings/:id/items/:product_id", stockTakingHandler.RemoveItem)
	inventoryRoutes.POST("/stock-takings/:id/start", stockTakingHandler.StartCounting)
	inventoryRoutes.POST("/stock-takings/:id/count", stockTakingHandler.RecordCount)
	inventoryRoutes.POST("/stock-takings/:id/counts", stockTakingHandler.RecordCounts)
	inventoryRoutes.POST("/stock-takings/:id/submit", stockTakingHandler.SubmitForApproval)
	inventoryRoutes.POST("/stock-takings/:id/approve", stockTakingHandler.Approve)
	inventoryRoutes.POST("/stock-takings/:id/reject", stockTakingHandler.Reject)
	inventoryRoutes.POST("/stock-takings/:id/cancel", stockTakingHandler.Cancel)

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

	// Sales Return routes
	tradeRoutes.POST("/sales-returns", salesReturnHandler.Create)
	tradeRoutes.GET("/sales-returns", salesReturnHandler.List)
	tradeRoutes.GET("/sales-returns/stats/summary", salesReturnHandler.GetStatusSummary)
	tradeRoutes.GET("/sales-returns/number/:return_number", salesReturnHandler.GetByReturnNumber)
	tradeRoutes.GET("/sales-returns/:id", salesReturnHandler.GetByID)
	tradeRoutes.PUT("/sales-returns/:id", salesReturnHandler.Update)
	tradeRoutes.DELETE("/sales-returns/:id", salesReturnHandler.Delete)
	tradeRoutes.POST("/sales-returns/:id/items", salesReturnHandler.AddItem)
	tradeRoutes.PUT("/sales-returns/:id/items/:item_id", salesReturnHandler.UpdateItem)
	tradeRoutes.DELETE("/sales-returns/:id/items/:item_id", salesReturnHandler.RemoveItem)
	tradeRoutes.POST("/sales-returns/:id/submit", salesReturnHandler.Submit)
	tradeRoutes.POST("/sales-returns/:id/approve", salesReturnHandler.Approve)
	tradeRoutes.POST("/sales-returns/:id/reject", salesReturnHandler.Reject)
	tradeRoutes.POST("/sales-returns/:id/receive", salesReturnHandler.Receive)
	tradeRoutes.POST("/sales-returns/:id/complete", salesReturnHandler.Complete)
	tradeRoutes.POST("/sales-returns/:id/cancel", salesReturnHandler.Cancel)

	// Purchase Return routes
	tradeRoutes.POST("/purchase-returns", purchaseReturnHandler.Create)
	tradeRoutes.GET("/purchase-returns", purchaseReturnHandler.List)
	tradeRoutes.GET("/purchase-returns/stats/summary", purchaseReturnHandler.GetStatusSummary)
	tradeRoutes.GET("/purchase-returns/number/:return_number", purchaseReturnHandler.GetByReturnNumber)
	tradeRoutes.GET("/purchase-returns/:id", purchaseReturnHandler.GetByID)
	tradeRoutes.PUT("/purchase-returns/:id", purchaseReturnHandler.Update)
	tradeRoutes.DELETE("/purchase-returns/:id", purchaseReturnHandler.Delete)
	tradeRoutes.POST("/purchase-returns/:id/items", purchaseReturnHandler.AddItem)
	tradeRoutes.PUT("/purchase-returns/:id/items/:item_id", purchaseReturnHandler.UpdateItem)
	tradeRoutes.DELETE("/purchase-returns/:id/items/:item_id", purchaseReturnHandler.RemoveItem)
	tradeRoutes.POST("/purchase-returns/:id/submit", purchaseReturnHandler.Submit)
	tradeRoutes.POST("/purchase-returns/:id/approve", purchaseReturnHandler.Approve)
	tradeRoutes.POST("/purchase-returns/:id/reject", purchaseReturnHandler.Reject)
	tradeRoutes.POST("/purchase-returns/:id/ship", purchaseReturnHandler.Ship)
	tradeRoutes.POST("/purchase-returns/:id/complete", purchaseReturnHandler.Complete)
	tradeRoutes.POST("/purchase-returns/:id/cancel", purchaseReturnHandler.Cancel)

	// Finance domain
	financeRoutes := router.NewDomainGroup("finance", "/finance")
	financeRoutes.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "finance service ready"})
	})

	// Expense routes
	financeRoutes.GET("/expenses", expenseIncomeHandler.ListExpenses)
	financeRoutes.GET("/expenses/summary", expenseIncomeHandler.GetExpensesSummary)
	financeRoutes.GET("/expenses/:id", expenseIncomeHandler.GetExpense)
	financeRoutes.POST("/expenses", expenseIncomeHandler.CreateExpense)
	financeRoutes.PUT("/expenses/:id", expenseIncomeHandler.UpdateExpense)
	financeRoutes.DELETE("/expenses/:id", expenseIncomeHandler.DeleteExpense)
	financeRoutes.POST("/expenses/:id/submit", expenseIncomeHandler.SubmitExpense)
	financeRoutes.POST("/expenses/:id/approve", expenseIncomeHandler.ApproveExpense)
	financeRoutes.POST("/expenses/:id/reject", expenseIncomeHandler.RejectExpense)
	financeRoutes.POST("/expenses/:id/cancel", expenseIncomeHandler.CancelExpense)
	financeRoutes.POST("/expenses/:id/pay", expenseIncomeHandler.MarkExpensePaid)

	// Other income routes
	financeRoutes.GET("/incomes", expenseIncomeHandler.ListIncomes)
	financeRoutes.GET("/incomes/summary", expenseIncomeHandler.GetIncomesSummary)
	financeRoutes.GET("/incomes/:id", expenseIncomeHandler.GetIncome)
	financeRoutes.POST("/incomes", expenseIncomeHandler.CreateIncome)
	financeRoutes.PUT("/incomes/:id", expenseIncomeHandler.UpdateIncome)
	financeRoutes.DELETE("/incomes/:id", expenseIncomeHandler.DeleteIncome)
	financeRoutes.POST("/incomes/:id/confirm", expenseIncomeHandler.ConfirmIncome)
	financeRoutes.POST("/incomes/:id/cancel", expenseIncomeHandler.CancelIncome)
	financeRoutes.POST("/incomes/:id/receive", expenseIncomeHandler.MarkIncomeReceived)

	// Cash flow route
	financeRoutes.GET("/cash-flow", expenseIncomeHandler.GetCashFlow)

	// Account Receivable routes
	financeRoutes.GET("/receivables", financeHandler.ListReceivables)
	financeRoutes.GET("/receivables/summary", financeHandler.GetReceivableSummary)
	financeRoutes.GET("/receivables/:id", financeHandler.GetReceivableByID)

	// Account Payable routes
	financeRoutes.GET("/payables", financeHandler.ListPayables)
	financeRoutes.GET("/payables/summary", financeHandler.GetPayableSummary)
	financeRoutes.GET("/payables/:id", financeHandler.GetPayableByID)

	// Receipt Voucher routes (收款单)
	financeRoutes.GET("/receipts", financeHandler.ListReceiptVouchers)
	financeRoutes.GET("/receipts/:id", financeHandler.GetReceiptVoucherByID)
	financeRoutes.POST("/receipts", financeHandler.CreateReceiptVoucher)
	financeRoutes.POST("/receipts/:id/confirm", financeHandler.ConfirmReceiptVoucher)
	financeRoutes.POST("/receipts/:id/cancel", financeHandler.CancelReceiptVoucher)
	financeRoutes.POST("/receipts/:id/reconcile", financeHandler.ReconcileReceiptVoucher)

	// Payment Voucher routes (付款单)
	financeRoutes.GET("/payments", financeHandler.ListPaymentVouchers)
	financeRoutes.GET("/payments/:id", financeHandler.GetPaymentVoucherByID)
	financeRoutes.POST("/payments", financeHandler.CreatePaymentVoucher)
	financeRoutes.POST("/payments/:id/confirm", financeHandler.ConfirmPaymentVoucher)
	financeRoutes.POST("/payments/:id/cancel", financeHandler.CancelPaymentVoucher)
	financeRoutes.POST("/payments/:id/reconcile", financeHandler.ReconcilePaymentVoucher)

	// Report domain
	reportRoutes := router.NewDomainGroup("report", "/reports")
	reportRoutes.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "report service ready"})
	})
	// Sales reports
	reportRoutes.GET("/sales/summary", reportHandler.GetSalesSummary)
	reportRoutes.GET("/sales/daily-trend", reportHandler.GetDailySalesTrend)
	reportRoutes.GET("/sales/products/ranking", reportHandler.GetProductSalesRanking)
	reportRoutes.GET("/sales/customers/ranking", reportHandler.GetCustomerSalesRanking)
	// Inventory reports
	reportRoutes.GET("/inventory/summary", reportHandler.GetInventorySummary)
	reportRoutes.GET("/inventory/turnover", reportHandler.GetInventoryTurnover)
	reportRoutes.GET("/inventory/value-by-category", reportHandler.GetInventoryValueByCategory)
	reportRoutes.GET("/inventory/value-by-warehouse", reportHandler.GetInventoryValueByWarehouse)
	reportRoutes.GET("/inventory/slow-moving", reportHandler.GetSlowMovingProducts)
	// Finance reports
	reportRoutes.GET("/finance/profit-loss", reportHandler.GetProfitLossStatement)
	reportRoutes.GET("/finance/monthly-trend", reportHandler.GetMonthlyProfitTrend)
	reportRoutes.GET("/finance/profit-by-product", reportHandler.GetProfitByProduct)
	reportRoutes.GET("/finance/cash-flow", reportHandler.GetCashFlowStatement)
	reportRoutes.GET("/finance/cash-flow/items", reportHandler.GetCashFlowItems)
	// Report aggregation/refresh endpoints
	reportRoutes.POST("/refresh", reportHandler.RefreshReport)
	reportRoutes.POST("/refresh/all", reportHandler.RefreshAllReports)
	reportRoutes.GET("/scheduler/status", reportHandler.GetSchedulerStatus)

	// Identity domain (authentication, users, roles) - public routes
	authRoutes := router.NewDomainGroup("auth", "/auth")
	authRoutes.POST("/login", authHandler.Login)
	authRoutes.POST("/refresh", authHandler.RefreshToken)

	// Identity domain - protected routes
	identityRoutes := router.NewDomainGroup("identity", "/identity")
	identityRoutes.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "identity service ready"})
	})

	// Auth routes requiring authentication
	identityRoutes.POST("/auth/logout", authHandler.Logout)
	identityRoutes.GET("/auth/me", authHandler.GetCurrentUser)
	identityRoutes.PUT("/auth/password", authHandler.ChangePassword)

	// User management routes
	identityRoutes.POST("/users", userHandler.Create)
	identityRoutes.GET("/users", userHandler.List)
	identityRoutes.GET("/users/stats/count", userHandler.Count)
	identityRoutes.GET("/users/:id", userHandler.GetByID)
	identityRoutes.PUT("/users/:id", userHandler.Update)
	identityRoutes.DELETE("/users/:id", userHandler.Delete)
	identityRoutes.POST("/users/:id/activate", userHandler.Activate)
	identityRoutes.POST("/users/:id/deactivate", userHandler.Deactivate)
	identityRoutes.POST("/users/:id/lock", userHandler.Lock)
	identityRoutes.POST("/users/:id/unlock", userHandler.Unlock)
	identityRoutes.POST("/users/:id/reset-password", userHandler.ResetPassword)
	identityRoutes.PUT("/users/:id/roles", userHandler.AssignRoles)

	// Role management routes
	identityRoutes.POST("/roles", roleHandler.Create)
	identityRoutes.GET("/roles", roleHandler.List)
	identityRoutes.GET("/roles/system", roleHandler.GetSystemRoles)
	identityRoutes.GET("/roles/stats/count", roleHandler.Count)
	identityRoutes.GET("/roles/:id", roleHandler.GetByID)
	identityRoutes.GET("/roles/code/:code", roleHandler.GetByCode)
	identityRoutes.PUT("/roles/:id", roleHandler.Update)
	identityRoutes.DELETE("/roles/:id", roleHandler.Delete)
	identityRoutes.POST("/roles/:id/enable", roleHandler.Enable)
	identityRoutes.POST("/roles/:id/disable", roleHandler.Disable)
	identityRoutes.PUT("/roles/:id/permissions", roleHandler.SetPermissions)

	// Permission management
	identityRoutes.GET("/permissions", roleHandler.GetPermissions)

	// Tenant management routes
	identityRoutes.POST("/tenants", tenantHandler.Create)
	identityRoutes.GET("/tenants", tenantHandler.List)
	identityRoutes.GET("/tenants/stats", tenantHandler.GetStats)
	identityRoutes.GET("/tenants/stats/count", tenantHandler.Count)
	identityRoutes.GET("/tenants/:id", tenantHandler.GetByID)
	identityRoutes.GET("/tenants/code/:code", tenantHandler.GetByCode)
	identityRoutes.PUT("/tenants/:id", tenantHandler.Update)
	identityRoutes.PUT("/tenants/:id/config", tenantHandler.UpdateConfig)
	identityRoutes.PUT("/tenants/:id/plan", tenantHandler.SetPlan)
	identityRoutes.DELETE("/tenants/:id", tenantHandler.Delete)
	identityRoutes.POST("/tenants/:id/activate", tenantHandler.Activate)
	identityRoutes.POST("/tenants/:id/deactivate", tenantHandler.Deactivate)
	identityRoutes.POST("/tenants/:id/suspend", tenantHandler.Suspend)

	// Register all domain groups
	r.Register(catalogRoutes).
		Register(partnerRoutes).
		Register(inventoryRoutes).
		Register(tradeRoutes).
		Register(financeRoutes).
		Register(reportRoutes).
		Register(authRoutes).
		Register(identityRoutes)

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
