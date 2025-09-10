package bookings

type Status string

const (
	StatusConfirmed Status = "CONFIRMED"
	StatusCancelled Status = "CANCELLED"
)

// IsValid checks if the booking status is valid
func (s Status) IsValid() bool {
	switch s {
	case StatusConfirmed, StatusCancelled:
		return true
	}
	return false
}

// String returns the string representation of Status
func (s Status) String() string {
	return string(s)
}

// CanBeCancelled checks if a booking with this status can be cancelled
func (s Status) CanBeCancelled() bool {
	return s == StatusConfirmed
}

// IsActive checks if the booking is active (not cancelled)
func (s Status) IsActive() bool {
	return s == StatusConfirmed
}
