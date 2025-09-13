package main

import (
	"context"
	"evently/api/routes"
	"evently/internal/notifications"
	"evently/internal/shared/config"
	"evently/internal/shared/database"
	"evently/pkg/cache"
	"evently/pkg/logger"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	// Load environment variables
	appLogger := logger.GetDefault()
	if err := godotenv.Load(); err != nil {
		appLogger.Info("No .env file found, using system environment variables")
	}

	// Load config
	cfg := config.Load()

	// Set Gin mode (debug/release)
	gin.SetMode(cfg.GinMode)

	// Initialize DB
	db, err := database.InitDB(cfg)
	if err != nil {
		appLogger.Error("Failed to connect to database:", err)
	}
	defer db.Close()

	// Initialize Unified Notification Service
	notificationCtx, notificationCancel := context.WithCancel(context.Background())
	defer notificationCancel()

	// Create unified notification service
	notificationService, err := notifications.NewUnifiedNotificationService(nil) // Uses env config by default
	if err != nil {
		appLogger.Error("Failed to initialize notification service: %v", err)
		appLogger.Info("Continuing without notification service - notifications will not be processed")
	} else {
		// Start the unified notification service
		go func() {
			if err := notificationService.Start(notificationCtx); err != nil {
				appLogger.Error(" Failed to start notification service: %v", err)
			}
		}()

		appLogger.Info("Unified notification service initialized and started")

		// Ensure notification service is stopped on shutdown
		defer func() {
			appLogger.Info("Stopping notification service...")
			if err := notificationService.Stop(); err != nil {
				appLogger.Error("Error stopping notification service: %v", err)
			}
		}()
	}

	// Setup router
	router := setupRouter(cfg, db)

	// HTTP server
	srv := &http.Server{
		Addr:    cfg.GetServerAddress(),
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		appLogger.Info("🚀 Server running",
			slog.String("address", cfg.GetServerAddress()),
			slog.String("health_check", fmt.Sprintf("http://localhost:%s/health", cfg.Port)),
			slog.String("api_status", fmt.Sprintf("http://localhost:%s%s/status", cfg.Port, cfg.GetAPIBasePath())),
			slog.String("version", cfg.APIVersion),
			slog.Bool("redis_cache", cache.IsInitialized()),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Error("Server failed: %v\n", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	appLogger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		appLogger.Error("Forced shutdown: %v", err)
	}

	appLogger.Info("Server exited gracefully")
}

func setupRouter(cfg *config.Config, db *database.DB) *gin.Engine {
	engine := gin.New()
	appLogger := logger.GetDefault()

	// Built-in middleware: logs requests + recovers from panics
	engine.Use(RequestLoggerMiddleware(appLogger), gin.Recovery())

	// CORS configuration
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Configure based on your needs
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Initialize and setup routes
	appRouter := routes.NewRouter(cfg, db)
	appRouter.SetupRoutes(engine)

	return engine
}

func RequestLoggerMiddleware(l *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		l.LogHTTPRequest(c, duration)
	}
}
