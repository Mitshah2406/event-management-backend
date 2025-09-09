package database

import (
	"evently/internal/bookings"
	"evently/internal/events"
	"evently/internal/tags"
	"evently/internal/users"
	"log"

	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	log.Println("Running database migrations...")
	return db.AutoMigrate(
		&users.User{},
		&tags.Tag{},
		&events.Event{},
		&tags.EventTag{},
		&bookings.Booking{},
	)
}
