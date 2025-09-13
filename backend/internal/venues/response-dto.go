package venues

import "time"

type VenueLayoutResponse struct {
	EventID        string                 `json:"event_id"`
	EventName      string                 `json:"event_name"`
	VenueInfo      VenueInfo              `json:"venue_info"`
	BasePrice      float64                `json:"base_price"`
	Sections       []VenueSectionResponse `json:"sections"`
	TotalSeats     int                    `json:"total_seats"`
	AvailableSeats int                    `json:"available_seats"`
}

type VenueInfo struct {
	TemplateID   string `json:"template_id"`
	TemplateName string `json:"template_name"`
	LayoutType   string `json:"layout_type"`
	Description  string `json:"description"`
}

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

type SeatResponse struct {
	ID         string  `json:"id"`
	SeatNumber string  `json:"seat_number"`
	Row        string  `json:"row"`
	Position   int     `json:"position"`
	Status     string  `json:"status"`
	Price      float64 `json:"price"`
	IsHeld     bool    `json:"is_held"`
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

type EventPricingResponse struct {
	ID              string  `json:"id"`
	EventID         string  `json:"event_id"`
	SectionID       string  `json:"section_id"`
	SectionName     string  `json:"section_name"`
	PriceMultiplier float64 `json:"price_multiplier"`
	Price           float64 `json:"price"`
	IsActive        bool    `json:"is_active"`
}
