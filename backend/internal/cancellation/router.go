package cancellation

import (
	"evently/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

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
