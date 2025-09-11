package cancellation

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service interface defines the contract for cancellation business logic
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

// BookingService interface for booking-related operations (to avoid circular dependency)
type BookingService interface {
	GetBooking(ctx context.Context, bookingID uuid.UUID) (BookingInfo, error)
	CancelBookingInternal(ctx context.Context, bookingID uuid.UUID) error
}

// BookingInfo represents booking information for cancellation calculations
type BookingInfo struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	EventID    uuid.UUID `json:"event_id"`
	TotalPrice float64   `json:"total_price"`
	Status     string    `json:"status"`
	BookingRef string    `json:"booking_ref"`
	CreatedAt  time.Time `json:"created_at"`
}

// CancellationPolicyRequest represents a request to create/update cancellation policy
type CancellationPolicyRequest struct {
	AllowCancellation    bool      `json:"allow_cancellation" binding:"required"`
	CancellationDeadline time.Time `json:"cancellation_deadline" binding:"required"`
	FeeType              string    `json:"fee_type" binding:"required,oneof=NONE FIXED PERCENTAGE"`
	FeeAmount            float64   `json:"fee_amount"`
	RefundProcessingDays int       `json:"refund_processing_days" binding:"min=1,max=30"`
}

// CancellationRequest represents a request to cancel a booking
type CancellationRequest struct {
	Reason string `json:"reason" binding:"required,min=10,max=500"`
}

// service implements the Service interface
type service struct {
	repo           Repository
	bookingService BookingService
}

// NewService creates a new cancellation service instance
func NewService(repo Repository, bookingService BookingService) Service {
	return &service{
		repo:           repo,
		bookingService: bookingService,
	}
}

// CreateCancellationPolicy creates a new cancellation policy for an event
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

// GetCancellationPolicy retrieves a cancellation policy by event ID
func (s *service) GetCancellationPolicy(ctx context.Context, eventID uuid.UUID) (*CancellationPolicy, error) {
	return s.repo.GetCancellationPolicyByEventID(ctx, eventID)
}

// UpdateCancellationPolicy updates an existing cancellation policy
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

// RequestCancellation processes a cancellation request
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

	// Update booking status to CANCELLED and free up seats
	if err := s.bookingService.CancelBookingInternal(ctx, bookingID); err != nil {
		// Log error but don't fail the cancellation since refund record is already created
		// TODO: Add proper logging here
		return cancellation, fmt.Errorf("cancellation created but failed to update booking status: %w", err)
	}

	return cancellation, nil
}

// GetCancellation retrieves a cancellation by ID
func (s *service) GetCancellation(ctx context.Context, cancellationID uuid.UUID) (*Cancellation, error) {
	return s.repo.GetCancellationByID(ctx, cancellationID)
}

// GetUserCancellations retrieves all cancellations for a user
func (s *service) GetUserCancellations(ctx context.Context, userID uuid.UUID) ([]Cancellation, error) {
	return s.repo.GetCancellationsByUserID(ctx, userID)
}

// CalculateCancellationFee calculates the cancellation fee and refund amount
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

// ValidateCancellationEligibility checks if a booking can be cancelled
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

// validatePolicyRequest validates a cancellation policy request
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
