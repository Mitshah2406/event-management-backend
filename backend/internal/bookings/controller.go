package bookings

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Controller struct {
	service Service
}

func NewController(service Service) *Controller {
	return &Controller{service: service}
}

// ConfirmBooking handles POST /api/v1/bookings/confirm
func (c *Controller) ConfirmBooking(ctx *gin.Context) {
	// Get user ID from JWT context (set by middleware)
	userIDInterface, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userIDStr, ok := userIDInterface.(string)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Parse request body
	var req BookingConfirmationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Confirm booking
	response, err := c.service.ConfirmBooking(ctx.Request.Context(), userID, req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to confirm booking",
			"details": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "Booking confirmed successfully",
		"data":    response,
	})
}

// GetBooking handles GET /api/v1/bookings/:id
func (c *Controller) GetBooking(ctx *gin.Context) {
	// Parse booking ID from URL
	bookingIDStr := ctx.Param("id")
	bookingID, err := uuid.Parse(bookingIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
		return
	}

	// Get user ID from JWT context
	userIDInterface, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userIDStr, ok := userIDInterface.(string)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get booking
	booking, err := c.service.GetBooking(ctx.Request.Context(), bookingID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error":   "Booking not found",
			"details": err.Error(),
		})
		return
	}

	// Verify ownership (non-admin users can only see their own bookings)
	roleInterface, _ := ctx.Get("role")
	role, _ := roleInterface.(string)

	if role != "ADMIN" && booking.UserID != userID {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Booking retrieved successfully",
		"data":    booking,
	})
}

// GetUserBookings handles GET /api/v1/users/bookings
func (c *Controller) GetUserBookings(ctx *gin.Context) {
	// Get user ID from JWT context
	userIDInterface, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userIDStr, ok := userIDInterface.(string)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Parse pagination parameters
	limitStr := ctx.DefaultQuery("limit", "10")
	offsetStr := ctx.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 0 {
		limit = 10
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Get user bookings
	bookings, err := c.service.GetUserBookings(ctx.Request.Context(), userID, limit, offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get user bookings",
			"details": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Bookings retrieved successfully",
		"data": gin.H{
			"bookings": bookings,
			"count":    len(bookings),
			"limit":    limit,
			"offset":   offset,
		},
	})
}

// CancelBooking handles POST /api/v1/bookings/:id/cancel
func (c *Controller) CancelBooking(ctx *gin.Context) {
	// Parse booking ID from URL
	bookingIDStr := ctx.Param("id")
	bookingID, err := uuid.Parse(bookingIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
		return
	}

	// Get user ID from JWT context
	userIDInterface, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userIDStr, ok := userIDInterface.(string)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Cancel booking
	err = c.service.CancelBooking(ctx.Request.Context(), bookingID, userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to cancel booking",
			"details": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Booking cancelled successfully",
	})
}
