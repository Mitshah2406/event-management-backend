package events

import (
	"evently/internal/tags"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	Create(event *Event) error
	GetByID(id uuid.UUID) (*Event, error)
	Update(id uuid.UUID, updates map[string]interface{}) (*Event, error)
	Delete(id uuid.UUID) error
	GetAll(query EventListQuery) ([]Event, int64, error)
	GetByStatus(status EventStatus) ([]Event, error)
	GetEventCapacityAndBookings(eventID uuid.UUID) (int, int, error)
	GetEventAnalytics(eventID uuid.UUID) (*EventAnalytics, error)
	GetGlobalAnalytics() (*GlobalAnalytics, error)
	GetUpcomingEvents(limit int) ([]Event, error)
	CheckSeatAvailability(eventID uuid.UUID, requestedSeats int) (bool, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(event *Event) error {
	return r.db.Create(event).Error
}

func (r *repository) GetByID(id uuid.UUID) (*Event, error) {
	var event Event
	err := r.db.Where("id = ?", id).First(&event).Error
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *repository) Update(id uuid.UUID, updates map[string]interface{}) (*Event, error) {
	var event Event

	// First, get the current event
	if err := r.db.Where("id = ?", id).First(&event).Error; err != nil {
		return nil, err
	}

	// Update the event
	if err := r.db.Model(&event).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Return updated event
	if err := r.db.Where("id = ?", id).First(&event).Error; err != nil {
		return nil, err
	}

	return &event, nil
}

func (r *repository) Delete(id uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// First, delete all event-tag associations
		if err := tx.Where("event_id = ?", id).Delete(&tags.EventTag{}).Error; err != nil {
			return fmt.Errorf("failed to delete event-tag associations: %w", err)
		}

		// Then delete the event itself
		if err := tx.Where("id = ?", id).Delete(&Event{}).Error; err != nil {
			return fmt.Errorf("failed to delete event: %w", err)
		}

		return nil
	})
}

func (r *repository) GetAll(query EventListQuery) ([]Event, int64, error) {
	var events []Event
	var totalCount int64

	// Build the query
	db := r.db.Model(&Event{})

	// Apply filters
	if query.Search != "" {
		searchTerm := "%" + strings.ToLower(query.Search) + "%"
		db = db.Where("LOWER(name) LIKE ? OR LOWER(description) LIKE ? OR LOWER(venue) LIKE ?",
			searchTerm, searchTerm, searchTerm)
	}

	if query.Venue != "" {
		db = db.Where("LOWER(venue) LIKE ?", "%"+strings.ToLower(query.Venue)+"%")
	}

	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}

	if query.Tags != "" {
		tags := strings.Split(query.Tags, ",")
		var cleanTags []string
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				cleanTags = append(cleanTags, tag)
			}
		}
		if len(cleanTags) > 0 {
			// Join with event_tags and tags table to filter by tag names
			subquery := r.db.Table("event_tags").
				Joins("JOIN tags ON event_tags.tag_id = tags.id").
				Where("tags.name IN ? AND tags.is_active = ?", cleanTags, true).
				Select("event_tags.event_id")

			db = db.Where("id IN (?)", subquery)
		}
	}

	// Date filters
	if query.DateFrom != "" {
		if dateFrom, err := time.Parse("2006-01-02", query.DateFrom); err == nil {
			db = db.Where("date_time >= ?", dateFrom)
		}
	}

	if query.DateTo != "" {
		if dateTo, err := time.Parse("2006-01-02", query.DateTo); err == nil {
			// Add 24 hours to include the entire day
			dateTo = dateTo.Add(24 * time.Hour)
			db = db.Where("date_time < ?", dateTo)
		}
	}

	// Count total records
	if err := db.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if query.Page == 0 {
		query.Page = 1
	}
	if query.Limit == 0 {
		query.Limit = 10
	}

	offset := (query.Page - 1) * query.Limit

	// Get paginated results
	err := db.Order("date_time ASC").
		Offset(offset).
		Limit(query.Limit).
		Find(&events).Error

	return events, totalCount, err
}

func (r *repository) GetByStatus(status EventStatus) ([]Event, error) {
	var events []Event
	err := r.db.Where("status = ?", status).Find(&events).Error
	return events, err
}

func (r *repository) GetEventCapacityAndBookings(eventID uuid.UUID) (int, int, error) {
	// First get the event's venue template ID
	var event Event
	if err := r.db.Where("id = ?", eventID).First(&event).Error; err != nil {
		return 0, 0, fmt.Errorf("failed to get event: %w", err)
	}

	// Get total capacity from venue sections that belong to the event's template
	var totalCapacity int64
	err := r.db.Table("venue_sections").
		Select("COALESCE(SUM(total_seats), 0) as total_capacity").
		Where("template_id = ?", event.VenueTemplateID).
		Scan(&totalCapacity).Error
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get total capacity: %w", err)
	}

	// Get booked count from seat bookings for seats in sections of this template
	var bookedCount int64
	err = r.db.Table("seat_bookings").
		Joins("JOIN bookings ON seat_bookings.booking_id = bookings.id").
		Joins("JOIN seats ON seat_bookings.seat_id = seats.id").
		Joins("JOIN venue_sections ON seats.section_id = venue_sections.id").
		Where("venue_sections.template_id = ? AND seat_bookings.event_id = ? AND bookings.status = 'CONFIRMED'", 
			event.VenueTemplateID, eventID).
		Count(&bookedCount).Error
	if err != nil {
		return int(totalCapacity), 0, fmt.Errorf("failed to get booked count: %w", err)
	}

	return int(totalCapacity), int(bookedCount), nil
}

func (r *repository) CheckSeatAvailability(eventID uuid.UUID, requestedSeats int) (bool, error) {
	// First get the event's venue template ID
	var event Event
	if err := r.db.Where("id = ?", eventID).First(&event).Error; err != nil {
		return false, fmt.Errorf("failed to get event: %w", err)
	}

	// Get available seats count for the event's template (excluding booked/held seats for this event)
	var availableSeats int64
	err := r.db.Table("seats").
		Joins("JOIN venue_sections ON seats.section_id = venue_sections.id").
		Joins("LEFT JOIN seat_bookings ON seats.id = seat_bookings.seat_id AND seat_bookings.event_id = ?", eventID).
		Where("venue_sections.template_id = ? AND seats.status = 'AVAILABLE' AND seat_bookings.id IS NULL", event.VenueTemplateID).
		Count(&availableSeats).Error
	if err != nil {
		return false, fmt.Errorf("failed to check seat availability: %w", err)
	}

	return int(availableSeats) >= requestedSeats, nil
}

func (r *repository) GetEventAnalytics(eventID uuid.UUID) (*EventAnalytics, error) {
	var analytics EventAnalytics

	// Get basic event info
	var event Event
	if err := r.db.Where("id = ?", eventID).First(&event).Error; err != nil {
		return nil, err
	}

	analytics.EventID = event.ID.String()
	analytics.EventName = event.Name

	// Get capacity and booking data from seat-based system
	totalCapacity, bookedCount, err := r.GetEventCapacityAndBookings(eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get capacity data: %w", err)
	}

	analytics.TotalBookings = bookedCount

	// Calculate revenue from actual seat bookings
	var totalRevenue float64
	err = r.db.Table("seat_bookings").
		Joins("JOIN bookings ON seat_bookings.booking_id = bookings.id").
		Select("COALESCE(SUM(seat_bookings.seat_price), 0) as total_revenue").
		Where("seat_bookings.event_id = ? AND bookings.status = 'CONFIRMED'", eventID).
		Scan(&totalRevenue).Error
	if err != nil {
		return nil, fmt.Errorf("failed to calculate revenue: %w", err)
	}

	analytics.TotalRevenue = totalRevenue

	if totalCapacity > 0 {
		analytics.CapacityUtilization = (float64(bookedCount) / float64(totalCapacity)) * 100
	}

	// Note: For cancellation rate and daily bookings, we would need the bookings table
	// This is a placeholder implementation
	analytics.CancellationRate = 0.0
	analytics.BookingsByDay = []DailyBooking{}

	return &analytics, nil
}

func (r *repository) GetUpcomingEvents(limit int) ([]Event, error) {
	var events []Event
	now := time.Now()

	err := r.db.Where("date_time > ? AND status = ?", now, EventStatusPublished).
		Order("date_time ASC").
		Limit(limit).
		Find(&events).Error

	return events, err
}

func (r *repository) GetGlobalAnalytics() (*GlobalAnalytics, error) {
	var analytics GlobalAnalytics

	// Get total events count
	var totalEvents int64
	if err := r.db.Model(&Event{}).Count(&totalEvents).Error; err != nil {
		return nil, fmt.Errorf("failed to count total events: %w", err)
	}
	analytics.TotalEvents = int(totalEvents)

	// Get total bookings and revenue from seat-based bookings
	type aggregateResult struct {
		TotalBookings int64   `json:"total_bookings"`
		TotalRevenue  float64 `json:"total_revenue"`
	}

	var result aggregateResult
	if err := r.db.Table("seat_bookings").
		Joins("JOIN bookings ON seat_bookings.booking_id = bookings.id").
		Select("COUNT(seat_bookings.id) as total_bookings, COALESCE(SUM(seat_bookings.seat_price), 0) as total_revenue").
		Where("bookings.status = 'CONFIRMED'").
		Scan(&result).Error; err != nil {
		return nil, fmt.Errorf("failed to get aggregate data: %w", err)
	}

	analytics.TotalBookings = int(result.TotalBookings)
	analytics.TotalRevenue = result.TotalRevenue

	// Calculate average utilization across all events
	type utilizationResult struct {
		AverageUtilization float64 `json:"average_utilization"`
	}

	var utilResult utilizationResult
	subquery := r.db.Table("events e").
		Select(`
			e.id,
			COALESCE(capacity_data.total_capacity, 0) as total_capacity,
			COALESCE(booking_data.booked_count, 0) as booked_count,
			CASE 
				WHEN COALESCE(capacity_data.total_capacity, 0) > 0 
				THEN (COALESCE(booking_data.booked_count, 0) * 100.0 / capacity_data.total_capacity)
				ELSE 0 
			END as utilization
		`).
		Joins(`
			LEFT JOIN (
				SELECT event_id, SUM(total_seats) as total_capacity 
				FROM venue_sections 
				GROUP BY event_id
			) capacity_data ON e.id = capacity_data.event_id
		`).
		Joins(`
			LEFT JOIN (
				SELECT vs.event_id, COUNT(sb.id) as booked_count
				FROM seat_bookings sb
				JOIN bookings b ON sb.booking_id = b.id
				JOIN venue_sections vs ON sb.section_id = vs.id
				WHERE b.status = 'CONFIRMED'
				GROUP BY vs.event_id
			) booking_data ON e.id = booking_data.event_id
		`)

	if err := r.db.Table("(?) as event_utilization", subquery).
		Select("AVG(utilization) as average_utilization").
		Where("total_capacity > 0").
		Scan(&utilResult).Error; err != nil {
		return nil, fmt.Errorf("failed to calculate average utilization: %w", err)
	}

	analytics.AverageUtilization = utilResult.AverageUtilization

	// Get most popular events (top 5) based on actual bookings
	type popularEventData struct {
		EventID      string  `json:"event_id"`
		EventName    string  `json:"event_name"`
		BookingCount int     `json:"booking_count"`
		TotalCapacity int    `json:"total_capacity"`
		Revenue      float64 `json:"revenue"`
	}

	var popularEventsData []popularEventData
	if err := r.db.Table("events e").
		Select(`
			e.id as event_id,
			e.name as event_name,
			COALESCE(booking_data.booking_count, 0) as booking_count,
			COALESCE(capacity_data.total_capacity, 0) as total_capacity,
			COALESCE(booking_data.revenue, 0) as revenue
		`).
		Joins(`
			LEFT JOIN (
				SELECT vs.event_id, COUNT(sb.id) as booking_count, SUM(sb.seat_price) as revenue
				FROM seat_bookings sb
				JOIN bookings b ON sb.booking_id = b.id
				JOIN venue_sections vs ON sb.section_id = vs.id
				WHERE b.status = 'CONFIRMED'
				GROUP BY vs.event_id
			) booking_data ON e.id = booking_data.event_id
		`).
		Joins(`
			LEFT JOIN (
				SELECT event_id, SUM(total_seats) as total_capacity 
				FROM venue_sections 
				GROUP BY event_id
			) capacity_data ON e.id = capacity_data.event_id
		`).
		Where("booking_data.booking_count > 0").
		Order("booking_data.booking_count DESC").
		Limit(5).
		Scan(&popularEventsData).Error; err != nil {
		return nil, fmt.Errorf("failed to get popular events: %w", err)
	}

	analytics.MostPopularEvents = make([]EventPopularity, len(popularEventsData))
	for i, event := range popularEventsData {
		utilization := float64(0)
		if event.TotalCapacity > 0 {
			utilization = (float64(event.BookingCount) / float64(event.TotalCapacity)) * 100
		}

		analytics.MostPopularEvents[i] = EventPopularity{
			EventID:      event.EventID,
			EventName:    event.EventName,
			BookingCount: event.BookingCount,
			Utilization:  utilization,
			Revenue:      event.Revenue,
		}
	}

	// Get events by status
	analytics.EventsByStatus = make(map[string]int)
	type statusCount struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}

	var statusCounts []statusCount
	if err := r.db.Model(&Event{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&statusCounts).Error; err != nil {
		return nil, fmt.Errorf("failed to get events by status: %w", err)
	}

	for _, sc := range statusCounts {
		analytics.EventsByStatus[sc.Status] = sc.Count
	}

	// Get revenue by month from actual bookings (last 12 months)
	type monthlyRevenueResult struct {
		Month   string  `json:"month"`
		Revenue float64 `json:"revenue"`
		Events  int64   `json:"events"`
	}

	var monthlyRevenues []monthlyRevenueResult
	if err := r.db.Table("bookings b").
		Select(`
			TO_CHAR(b.created_at, 'YYYY-MM') as month,
			COALESCE(SUM(sb.seat_price), 0) as revenue,
			COUNT(DISTINCT vs.event_id) as events
		`).
		Joins("JOIN seat_bookings sb ON b.id = sb.booking_id").
		Joins("JOIN venue_sections vs ON sb.section_id = vs.id").
		Joins("JOIN events e ON vs.event_id = e.id").
		Where("b.status = 'CONFIRMED' AND b.created_at >= ?", time.Now().AddDate(0, -12, 0)).
		Group("TO_CHAR(b.created_at, 'YYYY-MM')").
		Order("month DESC").
		Scan(&monthlyRevenues).Error; err != nil {
		return nil, fmt.Errorf("failed to get monthly revenue: %w", err)
	}

	analytics.RevenueByMonth = make([]MonthlyRevenue, len(monthlyRevenues))
	for i, mr := range monthlyRevenues {
		analytics.RevenueByMonth[i] = MonthlyRevenue{
			Month:   mr.Month,
			Revenue: mr.Revenue,
			Events:  int(mr.Events),
		}
	}

	// Placeholder for booking trends - would need bookings table for real implementation
	analytics.BookingTrends = []DailyBooking{}

	return &analytics, nil
}
