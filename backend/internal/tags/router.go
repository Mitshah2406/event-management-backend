package tags

import (
	"evently/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

func SetupTagRoutes(router *gin.RouterGroup, controller Controller) {
	// Public routes
	publicTags := router.Group("/tags")
	{
		publicTags.GET("/active", controller.GetActiveTags)    // GET /api/v1/tags/active - Get active tags for filtering
		publicTags.GET("/slug/:slug", controller.GetTagBySlug) // GET /api/v1/tags/slug/:slug - Get tag by slug
	}

	// Admin routes
	adminTags := router.Group("/admin/tags")
	adminTags.Use(middleware.JWTAuth(), middleware.RequireAdmin())
	{
		// Tag management - Admin only
		adminTags.POST("", controller.CreateTag)           // POST /api/v1/admin/tags - Create tag
		adminTags.GET("", controller.GetAllTags)           // GET /api/v1/admin/tags - Get all tags (with filters)
		adminTags.GET("/:id", controller.GetTag)           // GET /api/v1/admin/tags/:id - Get tag by ID
		adminTags.PUT("/:id", controller.UpdateTag)        // PUT /api/v1/admin/tags/:id - Update tag
		adminTags.DELETE("/:id", controller.DeleteTag)     // DELETE /api/v1/admin/tags/:id - Delete tag
		adminTags.GET("/active", controller.GetActiveTags) // GET /api/v1/admin/tags/active - Admin get active tags
	}
}
