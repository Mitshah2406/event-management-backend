package bookings

type Status string

const (
	StatusConfirmed Status = "CONFIRMED"
	StatusCancelled Status = "CANCELLED"
)

func (s Status) IsValid() bool {
	switch s {
	case StatusConfirmed, StatusCancelled:
		return true
	}
	return false
}

func (s Status) String() string {
	return string(s)
}

func (s Status) CanBeCancelled() bool {
	return s == StatusConfirmed
}

func (s Status) IsActive() bool {
	return s == StatusConfirmed
}
