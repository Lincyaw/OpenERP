package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	catalogapp "github.com/erp/backend/internal/application/catalog"
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

	// Initialize application services
	productService := catalogapp.NewProductService(productRepo, categoryRepo)

	// Initialize HTTP handlers
	productHandler := handler.NewProductHandler(productService)

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

	// Inventory domain
	inventoryRoutes := router.NewDomainGroup("inventory", "/inventory")
	inventoryRoutes.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "inventory service ready"})
	})

	// Trade domain (sales orders, purchase orders)
	tradeRoutes := router.NewDomainGroup("trade", "/trade")
	tradeRoutes.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "trade service ready"})
	})

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
