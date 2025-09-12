package analytics

import (
	"time"
)

// Dashboard & Overview Models

type DashboardAnalytics struct {
	Overview       OverviewMetrics      `json:"overview"`
	EventMetrics   EventOverview        `json:"event_metrics"`
	BookingMetrics BookingOverview      `json:"booking_metrics"`
	UserMetrics    UserOverview         `json:"user_metrics"`
	TagMetrics     TagOverview          `json:"tag_metrics"`
	RecentActivity []RecentActivityItem `json:"recent_activity"`
	TopPerformers  TopPerformersData    `json:"top_performers"`
	TrendCharts    TrendChartsData      `json:"trend_charts"`
}

type OverviewMetrics struct {
	TotalEvents      int     `json:"total_events"`
	TotalBookings    int     `json:"total_bookings"`
	TotalRevenue     float64 `json:"total_revenue"`
	TotalUsers       int     `json:"total_users"`
	ActiveEvents     int     `json:"active_events"`
	CancellationRate float64 `json:"cancellation_rate"`
	AvgUtilization   float64 `json:"avg_utilization"`
	RevenueGrowth    float64 `json:"revenue_growth"`
}

type RecentActivityItem struct {
	Type        string    `json:"type"` // "booking", "cancellation", "event_created"
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
	UserID      *string   `json:"user_id,omitempty"`
	EventID     *string   `json:"event_id,omitempty"`
}

type TopPerformersData struct {
	Events []EventPerformance `json:"events"`
	Tags   []TagPerformance   `json:"tags"`
	Venues []VenuePerformance `json:"venues"`
}

type TrendChartsData struct {
	BookingTrends []DailyMetric `json:"booking_trends"`
	RevenueTrends []DailyMetric `json:"revenue_trends"`
	UserGrowth    []DailyMetric `json:"user_growth"`
}

type DailyMetric struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
	Count int     `json:"count,omitempty"`
}

// Event Analytics Models (migrated from events package)

type EventOverview struct {
	TotalEvents        int                `json:"total_events"`
	PublishedEvents    int                `json:"published_events"`
	CancelledEvents    int                `json:"cancelled_events"`
	CompletedEvents    int                `json:"completed_events"`
	UpcomingEvents     int                `json:"upcoming_events"`
	AverageUtilization float64            `json:"average_utilization"`
	TotalRevenue       float64            `json:"total_revenue"`
	EventsByStatus     map[string]int     `json:"events_by_status"`
	MostPopularEvents  []EventPerformance `json:"most_popular_events"`
	RevenueByMonth     []MonthlyRevenue   `json:"revenue_by_month"`
}

type EventAnalytics struct {
	EventID             string         `json:"event_id"`
	EventName           string         `json:"event_name"`
	TotalBookings       int            `json:"total_bookings"`
	TotalRevenue        float64        `json:"total_revenue"`
	CapacityUtilization float64        `json:"capacity_utilization"`
	CancellationRate    float64        `json:"cancellation_rate"`
	BookingsByDay       []DailyBooking `json:"bookings_by_day"`
	TopSections         []SectionStats `json:"top_sections"`
	BookingTrends       []HourlyStats  `json:"booking_trends"`
}

type GlobalEventAnalytics struct {
	TotalEvents        int                `json:"total_events"`
	TotalBookings      int                `json:"total_bookings"`
	TotalRevenue       float64            `json:"total_revenue"`
	AverageUtilization float64            `json:"average_utilization"`
	MostPopularEvents  []EventPerformance `json:"most_popular_events"`
	BookingTrends      []DailyBooking     `json:"booking_trends"`
	EventsByStatus     map[string]int     `json:"events_by_status"`
	RevenueByMonth     []MonthlyRevenue   `json:"revenue_by_month"`
}

type EventPerformance struct {
	EventID      string  `json:"event_id"`
	EventName    string  `json:"event_name"`
	BookingCount int     `json:"booking_count"`
	Utilization  float64 `json:"utilization"`
	Revenue      float64 `json:"revenue"`
	Venue        string  `json:"venue"`
	DateTime     string  `json:"date_time"`
}

type DailyBooking struct {
	Date     string  `json:"date"`
	Bookings int     `json:"bookings"`
	Revenue  float64 `json:"revenue"`
}

type MonthlyRevenue struct {
	Month   string  `json:"month"`
	Revenue float64 `json:"revenue"`
	Events  int     `json:"events"`
}

type SectionStats struct {
	SectionID   string  `json:"section_id"`
	SectionName string  `json:"section_name"`
	Bookings    int     `json:"bookings"`
	Revenue     float64 `json:"revenue"`
	Utilization float64 `json:"utilization"`
}

type HourlyStats struct {
	Hour     int     `json:"hour"`
	Bookings int     `json:"bookings"`
	Revenue  float64 `json:"revenue"`
}

// Tag Analytics Models (migrated from tags package)

type TagOverview struct {
	TotalTags       int     `json:"total_tags"`
	ActiveTags      int     `json:"active_tags"`
	TagsWithEvents  int     `json:"tags_with_events"`
	AvgTagsPerEvent float64 `json:"avg_tags_per_event"`
	MostPopularTag  string  `json:"most_popular_tag"`
	LeastUsedTag    string  `json:"least_used_tag"`
}

type TagAnalyticsResponse struct {
	Overview    TagOverview     `json:"overview"`
	TopTags     []TagAnalytics  `json:"top_tags"`
	TagTrends   []TagTrend      `json:"tag_trends"`
	Comparisons []TagComparison `json:"comparisons"`
}

type TagAnalytics struct {
	TagID           string  `json:"tag_id"`
	TagName         string  `json:"tag_name"`
	EventCount      int     `json:"event_count"`
	TotalBookings   int     `json:"total_bookings"`
	TotalRevenue    float64 `json:"total_revenue"`
	AvgUtilization  float64 `json:"avg_utilization"`
	PopularityScore float64 `json:"popularity_score"`
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

type TagPerformance struct {
	TagID       string  `json:"tag_id"`
	TagName     string  `json:"tag_name"`
	EventCount  int     `json:"event_count"`
	Revenue     float64 `json:"revenue"`
	Utilization float64 `json:"utilization"`
}

// Booking Analytics Models (new)

type BookingOverview struct {
	TotalBookings      int                  `json:"total_bookings"`
	ConfirmedBookings  int                  `json:"confirmed_bookings"`
	CancelledBookings  int                  `json:"cancelled_bookings"`
	TotalRevenue       float64              `json:"total_revenue"`
	AverageBookingSize float64              `json:"average_booking_size"`
	AverageTicketPrice float64              `json:"average_ticket_price"`
	CancellationRate   float64              `json:"cancellation_rate"`
	BookingsByStatus   map[string]int       `json:"bookings_by_status"`
	DailyBookings      []DailyBookingStats  `json:"daily_bookings"`
	PaymentMethods     []PaymentMethodStats `json:"payment_methods"`
}

type BookingAnalytics struct {
	Overview         BookingOverview      `json:"overview"`
	TrendAnalysis    BookingTrendAnalysis `json:"trend_analysis"`
	PerformanceStats BookingPerformance   `json:"performance_stats"`
	Insights         []BookingInsight     `json:"insights"`
}

type DailyBookingStats struct {
	Date              string  `json:"date"`
	TotalBookings     int     `json:"total_bookings"`
	ConfirmedBookings int     `json:"confirmed_bookings"`
	CancelledBookings int     `json:"cancelled_bookings"`
	Revenue           float64 `json:"revenue"`
	AverageValue      float64 `json:"average_value"`
}

type BookingTrendAnalysis struct {
	PeriodComparison PeriodComparison `json:"period_comparison"`
	Seasonality      SeasonalityData  `json:"seasonality"`
	Growth           GrowthMetrics    `json:"growth"`
}

type PeriodComparison struct {
	CurrentPeriod  PeriodStats `json:"current_period"`
	PreviousPeriod PeriodStats `json:"previous_period"`
	PercentChange  float64     `json:"percent_change"`
}

type PeriodStats struct {
	Bookings int     `json:"bookings"`
	Revenue  float64 `json:"revenue"`
	Users    int     `json:"users"`
}

type SeasonalityData struct {
	ByDayOfWeek []WeekdayStats `json:"by_day_of_week"`
	ByHour      []HourlyStats  `json:"by_hour"`
	ByMonth     []MonthStats   `json:"by_month"`
}

type WeekdayStats struct {
	Weekday  string  `json:"weekday"`
	Bookings int     `json:"bookings"`
	Revenue  float64 `json:"revenue"`
}

type MonthStats struct {
	Month    string  `json:"month"`
	Bookings int     `json:"bookings"`
	Revenue  float64 `json:"revenue"`
}

type GrowthMetrics struct {
	BookingGrowth float64 `json:"booking_growth"`
	RevenueGrowth float64 `json:"revenue_growth"`
	UserGrowth    float64 `json:"user_growth"`
}

type BookingPerformance struct {
	ConversonRate   float64              `json:"conversion_rate"`
	AbandonmentRate float64              `json:"abandonment_rate"`
	AvgTimeToBook   float64              `json:"avg_time_to_book"`
	PopularEvents   []EventBookingStats  `json:"popular_events"`
	TopVenues       []VenueBookingStats  `json:"top_venues"`
	BookingSources  []BookingSourceStats `json:"booking_sources"`
}

type EventBookingStats struct {
	EventID       string    `json:"event_id"`
	EventName     string    `json:"event_name"`
	TotalBookings int       `json:"total_bookings"`
	Revenue       float64   `json:"revenue"`
	Utilization   float64   `json:"utilization"`
	DateTime      time.Time `json:"date_time"`
}

type VenueBookingStats struct {
	VenueID       string  `json:"venue_id"`
	VenueName     string  `json:"venue_name"`
	TotalBookings int     `json:"total_bookings"`
	Revenue       float64 `json:"revenue"`
	EventCount    int     `json:"event_count"`
	AvgCapacity   float64 `json:"avg_capacity"`
}

type BookingSourceStats struct {
	Source    string  `json:"source"`
	Bookings  int     `json:"bookings"`
	Revenue   float64 `json:"revenue"`
	UserCount int     `json:"user_count"`
}

type PaymentMethodStats struct {
	Method   string  `json:"method"`
	Bookings int     `json:"bookings"`
	Revenue  float64 `json:"revenue"`
	Success  float64 `json:"success_rate"`
}

type BookingInsight struct {
	Type        string `json:"type"` // "trend", "alert", "recommendation"
	Title       string `json:"title"`
	Description string `json:"description"`
	Impact      string `json:"impact"` // "high", "medium", "low"
	Metric      string `json:"metric"`
	Value       string `json:"value"`
}

type CancellationAnalytics struct {
	Overview            CancellationOverview  `json:"overview"`
	CancellationReasons []CancellationReason  `json:"cancellation_reasons"`
	TimingAnalysis      CancellationTiming    `json:"timing_analysis"`
	FinancialImpact     CancellationFinancial `json:"financial_impact"`
	Trends              []CancellationTrend   `json:"trends"`
}

type CancellationOverview struct {
	TotalCancellations  int     `json:"total_cancellations"`
	CancellationRate    float64 `json:"cancellation_rate"`
	RefundAmount        float64 `json:"refund_amount"`
	RefundRate          float64 `json:"refund_rate"`
	AvgTimeToCancel     float64 `json:"avg_time_to_cancel"` // hours
	MostCancelledEvent  string  `json:"most_cancelled_event"`
	HighestCancelledTag string  `json:"highest_cancelled_tag"`
}

type CancellationReason struct {
	Reason      string  `json:"reason"`
	Count       int     `json:"count"`
	Percentage  float64 `json:"percentage"`
	RefundTotal float64 `json:"refund_total"`
}

type CancellationTiming struct {
	ByTimeToEvent []CancellationTimeWindow `json:"by_time_to_event"`
	ByDayOfWeek   []WeekdayStats           `json:"by_day_of_week"`
	ByHour        []HourlyStats            `json:"by_hour"`
}

type CancellationTimeWindow struct {
	WindowDesc   string  `json:"window_description"` // "24h before", "1 week before", etc.
	Count        int     `json:"count"`
	Percentage   float64 `json:"percentage"`
	RefundAmount float64 `json:"refund_amount"`
}

type CancellationFinancial struct {
	TotalRefunds        float64             `json:"total_refunds"`
	RefundsByPolicy     []RefundPolicyStats `json:"refunds_by_policy"`
	LostRevenue         float64             `json:"lost_revenue"`
	RefundProcessingFee float64             `json:"refund_processing_fee"`
}

type RefundPolicyStats struct {
	Policy       string  `json:"policy"` // "full", "partial", "no_refund"
	Count        int     `json:"count"`
	RefundAmount float64 `json:"refund_amount"`
	Percentage   float64 `json:"percentage"`
}

type CancellationTrend struct {
	Date             string  `json:"date"`
	Cancellations    int     `json:"cancellations"`
	CancellationRate float64 `json:"cancellation_rate"`
	RefundAmount     float64 `json:"refund_amount"`
}

// User Analytics Models (new)

type UserOverview struct {
	TotalUsers         int               `json:"total_users"`
	ActiveUsers        int               `json:"active_users"`
	NewUsers           int               `json:"new_users"`
	RetentionRate      float64           `json:"retention_rate"`
	AvgBookingsPerUser float64           `json:"avg_bookings_per_user"`
	UserGrowth         []UserGrowthStats `json:"user_growth"`
	UserSegments       []UserSegment     `json:"user_segments"`
}

type UserAnalytics struct {
	Overview         UserOverview     `json:"overview"`
	RetentionMetrics UserRetention    `json:"retention_metrics"`
	Demographics     UserDemographics `json:"demographics"`
	Behavior         UserBehavior     `json:"behavior"`
	Insights         []UserInsight    `json:"insights"`
}

type UserGrowthStats struct {
	Date        string `json:"date"`
	NewUsers    int    `json:"new_users"`
	TotalUsers  int    `json:"total_users"`
	ActiveUsers int    `json:"active_users"`
}

type UserSegment struct {
	Segment     string  `json:"segment"` // "new", "regular", "vip", "inactive"
	UserCount   int     `json:"user_count"`
	Revenue     float64 `json:"revenue"`
	AvgBookings float64 `json:"avg_bookings"`
	Percentage  float64 `json:"percentage"`
}

type UserRetention struct {
	RetentionByPeriod []RetentionPeriod `json:"retention_by_period"`
	ChurnRate         float64           `json:"churn_rate"`
	LifetimeValue     float64           `json:"lifetime_value"`
	RetentionCohorts  []CohortData      `json:"retention_cohorts"`
}

type RetentionPeriod struct {
	Period     string  `json:"period"` // "1 month", "3 months", "6 months"
	Percentage float64 `json:"percentage"`
	UserCount  int     `json:"user_count"`
}

type CohortData struct {
	CohortMonth string    `json:"cohort_month"`
	CohortSize  int       `json:"cohort_size"`
	Retention   []float64 `json:"retention"` // Retention % for each subsequent month
}

type UserDemographics struct {
	AgeGroups       []DemographicStat `json:"age_groups"`
	Locations       []LocationStat    `json:"locations"`
	JoinedPeriods   []PeriodStat      `json:"joined_periods"`
	BookingPatterns []PatternStat     `json:"booking_patterns"`
}

type DemographicStat struct {
	Category   string  `json:"category"`
	UserCount  int     `json:"user_count"`
	Percentage float64 `json:"percentage"`
	AvgRevenue float64 `json:"avg_revenue"`
}

type LocationStat struct {
	Location    string  `json:"location"`
	UserCount   int     `json:"user_count"`
	Percentage  float64 `json:"percentage"`
	AvgBookings float64 `json:"avg_bookings"`
}

type PeriodStat struct {
	Period      string  `json:"period"`
	UserCount   int     `json:"user_count"`
	RetainedPct float64 `json:"retained_percentage"`
}

type PatternStat struct {
	Pattern    string  `json:"pattern"` // "frequent", "occasional", "one-time"
	UserCount  int     `json:"user_count"`
	Revenue    float64 `json:"revenue"`
	Percentage float64 `json:"percentage"`
}

type UserBehavior struct {
	AvgSessionTime       float64                   `json:"avg_session_time"`
	EventPreferences     []PreferenceStats         `json:"event_preferences"`
	BookingFrequency     BookingFrequencyStats     `json:"booking_frequency"`
	PricePreferences     PricePreferenceStats      `json:"price_preferences"`
	CancellationBehavior CancellationBehaviorStats `json:"cancellation_behavior"`
}

type PreferenceStats struct {
	Category   string  `json:"category"` // tag, venue type, time preference
	Value      string  `json:"value"`
	UserCount  int     `json:"user_count"`
	Percentage float64 `json:"percentage"`
	Revenue    float64 `json:"revenue"`
}

type BookingFrequencyStats struct {
	AvgBookingsPerMonth   float64           `json:"avg_bookings_per_month"`
	FrequencyDistribution []FrequencyBucket `json:"frequency_distribution"`
}

type FrequencyBucket struct {
	Range      string  `json:"range"` // "1", "2-3", "4-5", "6+"
	UserCount  int     `json:"user_count"`
	Percentage float64 `json:"percentage"`
	AvgRevenue float64 `json:"avg_revenue"`
}

type PricePreferenceStats struct {
	AvgTicketPrice   float64      `json:"avg_ticket_price"`
	PriceRanges      []PriceRange `json:"price_ranges"`
	PriceSensitivity float64      `json:"price_sensitivity"`
}

type PriceRange struct {
	Range       string  `json:"range"` // "$0-50", "$51-100", etc.
	UserCount   int     `json:"user_count"`
	Percentage  float64 `json:"percentage"`
	AvgBookings float64 `json:"avg_bookings"`
}

type CancellationBehaviorStats struct {
	CancellationRate float64  `json:"cancellation_rate"`
	AvgTimeToCancel  float64  `json:"avg_time_to_cancel"`
	RepeatCancellers int      `json:"repeat_cancellers"`
	TopCancelReasons []string `json:"top_cancel_reasons"`
}

type UserInsight struct {
	Type        string `json:"type"` // "behavior", "opportunity", "risk"
	Title       string `json:"title"`
	Description string `json:"description"`
	UserCount   int    `json:"user_count"`
	Impact      string `json:"impact"`
	Action      string `json:"recommended_action"`
}

// Venue Performance Models (new)

type VenuePerformance struct {
	VenueID         string  `json:"venue_id"`
	VenueName       string  `json:"venue_name"`
	EventCount      int     `json:"event_count"`
	TotalBookings   int     `json:"total_bookings"`
	Revenue         float64 `json:"revenue"`
	AvgUtilization  float64 `json:"avg_utilization"`
	PopularityScore float64 `json:"popularity_score"`
}

// User-facing Analytics Models

type UserBookingHistory struct {
	Overview         UserBookingOverview   `json:"overview"`
	BookingHistory   []UserBookingRecord   `json:"booking_history"`
	SpendingAnalysis UserSpendingAnalysis  `json:"spending_analysis"`
	Preferences      UserPreferences       `json:"preferences"`
	Insights         []UserPersonalInsight `json:"insights"`
}

type UserBookingOverview struct {
	TotalBookings     int     `json:"total_bookings"`
	TotalSpent        float64 `json:"total_spent"`
	AvgBookingValue   float64 `json:"avg_booking_value"`
	FavoriteVenue     string  `json:"favorite_venue"`
	FavoriteEventType string  `json:"favorite_event_type"`
	MemberSince       string  `json:"member_since"`
}

type UserBookingRecord struct {
	BookingID   string    `json:"booking_id"`
	EventName   string    `json:"event_name"`
	VenueName   string    `json:"venue_name"`
	BookingDate time.Time `json:"booking_date"`
	EventDate   time.Time `json:"event_date"`
	TotalAmount float64   `json:"total_amount"`
	SeatCount   int       `json:"seat_count"`
	Status      string    `json:"status"`
}

type UserSpendingAnalysis struct {
	SpendingByMonth    []MonthlySpending  `json:"spending_by_month"`
	SpendingByCategory []CategorySpending `json:"spending_by_category"`
	SpendingTrend      string             `json:"spending_trend"`
	YearlyTotal        float64            `json:"yearly_total"`
}

type MonthlySpending struct {
	Month    string  `json:"month"`
	Amount   float64 `json:"amount"`
	Bookings int     `json:"bookings"`
}

type CategorySpending struct {
	Category   string  `json:"category"` // event tag or venue type
	Amount     float64 `json:"amount"`
	Bookings   int     `json:"bookings"`
	Percentage float64 `json:"percentage"`
}

type UserPreferences struct {
	PreferredTags    []string `json:"preferred_tags"`
	PreferredVenues  []string `json:"preferred_venues"`
	PreferredTimes   []string `json:"preferred_times"`
	PriceRange       string   `json:"price_range"`
	BookingFrequency string   `json:"booking_frequency"`
}

type UserPersonalInsight struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Value       string `json:"value"`
}

type PersonalAnalytics struct {
	BookingPatterns  PersonalBookingPatterns  `json:"booking_patterns"`
	SpendingInsights PersonalSpendingInsights `json:"spending_insights"`
	Recommendations  []PersonalRecommendation `json:"recommendations"`
	Achievements     []Achievement            `json:"achievements"`
}

type PersonalBookingPatterns struct {
	BookingFrequency    string   `json:"booking_frequency"`
	PreferredDay        string   `json:"preferred_day"`
	PreferredTime       string   `json:"preferred_time"`
	AdvanceBookingTime  int      `json:"advance_booking_time"` // days in advance
	SeasonalPreferences []string `json:"seasonal_preferences"`
}

type PersonalSpendingInsights struct {
	MonthlyAverage     float64 `json:"monthly_average"`
	YearOverYearGrowth float64 `json:"year_over_year_growth"`
	PeakSpendingMonth  string  `json:"peak_spending_month"`
	BudgetSuggestion   float64 `json:"budget_suggestion"`
	SavingsOpportunity float64 `json:"savings_opportunity"`
}

type PersonalRecommendation struct {
	Type        string  `json:"type"` // "event", "venue", "time"
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Reason      string  `json:"reason"`
	Confidence  float64 `json:"confidence"`
}

type Achievement struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	UnlockedAt  time.Time `json:"unlocked_at"`
	Rarity      string    `json:"rarity"` // "common", "rare", "epic", "legendary"
}
