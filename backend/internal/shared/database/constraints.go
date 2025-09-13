package database

import (
	"gorm.io/gorm"
)

// MigrateConstraints adds critical database constraints for concurrency control
func MigrateConstraints(db *gorm.DB) error {
	// Add unique constraint to prevent double booking of seats for same event
	err := db.Exec(`
		ALTER TABLE seat_bookings 
		ADD CONSTRAINT IF NOT EXISTS unique_seat_per_event 
		UNIQUE (seat_id, event_id);
	`).Error
	if err != nil {
		return err
	}

	// Add index for better performance on seat availability queries
	err = db.Exec(`
		CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_seat_bookings_seat_event_performance 
		ON seat_bookings (seat_id, event_id);
	`).Error
	if err != nil {
		return err
	}

	// Add index for booking queries by event
	err = db.Exec(`
		CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_seat_bookings_event_id 
		ON seat_bookings (event_id);
	`).Error
	if err != nil {
		return err
	}

	return nil
}
