package database

import (
	"evently/internal/bookings"
	"evently/internal/cancellation"
	"evently/internal/events"
	"evently/internal/seats"
	"evently/internal/tags"
	"evently/internal/users"
	"evently/internal/venues"
	"evently/internal/waitlist"

	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	// Run auto-migration first
	err := db.AutoMigrate(
		// Users first
		&users.User{},

		// Tags
		&tags.Tag{},

		// Venue templates and sections
		&venues.VenueTemplate{},
		&venues.VenueSection{},

		// Seats
		&seats.Seat{},

		// Events and relationships
		&events.Event{},
		&tags.EventTag{},
		&venues.EventPricing{},

		// Bookings and payments
		&bookings.Booking{},
		&bookings.SeatBooking{},
		&bookings.Payment{},

		// Cancellation policies and cancellations
		&cancellation.CancellationPolicy{},
		&cancellation.Cancellation{},

		// Waitlist tables
		&waitlist.WaitlistEntry{},
		&waitlist.WaitlistNotification{},
		&waitlist.WaitlistAnalytics{},
	)
	if err != nil {
		return err
	}

	// Add critical constraints for concurrency control
	return MigrateConstraints(db)
}
