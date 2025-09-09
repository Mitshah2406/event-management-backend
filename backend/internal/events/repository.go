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
	UpdateBookedCount(eventID uuid.UUID, increment int) error
	GetEventAnalytics(eventID uuid.UUID) (*EventAnalytics, error)
	GetGlobalAnalytics() (*GlobalAnalytics, error)
	GetUpcomingEvents(limit int) ([]Event, error)
	CheckCapacityAvailability(eventID uuid.UUID, requestedTickets int) (bool, error)
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

func (r *repository) UpdateBookedCount(eventID uuid.UUID, increment int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var event Event

		// Lock the row for update
		if err := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("id = ?", eventID).
			First(&event).Error; err != nil {
			return err
		}

		// Check if the operation would result in negative booked count
		newBookedCount := event.BookedCount + increment
		if newBookedCount < 0 {
			return fmt.Errorf("cannot reduce booked count below 0")
		}

		// Check if the operation would exceed capacity
		if newBookedCount > event.TotalCapacity {
			return fmt.Errorf("cannot exceed event capacity")
		}

		// Update the booked count
		return tx.Model(&event).Update("booked_count", newBookedCount).Error
	})
}

func (r *repository) CheckCapacityAvailability(eventID uuid.UUID, requestedTickets int) (bool, error) {
	var event Event

	if err := r.db.Where("id = ?", eventID).First(&event).Error; err != nil {
		return false, err
	}

	availableTickets := event.TotalCapacity - event.BookedCount
	return availableTickets >= requestedTickets, nil
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
	analytics.TotalBookings = event.BookedCount
	analytics.TotalRevenue = float64(event.BookedCount) * event.Price

	if event.TotalCapacity > 0 {
		analytics.CapacityUtilization = (float64(event.BookedCount) / float64(event.TotalCapacity)) * 100
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

	// Get total bookings and revenue
	type aggregateResult struct {
		TotalBookings int     `json:"total_bookings"`
		TotalRevenue  float64 `json:"total_revenue"`
	}

	var result aggregateResult
	if err := r.db.Model(&Event{}).
		Select("SUM(booked_count) as total_bookings, SUM(booked_count * price) as total_revenue").
		Scan(&result).Error; err != nil {
		return nil, fmt.Errorf("failed to get aggregate data: %w", err)
	}

	analytics.TotalBookings = result.TotalBookings
	analytics.TotalRevenue = result.TotalRevenue

	// Calculate average utilization
	type utilizationResult struct {
		AverageUtilization float64 `json:"average_utilization"`
	}

	var utilResult utilizationResult
	if err := r.db.Model(&Event{}).
		Select("AVG(CASE WHEN total_capacity > 0 THEN (booked_count * 100.0 / total_capacity) ELSE 0 END) as average_utilization").
		Where("total_capacity > 0").
		Scan(&utilResult).Error; err != nil {
		return nil, fmt.Errorf("failed to calculate average utilization: %w", err)
	}

	analytics.AverageUtilization = utilResult.AverageUtilization

	// Get most popular events (top 5)
	var popularEvents []Event
	if err := r.db.Where("booked_count > 0").
		Order("booked_count DESC").
		Limit(5).
		Find(&popularEvents).Error; err != nil {
		return nil, fmt.Errorf("failed to get popular events: %w", err)
	}

	analytics.MostPopularEvents = make([]EventPopularity, len(popularEvents))
	for i, event := range popularEvents {
		utilization := float64(0)
		if event.TotalCapacity > 0 {
			utilization = (float64(event.BookedCount) / float64(event.TotalCapacity)) * 100
		}

		analytics.MostPopularEvents[i] = EventPopularity{
			EventID:      event.ID.String(),
			EventName:    event.Name,
			BookingCount: event.BookedCount,
			Utilization:  utilization,
			Revenue:      float64(event.BookedCount) * event.Price,
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

	// Get revenue by month (last 12 months)
	type monthlyRevenueResult struct {
		Month   string  `json:"month"`
		Revenue float64 `json:"revenue"`
		Events  int     `json:"events"`
	}

	var monthlyRevenues []monthlyRevenueResult
	if err := r.db.Model(&Event{}).
		Select("TO_CHAR(date_time, 'YYYY-MM') as month, SUM(booked_count * price) as revenue, COUNT(*) as events").
		Where("date_time >= ?", time.Now().AddDate(0, -12, 0)).
		Group("TO_CHAR(date_time, 'YYYY-MM')").
		Order("month DESC").
		Scan(&monthlyRevenues).Error; err != nil {
		return nil, fmt.Errorf("failed to get monthly revenue: %w", err)
	}

	analytics.RevenueByMonth = make([]MonthlyRevenue, len(monthlyRevenues))
	for i, mr := range monthlyRevenues {
		analytics.RevenueByMonth[i] = MonthlyRevenue(mr)
	}

	// Placeholder for booking trends - would need bookings table for real implementation
	analytics.BookingTrends = []DailyBooking{}

	return &analytics, nil
}
