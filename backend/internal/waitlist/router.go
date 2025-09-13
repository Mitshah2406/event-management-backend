package waitlist

import (
	"evently/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

func SetupWaitlistRoutes(rg *gin.RouterGroup, controller *Controller) {
	waitlist := rg.Group("/waitlist")
	{
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
		adminWaitlist.GET("/stats/:event_id", controller.GetWaitlistStats)     // Get stats
		adminWaitlist.GET("/entries/:event_id", controller.GetWaitlistEntries) // List entries

		adminWaitlist.POST("/notify/:event_id", controller.NotifyNextInLine)          // Manual notify
		adminWaitlist.POST("/cancellation/:event_id", controller.ProcessCancellation) // Process cancellation
	}
}
