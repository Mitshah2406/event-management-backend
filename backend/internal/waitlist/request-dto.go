package waitlist

import "github.com/google/uuid"

type JoinWaitlistRequest struct {
	EventID     uuid.UUID `json:"event_id" validate:"required"`
	Quantity    int       `json:"quantity" validate:"required,min=1,max=10"`
	Preferences JSONMap   `json:"preferences,omitempty"`
}
