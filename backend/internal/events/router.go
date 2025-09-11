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
		publicEvents.GET("/:eventId", controller.GetEvent)          // GET /api/v1/events/:eventId - Get event details
		publicEvents.GET("/upcoming", controller.GetUpcomingEvents) // GET /api/v1/events/upcoming - Browse upcoming events
	}

	// Admin routes - only admins can create, update, delete and manage events
	adminEvents := router.Group("/admin/events")
	adminEvents.Use(middleware.JWTAuth(), middleware.RequireAdmin()) // Only admin users
	{
		// Event management - Admin only
		adminEvents.POST("", controller.CreateEvent)            // POST /api/v1/admin/events - Create event
		adminEvents.PUT("/:eventId", controller.UpdateEvent)    // PUT /api/v1/admin/events/:eventId - Update event
		adminEvents.DELETE("/:eventId", controller.DeleteEvent) // DELETE /api/v1/admin/events/:eventId - Delete event

		// Event analytics - Admin only
		adminEvents.GET("/analytics", controller.GetAllEventAnalytics)       // GET /api/v1/admin/events/analytics - Overall analytics
		adminEvents.GET("/:eventId/analytics", controller.GetEventAnalytics) // GET /api/v1/admin/events/:eventId/analytics - Specific event analytics

		// Admin can also browse events (same endpoints as users)
		adminEvents.GET("", controller.GetAllEvents)      // GET /api/v1/admin/events - Admin browse events
		adminEvents.GET("/:eventId", controller.GetEvent) // GET /api/v1/admin/events/:eventId - Admin get event details
	}
}
