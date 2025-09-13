package cancellation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Service interface {
	// Cancellation Policy management
	CreateCancellationPolicy(ctx context.Context, eventID uuid.UUID, req CancellationPolicyRequest) (*CancellationPolicy, error)
	GetCancellationPolicy(ctx context.Context, eventID uuid.UUID) (*CancellationPolicy, error)
	UpdateCancellationPolicy(ctx context.Context, eventID uuid.UUID, req CancellationPolicyRequest) (*CancellationPolicy, error)

	// Cancellation management
	RequestCancellation(ctx context.Context, bookingID uuid.UUID, userID uuid.UUID, req CancellationRequest) (*Cancellation, error)
	GetCancellation(ctx context.Context, cancellationID uuid.UUID) (*Cancellation, error)
	GetUserCancellations(ctx context.Context, userID uuid.UUID) ([]Cancellation, error)

	// Business logic helpers
	CalculateCancellationFee(ctx context.Context, bookingID uuid.UUID) (float64, float64, error) // fee, refund
	ValidateCancellationEligibility(ctx context.Context, bookingID uuid.UUID) error
}

type BookingService interface {
	GetBooking(ctx context.Context, bookingID uuid.UUID) (BookingInfo, error)
	CancelBookingInternal(ctx context.Context, bookingID uuid.UUID) error
	CancelBookingWithVersion(ctx context.Context, bookingID uuid.UUID, expectedVersion int) error
}

type WaitlistService interface {
	ProcessCancellation(ctx context.Context, eventID uuid.UUID, freedTickets int) error
}

type BookingInfo struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	EventID    uuid.UUID `json:"event_id"`
	TotalPrice float64   `json:"total_price"`
	TotalSeats int       `json:"total_seats"`
	Status     string    `json:"status"`
	BookingRef string    `json:"booking_ref"`
	Version    int       `json:"version"`
	CreatedAt  time.Time `json:"created_at"`
}

type CancellationPolicyRequest struct {
	AllowCancellation    bool      `json:"allow_cancellation" binding:"required"`
	CancellationDeadline time.Time `json:"cancellation_deadline" binding:"required"`
	FeeType              string    `json:"fee_type" binding:"required,oneof=NONE FIXED PERCENTAGE"`
	FeeAmount            float64   `json:"fee_amount"`
	RefundProcessingDays int       `json:"refund_processing_days" binding:"min=1,max=30"`
}

type CancellationRequest struct {
	Reason string `json:"reason" binding:"required,min=1,max=500"`
}

type service struct {
	repo            Repository
	bookingService  BookingService
	waitlistService WaitlistService
}

func NewService(repo Repository, bookingService BookingService, waitlistService WaitlistService) Service {
	return &service{
		repo:            repo,
		bookingService:  bookingService,
		waitlistService: waitlistService,
	}
}

func (s *service) CreateCancellationPolicy(ctx context.Context, eventID uuid.UUID, req CancellationPolicyRequest) (*CancellationPolicy, error) {
	// Check if policy already exists
	_, err := s.repo.GetCancellationPolicyByEventID(ctx, eventID)
	if err == nil {
		return nil, fmt.Errorf("cancellation policy already exists for this event")
	}

	// Validate request
	if err := s.validatePolicyRequest(req); err != nil {
		return nil, fmt.Errorf("invalid policy request: %w", err)
	}

	// Create policy
	policy := &CancellationPolicy{
		EventID:              eventID,
		AllowCancellation:    req.AllowCancellation,
		CancellationDeadline: req.CancellationDeadline,
		FeeType:              req.FeeType,
		FeeAmount:            req.FeeAmount,
		RefundProcessingDays: req.RefundProcessingDays,
	}

	if err := s.repo.CreateCancellationPolicy(ctx, policy); err != nil {
		return nil, fmt.Errorf("failed to create cancellation policy: %w", err)
	}

	return policy, nil
}

func (s *service) GetCancellationPolicy(ctx context.Context, eventID uuid.UUID) (*CancellationPolicy, error) {
	return s.repo.GetCancellationPolicyByEventID(ctx, eventID)
}

func (s *service) UpdateCancellationPolicy(ctx context.Context, eventID uuid.UUID, req CancellationPolicyRequest) (*CancellationPolicy, error) {
	// Get existing policy
	policy, err := s.repo.GetCancellationPolicyByEventID(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("cancellation policy not found: %w", err)
	}

	// Validate request
	if err := s.validatePolicyRequest(req); err != nil {
		return nil, fmt.Errorf("invalid policy request: %w", err)
	}

	// Update policy
	policy.AllowCancellation = req.AllowCancellation
	policy.CancellationDeadline = req.CancellationDeadline
	policy.FeeType = req.FeeType
	policy.FeeAmount = req.FeeAmount
	policy.RefundProcessingDays = req.RefundProcessingDays
	policy.UpdatedAt = time.Now()

	if err := s.repo.UpdateCancellationPolicy(ctx, policy); err != nil {
		return nil, fmt.Errorf("failed to update cancellation policy: %w", err)
	}

	return policy, nil
}

func (s *service) RequestCancellation(ctx context.Context, bookingID uuid.UUID, userID uuid.UUID, req CancellationRequest) (*Cancellation, error) {
	// Get booking information
	booking, err := s.bookingService.GetBooking(ctx, bookingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get booking: %w", err)
	}

	// Verify ownership
	if booking.UserID != userID {
		return nil, fmt.Errorf("unauthorized: booking does not belong to user")
	}

	// Validate cancellation eligibility
	if err := s.ValidateCancellationEligibility(ctx, bookingID); err != nil {
		return nil, fmt.Errorf("cancellation not allowed: %w", err)
	}

	// Check if cancellation already exists
	_, err = s.repo.GetCancellationByBookingID(ctx, bookingID)
	if err == nil {
		return nil, fmt.Errorf("cancellation request already exists for this booking")
	}

	// Calculate cancellation fee and refund amount
	cancellationFee, refundAmount, err := s.CalculateCancellationFee(ctx, bookingID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate cancellation fee: %w", err)
	}

	// Create cancellation record with instant approval
	now := time.Now()
	cancellation := &Cancellation{
		BookingID:       bookingID,
		RequestedAt:     now,
		ProcessedAt:     &now, // Process immediately
		CancellationFee: cancellationFee,
		RefundAmount:    refundAmount,
		Reason:          req.Reason,
		Status:          "PROCESSED", // Auto-approve and process instantly
	}

	if err := s.repo.CreateCancellation(ctx, cancellation); err != nil {
		return nil, fmt.Errorf("failed to create cancellation: %w", err)
	}

	// Update booking status to CANCELLED with version check
	if err := s.bookingService.CancelBookingWithVersion(ctx, bookingID, booking.Version); err != nil {
		// If version mismatch, provide a user-friendly message
		if strings.Contains(err.Error(), "version mismatch") || strings.Contains(err.Error(), "modified by another process") {
			return nil, fmt.Errorf("booking was recently modified, please refresh and try again")
		}
		return cancellation, fmt.Errorf("cancellation created but failed to update booking status: %w", err)
	}

	// Notify waitlist users about freed seats (run in background to avoid blocking)
	go func() {
		if s.waitlistService != nil {
			// Log the notification attempt
			fmt.Printf("🔔 NOTIFICATION DISPATCH: Starting waitlist notification for booking %s (event: %s, seats: %d)\n",
				bookingID, booking.EventID, booking.TotalSeats)

			if err := s.waitlistService.ProcessCancellation(context.Background(), booking.EventID, booking.TotalSeats); err != nil {
				fmt.Printf("❌ NOTIFICATION FAILED: Event %s - Error: %v\n", booking.EventID, err)
			} else {
				fmt.Printf("✅ NOTIFICATION SUCCESS: Event %s - %d seats freed and waitlist notified\n", booking.EventID, booking.TotalSeats)
			}
		} else {
			fmt.Printf("⚠️  NOTIFICATION SKIPPED: Waitlist service not available for booking %s\n", bookingID)
		}
	}()

	return cancellation, nil
}

func (s *service) GetCancellation(ctx context.Context, cancellationID uuid.UUID) (*Cancellation, error) {
	return s.repo.GetCancellationByID(ctx, cancellationID)
}

func (s *service) GetUserCancellations(ctx context.Context, userID uuid.UUID) ([]Cancellation, error) {
	return s.repo.GetCancellationsByUserID(ctx, userID)
}

func (s *service) CalculateCancellationFee(ctx context.Context, bookingID uuid.UUID) (float64, float64, error) {
	// Get booking information
	booking, err := s.bookingService.GetBooking(ctx, bookingID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get booking: %w", err)
	}

	// Get cancellation policy
	policy, err := s.repo.GetCancellationPolicyByEventID(ctx, booking.EventID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get cancellation policy: %w", err)
	}

	var cancellationFee float64
	totalPrice := booking.TotalPrice

	// Calculate fee based on policy
	switch policy.FeeType {
	case "NONE":
		cancellationFee = 0
	case "FIXED":
		cancellationFee = policy.FeeAmount
	case "PERCENTAGE":
		cancellationFee = totalPrice * (policy.FeeAmount / 100)
	default:
		return 0, 0, fmt.Errorf("invalid fee type: %s", policy.FeeType)
	}

	// Ensure fee doesn't exceed total price
	if cancellationFee > totalPrice {
		cancellationFee = totalPrice
	}

	refundAmount := totalPrice - cancellationFee

	return cancellationFee, refundAmount, nil
}

func (s *service) ValidateCancellationEligibility(ctx context.Context, bookingID uuid.UUID) error {
	// Get booking information
	booking, err := s.bookingService.GetBooking(ctx, bookingID)
	if err != nil {
		return fmt.Errorf("failed to get booking: %w", err)
	}

	// Check if booking is already cancelled
	if booking.Status == "CANCELLED" {
		return fmt.Errorf("booking is already cancelled")
	}

	// Get cancellation policy
	policy, err := s.repo.GetCancellationPolicyByEventID(ctx, booking.EventID)
	if err != nil {
		return fmt.Errorf("no cancellation policy found for this event")
	}

	// Check if cancellation is allowed
	if !policy.AllowCancellation {
		return fmt.Errorf("cancellation is not allowed for this event")
	}

	// Check if within cancellation deadline
	if time.Now().After(policy.CancellationDeadline) {
		return fmt.Errorf("cancellation deadline has passed")
	}

	return nil
}

func (s *service) validatePolicyRequest(req CancellationPolicyRequest) error {
	if req.FeeType == "FIXED" && req.FeeAmount <= 0 {
		return fmt.Errorf("fixed fee amount must be greater than 0")
	}

	if req.FeeType == "PERCENTAGE" && (req.FeeAmount < 0 || req.FeeAmount > 100) {
		return fmt.Errorf("percentage fee must be between 0 and 100")
	}

	if req.CancellationDeadline.Before(time.Now()) {
		return fmt.Errorf("cancellation deadline must be in the future")
	}

	return nil
}
