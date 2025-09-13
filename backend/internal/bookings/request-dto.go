package bookings

type BookingConfirmationRequest struct {
	HoldID        string `json:"hold_id" binding:"required"`
	EventID       string `json:"event_id" binding:"required,uuid"`
	PaymentMethod string `json:"payment_method" binding:"required"`
}
