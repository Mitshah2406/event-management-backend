package bookings

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EventService interface to interact with events (avoid circular dependency)
type EventService interface {
	CheckEventAvailability(eventID uuid.UUID, ticketCount int) (bool, error)
	IsEventInFuture(eventID uuid.UUID) (bool, error)
}

type Service interface {
	// Core booking operations
	CreateBooking(ctx context.Context, userID uuid.UUID, req CreateBookingRequest) (*BookingResponse, error)
	CancelBooking(ctx context.Context, userID uuid.UUID, bookingID uuid.UUID) error
	GetBookingDetails(ctx context.Context, userID uuid.UUID, bookingID uuid.UUID) (*BookingResponse, error)

	// User operations
	GetUserBookings(ctx context.Context, userID uuid.UUID, query BookingListQuery) (*PaginatedBookings, error)

	// Admin operations
	GetAllBookings(ctx context.Context, query BookingListQuery) (*PaginatedBookings, error)
	GetEventBookings(ctx context.Context, eventID uuid.UUID) ([]BookingResponse, error)
	CancelBookingAsAdmin(ctx context.Context, adminID uuid.UUID, bookingID uuid.UUID) error
	GetBookingDetailsAsAdmin(ctx context.Context, bookingID uuid.UUID) (*BookingResponse, error)

	// Utility methods
	ValidateBookingRequest(ctx context.Context, userID uuid.UUID, req CreateBookingRequest) error

	// Dependency injection
	SetEventService(eventService EventService)
}

type service struct {
	repo         Repository
	eventService EventService
}

func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

// SetEventService allows injection of event service to avoid circular dependency
func (s *service) SetEventService(eventService EventService) {
	s.eventService = eventService
}

func (s *service) CreateBooking(ctx context.Context, userID uuid.UUID, req CreateBookingRequest) (*BookingResponse, error) {
	// 1. Validate the request
	if err := s.ValidateBookingRequest(ctx, userID, req); err != nil {
		log.Println("Validation error:", err)
		return nil, err
	}
	log.Print("Creating booking for user:", userID, " event:", req.EventID, " quantity:", req.Quantity)
	// 2. Parse event ID
	eventID, err := uuid.Parse(req.EventID)
	if err != nil {
		return nil, errors.New("invalid event ID format")
	}

	// 3. Additional validation using event service if available
	if s.eventService != nil {
		available, err := s.eventService.CheckEventAvailability(eventID, req.Quantity)
		if err != nil {
			return nil, fmt.Errorf("failed to check event availability: %w", err)
		}
		if !available {
			return nil, errors.New("event is not available for the requested quantity")
		}
	}

	// 4. Create booking object - goes directly to CONFIRMED (immediate booking)
	booking := &Booking{
		UserID:   userID,
		EventID:  eventID,
		Quantity: req.Quantity,
		Status:   StatusConfirmed, // Direct confirmation for better UX
	}

	// 5. Create booking with atomic capacity check
	err = s.repo.CreateBookingWithCapacityCheck(ctx, booking)
	if err != nil {
		return nil, err
	}

	// 6. Get the created booking with relations
	createdBooking, err := s.repo.GetBookingByIDWithRelations(ctx, booking.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve created booking: %w", err)
	}

	response := createdBooking.ToResponse()
	return &response, nil
}

func (s *service) CancelBooking(ctx context.Context, userID uuid.UUID, bookingID uuid.UUID) error {
	// 1. Get the booking to verify ownership and current status
	booking, err := s.repo.GetBookingByID(ctx, bookingID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("booking not found")
		}
		return fmt.Errorf("failed to get booking: %w", err)
	}

	// 2. Verify ownership
	if booking.UserID != userID {
		return errors.New("unauthorized: you can only cancel your own bookings")
	}

	// 3. Check if booking can be cancelled
	if booking.Status == StatusCancelled {
		return errors.New("booking is already cancelled")
	}

	// 4. Update booking status
	cancelledAt := time.Now()
	err = s.repo.UpdateBookingStatus(ctx, bookingID, StatusCancelled, &cancelledAt)
	if err != nil {
		return fmt.Errorf("failed to cancel booking: %w", err)
	}

	// TODO: In a real system, you might want to:
	// - Decrement the event's booked_count
	// - Send cancellation notifications
	// - Handle refunds if payment was processed

	return nil
}

func (s *service) GetBookingDetails(ctx context.Context, userID uuid.UUID, bookingID uuid.UUID) (*BookingResponse, error) {
	booking, err := s.repo.GetBookingByIDWithRelations(ctx, bookingID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("booking not found")
		}
		return nil, fmt.Errorf("failed to get booking: %w", err)
	}

	// Verify ownership
	if booking.UserID != userID {
		return nil, errors.New("unauthorized: you can only view your own bookings")
	}

	response := booking.ToResponse()
	return &response, nil
}

func (s *service) GetUserBookings(ctx context.Context, userID uuid.UUID, query BookingListQuery) (*PaginatedBookings, error) {
	bookings, totalCount, err := s.repo.GetUserBookings(ctx, userID, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get user bookings: %w", err)
	}

	// Convert to response format
	bookingResponses := make([]BookingResponse, len(bookings))
	for i, booking := range bookings {
		bookingResponses[i] = booking.ToResponse()
	}

	// Set defaults for pagination
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Limit <= 0 {
		query.Limit = 10
	}

	totalPages := CalculateTotalPages(totalCount, query.Limit)

	return &PaginatedBookings{
		Bookings:   bookingResponses,
		TotalCount: totalCount,
		Page:       query.Page,
		Limit:      query.Limit,
		TotalPages: totalPages,
	}, nil
}

// Admin operations

func (s *service) GetAllBookings(ctx context.Context, query BookingListQuery) (*PaginatedBookings, error) {
	bookings, totalCount, err := s.repo.GetAllBookings(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all bookings: %w", err)
	}

	// Convert to response format
	bookingResponses := make([]BookingResponse, len(bookings))
	for i, booking := range bookings {
		bookingResponses[i] = booking.ToResponse()
	}

	// Set defaults for pagination
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Limit <= 0 {
		query.Limit = 10
	}

	totalPages := CalculateTotalPages(totalCount, query.Limit)

	return &PaginatedBookings{
		Bookings:   bookingResponses,
		TotalCount: totalCount,
		Page:       query.Page,
		Limit:      query.Limit,
		TotalPages: totalPages,
	}, nil
}

func (s *service) GetEventBookings(ctx context.Context, eventID uuid.UUID) ([]BookingResponse, error) {
	bookings, err := s.repo.GetBookingsByEventID(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get event bookings: %w", err)
	}

	// Convert to response format
	bookingResponses := make([]BookingResponse, len(bookings))
	for i, booking := range bookings {
		bookingResponses[i] = booking.ToResponse()
	}

	return bookingResponses, nil
}

func (s *service) CancelBookingAsAdmin(ctx context.Context, adminID uuid.UUID, bookingID uuid.UUID) error {
	// Admin can cancel any booking without ownership check
	booking, err := s.repo.GetBookingByID(ctx, bookingID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("booking not found")
		}
		return fmt.Errorf("failed to get booking: %w", err)
	}

	// Check if booking can be cancelled
	if booking.Status == StatusCancelled {
		return errors.New("booking is already cancelled")
	}

	// Update booking status
	cancelledAt := time.Now()
	err = s.repo.UpdateBookingStatus(ctx, bookingID, StatusCancelled, &cancelledAt)
	if err != nil {
		return fmt.Errorf("failed to cancel booking: %w", err)
	}

	return nil
}

func (s *service) GetBookingDetailsAsAdmin(ctx context.Context, bookingID uuid.UUID) (*BookingResponse, error) {
	// Admin can view any booking without ownership check
	booking, err := s.repo.GetBookingByIDWithRelations(ctx, bookingID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("booking not found")
		}
		return nil, fmt.Errorf("failed to get booking: %w", err)
	}

	response := booking.ToResponse()
	return &response, nil
}

func (s *service) ValidateBookingRequest(ctx context.Context, userID uuid.UUID, req CreateBookingRequest) error {
	// Basic validation
	if req.Quantity <= 0 {
		return errors.New("quantity must be greater than 0")
	}

	if req.Quantity > 10 {
		return errors.New("cannot book more than 10 tickets at once")
	}

	// Validate event ID format
	eventID, err := uuid.Parse(req.EventID)
	if err != nil {
		return errors.New("invalid event ID format")
	}

	// Check if event date is in the future
	if s.eventService != nil {
		isInFuture, err := s.eventService.IsEventInFuture(eventID)
		log.Println("Event is in future:", isInFuture, " for event:", eventID)
		if err != nil {
			return fmt.Errorf("failed to check event date: %w", err)
		}

		if !isInFuture {
			return errors.New("cannot book tickets for past events")
		}
	}

	return nil
}
