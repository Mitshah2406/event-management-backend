package seats

import (
	"time"

	"github.com/google/uuid"
)

// Seat Schema
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

// Helpers

func (Seat) TableName() string {
	return "seats"
}

func (s *Seat) IsAvailable() bool {
	return s.Status == "AVAILABLE"
}

func (s *Seat) IsBlocked() bool {
	return s.Status == "BLOCKED"
}

func (s *Seat) IsBookedForEvent(eventID uuid.UUID, seatBookings []SeatBooking) bool {
	if s.IsBlocked() {
		return false
	}

	for _, booking := range seatBookings {
		if booking.SeatID == s.ID {
			return true
		}
	}
	return false
}

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
