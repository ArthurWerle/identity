package main

import (
	"context"
	"fmt"
	"identity/internal/config"
	"identity/internal/handler"
	"identity/internal/middleware"
	"identity/internal/migrations"
	"identity/internal/repository"
	"identity/internal/service"
	"identity/internal/service/dto"
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

	// Run migrations
	if err := migrations.RunMigrations(db, logger); err != nil {
		logger.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Setup repositories
	userRepo := repository.NewUserRepository(db)
	featureFlagRepo := repository.NewFeatureFlagRepository(db)
	userFFRepo := repository.NewUserFeatureFlagRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	auditLogRepo := repository.NewAuditLogRepository(db)

	// Setup services
	auditLogger := service.NewAuditLogger(auditLogRepo, logger)
	userService := service.NewUserService(userRepo, featureFlagRepo, userFFRepo, auditLogger)
	featureFlagService := service.NewFeatureFlagService(featureFlagRepo, userFFRepo, auditLogger)
	sessionDuration := time.Duration(cfg.Auth.SessionDurationHours) * time.Hour
	authService := service.NewAuthService(userRepo, sessionRepo, auditLogger, sessionDuration)

	// Seed the initial admin user (first boot only)
	if err := seedAdminUser(cfg, userRepo, authService, logger); err != nil {
		logger.Error("failed to seed admin user", "error", err)
		os.Exit(1)
	}

	// Setup handlers
	userHandler := handler.NewUserHandler(userService, logger)
	featureFlagHandler := handler.NewFeatureFlagHandler(featureFlagService, logger)
	authHandler := handler.NewAuthHandler(authService, logger, cfg.Auth.CookieSecure)
	webHandler := handler.NewWebHandler(authService, userService, featureFlagService, auditLogRepo, logger, cfg.Auth.CookieSecure, cfg.Environment)

	// Setup HTTP server
	router := setupRouter(cfg, logger, userHandler, featureFlagHandler, authHandler, webHandler, authService)

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

// seedAdminUser creates the initial admin account on first boot when the
// users table is empty and ADMIN_EMAIL/ADMIN_PASSWORD are configured.
func seedAdminUser(cfg *config.Config, userRepo repository.UserRepository, authService service.AuthService, logger *slog.Logger) error {
	if cfg.Admin.Email == "" || cfg.Admin.Password == "" {
		return nil
	}

	ctx := context.Background()
	_, total, err := userRepo.GetAll(ctx, 1, 0)
	if err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}
	if total > 0 {
		return nil
	}

	_, err = authService.Register(ctx, &dto.RegisterRequest{
		Name:     "Admin",
		Email:    cfg.Admin.Email,
		Password: cfg.Admin.Password,
	})
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	logger.Info("admin user seeded", "email", cfg.Admin.Email)
	return nil
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

func setupRouter(
	cfg *config.Config,
	logger *slog.Logger,
	userHandler *handler.UserHandler,
	featureFlagHandler *handler.FeatureFlagHandler,
	authHandler *handler.AuthHandler,
	webHandler *handler.WebHandler,
	authService service.AuthService,
) *gin.Engine {
	// Set gin mode
	if cfg.Log.Level != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Middleware
	router.Use(middleware.Recovery(logger))
	router.Use(middleware.Logger(logger))

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
		// Auth routes (public: login/logout/validate/me self-validate the
		// session they are given)
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", authHandler.Logout)
			auth.GET("/me", authHandler.Me)
			auth.POST("/validate", authHandler.ValidateSession)
		}

		// Flag check is public within the docker network so other services
		// can evaluate flags without a user session
		v1.GET("/feature-flags/check", featureFlagHandler.CheckFeatureFlag)

		// Minimal user lookup (id, name) is public within the docker network so
		// other services can resolve a user's display name without a user
		// session (e.g. the transactions service labelling a transaction's
		// creator). Only non-sensitive fields are returned.
		v1.GET("/internal/users/:id", userHandler.GetPublicUser)

		// Everything below requires a valid session (cookie or X-Session-ID)
		authed := v1.Group("")
		authed.Use(middleware.Auth(authService, logger))
		{
			users := authed.Group("/users")
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

			featureFlags := authed.Group("/feature-flags")
			{
				featureFlags.POST("", featureFlagHandler.CreateFeatureFlag)
				featureFlags.GET("", featureFlagHandler.GetFeatureFlags)
				featureFlags.GET("/:id", featureFlagHandler.GetFeatureFlag)
				featureFlags.PUT("/:id", featureFlagHandler.UpdateFeatureFlag)
				featureFlags.DELETE("/:id", featureFlagHandler.DeleteFeatureFlag)
			}
		}
	}

	// Web admin interface routes
	admin := router.Group("/admin")
	{
		// Public routes
		admin.GET("/login", webHandler.LoginPage)
		admin.POST("/login", webHandler.LoginSubmit)
		admin.GET("/logout", webHandler.Logout)

		// Protected routes
		protected := admin.Group("")
		protected.Use(middleware.WebAuth(authService, logger))
		{
			protected.GET("", webHandler.Dashboard)
			protected.GET("/flags", webHandler.FlagsTab)
			protected.GET("/users", webHandler.UsersTab)
			protected.POST("/flags", webHandler.CreateFlag)
			protected.PUT("/flags/:id/toggle", webHandler.ToggleFlag)
			protected.DELETE("/flags/:id", webHandler.DeleteFlag)
			protected.GET("/users/:id/flags", webHandler.UserFlags)
			protected.POST("/users/:id/flags/:key/toggle", webHandler.ToggleUserFlag)
			protected.POST("/users", webHandler.CreateUser)
			protected.GET("/users/:id/edit", webHandler.EditUserModal)
			protected.PUT("/users/:id", webHandler.UpdateUser)
			protected.DELETE("/users/:id", webHandler.DeleteUser)
			protected.POST("/users/:id/force-logout", webHandler.ForceLogoutUser)
			protected.GET("/audit", webHandler.AuditTab)
		}
	}

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return router
}
