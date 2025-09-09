package events

import (
	"evently/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

func SetupEventRoutes(router *gin.RouterGroup, controller Controller) {
	// Public routes - anyone can view events (for browsing)
	publicEvents := router.Group("/events")
	{
		publicEvents.GET("", controller.GetAllEvents)               // GET /api/v1/events - Browse all events
		publicEvents.GET("/:id", controller.GetEvent)               // GET /api/v1/events/:id - Get event details
		publicEvents.GET("/upcoming", controller.GetUpcomingEvents) // GET /api/v1/events/upcoming - Browse upcoming events
	}

	// // User routes - authenticated users can only browse (same as public for now)
	// userEvents := router.Group("/user/events")
	// userEvents.Use(middleware.JWTAuth()) // Apply JWT authentication middleware
	// {
	// 	userEvents.GET("", controller.GetAllEvents)               // GET /api/v1/user/events - User browse events
	// 	userEvents.GET("/:id", controller.GetEvent)               // GET /api/v1/user/events/:id - User get event details
	// 	userEvents.GET("/upcoming", controller.GetUpcomingEvents) // GET /api/v1/user/events/upcoming - User browse upcoming events
	// }

	// Admin routes - only admins can create, update, delete and manage events
	adminEvents := router.Group("/admin/events")
	adminEvents.Use(middleware.JWTAuth(), middleware.RequireAdmin()) // Only admin users
	{
		// Event management - Admin only
		adminEvents.POST("", controller.CreateEvent)       // POST /api/v1/admin/events - Create event
		adminEvents.PUT("/:id", controller.UpdateEvent)    // PUT /api/v1/admin/events/:id - Update event
		adminEvents.DELETE("/:id", controller.DeleteEvent) // DELETE /api/v1/admin/events/:id - Delete event

		// Event analytics - Admin only
		adminEvents.GET("/analytics", controller.GetAllEventAnalytics)  // GET /api/v1/admin/events/analytics - Overall analytics
		adminEvents.GET("/:id/analytics", controller.GetEventAnalytics) // GET /api/v1/admin/events/:id/analytics - Specific event analytics

		// Admin can also browse events (same endpoints as users)
		adminEvents.GET("", controller.GetAllEvents) // GET /api/v1/admin/events - Admin browse events
		adminEvents.GET("/:id", controller.GetEvent) // GET /api/v1/admin/events/:id - Admin get event details
	}
}
