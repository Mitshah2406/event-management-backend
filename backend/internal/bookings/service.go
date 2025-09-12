package bookings

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SeatService interface for seat-related operations (to avoid circular dependency)
type SeatService interface {
	ValidateHold(ctx context.Context, holdID string, userID string) (*HoldValidationResult, error)
	ReleaseHold(ctx context.Context, holdID string) error
	GetSeatsByHoldID(ctx context.Context, holdID string) ([]SeatInfo, error)
	GetHoldDetails(ctx context.Context, holdID string) (*SeatHoldDetails, error)
	UpdateSeatStatusToBulk(ctx context.Context, seatIDs []uuid.UUID, status string) error
}

// WaitlistService interface for waitlist-related operations (to avoid circular dependency)
type WaitlistService interface {
	GetWaitlistStatusForBooking(ctx context.Context, userID, eventID uuid.UUID) (*WaitlistStatusForBooking, error)
	MarkAsConverted(ctx context.Context, userID, eventID, bookingID uuid.UUID) error
}

// WaitlistStatusForBooking represents waitlist status (simplified for bookings)
type WaitlistStatusForBooking struct {
	Status    string `json:"status"`
	IsExpired bool   `json:"is_expired"`
}

// SeatHoldDetails represents hold details (matching seats service structure)
type SeatHoldDetails struct {
	HoldID  string   `json:"hold_id"`
	UserID  string   `json:"user_id"`
	EventID string   `json:"event_id"`
	SeatIDs []string `json:"seat_ids"`
	TTL     int      `json:"ttl_seconds"`
}

// Service interface defines the contract for booking business logic
type Service interface {
	ConfirmBooking(ctx context.Context, userID uuid.UUID, req BookingConfirmationRequest) (*BookingConfirmationResponse, error)
	GetBooking(ctx context.Context, bookingID uuid.UUID) (*Booking, error)
	GetUserBookings(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Booking, error)
	CancelBooking(ctx context.Context, bookingID uuid.UUID, userID uuid.UUID) error
	CancelBookingInternal(ctx context.Context, bookingID uuid.UUID) error

	// Payment operations
	ProcessPayment(ctx context.Context, bookingID uuid.UUID, amount float64, method string) (*PaymentInfo, error)
}

// service implements the Service interface
type service struct {
	repo            Repository
	seatService     SeatService
	waitlistService WaitlistService
}

// HoldValidationResult represents the result of hold validation
type HoldValidationResult struct {
	Valid     bool       `json:"valid"`
	HoldID    string     `json:"hold_id"`
	UserID    string     `json:"user_id"`
	ExpiresAt time.Time  `json:"expires_at"`
	Seats     []SeatInfo `json:"seats"`
}

// SeatInfo represents seat information
type SeatInfo struct {
	ID          uuid.UUID `json:"id"`
	SectionID   uuid.UUID `json:"section_id"`
	SeatNumber  string    `json:"seat_number"`
	Row         string    `json:"row"`
	Price       float64   `json:"price"`
	SectionName string    `json:"section_name"`
}

// NewService creates a new booking service instance
func NewService(repo Repository, seatService SeatService, waitlistService WaitlistService) Service {
	return &service{
		repo:            repo,
		seatService:     seatService,
		waitlistService: waitlistService,
	}
}

// ConfirmBooking processes a booking confirmation
func (s *service) ConfirmBooking(ctx context.Context, userID uuid.UUID, req BookingConfirmationRequest) (*BookingConfirmationResponse, error) {
	// Step 1: Validate the hold
	holdValidation, err := s.seatService.ValidateHold(ctx, req.HoldID, userID.String())
	if err != nil {
		return nil, fmt.Errorf("hold validation failed: %w", err)
	}

	if !holdValidation.Valid {
		return nil, fmt.Errorf("hold is invalid or expired")
	}

	// Step 1.5: Get hold details to validate event ID
	holdDetails, err := s.seatService.GetHoldDetails(ctx, req.HoldID)
	if err != nil {
		return nil, fmt.Errorf("failed to get hold details: %w", err)
	}

	// Verify that the event ID in request matches the hold's event ID
	if holdDetails.EventID != req.EventID {
		return nil, fmt.Errorf("event ID mismatch: hold is for event %s but request is for event %s",
			holdDetails.EventID, req.EventID)
	}

	// Step 1.7: Validate waitlist eligibility (if applicable)
	eventIDForWaitlist, err := uuid.Parse(req.EventID)
	if err != nil {
		return nil, fmt.Errorf("invalid event ID format: %w", err)
	}

	if s.waitlistService != nil {
		waitlistStatus, err := s.waitlistService.GetWaitlistStatusForBooking(ctx, userID, eventIDForWaitlist)
		if err == nil && waitlistStatus != nil {
			// User is/was on waitlist - validate their booking eligibility
			if waitlistStatus.Status == "NOTIFIED" {
				// Check if still within booking window (not expired)
				if waitlistStatus.IsExpired {
					return nil, fmt.Errorf("waitlist booking window has expired - you have been moved back to the queue")
				}
			} else if waitlistStatus.Status == "ACTIVE" {
				// User is on waitlist but not notified yet
				return nil, fmt.Errorf("you are still on the waitlist and have not been notified yet")
			} else if waitlistStatus.Status == "EXPIRED" {
				// User's previous notification expired
				return nil, fmt.Errorf("your previous booking opportunity has expired - you have been moved back to the queue")
			}
			// If status is CONVERTED, allow booking (user already successfully converted)
		}
		// If no waitlist entry found, user can book normally (not from waitlist)
	}

	// Step 2: Get seat information for pricing
	seats, err := s.seatService.GetSeatsByHoldID(ctx, req.HoldID)
	if err != nil {
		return nil, fmt.Errorf("failed to get seats for hold: %w", err)
	}

	if len(seats) == 0 {
		return nil, fmt.Errorf("no seats found for hold")
	}

	// Step 3: Calculate total amount
	var totalAmount float64
	var seatBookings []SeatBooking
	var bookedSeats []BookedSeatInfo

	for _, seat := range seats {
		totalAmount += seat.Price

		seatBooking := SeatBooking{
			SeatID:    seat.ID,
			SectionID: seat.SectionID,
			SeatPrice: seat.Price,
		}
		seatBookings = append(seatBookings, seatBooking)

		bookedSeat := BookedSeatInfo{
			SeatID:      seat.ID.String(),
			SectionID:   seat.SectionID.String(),
			SeatNumber:  seat.SeatNumber,
			Row:         seat.Row,
			SectionName: seat.SectionName,
			Price:       seat.Price,
		}
		bookedSeats = append(bookedSeats, bookedSeat)
	}

	// Step 4: Generate booking reference
	bookingRef, err := s.generateBookingReference()
	if err != nil {
		return nil, fmt.Errorf("failed to generate booking reference: %w", err)
	}

	// Step 5: Parse event ID and create booking record
	eventUUID, err := uuid.Parse(req.EventID)
	if err != nil {
		return nil, fmt.Errorf("invalid event ID: %w", err)
	}

	booking := &Booking{
		UserID:       userID,
		EventID:      eventUUID, // Use the correct event ID from request
		TotalSeats:   len(seats),
		TotalPrice:   totalAmount,
		Status:       "CONFIRMED",
		BookingRef:   bookingRef,
		SeatBookings: seatBookings,
	}

	// Step 6: Generate transaction ID for payment
	transactionID := s.generateTransactionID()

	// Step 7: Create payment record with transaction ID
	payment := &Payment{
		Amount:        totalAmount,
		Currency:      "USD",
		Status:        "PENDING",
		PaymentMethod: req.PaymentMethod,
		TransactionID: transactionID,
	}
	booking.Payments = []Payment{*payment}

	// Step 8: Process in transaction (create booking, seat bookings, and payment)
	if err := s.repo.Create(ctx, booking); err != nil {
		return nil, fmt.Errorf("failed to create booking: %w", err)
	}

	// Note: No need to update seat status to BOOKED since:
	// 1. Seat status constraint only allows AVAILABLE/BLOCKED
	// 2. Booking status is tracked via seat_bookings table
	// 3. GetEffectiveStatus() method handles event-specific booking logic

	// Step 9: Process mock payment (update the existing payment to completed)
	paymentInfo, err := s.ProcessPayment(ctx, booking.ID, totalAmount, req.PaymentMethod)
	if err != nil {
		return nil, fmt.Errorf("payment processing failed: %w", err)
	}

	// Step 10: Mark waitlist as converted (if booking was from waitlist)
	if s.waitlistService != nil {
		if err := s.waitlistService.MarkAsConverted(ctx, userID, eventIDForWaitlist, booking.ID); err != nil {
			// Log warning but don't fail the booking since payment is processed
			fmt.Printf("Warning: Failed to mark waitlist as converted for user %s, booking %s: %v\n",
				userID, booking.ID, err)
		}
	}

	// Step 11: Release Redis hold
	if err := s.seatService.ReleaseHold(ctx, req.HoldID); err != nil {
		// Log error but don't fail the booking since payment is processed
		fmt.Printf("Warning: Failed to release hold %s: %v\n", req.HoldID, err)
	}

	// Step 12: Return confirmation response
	response := &BookingConfirmationResponse{
		BookingID:  booking.ID.String(),
		BookingRef: booking.BookingRef,
		Status:     booking.Status,
		TotalPrice: booking.TotalPrice,
		TotalSeats: booking.TotalSeats,
		Seats:      bookedSeats,
		Payment:    *paymentInfo,
		CreatedAt:  booking.CreatedAt,
	}

	return response, nil
}

// GetBooking retrieves a booking by ID
func (s *service) GetBooking(ctx context.Context, bookingID uuid.UUID) (*Booking, error) {
	return s.repo.GetByID(ctx, bookingID)
}

// GetUserBookings retrieves bookings for a specific user
func (s *service) GetUserBookings(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Booking, error) {
	return s.repo.GetByUserID(ctx, userID, limit, offset)
}

// CancelBooking cancels a booking and releases the seats
func (s *service) CancelBooking(ctx context.Context, bookingID uuid.UUID, userID uuid.UUID) error {
	// Get the booking
	booking, err := s.repo.GetByID(ctx, bookingID)
	if err != nil {
		return fmt.Errorf("failed to get booking: %w", err)
	}

	// Verify ownership
	if booking.UserID != userID {
		return fmt.Errorf("unauthorized: booking does not belong to user")
	}

	// Check if already cancelled
	if booking.IsCancelled() {
		return fmt.Errorf("booking is already cancelled")
	}

	// Cancel the booking
	if err := s.repo.Cancel(ctx, bookingID); err != nil {
		return fmt.Errorf("failed to cancel booking: %w", err)
	}

	// Seats are automatically released when booking is cancelled
	// No need to update seat status as booking records handle the "booked" state

	return nil
}

// CancelBookingInternal cancels a booking without user verification (for internal use by cancellation service)
func (s *service) CancelBookingInternal(ctx context.Context, bookingID uuid.UUID) error {
	// Get the booking
	booking, err := s.repo.GetByID(ctx, bookingID)
	if err != nil {
		return fmt.Errorf("failed to get booking: %w", err)
	}

	// Check if already cancelled
	if booking.IsCancelled() {
		return fmt.Errorf("booking is already cancelled")
	}

	// Cancel the booking
	if err := s.repo.Cancel(ctx, bookingID); err != nil {
		return fmt.Errorf("failed to cancel booking: %w", err)
	}

	// Seats are automatically released when booking is cancelled
	// The seat_bookings table tracks which seats are booked for which booking
	// When booking status becomes CANCELLED, those seats become available again

	return nil
}

// ProcessPayment processes a mock payment
func (s *service) ProcessPayment(ctx context.Context, bookingID uuid.UUID, amount float64, method string) (*PaymentInfo, error) {
	// Get the existing payment record from the booking
	booking, err := s.repo.GetByID(ctx, bookingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get booking: %w", err)
	}

	if len(booking.Payments) == 0 {
		return nil, fmt.Errorf("no payment record found for booking")
	}

	// Update the existing payment to completed status
	payment := &booking.Payments[0]
	now := time.Now()
	payment.Status = "COMPLETED"
	payment.ProcessedAt = &now
	payment.UpdatedAt = now

	if err := s.repo.UpdatePayment(ctx, payment); err != nil {
		return nil, fmt.Errorf("failed to update payment record: %w", err)
	}

	return &PaymentInfo{
		ID:            payment.ID.String(),
		Amount:        payment.Amount,
		Currency:      payment.Currency,
		Status:        payment.Status,
		PaymentMethod: payment.PaymentMethod,
		TransactionID: payment.TransactionID,
		ProcessedAt:   payment.ProcessedAt,
	}, nil
}

// generateBookingReference generates a unique booking reference
func (s *service) generateBookingReference() (string, error) {
	timestamp := time.Now().Format("20060102")

	// Generate 6 random uppercase letters
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	randomPart := make([]byte, 6)

	for i := range randomPart {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		randomPart[i] = letters[num.Int64()]
	}

	return fmt.Sprintf("EVT-%s-%s", timestamp, string(randomPart)), nil
}

// generateTransactionID generates a mock transaction ID
func (s *service) generateTransactionID() string {
	timestamp := time.Now().Unix()
	uuid := uuid.New().String()
	shortUUID := strings.ReplaceAll(uuid, "-", "")[:8]
	return fmt.Sprintf("TXN_%d_%s", timestamp, strings.ToUpper(shortUUID))
}
