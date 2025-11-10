package main

import (
	"context"
	"fmt"
	"identity/internal/config"
	"identity/internal/handler"
	"identity/internal/middleware"
	"identity/internal/model"
	"identity/internal/repository"
	"identity/internal/service"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

// @title Identity Service API
// @version 1.0
// @description Identity service for managing users and feature flags
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@example.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /
// @schemes http https
func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	// Setup logger
	logger := setupLogger(cfg.Log.Level)
	logger.Info("starting identity service")

	// Setup database
	db, err := setupDatabase(cfg, logger)
	if err != nil {
		logger.Error("failed to setup database", "error", err)
		os.Exit(1)
	}

	// Auto migrate (optional, for development)
	if err := db.AutoMigrate(
		&model.User{},
		&model.FeatureFlag{},
		&model.UserFeatureFlag{},
	); err != nil {
		logger.Error("failed to auto migrate", "error", err)
		os.Exit(1)
	}

	// Setup repositories
	userRepo := repository.NewUserRepository(db)
	featureFlagRepo := repository.NewFeatureFlagRepository(db)
	userFFRepo := repository.NewUserFeatureFlagRepository(db)

	// Setup services
	userService := service.NewUserService(userRepo, featureFlagRepo, userFFRepo)
	featureFlagService := service.NewFeatureFlagService(featureFlagRepo)

	// Setup handlers
	userHandler := handler.NewUserHandler(userService, logger)
	featureFlagHandler := handler.NewFeatureFlagHandler(featureFlagService, logger)

	// Setup HTTP server
	router := setupRouter(cfg, logger, userHandler, featureFlagHandler)

	// Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Server.Port),
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("starting HTTP server", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	// Context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown server
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
	}

	logger.Info("server exited")
}

func setupLogger(level string) *slog.Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	return slog.New(handler)
}

func setupDatabase(cfg *config.Config, logger *slog.Logger) (*gorm.DB, error) {
	// GORM logger configuration
	gormLogLevel := gormLogger.Silent
	if cfg.Log.Level == "debug" {
		gormLogLevel = gormLogger.Info
	}

	gormCfg := &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogLevel),
	}

	// Connect to database
	db, err := gorm.Open(postgres.Open(cfg.Database.DSN()), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying SQL database
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	logger.Info("database connection established")
	return db, nil
}

func setupRouter(cfg *config.Config, logger *slog.Logger, userHandler *handler.UserHandler, featureFlagHandler *handler.FeatureFlagHandler) *gin.Engine {
	// Set gin mode
	if cfg.Log.Level != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Middleware
	router.Use(middleware.Recovery(logger))
	router.Use(middleware.Logger(logger))
	router.Use(middleware.CORS())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	})

	// API routes
	v1 := router.Group("/api/v1")
	{
		// User routes
		users := v1.Group("/users")
		{
			users.POST("", userHandler.CreateUser)
			users.GET("", userHandler.GetUsers)
			users.GET("/:id", userHandler.GetUser)
			users.PUT("/:id", userHandler.UpdateUser)
			users.DELETE("/:id", userHandler.DeleteUser)
			users.GET("/:id/feature-flags", userHandler.GetUserFeatureFlags)
			users.POST("/:id/feature-flags/:key", userHandler.AssignFeatureFlagToUser)
			users.DELETE("/:id/feature-flags/:key", userHandler.UnassignFeatureFlagFromUser)
		}

		// Feature flag routes
		featureFlags := v1.Group("/feature-flags")
		{
			featureFlags.POST("", featureFlagHandler.CreateFeatureFlag)
			featureFlags.GET("", featureFlagHandler.GetFeatureFlags)
			featureFlags.GET("/:id", featureFlagHandler.GetFeatureFlag)
			featureFlags.PUT("/:id", featureFlagHandler.UpdateFeatureFlag)
			featureFlags.DELETE("/:id", featureFlagHandler.DeleteFeatureFlag)
		}
	}

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return router
}
