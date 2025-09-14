package routes

import (
	"context"
	"evently/internal/analytics"
	"evently/internal/auth"
	"evently/internal/bookings"
	"evently/internal/cancellation"
	"evently/internal/events"
	"evently/internal/notifications"
	"evently/internal/seats"
	"evently/internal/shared/config"
	"evently/internal/shared/database"
	"evently/internal/shared/middleware"
	"evently/internal/tags"
	"evently/internal/venues"
	"evently/internal/waitlist"
	"evently/pkg/cache"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	files "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

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

type Router struct {
	config                 *config.Config
	db                     *database.DB
	tagService             tags.Service             // For dependency injection
	eventService           events.Service           // For dependency injection
	venueService           venues.Service           // For dependency injection
	bookingService         bookings.Service         // For dependency injection
	cancellationService    cancellation.Service     // For dependency injection
	cancellationController *cancellation.Controller // For controller recreation when service updates
	analyticsService       analytics.Service        // For analytics
	waitlistService        waitlist.Service         // For waitlist operations
	cacheService           cache.Service            // For caching
	notificationService    notifications.NotificationService
}

func NewRouter(cfg *config.Config, db *database.DB, notificationService notifications.NotificationService) *Router {

	cacheService := cache.NewService(db.GetRedis())

	return &Router{
		config:              cfg,
		db:                  db,
		cacheService:        cacheService,
		notificationService: notificationService,
	}
}

func (r *Router) SetupRoutes(engine *gin.Engine) {

	r.setupHealthRoutes(engine)

	// Redirect root path to health check
	engine.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/health")
	})

	r.setupSwaggerRoutes(engine)

	api := engine.Group(r.config.GetAPIBasePath())
	{

		r.setupAuthRoutes(api)

		r.setupTagRoutes(api)

		r.setupVenueRoutes(api)

		r.setupSeatRoutes(api)

		r.setupEventRoutes(api)

		r.setupCancellationRoutes(api)

		r.setupWaitlistRoutes(api)

		r.setupBookingRoutes(api)

		r.setupAnalyticsRoutes(api)
	}
}

func (r *Router) setupHealthRoutes(engine *gin.Engine) {
	engine.GET("/health", func(c *gin.Context) {

		if err := r.db.HealthCheckDB(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":    "unhealthy",
				"error":     err.Error(),
				"timestamp": time.Now(),
				"docs":      "/docs",
				"service":   "event-backend",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now(),
			"service":   "event-backend",
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

func (r *Router) setupAuthRoutes(rg *gin.RouterGroup) {

	authRepo := auth.NewRepository(r.db.GetPostgreSQL())
	authService := auth.NewService(authRepo, r.config)
	authController := auth.NewController(authService)
	authRouter := auth.NewRouter(authController)

	authRouter.SetupRoutes(rg)
}

func (r *Router) setupTagRoutes(rg *gin.RouterGroup) {

	tagRepo := tags.NewRepository(r.db.GetPostgreSQL())
	tagService := tags.NewService(tagRepo)
	tagController := tags.NewController(tagService)

	// Store tag service for dependency injection
	r.tagService = tagService

	tags.SetupTagRoutes(rg, tagController)
}

func (r *Router) setupEventRoutes(rg *gin.RouterGroup) {
	// Initialize event dependencies
	eventRepo := events.NewRepository(r.db.GetPostgreSQL())
	eventService := events.NewService(eventRepo)

	// Inject cache service dependency
	if eventService, ok := eventService.(interface{ SetCacheService(cache.Service) }); ok && r.cacheService != nil {
		eventService.SetCacheService(r.cacheService)
	}

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

	events.SetupEventRoutes(rg, eventController)
}

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

	venues.SetupVenueRoutes(rg, venueController)
}

func (r *Router) setupSeatRoutes(rg *gin.RouterGroup) {
	// Initialize seat dependencies with Redis support
	seatRepo := seats.NewRepository(r.db.GetPostgreSQL(), r.db.GetRedis())
	seatService := seats.NewService(seatRepo, r.config)

	// Inject cache service dependency
	if seatService, ok := seatService.(interface{ SetCacheService(cache.Service) }); ok && r.cacheService != nil {
		seatService.SetCacheService(r.cacheService)
	}

	seatController := seats.NewController(seatService)

	seats.SetupSeatRoutes(rg, seatController)
}

func (r *Router) setupBookingRoutes(rg *gin.RouterGroup) {
	// Initialize booking dependencies
	bookingRepo := bookings.NewRepository(r.db.GetPostgreSQL())

	// Create seat service adapter for booking service
	seatRepo := seats.NewRepository(r.db.GetPostgreSQL(), r.db.GetRedis())
	seatService := seats.NewService(seatRepo, r.config)
	seatServiceAdapter := &SeatServiceAdapter{seatService: seatService}

	// Create waitlist service adapter for booking service
	var waitlistServiceAdapter bookings.WaitlistService
	if r.waitlistService != nil {
		waitlistServiceAdapter = &WaitlistServiceAdapterForBookings{waitlistService: r.waitlistService}
	}

	// Create booking service
	bookingService := bookings.NewService(bookingRepo, seatServiceAdapter, waitlistServiceAdapter)
	bookingController := bookings.NewController(bookingService)

	// Store booking service for dependency injection
	r.bookingService = bookingService

	// Update cancellation service with booking service dependency (if cancellation service exists)
	if r.cancellationService != nil {
		// Create booking service adapter for cancellation service
		bookingServiceAdapter := &BookingServiceAdapter{bookingService: bookingService}

		// Get existing waitlist service adapter if available
		var waitlistAdapter cancellation.WaitlistService
		if r.waitlistService != nil {
			waitlistAdapter = &WaitlistServiceAdapter{waitlistService: r.waitlistService}
		}

		// Re-create cancellation service with booking dependency
		cancellationRepo := cancellation.NewRepository(r.db.GetPostgreSQL())
		r.cancellationService = cancellation.NewService(cancellationRepo, bookingServiceAdapter, waitlistAdapter)

		// Recreate the controller with the updated service
		r.cancellationController = cancellation.NewController(r.cancellationService)
	}

	bookings.SetupBookingRoutes(rg, bookingController)
}

func (r *Router) setupCancellationRoutes(rg *gin.RouterGroup) {
	// Initialize cancellation dependencies
	cancellationRepo := cancellation.NewRepository(r.db.GetPostgreSQL())

	// Create booking service adapter for cancellation service
	var bookingServiceAdapter cancellation.BookingService
	if r.bookingService != nil {
		bookingServiceAdapter = &BookingServiceAdapter{bookingService: r.bookingService}
	}

	// Initialize without waitlist service (will be injected later)
	cancellationService := cancellation.NewService(cancellationRepo, bookingServiceAdapter, nil)
	cancellationController := cancellation.NewController(cancellationService)

	// Store cancellation service and controller for dependency injection
	r.cancellationService = cancellationService
	r.cancellationController = cancellationController

	r.setupCancellationRoutesWithWrappers(rg)
}

type BookingServiceAdapter struct {
	bookingService bookings.Service
}

func (b *BookingServiceAdapter) GetBooking(ctx context.Context, bookingID uuid.UUID) (cancellation.BookingInfo, error) {
	booking, err := b.bookingService.GetBooking(ctx, bookingID)
	if err != nil {
		return cancellation.BookingInfo{}, err
	}

	return cancellation.BookingInfo{
		ID:         booking.ID,
		UserID:     booking.UserID,
		EventID:    booking.EventID,
		TotalPrice: booking.TotalPrice,
		TotalSeats: booking.TotalSeats,
		Status:     booking.Status,
		BookingRef: booking.BookingRef,
		Version:    booking.Version,
		CreatedAt:  booking.CreatedAt,
	}, nil
}

func (b *BookingServiceAdapter) CancelBookingInternal(ctx context.Context, bookingID uuid.UUID) error {
	return b.bookingService.CancelBookingInternal(ctx, bookingID)
}

func (b *BookingServiceAdapter) CancelBookingWithVersion(ctx context.Context, bookingID uuid.UUID, expectedVersion int) error {
	return b.bookingService.CancelBookingWithVersion(ctx, bookingID, expectedVersion)
}

type WaitlistServiceAdapter struct {
	waitlistService waitlist.Service
}

func (w *WaitlistServiceAdapter) ProcessCancellation(ctx context.Context, eventID uuid.UUID, freedTickets int) error {
	return w.waitlistService.ProcessCancellation(ctx, eventID, freedTickets)
}

type SeatServiceAdapter struct {
	seatService seats.Service
}

func (s *SeatServiceAdapter) ValidateHold(ctx context.Context, holdID string, userID string) (*bookings.HoldValidationResult, error) {
	result, err := s.seatService.ValidateHold(ctx, holdID, userID)
	if err != nil {
		return nil, err
	}

	var seats []bookings.SeatInfo
	if result.Details != nil {

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

	return &bookings.HoldValidationResult{
		Valid:     result.Valid,
		HoldID:    holdID,
		UserID:    userID,
		ExpiresAt: time.Now().Add(10 * time.Minute),
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

	return &bookings.SeatHoldDetails{
		HoldID:  details.HoldID,
		UserID:  details.UserID,
		EventID: details.EventID,
		SeatIDs: details.SeatIDs,
		TTL:     details.TTL,
	}, nil
}

type WaitlistServiceAdapterForBookings struct {
	waitlistService waitlist.Service
}

func (w *WaitlistServiceAdapterForBookings) GetWaitlistStatusForBooking(ctx context.Context, userID, eventID uuid.UUID) (*bookings.WaitlistStatusForBooking, error) {
	status, err := w.waitlistService.GetWaitlistStatusForBooking(ctx, userID, eventID)
	if err != nil {
		return nil, err
	}
	if status == nil {
		return nil, nil
	}
	return &bookings.WaitlistStatusForBooking{
		Status:    status.Status,
		IsExpired: status.IsExpired,
	}, nil
}

func (w *WaitlistServiceAdapterForBookings) MarkAsConverted(ctx context.Context, userID, eventID, bookingID uuid.UUID) error {
	return w.waitlistService.MarkAsConverted(ctx, userID, eventID, bookingID)
}

func (r *Router) setupAnalyticsRoutes(rg *gin.RouterGroup) {

	analyticsRepo := analytics.NewRepository(r.db.GetPostgreSQL())
	analyticsService := analytics.NewService(analyticsRepo)

	if analyticsService, ok := analyticsService.(interface{ SetCacheService(cache.Service) }); ok && r.cacheService != nil {
		analyticsService.SetCacheService(r.cacheService)
	}

	analyticsController := analytics.NewController(analyticsService)

	// Store analytics service for dependency injection
	r.analyticsService = analyticsService

	analytics.SetupAnalyticsRoutes(rg, analyticsController)
}

func (r *Router) setupWaitlistRoutes(rg *gin.RouterGroup) {
	// Initialize waitlist dependencies
	waitlistRepo := waitlist.NewRepository(r.db.GetPostgreSQL(), r.db.GetRedis())

	var notificationAdapter waitlist.NotificationService
	if r.notificationService != nil {
		// Use the notification service passed from main.go
		notificationAdapter = notifications.NewWaitlistServiceAdapter(r.notificationService)
		log.Printf("✅ Waitlist service initialized with email notification adapter")
	} else {
		log.Printf("⚠️ No notification service available for waitlist - notifications will be disabled")
	}

	// Create user service adapter
	authRepo := auth.NewRepository(r.db.GetPostgreSQL())
	userServiceAdapter := auth.NewUserServiceAdapter(authRepo)

	// Create waitlist service
	waitlistService := waitlist.NewService(waitlistRepo, notificationAdapter, userServiceAdapter, nil)
	waitlistController := waitlist.NewController(waitlistService)

	// Store waitlist service for dependency injection
	r.waitlistService = waitlistService

	// Update cancellation service with waitlist service dependency (if cancellation service exists)
	if r.cancellationService != nil {
		// Create waitlist service adapter for cancellation service
		waitlistAdapter := &WaitlistServiceAdapter{waitlistService: waitlistService}

		// Re-create cancellation service with waitlist dependency
		cancellationRepo := cancellation.NewRepository(r.db.GetPostgreSQL())
		var bookingServiceAdapter cancellation.BookingService
		if r.bookingService != nil {
			bookingServiceAdapter = &BookingServiceAdapter{bookingService: r.bookingService}
		}

		// Update the cancellation service with waitlist integration
		r.cancellationService = cancellation.NewService(cancellationRepo, bookingServiceAdapter, waitlistAdapter)

		// Recreate the controller with the updated service
		r.cancellationController = cancellation.NewController(r.cancellationService)
	}

	// Setup waitlist routes
	waitlist.SetupWaitlistRoutes(rg, waitlistController)
}

func (r *Router) setupCancellationRoutesWithWrappers(rg *gin.RouterGroup) {
	// Event cancellation policy routes (Admin only)
	events := rg.Group("/admin/events")
	events.Use(middleware.JWTAuth(), middleware.RequireRoles("ADMIN"))
	{
		events.POST("/:eventId/cancellation-policy", func(c *gin.Context) {
			r.cancellationController.CreateCancellationPolicy(c)
		})
		events.GET("/:eventId/cancellation-policy", func(c *gin.Context) {
			r.cancellationController.GetCancellationPolicy(c)
		})
		events.PUT("/:eventId/cancellation-policy", func(c *gin.Context) {
			r.cancellationController.UpdateCancellationPolicy(c)
		})
	}

	// Booking cancellation routes (Users and Admins)
	bookings := rg.Group("/bookings")
	bookings.Use(middleware.JWTAuth(), middleware.RequireRoles("USER", "ADMIN"))
	{
		bookings.POST("/:id/request-cancel", func(c *gin.Context) {
			r.cancellationController.RequestCancellation(c)
		})
	}

	// Cancellation management routes
	cancellations := rg.Group("/cancellations")
	cancellations.Use(middleware.JWTAuth(), middleware.RequireRoles("USER", "ADMIN"))
	{
		cancellations.GET("/:id", func(c *gin.Context) {
			r.cancellationController.GetCancellation(c)
		})
	}

	// User-specific cancellation routes
	users := rg.Group("/users")
	users.Use(middleware.JWTAuth(), middleware.RequireRoles("USER", "ADMIN"))
	{
		users.GET("/cancellations", func(c *gin.Context) {
			r.cancellationController.GetUserCancellations(c)
		})
	}
}

func (r *Router) setupSwaggerRoutes(engine *gin.Engine) {

	swaggerPath := r.findSwaggerFile()
	if swaggerPath == "" {
		log.Println("⚠️ swagger.yaml not found, Swagger UI will not be available")
		return
	}

	log.Printf("✅ Found swagger.yaml at: %s", swaggerPath)

	engine.StaticFile("/swagger.yaml", swaggerPath)

	engine.GET("/swagger-debug", func(c *gin.Context) {
		wd, _ := os.Getwd()
		c.JSON(http.StatusOK, gin.H{
			"working_directory": wd,
			"swagger_path":      swaggerPath,
			"file_exists":       r.fileExists(swaggerPath),
			"available_paths": []string{
				"GET /docs - Swagger UI",
				"GET /swagger - Alternative Swagger UI",
				"GET /swagger.yaml - Raw YAML file",
				"GET /swagger-debug - This debug endpoint",
			},
		})
	})

	engine.GET("/docs/*any", ginSwagger.WrapHandler(files.Handler, ginSwagger.URL("/swagger.yaml")))

	engine.GET("/swagger/*any", ginSwagger.WrapHandler(files.Handler, ginSwagger.URL("/swagger.yaml")))

	engine.GET("/docs", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/docs/index.html")
	})

	engine.GET("/swagger", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})

	log.Println("📖 Swagger UI available at:")
	log.Println("   http://localhost:8080/docs")
	log.Println("   http://localhost:8080/swagger")
	log.Println("   http://localhost:8080/swagger.yaml (raw file)")
	log.Println("   http://localhost:8080/swagger-debug (debug info)")
}

func (r *Router) findSwaggerFile() string {
	possiblePaths := []string{
		"./docs/swagger.yaml",     // If running from backend/
		"../docs/swagger.yaml",    // If running from backend/server/
		"docs/swagger.yaml",       // Alternative relative path
		"../../docs/swagger.yaml", // If running from backend/cmd/something/
		"./swagger.yaml",          // If swagger.yaml is in current dir
	}

	// Try each possible path
	for _, path := range possiblePaths {
		if r.fileExists(path) {
			return path
		}
	}

	// Try to find it by searching from current working directory
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("Error getting working directory: %v", err)
		return ""
	}

	// Search in parent directories
	currentDir := wd
	for i := 0; i < 5; i++ { // Search up to 5 levels up
		swaggerPath := filepath.Join(currentDir, "docs", "swagger.yaml")
		if r.fileExists(swaggerPath) {
			return swaggerPath
		}
		currentDir = filepath.Dir(currentDir)
	}

	log.Printf("❌ swagger.yaml not found. Tried paths: %v", possiblePaths)
	log.Printf("Current working directory: %s", wd)
	return ""
}

func (r *Router) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
