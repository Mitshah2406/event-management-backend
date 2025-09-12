package waitlist

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Controller handles HTTP requests for waitlist operations
type Controller struct {
	service Service
}

// NewController creates a new waitlist controller
func NewController(service Service) *Controller {
	return &Controller{
		service: service,
	}
}

// JoinWaitlist handles POST /api/v1/waitlist requests
func (c *Controller) JoinWaitlist(ctx *gin.Context) {
	var request JoinWaitlistRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Get user ID from context (would be set by auth middleware)
	userIDStr, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	// Call service to join waitlist
	response, err := c.service.JoinWaitlist(ctx.Request.Context(), userID, &request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "Successfully joined waitlist",
		"data":    response,
	})
}

// LeaveWaitlist handles DELETE /api/v1/waitlist/:event_id requests
func (c *Controller) LeaveWaitlist(ctx *gin.Context) {
	eventIDStr := ctx.Param("event_id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event ID",
		})
		return
	}

	// Get user ID from context
	userIDStr, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	// Call service to leave waitlist
	err = c.service.LeaveWaitlist(ctx.Request.Context(), userID, eventID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Successfully left waitlist",
	})
}

// GetWaitlistStatus handles GET /api/v1/waitlist/status/:event_id requests
func (c *Controller) GetWaitlistStatus(ctx *gin.Context) {
	eventIDStr := ctx.Param("event_id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event ID",
		})
		return
	}

	// Get user ID from context
	userIDStr, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	// Get waitlist status
	status, err := c.service.GetWaitlistStatus(ctx.Request.Context(), userID, eventID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": status,
	})
}

// GetWaitlistStats handles GET /api/v1/admin/waitlist/stats/:event_id requests
func (c *Controller) GetWaitlistStats(ctx *gin.Context) {
	eventIDStr := ctx.Param("event_id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event ID",
		})
		return
	}

	// TODO: Add admin role check here

	stats, err := c.service.GetWaitlistStats(ctx.Request.Context(), eventID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": stats,
	})
}

// GetWaitlistEntries handles GET /api/v1/admin/waitlist/entries/:event_id requests
func (c *Controller) GetWaitlistEntries(ctx *gin.Context) {
	eventIDStr := ctx.Param("event_id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event ID",
		})
		return
	}

	// Get status filter from query params
	statusStr := ctx.Query("status")
	var status WaitlistStatus
	if statusStr != "" {
		status = WaitlistStatus(statusStr)
		if !status.IsValid() {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid status filter",
			})
			return
		}
	}

	// Get pagination parameters
	pageStr := ctx.DefaultQuery("page", "1")
	limitStr := ctx.DefaultQuery("limit", "50")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 50
	}

	// TODO: Add admin role check here

	entries, err := c.service.GetWaitlistEntries(ctx.Request.Context(), eventID, status)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Apply pagination
	start := (page - 1) * limit
	end := start + limit
	if start > len(entries) {
		start = len(entries)
	}
	if end > len(entries) {
		end = len(entries)
	}

	paginatedEntries := entries[start:end]

	ctx.JSON(http.StatusOK, gin.H{
		"data": paginatedEntries,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       len(entries),
			"total_pages": (len(entries) + limit - 1) / limit,
		},
	})
}

// NotifyNextInLine handles POST /api/v1/admin/waitlist/notify/:event_id requests
func (c *Controller) NotifyNextInLine(ctx *gin.Context) {
	eventIDStr := ctx.Param("event_id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event ID",
		})
		return
	}

	var request struct {
		AvailableTickets int `json:"available_tickets" binding:"required,min=1"`
	}

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// TODO: Add admin role check here

	err = c.service.NotifyNextInLine(ctx.Request.Context(), eventID, request.AvailableTickets)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Notifications sent to next users in line",
	})
}

// ProcessCancellation handles POST /api/v1/admin/waitlist/cancellation/:event_id requests
func (c *Controller) ProcessCancellation(ctx *gin.Context) {
	eventIDStr := ctx.Param("event_id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event ID",
		})
		return
	}

	var request struct {
		FreedTickets int `json:"freed_tickets" binding:"required,min=1"`
	}

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// TODO: Add admin role check here

	err = c.service.ProcessCancellation(ctx.Request.Context(), eventID, request.FreedTickets)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Cancellation processed successfully",
	})
}

// HealthCheck handles GET /api/v1/waitlist/health requests
func (c *Controller) HealthCheck(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "waitlist",
		"timestamp": gin.H{
			"utc": ctx.Request.Header.Get("X-Request-Time"),
		},
	})
}

// GetRecentNotifications handles GET /api/v1/admin/waitlist/notifications/recent requests
func (c *Controller) GetRecentNotifications(ctx *gin.Context) {
	eventIDStr := ctx.Query("event_id")
	var eventID *uuid.UUID

	if eventIDStr != "" {
		parsed, err := uuid.Parse(eventIDStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid event ID format",
			})
			return
		}
		eventID = &parsed
	}

	// TODO: Add actual method to service to get recent notifications
	// For now, return a mock response to verify the endpoint works
	response := gin.H{
		"message":  "Recent notifications endpoint working",
		"event_id": eventID,
		"note":     "Check server logs for detailed notification dispatch information",
		"log_patterns": []string{
			"🔔 NOTIFICATION DISPATCH:",
			"🎫 WAITLIST:",
			"📧 SENDING:",
			"✅ NOTIFICATION SENT:",
			"🚀 KAFKA:",
			"❌ NOTIFICATION FAILED:",
		},
	}

	ctx.JSON(http.StatusOK, response)
}
