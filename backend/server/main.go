package main

import (
	"context"
	"evently/api/routes"
	"evently/internal/shared/config"
	"evently/internal/shared/database"
	"evently/pkg/cache"
	"fmt"
	"log"
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
	startTime = time.Now()
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Load config
	cfg := config.Load()

	// Set Gin mode (debug/release)
	gin.SetMode(cfg.GinMode)

	// Initialize Redis Cache
	if err := initializeRedis(cfg); err != nil {
		log.Printf("Warning: Redis initialization failed: %v", err)
		log.Println("Continuing without Redis cache...")
	} else {
		log.Println("✅ Redis cache initialized successfully")
		defer func() {
			if err := cache.Close(); err != nil {
				log.Printf("Error closing Redis connection: %v", err)
			}
		}()
	}

	// Initialize DB
	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Setup router
	router := setupRouter(cfg, db)

	// HTTP server
	srv := &http.Server{
		Addr:    cfg.GetServerAddress(),
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		fmt.Printf("🚀 Server running at http://localhost:%s\n", cfg.Port)
		fmt.Printf("📊 Health Check: http://localhost:%s/health\n", cfg.Port)
		fmt.Printf("📋 API Status: http://localhost:%s%s/status\n", cfg.Port, cfg.GetAPIBasePath())
		fmt.Printf("🔍 API Version: %s\n", cfg.APIVersion)
		fmt.Printf("🗄️  Redis Cache: %v\n", cache.IsInitialized())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v\n", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Forced shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}

func setupRouter(cfg *config.Config, db *database.DB) *gin.Engine {
	engine := gin.New()

	// Built-in middleware: logs requests + recovers from panics
	engine.Use(gin.Logger(), gin.Recovery())

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

// initializeRedis sets up Redis cache using the application config
func initializeRedis(cfg *config.Config) error {
	// Convert app config to cache config
	redisConfig := cache.RedisConfig{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		Addr:     cfg.Redis.Addr,
	}

	// Initialize Redis
	if err := cache.InitWithRedisConfig(redisConfig); err != nil {
		return fmt.Errorf("failed to initialize Redis: %w", err)
	}

	// Test connection
	if err := cache.Ping(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	return nil
}
