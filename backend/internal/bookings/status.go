package bookings

type Status string

const (
	StatusPending   Status = "PENDING"
	StatusConfirmed Status = "CONFIRMED"
	StatusCancelled Status = "CANCELLED"
)
