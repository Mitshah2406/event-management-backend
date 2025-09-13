package seats

import "time"

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
