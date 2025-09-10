package bookings

import (
	"evently/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

func SetupBookingRoutes(router *gin.RouterGroup, controller *Controller) {
	// User routes - authenticated users can manage their own bookings
	userBookings := router.Group("/bookings")
	userBookings.Use(middleware.JWTAuth()) // Apply JWT authentication middleware
	{
		// Core booking operations
		userBookings.POST("", controller.CreateBooking)          // POST /api/v1/bookings - Create a booking
		userBookings.DELETE("/:id", controller.CancelBooking)    // DELETE /api/v1/bookings/:id - Cancel a booking
		userBookings.GET("/:id", controller.GetBookingDetails)   // GET /api/v1/bookings/:id - Get booking details
		userBookings.GET("/user/me", controller.GetUserBookings) // GET /api/v1/bookings/user/me - Get current user's bookings
	}

	// Admin routes - admin users can manage all bookings
	adminBookings := router.Group("/admin/bookings")
	adminBookings.Use(middleware.JWTAuth())      // Apply JWT authentication
	adminBookings.Use(middleware.RequireAdmin()) // Require admin role
	{
		// Admin booking management
		adminBookings.GET("", controller.GetAllBookings)               // GET /api/v1/admin/bookings - Get all bookings
		adminBookings.GET("/:id", controller.GetBookingDetailsAsAdmin) // GET /api/v1/admin/bookings/:id - Get any booking details
		adminBookings.DELETE("/:id", controller.CancelBookingAsAdmin)  // DELETE /api/v1/admin/bookings/:id - Cancel any booking
	}

	// Admin event-specific booking routes
	adminEvents := router.Group("/admin/events/bookings")
	adminEvents.Use(middleware.JWTAuth())      // Apply JWT authentication
	adminEvents.Use(middleware.RequireAdmin()) // Require admin role
	{
		adminEvents.GET("/:eventId", controller.GetEventBookings) // GET /api/v1/admin/events/:eventId/bookings - Get event bookings
	}
}
