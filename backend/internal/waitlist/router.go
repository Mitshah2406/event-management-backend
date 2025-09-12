package waitlist

import (
	"evently/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers waitlist routes with the router
func RegisterRoutes(router *gin.Engine, controller *Controller) {
	// Public waitlist routes (require authentication)
	v1 := router.Group("/api/v1")
	{
		waitlist := v1.Group("/waitlist")
		{
			// User waitlist operations
			waitlist.POST("", controller.JoinWaitlist)                      // Join waitlist
			waitlist.DELETE("/:event_id", controller.LeaveWaitlist)         // Leave waitlist
			waitlist.GET("/status/:event_id", controller.GetWaitlistStatus) // Get status
			waitlist.GET("/health", controller.HealthCheck)                 // Health check
		}

		// Admin waitlist routes (require admin role)
		admin := v1.Group("/admin")
		{
			adminWaitlist := admin.Group("/waitlist")
			{
				// Admin operations
				adminWaitlist.GET("/stats/:event_id", controller.GetWaitlistStats)            // Get stats
				adminWaitlist.GET("/entries/:event_id", controller.GetWaitlistEntries)        // List entries
				adminWaitlist.POST("/notify/:event_id", controller.NotifyNextInLine)          // Manual notify
				adminWaitlist.POST("/cancellation/:event_id", controller.ProcessCancellation) // Process cancellation
			}
		}
	}
}

// RegisterRoutesWithMiddleware registers waitlist routes with custom middleware
func RegisterRoutesWithMiddleware(
	router *gin.Engine,
	controller *Controller,
	authMiddleware gin.HandlerFunc,
	adminMiddleware gin.HandlerFunc,
	rateLimitMiddleware gin.HandlerFunc,
) {
	// Public waitlist routes (require authentication)
	v1 := router.Group("/api/v1")
	{
		waitlist := v1.Group("/waitlist")
		{
			// Health check - no auth required
			waitlist.GET("/health", controller.HealthCheck)

			// Authenticated user operations
			authenticated := waitlist.Group("")
			authenticated.Use(authMiddleware)
			if rateLimitMiddleware != nil {
				authenticated.Use(rateLimitMiddleware)
			}
			{
				authenticated.POST("", controller.JoinWaitlist)                      // Join waitlist
				authenticated.DELETE("/:event_id", controller.LeaveWaitlist)         // Leave waitlist
				authenticated.GET("/status/:event_id", controller.GetWaitlistStatus) // Get status
			}
		}

		// Admin waitlist routes (require admin role)
		admin := v1.Group("/admin")
		admin.Use(authMiddleware)
		admin.Use(adminMiddleware)
		{
			adminWaitlist := admin.Group("/waitlist")
			{
				// Admin operations
				adminWaitlist.GET("/stats/:event_id", controller.GetWaitlistStats)            // Get stats
				adminWaitlist.GET("/entries/:event_id", controller.GetWaitlistEntries)        // List entries
				adminWaitlist.POST("/notify/:event_id", controller.NotifyNextInLine)          // Manual notify
				adminWaitlist.POST("/cancellation/:event_id", controller.ProcessCancellation) // Process cancellation
			}
		}
	}
}

// WaitlistRoutes contains route information for documentation
type WaitlistRoutes struct {
	UserRoutes  []RouteInfo `json:"user_routes"`
	AdminRoutes []RouteInfo `json:"admin_routes"`
}

// RouteInfo contains information about a route
type RouteInfo struct {
	Method        string `json:"method"`
	Path          string `json:"path"`
	Description   string `json:"description"`
	AuthRequired  bool   `json:"auth_required"`
	AdminRequired bool   `json:"admin_required"`
}

// GetRouteInfo returns information about all waitlist routes
func GetRouteInfo() WaitlistRoutes {
	return WaitlistRoutes{
		UserRoutes: []RouteInfo{
			{
				Method:        "POST",
				Path:          "/api/v1/waitlist",
				Description:   "Join an event waitlist",
				AuthRequired:  true,
				AdminRequired: false,
			},
			{
				Method:        "DELETE",
				Path:          "/api/v1/waitlist/:event_id",
				Description:   "Leave an event waitlist",
				AuthRequired:  true,
				AdminRequired: false,
			},
			{
				Method:        "GET",
				Path:          "/api/v1/waitlist/status/:event_id",
				Description:   "Get waitlist status for an event",
				AuthRequired:  true,
				AdminRequired: false,
			},
			{
				Method:        "GET",
				Path:          "/api/v1/waitlist/health",
				Description:   "Health check for waitlist service",
				AuthRequired:  false,
				AdminRequired: false,
			},
		},
		AdminRoutes: []RouteInfo{
			{
				Method:        "GET",
				Path:          "/api/v1/admin/waitlist/stats/:event_id",
				Description:   "Get waitlist statistics for an event",
				AuthRequired:  true,
				AdminRequired: true,
			},
			{
				Method:        "GET",
				Path:          "/api/v1/admin/waitlist/entries/:event_id",
				Description:   "Get waitlist entries for an event",
				AuthRequired:  true,
				AdminRequired: true,
			},
			{
				Method:        "POST",
				Path:          "/api/v1/admin/waitlist/notify/:event_id",
				Description:   "Manually notify next users in waitlist",
				AuthRequired:  true,
				AdminRequired: true,
			},
			{
				Method:        "POST",
				Path:          "/api/v1/admin/waitlist/cancellation/:event_id",
				Description:   "Process event cancellation and notify waitlist",
				AuthRequired:  true,
				AdminRequired: true,
			},
		},
	}
}

// RegisterWaitlistModule registers the complete waitlist module with dependencies
func RegisterWaitlistModule(
	router *gin.Engine,
	service Service,
	authMiddleware gin.HandlerFunc,
	adminMiddleware gin.HandlerFunc,
	rateLimitMiddleware gin.HandlerFunc,
) *Controller {
	controller := NewController(service)

	RegisterRoutesWithMiddleware(
		router,
		controller,
		authMiddleware,
		adminMiddleware,
		rateLimitMiddleware,
	)

	return controller
}

// SetupWaitlistRoutes configures all waitlist-related routes following the same pattern as other modules
func SetupWaitlistRoutes(rg *gin.RouterGroup, controller *Controller) {
	// Public waitlist routes (require authentication)
	waitlist := rg.Group("/waitlist")
	{
		// Health check - no auth required
		waitlist.GET("/health", controller.HealthCheck)

		// Authenticated user operations
		authenticated := waitlist.Group("")
		authenticated.Use(middleware.JWTAuth(), middleware.RequireRoles("USER", "ADMIN"))
		{
			authenticated.POST("", controller.JoinWaitlist)                      // JOIN waitlist
			authenticated.DELETE("/:event_id", controller.LeaveWaitlist)         // LEAVE waitlist
			authenticated.GET("/status/:event_id", controller.GetWaitlistStatus) // GET status
		}
	}

	// Admin waitlist routes
	adminWaitlist := rg.Group("/admin/waitlist")
	adminWaitlist.Use(middleware.JWTAuth(), middleware.RequireAdmin())
	{
		adminWaitlist.GET("/stats/:event_id", controller.GetWaitlistStats)            // Get stats
		adminWaitlist.GET("/entries/:event_id", controller.GetWaitlistEntries)        // List entries
		adminWaitlist.GET("/notifications/recent", controller.GetRecentNotifications) // Get recent notifications
		adminWaitlist.POST("/notify/:event_id", controller.NotifyNextInLine)          // Manual notify
		adminWaitlist.POST("/cancellation/:event_id", controller.ProcessCancellation) // Process cancellation
	}
}
