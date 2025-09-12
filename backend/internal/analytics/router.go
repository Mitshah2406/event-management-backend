package analytics

import (
	"evently/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

// SetupAnalyticsRoutes configures all analytics routes
func SetupAnalyticsRoutes(rg *gin.RouterGroup, controller Controller) {
	// Create analytics route group
	analytics := rg.Group("/analytics")

	// Setup admin analytics routes (protected)
	setupAdminAnalyticsRoutes(analytics, controller)

	// Setup user analytics routes (protected)
	setupUserAnalyticsRoutes(analytics, controller)
}

// setupAdminAnalyticsRoutes configures admin-only analytics routes
func setupAdminAnalyticsRoutes(rg *gin.RouterGroup, controller Controller) {
	admin := rg.Group("/admin")
	admin.Use(middleware.JWTAuth())
	admin.Use(middleware.RequireAdmin())

	// Dashboard & Overview
	admin.GET("/dashboard", controller.GetDashboardAnalytics)

	// Event Analytics (migrated from /admin/events/analytics)
	events := admin.Group("/events")
	{
		events.GET("", controller.GetGlobalEventAnalytics) // Global event analytics
		events.GET("/:id", controller.GetEventAnalytics)   // Specific event analytics
	}

	// Tag Analytics (migrated from /admin/tags/analytics)
	tags := admin.Group("/tags")
	{
		tags.GET("", controller.GetTagAnalytics)                      // Overall tag analytics
		tags.GET("/popularity", controller.GetTagPopularityAnalytics) // Tag popularity metrics
		tags.GET("/trends", controller.GetTagTrends)                  // Tag trends (with ?months=6 param)
		tags.GET("/comparisons", controller.GetTagComparisons)        // Tag performance comparisons
	}

	// Booking Analytics (new)
	bookings := admin.Group("/bookings")
	{
		bookings.GET("", controller.GetBookingAnalytics)                    // Overall booking analytics
		bookings.GET("/daily", controller.GetBookingDailyStats)             // Daily booking statistics
		bookings.GET("/cancellations", controller.GetCancellationAnalytics) // Cancellation rates & analysis
	}

	// User Analytics (new)
	users := admin.Group("/users")
	{
		users.GET("", controller.GetUserAnalytics)                  // User behavior analytics
		users.GET("/retention", controller.GetUserRetentionMetrics) // User retention metrics
		users.GET("/demographics", controller.GetUserDemographics)  // User demographics breakdown
	}
}

// setupUserAnalyticsRoutes configures user-facing analytics routes
func setupUserAnalyticsRoutes(rg *gin.RouterGroup, controller Controller) {
	user := rg.Group("/user")
	user.Use(middleware.JWTAuth()) // User must be authenticated

	// User-facing analytics
	bookings := user.Group("/bookings")
	{
		bookings.GET("/history", controller.GetUserBookingHistory) // User's booking history with insights
	}

	user.GET("/personal", controller.GetPersonalAnalytics) // Personal booking insights
}

// Alternative setup function for more granular control
func SetupAnalyticsRoutesDetailed(rg *gin.RouterGroup, controller Controller) {
	// Version prefix (if using API versioning)
	v1 := rg.Group("/v1")

	// Admin routes
	admin := v1.Group("/admin/analytics")
	admin.Use(middleware.JWTAuth())
	admin.Use(middleware.RequireAdmin())

	// Dashboard endpoints
	admin.GET("/dashboard", controller.GetDashboardAnalytics)

	// Event analytics endpoints
	admin.GET("/events", controller.GetGlobalEventAnalytics)
	admin.GET("/events/:id", controller.GetEventAnalytics)

	// Tag analytics endpoints
	admin.GET("/tags", controller.GetTagAnalytics)
	admin.GET("/tags/popularity", controller.GetTagPopularityAnalytics)
	admin.GET("/tags/trends", controller.GetTagTrends)
	admin.GET("/tags/comparisons", controller.GetTagComparisons)

	// Booking analytics endpoints
	admin.GET("/bookings", controller.GetBookingAnalytics)
	admin.GET("/bookings/daily", controller.GetBookingDailyStats)
	admin.GET("/bookings/cancellations", controller.GetCancellationAnalytics)

	// User analytics endpoints
	admin.GET("/users", controller.GetUserAnalytics)
	admin.GET("/users/retention", controller.GetUserRetentionMetrics)
	admin.GET("/users/demographics", controller.GetUserDemographics)

	// User-facing routes
	user := v1.Group("/user/analytics")
	user.Use(middleware.JWTAuth())

	user.GET("/bookings/history", controller.GetUserBookingHistory)
	user.GET("/personal", controller.GetPersonalAnalytics)
}

// SetupAnalyticsMiddleware configures middleware specific to analytics routes
func SetupAnalyticsMiddleware() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		// Rate limiting middleware (if needed)
		// gin.Recovery(),

		// Analytics-specific CORS headers (if needed)
		func(c *gin.Context) {
			c.Header("X-Analytics-Version", "1.0")
			c.Next()
		},

		// Request logging for analytics (if needed)
		gin.Logger(),
	}
}

// Route documentation for reference:
/*
Admin Analytics Routes:
GET /api/v1/admin/analytics/dashboard          # Cross-domain dashboard metrics
GET /api/v1/admin/analytics/events             # Global event analytics
GET /api/v1/admin/analytics/events/:id         # Specific event analytics
GET /api/v1/admin/analytics/tags               # Overall tag analytics
GET /api/v1/admin/analytics/tags/popularity    # Tag popularity metrics
GET /api/v1/admin/analytics/tags/trends        # Tag trends (with ?months=6 param)
GET /api/v1/admin/analytics/tags/comparisons   # Tag performance comparisons
GET /api/v1/admin/analytics/bookings           # Overall booking analytics
GET /api/v1/admin/analytics/bookings/daily     # Daily booking statistics
GET /api/v1/admin/analytics/bookings/cancellations # Cancellation rates & analysis
GET /api/v1/admin/analytics/users              # User behavior analytics
GET /api/v1/admin/analytics/users/retention    # User retention metrics
GET /api/v1/admin/analytics/users/demographics # User demographics breakdown

User Analytics Routes:
GET /api/v1/user/analytics/bookings/history    # User's booking history with insights
GET /api/v1/user/analytics/personal            # Personal booking insights
*/
