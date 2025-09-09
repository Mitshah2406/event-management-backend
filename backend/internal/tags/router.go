package tags

import (
	"evently/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

func SetupTagRoutes(router *gin.RouterGroup, controller Controller) {
	// Public routes - anyone can view active tags (for browsing and filtering)
	publicTags := router.Group("/tags")
	{
		publicTags.GET("/active", controller.GetActiveTags)    // GET /api/v1/tags/active - Get active tags for filtering
		publicTags.GET("/slug/:slug", controller.GetTagBySlug) // GET /api/v1/tags/slug/:slug - Get tag by slug
	}

	// User routes - authenticated users can view tags (same as public for now)
	// userTags := router.Group("/user/tags")
	// userTags.Use(middleware.JWTAuth()) // Apply JWT authentication middleware
	// {
	// 	userTags.GET("/active", controller.GetActiveTags) // GET /api/v1/user/tags/active - User get active tags
	// 	userTags.GET("/slug/:slug", controller.GetTagBySlug) // GET /api/v1/user/tags/slug/:slug - User get tag by slug
	// }

	// Admin routes - only admins can manage tags and view analytics
	adminTags := router.Group("/admin/tags")
	adminTags.Use(middleware.JWTAuth(), middleware.RequireAdmin()) // Only admin users
	{
		// Tag management - Admin only
		adminTags.POST("", controller.CreateTag)       // POST /api/v1/admin/tags - Create tag
		adminTags.GET("", controller.GetAllTags)       // GET /api/v1/admin/tags - Get all tags (with filters)
		adminTags.GET("/:id", controller.GetTag)       // GET /api/v1/admin/tags/:id - Get tag by ID
		adminTags.PUT("/:id", controller.UpdateTag)    // PUT /api/v1/admin/tags/:id - Update tag
		adminTags.DELETE("/:id", controller.DeleteTag) // DELETE /api/v1/admin/tags/:id - Delete tag

		// Tag analytics - Admin only
		adminTags.GET("/analytics", controller.GetTagAnalytics)                      // GET /api/v1/admin/tags/analytics - Overall tag analytics
		adminTags.GET("/analytics/popularity", controller.GetTagPopularityAnalytics) // GET /api/v1/admin/tags/analytics/popularity - Tag popularity analytics
		adminTags.GET("/analytics/trends", controller.GetTagTrends)                  // GET /api/v1/admin/tags/analytics/trends - Tag trends
		adminTags.GET("/analytics/comparisons", controller.GetTagComparisons)        // GET /api/v1/admin/tags/analytics/comparisons - Tag comparisons

		// Active tags for admin (same as public but in admin context)
		adminTags.GET("/active", controller.GetActiveTags) // GET /api/v1/admin/tags/active - Admin get active tags
	}
}
