package cancellation

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	// Cancellation Policy operations
	CreateCancellationPolicy(ctx context.Context, policy *CancellationPolicy) error
	GetCancellationPolicyByEventID(ctx context.Context, eventID uuid.UUID) (*CancellationPolicy, error)
	UpdateCancellationPolicy(ctx context.Context, policy *CancellationPolicy) error

	// Cancellation operations
	CreateCancellation(ctx context.Context, cancellation *Cancellation) error
	GetCancellationByID(ctx context.Context, id uuid.UUID) (*Cancellation, error)
	GetCancellationsByUserID(ctx context.Context, userID uuid.UUID) ([]Cancellation, error)
	GetCancellationByBookingID(ctx context.Context, bookingID uuid.UUID) (*Cancellation, error)
	UpdateCancellation(ctx context.Context, cancellation *Cancellation) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreateCancellationPolicy(ctx context.Context, policy *CancellationPolicy) error {
	err := r.db.WithContext(ctx).Create(policy).Error
	if err != nil {
		return fmt.Errorf("failed to create cancellation policy: %w", err)
	}
	return nil
}

func (r *repository) GetCancellationPolicyByEventID(ctx context.Context, eventID uuid.UUID) (*CancellationPolicy, error) {
	var policy CancellationPolicy
	err := r.db.WithContext(ctx).First(&policy, "event_id = ?", eventID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("cancellation policy not found for event: %s", eventID)
		}
		return nil, fmt.Errorf("failed to get cancellation policy: %w", err)
	}
	return &policy, nil
}

func (r *repository) UpdateCancellationPolicy(ctx context.Context, policy *CancellationPolicy) error {
	err := r.db.WithContext(ctx).Save(policy).Error
	if err != nil {
		return fmt.Errorf("failed to update cancellation policy: %w", err)
	}
	return nil
}

func (r *repository) CreateCancellation(ctx context.Context, cancellation *Cancellation) error {
	err := r.db.WithContext(ctx).Create(cancellation).Error
	if err != nil {
		return fmt.Errorf("failed to create cancellation: %w", err)
	}
	return nil
}

func (r *repository) GetCancellationByID(ctx context.Context, id uuid.UUID) (*Cancellation, error) {
	var cancellation Cancellation
	err := r.db.WithContext(ctx).First(&cancellation, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("cancellation not found")
		}
		return nil, fmt.Errorf("failed to get cancellation: %w", err)
	}
	return &cancellation, nil
}

func (r *repository) GetCancellationsByUserID(ctx context.Context, userID uuid.UUID) ([]Cancellation, error) {
	var cancellations []Cancellation

	err := r.db.WithContext(ctx).
		Joins("JOIN bookings ON cancellations.booking_id = bookings.id").
		Where("bookings.user_id = ?", userID).
		Order("cancellations.created_at DESC").
		Find(&cancellations).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get user cancellations: %w", err)
	}

	return cancellations, nil
}

func (r *repository) GetCancellationByBookingID(ctx context.Context, bookingID uuid.UUID) (*Cancellation, error) {
	var cancellation Cancellation
	err := r.db.WithContext(ctx).First(&cancellation, "booking_id = ?", bookingID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("cancellation not found for booking: %s", bookingID)
		}
		return nil, fmt.Errorf("failed to get cancellation by booking ID: %w", err)
	}
	return &cancellation, nil
}

func (r *repository) UpdateCancellation(ctx context.Context, cancellation *Cancellation) error {
	err := r.db.WithContext(ctx).Save(cancellation).Error
	if err != nil {
		return fmt.Errorf("failed to update cancellation: %w", err)
	}
	return nil
}
