package database

import (
	"evently/internal/bookings"
	"evently/internal/events"
	"evently/internal/users"

	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&users.User{},
		&events.Event{},
		&bookings.Booking{},
	)
}
