package events

import "time"

type Event struct {
	ID          string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Name        string    `json:"name" gorm:"not null"`
	Description string    `json:"description"`
	Venue       string    `json:"venue" gorm:"not null"`
	DateTime    string    `json:"date_time" gorm:"not null"`
	Capacity    int       `json:"capacity" gorm:"not null"`
	Available   int       `json:"available" gorm:"not null"`
	Price       float64   `json:"price" gorm:"not null"`
	Status      Status    `json:"status" gorm:"not null;default:'UPCOMING'"` // upcoming, ongoing, completed, cancelled
	OrganizerID string    `json:"organizer_id" gorm:"not null"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
