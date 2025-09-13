package seats

import (
	"context"
	"errors"
	"fmt"
	"time"

	"evently/internal/shared/config"
	"evently/internal/shared/utils/constants"
	"evently/pkg/cache"
	"evently/pkg/logger"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service interface {
	// Seat Management
	GetSeatsBySectionID(ctx context.Context, sectionID string) ([]Seat, error)
	GetSeatByID(ctx context.Context, id string) (*Seat, error)
	UpdateSeat(ctx context.Context, id string, req UpdateSeatRequest) (*Seat, error)
	DeleteSeat(ctx context.Context, id string) error

	// Seat Holding (Core Flow)
	HoldSeats(ctx context.Context, req SeatHoldRequest) (*SeatHoldResponse, error)
	ReleaseHold(ctx context.Context, holdID string) error
	ValidateHold(ctx context.Context, holdID string, userID string) (*HoldValidationResult, error)
	GetUserHolds(ctx context.Context, userID string) ([]SeatHoldDetails, error)

	// Availability Checks
	CheckSeatAvailability(ctx context.Context, seatIDs []string) (*SeatAvailabilityResponse, error)
	GetAvailableSeatsInSection(ctx context.Context, sectionID string) ([]SeatResponse, error)
	GetAvailableSeatsInSectionForEvent(ctx context.Context, sectionID string, eventID string) ([]SeatResponse, error)

	// Additional helper methods
	GetSeatsByHoldID(ctx context.Context, holdID string) ([]SeatInfo, error)
	GetHoldDetails(ctx context.Context, holdID string) (*SeatHoldDetails, error)
}

type service struct {
	repo         Repository
	config       *config.Config
	cacheService cache.Service
}

func NewService(repo Repository, cfg *config.Config) Service {
	return &service{
		repo:   repo,
		config: cfg,
	}
}

func (s *service) SetCacheService(cacheService cache.Service) {
	s.cacheService = cacheService
}

//  SEAT MANAGEMENT

func (s *service) GetSeatsBySectionID(ctx context.Context, sectionID string) ([]Seat, error) {
	sectionUUID, err := uuid.Parse(sectionID)
	if err != nil {
		return nil, fmt.Errorf("invalid section ID: %w", err)
	}

	return s.repo.GetSeatsBySectionID(ctx, sectionUUID)
}

func (s *service) GetSeatByID(ctx context.Context, id string) (*Seat, error) {
	seatID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid seat ID: %w", err)
	}

	seat, err := s.repo.GetSeatByID(ctx, seatID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("seat not found")
		}
		return nil, fmt.Errorf("failed to get seat: %w", err)
	}

	return seat, nil
}

func (s *service) UpdateSeat(ctx context.Context, id string, req UpdateSeatRequest) (*Seat, error) {
	seatID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid seat ID: %w", err)
	}

	updates := make(map[string]interface{})
	if req.SeatNumber != nil {
		updates["seat_number"] = *req.SeatNumber
	}
	if req.Row != nil {
		updates["row"] = *req.Row
	}
	if req.Position != nil {
		updates["position"] = *req.Position
	}
	if req.Status != nil {
		validStatuses := map[string]bool{"AVAILABLE": true, "BLOCKED": true}
		if !validStatuses[*req.Status] {
			return nil, fmt.Errorf("invalid seat status: %s", *req.Status)
		}
		updates["status"] = *req.Status
	}

	if len(updates) > 0 {
		if err := s.repo.UpdateSeat(ctx, seatID, updates); err != nil {
			return nil, fmt.Errorf("failed to update seat: %w", err)
		}
	}

	return s.repo.GetSeatByID(ctx, seatID)
}

func (s *service) DeleteSeat(ctx context.Context, id string) error {
	seatID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid seat ID: %w", err)
	}

	// Check if seat exists
	_, err = s.repo.GetSeatByID(ctx, seatID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("seat not found")
		}
		return fmt.Errorf("failed to get seat: %w", err)
	}

	return s.repo.DeleteSeat(ctx, seatID)
}

//  SEAT HOLDING (CORE FLOW)

func (s *service) HoldSeats(ctx context.Context, req SeatHoldRequest) (*SeatHoldResponse, error) {
	// Validate input
	if len(req.SeatIDs) == 0 {
		return nil, fmt.Errorf("no seats specified")
	}
	// Parse seat IDs
	var seatUUIDs []uuid.UUID
	for _, idStr := range req.SeatIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("invalid seat ID: %s", idStr)
		}
		seatUUIDs = append(seatUUIDs, id)
	}

	// Check if seats exist and are available in Postgres (base availability) - checkmate
	availability, err := s.repo.CheckSeatsAvailability(ctx, seatUUIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to check seat availability: %w", err)
	}

	var unavailableSeats []string
	for seatID, available := range availability {
		if !available {
			unavailableSeats = append(unavailableSeats, seatID)
		}
	}

	if len(unavailableSeats) > 0 {
		return nil, fmt.Errorf("seats not available: %v", unavailableSeats)
	}

	// Parse event ID for booking checks
	eventUUID, err := uuid.Parse(req.EventID)
	if err != nil {
		return nil, fmt.Errorf("invalid event ID: %w", err)
	}

	// Check if any of the seats are already booked for this specific event
	bookedSeats, err := s.checkSeatsBookedForEvent(ctx, seatUUIDs, eventUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to check event-specific bookings: %w", err)
	}

	if len(bookedSeats) > 0 {
		return nil, fmt.Errorf("seats already booked for this event: %v", bookedSeats)
	}

	// Check if seats are already held in Redis
	holds, err := s.repo.CheckSeatHolds(ctx, seatUUIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to check seat holds: %w", err)
	}

	var heldSeats []string
	for seatID, holdValue := range holds {
		if holdValue != "" {
			heldSeats = append(heldSeats, seatID)
		}
	}

	if len(heldSeats) > 0 {
		return nil, fmt.Errorf("seats already held: %v", heldSeats)
	}

	// Get seat details for response
	seats, err := s.repo.GetSeatsByIDs(ctx, seatUUIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get seat details: %w", err)
	}

	// Generate hold ID and hold seats in Redis atomically
	holdID := uuid.New().String()
	ttl := s.config.Redis.SeatHoldTTL // Use configurable TTL
	logger.GetDefault().Info("Holding seats with hold ID:", holdID, "for user:", req.UserID, "with TTL:", ttl)
	if err := s.repo.AtomicHoldSeats(ctx, seatUUIDs, req.UserID, holdID, req.EventID, ttl); err != nil {
		return nil, fmt.Errorf("failed to hold seats atomically: %w", err)
	}

	// Build response
	var heldSeatInfo []HeldSeatInfo
	var totalPrice float64

	// Calculate actual seat prices based on event and section
	seatPrices, err := s.calculateSeatPrices(req.EventID, seats)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate seat prices: %w", err)
	}

	for _, seat := range seats {
		seatPrice := seatPrices[seat.ID.String()]

		heldSeatInfo = append(heldSeatInfo, HeldSeatInfo{
			SeatID:      seat.ID.String(),
			SectionID:   seat.SectionID.String(),
			SeatNumber:  seat.SeatNumber,
			Row:         seat.Row,
			SectionName: "", // Will be populated later from section data
			Price:       seatPrice,
		})
		totalPrice += seatPrice
	}

	return &SeatHoldResponse{
		HoldID:     holdID,
		EventID:    req.EventID,
		UserID:     req.UserID,
		Seats:      heldSeatInfo,
		TotalPrice: totalPrice,
		ExpiresAt:  time.Now().Add(ttl),
		TTL:        int(ttl.Seconds()),
	}, nil
}

func (s *service) ReleaseHold(ctx context.Context, holdID string) error {
	// Validate hold exists
	valid, err := s.repo.IsHoldValid(ctx, holdID)
	if err != nil {
		return fmt.Errorf("failed to check hold validity: %w", err)
	}
	if !valid {
		return fmt.Errorf("hold not found or expired")
	}

	return s.repo.ReleaseHold(ctx, holdID)
}

func (s *service) ValidateHold(ctx context.Context, holdID string, userID string) (*HoldValidationResult, error) {
	details, err := s.repo.GetHoldDetails(ctx, holdID)
	if err != nil {
		return &HoldValidationResult{
			Valid:  false,
			Reason: err.Error(),
		}, nil
	}

	if details.UserID != userID {
		return &HoldValidationResult{
			Valid:  false,
			Reason: "hold belongs to different user",
		}, nil
	}

	if details.TTL <= 0 {
		return &HoldValidationResult{
			Valid:  false,
			Reason: "hold has expired",
		}, nil
	}

	return &HoldValidationResult{
		Valid:   true,
		Details: details,
		TTL:     details.TTL,
	}, nil
}

func (s *service) GetUserHolds(ctx context.Context, userID string) ([]SeatHoldDetails, error) {
	holdIDs, err := s.repo.GetUserHolds(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user holds: %w", err)
	}

	var holdDetails []SeatHoldDetails
	for _, holdID := range holdIDs {
		details, err := s.repo.GetHoldDetails(ctx, holdID)
		if err != nil {
			continue // skip invalid holds
		}
		holdDetails = append(holdDetails, *details)
	}

	return holdDetails, nil
}

//  AVAILABILITY CHECKS

func (s *service) CheckSeatAvailability(ctx context.Context, seatIDs []string) (*SeatAvailabilityResponse, error) {
	var seatUUIDs []uuid.UUID
	for _, idStr := range seatIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("invalid seat ID: %s", idStr)
		}
		seatUUIDs = append(seatUUIDs, id)
	}

	// Check Postgres first
	pgAvailability, err := s.repo.CheckSeatsAvailability(ctx, seatUUIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to check postgres availability: %w", err)
	}

	// Check Redis holds also
	redisHolds, err := s.repo.CheckSeatHolds(ctx, seatUUIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to check redis holds: %w", err)
	}

	var availability []SeatAvailabilityInfo
	for _, id := range seatIDs {
		pgAvailable := pgAvailability[id]
		isHeld := redisHolds[id] != ""

		status := "UNAVAILABLE"
		if pgAvailable && !isHeld {
			status = "AVAILABLE"
		} else if isHeld {
			status = "HELD"
		}

		availability = append(availability, SeatAvailabilityInfo{
			SeatID:    id,
			Available: pgAvailable && !isHeld,
			Status:    status,
			HoldInfo:  redisHolds[id],
		})
	}

	return &SeatAvailabilityResponse{
		Seats: availability,
	}, nil
}

func (s *service) GetAvailableSeatsInSection(ctx context.Context, sectionID string) ([]SeatResponse, error) {
	return nil, fmt.Errorf("GetAvailableSeatsInSection is deprecated - use GetAvailableSeatsInSectionForEvent instead")
}

func (s *service) GetAvailableSeatsInSectionForEvent(ctx context.Context, sectionID string, eventID string) ([]SeatResponse, error) {
	logger.GetDefault().Info("Fetching available seats for section:", sectionID, "and event:", eventID)
	sectionUUID, err := uuid.Parse(sectionID)
	if err != nil {
		return nil, fmt.Errorf("invalid section ID: %w", err)
	}
	logger.GetDefault().Debug("getting available seats for section:", sectionID, "and event:", eventID)
	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		return nil, fmt.Errorf("invalid event ID: %w", err)
	}

	cacheKey := constants.BuildSeatAvailabilityKey(sectionID, eventID)
	if s.cacheService != nil {
		var cachedSeats []SeatResponse
		if err := s.cacheService.Get(ctx, cacheKey, &cachedSeats); err == nil {
			logger.GetDefault().Debug("cache hit for seat availability:", cacheKey)
			return cachedSeats, nil
		} else {
			logger.GetDefault().Debug("cache miss for seat availability:", cacheKey)
		}
	}

	// Get all seats in section
	seats, err := s.repo.GetSeatsBySectionID(ctx, sectionUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get seats: %w", err)
	}

	// Get seat bookings for this event
	seatBookings, err := s.getSeatBookingsForEvent(ctx, eventUUID, sectionUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get seat bookings: %w", err)
	}

	// Check Redis holds
	var seatUUIDs []uuid.UUID
	for _, seat := range seats {
		seatUUIDs = append(seatUUIDs, seat.ID)
	}

	holds, err := s.repo.CheckSeatHolds(ctx, seatUUIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to check holds: %w", err)
	}

	var response []SeatResponse
	for _, seat := range seats {
		isHeld := holds[seat.ID.String()] != ""

		// Use the new event-specific status logic
		effectiveStatus := seat.GetEffectiveStatus(eventUUID, seatBookings, isHeld)

		// Only include seats that are effectively available
		if effectiveStatus == "AVAILABLE" {
			response = append(response, SeatResponse{
				ID:         seat.ID.String(),
				SeatNumber: seat.SeatNumber,
				Row:        seat.Row,
				Position:   seat.Position,
				Status:     effectiveStatus,
				Price:      0, // Will be calculated with section multiplier
				IsHeld:     isHeld,
			})
		}
	}

	// Cache the result
	if s.cacheService != nil {
		if err := s.cacheService.Set(ctx, cacheKey, response, constants.TTL_SEATS_AVAILABLE); err != nil {
			logger.GetDefault().Debug("Warning: failed to cache seat availability:", err)
		} else {
			logger.GetDefault().Debug("Cached seat availability:", cacheKey)
		}
	}

	return response, nil
}

// calculates the actual price for each seat based on event pricing
func (s *service) calculateSeatPrices(eventID string, seats []Seat) (map[string]float64, error) {
	prices := make(map[string]float64)

	// Parse event ID
	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		return nil, fmt.Errorf("invalid event ID: %w", err)
	}

	// Get event details to get base price
	// For now, we'll use a simple query. In production, you might want to inject an events service
	var event struct {
		BasePrice float64 `json:"base_price"`
	}

	// Get base price from events table
	if err := s.repo.(*repository).db.Table("events").
		Select("base_price").
		Where("id = ?", eventUUID).
		First(&event).Error; err != nil {
		event.BasePrice = 50.0 // fallback
	}

	// Get event pricing for each unique section
	sectionIDs := make(map[uuid.UUID]bool)
	for _, seat := range seats {
		sectionIDs[seat.SectionID] = true
	}

	// Get pricing multipliers for all sections
	var eventPricing []struct {
		SectionID       uuid.UUID `json:"section_id"`
		PriceMultiplier float64   `json:"price_multiplier"`
	}

	var sectionUUIDs []uuid.UUID
	for sectionID := range sectionIDs {
		sectionUUIDs = append(sectionUUIDs, sectionID)
	}

	if err := s.repo.(*repository).db.Table("event_pricing").
		Select("section_id, price_multiplier").
		Where("event_id = ? AND section_id IN ? AND is_active = true", eventUUID, sectionUUIDs).
		Find(&eventPricing).Error; err != nil {
		// If no pricing found, use base price for all seats
		for _, seat := range seats {
			prices[seat.ID.String()] = event.BasePrice
		}
		return prices, nil
	}

	// Create a map of section ID to price multiplier
	sectionMultipliers := make(map[uuid.UUID]float64)
	for _, pricing := range eventPricing {
		sectionMultipliers[pricing.SectionID] = pricing.PriceMultiplier
	}

	// Calculate price for each seat
	for _, seat := range seats {
		multiplier := sectionMultipliers[seat.SectionID]
		if multiplier == 0 {
			multiplier = 1.0 // Default multiplier if no pricing found
		}

		finalPrice := event.BasePrice * multiplier
		prices[seat.ID.String()] = finalPrice
	}

	return prices, nil
}

// retrieves seats associated with a hold ID
func (s *service) GetSeatsByHoldID(ctx context.Context, holdID string) ([]SeatInfo, error) {
	// Get hold data from Redis
	holdData, err := s.repo.GetHoldDetails(ctx, holdID)
	if err != nil {
		return nil, fmt.Errorf("failed to get hold details: %w", err)
	}

	if len(holdData.SeatIDs) == 0 {
		return []SeatInfo{}, nil
	}

	// Parse seat IDs and get seat details
	var seatUUIDs []uuid.UUID
	var seats []Seat

	for _, seatIDStr := range holdData.SeatIDs {
		seatID, err := uuid.Parse(seatIDStr)
		if err != nil {
			continue // Skip invalid seat IDs
		}

		seat, err := s.repo.GetSeatByID(ctx, seatID)
		if err != nil {
			continue // Skip invalid seats
		}

		seatUUIDs = append(seatUUIDs, seatID)
		seats = append(seats, *seat)
	}

	if len(seats) == 0 {
		return []SeatInfo{}, nil
	}

	// Calculate actual seat prices using the existing calculateSeatPrices method
	seatPrices, err := s.calculateSeatPrices(holdData.EventID, seats)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate seat prices: %w", err)
	}

	var seatInfos []SeatInfo
	for _, seat := range seats {
		// Get section details for name
		var section struct {
			ID   uuid.UUID `json:"id"`
			Name string    `json:"name"`
		}

		// Get section name
		if err := s.repo.(*repository).db.Table("venue_sections").
			Select("id, name").
			Where("id = ?", seat.SectionID).
			First(&section).Error; err == nil {

			// Use calculated price instead of hardcoded value
			seatPrice := seatPrices[seat.ID.String()]

			seatInfo := SeatInfo{
				ID:          seat.ID,
				SectionID:   seat.SectionID,
				SeatNumber:  seat.SeatNumber,
				Row:         seat.Row,
				Price:       seatPrice, // Now using calculated price based on event and section
				SectionName: section.Name,
			}
			seatInfos = append(seatInfos, seatInfo)
		}
	}

	return seatInfos, nil
}

func (s *service) GetHoldDetails(ctx context.Context, holdID string) (*SeatHoldDetails, error) {
	return s.repo.GetHoldDetails(ctx, holdID)
}

func (s *service) checkSeatsBookedForEvent(ctx context.Context, seatIDs []uuid.UUID, eventID uuid.UUID) ([]string, error) {
	var bookedSeatIDs []string

	// Query seat_bookings table to check if any of the seats are already booked for this event
	if err := s.repo.(*repository).db.WithContext(ctx).
		Table("seat_bookings sb").
		Joins("JOIN bookings b ON b.id = sb.booking_id").
		Where("b.event_id = ? AND sb.seat_id IN ? AND b.status != 'CANCELLED'", eventID, seatIDs).
		Pluck("sb.seat_id", &bookedSeatIDs).Error; err != nil {
		return nil, fmt.Errorf("failed to query seat bookings: %w", err)
	}

	// Convert to slice and return
	return bookedSeatIDs, nil
}

func (s *service) getSeatBookingsForEvent(ctx context.Context, eventID uuid.UUID, sectionID uuid.UUID) ([]SeatBooking, error) {
	var seatBookings []SeatBooking

	// Query seat_bookings table for this event and section
	if err := s.repo.(*repository).db.WithContext(ctx).
		Table("seat_bookings sb").
		Joins("JOIN bookings b ON b.id = sb.booking_id").
		Where("b.event_id = ? AND sb.section_id = ? AND b.status != 'CANCELLED'", eventID, sectionID).
		Select("sb.*").
		Find(&seatBookings).Error; err != nil {
		return nil, fmt.Errorf("failed to query seat bookings: %w", err)
	}

	return seatBookings, nil
}
