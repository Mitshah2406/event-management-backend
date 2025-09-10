// api/routes/router.go
package routes

import (
	"evently/internal/auth"
	"evently/internal/bookings"
	"evently/internal/events"
	"evently/internal/shared/config"
	"evently/internal/shared/database"
	"evently/internal/tags"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Router holds all route dependencies
type Router struct {
	config       *config.Config
	db           *database.DB
	tagService   tags.Service   // For dependency injection
	eventService events.Service // For dependency injection
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

		// Setup tag routes (must be before event routes for dependency injection)
		r.setupTagRoutes(api)

		// Setup event routes
		r.setupEventRoutes(api)

		// Setup booking routes
		r.setupBookingRoutes(api)
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

// setupTagRoutes configures tag management routes
func (r *Router) setupTagRoutes(rg *gin.RouterGroup) {
	// Initialize tag dependencies
	tagRepo := tags.NewRepository(r.db.GetPostgreSQL())
	tagService := tags.NewService(tagRepo)
	tagController := tags.NewController(tagService)

	// Store tag service for dependency injection
	r.tagService = tagService

	// Setup tag routes
	tags.SetupTagRoutes(rg, tagController)
}

// setupEventRoutes configures event management routes
func (r *Router) setupEventRoutes(rg *gin.RouterGroup) {
	// Initialize event dependencies
	eventRepo := events.NewRepository(r.db.GetPostgreSQL())
	eventService := events.NewService(eventRepo)

	// Inject tag service dependency
	if r.tagService != nil {
		eventService.SetTagService(r.tagService)
	}

	// Store event service for dependency injection
	r.eventService = eventService

	eventController := events.NewController(eventService)

	// Setup event routes
	events.SetupEventRoutes(rg, eventController)
}

// setupBookingRoutes configures booking management routes
func (r *Router) setupBookingRoutes(rg *gin.RouterGroup) {
	// Initialize booking dependencies
	bookingRepo := bookings.NewRepository(r.db.GetPostgreSQL())
	bookingService := bookings.NewService(bookingRepo)

	// Inject event service dependency if available
	if r.eventService != nil {
		bookingService.SetEventService(r.eventService)
	}

	bookingController := bookings.NewController(bookingService)

	// Setup booking routes
	bookings.SetupBookingRoutes(rg, bookingController)
}

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
