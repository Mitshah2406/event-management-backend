package cancellation

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Controller handles HTTP requests for cancellation policies
type Controller struct {
	service Service
}

// NewController creates a new cancellation controller
func NewController(service Service) *Controller {
	return &Controller{service: service}
}

// CreateCancellationPolicy handles POST /api/v1/events/:eventId/cancellation-policy
func (c *Controller) CreateCancellationPolicy(ctx *gin.Context) {
	// Parse event ID from URL
	eventIDStr := ctx.Param("eventId")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	// Parse request body
	var req CancellationPolicyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Create cancellation policy
	policy, err := c.service.CreateCancellationPolicy(ctx.Request.Context(), eventID, req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to create cancellation policy",
			"details": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "Cancellation policy created successfully",
		"data":    policy,
	})
}

// GetCancellationPolicy handles GET /api/v1/events/:eventId/cancellation-policy
func (c *Controller) GetCancellationPolicy(ctx *gin.Context) {
	// Parse event ID from URL
	eventIDStr := ctx.Param("eventId")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	// Get cancellation policy
	policy, err := c.service.GetCancellationPolicy(ctx.Request.Context(), eventID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error":   "Cancellation policy not found",
			"details": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Cancellation policy retrieved successfully",
		"data":    policy,
	})
}

// UpdateCancellationPolicy handles PUT /api/v1/events/:eventId/cancellation-policy
func (c *Controller) UpdateCancellationPolicy(ctx *gin.Context) {
	// Parse event ID from URL
	eventIDStr := ctx.Param("eventId")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	// Parse request body
	var req CancellationPolicyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Update cancellation policy
	policy, err := c.service.UpdateCancellationPolicy(ctx.Request.Context(), eventID, req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to update cancellation policy",
			"details": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Cancellation policy updated successfully",
		"data":    policy,
	})
}

func (c *Controller) RequestCancellation(ctx *gin.Context) {
	bookingIDStr := ctx.Param("id")
	bookingID, err := uuid.Parse(bookingIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
		return
	}

	// Get user ID from JWT
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

	var req CancellationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	cancellation, err := c.service.RequestCancellation(ctx.Request.Context(), bookingID, userID, req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to request cancellation",
			"details": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "Cancellation processed successfully. Refund will be credited within the specified processing days.",
		"data":    cancellation,
	})
}

// GetCancellation handles GET /api/v1/cancellations/:id
func (c *Controller) GetCancellation(ctx *gin.Context) {
	// Parse cancellation ID from URL
	cancellationIDStr := ctx.Param("id")
	cancellationID, err := uuid.Parse(cancellationIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cancellation ID"})
		return
	}

	// Get cancellation
	cancellation, err := c.service.GetCancellation(ctx.Request.Context(), cancellationID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error":   "Cancellation not found",
			"details": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Cancellation retrieved successfully",
		"data":    cancellation,
	})
}

// GetUserCancellations handles GET /api/v1/users/cancellations
func (c *Controller) GetUserCancellations(ctx *gin.Context) {
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

	// Get user cancellations
	cancellations, err := c.service.GetUserCancellations(ctx.Request.Context(), userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get user cancellations",
			"details": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "User cancellations retrieved successfully",
		"data": gin.H{
			"cancellations": cancellations,
			"count":         len(cancellations),
		},
	})
}
