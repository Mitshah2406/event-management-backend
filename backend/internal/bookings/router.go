package bookings

import (
	"evently/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

// SetupBookingRoutes configures all booking-related routes
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

// Route definitions for reference:
//
// BOOKING CONFIRMATION
// POST   /api/v1/bookings/confirm                     - Confirm a held booking
// Request body: { "hold_id": "hold_xxx", "payment_method": "credit_card" }
//
// BOOKING RETRIEVAL
// GET    /api/v1/bookings/:id                         - Get specific booking
//
// BOOKING CANCELLATION
// POST   /api/v1/bookings/:id/cancel                  - Cancel a booking
//
// USER BOOKINGS
// GET    /api/v1/users/bookings?limit=10&offset=0     - Get user's bookings with pagination
//
// Key Flow After Seat Holding:
// 1. User holds seats with POST /seats/hold
// 2. User confirms booking with POST /bookings/confirm
// 3. System validates hold, processes payment, creates booking
// 4. Seats are marked as BOOKED, Redis hold is released
// 5. User can view booking with GET /bookings/:id
// 6. User can cancel booking with POST /bookings/:id/cancel
