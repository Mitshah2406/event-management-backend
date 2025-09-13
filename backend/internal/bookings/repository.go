package bookings

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, booking *Booking) error
	CreateAtomic(ctx context.Context, booking *Booking) error
	CheckSeatBookingConflicts(ctx context.Context, seatIDs []uuid.UUID, eventID uuid.UUID) ([]uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Booking, error)
	GetByHoldID(ctx context.Context, holdID string) (*Booking, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Booking, error)
	Update(ctx context.Context, booking *Booking) error
	UpdateWithVersion(ctx context.Context, booking *Booking) error
	Cancel(ctx context.Context, id uuid.UUID) error
	CancelWithVersion(ctx context.Context, id uuid.UUID, expectedVersion int) error

	// Payment operations
	CreatePayment(ctx context.Context, payment *Payment) error
	UpdatePayment(ctx context.Context, payment *Payment) error
	GetPaymentByID(ctx context.Context, paymentID uuid.UUID) (*Payment, error)

	// Seat booking operations
	CreateSeatBookings(ctx context.Context, seatBookings []SeatBooking) error
	GetSeatBookingsByBookingID(ctx context.Context, bookingID uuid.UUID) ([]SeatBooking, error)
	DeleteSeatBookingsByBookingID(ctx context.Context, bookingID uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, booking *Booking) error {
	// Use atomic version for better concurrency control
	return r.CreateAtomic(ctx, booking)
}

// CreateAtomic creates a booking with atomic conflict checking
func (r *repository) CreateAtomic(ctx context.Context, booking *Booking) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Store associations temporarily
		seatBookings := booking.SeatBookings
		payments := booking.Payments

		// Extract seat IDs and event ID for conflict checking
		if len(seatBookings) > 0 {
			seatIDs := make([]uuid.UUID, len(seatBookings))
			for i, sb := range seatBookings {
				seatIDs[i] = sb.SeatID
			}

			// Check for existing bookings with SELECT FOR UPDATE to prevent race conditions
			// Only consider non-cancelled bookings as conflicts
			var existingCount int64
			err := tx.Table("seat_bookings sb").
				Joins("JOIN bookings b ON b.id = sb.booking_id").
				Where("sb.seat_id IN ? AND sb.event_id = ? AND b.status != 'CANCELLED'", seatIDs, booking.EventID).
				Count(&existingCount).Error
			if err != nil {
				return fmt.Errorf("failed to check seat booking conflicts: %w", err)
			}

			if existingCount > 0 {
				return fmt.Errorf("one or more seats are already booked for this event")
			}
		}

		// Clear associations to avoid GORM auto-creating them
		booking.SeatBookings = nil
		booking.Payments = nil

		// Set initial version if not set
		if booking.Version == 0 {
			booking.Version = 1
		}

		// Create the main booking record first
		if err := tx.Create(booking).Error; err != nil {
			return fmt.Errorf("failed to create booking: %w", err)
		}

		// Create seat bookings with the generated BookingID and EventID
		if len(seatBookings) > 0 {
			for i := range seatBookings {
				seatBookings[i].BookingID = booking.ID
				seatBookings[i].EventID = booking.EventID // Ensure EventID is set
			}
			if err := tx.Create(&seatBookings).Error; err != nil {
				return fmt.Errorf("failed to create seat bookings: %w", err)
			}
			booking.SeatBookings = seatBookings
		}

		// Create payment records with the generated BookingID
		if len(payments) > 0 {
			for i := range payments {
				payments[i].BookingID = booking.ID
			}
			if err := tx.Create(&payments).Error; err != nil {
				return fmt.Errorf("failed to create payments: %w", err)
			}
			booking.Payments = payments
		}

		return nil
	})
}

// CheckSeatBookingConflicts checks for existing seat bookings that would conflict
func (r *repository) CheckSeatBookingConflicts(ctx context.Context, seatIDs []uuid.UUID, eventID uuid.UUID) ([]uuid.UUID, error) {
	var conflictingSeats []uuid.UUID
	err := r.db.WithContext(ctx).
		Table("seat_bookings sb").
		Joins("JOIN bookings b ON b.id = sb.booking_id").
		Where("sb.seat_id IN ? AND sb.event_id = ? AND b.status != 'CANCELLED'", seatIDs, eventID).
		Pluck("sb.seat_id", &conflictingSeats).Error

	if err != nil {
		return nil, fmt.Errorf("failed to check seat booking conflicts: %w", err)
	}

	return conflictingSeats, nil
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*Booking, error) {
	var booking Booking
	err := r.db.WithContext(ctx).
		Preload("SeatBookings").
		Preload("Payments").
		First(&booking, "id = ?", id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("booking not found")
		}
		return nil, fmt.Errorf("failed to get booking: %w", err)
	}

	return &booking, nil
}

func (r *repository) GetByHoldID(ctx context.Context, holdID string) (*Booking, error) {
	var booking Booking
	err := r.db.WithContext(ctx).
		Preload("SeatBookings").
		Preload("Payments").
		First(&booking, "booking_ref = ?", holdID).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("booking not found for hold ID: %s", holdID)
		}
		return nil, fmt.Errorf("failed to get booking by hold ID: %w", err)
	}

	return &booking, nil
}

func (r *repository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Booking, error) {
	var bookings []Booking
	query := r.db.WithContext(ctx).
		Preload("SeatBookings").
		Preload("Payments").
		Where("user_id = ?", userID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&bookings).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get user bookings: %w", err)
	}

	return bookings, nil
}

func (r *repository) Update(ctx context.Context, booking *Booking) error {
	booking.UpdatedAt = time.Now()
	err := r.db.WithContext(ctx).Save(booking).Error
	if err != nil {
		return fmt.Errorf("failed to update booking: %w", err)
	}
	return nil
}

// UpdateWithVersion updates a booking with optimistic locking
func (r *repository) UpdateWithVersion(ctx context.Context, booking *Booking) error {
	// Read current version first
	var currentBooking Booking
	if err := r.db.WithContext(ctx).First(&currentBooking, "id = ?", booking.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("booking not found")
		}
		return fmt.Errorf("failed to get current booking: %w", err)
	}

	// Check if version matches (optimistic lock check)
	if currentBooking.Version != booking.Version {
		return fmt.Errorf("booking was modified by another process (version mismatch: expected %d, got %d)",
			booking.Version, currentBooking.Version)
	}

	// Update with version increment using atomic transaction
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(booking).
			Where("id = ? AND version = ?", booking.ID, booking.Version).
			Updates(map[string]interface{}{
				"user_id":      booking.UserID,
				"event_id":     booking.EventID,
				"total_seats":  booking.TotalSeats,
				"total_price":  booking.TotalPrice,
				"status":       booking.Status,
				"booking_ref":  booking.BookingRef,
				"updated_at":   time.Now(),
				"cancelled_at": booking.CancelledAt,
				"version":      gorm.Expr("version + 1"), // Atomic increment
			})

		if result.Error != nil {
			return fmt.Errorf("failed to update booking: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			return fmt.Errorf("booking was modified by another process during update")
		}

		booking.Version++ // Update in-memory version
		booking.UpdatedAt = time.Now()
		return nil
	})
}

func (r *repository) Cancel(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get the booking first
		var booking Booking
		if err := tx.First(&booking, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("booking not found")
			}
			return fmt.Errorf("failed to get booking: %w", err)
		}

		// Cancel the booking
		booking.Cancel()
		if err := tx.Save(&booking).Error; err != nil {
			return fmt.Errorf("failed to cancel booking: %w", err)
		}

		// Delete associated seat bookings to free up seats for future bookings
		if err := tx.Where("booking_id = ?", id).Delete(&SeatBooking{}).Error; err != nil {
			return fmt.Errorf("failed to delete seat bookings: %w", err)
		}

		return nil
	})
}

// CancelWithVersion cancels a booking with optimistic locking
func (r *repository) CancelWithVersion(ctx context.Context, id uuid.UUID, expectedVersion int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get current booking with version
		var booking Booking
		if err := tx.First(&booking, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("booking not found")
			}
			return fmt.Errorf("failed to get booking: %w", err)
		}

		// Optimistic lock check
		if booking.Version != expectedVersion {
			return fmt.Errorf("booking was modified by another process (version mismatch: expected %d, got %d)",
				expectedVersion, booking.Version)
		}

		// Update status with version increment
		now := time.Now()
		result := tx.Model(&booking).
			Where("id = ? AND version = ?", id, expectedVersion).
			Updates(map[string]interface{}{
				"status":       "CANCELLED",
				"cancelled_at": &now,
				"updated_at":   now,
				"version":      gorm.Expr("version + 1"),
			})

		if result.Error != nil {
			return fmt.Errorf("failed to cancel booking: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			return fmt.Errorf("booking was modified by another process during cancellation")
		}

		// Delete associated seat bookings to free up seats for future bookings
		if err := tx.Where("booking_id = ?", id).Delete(&SeatBooking{}).Error; err != nil {
			return fmt.Errorf("failed to delete seat bookings: %w", err)
		}

		return nil
	})
}

func (r *repository) CreatePayment(ctx context.Context, payment *Payment) error {
	err := r.db.WithContext(ctx).Create(payment).Error
	if err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}
	return nil
}

func (r *repository) UpdatePayment(ctx context.Context, payment *Payment) error {
	payment.UpdatedAt = time.Now()
	err := r.db.WithContext(ctx).Save(payment).Error
	if err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}
	return nil
}

func (r *repository) GetPaymentByID(ctx context.Context, paymentID uuid.UUID) (*Payment, error) {
	var payment Payment
	err := r.db.WithContext(ctx).First(&payment, "id = ?", paymentID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("payment not found")
		}
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}
	return &payment, nil
}

func (r *repository) CreateSeatBookings(ctx context.Context, seatBookings []SeatBooking) error {
	if len(seatBookings) == 0 {
		return nil
	}

	err := r.db.WithContext(ctx).Create(&seatBookings).Error
	if err != nil {
		return fmt.Errorf("failed to create seat bookings: %w", err)
	}
	return nil
}

func (r *repository) GetSeatBookingsByBookingID(ctx context.Context, bookingID uuid.UUID) ([]SeatBooking, error) {
	var seatBookings []SeatBooking
	err := r.db.WithContext(ctx).
		Where("booking_id = ?", bookingID).
		Find(&seatBookings).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get seat bookings: %w", err)
	}

	return seatBookings, nil
}

func (r *repository) DeleteSeatBookingsByBookingID(ctx context.Context, bookingID uuid.UUID) error {
	err := r.db.WithContext(ctx).
		Where("booking_id = ?", bookingID).
		Delete(&SeatBooking{}).Error

	if err != nil {
		return fmt.Errorf("failed to delete seat bookings: %w", err)
	}

	return nil
}
