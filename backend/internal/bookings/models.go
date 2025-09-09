package bookings

import (
	"evently/internal/events"
	"evently/internal/users"
)

type Booking struct {
	ID       string       `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID   string       `json:"user_id" gorm:"not null;type:uuid"`
	EventID  string       `json:"event_id" gorm:"not null;type:uuid"`
	Quantity int          `json:"quantity" gorm:"not null"`
	Status   Status       `json:"status" gorm:"not null;default:'PENDING'"` // pending, confirmed, cancelled
	BookedAt string       `json:"booked_at" gorm:"autoCreateTime"`
	User     users.User   `json:"user" gorm:"foreignKey:UserID;references:ID"`
	Event    events.Event `json:"event" gorm:"foreignKey:EventID;references:ID"`
}
