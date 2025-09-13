package seats

import (
	"evently/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

func SetupSeatRoutes(rg *gin.RouterGroup, controller *Controller) {

	// USER SEAT OPERATIONS

	seats := rg.Group("/seats")
	seats.Use(middleware.JWTAuth(), middleware.RequireRoles("USER", "ADMIN"))
	{
		// Individual seat
		seats.GET("/:id", controller.GetSeat) // GET /api/v1/seats/:id

		// Core seat holding endpoints (booking flow)
		seats.POST("/hold", controller.HoldSeats)                    // POST /api/v1/seats/hold
		seats.DELETE("/hold/:holdId", controller.ReleaseHold)        // DELETE /api/v1/seats/hold/:holdId
		seats.GET("/hold/:holdId/validate", controller.ValidateHold) // GET /api/v1/seats/hold/:holdId/validate

		// Availability checks
		seats.POST("/availability", controller.CheckSeatAvailability) // POST /api/v1/seats/availability
	}

	// ADMIN SEAT OPERATIONS
	adminSeats := rg.Group("/admin/seats")
	adminSeats.Use(middleware.JWTAuth(), middleware.RequireAdmin())
	{
		adminSeats.PUT("/:id", controller.UpdateSeat)    // PUT /api/v1/admin/seats/:id
		adminSeats.DELETE("/:id", controller.DeleteSeat) // DELETE /api/v1/admin/seats/:id
	}

	// SECTION-BASED OPERATIONS

	sections := rg.Group("/sections")
	{
		// Seat retrieval
		sections.GET("/:sectionId/seats", controller.GetSeatsBySectionID)                  // GET /api/v1/sections/:sectionId/seats
		sections.GET("/:sectionId/seats/available", controller.GetAvailableSeatsInSection) // GET /api/v1/sections/:sectionId/seats/available?event_id=xxx
	}

	// USER-SPECIFIC HOLDS

	users := rg.Group("/users")
	users.Use(middleware.JWTAuth(), middleware.RequireRoles("USER", "ADMIN"))
	{
		users.GET("/:userId/holds", controller.GetUserHolds) // GET /api/v1/users/:userId/holds
	}
}
