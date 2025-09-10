package bookings

import (
	"context"
	"errors"
	"evently/internal/events"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	// Core booking operations
	CreateBooking(ctx context.Context, booking *Booking) error
	GetBookingByID(ctx context.Context, id uuid.UUID) (*Booking, error)
	GetBookingByIDWithRelations(ctx context.Context, id uuid.UUID) (*Booking, error)
	UpdateBookingStatus(ctx context.Context, id uuid.UUID, status Status, cancelledAt *time.Time) error

	// User booking operations
	GetUserBookings(ctx context.Context, userID uuid.UUID, query BookingListQuery) ([]Booking, int64, error)

	// Admin operations
	GetAllBookings(ctx context.Context, query BookingListQuery) ([]Booking, int64, error)
	GetBookingsByEventID(ctx context.Context, eventID uuid.UUID) ([]Booking, error)

	// Concurrency-safe booking creation
	CreateBookingWithCapacityCheck(ctx context.Context, booking *Booking) error

	// Capacity and validation
	CheckEventCapacity(ctx context.Context, eventID uuid.UUID, requestedQuantity int) (bool, error)
	GetEventBookedCount(ctx context.Context, eventID uuid.UUID) (int, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreateBooking(ctx context.Context, booking *Booking) error {
	return r.db.WithContext(ctx).Create(booking).Error
}

func (r *repository) GetBookingByID(ctx context.Context, id uuid.UUID) (*Booking, error) {
	var booking Booking
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&booking).Error
	if err != nil {
		return nil, err
	}
	return &booking, nil
}

func (r *repository) GetBookingByIDWithRelations(ctx context.Context, id uuid.UUID) (*Booking, error) {
	var booking Booking
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Event").
		Where("id = ?", id).
		First(&booking).Error
	if err != nil {
		return nil, err
	}
	return &booking, nil
}

func (r *repository) UpdateBookingStatus(ctx context.Context, id uuid.UUID, status Status, cancelledAt *time.Time) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if cancelledAt != nil {
		updates["cancelled_at"] = *cancelledAt
	}

	return r.db.WithContext(ctx).
		Model(&Booking{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (r *repository) GetUserBookings(ctx context.Context, userID uuid.UUID, query BookingListQuery) ([]Booking, int64, error) {
	var bookings []Booking
	var totalCount int64

	// Set defaults
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Limit <= 0 {
		query.Limit = 10
	}

	// Build base query
	baseQuery := r.db.WithContext(ctx).
		Model(&Booking{}).
		Where("user_id = ?", userID)

	// Apply filters
	baseQuery = r.applyFilters(baseQuery, query)

	// Get total count
	if err := baseQuery.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results with relations
	offset := (query.Page - 1) * query.Limit
	err := baseQuery.
		Preload("User").
		Preload("Event").
		Order("created_at DESC").
		Offset(offset).
		Limit(query.Limit).
		Find(&bookings).Error

	return bookings, totalCount, err
}

func (r *repository) GetAllBookings(ctx context.Context, query BookingListQuery) ([]Booking, int64, error) {
	var bookings []Booking
	var totalCount int64

	// Set defaults
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Limit <= 0 {
		query.Limit = 10
	}

	// Build base query
	baseQuery := r.db.WithContext(ctx).Model(&Booking{})

	// Apply filters
	baseQuery = r.applyFilters(baseQuery, query)

	// Get total count
	if err := baseQuery.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results with relations
	offset := (query.Page - 1) * query.Limit
	err := baseQuery.
		Preload("User").
		Preload("Event").
		Order("created_at DESC").
		Offset(offset).
		Limit(query.Limit).
		Find(&bookings).Error

	return bookings, totalCount, err
}

func (r *repository) GetBookingsByEventID(ctx context.Context, eventID uuid.UUID) ([]Booking, error) {
	var bookings []Booking
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("event_id = ?", eventID).
		Where("status = ?", StatusConfirmed). // Only confirmed bookings
		Order("created_at DESC").
		Find(&bookings).Error

	return bookings, err
}

// CreateBookingWithCapacityCheck creates a booking atomically with capacity validation
func (r *repository) CreateBookingWithCapacityCheck(ctx context.Context, booking *Booking) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Lock the event row for update to prevent race conditions
		var event struct {
			ID            uuid.UUID `gorm:"column:id"`
			TotalCapacity int       `gorm:"column:total_capacity"`
			BookedCount   int       `gorm:"column:booked_count"`
			Status        string    `gorm:"column:status"`
		}

		err := tx.Table("events").
			Select("id, total_capacity, booked_count, status").
			Where("id = ?", booking.EventID).
			Set("gorm:query_option", "FOR UPDATE").
			First(&event).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("event not found")
			}
			return fmt.Errorf("failed to lock event: %w", err)
		}

		// 2. Check if event can be booked
		if event.Status != "published" {
			return errors.New("event is not available for booking")
		}

		// 3. Check capacity
		newBookedCount := event.BookedCount + booking.Quantity
		if newBookedCount > event.TotalCapacity {
			availableTickets := event.TotalCapacity - event.BookedCount
			if availableTickets <= 0 {
				return errors.New("event is fully booked")
			}
			return fmt.Errorf("insufficient capacity: only %d tickets available, requested %d",
				availableTickets, booking.Quantity)
		}
		log.Println("Capacity check passed: event", event.ID, " total capacity:", event.TotalCapacity, " booked count:", event.BookedCount, " requested:", booking.Quantity, " new booked count:", newBookedCount)
		// 4. Create the booking
		if err := tx.Create(booking).Error; err != nil {
			return fmt.Errorf("failed to create booking: %w", err)
		}

		// 5. Update event booked count
		err = tx.Model(&events.Event{}).
			Where("id = ?", booking.EventID).
			Update("booked_count", newBookedCount).Error
		if err != nil {
			return fmt.Errorf("failed to update event booked count: %w", err)
		}

		return nil
	})
}

func (r *repository) CheckEventCapacity(ctx context.Context, eventID uuid.UUID, requestedQuantity int) (bool, error) {
	var event struct {
		TotalCapacity int    `gorm:"column:total_capacity"`
		BookedCount   int    `gorm:"column:booked_count"`
		Status        string `gorm:"column:status"`
	}

	err := r.db.WithContext(ctx).
		Table("events").
		Select("total_capacity, booked_count, status").
		Where("id = ?", eventID).
		First(&event).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, errors.New("event not found")
		}
		return false, err
	}

	if event.Status != "published" {
		return false, errors.New("event is not available for booking")
	}

	availableCapacity := event.TotalCapacity - event.BookedCount
	return availableCapacity >= requestedQuantity, nil
}

func (r *repository) GetEventBookedCount(ctx context.Context, eventID uuid.UUID) (int, error) {
	var bookedCount int
	err := r.db.WithContext(ctx).
		Model(&Booking{}).
		Where("event_id = ?", eventID).
		Where("status = ?", StatusConfirmed). // Only count confirmed bookings
		Select("COALESCE(SUM(quantity), 0)").
		Scan(&bookedCount).Error

	return bookedCount, err
}

// applyFilters applies query filters to the GORM query
func (r *repository) applyFilters(query *gorm.DB, filters BookingListQuery) *gorm.DB {
	// Filter by status
	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}

	// Filter by event ID
	if filters.EventID != "" {
		if eventID, err := uuid.Parse(filters.EventID); err == nil {
			query = query.Where("event_id = ?", eventID)
		}
	}

	// Filter by date range
	if filters.DateFrom != "" {
		if dateFrom, err := time.Parse("2006-01-02", filters.DateFrom); err == nil {
			query = query.Where("created_at >= ?", dateFrom)
		}
	}

	if filters.DateTo != "" {
		if dateTo, err := time.Parse("2006-01-02", filters.DateTo); err == nil {
			// Add 23:59:59 to include the entire day
			dateTo = dateTo.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			query = query.Where("created_at <= ?", dateTo)
		}
	}

	return query
}

// Helper function to calculate total pages
func CalculateTotalPages(totalCount int64, limit int) int {
	if limit <= 0 {
		return 0
	}
	return int(math.Ceil(float64(totalCount) / float64(limit)))
}
