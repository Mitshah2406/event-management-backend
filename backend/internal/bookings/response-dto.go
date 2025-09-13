package bookings

import "time"

type BookingConfirmationResponse struct {
	BookingID  string           `json:"booking_id"`
	BookingRef string           `json:"booking_ref"`
	Status     string           `json:"status"`
	TotalPrice float64          `json:"total_price"`
	TotalSeats int              `json:"total_seats"`
	Seats      []BookedSeatInfo `json:"seats"`
	Payment    PaymentInfo      `json:"payment"`
	CreatedAt  time.Time        `json:"created_at"`
}

type BookedSeatInfo struct {
	SeatID      string  `json:"seat_id"`
	SectionID   string  `json:"section_id"`
	SeatNumber  string  `json:"seat_number"`
	Row         string  `json:"row"`
	SectionName string  `json:"section_name"`
	Price       float64 `json:"price"`
}

type PaymentInfo struct {
	ID            string     `json:"id"`
	Amount        float64    `json:"amount"`
	Currency      string     `json:"currency"`
	Status        string     `json:"status"`
	PaymentMethod string     `json:"payment_method"`
	TransactionID string     `json:"transaction_id"`
	ProcessedAt   *time.Time `json:"processed_at,omitempty"`
}
