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
	GetByID(ctx context.Context, id uuid.UUID) (*Booking, error)
	GetByHoldID(ctx context.Context, holdID string) (*Booking, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Booking, error)
	Update(ctx context.Context, booking *Booking) error
	Cancel(ctx context.Context, id uuid.UUID) error

	// Payment operations
	CreatePayment(ctx context.Context, payment *Payment) error
	UpdatePayment(ctx context.Context, payment *Payment) error
	GetPaymentByID(ctx context.Context, paymentID uuid.UUID) (*Payment, error)

	// Seat booking operations
	CreateSeatBookings(ctx context.Context, seatBookings []SeatBooking) error
	GetSeatBookingsByBookingID(ctx context.Context, bookingID uuid.UUID) ([]SeatBooking, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, booking *Booking) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Store associations temporarily
		seatBookings := booking.SeatBookings
		payments := booking.Payments

		// Clear associations to avoid GORM auto-creating them
		booking.SeatBookings = nil
		booking.Payments = nil

		// Create the main booking record first
		if err := tx.Create(booking).Error; err != nil {
			return fmt.Errorf("failed to create booking: %w", err)
		}

		// Create seat bookings with the generated BookingID
		if len(seatBookings) > 0 {
			for i := range seatBookings {
				seatBookings[i].BookingID = booking.ID
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
