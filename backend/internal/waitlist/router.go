package waitlist

import (
	"evently/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

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
