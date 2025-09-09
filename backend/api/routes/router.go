// api/routes/router.go
package routes

import (
	"evently/internal/auth"
	"evently/internal/shared/config"
	"evently/internal/shared/database"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Router holds all route dependencies
type Router struct {
	config *config.Config
	db     *database.DB
}

// NewRouter creates a new router instance
func NewRouter(cfg *config.Config, db *database.DB) *Router {
	return &Router{
		config: cfg,
		db:     db,
	}
}

// SetupRoutes configures all application routes
func (r *Router) SetupRoutes(engine *gin.Engine) {
	// Health check and basic info endpoints
	r.setupHealthRoutes(engine)

	// API routes
	api := engine.Group(r.config.GetAPIBasePath())
	{
		// Setup auth routes
		r.setupAuthRoutes(api)
		
		// TODO: Add other route groups here
		// r.setupEventRoutes(api)
		// r.setupBookingRoutes(api)
		// r.setupAnalyticsRoutes(api)
	}
}

// setupHealthRoutes sets up health check and system status routes
func (r *Router) setupHealthRoutes(engine *gin.Engine) {
	engine.GET("/health", func(c *gin.Context) {
		// Perform health checks
		if err := r.db.HealthCheck(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":    "unhealthy",
				"error":     err.Error(),
				"timestamp": time.Now(),
				"service":   "evently-backend",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now(),
			"service":   "evently-backend",
		})
	})

	engine.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
			"version": r.config.APIVersion,
		})
	})

	engine.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":      "operational",
			"api_version": r.config.APIVersion,
			"timestamp":   time.Now(),
		})
	})
}

// setupAuthRoutes configures authentication routes
func (r *Router) setupAuthRoutes(rg *gin.RouterGroup) {
	// Initialize auth dependencies
	authRepo := auth.NewRepository(r.db.GetPostgreSQL()) // Use the correct method
	authService := auth.NewService(authRepo, r.config)
	authController := auth.NewController(authService)
	authRouter := auth.NewRouter(authController)

	// Setup auth routes
	authRouter.SetupRoutes(rg)
}

// setupEventRoutes configures event management routes
// func (r *Router) setupEventRoutes(rg *gin.RouterGroup) {
// 	// TODO: Implement event routes
// 	events := rg.Group("/events")
// 	{
// 		events.GET("/", func(c *gin.Context) {
// 			c.JSON(http.StatusOK, gin.H{"message": "events endpoint"})
// 		})
// 	}
// }

// setupBookingRoutes configures booking management routes
// func (r *Router) setupBookingRoutes(rg *gin.RouterGroup) {
// 	// TODO: Implement booking routes
// 	bookings := rg.Group("/bookings")
// 	{
// 		bookings.GET("/", func(c *gin.Context) {
// 			c.JSON(http.StatusOK, gin.H{"message": "bookings endpoint"})
// 		})
// 	}
// }

// setupAnalyticsRoutes configures analytics routes
// func (r *Router) setupAnalyticsRoutes(rg *gin.RouterGroup) {
// 	// TODO: Implement analytics routes
// 	analytics := rg.Group("/analytics")
// 	{
// 		analytics.GET("/", func(c *gin.Context) {
// 			c.JSON(http.StatusOK, gin.H{"message": "analytics endpoint"})
// 		})
// 	}
// }
