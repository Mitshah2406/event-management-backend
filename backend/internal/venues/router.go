package venues

import (
	"evently/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

func SetupVenueRoutes(rg *gin.RouterGroup, controller *Controller) {
	// Venue Templates routes
	templates := rg.Group("/admin/venue-templates")
	templates.Use(middleware.JWTAuth(), middleware.RequireAdmin())
	{
		templates.POST("", controller.CreateTemplate)       // POST /api/v1/venue-templates
		templates.GET("", controller.GetTemplates)          // GET /api/v1/venue-templates
		templates.GET("/:id", controller.GetTemplate)       // GET /api/v1/venue-templates/:id
		templates.PUT("/:id", controller.UpdateTemplate)    // PUT /api/v1/venue-templates/:id
		templates.DELETE("/:id", controller.DeleteTemplate) // DELETE /api/v1/venue-templates/:id

		// Template sections routes
		templates.POST("/:id/sections", controller.CreateSection)          // POST /api/v1/venue-templates/:id/sections
		templates.GET("/:id/sections", controller.GetSectionsByTemplateID) // GET /api/v1/venue-templates/:id/sections
	}

	// Event-specific venue reading routes
	events := rg.Group("/events")
	events.Use(middleware.JWTAuth(), middleware.RequireRole("USER"))
	{
		events.GET("/:eventId/sections", controller.GetSectionsByEventID) // GET /api/v1/events/:eventId/sections
		events.GET("/:eventId/venue/layout", controller.GetVenueLayout)   // GET /api/v1/events/:eventId/venue/layout
	}

	// Individual section routes
	sections := rg.Group("/admin/sections")
	sections.Use(middleware.JWTAuth(), middleware.RequireAdmin())
	{
		sections.PUT("/:id", controller.UpdateSection)    // PUT /api/v1/sections/:id
		sections.DELETE("/:id", controller.DeleteSection) // DELETE /api/v1/sections/:id
	}
}
