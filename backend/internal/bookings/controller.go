package bookings

import (
	"log"
	"net/http"
	"strconv"

	"evently/internal/shared/utils/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type Controller struct {
	service   Service
	validator *validator.Validate
}

func NewController(service Service) *Controller {
	return &Controller{
		service:   service,
		validator: validator.New(),
	}
}

// CreateBooking handles POST /bookings - Create a new booking
func (c *Controller) CreateBooking(ctx *gin.Context) {
	// Get user ID from JWT middleware
	userID, exists := ctx.Get("user_id")
	if !exists {
		response.RespondJSON(ctx, "error", http.StatusUnauthorized, "User not authenticated", nil, nil)
		return
	}

	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		response.RespondJSON(ctx, "error", http.StatusUnauthorized, "Invalid user ID", nil, nil)
		return
	}

	// Parse request body
	var req CreateBookingRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Invalid request body", nil, err.Error())
		return
	}

	// Validate request
	if err := c.validator.Struct(&req); err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Validation failed", nil, err.Error())
		return
	}

	// Create booking
	booking, err := c.service.CreateBooking(ctx.Request.Context(), userUUID, req)
	if err != nil {
		// Handle different types of errors appropriately
		switch {
		case err.Error() == "event not found":
			response.RespondJSON(ctx, "error", http.StatusNotFound, "Event not found", nil, nil)
		case err.Error() == "event is fully booked" || err.Error() == "event is not available for booking":
			response.RespondJSON(ctx, "error", http.StatusConflict, err.Error(), nil, nil)
		case err.Error() == "invalid event ID format":
			response.RespondJSON(ctx, "error", http.StatusBadRequest, err.Error(), nil, nil)
		default:
			response.RespondJSON(ctx, "error", http.StatusInternalServerError, "Failed to create booking: "+err.Error(), nil, nil)
		}
		return
	}

	response.RespondJSON(ctx, "success", http.StatusCreated, "Booking created successfully", booking, nil)
}

// CancelBooking handles DELETE /bookings/:id - Cancel a booking
func (c *Controller) CancelBooking(ctx *gin.Context) {
	// Get user ID from JWT middleware
	userID, exists := ctx.Get("user_id")
	if !exists {
		response.RespondJSON(ctx, "error", http.StatusUnauthorized, "User not authenticated", nil, nil)
		return
	}

	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		response.RespondJSON(ctx, "error", http.StatusUnauthorized, "Invalid user ID", nil, nil)
		return
	}

	// Get booking ID from URL parameter
	bookingIDStr := ctx.Param("id")
	bookingID, err := uuid.Parse(bookingIDStr)
	if err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Invalid booking ID", nil, nil)
		return
	}

	// Cancel booking
	err = c.service.CancelBooking(ctx.Request.Context(), userUUID, bookingID)
	if err != nil {
		switch {
		case err.Error() == "booking not found":
			response.RespondJSON(ctx, "error", http.StatusNotFound, "Booking not found", nil, nil)
		case err.Error() == "unauthorized: you can only cancel your own bookings":
			response.RespondJSON(ctx, "error", http.StatusForbidden, "Unauthorized to cancel this booking", nil, nil)
		case err.Error() == "booking is already cancelled":
			response.RespondJSON(ctx, "error", http.StatusConflict, "Booking is already cancelled", nil, nil)
		default:
			response.RespondJSON(ctx, "error", http.StatusInternalServerError, "Failed to cancel booking", nil, nil)
		}
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "Booking cancelled successfully", nil, nil)
}

// GetBookingDetails handles GET /bookings/:id - Get booking details
func (c *Controller) GetBookingDetails(ctx *gin.Context) {
	// Get user ID from JWT middleware
	userID, exists := ctx.Get("user_id")
	if !exists {
		response.RespondJSON(ctx, "error", http.StatusUnauthorized, "User not authenticated", nil, nil)
		return
	}

	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		response.RespondJSON(ctx, "error", http.StatusUnauthorized, "Invalid user ID", nil, nil)
		return
	}

	// Get booking ID from URL parameter
	bookingIDStr := ctx.Param("id")
	bookingID, err := uuid.Parse(bookingIDStr)
	if err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Invalid booking ID", nil, nil)
		return
	}

	// Get booking details
	booking, err := c.service.GetBookingDetails(ctx.Request.Context(), userUUID, bookingID)
	if err != nil {
		switch {
		case err.Error() == "booking not found":
			response.RespondJSON(ctx, "error", http.StatusNotFound, "Booking not found", nil, nil)
		case err.Error() == "unauthorized: you can only view your own bookings":
			response.RespondJSON(ctx, "error", http.StatusForbidden, "Unauthorized to view this booking", nil, nil)
		default:
			response.RespondJSON(ctx, "error", http.StatusInternalServerError, "Failed to get booking details", nil, nil)
		}
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "Booking details retrieved successfully", booking, nil)
}

// GetUserBookings handles GET /bookings/user/:id - Get user's booking history
func (c *Controller) GetUserBookings(ctx *gin.Context) {
	// Get user ID from JWT middleware
	userID, exists := ctx.Get("user_id")
	log.Println("userID from context:", userID, " exists:", exists)
	if !exists {
		response.RespondJSON(ctx, "error", http.StatusUnauthorized, "User not authenticated", nil, nil)
		return
	}

	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		response.RespondJSON(ctx, "error", http.StatusUnauthorized, "Invalid user ID", nil, nil)
		return
	}

	// Parse query parameters
	var query BookingListQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Invalid query parameters", nil, err.Error())
		return
	}

	// Get user bookings
	bookings, err := c.service.GetUserBookings(ctx.Request.Context(), userUUID, query)
	if err != nil {
		response.RespondJSON(ctx, "error", http.StatusInternalServerError, "Failed to get user bookings", nil, nil)
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "User bookings retrieved successfully", bookings, nil)
}

// Admin endpoints

// GetAllBookings handles GET /admin/bookings - Get all bookings (admin only)
func (c *Controller) GetAllBookings(ctx *gin.Context) {
	// Parse query parameters
	var query BookingListQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Invalid query parameters", nil, err.Error())
		return
	}

	// Get all bookings
	bookings, err := c.service.GetAllBookings(ctx.Request.Context(), query)
	if err != nil {
		response.RespondJSON(ctx, "error", http.StatusInternalServerError, "Failed to get all bookings", nil, nil)
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "All bookings retrieved successfully", bookings, nil)
}

// GetEventBookings handles GET /admin/events/:eventId/bookings - Get event bookings (admin only)
func (c *Controller) GetEventBookings(ctx *gin.Context) {
	// Get event ID from URL parameter
	eventIDStr := ctx.Param("eventId")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Invalid event ID", nil, nil)
		return
	}

	// Get event bookings
	bookings, err := c.service.GetEventBookings(ctx.Request.Context(), eventID)
	if err != nil {
		response.RespondJSON(ctx, "error", http.StatusInternalServerError, "Failed to get event bookings", nil, nil)
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "Event bookings retrieved successfully", bookings, nil)
}

// CancelBookingAsAdmin handles DELETE /admin/bookings/:id - Cancel booking as admin
func (c *Controller) CancelBookingAsAdmin(ctx *gin.Context) {
	// Get admin ID from JWT middleware
	adminID, exists := ctx.Get("user_id")
	if !exists {
		response.RespondJSON(ctx, "error", http.StatusUnauthorized, "User not authenticated", nil, nil)
		return
	}

	adminUUID, err := uuid.Parse(adminID.(string))
	if err != nil {
		response.RespondJSON(ctx, "error", http.StatusUnauthorized, "Invalid admin ID", nil, nil)
		return
	}

	// Get booking ID from URL parameter
	bookingIDStr := ctx.Param("id")
	bookingID, err := uuid.Parse(bookingIDStr)
	if err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Invalid booking ID", nil, nil)
		return
	}

	// Cancel booking as admin
	err = c.service.CancelBookingAsAdmin(ctx.Request.Context(), adminUUID, bookingID)
	if err != nil {
		switch {
		case err.Error() == "booking not found":
			response.RespondJSON(ctx, "error", http.StatusNotFound, "Booking not found", nil, nil)
		case err.Error() == "booking is already cancelled":
			response.RespondJSON(ctx, "error", http.StatusConflict, "Booking is already cancelled", nil, nil)
		default:
			response.RespondJSON(ctx, "error", http.StatusInternalServerError, "Failed to cancel booking", nil, nil)
		}
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "Booking cancelled successfully by admin", nil, nil)
}

// GetBookingDetailsAsAdmin handles GET /admin/bookings/:id - Get booking details as admin
func (c *Controller) GetBookingDetailsAsAdmin(ctx *gin.Context) {
	// Get booking ID from URL parameter
	bookingIDStr := ctx.Param("id")
	bookingID, err := uuid.Parse(bookingIDStr)
	if err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Invalid booking ID", nil, nil)
		return
	}

	// Get booking details as admin
	booking, err := c.service.GetBookingDetailsAsAdmin(ctx.Request.Context(), bookingID)
	if err != nil {
		switch {
		case err.Error() == "booking not found":
			response.RespondJSON(ctx, "error", http.StatusNotFound, "Booking not found", nil, nil)
		default:
			response.RespondJSON(ctx, "error", http.StatusInternalServerError, "Failed to get booking details", nil, nil)
		}
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "Booking details retrieved successfully", booking, nil)
}

// Helper function to parse integer from string with default value
func parseIntWithDefault(str string, defaultValue int) int {
	if str == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(str)
	if err != nil {
		return defaultValue
	}
	return val
}
