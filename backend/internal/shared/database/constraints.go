package database

import (
	"gorm.io/gorm"
)

// MigrateConstraints adds database constraints that cannot be handled by GORM AutoMigrate
func MigrateConstraints(db *gorm.DB) error {
	// Note: Unique constraints and basic indexes are now handled by GORM tags in model definitions:
	// - SeatBooking: uniqueIndex:idx_unique_seat_event on (seat_id, event_id)
	// - Booking: index tags on user_id, event_id, status
	// - Individual field indexes are created automatically by GORM

	// Add version column for optimistic locking if it doesn't exist
	// This is needed for existing tables that may not have this column
	err := db.Exec(`
		ALTER TABLE bookings 
		ADD COLUMN IF NOT EXISTS version INTEGER NOT NULL DEFAULT 1;
	`).Error
	if err != nil {
		return err
	}

	// PostgreSQL-specific: Create indexes CONCURRENTLY for better performance during migration
	// GORM doesn't support CONCURRENTLY, so we handle critical performance indexes manually
	err = db.Exec(`
		CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_bookings_id_version 
		ON bookings (id, version);
	`).Error
	if err != nil {
		return err
	}

	return nil
}
