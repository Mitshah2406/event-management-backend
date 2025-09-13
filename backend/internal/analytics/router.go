package analytics

import (
	"evently/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

func SetupAnalyticsRoutes(rg *gin.RouterGroup, controller Controller) {

	analytics := rg.Group("/analytics")

	// Setup admin analytics routes (protected)
	setupAdminAnalyticsRoutes(analytics, controller)

	// Setup user analytics routes (protected)
	setupUserAnalyticsRoutes(analytics, controller)
}

func setupAdminAnalyticsRoutes(rg *gin.RouterGroup, controller Controller) {
	admin := rg.Group("/admin")
	admin.Use(middleware.JWTAuth())
	admin.Use(middleware.RequireAdmin())

	// Dashboard & Overview
	admin.GET("/dashboard", controller.GetDashboardAnalytics)

	// Event Analytics
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

	// Booking Analytics
	bookings := admin.Group("/bookings")
	{
		bookings.GET("", controller.GetBookingAnalytics)                    // Overall booking analytics
		bookings.GET("/daily", controller.GetBookingDailyStats)             // Daily booking statistics
		bookings.GET("/cancellations", controller.GetCancellationAnalytics) // Cancellation rates & analysis
	}

	// User Analytics
	users := admin.Group("/users")
	{
		users.GET("", controller.GetUserAnalytics)                  // User behavior analytics
		users.GET("/retention", controller.GetUserRetentionMetrics) // User retention metrics
		users.GET("/demographics", controller.GetUserDemographics)  // User demographics breakdown
	}
}

func setupUserAnalyticsRoutes(rg *gin.RouterGroup, controller Controller) {
	user := rg.Group("/user")
	user.Use(middleware.JWTAuth())

	// User-facing analytics
	bookings := user.Group("/bookings")
	{
		bookings.GET("/history", controller.GetUserBookingHistory) // User's booking history with insights
	}

	user.GET("/personal", controller.GetPersonalAnalytics) // Personal booking insights
}
