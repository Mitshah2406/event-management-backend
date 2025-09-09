package events

import (
	"evently/internal/tags"
	"time"

	"github.com/google/uuid"
)

type TagResponse = tags.TagResponse

// TagInfo represents basic tag information for event responses
type TagInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Slug  string `json:"slug"`
	Color string `json:"color"`
}

type Event struct {
	ID            uuid.UUID   `json:"id" gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	Name          string      `json:"name" gorm:"not null;size:255"`
	Description   string      `json:"description" gorm:"type:text"`
	Venue         string      `json:"venue" gorm:"not null;size:255"`
	DateTime      time.Time   `json:"date_time" gorm:"not null"`
	TotalCapacity int         `json:"total_capacity" gorm:"not null;check:total_capacity > 0"`
	BookedCount   int         `json:"booked_count" gorm:"default:0;check:booked_count >= 0"`
	Price         float64     `json:"price" gorm:"not null;check:price >= 0"`
	Status        EventStatus `json:"status" gorm:"type:varchar(20);default:'draft'"`
	ImageURL      string      `json:"image_url" gorm:"size:500"`

	// Many-to-many relationship with tags
	Tags []tags.Tag `json:"-" gorm:"many2many:event_tags;constraint:OnDelete:CASCADE;"`

	CreatedBy uuid.UUID  `json:"created_by" gorm:"type:uuid;not null"`
	UpdatedBy *uuid.UUID `json:"updated_by" gorm:"type:uuid"`
	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

type EventResponse struct {
	ID               string      `json:"id"`
	Name             string      `json:"name"`
	Description      string      `json:"description"`
	Venue            string      `json:"venue"`
	DateTime         time.Time   `json:"date_time"`
	TotalCapacity    int         `json:"total_capacity"`
	BookedCount      int         `json:"booked_count"`
	AvailableTickets int         `json:"available_tickets"`
	Price            float64     `json:"price"`
	Status           EventStatus `json:"status"`
	ImageURL         string      `json:"image_url"`
	Tags             []TagInfo   `json:"tags"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}

type CreateEventRequest struct {
	Name          string    `json:"name" binding:"required,min=3,max=255"`
	Description   string    `json:"description" binding:"max=2000"`
	Venue         string    `json:"venue" binding:"required,min=3,max=255"`
	DateTime      time.Time `json:"date_time" binding:"required"`
	TotalCapacity int       `json:"total_capacity" binding:"required,min=1,max=100000"`
	Price         float64   `json:"price" binding:"required,min=0"`
	ImageURL      string    `json:"image_url" binding:"omitempty,url"`
	Tags          []string  `json:"tags"`
}

type UpdateEventRequest struct {
	Name          *string    `json:"name" binding:"omitempty,min=3,max=255"`
	Description   *string    `json:"description" binding:"omitempty,max=2000"`
	Venue         *string    `json:"venue" binding:"omitempty,min=3,max=255"`
	DateTime      *time.Time `json:"date_time"`
	TotalCapacity *int       `json:"total_capacity" binding:"omitempty,min=1,max=100000"`
	Price         *float64   `json:"price" binding:"omitempty,min=0"`
	Status        *string    `json:"status" binding:"omitempty,oneof=draft published cancelled completed"`
	ImageURL      *string    `json:"image_url" binding:"omitempty,url"`
	Tags          []string   `json:"tags"`
}

type EventListQuery struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	Limit    int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Search   string `form:"search"`
	Venue    string `form:"venue"`
	DateFrom string `form:"date_from"`
	DateTo   string `form:"date_to"`
	Status   string `form:"status" binding:"omitempty,oneof=draft published cancelled completed"`
	Tags     string `form:"tags"`
}

type EventAnalytics struct {
	EventID             string         `json:"event_id"`
	EventName           string         `json:"event_name"`
	TotalBookings       int            `json:"total_bookings"`
	TotalRevenue        float64        `json:"total_revenue"`
	CapacityUtilization float64        `json:"capacity_utilization"`
	CancellationRate    float64        `json:"cancellation_rate"`
	BookingsByDay       []DailyBooking `json:"bookings_by_day"`
}

type DailyBooking struct {
	Date     string  `json:"date"`
	Bookings int     `json:"bookings"`
	Revenue  float64 `json:"revenue"`
}

type PaginatedEvents struct {
	Events     []EventResponse `json:"events"`
	TotalCount int64           `json:"total_count"`
	Page       int             `json:"page"`
	Limit      int             `json:"limit"`
	TotalPages int             `json:"total_pages"`
}

type GlobalAnalytics struct {
	TotalEvents        int               `json:"total_events"`
	TotalBookings      int               `json:"total_bookings"`
	TotalRevenue       float64           `json:"total_revenue"`
	AverageUtilization float64           `json:"average_utilization"`
	MostPopularEvents  []EventPopularity `json:"most_popular_events"`
	BookingTrends      []DailyBooking    `json:"booking_trends"`
	EventsByStatus     map[string]int    `json:"events_by_status"`
	RevenueByMonth     []MonthlyRevenue  `json:"revenue_by_month"`
}

type EventPopularity struct {
	EventID      string  `json:"event_id"`
	EventName    string  `json:"event_name"`
	BookingCount int     `json:"booking_count"`
	Utilization  float64 `json:"utilization"`
	Revenue      float64 `json:"revenue"`
}

type MonthlyRevenue struct {
	Month   string  `json:"month"`
	Revenue float64 `json:"revenue"`
	Events  int     `json:"events"`
}

// Helper method to convert Event to EventResponse
// Note: Tags field will be populated separately by the service layer
func (e *Event) ToResponse() EventResponse {
	availableTickets := e.TotalCapacity - e.BookedCount
	if availableTickets < 0 {
		availableTickets = 0
	}

	return EventResponse{
		ID:               e.ID.String(),
		Name:             e.Name,
		Description:      e.Description,
		Venue:            e.Venue,
		DateTime:         e.DateTime,
		TotalCapacity:    e.TotalCapacity,
		BookedCount:      e.BookedCount,
		AvailableTickets: availableTickets,
		Price:            e.Price,
		Status:           e.Status,
		ImageURL:         e.ImageURL,
		Tags:             []TagInfo{}, // Will be populated by service layer
		CreatedAt:        e.CreatedAt,
		UpdatedAt:        e.UpdatedAt,
	}
}

// TableName specifies the table name for GORM
func (Event) TableName() string {
	return "events"
}
