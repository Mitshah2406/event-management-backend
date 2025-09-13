package venues

import (
	"time"

	"github.com/google/uuid"
)

// Forward declaration for seat
type Seat struct {
	ID         uuid.UUID `json:"id"`
	SectionID  uuid.UUID `json:"section_id"`
	SeatNumber string    `json:"seat_number"`
	Row        string    `json:"row"`
	Position   int       `json:"position"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// VenueTemplate defines the structure for venue templates
type VenueTemplate struct {
	ID                 uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	Name               string    `gorm:"unique;not null" json:"name"`
	Description        string    `json:"description"`
	DefaultRows        int       `json:"default_rows"`
	DefaultSeatsPerRow int       `json:"default_seats_per_row"`
	LayoutType         string    `gorm:"type:varchar(20);index;check:layout_type IN ('THEATER', 'STADIUM', 'CONFERENCE', 'GENERAL')" json:"layout_type"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// VenueSection defines the structure for venue sections (fixed per venue template)
type VenueSection struct {
	ID          uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TemplateID  uuid.UUID `gorm:"type:uuid;not null;index" json:"template_id"`
	Name        string    `gorm:"not null" json:"name"`
	Description string    `json:"description"`
	RowStart    string    `json:"row_start"`
	RowEnd      string    `json:"row_end"`
	SeatsPerRow int       `json:"seats_per_row"`
	TotalSeats  int       `json:"total_seats"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relationships
	Template *VenueTemplate `json:"template,omitempty" gorm:"foreignKey:TemplateID;constraint:OnDelete:RESTRICT;"`
	Seats    []Seat         `json:"seats,omitempty" gorm:"foreignKey:SectionID;constraint:OnDelete:CASCADE;"`
}

// EventPricing defines pricing for venue sections per event
type EventPricing struct {
	ID              uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	EventID         uuid.UUID `gorm:"type:uuid;not null;index" json:"event_id"`
	SectionID       uuid.UUID `gorm:"type:uuid;not null;index" json:"section_id"`
	PriceMultiplier float64   `gorm:"not null;default:1.0" json:"price_multiplier"`
	IsActive        bool      `gorm:"default:true" json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Relationships
	Section *VenueSection `json:"section,omitempty" gorm:"foreignKey:SectionID;constraint:OnDelete:CASCADE;"`

	// Unique constraint: one pricing per event-section combination
	// This is defined in the migration, not here
}

// TableName sets the table name for VenueTemplate
func (VenueTemplate) TableName() string {
	return "venue_templates"
}

// TableName sets the table name for VenueSection
func (VenueSection) TableName() string {
	return "venue_sections"
}

// TableName sets the table name for EventPricing
func (EventPricing) TableName() string {
	return "event_pricing"
}

// Helper methods for event pricing calculations
func (ep *EventPricing) CalculatePrice(basePrice float64) float64 {
	return basePrice * ep.PriceMultiplier
}

// Helper to convert VenueSection to response format with pricing
func (vs *VenueSection) ToResponseWithPricing(basePrice float64, priceMultiplier float64, seats []SeatResponse) VenueSectionResponse {
	availableSeats := 0
	for _, seat := range seats {
		// Count seats that are effectively available (not booked, blocked, or held)
		if seat.Status == "AVAILABLE" {
			availableSeats++
		}
	}

	return VenueSectionResponse{
		ID:              vs.ID.String(),
		Name:            vs.Name,
		PriceMultiplier: priceMultiplier,
		Price:           basePrice * priceMultiplier,
		RowStart:        vs.RowStart,
		RowEnd:          vs.RowEnd,
		SeatsPerRow:     vs.SeatsPerRow,
		TotalSeats:      vs.TotalSeats,
		AvailableSeats:  availableSeats,
		Seats:           seats,
	}
}

// Helper to convert EventPricing to response format
func (ep *EventPricing) ToResponse(sectionName string, basePrice float64) EventPricingResponse {
	return EventPricingResponse{
		ID:              ep.ID.String(),
		EventID:         ep.EventID.String(),
		SectionID:       ep.SectionID.String(),
		SectionName:     sectionName,
		PriceMultiplier: ep.PriceMultiplier,
		Price:           ep.CalculatePrice(basePrice),
		IsActive:        ep.IsActive,
	}
}
