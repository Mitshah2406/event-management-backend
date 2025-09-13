package seats

type UpdateSeatRequest struct {
	SeatNumber *string `json:"seat_number" binding:"omitempty"`
	Row        *string `json:"row" binding:"omitempty"`
	Position   *int    `json:"position" binding:"omitempty,min=1"`
	Status     *string `json:"status" binding:"omitempty,oneof=AVAILABLE BLOCKED"`
}

// Seat holding models (Your core booking flow)
type SeatHoldRequest struct {
	EventID string   `json:"event_id" binding:"required,uuid"`
	SeatIDs []string `json:"seat_ids" binding:"required,min=1"`
	UserID  string   `json:"user_id" binding:"required,uuid"`
}
