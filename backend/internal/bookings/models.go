package bookings

import (
	"evently/internal/events"
	"evently/internal/users"

	"github.com/google/uuid"
)

type Booking struct {
	ID       uuid.UUID    `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	UserID   uuid.UUID    `json:"user_id" gorm:"not null;type:uuid"`
	EventID  uuid.UUID    `json:"event_id" gorm:"not null;type:uuid"`
	Quantity int          `json:"quantity" gorm:"not null"`
	Status   Status       `json:"status" gorm:"not null;default:'PENDING'"` // pending, confirmed, cancelled
	BookedAt string       `json:"booked_at" gorm:"autoCreateTime"`
	User     users.User   `json:"user" gorm:"foreignKey:UserID;references:ID"`
	Event    events.Event `json:"event" gorm:"foreignKey:EventID;references:ID"`
}
