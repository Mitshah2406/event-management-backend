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
	LayoutType         string    `gorm:"type:varchar(20);check:layout_type IN ('THEATER', 'STADIUM', 'CONFERENCE', 'GENERAL')" json:"layout_type"`
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

// VenueLayoutResponse represents the venue layout for an event
type VenueLayoutResponse struct {
	EventID        string                 `json:"event_id"`
	EventName      string                 `json:"event_name"`
	VenueInfo      VenueInfo              `json:"venue_info"`
	BasePrice      float64                `json:"base_price"`
	Sections       []VenueSectionResponse `json:"sections"`
	TotalSeats     int                    `json:"total_seats"`
	AvailableSeats int                    `json:"available_seats"`
}

// VenueInfo represents basic venue information
type VenueInfo struct {
	TemplateID   string `json:"template_id"`
	TemplateName string `json:"template_name"`
	LayoutType   string `json:"layout_type"`
	Description  string `json:"description"`
}

// VenueSectionResponse represents venue section with seat details
type VenueSectionResponse struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	PriceMultiplier float64        `json:"price_multiplier"`
	Price           float64        `json:"price"`
	RowStart        string         `json:"row_start"`
	RowEnd          string         `json:"row_end"`
	SeatsPerRow     int            `json:"seats_per_row"`
	TotalSeats      int            `json:"total_seats"`
	AvailableSeats  int            `json:"available_seats"`
	Seats           []SeatResponse `json:"seats"`
}

// SeatResponse represents individual seat information
type SeatResponse struct {
	ID         string  `json:"id"`
	SeatNumber string  `json:"seat_number"`
	Row        string  `json:"row"`
	Position   int     `json:"position"`
	Status     string  `json:"status"`
	Price      float64 `json:"price"`
	IsHeld     bool    `json:"is_held"`
}

// SeatHoldRequest represents seat holding request
type SeatHoldRequest struct {
	EventID string   `json:"event_id" binding:"required,uuid"`
	SeatIDs []string `json:"seat_ids" binding:"required,min=1,max=10"`
	UserID  string   `json:"user_id" binding:"required,uuid"`
}

// SeatHoldResponse represents seat holding response
type SeatHoldResponse struct {
	HoldID     string         `json:"hold_id"`
	EventID    string         `json:"event_id"`
	UserID     string         `json:"user_id"`
	Seats      []HeldSeatInfo `json:"seats"`
	TotalPrice float64        `json:"total_price"`
	ExpiresAt  time.Time      `json:"expires_at"`
	TTL        int            `json:"ttl_seconds"`
}

// HeldSeatInfo represents information about held seats
type HeldSeatInfo struct {
	SeatID      string  `json:"seat_id"`
	SectionID   string  `json:"section_id"`
	SeatNumber  string  `json:"seat_number"`
	Row         string  `json:"row"`
	SectionName string  `json:"section_name"`
	Price       float64 `json:"price"`
}

// Request/Response models for venue templates
type CreateTemplateRequest struct {
	Name               string `json:"name" binding:"required,min=3,max=255"`
	Description        string `json:"description" binding:"max=1000"`
	DefaultRows        int    `json:"default_rows" binding:"required,min=1,max=50"`
	DefaultSeatsPerRow int    `json:"default_seats_per_row" binding:"required,min=1,max=100"`
	LayoutType         string `json:"layout_type" binding:"required,oneof=THEATER STADIUM CONFERENCE GENERAL"`
}

type UpdateTemplateRequest struct {
	Name               *string `json:"name" binding:"omitempty,min=3,max=255"`
	Description        *string `json:"description" binding:"omitempty,max=1000"`
	DefaultRows        *int    `json:"default_rows" binding:"omitempty,min=1,max=50"`
	DefaultSeatsPerRow *int    `json:"default_seats_per_row" binding:"omitempty,min=1,max=100"`
	LayoutType         *string `json:"layout_type" binding:"omitempty,oneof=THEATER STADIUM CONFERENCE GENERAL"`
}

// Request/Response models for venue sections
type CreateSectionRequest struct {
	TemplateID  string `json:"template_id" binding:"required,uuid"`
	Name        string `json:"name" binding:"required,min=1,max=255"`
	Description string `json:"description" binding:"omitempty,max=500"`
	RowStart    string `json:"row_start" binding:"max=10"`
	RowEnd      string `json:"row_end" binding:"max=10"`
	SeatsPerRow int    `json:"seats_per_row" binding:"required,min=1,max=100"`
	TotalSeats  int    `json:"total_seats" binding:"required,min=1"`
}

type UpdateSectionRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=1,max=255"`
	Description *string `json:"description" binding:"omitempty,max=500"`
	RowStart    *string `json:"row_start" binding:"omitempty,max=10"`
	RowEnd      *string `json:"row_end" binding:"omitempty,max=10"`
	SeatsPerRow *int    `json:"seats_per_row" binding:"omitempty,min=1,max=100"`
	TotalSeats  *int    `json:"total_seats" binding:"omitempty,min=1"`
}

// Request/Response models for event pricing
type CreateEventPricingRequest struct {
	EventID         string  `json:"event_id" binding:"required,uuid"`
	SectionID       string  `json:"section_id" binding:"required,uuid"`
	PriceMultiplier float64 `json:"price_multiplier" binding:"required,min=0.1,max=10"`
}

type UpdateEventPricingRequest struct {
	PriceMultiplier *float64 `json:"price_multiplier" binding:"omitempty,min=0.1,max=10"`
	IsActive        *bool    `json:"is_active"`
}

type EventPricingResponse struct {
	ID              string  `json:"id"`
	EventID         string  `json:"event_id"`
	SectionID       string  `json:"section_id"`
	SectionName     string  `json:"section_name"`
	PriceMultiplier float64 `json:"price_multiplier"`
	Price           float64 `json:"price"`
	IsActive        bool    `json:"is_active"`
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
