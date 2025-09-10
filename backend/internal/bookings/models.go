package bookings

import (
	"evently/internal/events"
	"evently/internal/users"
	"time"

	"github.com/google/uuid"
)

type Booking struct {
	ID          uuid.UUID    `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	UserID      uuid.UUID    `json:"user_id" gorm:"not null;type:uuid"`
	EventID     uuid.UUID    `json:"event_id" gorm:"not null;type:uuid"`
	Quantity    int          `json:"quantity" gorm:"not null;check:quantity > 0"`
	Status      Status       `json:"status" gorm:"not null;default:'CONFIRMED'"` // confirmed, cancelled (immediate booking)
	CreatedAt   time.Time    `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time    `json:"updated_at" gorm:"autoUpdateTime"`
	CancelledAt *time.Time   `json:"cancelled_at,omitempty" gorm:"default:null"`
	User        users.User   `json:"user" gorm:"foreignKey:UserID;references:ID"`
	Event       events.Event `json:"event" gorm:"foreignKey:EventID;references:ID"`
}

// CreateBookingRequest represents the request to create a booking
type CreateBookingRequest struct {
	EventID  string `json:"event_id" binding:"required,uuid"`
	Quantity int    `json:"quantity" binding:"required,min=1,max=10"`
}

// BookingResponse represents booking data in API responses
type BookingResponse struct {
	ID          string            `json:"id"`
	UserID      string            `json:"user_id"`
	EventID     string            `json:"event_id"`
	Quantity    int               `json:"quantity"`
	Status      Status            `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	CancelledAt *time.Time        `json:"cancelled_at,omitempty"`
	User        *BookingUserInfo  `json:"user,omitempty"`
	Event       *BookingEventInfo `json:"event,omitempty"`
}

// BookingUserInfo represents user info in booking responses
type BookingUserInfo struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

// BookingEventInfo represents event info in booking responses
type BookingEventInfo struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Venue    string    `json:"venue"`
	DateTime time.Time `json:"date_time"`
	Price    float64   `json:"price"`
}

// PaginatedBookings represents paginated booking results
type PaginatedBookings struct {
	Bookings   []BookingResponse `json:"bookings"`
	TotalCount int64             `json:"total_count"`
	Page       int               `json:"page"`
	Limit      int               `json:"limit"`
	TotalPages int               `json:"total_pages"`
}

// BookingListQuery represents query parameters for listing bookings
type BookingListQuery struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	Limit    int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Status   string `form:"status" binding:"omitempty,oneof=PENDING CONFIRMED CANCELLED"`
	EventID  string `form:"event_id" binding:"omitempty,uuid"`
	DateFrom string `form:"date_from"`
	DateTo   string `form:"date_to"`
}

// Helper method to convert Booking to BookingResponse
func (b *Booking) ToResponse() BookingResponse {
	response := BookingResponse{
		ID:          b.ID.String(),
		UserID:      b.UserID.String(),
		EventID:     b.EventID.String(),
		Quantity:    b.Quantity,
		Status:      b.Status,
		CreatedAt:   b.CreatedAt,
		UpdatedAt:   b.UpdatedAt,
		CancelledAt: b.CancelledAt,
	}

	// Populate user info if available
	if b.User.ID != uuid.Nil {
		response.User = &BookingUserInfo{
			ID:        b.User.ID.String(),
			FirstName: b.User.FirstName,
			LastName:  b.User.LastName,
			Email:     b.User.Email,
		}
	}

	// Populate event info if available
	if b.Event.ID != uuid.Nil {
		response.Event = &BookingEventInfo{
			ID:       b.Event.ID.String(),
			Name:     b.Event.Name,
			Venue:    b.Event.Venue,
			DateTime: b.Event.DateTime,
			Price:    b.Event.Price,
		}
	}

	return response
}

// TableName specifies the table name for GORM
func (Booking) TableName() string {
	return "bookings"
}
