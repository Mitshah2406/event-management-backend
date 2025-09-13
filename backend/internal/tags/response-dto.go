package tags

import "time"

type TagResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Color       string    `json:"color"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PaginatedTags struct {
	Tags       []TagResponse `json:"tags"`
	TotalCount int64         `json:"total_count"`
	Page       int           `json:"page"`
	Limit      int           `json:"limit"`
	TotalPages int           `json:"total_pages"`
}

// Tag Analytics
type TagAnalytics struct {
	TagID           string  `json:"tag_id"`
	TagName         string  `json:"tag_name"`
	EventCount      int     `json:"event_count"`
	TotalBookings   int     `json:"total_bookings"`
	TotalRevenue    float64 `json:"total_revenue"`
	AvgUtilization  float64 `json:"avg_utilization"`
	PopularityScore float64 `json:"popularity_score"` // Calculated metric
}

type TagAnalyticsResponse struct {
	Overview    TagOverview     `json:"overview"`
	TopTags     []TagAnalytics  `json:"top_tags"`
	TagTrends   []TagTrend      `json:"tag_trends"`
	Comparisons []TagComparison `json:"comparisons"`
}

type TagOverview struct {
	TotalTags       int     `json:"total_tags"`
	ActiveTags      int     `json:"active_tags"`
	TagsWithEvents  int     `json:"tags_with_events"`
	AvgTagsPerEvent float64 `json:"avg_tags_per_event"`
	MostPopularTag  string  `json:"most_popular_tag"`
	LeastUsedTag    string  `json:"least_used_tag"`
}

type TagTrend struct {
	TagID      string  `json:"tag_id"`
	TagName    string  `json:"tag_name"`
	Month      string  `json:"month"`
	EventCount int     `json:"event_count"`
	Revenue    float64 `json:"revenue"`
}

type TagComparison struct {
	TagID             string  `json:"tag_id"`
	TagName           string  `json:"tag_name"`
	EventCount        int     `json:"event_count"`
	AvgCapacityUtil   float64 `json:"avg_capacity_utilization"`
	AvgTicketPrice    float64 `json:"avg_ticket_price"`
	TotalRevenue      float64 `json:"total_revenue"`
	BookingConversion float64 `json:"booking_conversion"`
}
