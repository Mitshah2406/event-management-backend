package seats

import (
	"time"

	"github.com/google/uuid"
)

type Seat struct {
	ID         uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	SectionID  uuid.UUID `gorm:"type:uuid;not null;index"`
	SeatNumber string    `gorm:"not null"`
	Row        string    `gorm:"not null"`
	Position   int       `gorm:"not null"`
	Status     string    `gorm:"not null;default:'AVAILABLE'"` // AVAILABLE, BOOKED, BLOCKED
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
