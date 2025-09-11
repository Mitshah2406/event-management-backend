package cancellation

import (
	"evently/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

// SetupCancellationRoutes configures all cancellation-related routes
func SetupCancellationRoutes(rg *gin.RouterGroup, controller *Controller) {
	// Event cancellation policy routes (Admin only)
	events := rg.Group("/admin/events")
	events.Use(middleware.JWTAuth(), middleware.RequireRoles("ADMIN"))
	{
		events.POST("/:eventId/cancellation-policy", controller.CreateCancellationPolicy) // POST /api/v1/events/:eventId/cancellation-policy
		events.GET("/:eventId/cancellation-policy", controller.GetCancellationPolicy)     // GET /api/v1/events/:eventId/cancellation-policy
		events.PUT("/:eventId/cancellation-policy", controller.UpdateCancellationPolicy)  // PUT /api/v1/events/:eventId/cancellation-policy
	}

	// Booking cancellation routes (Users and Admins)
	bookings := rg.Group("/bookings")
	bookings.Use(middleware.JWTAuth(), middleware.RequireRoles("USER", "ADMIN"))
	{
		bookings.POST("/:id/request-cancel", controller.RequestCancellation) // POST /api/v1/bookings/:id/request-cancel
	}

	// Cancellation management routes
	cancellations := rg.Group("/cancellations")
	cancellations.Use(middleware.JWTAuth(), middleware.RequireRoles("USER", "ADMIN"))
	{
		cancellations.GET("/:id", controller.GetCancellation) // GET /api/v1/cancellations/:id
	}

	// User-specific cancellation routes
	users := rg.Group("/users")
	users.Use(middleware.JWTAuth(), middleware.RequireRoles("USER", "ADMIN"))
	{
		users.GET("/cancellations", controller.GetUserCancellations) // GET /api/v1/users/cancellations
	}
}

// Route definitions for reference:
//
// ADMIN - CANCELLATION POLICY MANAGEMENT
// POST   /api/v1/events/:id/cancellation-policy      - Create cancellation policy for event
// GET    /api/v1/events/:id/cancellation-policy      - Get cancellation policy for event
// PUT    /api/v1/events/:id/cancellation-policy      - Update cancellation policy for event
//
// Request body for policy creation/update:
// {
//   "allow_cancellation": true,
//   "cancellation_deadline": "2024-12-31T23:59:59Z",
//   "fee_type": "PERCENTAGE", // NONE, FIXED, or PERCENTAGE
//   "fee_amount": 10.0,      // 10% or $10 depending on fee_type
//   "refund_processing_days": 5
// }
//
// USER - CANCELLATION REQUESTS
// POST   /api/v1/bookings/:id/request-cancel         - Request cancellation for booking (Auto-processed)
// Request body: { "reason": "Unable to attend due to personal reasons" }
//
// CANCELLATION TRACKING
// GET    /api/v1/cancellations/:id                   - Get specific cancellation details
// GET    /api/v1/users/cancellations                 - Get user's cancellation history
//
// Key Flow:
// 1. Admin creates cancellation policy for event
// 2. User requests cancellation for their booking
// 3. System validates eligibility and calculates fees
// 4. Cancellation is automatically approved and processed instantly
// 5. Booking status updated to CANCELLED and seats are freed for other users
// 6. Refund amount is calculated and will be processed within policy timeframe
