package main

import (
	"context"
	"evently/api/routes"
	"evently/internal/notifications"
	"evently/internal/seats"
	"evently/internal/shared/config"
	"evently/internal/shared/database"
	"evently/pkg/logger"
	"evently/pkg/ratelimit"
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

	// Smart environment loading
	if err := godotenv.Load(); err != nil {
		// Check if we're in production/container mode
		if os.Getenv("GIN_MODE") == "release" || os.Getenv("DOCKER_CONTAINER") == "true" {
			appLogger.Info("Production environment: using container environment variables")
		} else {
			appLogger.Info("No .env file found, using system environment variables")
		}
	} else {
		appLogger.Info("Development environment: loaded .env file")
	}

	// Load config
	cfg := config.Load()

	// Set Gin mode (debug/release)
	gin.SetMode(cfg.GinMode)

	// Initialize DB
	db, err := database.InitDB(cfg)
	if err != nil {
		appLogger.Error("failed to connect:", slog.Any("error", err))
	}
	defer db.Close()

	// Initialize Redis Lua scripts for atomic operations (critical for concurrency)
	if db.Redis != nil {
		atomicRedis := seats.NewAtomicRedisOperations(db.Redis)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := atomicRedis.PreloadScripts(ctx); err != nil {
			appLogger.Error("Failed to preload Redis Lua scripts", slog.Any("error", err))
			// Continue without failing - scripts will be loaded on first use
		} else {
			appLogger.Info("✅ Redis Lua scripts preloaded for atomic seat operations")
		}
		cancel()
	}

	// Initialize Rate Limiter
	var rateLimiter *ratelimit.RateLimiter
	if cfg.RateLimit.Enabled {
		rateLimiterConfig := &ratelimit.Config{
			Enabled:                 cfg.RateLimit.Enabled,
			WindowDuration:          cfg.RateLimit.WindowDuration,
			DefaultRequests:         cfg.RateLimit.DefaultRequests,
			PublicRequests:          cfg.RateLimit.PublicRequests,
			AuthRequests:            cfg.RateLimit.AuthRequests,
			BookingRequests:         cfg.RateLimit.BookingRequests,
			AdminRequests:           cfg.RateLimit.AdminRequests,
			AnalyticsRequests:       cfg.RateLimit.AnalyticsRequests,
			WhitelistedIPs:          cfg.RateLimit.WhitelistedIPs,
			BookingCriticalRequests: cfg.RateLimit.BookingCriticalRequests,
			UserRequests:            cfg.RateLimit.UserRequests,
			HealthRequests:          cfg.RateLimit.HealthRequests,
		}

		rateLimiter = ratelimit.NewRateLimiter(db.GetRedis(), rateLimiterConfig)
		appLogger.Info("Rate limiter initialized",
			slog.Bool("enabled", cfg.RateLimit.Enabled),
			slog.Duration("window", cfg.RateLimit.WindowDuration),
			slog.Int("default_requests", cfg.RateLimit.DefaultRequests),
		)
	} else {
		appLogger.Info("Rate limiting disabled")
	}

	// Initialize Unified Notification Service
	notificationCtx, notificationCancel := context.WithCancel(context.Background())
	defer notificationCancel()

	// Create unified notification service
	notificationService, err := notifications.NewUnifiedNotificationService(nil) // Uses env config by default
	if err != nil {
		appLogger.Error("Failed to initialize notification service", slog.Any("error", err))
		appLogger.Info("Continuing without notification service - notifications will not be processed")
	} else {
		// Start the unified notification service
		go func() {
			if err := notificationService.Start(notificationCtx); err != nil {
				appLogger.Error("Failed to start notification service", slog.Any("error", err))
			}
		}()

		appLogger.Info("Unified notification service initialized and started")

		// Ensure notification service is stopped on shutdown
		defer func() {
			appLogger.Info("Stopping notification service...")
			if err := notificationService.Stop(); err != nil {
				appLogger.Error("Error stopping notification service", slog.Any("error", err))
			}
		}()
	}

	// Setup router with rate limiter
	router := setupRouter(cfg, db, rateLimiter)

	// HTTP server
	srv := &http.Server{
		Addr:           cfg.GetServerAddress(),
		Handler:        router,
		ReadTimeout:    cfg.ReadTimeout,
		WriteTimeout:   cfg.WriteTimeout,
		IdleTimeout:    cfg.IdleTimeout,
		MaxHeaderBytes: cfg.MaxHeaderBytes,
	}

	// Start server in goroutine
	go func() {
		appLogger.Info("🚀 Server running",
			slog.String("address", cfg.GetServerAddress()),
			slog.String("health_check", fmt.Sprintf("http://localhost:%s/health", cfg.Port)),
			slog.String("api_status", fmt.Sprintf("http://localhost:%s%s/status", cfg.Port, cfg.GetAPIBasePath())),
			slog.String("version", cfg.APIVersion),
			slog.Bool("redis_cache", (db.Redis != nil)),
			slog.Bool("rate_limiting", cfg.RateLimit.Enabled),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Error("Server failed", slog.Any("error", err))
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
		appLogger.Error("Forced shutdown", slog.Any("error", err))
	}

	appLogger.Info("Server exited gracefully")
}

func setupRouter(cfg *config.Config, db *database.DB, rateLimiter *ratelimit.RateLimiter) *gin.Engine {
	engine := gin.New()
	appLogger := logger.GetDefault()

	// Built-in middleware: logs requests + recovers from panics
	engine.Use(RequestLoggerMiddleware(appLogger), gin.Recovery())

	// CORS configuration
	engine.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			return true // allow every origin dynamically
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-RateLimit-*"},
		ExposeHeaders:    []string{"Content-Length", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Global rate limiting middleware (applied to all routes)
	if rateLimiter != nil {
		engine.Use(ratelimit.Middleware(rateLimiter))
		appLogger.Info("Rate limiting middleware applied to all routes")
	}

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
