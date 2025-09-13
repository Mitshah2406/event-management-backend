package analytics

import (
	"context"
	"fmt"
	"time"

	"evently/internal/shared/utils/constants"
	"evently/pkg/cache"

	"github.com/google/uuid"
)

// Service defines the analytics service interface
type Service interface {
	// Dashboard Analytics
	GetDashboardAnalytics() (*DashboardAnalytics, error)

	// Event Analytics (migrated from events package)
	GetEventAnalytics(eventID uuid.UUID) (*EventAnalytics, error)
	GetGlobalEventAnalytics() (*GlobalEventAnalytics, error)

	// Tag Analytics (migrated from tags package)
	GetTagAnalytics() (*TagAnalyticsResponse, error)
	GetTagPopularityAnalytics() ([]TagAnalytics, error)
	GetTagTrends(months int) ([]TagTrend, error)
	GetTagComparisons() ([]TagComparison, error)

	// Booking Analytics (new)
	GetBookingAnalytics() (*BookingAnalytics, error)
	GetBookingDailyStats() ([]DailyBookingStats, error)
	GetCancellationAnalytics() (*CancellationAnalytics, error)

	// User Analytics (new)
	GetUserAnalytics() (*UserAnalytics, error)

	// User-facing Analytics
	GetUserBookingHistory(userID uuid.UUID) (*UserBookingHistory, error)
	GetPersonalAnalytics(userID uuid.UUID) (*PersonalAnalytics, error)
}

// service implements the Service interface
type service struct {
	repo         Repository
	cacheService cache.Service
}

// NewService creates a new analytics service instance
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// SetCacheService injects the cache service dependency
func (s *service) SetCacheService(cacheService cache.Service) {
	s.cacheService = cacheService
}

// Dashboard Analytics Implementation

func (s *service) GetDashboardAnalytics() (*DashboardAnalytics, error) {
	ctx := context.Background()
	cacheKey := constants.CACHE_KEY_ANALYTICS_DASHBOARD

	// Try to get from cache first
	if s.cacheService != nil {
		var cachedDashboard DashboardAnalytics
		if err := s.cacheService.Get(ctx, cacheKey, &cachedDashboard); err == nil {
			return &cachedDashboard, nil
		}
	}

	// Cache miss - get from repository
	dashboard, err := s.repo.GetDashboardAnalytics()
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard analytics: %w", err)
	}

	// Cache the result
	if s.cacheService != nil {
		if err := s.cacheService.Set(ctx, cacheKey, dashboard, constants.TTL_ANALYTICS_DASHBOARD); err != nil {
			// Log error but don't fail the request
			fmt.Printf("Warning: failed to cache dashboard analytics: %v\n", err)
		}
	}

	return dashboard, nil
}

// Event Analytics Implementation

func (s *service) GetEventAnalytics(eventID uuid.UUID) (*EventAnalytics, error) {
	ctx := context.Background()
	cacheKey := constants.BuildAnalyticsEventKey(eventID.String())

	// Try to get from cache first
	if s.cacheService != nil {
		var cachedAnalytics EventAnalytics
		if err := s.cacheService.Get(ctx, cacheKey, &cachedAnalytics); err == nil {
			return &cachedAnalytics, nil
		}
	}

	// Cache miss - get from repository
	analytics, err := s.repo.GetEventAnalytics(eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get event analytics: %w", err)
	}

	// Add business logic processing here if needed
	// For example, calculating additional metrics, applying business rules, etc.

	// Cache the result
	if s.cacheService != nil {
		if err := s.cacheService.Set(ctx, cacheKey, analytics, constants.TTL_ANALYTICS_EVENT); err != nil {
			// Log error but don't fail the request
			fmt.Printf("Warning: failed to cache event analytics: %v\n", err)
		}
	}

	return analytics, nil
}

func (s *service) GetGlobalEventAnalytics() (*GlobalEventAnalytics, error) {
	analytics, err := s.repo.GetGlobalEventAnalytics()
	if err != nil {
		return nil, fmt.Errorf("failed to get global event analytics: %w", err)
	}

	// Add any additional business logic processing
	// For example, calculating performance scores, rankings, etc.

	return analytics, nil
}

// Tag Analytics Implementation

func (s *service) GetTagAnalytics() (*TagAnalyticsResponse, error) {
	analytics, err := s.repo.GetTagAnalytics()
	if err != nil {
		return nil, fmt.Errorf("failed to get tag analytics: %w", err)
	}

	// Add business logic processing
	// For example, calculating popularity scores, trend analysis, etc.

	return analytics, nil
}

func (s *service) GetTagPopularityAnalytics() ([]TagAnalytics, error) {
	analytics, err := s.repo.GetTagPopularityAnalytics()
	if err != nil {
		return nil, fmt.Errorf("failed to get tag popularity analytics: %w", err)
	}

	// Add business logic for popularity scoring
	for i := range analytics {
		// Calculate popularity score based on multiple factors
		analytics[i].PopularityScore = s.calculateTagPopularityScore(analytics[i])
	}

	return analytics, nil
}

func (s *service) GetTagTrends(months int) ([]TagTrend, error) {
	// Validate input
	if months <= 0 || months > 24 {
		months = 6 // Default to 6 months
	}

	trends, err := s.repo.GetTagTrends(months)
	if err != nil {
		return nil, fmt.Errorf("failed to get tag trends: %w", err)
	}

	// Add trend analysis logic
	// For example, calculating growth rates, identifying seasonal patterns, etc.

	return trends, nil
}

func (s *service) GetTagComparisons() ([]TagComparison, error) {
	comparisons, err := s.repo.GetTagComparisons()
	if err != nil {
		return nil, fmt.Errorf("failed to get tag comparisons: %w", err)
	}

	// Add comparative analysis logic
	// For example, calculating relative performance metrics, rankings, etc.

	return comparisons, nil
}

// Booking Analytics Implementation

func (s *service) GetBookingAnalytics() (*BookingAnalytics, error) {
	analytics, err := s.repo.GetBookingAnalytics()
	if err != nil {
		return nil, fmt.Errorf("failed to get booking analytics: %w", err)
	}

	// Add business logic processing
	// For example, generating insights, calculating performance indicators, etc.
	analytics.Insights = s.generateBookingInsights(analytics)

	return analytics, nil
}

func (s *service) GetBookingDailyStats() ([]DailyBookingStats, error) {
	stats, err := s.repo.GetDailyBookingStats(30) // Default to 30 days
	if err != nil {
		return nil, fmt.Errorf("failed to get daily booking stats: %w", err)
	}

	// Add additional calculations
	for i := range stats {
		if stats[i].TotalBookings > 0 {
			stats[i].AverageValue = stats[i].Revenue / float64(stats[i].TotalBookings)
		}
	}

	return stats, nil
}

func (s *service) GetCancellationAnalytics() (*CancellationAnalytics, error) {
	analytics, err := s.repo.GetCancellationAnalytics()
	if err != nil {
		return nil, fmt.Errorf("failed to get cancellation analytics: %w", err)
	}

	// Add business logic for cancellation analysis
	// For example, identifying patterns, calculating impact, etc.

	return analytics, nil
}

// User Analytics Implementation

func (s *service) GetUserAnalytics() (*UserAnalytics, error) {
	analytics, err := s.repo.GetUserAnalytics()
	if err != nil {
		return nil, fmt.Errorf("failed to get user analytics: %w", err)
	}

	// Add business logic processing
	// For example, generating user insights, segmentation analysis, etc.
	analytics.Insights = s.generateUserInsights(analytics)

	return analytics, nil
}

// User-facing Analytics Implementation

func (s *service) GetUserBookingHistory(userID uuid.UUID) (*UserBookingHistory, error) {
	history, err := s.repo.GetUserBookingHistory(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user booking history: %w", err)
	}

	// Add personalization logic
	// For example, calculating personal metrics, generating recommendations, etc.
	history.Insights = s.generatePersonalInsights(history)

	return history, nil
}

func (s *service) GetPersonalAnalytics(userID uuid.UUID) (*PersonalAnalytics, error) {
	analytics, err := s.repo.GetPersonalAnalytics(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get personal analytics: %w", err)
	}

	// Add personal analytics logic
	// For example, generating personalized recommendations, achievements, etc.
	analytics.Recommendations = s.generatePersonalRecommendations(userID, analytics)
	analytics.Achievements = s.calculateAchievements(userID, analytics)

	return analytics, nil
}

// Helper methods for business logic

func (s *service) calculateTagPopularityScore(tagAnalytics TagAnalytics) float64 {
	// Calculate popularity score based on multiple factors
	// This is a simplified example - in practice, you might use more sophisticated algorithms

	eventCountScore := float64(tagAnalytics.EventCount) * 0.3
	bookingScore := float64(tagAnalytics.TotalBookings) * 0.4
	revenueScore := tagAnalytics.TotalRevenue / 1000 * 0.2 // Normalize revenue
	utilizationScore := tagAnalytics.AvgUtilization * 0.1

	return eventCountScore + bookingScore + revenueScore + utilizationScore
}

func (s *service) generateBookingInsights(analytics *BookingAnalytics) []BookingInsight {
	var insights []BookingInsight

	// Example insight generation logic
	if analytics.Overview.CancellationRate > 15.0 {
		insights = append(insights, BookingInsight{
			Type:        "alert",
			Title:       "High Cancellation Rate",
			Description: fmt.Sprintf("Cancellation rate is %.1f%%, which is above the recommended 15%%", analytics.Overview.CancellationRate),
			Impact:      "high",
			Metric:      "cancellation_rate",
			Value:       fmt.Sprintf("%.1f%%", analytics.Overview.CancellationRate),
		})
	}

	if analytics.Overview.AverageBookingSize < 2.0 {
		insights = append(insights, BookingInsight{
			Type:        "opportunity",
			Title:       "Low Average Booking Size",
			Description: "Consider promoting group bookings or family packages to increase average booking size",
			Impact:      "medium",
			Metric:      "avg_booking_size",
			Value:       fmt.Sprintf("%.1f", analytics.Overview.AverageBookingSize),
		})
	}

	// Add trend-based insights
	if analytics.TrendAnalysis.Growth.BookingGrowth > 20.0 {
		insights = append(insights, BookingInsight{
			Type:        "trend",
			Title:       "Strong Booking Growth",
			Description: "Booking volume is showing strong upward trend",
			Impact:      "high",
			Metric:      "booking_growth",
			Value:       fmt.Sprintf("%.1f%%", analytics.TrendAnalysis.Growth.BookingGrowth),
		})
	}

	return insights
}

func (s *service) generateUserInsights(analytics *UserAnalytics) []UserInsight {
	var insights []UserInsight

	// Generate insights based on available data
	if analytics.Overview.RetentionRate < 50.0 {
		insights = append(insights, UserInsight{
			Type:        "opportunity",
			Title:       "User Retention Opportunity",
			Description: fmt.Sprintf("User retention rate is %.1f%%, consider implementing engagement campaigns", analytics.Overview.RetentionRate),
			UserCount:   analytics.Overview.TotalUsers - analytics.Overview.ActiveUsers,
			Impact:      "medium",
			Action:      "Implement user engagement and retention strategies",
		})
	}

	if analytics.Overview.AvgBookingsPerUser < 2.0 {
		insights = append(insights, UserInsight{
			Type:        "opportunity",
			Title:       "Low Booking Frequency",
			Description: fmt.Sprintf("Average bookings per user is %.1f, focus on repeat engagement", analytics.Overview.AvgBookingsPerUser),
			UserCount:   analytics.Overview.TotalUsers,
			Impact:      "medium",
			Action:      "Develop loyalty programs and targeted recommendations",
		})
	}

	return insights
}

func (s *service) generatePersonalInsights(history *UserBookingHistory) []UserPersonalInsight {
	var insights []UserPersonalInsight

	// Example personal insight generation
	if history.Overview.TotalBookings > 10 {
		insights = append(insights, UserPersonalInsight{
			Type:        "achievement",
			Title:       "Frequent Attendee",
			Description: fmt.Sprintf("You've attended %d events! You're truly an event enthusiast.", history.Overview.TotalBookings),
			Value:       fmt.Sprintf("%d events", history.Overview.TotalBookings),
		})
	}

	if history.Overview.TotalSpent > 1000.0 {
		insights = append(insights, UserPersonalInsight{
			Type:        "milestone",
			Title:       "Big Spender",
			Description: fmt.Sprintf("You've invested $%.0f in amazing experiences!", history.Overview.TotalSpent),
			Value:       fmt.Sprintf("$%.0f", history.Overview.TotalSpent),
		})
	}

	if history.Overview.FavoriteVenue != "" {
		insights = append(insights, UserPersonalInsight{
			Type:        "preference",
			Title:       "Venue Loyalty",
			Description: fmt.Sprintf("Your favorite venue is %s - you have great taste!", history.Overview.FavoriteVenue),
			Value:       history.Overview.FavoriteVenue,
		})
	}

	return insights
}

func (s *service) generatePersonalRecommendations(userID uuid.UUID, analytics *PersonalAnalytics) []PersonalRecommendation {
	var recommendations []PersonalRecommendation

	// Example recommendation generation based on user behavior
	if analytics.BookingPatterns.PreferredDay != "" {
		recommendations = append(recommendations, PersonalRecommendation{
			Type:        "time",
			Title:       "Perfect Day Events",
			Description: fmt.Sprintf("New events are available on %s - your preferred day!", analytics.BookingPatterns.PreferredDay),
			Reason:      fmt.Sprintf("You typically book events on %s", analytics.BookingPatterns.PreferredDay),
			Confidence:  0.8,
		})
	}

	if analytics.SpendingInsights.MonthlyAverage > 0 {
		recommendations = append(recommendations, PersonalRecommendation{
			Type:        "event",
			Title:       "Budget-Friendly Options",
			Description: fmt.Sprintf("Check out events under $%.0f to stay within your usual budget", analytics.SpendingInsights.MonthlyAverage),
			Reason:      fmt.Sprintf("Based on your average monthly spending of $%.0f", analytics.SpendingInsights.MonthlyAverage),
			Confidence:  0.7,
		})
	}

	return recommendations
}

func (s *service) calculateAchievements(userID uuid.UUID, analytics *PersonalAnalytics) []Achievement {
	var achievements []Achievement

	// Example achievement calculation
	// This would typically be more sophisticated and stored in the database

	// Early Bird Achievement
	if analytics.BookingPatterns.AdvanceBookingTime > 30 {
		achievements = append(achievements, Achievement{
			ID:          "early_bird",
			Title:       "Early Bird",
			Description: "Books events well in advance",
			Icon:        "🐦",
			UnlockedAt:  time.Now(), // This should come from when they actually unlocked it
			Rarity:      "common",
		})
	}

	// High Roller Achievement
	if analytics.SpendingInsights.MonthlyAverage > 500 {
		achievements = append(achievements, Achievement{
			ID:          "high_roller",
			Title:       "High Roller",
			Description: "Spends significantly on premium events",
			Icon:        "💎",
			UnlockedAt:  time.Now(),
			Rarity:      "rare",
		})
	}

	return achievements
}
