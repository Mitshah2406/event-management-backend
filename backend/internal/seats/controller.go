package seats

import (
	"evently/internal/shared/utils/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	service Service
}

func NewController(service Service) *Controller {
	return &Controller{service: service}
}

// SEAT MANAGEMENT

func (c *Controller) GetSeatsBySectionID(ctx *gin.Context) {
	sectionID := ctx.Param("sectionId")
	if sectionID == "" {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Section ID is required", nil, "missing section ID")
		return
	}

	seats, err := c.service.GetSeatsBySectionID(ctx.Request.Context(), sectionID)
	if err != nil {
		response.RespondJSON(ctx, "error", http.StatusInternalServerError, "Failed to get seats", nil, err.Error())
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "Seats retrieved successfully", seats, nil)
}

func (c *Controller) GetSeat(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Seat ID is required", nil, "missing seat ID")
		return
	}

	seat, err := c.service.GetSeatByID(ctx.Request.Context(), id)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "seat not found" {
			statusCode = http.StatusNotFound
		}
		response.RespondJSON(ctx, "error", statusCode, "Failed to get seat", nil, err.Error())
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "Seat retrieved successfully", seat, nil)
}

func (c *Controller) UpdateSeat(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Seat ID is required", nil, "missing seat ID")
		return
	}

	var req UpdateSeatRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Invalid request data", nil, err.Error())
		return
	}

	seat, err := c.service.UpdateSeat(ctx.Request.Context(), id, req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "seat not found" {
			statusCode = http.StatusNotFound
		}
		response.RespondJSON(ctx, "error", statusCode, "Failed to update seat", nil, err.Error())
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "Seat updated successfully", seat, nil)
}

func (c *Controller) DeleteSeat(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Seat ID is required", nil, "missing seat ID")
		return
	}

	err := c.service.DeleteSeat(ctx.Request.Context(), id)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "seat not found" {
			statusCode = http.StatusNotFound
		}
		response.RespondJSON(ctx, "error", statusCode, "Failed to delete seat", nil, err.Error())
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "Seat deleted successfully", nil, nil)
}

//  SEAT HOLDING

func (c *Controller) HoldSeats(ctx *gin.Context) {
	var req SeatHoldRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Invalid request data", nil, err.Error())
		return
	}

	holdResponse, err := c.service.HoldSeats(ctx.Request.Context(), req)
	if err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Failed to hold seats", nil, err.Error())
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "Seats held successfully", holdResponse, nil)
}

func (c *Controller) ReleaseHold(ctx *gin.Context) {
	holdID := ctx.Param("holdId")
	if holdID == "" {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Hold ID is required", nil, "missing hold ID")
		return
	}

	err := c.service.ReleaseHold(ctx.Request.Context(), holdID)
	if err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Failed to release hold", nil, err.Error())
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "Hold released successfully", nil, nil)
}

func (c *Controller) ValidateHold(ctx *gin.Context) {
	holdID := ctx.Param("holdId")
	if holdID == "" {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Hold ID is required", nil, "missing hold ID")
		return
	}

	userID := ctx.Query("user_id")
	if userID == "" {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "User ID is required", nil, "missing user_id parameter")
		return
	}

	result, err := c.service.ValidateHold(ctx.Request.Context(), holdID, userID)
	if err != nil {
		response.RespondJSON(ctx, "error", http.StatusInternalServerError, "Failed to validate hold", nil, err.Error())
		return
	}

	if !result.Valid {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Invalid hold", result, result.Reason)
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "Hold is valid", result, nil)
}

func (c *Controller) GetUserHolds(ctx *gin.Context) {
	userID := ctx.Param("userId")
	if userID == "" {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "User ID is required", nil, "missing user ID")
		return
	}

	holds, err := c.service.GetUserHolds(ctx.Request.Context(), userID)
	if err != nil {
		response.RespondJSON(ctx, "error", http.StatusInternalServerError, "Failed to get user holds", nil, err.Error())
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "User holds retrieved successfully", holds, nil)
}

//  AVAILABILITY CHECKS

func (c *Controller) CheckSeatAvailability(ctx *gin.Context) {
	var req struct {
		SeatIDs []string `json:"seat_ids" binding:"required,min=1"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Invalid request data", nil, err.Error())
		return
	}

	availability, err := c.service.CheckSeatAvailability(ctx.Request.Context(), req.SeatIDs)
	if err != nil {
		response.RespondJSON(ctx, "error", http.StatusInternalServerError, "Failed to check availability", nil, err.Error())
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "Seat availability checked successfully", availability, nil)
}

func (c *Controller) GetAvailableSeatsInSection(ctx *gin.Context) {
	sectionID := ctx.Param("sectionId")
	if sectionID == "" {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Section ID is required", nil, "missing section ID")
		return
	}

	// Get event ID from query parameter - required for event-specific availability
	eventID := ctx.Query("event_id")
	if eventID == "" {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Event ID is required", nil, "missing event_id query parameter")
		return
	}

	seats, err := c.service.GetAvailableSeatsInSectionForEvent(ctx.Request.Context(), sectionID, eventID)
	if err != nil {
		response.RespondJSON(ctx, "error", http.StatusInternalServerError, "Failed to get available seats", nil, err.Error())
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "Available seats retrieved successfully", seats, nil)
}
