// api/routes/router.go
package routes

import (
	"context"
	"evently/internal/auth"
	"evently/internal/bookings"
	"evently/internal/cancellation"
	"evently/internal/events"
	"evently/internal/seats"
	"evently/internal/shared/config"
	"evently/internal/shared/database"
	"evently/internal/tags"
	"evently/internal/venues"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// VenueServiceAdapter adapts venues.Service to events.VenueService interface
type VenueServiceAdapter struct {
	venueService venues.Service
}

func (v *VenueServiceAdapter) GetSectionsByTemplateID(ctx context.Context, templateID string) (interface{}, error) {
	sections, err := v.venueService.GetSectionsByTemplateID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	return sections, nil
}

// Router holds all route dependencies
type Router struct {
	config              *config.Config
	db                  *database.DB
	tagService          tags.Service         // For dependency injection
	eventService        events.Service       // For dependency injection
	venueService        venues.Service       // For dependency injection
	bookingService      bookings.Service     // For dependency injection
	cancellationService cancellation.Service // For dependency injection
}

// NewRouter creates a new router instance
func NewRouter(cfg *config.Config, db *database.DB) *Router {
	return &Router{
		config: cfg,
		db:     db,
	}
}

// SetupRoutes configures all application routes
func (r *Router) SetupRoutes(engine *gin.Engine) {
	// Health check and basic info endpoints
	r.setupHealthRoutes(engine)

	// API routes
	api := engine.Group(r.config.GetAPIBasePath())
	{
		// Setup auth routes
		r.setupAuthRoutes(api)

		// Setup tag routes (must be before event routes for dependency injection)
		r.setupTagRoutes(api)

		// Setup venue routes (foundation for events)
		r.setupVenueRoutes(api)

		// Setup seat routes (seat management and holding)
		r.setupSeatRoutes(api)

		// Setup event routes
		r.setupEventRoutes(api)

		// Setup booking routes
		r.setupBookingRoutes(api)

		// Setup cancellation routes
		r.setupCancellationRoutes(api)
	}
}

// setupHealthRoutes sets up health check and system status routes
func (r *Router) setupHealthRoutes(engine *gin.Engine) {
	engine.GET("/health", func(c *gin.Context) {
		// Perform health checks
		if err := r.db.HealthCheck(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":    "unhealthy",
				"error":     err.Error(),
				"timestamp": time.Now(),
				"service":   "evently-backend",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now(),
			"service":   "evently-backend",
		})
	})

	engine.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
			"version": r.config.APIVersion,
		})
	})

	engine.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":      "operational",
			"api_version": r.config.APIVersion,
			"timestamp":   time.Now(),
		})
	})
}

// setupAuthRoutes configures authentication routes
func (r *Router) setupAuthRoutes(rg *gin.RouterGroup) {
	// Initialize auth dependencies
	authRepo := auth.NewRepository(r.db.GetPostgreSQL()) // Use the correct method
	authService := auth.NewService(authRepo, r.config)
	authController := auth.NewController(authService)
	authRouter := auth.NewRouter(authController)

	// Setup auth routes
	authRouter.SetupRoutes(rg)
}

// setupTagRoutes configures tag management routes
func (r *Router) setupTagRoutes(rg *gin.RouterGroup) {
	// Initialize tag dependencies
	tagRepo := tags.NewRepository(r.db.GetPostgreSQL())
	tagService := tags.NewService(tagRepo)
	tagController := tags.NewController(tagService)

	// Store tag service for dependency injection
	r.tagService = tagService

	// Setup tag routes
	tags.SetupTagRoutes(rg, tagController)
}

// setupEventRoutes configures event management routes
func (r *Router) setupEventRoutes(rg *gin.RouterGroup) {
	// Initialize event dependencies
	eventRepo := events.NewRepository(r.db.GetPostgreSQL())
	eventService := events.NewService(eventRepo)

	// Inject tag service dependency
	if r.tagService != nil {
		eventService.SetTagService(r.tagService)
	}

	// Inject venue service dependency through an adapter
	if r.venueService != nil {
		venueServiceAdapter := &VenueServiceAdapter{venueService: r.venueService}
		eventService.SetVenueService(venueServiceAdapter)
	}

	// Store event service for dependency injection
	r.eventService = eventService

	eventController := events.NewController(eventService)

	// Setup event routes
	events.SetupEventRoutes(rg, eventController)
}

// setupVenueRoutes configures venue management routes
func (r *Router) setupVenueRoutes(rg *gin.RouterGroup) {
	// Initialize venue dependencies
	venueRepo := venues.NewRepository(r.db.GetPostgreSQL())

	// Initialize seat repository for automatic seat generation
	seatRepo := seats.NewRepository(r.db.GetPostgreSQL(), r.db.GetRedis())

	// Create venue service with both venue and seat repositories
	venueService := venues.NewService(venueRepo, seatRepo)
	venueController := venues.NewController(venueService)

	// Store venue service for dependency injection
	r.venueService = venueService

	// Setup venue routes
	venues.SetupVenueRoutes(rg, venueController)
}

// setupSeatRoutes configures seat management and holding routes
func (r *Router) setupSeatRoutes(rg *gin.RouterGroup) {
	// Initialize seat dependencies with Redis support
	seatRepo := seats.NewRepository(r.db.GetPostgreSQL(), r.db.GetRedis())
	seatService := seats.NewService(seatRepo, r.config)
	seatController := seats.NewController(seatService)

	// Setup seat routes
	seats.SetupSeatRoutes(rg, seatController)
}

// setupBookingRoutes configures booking management routes
func (r *Router) setupBookingRoutes(rg *gin.RouterGroup) {
	// Initialize booking dependencies
	bookingRepo := bookings.NewRepository(r.db.GetPostgreSQL())

	// Create seat service adapter for booking service
	seatRepo := seats.NewRepository(r.db.GetPostgreSQL(), r.db.GetRedis())
	seatService := seats.NewService(seatRepo, r.config)
	seatServiceAdapter := &SeatServiceAdapter{seatService: seatService}

	// Create booking service
	bookingService := bookings.NewService(bookingRepo, seatServiceAdapter)
	bookingController := bookings.NewController(bookingService)

	// Store booking service for dependency injection
	r.bookingService = bookingService

	// Setup booking routes
	bookings.SetupBookingRoutes(rg, bookingController)
}

// setupCancellationRoutes configures cancellation management routes
func (r *Router) setupCancellationRoutes(rg *gin.RouterGroup) {
	// Initialize cancellation dependencies
	cancellationRepo := cancellation.NewRepository(r.db.GetPostgreSQL())

	// Create booking service adapter for cancellation service
	var bookingServiceAdapter cancellation.BookingService
	if r.bookingService != nil {
		bookingServiceAdapter = &BookingServiceAdapter{bookingService: r.bookingService}
	}

	cancellationService := cancellation.NewService(cancellationRepo, bookingServiceAdapter)
	cancellationController := cancellation.NewController(cancellationService)

	// Store cancellation service for dependency injection
	r.cancellationService = cancellationService

	// Setup cancellation routes
	cancellation.SetupCancellationRoutes(rg, cancellationController)
}

// BookingServiceAdapter adapts bookings.Service to cancellation.BookingService interface
type BookingServiceAdapter struct {
	bookingService bookings.Service
}

func (b *BookingServiceAdapter) GetBooking(ctx context.Context, bookingID uuid.UUID) (cancellation.BookingInfo, error) {
	booking, err := b.bookingService.GetBooking(ctx, bookingID)
	if err != nil {
		return cancellation.BookingInfo{}, err
	}

	// Convert bookings.Booking to cancellation.BookingInfo
	return cancellation.BookingInfo{
		ID:         booking.ID,
		UserID:     booking.UserID,
		EventID:    booking.EventID,
		TotalPrice: booking.TotalPrice,
		Status:     booking.Status,
		BookingRef: booking.BookingRef,
		CreatedAt:  booking.CreatedAt,
	}, nil
}

func (b *BookingServiceAdapter) CancelBookingInternal(ctx context.Context, bookingID uuid.UUID) error {
	return b.bookingService.CancelBookingInternal(ctx, bookingID)
}

// SeatServiceAdapter adapts seats.Service to bookings.SeatService interface
type SeatServiceAdapter struct {
	seatService seats.Service
}

func (s *SeatServiceAdapter) ValidateHold(ctx context.Context, holdID string, userID string) (*bookings.HoldValidationResult, error) {
	result, err := s.seatService.ValidateHold(ctx, holdID, userID)
	if err != nil {
		return nil, err
	}

	// Convert seats.HoldValidationResult to bookings.HoldValidationResult
	var seats []bookings.SeatInfo
	if result.Details != nil {
		// Get seat info from hold details
		seatInfos, _ := s.seatService.GetSeatsByHoldID(ctx, holdID)
		for _, seatInfo := range seatInfos {
			seats = append(seats, bookings.SeatInfo{
				ID:          seatInfo.ID,
				SectionID:   seatInfo.SectionID,
				SeatNumber:  seatInfo.SeatNumber,
				Row:         seatInfo.Row,
				Price:       seatInfo.Price,
				SectionName: seatInfo.SectionName,
			})
		}
	}

	// Create a simplified validation result
	return &bookings.HoldValidationResult{
		Valid:     result.Valid,
		HoldID:    holdID,
		UserID:    userID,
		ExpiresAt: time.Now().Add(10 * time.Minute), // Default TTL
		Seats:     seats,
	}, nil
}

func (s *SeatServiceAdapter) ReleaseHold(ctx context.Context, holdID string) error {
	return s.seatService.ReleaseHold(ctx, holdID)
}

func (s *SeatServiceAdapter) GetSeatsByHoldID(ctx context.Context, holdID string) ([]bookings.SeatInfo, error) {
	seatsInfo, err := s.seatService.GetSeatsByHoldID(ctx, holdID)
	if err != nil {
		return nil, err
	}

	// Convert []seats.SeatInfo to []bookings.SeatInfo
	var result []bookings.SeatInfo
	for _, seatInfo := range seatsInfo {
		result = append(result, bookings.SeatInfo{
			ID:          seatInfo.ID,
			SectionID:   seatInfo.SectionID,
			SeatNumber:  seatInfo.SeatNumber,
			Row:         seatInfo.Row,
			Price:       seatInfo.Price,
			SectionName: seatInfo.SectionName,
		})
	}

	return result, nil
}

func (s *SeatServiceAdapter) GetHoldDetails(ctx context.Context, holdID string) (*bookings.SeatHoldDetails, error) {
	details, err := s.seatService.GetHoldDetails(ctx, holdID)
	if err != nil {
		return nil, err
	}

	// Convert seats.SeatHoldDetails to bookings.SeatHoldDetails
	return &bookings.SeatHoldDetails{
		HoldID:  details.HoldID,
		UserID:  details.UserID,
		EventID: details.EventID,
		SeatIDs: details.SeatIDs,
		TTL:     details.TTL,
	}, nil
}

func (s *SeatServiceAdapter) UpdateSeatStatusToBulk(ctx context.Context, seatIDs []uuid.UUID, status string) error {
	return s.seatService.UpdateSeatStatusToBulk(ctx, seatIDs, status)
}
