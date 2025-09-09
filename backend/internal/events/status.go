package events

type Status string

const (
	StatusUpcoming Status = "UPCOMING"
	StatusActive   Status = "ACTIVE"
	StatusEnded    Status = "ENDED"
)
