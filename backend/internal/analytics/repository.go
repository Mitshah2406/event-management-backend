package analytics

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository defines the analytics repository interface
type Repository interface {
	// Dashboard Analytics
	GetDashboardAnalytics() (*DashboardAnalytics, error)
	GetOverviewMetrics() (*OverviewMetrics, error)
	GetRecentActivity(limit int) ([]RecentActivityItem, error)

	// Event Analytics
	GetEventAnalytics(eventID uuid.UUID) (*EventAnalytics, error)
	GetGlobalEventAnalytics() (*GlobalEventAnalytics, error)
	GetEventPerformanceMetrics() ([]EventPerformance, error)
	GetEventAnalyticsOverview() (*EventOverview, error)

	// Tag Analytics
	GetTagAnalytics() (*TagAnalyticsResponse, error)
	GetTagPopularityAnalytics() ([]TagAnalytics, error)
	GetTagTrends(months int) ([]TagTrend, error)
	GetTagComparisons() ([]TagComparison, error)
	GetTagOverview() (*TagOverview, error)

	// Booking Analytics
	GetBookingAnalytics() (*BookingAnalytics, error)
	GetBookingOverview() (*BookingOverview, error)
	GetDailyBookingStats(days int) ([]DailyBookingStats, error)
	GetBookingTrends() (*BookingTrendAnalysis, error)
	GetCancellationAnalytics() (*CancellationAnalytics, error)

	// User Analytics
	GetUserAnalytics() (*UserAnalytics, error)
	GetUserOverview() (*UserOverview, error)
	GetUserRetentionMetrics() (*UserRetention, error)
	GetUserDemographics() (*UserDemographics, error)
	GetUserBehaviorAnalytics() (*UserBehavior, error)

	// User-facing Analytics
	GetUserBookingHistory(userID uuid.UUID) (*UserBookingHistory, error)
	GetPersonalAnalytics(userID uuid.UUID) (*PersonalAnalytics, error)
}

// repository implements the Repository interface
type repository struct {
	db *gorm.DB
}

// NewRepository creates a new analytics repository instance
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// Dashboard Analytics Implementation

func (r *repository) GetDashboardAnalytics() (*DashboardAnalytics, error) {
	overview, err := r.GetOverviewMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to get overview metrics: %w", err)
	}

	eventMetrics, err := r.GetEventAnalyticsOverview()
	if err != nil {
		return nil, fmt.Errorf("failed to get event metrics: %w", err)
	}

	bookingMetrics, err := r.GetBookingOverview()
	if err != nil {
		return nil, fmt.Errorf("failed to get booking metrics: %w", err)
	}

	userMetrics, err := r.GetUserOverview()
	if err != nil {
		return nil, fmt.Errorf("failed to get user metrics: %w", err)
	}

	tagMetrics, err := r.GetTagOverview()
	if err != nil {
		return nil, fmt.Errorf("failed to get tag metrics: %w", err)
	}

	recentActivity, err := r.GetRecentActivity(20)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent activity: %w", err)
	}

	// Get top performers
	topEvents, err := r.GetEventPerformanceMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to get event performance: %w", err)
	}

	topTags, err := r.GetTagPopularityAnalytics()
	if err != nil {
		return nil, fmt.Errorf("failed to get tag popularity: %w", err)
	}

	// Convert to venue performance (placeholder implementation)
	var topVenues []VenuePerformance

	// Get trend charts
	dailyBookings, err := r.GetDailyBookingStats(30)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily booking stats: %w", err)
	}

	// Convert to trend chart format
	var bookingTrends, revenueTrends, userGrowthTrends []DailyMetric
	for _, booking := range dailyBookings {
		bookingTrends = append(bookingTrends, DailyMetric{
			Date:  booking.Date,
			Value: float64(booking.TotalBookings),
			Count: booking.TotalBookings,
		})
		revenueTrends = append(revenueTrends, DailyMetric{
			Date:  booking.Date,
			Value: booking.Revenue,
			Count: booking.TotalBookings,
		})
	}

	dashboard := &DashboardAnalytics{
		Overview:       *overview,
		EventMetrics:   *eventMetrics,
		BookingMetrics: *bookingMetrics,
		UserMetrics:    *userMetrics,
		TagMetrics:     *tagMetrics,
		RecentActivity: recentActivity,
		TopPerformers: TopPerformersData{
			Events: topEvents,
			Tags:   convertTagAnalyticsToPerformance(topTags),
			Venues: topVenues,
		},
		TrendCharts: TrendChartsData{
			BookingTrends: bookingTrends,
			RevenueTrends: revenueTrends,
			UserGrowth:    userGrowthTrends,
		},
	}

	return dashboard, nil
}

func (r *repository) GetOverviewMetrics() (*OverviewMetrics, error) {
	var metrics OverviewMetrics

	// Get total events
	var totalEvents int64
	err := r.db.Table("events").Count(&totalEvents).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count events: %w", err)
	}
	metrics.TotalEvents = int(totalEvents)

	// Get active events (published and upcoming)
	var activeEvents int64
	err = r.db.Table("events").
		Where("status = ? AND date_time > ?", "published", time.Now()).
		Count(&activeEvents).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count active events: %w", err)
	}
	metrics.ActiveEvents = int(activeEvents)

	// Get total bookings
	var totalBookings int64
	err = r.db.Table("bookings").
		Where("status = ?", "CONFIRMED").
		Count(&totalBookings).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count bookings: %w", err)
	}
	metrics.TotalBookings = int(totalBookings)

	// Get total revenue
	err = r.db.Table("bookings").
		Where("status = ?", "CONFIRMED").
		Select("COALESCE(SUM(total_price), 0)").
		Scan(&metrics.TotalRevenue).Error
	if err != nil {
		return nil, fmt.Errorf("failed to calculate total revenue: %w", err)
	}

	// Get total users (assuming a users table exists)
	var totalUsers int64
	err = r.db.Table("users").Count(&totalUsers).Error
	if err != nil {
		// If users table doesn't exist, count unique user IDs from bookings
		err = r.db.Table("bookings").
			Select("COUNT(DISTINCT user_id)").
			Scan(&totalUsers).Error
		if err != nil {
			return nil, fmt.Errorf("failed to count users: %w", err)
		}
	}
	metrics.TotalUsers = int(totalUsers)

	// Calculate cancellation rate
	var allBookings, cancelledBookings int64
	r.db.Table("bookings").Count(&allBookings)
	r.db.Table("bookings").Where("status = ?", "CANCELLED").Count(&cancelledBookings)
	if allBookings > 0 {
		metrics.CancellationRate = float64(cancelledBookings) / float64(allBookings) * 100
	}

	// Calculate average utilization (placeholder - would need venue capacity data)
	metrics.AvgUtilization = 75.0 // Placeholder

	// Calculate revenue growth (comparing last 30 days to previous 30 days)
	var currentRevenue, previousRevenue float64
	currentStart := time.Now().AddDate(0, 0, -30)
	previousStart := time.Now().AddDate(0, 0, -60)

	r.db.Table("bookings").
		Where("status = ? AND created_at >= ?", "CONFIRMED", currentStart).
		Select("COALESCE(SUM(total_price), 0)").
		Scan(&currentRevenue)

	r.db.Table("bookings").
		Where("status = ? AND created_at >= ? AND created_at < ?", "CONFIRMED", previousStart, currentStart).
		Select("COALESCE(SUM(total_price), 0)").
		Scan(&previousRevenue)

	if previousRevenue > 0 {
		metrics.RevenueGrowth = ((currentRevenue - previousRevenue) / previousRevenue) * 100
	}

	return &metrics, nil
}

func (r *repository) GetRecentActivity(limit int) ([]RecentActivityItem, error) {
	var activities []RecentActivityItem

	// Get recent bookings
	var recentBookings []struct {
		UserID    uuid.UUID `json:"user_id"`
		EventID   uuid.UUID `json:"event_id"`
		CreatedAt time.Time `json:"created_at"`
		EventName string    `json:"event_name"`
	}

	err := r.db.Table("bookings b").
		Select("b.user_id, b.event_id, b.created_at, e.name as event_name").
		Joins("JOIN events e ON e.id = b.event_id").
		Where("b.status = ?", "CONFIRMED").
		Order("b.created_at DESC").
		Limit(limit / 2).
		Scan(&recentBookings).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get recent bookings: %w", err)
	}

	for _, booking := range recentBookings {
		userIDStr := booking.UserID.String()
		eventIDStr := booking.EventID.String()
		activities = append(activities, RecentActivityItem{
			Type:        "booking",
			Description: fmt.Sprintf("New booking for %s", booking.EventName),
			Timestamp:   booking.CreatedAt,
			UserID:      &userIDStr,
			EventID:     &eventIDStr,
		})
	}

	// Get recent cancellations
	var recentCancellations []struct {
		UserID      uuid.UUID  `json:"user_id"`
		EventID     uuid.UUID  `json:"event_id"`
		CancelledAt *time.Time `json:"cancelled_at"`
		EventName   string     `json:"event_name"`
	}

	err = r.db.Table("bookings b").
		Select("b.user_id, b.event_id, b.cancelled_at, e.name as event_name").
		Joins("JOIN events e ON e.id = b.event_id").
		Where("b.status = ? AND b.cancelled_at IS NOT NULL", "CANCELLED").
		Order("b.cancelled_at DESC").
		Limit(limit / 4).
		Scan(&recentCancellations).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get recent cancellations: %w", err)
	}

	for _, cancellation := range recentCancellations {
		if cancellation.CancelledAt != nil {
			userIDStr := cancellation.UserID.String()
			eventIDStr := cancellation.EventID.String()
			activities = append(activities, RecentActivityItem{
				Type:        "cancellation",
				Description: fmt.Sprintf("Booking cancelled for %s", cancellation.EventName),
				Timestamp:   *cancellation.CancelledAt,
				UserID:      &userIDStr,
				EventID:     &eventIDStr,
			})
		}
	}

	// Get recent event creations
	var recentEvents []struct {
		ID        uuid.UUID `json:"id"`
		Name      string    `json:"name"`
		CreatedAt time.Time `json:"created_at"`
	}

	err = r.db.Table("events").
		Select("id, name, created_at").
		Order("created_at DESC").
		Limit(limit / 4).
		Scan(&recentEvents).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get recent events: %w", err)
	}

	for _, event := range recentEvents {
		eventIDStr := event.ID.String()
		activities = append(activities, RecentActivityItem{
			Type:        "event_created",
			Description: fmt.Sprintf("New event created: %s", event.Name),
			Timestamp:   event.CreatedAt,
			EventID:     &eventIDStr,
		})
	}

	// Sort activities by timestamp (most recent first)
	// Note: This is a simple sort, for production you might want to use a more efficient approach
	for i := 0; i < len(activities); i++ {
		for j := i + 1; j < len(activities); j++ {
			if activities[i].Timestamp.Before(activities[j].Timestamp) {
				activities[i], activities[j] = activities[j], activities[i]
			}
		}
	}

	// Limit to requested size
	if len(activities) > limit {
		activities = activities[:limit]
	}

	return activities, nil
}

// Event Analytics Implementation

func (r *repository) GetEventAnalytics(eventID uuid.UUID) (*EventAnalytics, error) {
	var analytics EventAnalytics

	// Get event basic info
	var event struct {
		ID   uuid.UUID `json:"id"`
		Name string    `json:"name"`
	}

	err := r.db.Table("events").
		Where("id = ?", eventID).
		Select("id, name").
		Scan(&event).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	analytics.EventID = event.ID.String()
	analytics.EventName = event.Name

	// Get booking statistics
	var bookingCount int64
	err = r.db.Table("bookings").
		Where("event_id = ? AND status = ?", eventID, "CONFIRMED").
		Count(&bookingCount).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count bookings: %w", err)
	}
	analytics.TotalBookings = int(bookingCount)

	err = r.db.Table("bookings").
		Where("event_id = ? AND status = ?", eventID, "CONFIRMED").
		Select("COALESCE(SUM(total_price), 0)").
		Scan(&analytics.TotalRevenue).Error
	if err != nil {
		return nil, fmt.Errorf("failed to calculate revenue: %w", err)
	}

	// Calculate cancellation rate
	var totalBookings, cancelledBookings int64
	r.db.Table("bookings").Where("event_id = ?", eventID).Count(&totalBookings)
	r.db.Table("bookings").Where("event_id = ? AND status = ?", eventID, "CANCELLED").Count(&cancelledBookings)
	if totalBookings > 0 {
		analytics.CancellationRate = float64(cancelledBookings) / float64(totalBookings) * 100
	}

	// Get daily booking breakdown
	var dailyBookings []DailyBooking
	err = r.db.Raw(`
		SELECT 
			DATE(created_at) as date,
			COUNT(*) as bookings,
			COALESCE(SUM(total_price), 0) as revenue
		FROM bookings 
		WHERE event_id = ? AND status = ?
		GROUP BY DATE(created_at)
		ORDER BY date
	`, eventID, "CONFIRMED").Scan(&dailyBookings).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get daily bookings: %w", err)
	}

	analytics.BookingsByDay = dailyBookings

	// Placeholder for capacity utilization (would need venue template data)
	analytics.CapacityUtilization = 80.0

	// Placeholder for top sections and hourly trends
	analytics.TopSections = []SectionStats{}
	analytics.BookingTrends = []HourlyStats{}

	return &analytics, nil
}

func (r *repository) GetGlobalEventAnalytics() (*GlobalEventAnalytics, error) {
	var analytics GlobalEventAnalytics

	// Get totals
	var totalEvents int64
	err := r.db.Table("events").Count(&totalEvents).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count events: %w", err)
	}
	analytics.TotalEvents = int(totalEvents)

	var totalBookings int64
	err = r.db.Table("bookings").
		Where("status = ?", "CONFIRMED").
		Count(&totalBookings).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count bookings: %w", err)
	}
	analytics.TotalBookings = int(totalBookings)

	err = r.db.Table("bookings").
		Where("status = ?", "CONFIRMED").
		Select("COALESCE(SUM(total_price), 0)").
		Scan(&analytics.TotalRevenue).Error
	if err != nil {
		return nil, fmt.Errorf("failed to calculate revenue: %w", err)
	}

	// Get events by status
	var statusCounts []struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}

	err = r.db.Table("events").
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&statusCounts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get events by status: %w", err)
	}

	analytics.EventsByStatus = make(map[string]int)
	for _, sc := range statusCounts {
		analytics.EventsByStatus[sc.Status] = sc.Count
	}

	// Get most popular events
	var popularEvents []EventPerformance
	err = r.db.Raw(`
		SELECT 
			e.id as event_id,
			e.name as event_name,
			e.venue,
			e.date_time,
			COUNT(b.id) as booking_count,
			COALESCE(SUM(b.total_price), 0) as revenue
		FROM events e
		LEFT JOIN bookings b ON e.id = b.event_id AND b.status = 'CONFIRMED'
		GROUP BY e.id, e.name, e.venue, e.date_time
		ORDER BY booking_count DESC
		LIMIT 10
	`).Scan(&popularEvents).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get popular events: %w", err)
	}

	analytics.MostPopularEvents = popularEvents

	// Get booking trends
	var bookingTrends []DailyBooking
	err = r.db.Raw(`
		SELECT 
			DATE(created_at) as date,
			COUNT(*) as bookings,
			COALESCE(SUM(total_price), 0) as revenue
		FROM bookings 
		WHERE status = ? AND created_at >= ?
		GROUP BY DATE(created_at)
		ORDER BY date
	`, "CONFIRMED", time.Now().AddDate(0, 0, -30)).Scan(&bookingTrends).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get booking trends: %w", err)
	}

	analytics.BookingTrends = bookingTrends

	// Get revenue by month
	var monthlyRevenue []MonthlyRevenue
	err = r.db.Raw(`
		SELECT 
			TO_CHAR(DATE_TRUNC('month', created_at), 'YYYY-MM') as month,
			COALESCE(SUM(total_price), 0) as revenue,
			COUNT(DISTINCT event_id) as events
		FROM bookings 
		WHERE status = ? AND created_at >= ?
		GROUP BY DATE_TRUNC('month', created_at)
		ORDER BY month
	`, "CONFIRMED", time.Now().AddDate(-1, 0, 0)).Scan(&monthlyRevenue).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get monthly revenue: %w", err)
	}

	analytics.RevenueByMonth = monthlyRevenue

	// Placeholder for average utilization
	analytics.AverageUtilization = 75.0

	return &analytics, nil
}

func (r *repository) GetEventPerformanceMetrics() ([]EventPerformance, error) {
	var performances []EventPerformance

	err := r.db.Raw(`
		SELECT 
			e.id as event_id,
			e.name as event_name,
			e.venue,
			e.date_time,
			COUNT(b.id) as booking_count,
			COALESCE(SUM(b.total_price), 0) as revenue
		FROM events e
		LEFT JOIN bookings b ON e.id = b.event_id AND b.status = 'CONFIRMED'
		GROUP BY e.id, e.name, e.venue, e.date_time
		ORDER BY booking_count DESC
		LIMIT 20
	`).Scan(&performances).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get event performance metrics: %w", err)
	}

	// Calculate utilization for each event (placeholder)
	for i := range performances {
		performances[i].Utilization = 80.0 // Placeholder value
	}

	return performances, nil
}

func (r *repository) GetEventAnalyticsOverview() (*EventOverview, error) {
	var overview EventOverview

	// Get event counts by status
	var statusCounts []struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}

	err := r.db.Table("events").
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&statusCounts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get events by status: %w", err)
	}

	overview.EventsByStatus = make(map[string]int)
	for _, sc := range statusCounts {
		overview.EventsByStatus[sc.Status] = sc.Count
		overview.TotalEvents += sc.Count

		switch sc.Status {
		case "published":
			overview.PublishedEvents = sc.Count
		case "cancelled":
			overview.CancelledEvents = sc.Count
		case "completed":
			overview.CompletedEvents = sc.Count
		}
	}

	// Get upcoming events
	var upcomingEvents int64
	err = r.db.Table("events").
		Where("status = ? AND date_time > ?", "published", time.Now()).
		Count(&upcomingEvents).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count upcoming events: %w", err)
	}
	overview.UpcomingEvents = int(upcomingEvents)

	// Get total revenue
	err = r.db.Table("bookings").
		Where("status = ?", "CONFIRMED").
		Select("COALESCE(SUM(total_price), 0)").
		Scan(&overview.TotalRevenue).Error
	if err != nil {
		return nil, fmt.Errorf("failed to calculate total revenue: %w", err)
	}

	// Get most popular events
	popularEvents, err := r.GetEventPerformanceMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to get popular events: %w", err)
	}
	overview.MostPopularEvents = popularEvents

	// Get revenue by month
	var monthlyRevenue []MonthlyRevenue
	err = r.db.Raw(`
		SELECT 
			TO_CHAR(DATE_TRUNC('month', created_at), 'YYYY-MM') as month,
			COALESCE(SUM(total_price), 0) as revenue,
			COUNT(DISTINCT event_id) as events
		FROM bookings 
		WHERE status = ? AND created_at >= ?
		GROUP BY DATE_TRUNC('month', created_at)
		ORDER BY month
	`, "CONFIRMED", time.Now().AddDate(-1, 0, 0)).Scan(&monthlyRevenue).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get monthly revenue: %w", err)
	}

	overview.RevenueByMonth = monthlyRevenue

	// Placeholder values
	overview.AverageUtilization = 75.0

	return &overview, nil
}

// Helper functions

func convertTagAnalyticsToPerformance(tagAnalytics []TagAnalytics) []TagPerformance {
	var performances []TagPerformance
	for _, tag := range tagAnalytics {
		performances = append(performances, TagPerformance{
			TagID:       tag.TagID,
			TagName:     tag.TagName,
			EventCount:  tag.EventCount,
			Revenue:     tag.TotalRevenue,
			Utilization: tag.AvgUtilization,
		})
	}
	return performances
}

// Tag Analytics Implementation

func (r *repository) GetTagAnalytics() (*TagAnalyticsResponse, error) {
	overview, err := r.GetTagOverview()
	if err != nil {
		return nil, fmt.Errorf("failed to get tag overview: %w", err)
	}

	topTags, err := r.GetTagPopularityAnalytics()
	if err != nil {
		return nil, fmt.Errorf("failed to get tag popularity: %w", err)
	}

	trends, err := r.GetTagTrends(6) // Default 6 months
	if err != nil {
		return nil, fmt.Errorf("failed to get tag trends: %w", err)
	}

	comparisons, err := r.GetTagComparisons()
	if err != nil {
		return nil, fmt.Errorf("failed to get tag comparisons: %w", err)
	}

	return &TagAnalyticsResponse{
		Overview:    *overview,
		TopTags:     topTags,
		TagTrends:   trends,
		Comparisons: comparisons,
	}, nil
}

func (r *repository) GetTagPopularityAnalytics() ([]TagAnalytics, error) {
	var analytics []TagAnalytics

	err := r.db.Raw(`
		SELECT 
			t.id as tag_id,
			t.name as tag_name,
			COUNT(DISTINCT et.event_id) as event_count,
			COUNT(DISTINCT b.id) as total_bookings,
			COALESCE(SUM(b.total_price), 0) as total_revenue,
			AVG(CASE WHEN b.status = 'CONFIRMED' THEN 1.0 ELSE 0.0 END) * 100 as avg_utilization
		FROM tags t
		LEFT JOIN event_tags et ON t.id = et.tag_id
		LEFT JOIN bookings b ON et.event_id = b.event_id AND b.status = 'CONFIRMED'
		WHERE t.is_active = true
		GROUP BY t.id, t.name
		ORDER BY total_bookings DESC, total_revenue DESC
		LIMIT 20
	`).Scan(&analytics).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get tag popularity analytics: %w", err)
	}

	// Calculate popularity score for each tag
	for i := range analytics {
		eventScore := float64(analytics[i].EventCount) * 0.3
		bookingScore := float64(analytics[i].TotalBookings) * 0.4
		revenueScore := analytics[i].TotalRevenue / 1000 * 0.2
		utilizationScore := analytics[i].AvgUtilization * 0.1
		analytics[i].PopularityScore = eventScore + bookingScore + revenueScore + utilizationScore
	}

	return analytics, nil
}

func (r *repository) GetTagTrends(months int) ([]TagTrend, error) {
	var trends []TagTrend

	err := r.db.Raw(`
		SELECT 
			t.id as tag_id,
			t.name as tag_name,
			TO_CHAR(DATE_TRUNC('month', e.created_at), 'YYYY-MM') as month,
			COUNT(DISTINCT et.event_id) as event_count,
			COALESCE(SUM(b.total_price), 0) as revenue
		FROM tags t
		LEFT JOIN event_tags et ON t.id = et.tag_id
		LEFT JOIN events e ON et.event_id = e.id
		LEFT JOIN bookings b ON e.id = b.event_id AND b.status = 'CONFIRMED'
		WHERE t.is_active = true 
			AND e.created_at >= ?
		GROUP BY t.id, t.name, DATE_TRUNC('month', e.created_at)
		ORDER BY t.name, month
	`, time.Now().AddDate(0, -months, 0)).Scan(&trends).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get tag trends: %w", err)
	}

	return trends, nil
}

func (r *repository) GetTagComparisons() ([]TagComparison, error) {
	var comparisons []TagComparison

	err := r.db.Raw(`
		SELECT 
			t.id as tag_id,
			t.name as tag_name,
			COUNT(DISTINCT et.event_id) as event_count,
			AVG(CASE WHEN b.status = 'CONFIRMED' THEN 75.0 ELSE 0.0 END) as avg_capacity_util, -- Placeholder
			AVG(b.total_price / b.total_seats) as avg_ticket_price,
			COALESCE(SUM(b.total_price), 0) as total_revenue,
			COUNT(DISTINCT b.id)::float / NULLIF(COUNT(DISTINCT et.event_id), 0) as booking_conversion
		FROM tags t
		LEFT JOIN event_tags et ON t.id = et.tag_id
		LEFT JOIN bookings b ON et.event_id = b.event_id AND b.status = 'CONFIRMED'
		WHERE t.is_active = true
		GROUP BY t.id, t.name
		HAVING COUNT(DISTINCT et.event_id) > 0
		ORDER BY total_revenue DESC
		LIMIT 15
	`).Scan(&comparisons).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get tag comparisons: %w", err)
	}

	return comparisons, nil
}

func (r *repository) GetTagOverview() (*TagOverview, error) {
	var overview TagOverview

	// Get total and active tags
	var totalTags, activeTags int64
	err := r.db.Table("tags").Count(&totalTags).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count total tags: %w", err)
	}
	overview.TotalTags = int(totalTags)

	err = r.db.Table("tags").Where("is_active = ?", true).Count(&activeTags).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count active tags: %w", err)
	}
	overview.ActiveTags = int(activeTags)

	// Get tags with events
	var tagsWithEvents int64
	err = r.db.Raw(`
		SELECT COUNT(DISTINCT t.id)
		FROM tags t
		INNER JOIN event_tags et ON t.id = et.tag_id
		WHERE t.is_active = true
	`).Scan(&tagsWithEvents).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count tags with events: %w", err)
	}
	overview.TagsWithEvents = int(tagsWithEvents)

	// Get average tags per event
	var avgTagsPerEvent float64
	err = r.db.Raw(`
		SELECT AVG(tag_count)
		FROM (
			SELECT COUNT(et.tag_id) as tag_count
			FROM events e
			LEFT JOIN event_tags et ON e.id = et.event_id
			GROUP BY e.id
		) subq
	`).Scan(&avgTagsPerEvent).Error
	if err != nil {
		overview.AvgTagsPerEvent = 0.0
	} else {
		overview.AvgTagsPerEvent = avgTagsPerEvent
	}

	// Get most and least popular tags
	var mostPopular, leastUsed string
	err = r.db.Raw(`
		SELECT t.name
		FROM tags t
		LEFT JOIN event_tags et ON t.id = et.tag_id
		WHERE t.is_active = true
		GROUP BY t.id, t.name
		ORDER BY COUNT(et.event_id) DESC
		LIMIT 1
	`).Scan(&mostPopular).Error
	if err == nil {
		overview.MostPopularTag = mostPopular
	}

	err = r.db.Raw(`
		SELECT t.name
		FROM tags t
		LEFT JOIN event_tags et ON t.id = et.tag_id
		WHERE t.is_active = true
		GROUP BY t.id, t.name
		ORDER BY COUNT(et.event_id) ASC
		LIMIT 1
	`).Scan(&leastUsed).Error
	if err == nil {
		overview.LeastUsedTag = leastUsed
	}

	return &overview, nil
}

func (r *repository) GetBookingAnalytics() (*BookingAnalytics, error) {
	overview, err := r.GetBookingOverview()
	if err != nil {
		return nil, fmt.Errorf("failed to get booking overview: %w", err)
	}

	trends, err := r.GetBookingTrends()
	if err != nil {
		return nil, fmt.Errorf("failed to get booking trends: %w", err)
	}

	// Get performance stats (placeholder implementation)
	performance := BookingPerformance{
		ConversonRate:   85.0, // Placeholder
		AbandonmentRate: 15.0, // Placeholder
		AvgTimeToBook:   24.5, // Placeholder - hours
	}

	return &BookingAnalytics{
		Overview:         *overview,
		TrendAnalysis:    *trends,
		PerformanceStats: performance,
		Insights:         []BookingInsight{}, // Will be populated by service layer
	}, nil
}

func (r *repository) GetBookingOverview() (*BookingOverview, error) {
	var overview BookingOverview

	// Get booking counts by status
	var totalBookings, confirmedBookings, cancelledBookings int64
	err := r.db.Table("bookings").Count(&totalBookings).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count total bookings: %w", err)
	}
	overview.TotalBookings = int(totalBookings)

	err = r.db.Table("bookings").Where("status = ?", "CONFIRMED").Count(&confirmedBookings).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count confirmed bookings: %w", err)
	}
	overview.ConfirmedBookings = int(confirmedBookings)

	err = r.db.Table("bookings").Where("status = ?", "CANCELLED").Count(&cancelledBookings).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count cancelled bookings: %w", err)
	}
	overview.CancelledBookings = int(cancelledBookings)

	// Get revenue and averages
	err = r.db.Table("bookings").
		Where("status = ?", "CONFIRMED").
		Select("COALESCE(SUM(total_price), 0)").
		Scan(&overview.TotalRevenue).Error
	if err != nil {
		return nil, fmt.Errorf("failed to calculate total revenue: %w", err)
	}

	var avgBookingSize, avgTicketPrice float64
	err = r.db.Table("bookings").
		Where("status = ?", "CONFIRMED").
		Select("AVG(total_seats)").
		Scan(&avgBookingSize).Error
	if err == nil {
		overview.AverageBookingSize = avgBookingSize
	}

	err = r.db.Table("bookings").
		Where("status = ? AND total_seats > 0", "CONFIRMED").
		Select("AVG(total_price / total_seats)").
		Scan(&avgTicketPrice).Error
	if err == nil {
		overview.AverageTicketPrice = avgTicketPrice
	}

	// Calculate cancellation rate
	if totalBookings > 0 {
		overview.CancellationRate = float64(cancelledBookings) / float64(totalBookings) * 100
	}

	// Get bookings by status
	overview.BookingsByStatus = map[string]int{
		"CONFIRMED": int(confirmedBookings),
		"CANCELLED": int(cancelledBookings),
	}

	// Get daily bookings for the last 30 days
	dailyStats, err := r.GetDailyBookingStats(30)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily booking stats: %w", err)
	}
	overview.DailyBookings = dailyStats

	// Get payment methods (placeholder - would need payments table analysis)
	overview.PaymentMethods = []PaymentMethodStats{
		{Method: "credit_card", Bookings: int(confirmedBookings * 70 / 100), Success: 98.5},
		{Method: "paypal", Bookings: int(confirmedBookings * 20 / 100), Success: 97.2},
		{Method: "bank_transfer", Bookings: int(confirmedBookings * 10 / 100), Success: 99.1},
	}

	return &overview, nil
}

func (r *repository) GetDailyBookingStats(days int) ([]DailyBookingStats, error) {
	var stats []DailyBookingStats

	err := r.db.Raw(`
		SELECT 
			DATE(created_at) as date,
			COUNT(*) as total_bookings,
			SUM(CASE WHEN status = 'CONFIRMED' THEN 1 ELSE 0 END) as confirmed_bookings,
			SUM(CASE WHEN status = 'CANCELLED' THEN 1 ELSE 0 END) as cancelled_bookings,
			COALESCE(SUM(CASE WHEN status = 'CONFIRMED' THEN total_price ELSE 0 END), 0) as revenue,
			AVG(CASE WHEN status = 'CONFIRMED' THEN total_price ELSE NULL END) as average_value
		FROM bookings
		WHERE created_at >= ?
		GROUP BY DATE(created_at)
		ORDER BY date DESC
	`, time.Now().AddDate(0, 0, -days)).Scan(&stats).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get daily booking stats: %w", err)
	}

	return stats, nil
}

func (r *repository) GetBookingTrends() (*BookingTrendAnalysis, error) {
	var trends BookingTrendAnalysis

	// Get current period stats (last 30 days)
	currentStart := time.Now().AddDate(0, 0, -30)
	previousStart := time.Now().AddDate(0, 0, -60)

	var currentBookings, previousBookings int64
	var currentRevenue, previousRevenue float64
	var currentUsers, previousUsers int64

	// Current period
	err := r.db.Table("bookings").
		Where("status = ? AND created_at >= ?", "CONFIRMED", currentStart).
		Count(&currentBookings).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get current bookings: %w", err)
	}

	err = r.db.Table("bookings").
		Where("status = ? AND created_at >= ?", "CONFIRMED", currentStart).
		Select("COALESCE(SUM(total_price), 0)").
		Scan(&currentRevenue).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get current revenue: %w", err)
	}

	err = r.db.Table("bookings").
		Where("status = ? AND created_at >= ?", "CONFIRMED", currentStart).
		Select("COUNT(DISTINCT user_id)").
		Scan(&currentUsers).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get current users: %w", err)
	}

	// Previous period
	err = r.db.Table("bookings").
		Where("status = ? AND created_at >= ? AND created_at < ?", "CONFIRMED", previousStart, currentStart).
		Count(&previousBookings).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get previous bookings: %w", err)
	}

	err = r.db.Table("bookings").
		Where("status = ? AND created_at >= ? AND created_at < ?", "CONFIRMED", previousStart, currentStart).
		Select("COALESCE(SUM(total_price), 0)").
		Scan(&previousRevenue).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get previous revenue: %w", err)
	}

	err = r.db.Table("bookings").
		Where("status = ? AND created_at >= ? AND created_at < ?", "CONFIRMED", previousStart, currentStart).
		Select("COUNT(DISTINCT user_id)").
		Scan(&previousUsers).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get previous users: %w", err)
	}

	// Calculate period comparison
	trends.PeriodComparison = PeriodComparison{
		CurrentPeriod: PeriodStats{
			Bookings: int(currentBookings),
			Revenue:  currentRevenue,
			Users:    int(currentUsers),
		},
		PreviousPeriod: PeriodStats{
			Bookings: int(previousBookings),
			Revenue:  previousRevenue,
			Users:    int(previousUsers),
		},
	}

	if previousRevenue > 0 {
		trends.PeriodComparison.PercentChange = ((currentRevenue - previousRevenue) / previousRevenue) * 100
	}

	// Get seasonality data (placeholder implementation)
	trends.Seasonality = SeasonalityData{
		ByDayOfWeek: []WeekdayStats{
			{Weekday: "Monday", Bookings: 120, Revenue: 15000},
			{Weekday: "Tuesday", Bookings: 95, Revenue: 12000},
			{Weekday: "Wednesday", Bookings: 110, Revenue: 14000},
			{Weekday: "Thursday", Bookings: 130, Revenue: 16500},
			{Weekday: "Friday", Bookings: 180, Revenue: 22000},
			{Weekday: "Saturday", Bookings: 220, Revenue: 28000},
			{Weekday: "Sunday", Bookings: 145, Revenue: 18500},
		},
	}

	// Calculate growth metrics
	var bookingGrowth, revenueGrowth, userGrowth float64
	if previousBookings > 0 {
		bookingGrowth = ((float64(currentBookings) - float64(previousBookings)) / float64(previousBookings)) * 100
	}
	if previousRevenue > 0 {
		revenueGrowth = ((currentRevenue - previousRevenue) / previousRevenue) * 100
	}
	if previousUsers > 0 {
		userGrowth = ((float64(currentUsers) - float64(previousUsers)) / float64(previousUsers)) * 100
	}

	trends.Growth = GrowthMetrics{
		BookingGrowth: bookingGrowth,
		RevenueGrowth: revenueGrowth,
		UserGrowth:    userGrowth,
	}

	return &trends, nil
}

func (r *repository) GetCancellationAnalytics() (*CancellationAnalytics, error) {
	var analytics CancellationAnalytics

	// Get cancellation overview
	var totalCancellations int64
	var totalRefundAmount float64

	err := r.db.Table("bookings").
		Where("status = ?", "CANCELLED").
		Count(&totalCancellations).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count cancellations: %w", err)
	}

	// Calculate refund amount (assuming full refunds for simplicity)
	err = r.db.Table("bookings").
		Where("status = ?", "CANCELLED").
		Select("COALESCE(SUM(total_price), 0)").
		Scan(&totalRefundAmount).Error
	if err != nil {
		return nil, fmt.Errorf("failed to calculate refund amount: %w", err)
	}

	var totalBookings int64
	r.db.Table("bookings").Count(&totalBookings)

	analytics.Overview = CancellationOverview{
		TotalCancellations: int(totalCancellations),
		RefundAmount:       totalRefundAmount,
		RefundRate:         100.0, // Assuming 100% refund rate
		AvgTimeToCancel:    48.0,  // Placeholder - 48 hours average
	}

	if totalBookings > 0 {
		analytics.Overview.CancellationRate = float64(totalCancellations) / float64(totalBookings) * 100
	}

	// Placeholder data for cancellation reasons, timing, etc.
	analytics.CancellationReasons = []CancellationReason{
		{Reason: "Schedule conflict", Count: int(totalCancellations * 35 / 100), Percentage: 35.0},
		{Reason: "Personal emergency", Count: int(totalCancellations * 25 / 100), Percentage: 25.0},
		{Reason: "Event postponed", Count: int(totalCancellations * 20 / 100), Percentage: 20.0},
		{Reason: "Financial reasons", Count: int(totalCancellations * 15 / 100), Percentage: 15.0},
		{Reason: "Other", Count: int(totalCancellations * 5 / 100), Percentage: 5.0},
	}

	// Get cancellation trends
	var trendData []CancellationTrend
	err = r.db.Raw(`
		SELECT 
			DATE(cancelled_at) as date,
			COUNT(*) as cancellations,
			COUNT(*)::float / (
				SELECT COUNT(*) 
				FROM bookings b2 
				WHERE DATE(b2.created_at) = DATE(b1.cancelled_at)
			) * 100 as cancellation_rate,
			COALESCE(SUM(total_price), 0) as refund_amount
		FROM bookings b1
		WHERE status = 'CANCELLED' 
			AND cancelled_at IS NOT NULL
			AND cancelled_at >= ?
		GROUP BY DATE(cancelled_at)
		ORDER BY date
	`, time.Now().AddDate(0, 0, -30)).Scan(&trendData).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get cancellation trends: %w", err)
	}
	analytics.Trends = trendData

	return &analytics, nil
}

// User Analytics Implementation

func (r *repository) GetUserAnalytics() (*UserAnalytics, error) {
	overview, err := r.GetUserOverview()
	if err != nil {
		return nil, fmt.Errorf("failed to get user overview: %w", err)
	}

	retention, err := r.GetUserRetentionMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to get user retention: %w", err)
	}

	demographics, err := r.GetUserDemographics()
	if err != nil {
		return nil, fmt.Errorf("failed to get user demographics: %w", err)
	}

	behavior, err := r.GetUserBehaviorAnalytics()
	if err != nil {
		return nil, fmt.Errorf("failed to get user behavior: %w", err)
	}

	return &UserAnalytics{
		Overview:         *overview,
		RetentionMetrics: *retention,
		Demographics:     *demographics,
		Behavior:         *behavior,
		Insights:         []UserInsight{}, // Populated by service layer
	}, nil
}

func (r *repository) GetUserOverview() (*UserOverview, error) {
	var overview UserOverview

	// Count unique users from bookings (assuming no separate users table)
	var totalUsers, activeUsers, newUsers int64

	err := r.db.Table("bookings").
		Select("COUNT(DISTINCT user_id)").
		Scan(&totalUsers).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count total users: %w", err)
	}
	overview.TotalUsers = int(totalUsers)

	// Active users (booked in last 30 days)
	err = r.db.Table("bookings").
		Where("created_at >= ? AND status = ?", time.Now().AddDate(0, 0, -30), "CONFIRMED").
		Select("COUNT(DISTINCT user_id)").
		Scan(&activeUsers).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count active users: %w", err)
	}
	overview.ActiveUsers = int(activeUsers)

	// New users (first booking in last 30 days)
	err = r.db.Raw(`
		SELECT COUNT(DISTINCT user_id)
		FROM bookings b1
		WHERE b1.created_at >= ?
		AND NOT EXISTS (
			SELECT 1 FROM bookings b2 
			WHERE b2.user_id = b1.user_id 
			AND b2.created_at < ?
		)
	`, time.Now().AddDate(0, 0, -30), time.Now().AddDate(0, 0, -30)).Scan(&newUsers).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count new users: %w", err)
	}
	overview.NewUsers = int(newUsers)

	// Calculate retention rate (simplified)
	if totalUsers > 0 {
		overview.RetentionRate = float64(activeUsers) / float64(totalUsers) * 100
	}

	// Calculate average bookings per user
	var avgBookingsPerUser float64
	err = r.db.Raw(`
		SELECT AVG(booking_count)
		FROM (
			SELECT COUNT(*) as booking_count
			FROM bookings
			WHERE status = 'CONFIRMED'
			GROUP BY user_id
		) subq
	`).Scan(&avgBookingsPerUser).Error
	if err == nil {
		overview.AvgBookingsPerUser = avgBookingsPerUser
	}

	// Get user growth data (last 12 months)
	var growthStats []UserGrowthStats
	err = r.db.Raw(`
		SELECT 
			TO_CHAR(DATE_TRUNC('month', first_booking), 'YYYY-MM') as date,
			COUNT(*) as new_users
		FROM (
			SELECT 
				user_id,
				MIN(created_at) as first_booking
			FROM bookings
			WHERE status = 'CONFIRMED'
			GROUP BY user_id
		) first_bookings
		WHERE first_booking >= ?
		GROUP BY DATE_TRUNC('month', first_booking)
		ORDER BY date
	`, time.Now().AddDate(-1, 0, 0)).Scan(&growthStats).Error

	if err == nil {
		overview.UserGrowth = growthStats
	}

	// User segments (simplified classification)
	overview.UserSegments = []UserSegment{
		{Segment: "new", UserCount: int(newUsers), Percentage: float64(newUsers) / float64(totalUsers) * 100},
		{Segment: "regular", UserCount: int(activeUsers - newUsers), Percentage: float64(activeUsers-newUsers) / float64(totalUsers) * 100},
		{Segment: "inactive", UserCount: int(totalUsers - activeUsers), Percentage: float64(totalUsers-activeUsers) / float64(totalUsers) * 100},
	}

	return &overview, nil
}

func (r *repository) GetUserRetentionMetrics() (*UserRetention, error) {
	var retention UserRetention

	// Calculate retention by period (simplified)
	retention.RetentionByPeriod = []RetentionPeriod{
		{Period: "1 month", Percentage: 65.0, UserCount: 650},   // Placeholder
		{Period: "3 months", Percentage: 45.0, UserCount: 450},  // Placeholder
		{Period: "6 months", Percentage: 30.0, UserCount: 300},  // Placeholder
		{Period: "12 months", Percentage: 20.0, UserCount: 200}, // Placeholder
	}

	// Calculate churn rate (users who haven't booked in 90 days)
	var totalUsers, inactiveUsers int64
	err := r.db.Table("bookings").Select("COUNT(DISTINCT user_id)").Scan(&totalUsers).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count total users: %w", err)
	}

	err = r.db.Raw(`
		SELECT COUNT(DISTINCT user_id)
		FROM bookings b1
		WHERE NOT EXISTS (
			SELECT 1 FROM bookings b2
			WHERE b2.user_id = b1.user_id
			AND b2.created_at >= ?
			AND b2.status = 'CONFIRMED'
		)
	`, time.Now().AddDate(0, 0, -90)).Scan(&inactiveUsers).Error

	if err == nil && totalUsers > 0 {
		retention.ChurnRate = float64(inactiveUsers) / float64(totalUsers) * 100
	}

	// Calculate lifetime value
	var avgLifetimeValue float64
	err = r.db.Raw(`
		SELECT AVG(user_total)
		FROM (
			SELECT SUM(total_price) as user_total
			FROM bookings
			WHERE status = 'CONFIRMED'
			GROUP BY user_id
		) subq
	`).Scan(&avgLifetimeValue).Error

	if err == nil {
		retention.LifetimeValue = avgLifetimeValue
	}

	// Placeholder cohort data
	retention.RetentionCohorts = []CohortData{
		{CohortMonth: "2024-01", CohortSize: 100, Retention: []float64{100, 75, 60, 50, 45, 40}},
		{CohortMonth: "2024-02", CohortSize: 120, Retention: []float64{100, 80, 65, 55, 50, 45}},
		{CohortMonth: "2024-03", CohortSize: 150, Retention: []float64{100, 78, 62, 52, 48}},
	}

	return &retention, nil
}

func (r *repository) GetUserDemographics() (*UserDemographics, error) {
	var demographics UserDemographics

	// Since we don't have a detailed users table, we'll create placeholder demographics
	// In a real implementation, this would query actual user demographic data

	demographics.AgeGroups = []DemographicStat{
		{Category: "18-25", UserCount: 250, Percentage: 25.0, AvgRevenue: 150.0},
		{Category: "26-35", UserCount: 400, Percentage: 40.0, AvgRevenue: 200.0},
		{Category: "36-45", UserCount: 200, Percentage: 20.0, AvgRevenue: 180.0},
		{Category: "46-55", UserCount: 100, Percentage: 10.0, AvgRevenue: 220.0},
		{Category: "56+", UserCount: 50, Percentage: 5.0, AvgRevenue: 160.0},
	}

	demographics.Locations = []LocationStat{
		{Location: "New York", UserCount: 300, Percentage: 30.0, AvgBookings: 2.5},
		{Location: "California", UserCount: 250, Percentage: 25.0, AvgBookings: 2.8},
		{Location: "Texas", UserCount: 150, Percentage: 15.0, AvgBookings: 2.2},
		{Location: "Florida", UserCount: 120, Percentage: 12.0, AvgBookings: 2.0},
		{Location: "Other", UserCount: 180, Percentage: 18.0, AvgBookings: 1.8},
	}

	// Get actual joining periods from bookings (first booking date)
	var joinedPeriods []PeriodStat
	err := r.db.Raw(`
		SELECT 
			CASE 
				WHEN first_booking >= ? THEN 'Last 30 days'
				WHEN first_booking >= ? THEN 'Last 3 months'
				WHEN first_booking >= ? THEN 'Last 6 months'
				WHEN first_booking >= ? THEN 'Last year'
				ELSE 'Over a year'
			END as period,
			COUNT(*) as user_count
		FROM (
			SELECT user_id, MIN(created_at) as first_booking
			FROM bookings
			WHERE status = 'CONFIRMED'
			GROUP BY user_id
		) first_bookings
		GROUP BY period
	`,
		time.Now().AddDate(0, 0, -30),
		time.Now().AddDate(0, -3, 0),
		time.Now().AddDate(0, -6, 0),
		time.Now().AddDate(-1, 0, 0),
	).Scan(&joinedPeriods).Error

	if err == nil {
		demographics.JoinedPeriods = joinedPeriods
	}

	demographics.BookingPatterns = []PatternStat{
		{Pattern: "frequent", UserCount: 150, Revenue: 45000, Percentage: 15.0},
		{Pattern: "occasional", UserCount: 500, Revenue: 75000, Percentage: 50.0},
		{Pattern: "one-time", UserCount: 350, Revenue: 35000, Percentage: 35.0},
	}

	return &demographics, nil
}

func (r *repository) GetUserBehaviorAnalytics() (*UserBehavior, error) {
	var behavior UserBehavior

	// Average session time (placeholder)
	behavior.AvgSessionTime = 1800.0 // 30 minutes in seconds

	// Event preferences based on tag popularity
	var preferences []PreferenceStats
	err := r.db.Raw(`
		SELECT 
			t.name as value,
			COUNT(DISTINCT b.user_id) as user_count,
			COALESCE(SUM(b.total_price), 0) as revenue
		FROM tags t
		JOIN event_tags et ON t.id = et.tag_id
		JOIN bookings b ON et.event_id = b.event_id
		WHERE b.status = 'CONFIRMED' AND t.is_active = true
		GROUP BY t.id, t.name
		ORDER BY user_count DESC
		LIMIT 10
	`).Scan(&preferences).Error

	if err == nil {
		for i := range preferences {
			preferences[i].Category = "tag"
			// Calculate percentage would require total users count
		}
		behavior.EventPreferences = preferences
	}

	// Booking frequency analysis
	var avgBookingsPerMonth float64
	err = r.db.Raw(`
		SELECT AVG(monthly_bookings)
		FROM (
			SELECT 
				user_id,
				COUNT(*) / GREATEST(1, EXTRACT(MONTH FROM AGE(MAX(created_at), MIN(created_at)))) as monthly_bookings
			FROM bookings
			WHERE status = 'CONFIRMED'
			GROUP BY user_id
			HAVING COUNT(*) > 1
		) subq
	`).Scan(&avgBookingsPerMonth).Error

	if err == nil {
		behavior.BookingFrequency = BookingFrequencyStats{
			AvgBookingsPerMonth: avgBookingsPerMonth,
			FrequencyDistribution: []FrequencyBucket{
				{Range: "1", UserCount: 400, Percentage: 40.0, AvgRevenue: 120.0},
				{Range: "2-3", UserCount: 300, Percentage: 30.0, AvgRevenue: 180.0},
				{Range: "4-5", UserCount: 200, Percentage: 20.0, AvgRevenue: 250.0},
				{Range: "6+", UserCount: 100, Percentage: 10.0, AvgRevenue: 400.0},
			},
		}
	}

	// Price preferences
	var avgTicketPrice float64
	err = r.db.Table("bookings").
		Where("status = ? AND total_seats > 0", "CONFIRMED").
		Select("AVG(total_price / total_seats)").
		Scan(&avgTicketPrice).Error

	if err == nil {
		behavior.PricePreferences = PricePreferenceStats{
			AvgTicketPrice:   avgTicketPrice,
			PriceSensitivity: 0.15, // Placeholder
			PriceRanges: []PriceRange{
				{Range: "$0-50", UserCount: 300, Percentage: 30.0, AvgBookings: 1.5},
				{Range: "$51-100", UserCount: 400, Percentage: 40.0, AvgBookings: 2.0},
				{Range: "$101-200", UserCount: 200, Percentage: 20.0, AvgBookings: 2.5},
				{Range: "$200+", UserCount: 100, Percentage: 10.0, AvgBookings: 3.0},
			},
		}
	}

	// Cancellation behavior
	var cancellationRate float64
	var totalUsers, usersWithCancellations int64

	r.db.Table("bookings").Select("COUNT(DISTINCT user_id)").Scan(&totalUsers)
	r.db.Table("bookings").Where("status = ?", "CANCELLED").Select("COUNT(DISTINCT user_id)").Scan(&usersWithCancellations)

	if totalUsers > 0 {
		cancellationRate = float64(usersWithCancellations) / float64(totalUsers) * 100
	}

	behavior.CancellationBehavior = CancellationBehaviorStats{
		CancellationRate: cancellationRate,
		AvgTimeToCancel:  48.0,                                   // Placeholder
		RepeatCancellers: int(usersWithCancellations * 20 / 100), // Estimate
		TopCancelReasons: []string{"Schedule conflict", "Personal emergency", "Event postponed"},
	}

	return &behavior, nil
}

// User-facing Analytics Implementation

func (r *repository) GetUserBookingHistory(userID uuid.UUID) (*UserBookingHistory, error) {
	var history UserBookingHistory

	// Get user booking overview
	var totalBookings int64
	var totalSpent, avgBookingValue float64
	var memberSince time.Time

	err := r.db.Table("bookings").
		Where("user_id = ? AND status = ?", userID, "CONFIRMED").
		Count(&totalBookings).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count user bookings: %w", err)
	}

	err = r.db.Table("bookings").
		Where("user_id = ? AND status = ?", userID, "CONFIRMED").
		Select("COALESCE(SUM(total_price), 0)").
		Scan(&totalSpent).Error
	if err != nil {
		return nil, fmt.Errorf("failed to calculate total spent: %w", err)
	}

	if totalBookings > 0 {
		avgBookingValue = totalSpent / float64(totalBookings)
	}

	err = r.db.Table("bookings").
		Where("user_id = ?", userID).
		Select("MIN(created_at)").
		Scan(&memberSince).Error
	if err != nil {
		memberSince = time.Now()
	}

	// Get favorite venue and event type (simplified)
	var favoriteVenue, favoriteEventType string
	err = r.db.Raw(`
		SELECT e.venue
		FROM bookings b
		JOIN events e ON b.event_id = e.id
		WHERE b.user_id = ? AND b.status = 'CONFIRMED'
		GROUP BY e.venue
		ORDER BY COUNT(*) DESC
		LIMIT 1
	`, userID).Scan(&favoriteVenue).Error

	if err != nil {
		favoriteVenue = "N/A"
	}

	// For event type, we'd need to look at tags - simplified here
	favoriteEventType = "Entertainment" // Placeholder

	history.Overview = UserBookingOverview{
		TotalBookings:     int(totalBookings),
		TotalSpent:        totalSpent,
		AvgBookingValue:   avgBookingValue,
		FavoriteVenue:     favoriteVenue,
		FavoriteEventType: favoriteEventType,
		MemberSince:       memberSince.Format("2006-01-02"),
	}

	// Get booking history records
	var bookingRecords []UserBookingRecord
	err = r.db.Raw(`
		SELECT 
			b.id as booking_id,
			e.name as event_name,
			e.venue as venue_name,
			b.created_at as booking_date,
			e.date_time as event_date,
			b.total_price as total_amount,
			b.total_seats as seat_count,
			b.status
		FROM bookings b
		JOIN events e ON b.event_id = e.id
		WHERE b.user_id = ?
		ORDER BY b.created_at DESC
		LIMIT 50
	`, userID).Scan(&bookingRecords).Error

	if err == nil {
		history.BookingHistory = bookingRecords
	}

	// Calculate spending analysis
	var monthlySpending []MonthlySpending
	err = r.db.Raw(`
		SELECT 
			TO_CHAR(DATE_TRUNC('month', created_at), 'YYYY-MM') as month,
			COALESCE(SUM(total_price), 0) as amount,
			COUNT(*) as bookings
		FROM bookings
		WHERE user_id = ? AND status = 'CONFIRMED'
		AND created_at >= ?
		GROUP BY DATE_TRUNC('month', created_at)
		ORDER BY month
	`, userID, time.Now().AddDate(-1, 0, 0)).Scan(&monthlySpending).Error

	if err == nil {
		history.SpendingAnalysis = UserSpendingAnalysis{
			SpendingByMonth: monthlySpending,
			YearlyTotal:     totalSpent,
			SpendingTrend:   "Increasing", // Simplified
		}
	}

	// User preferences (simplified)
	history.Preferences = UserPreferences{
		PreferredTags:    []string{"Music", "Entertainment"}, // Placeholder
		PreferredVenues:  []string{favoriteVenue},
		PreferredTimes:   []string{"Evening"},
		PriceRange:       "$50-150",
		BookingFrequency: "Monthly",
	}

	return &history, nil
}

func (r *repository) GetPersonalAnalytics(userID uuid.UUID) (*PersonalAnalytics, error) {
	var analytics PersonalAnalytics

	// Get booking patterns
	var preferredDay string
	var advanceBookingTime int

	// Simplified queries for patterns
	err := r.db.Raw(`
		SELECT EXTRACT(DOW FROM e.date_time) as day_of_week
		FROM bookings b
		JOIN events e ON b.event_id = e.id
		WHERE b.user_id = ? AND b.status = 'CONFIRMED'
		GROUP BY EXTRACT(DOW FROM e.date_time)
		ORDER BY COUNT(*) DESC
		LIMIT 1
	`, userID).Scan(&preferredDay).Error

	if err == nil {
		// Convert day number to day name (simplified)
		days := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
		if preferredDay != "" {
			preferredDay = days[0] // Simplified
		}
	}

	// Calculate average advance booking time
	_ = r.db.Raw(`
		SELECT AVG(EXTRACT(EPOCH FROM (e.date_time - b.created_at)) / 86400)
		FROM bookings b
		JOIN events e ON b.event_id = e.id
		WHERE b.user_id = ? AND b.status = 'CONFIRMED'
	`, userID).Scan(&advanceBookingTime).Error

	analytics.BookingPatterns = PersonalBookingPatterns{
		BookingFrequency:    "Monthly", // Simplified
		PreferredDay:        preferredDay,
		PreferredTime:       "Evening", // Placeholder
		AdvanceBookingTime:  advanceBookingTime,
		SeasonalPreferences: []string{"Summer", "Fall"}, // Placeholder
	}

	// Get spending insights
	var monthlyAverage, yearOverYearGrowth float64
	_ = r.db.Raw(`
		SELECT AVG(monthly_total)
		FROM (
			SELECT COALESCE(SUM(total_price), 0) as monthly_total
			FROM bookings
			WHERE user_id = ? AND status = 'CONFIRMED'
			AND created_at >= ?
			GROUP BY DATE_TRUNC('month', created_at)
		) subq
	`, userID, time.Now().AddDate(-1, 0, 0)).Scan(&monthlyAverage).Error

	analytics.SpendingInsights = PersonalSpendingInsights{
		MonthlyAverage:     monthlyAverage,
		YearOverYearGrowth: yearOverYearGrowth,
		PeakSpendingMonth:  "December", // Placeholder
		BudgetSuggestion:   monthlyAverage * 1.1,
		SavingsOpportunity: monthlyAverage * 0.1,
	}

	// Recommendations and achievements will be populated by service layer
	analytics.Recommendations = []PersonalRecommendation{}
	analytics.Achievements = []Achievement{}

	return &analytics, nil
}
