package venues

import (
	"time"

	"github.com/google/uuid"
)

type VenueTemplate struct {
	ID                 uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	Name               string    `gorm:"uniqueIndex;not null"`
	Description        string
	DefaultRows        int
	DefaultSeatsPerRow int
	LayoutType         string `gorm:"not null"` // THEATER, STADIUM, etc.
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type VenueSection struct {
	ID              uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	EventID         uuid.UUID `gorm:"type:uuid;not null;index"`
	TemplateID      uuid.UUID `gorm:"type:uuid;not null"`
	Name            string    `gorm:"not null"`
	PriceMultiplier float64   `gorm:"default:1.0"`
	RowStart        string
	RowEnd          string
	SeatsPerRow     int
	TotalSeats      int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
