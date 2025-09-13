package bookings

import (
	"evently/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

func SetupBookingRoutes(rg *gin.RouterGroup, controller *Controller) {
	// Booking routes
	bookings := rg.Group("/bookings")
	bookings.Use(middleware.JWTAuth(), middleware.RequireRoles("USER", "ADMIN"))
	{
		// Core booking operations
		bookings.POST("/confirm", controller.ConfirmBooking)   // POST /api/v1/bookings/confirm
		bookings.GET("/:id", controller.GetBooking)            // GET /api/v1/bookings/:id
		bookings.POST("/:id/cancel", controller.CancelBooking) // POST /api/v1/bookings/:id/cancel
	}

	// User-specific booking routes
	users := rg.Group("/users")
	users.Use(middleware.JWTAuth(), middleware.RequireRoles("USER", "ADMIN"))
	{
		users.GET("/bookings", controller.GetUserBookings) // GET /api/v1/users/bookings
	}
}
