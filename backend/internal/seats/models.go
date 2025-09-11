package seats

import (
	"time"

	"github.com/google/uuid"
)

// Seat defines the structure for individual seats
type Seat struct {
	ID         uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	SectionID  uuid.UUID `gorm:"type:uuid;index;not null;uniqueIndex:idx_section_seat" json:"section_id"`
	SeatNumber string    `gorm:"not null;uniqueIndex:idx_section_seat" json:"seat_number"`
	Row        string    `gorm:"not null" json:"row"`
	Position   int       `gorm:"not null" json:"position"`
	Status     string    `gorm:"type:varchar(20);check:status IN ('AVAILABLE', 'BLOCKED');default:'AVAILABLE'" json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// Relationships
	Section      *VenueSection `json:"section,omitempty" gorm:"foreignKey:SectionID;constraint:OnDelete:CASCADE;"`
	SeatBookings []SeatBooking `json:"seat_bookings,omitempty" gorm:"foreignKey:SeatID;constraint:OnDelete:RESTRICT;"`
}

// Forward declarations
type VenueSection struct {
	ID              uuid.UUID `json:"id"`
	EventID         uuid.UUID `json:"event_id"`
	TemplateID      uuid.UUID `json:"template_id"`
	Name            string    `json:"name"`
	PriceMultiplier float64   `json:"price_multiplier"`
}

type SeatBooking struct {
	ID        uuid.UUID `json:"id"`
	BookingID uuid.UUID `json:"booking_id"`
	SeatID    uuid.UUID `json:"seat_id"`
	SectionID uuid.UUID `json:"section_id"`
	SeatPrice float64   `json:"seat_price"`
}

// SeatInfo represents seat information for external services
type SeatInfo struct {
	ID          uuid.UUID `json:"id"`
	SectionID   uuid.UUID `json:"section_id"`
	SeatNumber  string    `json:"seat_number"`
	Row         string    `json:"row"`
	Price       float64   `json:"price"`
	SectionName string    `json:"section_name"`
}

// TableName sets the table name for Seat
func (Seat) TableName() string {
	return "seats"
}

// Helper methods for seat management
func (s *Seat) IsAvailable() bool {
	return s.Status == "AVAILABLE"
}

func (s *Seat) IsBlocked() bool {
	return s.Status == "BLOCKED"
}

// IsBookedForEvent checks if seat is booked for a specific event
// This should be called with SeatBooking data from the service layer
func (s *Seat) IsBookedForEvent(eventID uuid.UUID, seatBookings []SeatBooking) bool {
	if s.IsBlocked() {
		return false // Blocked seats can't be booked
	}
	
	for _, booking := range seatBookings {
		if booking.SeatID == s.ID {
			return true
		}
	}
	return false
}

// GetEffectiveStatus returns the effective status for a specific event
func (s *Seat) GetEffectiveStatus(eventID uuid.UUID, seatBookings []SeatBooking, isHeld bool) string {
	if s.IsBlocked() {
		return "BLOCKED"
	}
	
	if isHeld {
		return "HELD"
	}
	
	if s.IsBookedForEvent(eventID, seatBookings) {
		return "BOOKED"
	}
	
	return "AVAILABLE"
}

// Convert Seat to SeatResponse with event-specific status
func (s *Seat) ToResponse(eventID uuid.UUID, price float64, isHeld bool, seatBookings []SeatBooking) SeatResponse {
	effectiveStatus := s.GetEffectiveStatus(eventID, seatBookings, isHeld)
	
	return SeatResponse{
		ID:         s.ID.String(),
		SeatNumber: s.SeatNumber,
		Row:        s.Row,
		Position:   s.Position,
		Status:     effectiveStatus, // Event-specific status (AVAILABLE/BOOKED/BLOCKED/HELD)
		Price:      price,
		IsHeld:     isHeld,
	}
}

// SeatResponse for API responses
type SeatResponse struct {
	ID         string  `json:"id"`
	SeatNumber string  `json:"seat_number"`
	Row        string  `json:"row"`
	Position   int     `json:"position"`
	Status     string  `json:"status"`
	Price      float64 `json:"price"`
	IsHeld     bool    `json:"is_held"`
}

// Request/Response models for seat operations
// Note: CreateSeatsRequest removed - seats are now automatically generated when venue sections are created

type UpdateSeatRequest struct {
	SeatNumber *string `json:"seat_number" binding:"omitempty"`
	Row        *string `json:"row" binding:"omitempty"`
	Position   *int    `json:"position" binding:"omitempty,min=1"`
	Status     *string `json:"status" binding:"omitempty,oneof=AVAILABLE BLOCKED"`
}

type BulkUpdateStatusRequest struct {
	SeatIDs []string `json:"seat_ids" binding:"required,min=1"`
	Status  string   `json:"status" binding:"required,oneof=AVAILABLE BLOCKED"`
}

// Seat holding models (Your core booking flow)
type SeatHoldRequest struct {
	EventID string   `json:"event_id" binding:"required,uuid"`
	SeatIDs []string `json:"seat_ids" binding:"required,min=1,max=10"`
	UserID  string   `json:"user_id" binding:"required,uuid"`
}

type SeatHoldResponse struct {
	HoldID     string         `json:"hold_id"`
	EventID    string         `json:"event_id"`
	UserID     string         `json:"user_id"`
	Seats      []HeldSeatInfo `json:"seats"`
	TotalPrice float64        `json:"total_price"`
	ExpiresAt  time.Time      `json:"expires_at"`
	TTL        int            `json:"ttl_seconds"`
}

type HeldSeatInfo struct {
	SeatID      string  `json:"seat_id"`
	SectionID   string  `json:"section_id"`
	SeatNumber  string  `json:"seat_number"`
	Row         string  `json:"row"`
	SectionName string  `json:"section_name"`
	Price       float64 `json:"price"`
}

// Hold validation models
type HoldValidationResult struct {
	Valid   bool             `json:"valid"`
	Reason  string           `json:"reason,omitempty"`
	Details *SeatHoldDetails `json:"details,omitempty"`
	TTL     int              `json:"ttl_seconds,omitempty"`
}

// Availability models
type SeatAvailabilityResponse struct {
	Seats []SeatAvailabilityInfo `json:"seats"`
}

type SeatAvailabilityInfo struct {
	SeatID    string `json:"seat_id"`
	Available bool   `json:"available"`
	Status    string `json:"status"` // AVAILABLE, BOOKED, BLOCKED, HELD
	HoldInfo  string `json:"hold_info,omitempty"`
}
