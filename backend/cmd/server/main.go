package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	catalogapp "github.com/erp/backend/internal/application/catalog"
	eventapp "github.com/erp/backend/internal/application/event"
	featureflagapp "github.com/erp/backend/internal/application/featureflag"
	financeapp "github.com/erp/backend/internal/application/finance"
	identityapp "github.com/erp/backend/internal/application/identity"
	inventoryapp "github.com/erp/backend/internal/application/inventory"
	partnerapp "github.com/erp/backend/internal/application/partner"
	reportapp "github.com/erp/backend/internal/application/report"
	tradeapp "github.com/erp/backend/internal/application/trade"
	financedomain "github.com/erp/backend/internal/domain/finance"
	domainStrategy "github.com/erp/backend/internal/domain/shared/strategy"
	"github.com/erp/backend/internal/infrastructure/auth"
	"github.com/erp/backend/internal/infrastructure/config"
	"github.com/erp/backend/internal/infrastructure/event"
	"github.com/erp/backend/internal/infrastructure/logger"
	"github.com/erp/backend/internal/infrastructure/persistence"
	infraPlugin "github.com/erp/backend/internal/infrastructure/plugin"
	"github.com/erp/backend/internal/infrastructure/scheduler"
	infraStrategy "github.com/erp/backend/internal/infrastructure/strategy"
	"github.com/erp/backend/internal/infrastructure/telemetry"
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

	// Initialize OpenTelemetry TracerProvider
	tracerProvider, err := telemetry.NewTracerProvider(context.Background(), telemetry.Config{
		Enabled:           cfg.Telemetry.Enabled,
		CollectorEndpoint: cfg.Telemetry.CollectorEndpoint,
		SamplingRatio:     cfg.Telemetry.SamplingRatio,
		ServiceName:       cfg.Telemetry.ServiceName,
		Insecure:          cfg.Telemetry.Insecure,
	}, log)
	if err != nil {
		log.Fatal("Failed to initialize OpenTelemetry TracerProvider", zap.Error(err))
	}
	defer func() {
		if err := tracerProvider.Shutdown(context.Background()); err != nil {
			log.Error("Error shutting down tracer provider", zap.Error(err))
		}
	}()

	// Initialize OpenTelemetry MeterProvider for metrics
	meterProvider, err := telemetry.NewMeterProvider(context.Background(), telemetry.MetricsConfig{
		Enabled:           cfg.Telemetry.Enabled,
		CollectorEndpoint: cfg.Telemetry.CollectorEndpoint,
		ExportInterval:    cfg.Telemetry.MetricsExportInterval,
		ServiceName:       cfg.Telemetry.ServiceName,
		Insecure:          cfg.Telemetry.Insecure,
	}, log)
	if err != nil {
		log.Fatal("Failed to initialize OpenTelemetry MeterProvider", zap.Error(err))
	}
	defer func() {
		if err := meterProvider.Shutdown(context.Background()); err != nil {
			log.Error("Error shutting down meter provider", zap.Error(err))
		}
	}()

	// Initialize OpenTelemetry LoggerProvider for logs bridge (Zap -> OTEL)
	// This enables exporting Zap logs to OTEL Collector alongside traces and metrics
	logsProvider, err := telemetry.NewLoggerProvider(context.Background(), telemetry.LogsConfig{
		Enabled:           cfg.Telemetry.Enabled && cfg.Telemetry.LogsEnabled,
		CollectorEndpoint: cfg.Telemetry.CollectorEndpoint,
		ServiceName:       cfg.Telemetry.ServiceName,
		Insecure:          cfg.Telemetry.Insecure,
	}, log)
	if err != nil {
		log.Fatal("Failed to initialize OpenTelemetry LoggerProvider", zap.Error(err))
	}
	defer func() {
		if err := logsProvider.Shutdown(context.Background()); err != nil {
			log.Error("Error shutting down logs provider", zap.Error(err))
		}
	}()

	// Bridge existing Zap logger to OTEL if logs export is enabled
	// This creates a combined logger that outputs to both stdout and OTEL Collector
	if logsProvider.IsEnabled() {
		otelCore := telemetry.NewZapOTELCore(telemetry.ZapBridgeConfig{
			ServiceName:    cfg.Telemetry.ServiceName,
			LoggerProvider: logsProvider,
			Level:          logger.ParseLevel(cfg.Log.Level),
		})
		log = telemetry.NewBridgedLogger(log.Core(), otelCore,
			zap.AddCaller(),
			zap.AddStacktrace(logger.ParseLevel("error")),
		)
		log.Info("Zap -> OTEL logs bridge enabled",
			zap.String("collector", cfg.Telemetry.CollectorEndpoint),
		)
	}

	// Initialize Pyroscope continuous profiler
	// Note: Unlike DB metrics which is non-fatal on failure, profiler initialization
	// failure is fatal because if profiling is enabled in config but fails to start,
	// it indicates a critical configuration or connection issue that should be fixed.
	// When profiling is disabled (default), NewProfiler returns a no-op profiler without error.
	profiler, err := telemetry.NewProfiler(telemetry.ProfilerConfig{
		Enabled:              cfg.Telemetry.Profiling.Enabled,
		ServerAddress:        cfg.Telemetry.Profiling.ServerAddress,
		ApplicationName:      cfg.Telemetry.Profiling.ApplicationName,
		BasicAuthUser:        cfg.Telemetry.Profiling.BasicAuthUser,
		BasicAuthPassword:    cfg.Telemetry.Profiling.BasicAuthPassword,
		ProfileCPU:           cfg.Telemetry.Profiling.ProfileCPU,
		ProfileAllocObjects:  cfg.Telemetry.Profiling.ProfileAllocObjects,
		ProfileAllocSpace:    cfg.Telemetry.Profiling.ProfileAllocSpace,
		ProfileInuseObjects:  cfg.Telemetry.Profiling.ProfileInuseObjects,
		ProfileInuseSpace:    cfg.Telemetry.Profiling.ProfileInuseSpace,
		ProfileGoroutines:    cfg.Telemetry.Profiling.ProfileGoroutines,
		ProfileMutexCount:    cfg.Telemetry.Profiling.ProfileMutexCount,
		ProfileMutexDuration: cfg.Telemetry.Profiling.ProfileMutexDuration,
		ProfileBlockCount:    cfg.Telemetry.Profiling.ProfileBlockCount,
		ProfileBlockDuration: cfg.Telemetry.Profiling.ProfileBlockDuration,
		MutexProfileFraction: cfg.Telemetry.Profiling.MutexProfileFraction,
		BlockProfileRate:     cfg.Telemetry.Profiling.BlockProfileRate,
		DisableGCRuns:        cfg.Telemetry.Profiling.DisableGCRuns,
	}, log)
	if err != nil {
		log.Fatal("Failed to initialize Pyroscope profiler", zap.Error(err))
	}
	defer func() {
		if err := profiler.Stop(); err != nil {
			log.Error("Error stopping profiler", zap.Error(err))
		}
	}()

	// Enable span profiles integration if configured
	// This wraps the TracerProvider with Pyroscope's otelpyroscope wrapper,
	// allowing CPU profiles to be associated with individual trace spans.
	// IMPORTANT: This must be done AFTER the profiler is started.
	if cfg.Telemetry.Profiling.SpanProfilesEnabled && profiler.IsEnabled() && tracerProvider.IsEnabled() {
		if err := tracerProvider.EnableSpanProfiles(); err != nil {
			log.Error("Failed to enable span profiles", zap.Error(err))
		} else {
			log.Info("Span profiles integration enabled",
				zap.String("note", "CPU profiles will be associated with trace spans"),
				zap.String("limitation", "Only CPU profiling supported; spans <10ms may not have profile data"),
			)
		}
	}

	// Log telemetry status
	if cfg.Telemetry.Enabled {
		log.Info("OpenTelemetry initialized",
			zap.Bool("tracing", tracerProvider.IsEnabled()),
			zap.Bool("metrics", meterProvider.IsEnabled()),
			zap.Bool("logs", logsProvider.IsEnabled()),
			zap.Bool("profiling", profiler.IsEnabled()),
			zap.Bool("span_profiles", tracerProvider.IsSpanProfilesEnabled()),
			zap.String("collector", cfg.Telemetry.CollectorEndpoint),
		)
	}

	// Create GORM logger backed by zap
	gormLogLevel := logger.MapGormLogLevel(cfg.Log.Level)
	gormLog := logger.NewGormLogger(log, gormLogLevel)

	// Initialize database connection with custom logger and tracing
	db, err := persistence.NewDatabaseWithTracing(&cfg.Database, gormLog, &cfg.Telemetry, log)
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("Error closing database", zap.Error(err))
		}
	}()

	// Initialize database metrics collection (connection pool and query metrics)
	dbMetrics, err := telemetry.RegisterDBMetrics(db.DB, meterProvider, telemetry.DBMetricsConfig{
		Enabled:            cfg.Telemetry.Enabled,
		SlowQueryThreshold: cfg.Telemetry.DBSlowQueryThresh,
		// PoolStatsInterval uses default (15s)
	}, log)
	if err != nil {
		log.Error("Failed to initialize database metrics", zap.Error(err))
		// Non-fatal: continue without metrics
	}
	if dbMetrics != nil {
		// Start connection pool stats collection
		dbMetrics.StartPoolStatsCollection(context.Background())
		defer dbMetrics.Stop()
	}

	log.Info("Database connected successfully",
		zap.Bool("tracing_enabled", cfg.Telemetry.DBTraceEnabled),
	)

	// Initialize repositories
	productRepo := persistence.NewGormProductRepository(db.DB)
	productUnitRepo := persistence.NewGormProductUnitRepository(db.DB)
	categoryRepo := persistence.NewGormCategoryRepository(db.DB)
	customerRepo := persistence.NewGormCustomerRepository(db.DB)
	customerLevelRepo := persistence.NewGormCustomerLevelRepository(db.DB)
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

	// Feature flag repositories
	featureFlagRepo := persistence.NewGormFeatureFlagRepository(db.DB)
	flagOverrideRepo := persistence.NewGormFlagOverrideRepository(db.DB)
	flagAuditLogRepo := persistence.NewGormFlagAuditLogRepository(db.DB)

	// Initialize event serializer and register all event types
	eventSerializer := event.NewEventSerializer()
	event.RegisterAllEvents(eventSerializer)

	// Create outbox publisher for transactional event saving
	outboxPublisher := event.NewOutboxPublisher(eventSerializer)

	// Inject outbox publisher into repositories that need transactional event publishing
	salesOrderRepo.SetOutboxEventSaver(outboxPublisher)
	purchaseOrderRepo.SetOutboxEventSaver(outboxPublisher)

	// Initialize strategy registry with default strategies
	strategyRegistry, err := infraStrategy.NewRegistryWithDefaults()
	if err != nil {
		log.Fatal("Failed to initialize strategy registry", zap.Error(err))
	}
	log.Info("Strategy registry initialized",
		zap.Int("cost_strategies", strategyRegistry.Stats()[domainStrategy.StrategyTypeCost]),
		zap.Int("allocation_strategies", strategyRegistry.Stats()[domainStrategy.StrategyTypeAllocation]),
		zap.Int("pricing_strategies", strategyRegistry.Stats()[domainStrategy.StrategyTypePricing]),
	)

	// Initialize plugin manager for industry-specific extensions
	pluginRegistryAdapter := infraPlugin.NewStrategyRegistryAdapter(strategyRegistry)
	pluginManager := infraPlugin.NewPluginManager(pluginRegistryAdapter)

	// Register industry plugins
	// Agricultural plugin provides validation for pesticides, seeds, fertilizers
	agriculturalPlugin := infraPlugin.NewAgriculturalPlugin()
	if err := pluginManager.Register(agriculturalPlugin); err != nil {
		log.Error("Failed to register agricultural plugin", zap.Error(err))
	} else {
		log.Info("Industry plugin registered",
			zap.String("plugin", agriculturalPlugin.Name()),
			zap.String("display_name", agriculturalPlugin.DisplayName()),
			zap.Int("required_attributes", len(agriculturalPlugin.GetRequiredProductAttributes())),
		)
	}

	log.Info("Plugin manager initialized",
		zap.Int("total_plugins", pluginManager.Count()),
		zap.Strings("plugins", pluginManager.ListPlugins()),
	)

	// Initialize application services
	productService := catalogapp.NewProductService(productRepo, categoryRepo, strategyRegistry)
	productService.SetSalesOrderRepo(salesOrderRepo)
	productService.SetPurchaseOrderRepo(purchaseOrderRepo)
	productService.SetInventoryRepo(inventoryItemRepo)
	productUnitService := catalogapp.NewProductUnitService(productRepo, productUnitRepo)
	categoryService := catalogapp.NewCategoryService(categoryRepo, productRepo)
	customerService := partnerapp.NewCustomerService(customerRepo)
	customerService.SetAccountReceivableRepo(accountReceivableRepo)
	customerService.SetSalesOrderRepo(salesOrderRepo)
	customerLevelService := partnerapp.NewCustomerLevelService(customerLevelRepo)
	supplierService := partnerapp.NewSupplierService(supplierRepo)
	supplierService.SetAccountPayableRepo(accountPayableRepo)
	supplierService.SetPurchaseOrderRepo(purchaseOrderRepo)
	warehouseService := partnerapp.NewWarehouseService(warehouseRepo, inventoryItemRepo)
	balanceTransactionService := partnerapp.NewBalanceTransactionService(balanceTransactionRepo, customerRepo)
	inventoryService := inventoryapp.NewInventoryService(inventoryItemRepo, stockBatchRepo, stockLockRepo, inventoryTxRepo)
	stockLockExpirationService := inventoryapp.NewStockLockExpirationService(stockLockRepo, inventoryItemRepo, nil, log) // eventBus will be set later
	salesOrderService := tradeapp.NewSalesOrderService(salesOrderRepo)
	purchaseOrderService := tradeapp.NewPurchaseOrderService(purchaseOrderRepo)
	salesReturnService := tradeapp.NewSalesReturnService(salesReturnRepo, salesOrderRepo)
	purchaseReturnService := tradeapp.NewPurchaseReturnService(purchaseReturnRepo, purchaseOrderRepo)
	stockTakingService := inventoryapp.NewStockTakingService(stockTakingRepo, nil) // eventBus will be set later

	// Identity services (auth, user, role, tenant)
	jwtService := auth.NewJWTService(cfg.JWT)
	authService := identityapp.NewAuthService(userRepo, roleRepo, jwtService, identityapp.DefaultAuthServiceConfig(), log)

	// Initialize token blacklist for secure logout and session invalidation
	// This uses Redis to store blacklisted token JTIs and user invalidation timestamps
	var tokenBlacklist auth.TokenBlacklist
	tokenBlacklistCfg := auth.RedisTokenBlacklistConfig{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}
	redisBlacklist, err := auth.NewRedisTokenBlacklist(tokenBlacklistCfg)
	if err != nil {
		log.Warn("Failed to initialize Redis token blacklist, using in-memory fallback",
			zap.Error(err),
			zap.String("note", "In-memory blacklist does not persist across restarts and does not work in multi-instance deployments"))
		tokenBlacklist = auth.NewInMemoryTokenBlacklist()
	} else {
		tokenBlacklist = redisBlacklist
		log.Info("Token blacklist initialized with Redis",
			zap.String("host", cfg.Redis.Host),
			zap.Int("port", cfg.Redis.Port))
	}

	// Set token blacklist on auth service for logout and password change handling
	authService.SetTokenBlacklist(tokenBlacklist)

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
	// Configure with FIFO as default reconciliation strategy (injected from strategy registry)
	financeService := financeapp.NewFinanceService(
		accountReceivableRepo,
		accountPayableRepo,
		receiptVoucherRepo,
		paymentVoucherRepo,
		financeapp.WithReconciliationStrategy(financedomain.ReconciliationStrategyTypeFIFO),
	)
	// Log strategy configuration
	log.Info("Finance service configured",
		zap.String("default_reconciliation_strategy", financeService.GetReconciliationService().GetDefaultStrategy().String()),
	)
	// Keep reference to strategy registry for potential future use (tenant-specific strategies)
	_ = strategyRegistry

	// Feature flag services
	flagService := featureflagapp.NewFlagService(
		featureFlagRepo,
		flagAuditLogRepo,
		outboxRepo,
		log,
	)
	evaluationService := featureflagapp.NewEvaluationService(
		featureFlagRepo,
		flagOverrideRepo,
		log,
	)
	overrideService := featureflagapp.NewOverrideService(
		featureFlagRepo,
		flagOverrideRepo,
		flagAuditLogRepo,
		outboxRepo,
		log,
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

	// Sales return cancelled -> inventory reversal (if goods were received)
	salesReturnCancelledHandler := tradeapp.NewSalesReturnCancelledHandler(inventoryService, log)
	eventBus.Subscribe(salesReturnCancelledHandler)

	// Purchase return shipped -> inventory deduction
	purchaseReturnShippedHandler := tradeapp.NewPurchaseReturnShippedHandler(inventoryService, log)
	eventBus.Subscribe(purchaseReturnShippedHandler)

	// Stock below threshold -> notifications/alerts
	stockBelowThresholdNotifier := inventoryapp.NewLoggingStockAlertNotifier(log)
	stockBelowThresholdHandler := inventoryapp.NewStockBelowThresholdHandler(log).
		WithNotifier(stockBelowThresholdNotifier)
	eventBus.Subscribe(stockBelowThresholdHandler)

	log.Info("Event handlers registered",
		zap.Strings("purchase_order_received_events", purchaseOrderReceivedHandler.EventTypes()),
		zap.Strings("sales_order_confirmed_events", salesOrderConfirmedHandler.EventTypes()),
		zap.Strings("sales_order_shipped_events", salesOrderShippedHandler.EventTypes()),
		zap.Strings("sales_order_cancelled_events", salesOrderCancelledHandler.EventTypes()),
		zap.Strings("sales_return_completed_events", salesReturnCompletedHandler.EventTypes()),
		zap.Strings("sales_return_cancelled_events", salesReturnCancelledHandler.EventTypes()),
		zap.Strings("purchase_return_shipped_events", purchaseReturnShippedHandler.EventTypes()),
		zap.Strings("stock_below_threshold_events", stockBelowThresholdHandler.EventTypes()),
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

	// Initialize outbox service for dead letter queue management
	outboxService := eventapp.NewOutboxService(outboxRepo, log)

	// Inject event bus into services that publish events
	purchaseOrderService.SetEventPublisher(eventBus)
	salesOrderService.SetEventPublisher(eventBus)
	salesReturnService.SetEventPublisher(eventBus)
	purchaseReturnService.SetEventPublisher(eventBus)
	stockLockExpirationService.SetEventBus(eventBus)
	inventoryService.SetEventPublisher(eventBus)

	// Inject pricing strategy provider into sales order service
	salesOrderService.SetPricingProvider(strategyRegistry)

	// Initialize report cron scheduler (if enabled)
	// This runs daily report aggregation at the configured cron time (default: 2 AM)
	var reportCronScheduler *scheduler.ReportCronScheduler
	if cfg.Scheduler.Enabled {
		// Parse cron schedule
		cronHour, cronMinute, _ := scheduler.ParseCronSchedule(cfg.Scheduler.DailyCronSchedule)

		cronConfig := scheduler.ReportCronSchedulerConfig{
			Enabled:           cfg.Scheduler.Enabled,
			CronHour:          cronHour,
			CronMinute:        cronMinute,
			DailyCronSchedule: cfg.Scheduler.DailyCronSchedule,
			MaxConcurrentJobs: cfg.Scheduler.MaxConcurrentJobs,
			JobTimeout:        cfg.Scheduler.JobTimeout,
			RetryAttempts:     cfg.Scheduler.RetryAttempts,
			RetryDelay:        cfg.Scheduler.RetryDelay,
		}

		// Create scheduler job repository for persistence
		schedulerJobRepo := scheduler.NewSchedulerJobRepository(db.DB)

		// Create cron scheduler
		reportCronScheduler = scheduler.NewReportCronScheduler(
			cronConfig,
			reportAggregationService,
			tenantRepo,
			schedulerJobRepo,
			log,
		)

		if err := reportCronScheduler.Start(context.Background()); err != nil {
			log.Fatal("Failed to start report cron scheduler", zap.Error(err))
		}
		defer func() {
			if err := reportCronScheduler.Stop(context.Background()); err != nil {
				log.Error("Error stopping report cron scheduler", zap.Error(err))
			}
		}()
		log.Info("Report cron scheduler started",
			zap.String("cron_schedule", cfg.Scheduler.DailyCronSchedule),
			zap.Int("cron_hour", cronHour),
			zap.Int("cron_minute", cronMinute),
			zap.Int("max_concurrent_jobs", cfg.Scheduler.MaxConcurrentJobs),
			zap.Duration("job_timeout", cfg.Scheduler.JobTimeout),
		)
	}

	// Initialize stock lock expiration job (if enabled)
	var stopStockLockExpiration context.CancelFunc
	if cfg.StockLock.AutoReleaseEnabled {
		stockLockCtx, cancel := context.WithCancel(context.Background())
		stopStockLockExpiration = cancel
		go func() {
			ticker := time.NewTicker(cfg.StockLock.CheckInterval)
			defer ticker.Stop()

			log.Info("Stock lock expiration job started",
				zap.Duration("check_interval", cfg.StockLock.CheckInterval),
				zap.Duration("default_expiration", cfg.StockLock.DefaultExpiration),
			)

			// Run once immediately at startup
			stats, err := stockLockExpirationService.ReleaseExpiredLocks(stockLockCtx)
			if err != nil {
				log.Error("Failed to release expired stock locks on startup", zap.Error(err))
			} else if stats.TotalExpired > 0 {
				log.Info("Released expired stock locks on startup",
					zap.Int("released", stats.SuccessReleased),
					zap.Int("failed", stats.FailedReleases),
				)
			}

			for {
				select {
				case <-stockLockCtx.Done():
					log.Info("Stock lock expiration job stopped")
					return
				case <-ticker.C:
					stats, err := stockLockExpirationService.ReleaseExpiredLocks(stockLockCtx)
					if err != nil {
						log.Error("Failed to release expired stock locks", zap.Error(err))
					} else if stats.TotalExpired > 0 {
						log.Info("Released expired stock locks",
							zap.Int("released", stats.SuccessReleased),
							zap.Int("failed", stats.FailedReleases),
						)
					}
				}
			}
		}()
	}

	// Initialize HTTP handlers
	productHandler := handler.NewProductHandler(productService)
	productUnitHandler := handler.NewProductUnitHandler(productUnitService)
	categoryHandler := handler.NewCategoryHandler(categoryService)
	customerHandler := handler.NewCustomerHandler(customerService)
	customerLevelHandler := handler.NewCustomerLevelHandler(customerLevelService)
	supplierHandler := handler.NewSupplierHandler(supplierService)
	warehouseHandler := handler.NewWarehouseHandler(warehouseService)
	balanceTransactionHandler := handler.NewBalanceTransactionHandler(balanceTransactionService)
	inventoryHandler := handler.NewInventoryHandler(inventoryService)
	salesOrderHandler := handler.NewSalesOrderHandler(salesOrderService)
	purchaseOrderHandler := handler.NewPurchaseOrderHandler(purchaseOrderService)
	salesReturnHandler := handler.NewSalesReturnHandler(salesReturnService)
	purchaseReturnHandler := handler.NewPurchaseReturnHandler(purchaseReturnService)
	stockTakingHandler := handler.NewStockTakingHandler(stockTakingService)
	authHandler := handler.NewAuthHandler(authService, cfg.Cookie, cfg.JWT)
	userHandler := handler.NewUserHandler(userService)
	roleHandler := handler.NewRoleHandler(roleService)
	tenantHandler := handler.NewTenantHandler(tenantService)
	reportHandler := handler.NewReportHandler(reportService)
	reportHandler.SetAggregationService(reportAggregationService)
	if reportCronScheduler != nil {
		reportHandler.SetCronScheduler(reportCronScheduler)
	}
	paymentCallbackHandler := handler.NewPaymentCallbackHandler(paymentCallbackService)
	expenseIncomeHandler := handler.NewExpenseIncomeHandler(expenseIncomeService)
	financeHandler := handler.NewFinanceHandler(financeService)
	outboxHandler := handler.NewOutboxHandler(outboxService)
	featureFlagHandler := handler.NewFeatureFlagHandler(flagService, evaluationService, overrideService)

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
	// 1. Tracing - OpenTelemetry HTTP tracing (must be first to capture full request)
	// 2. RequestID - Generate/propagate request ID
	// 3. Recovery - Catch panics
	// 4. Logger - Log requests
	// 5. TracingAttributeInjector - Inject custom attributes into span
	// 6. SpanErrorMarker - Mark spans with error status for 4xx/5xx
	// 7. Security - Add security headers
	// 8. CORS - Handle cross-origin requests
	// 9. BodyLimit - Limit request body size
	// 10. RateLimit - Apply rate limiting (if enabled)
	engine.Use(middleware.TracingWithConfig(middleware.TracingConfig{
		ServiceName: cfg.Telemetry.ServiceName,
		Enabled:     cfg.Telemetry.Enabled,
	}))
	engine.Use(middleware.RequestID())
	engine.Use(logger.Recovery(log))
	engine.Use(logger.GinMiddleware(log))
	engine.Use(middleware.TracingAttributeInjector())
	engine.Use(middleware.SpanErrorMarker())
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

	// Swagger documentation endpoint with protection
	// In production, Swagger must be disabled, require auth, or have IP restrictions (SEC-007)
	swaggerGroup := engine.Group("/swagger")
	{
		// Create JWT middleware for Swagger authentication (if enabled)
		var swaggerJWTMiddleware gin.HandlerFunc
		if cfg.Swagger.RequireAuth {
			swaggerJWTMiddleware = middleware.JWTAuthMiddlewareWithConfig(middleware.JWTMiddlewareConfig{
				JWTService: jwtService,
				SkipPaths:  []string{}, // Don't skip any paths for Swagger auth
				Logger:     log,
			})
		}

		// Apply Swagger protection middleware
		swaggerCfg := middleware.SwaggerConfig{
			Enabled:     cfg.Swagger.Enabled,
			RequireAuth: cfg.Swagger.RequireAuth,
			AllowedIPs:  cfg.Swagger.AllowedIPs,
		}
		swaggerGroup.Use(middleware.SwaggerProtection(swaggerCfg, swaggerJWTMiddleware))
		swaggerGroup.GET("/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// Log Swagger configuration status
	if cfg.Swagger.Enabled {
		swaggerProtection := []string{}
		if cfg.Swagger.RequireAuth {
			swaggerProtection = append(swaggerProtection, "JWT auth required")
		}
		if len(cfg.Swagger.AllowedIPs) > 0 {
			swaggerProtection = append(swaggerProtection, "IP whitelist")
		}
		if len(swaggerProtection) == 0 {
			swaggerProtection = append(swaggerProtection, "unrestricted (development only)")
		}
		log.Info("Swagger documentation enabled",
			zap.Strings("protection", swaggerProtection),
		)
	} else {
		log.Info("Swagger documentation disabled")
	}

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
		JWTService:     jwtService,
		TokenBlacklist: tokenBlacklist,
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

	// Customer Level routes
	partnerRoutes.POST("/customer-levels", customerLevelHandler.Create)
	partnerRoutes.GET("/customer-levels", customerLevelHandler.List)
	partnerRoutes.GET("/customer-levels/default", customerLevelHandler.GetDefault)
	partnerRoutes.POST("/customer-levels/initialize", customerLevelHandler.InitializeDefaultLevels)
	partnerRoutes.GET("/customer-levels/:id", customerLevelHandler.GetByID)
	partnerRoutes.GET("/customer-levels/code/:code", customerLevelHandler.GetByCode)
	partnerRoutes.PUT("/customer-levels/:id", customerLevelHandler.Update)
	partnerRoutes.DELETE("/customer-levels/:id", customerLevelHandler.Delete)
	partnerRoutes.POST("/customer-levels/:id/set-default", customerLevelHandler.SetDefault)
	partnerRoutes.POST("/customer-levels/:id/activate", customerLevelHandler.Activate)
	partnerRoutes.POST("/customer-levels/:id/deactivate", customerLevelHandler.Deactivate)

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
	reportRoutes.POST("/scheduler/trigger", reportHandler.TriggerDailyAggregation)

	// Identity domain (authentication, users, roles) - public routes
	authRoutes := router.NewDomainGroup("auth", "/auth")

	// Apply auth-specific rate limiting (stricter limits for login/refresh to prevent brute force)
	if cfg.HTTP.AuthRateLimitEnabled {
		authRateLimiter := middleware.NewRateLimiter(cfg.HTTP.AuthRateLimitRequests, cfg.HTTP.AuthRateLimitWindow)
		authRoutes.Use(middleware.AuthRateLimit(authRateLimiter))
		log.Info("Auth rate limiting enabled",
			zap.Int("requests", cfg.HTTP.AuthRateLimitRequests),
			zap.Duration("window", cfg.HTTP.AuthRateLimitWindow),
		)
	}

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
	identityRoutes.POST("/auth/force-logout", middleware.RequirePermission("user:force_logout"), authHandler.ForceLogout)

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
	strategyHandler := handler.NewStrategyHandler(strategyRegistry)
	systemRoutes := router.NewDomainGroup("system", "/system")
	systemRoutes.GET("/info", systemHandler.GetSystemInfo)
	systemRoutes.GET("/ping", systemHandler.Ping)

	// Strategy routes (list available strategies)
	systemRoutes.GET("/strategies", strategyHandler.ListStrategies)
	systemRoutes.GET("/strategies/batch", strategyHandler.GetBatchStrategies)
	systemRoutes.GET("/strategies/cost", strategyHandler.GetCostStrategies)
	systemRoutes.GET("/strategies/pricing", strategyHandler.GetPricingStrategies)
	systemRoutes.GET("/strategies/allocation", strategyHandler.GetAllocationStrategies)

	// Outbox management routes (for operators)
	systemRoutes.GET("/outbox/stats", outboxHandler.GetStats)
	systemRoutes.GET("/outbox/dead", outboxHandler.GetDeadLetterEntries)
	systemRoutes.GET("/outbox/:id", outboxHandler.GetEntry)
	systemRoutes.POST("/outbox/:id/retry", outboxHandler.RetryDeadEntry)
	systemRoutes.POST("/outbox/dead/retry-all", outboxHandler.RetryAllDeadEntries)

	r.Register(systemRoutes)

	// Feature Flag domain - global resources for controlling application behavior
	featureFlagRoutes := router.NewDomainGroup("feature-flags", "/feature-flags")
	featureFlagRoutes.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "feature-flag service ready"})
	})

	// Feature flag management routes (admin access - requires feature_flag permissions)
	featureFlagRoutes.GET("", middleware.RequirePermission("feature_flag:read"), featureFlagHandler.ListFlags)
	featureFlagRoutes.POST("", middleware.RequirePermission("feature_flag:create"), featureFlagHandler.CreateFlag)
	featureFlagRoutes.GET("/:key", middleware.RequirePermission("feature_flag:read"), featureFlagHandler.GetFlag)
	featureFlagRoutes.PUT("/:key", middleware.RequirePermission("feature_flag:update"), featureFlagHandler.UpdateFlag)
	featureFlagRoutes.DELETE("/:key", middleware.RequirePermission("feature_flag:delete"), featureFlagHandler.ArchiveFlag)
	featureFlagRoutes.POST("/:key/enable", middleware.RequirePermission("feature_flag:update"), featureFlagHandler.EnableFlag)
	featureFlagRoutes.POST("/:key/disable", middleware.RequirePermission("feature_flag:update"), featureFlagHandler.DisableFlag)

	// Feature flag evaluation routes (requires feature_flag:evaluate permission)
	featureFlagRoutes.POST("/:key/evaluate", middleware.RequirePermission("feature_flag:evaluate"), featureFlagHandler.EvaluateFlag)
	featureFlagRoutes.POST("/evaluate-batch", middleware.RequirePermission("feature_flag:evaluate"), featureFlagHandler.BatchEvaluate)
	featureFlagRoutes.POST("/client-config", middleware.RequirePermission("feature_flag:evaluate"), featureFlagHandler.GetClientConfig)

	// Flag override management routes (admin access - requires feature_flag:override permission)
	featureFlagRoutes.GET("/:key/overrides", middleware.RequirePermission("feature_flag:read"), featureFlagHandler.ListOverrides)
	featureFlagRoutes.POST("/:key/overrides", middleware.RequirePermission("feature_flag:override"), featureFlagHandler.CreateOverride)
	featureFlagRoutes.DELETE("/:key/overrides/:id", middleware.RequirePermission("feature_flag:override"), featureFlagHandler.DeleteOverride)

	// Audit log routes (admin access - requires feature_flag:audit permission)
	featureFlagRoutes.GET("/:key/audit-logs", middleware.RequirePermission("feature_flag:audit"), featureFlagHandler.GetAuditLogs)

	r.Register(featureFlagRoutes)

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

	// Stop stock lock expiration job
	if stopStockLockExpiration != nil {
		stopStockLockExpiration()
	}

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
