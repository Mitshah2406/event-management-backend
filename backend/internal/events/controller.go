package events

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"evently/internal/shared/utils/response"
)

type Controller interface {
	CreateEvent(c *gin.Context)
	GetEvent(c *gin.Context)
	UpdateEvent(c *gin.Context)
	DeleteEvent(c *gin.Context)
	GetAllEvents(c *gin.Context)
	GetEventAnalytics(c *gin.Context)
	GetAllEventAnalytics(c *gin.Context)
	GetUpcomingEvents(c *gin.Context)
}

type controller struct {
	service Service
}

func NewController(service Service) Controller {
	return &controller{service: service}
}

func (ctrl *controller) CreateEvent(c *gin.Context) {
	var req CreateEventRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespondJSON(c, "error", http.StatusBadRequest, "Invalid request body", nil, err.Error())
		return
	}

	// Get admin user ID from context (set by auth middleware)
	adminID, exists := c.Get("user_id")
	if !exists {
		response.RespondJSON(c, "error", http.StatusUnauthorized, "Admin not authenticated", nil, nil)
		return
	}

	adminUUID, err := uuid.Parse(adminID.(string))
	if err != nil {
		response.RespondJSON(c, "error", http.StatusInternalServerError, "Invalid admin ID format", nil, nil)
		return
	}

	event, err := ctrl.service.CreateEvent(adminUUID, req)
	if err != nil {
		response.RespondJSON(c, "error", http.StatusBadRequest, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusCreated, "Event created successfully", event, nil)
}

func (ctrl *controller) GetEvent(c *gin.Context) {
	eventIDStr := c.Param("eventId")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		response.RespondJSON(c, "error", http.StatusBadRequest, "Invalid event ID", nil, err.Error())
		return
	}

	event, err := ctrl.service.GetEventByID(eventID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "event not found" {
			statusCode = http.StatusNotFound
		}
		response.RespondJSON(c, "error", statusCode, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusOK, "Event retrieved successfully", event, nil)
}

func (ctrl *controller) UpdateEvent(c *gin.Context) {
	eventIDStr := c.Param("eventId")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		response.RespondJSON(c, "error", http.StatusBadRequest, "Invalid event ID", nil, err.Error())
		return
	}

	var req UpdateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespondJSON(c, "error", http.StatusBadRequest, "Invalid request body", nil, err.Error())
		return
	}

	// Get admin ID from context
	adminID, exists := c.Get("user_id")
	if !exists {
		response.RespondJSON(c, "error", http.StatusUnauthorized, "Admin not authenticated", nil, nil)
		return
	}

	adminUUID, err := uuid.Parse(adminID.(string))
	if err != nil {
		response.RespondJSON(c, "error", http.StatusInternalServerError, "Invalid admin ID format", nil, nil)
		return
	}

	// Admin can update any event
	event, err := ctrl.service.UpdateEventAsAdmin(eventID, adminUUID, req)
	if err != nil {
		statusCode := http.StatusBadRequest
		if err.Error() == "event not found" {
			statusCode = http.StatusNotFound
		}
		response.RespondJSON(c, "error", statusCode, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusOK, "Event updated successfully", event, nil)
}

func (ctrl *controller) DeleteEvent(c *gin.Context) {
	eventIDStr := c.Param("eventId")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		response.RespondJSON(c, "error", http.StatusBadRequest, "Invalid event ID", nil, err.Error())
		return
	}

	// Get admin ID from context
	adminID, exists := c.Get("user_id")
	if !exists {
		response.RespondJSON(c, "error", http.StatusUnauthorized, "Admin not authenticated", nil, nil)
		return
	}

	adminUUID, err := uuid.Parse(adminID.(string))
	if err != nil {
		response.RespondJSON(c, "error", http.StatusInternalServerError, "Invalid admin ID format", nil, nil)
		return
	}

	// Admin can delete any event
	err = ctrl.service.DeleteEventAsAdmin(eventID, adminUUID)
	if err != nil {
		statusCode := http.StatusBadRequest
		if err.Error() == "event not found" {
			statusCode = http.StatusNotFound
		}
		response.RespondJSON(c, "error", statusCode, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusOK, "Event deleted successfully", nil, nil)
}

func (ctrl *controller) GetAllEvents(c *gin.Context) {
	var query EventListQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		response.RespondJSON(c, "error", http.StatusBadRequest, "Invalid query parameters", nil, err.Error())
		return
	}

	events, err := ctrl.service.GetAllEvents(query)
	if err != nil {
		response.RespondJSON(c, "error", http.StatusInternalServerError, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusOK, "Events retrieved successfully", events, nil)
}

func (ctrl *controller) GetEventAnalytics(c *gin.Context) {
	eventIDStr := c.Param("eventId")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		response.RespondJSON(c, "error", http.StatusBadRequest, "Invalid event ID", nil, err.Error())
		return
	}

	analytics, err := ctrl.service.GetEventAnalyticsAsAdmin(eventID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "event not found" {
			statusCode = http.StatusNotFound
		}
		response.RespondJSON(c, "error", statusCode, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusOK, "Event analytics retrieved successfully", analytics, nil)
}

func (ctrl *controller) GetAllEventAnalytics(c *gin.Context) {
	analytics, err := ctrl.service.GetAllEventAnalyticsAsAdmin()
	if err != nil {
		response.RespondJSON(c, "error", http.StatusInternalServerError, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusOK, "All event analytics retrieved successfully", analytics, nil)
}

func (ctrl *controller) GetUpcomingEvents(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	events, err := ctrl.service.GetUpcomingEvents(limit)
	if err != nil {
		response.RespondJSON(c, "error", http.StatusInternalServerError, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusOK, "Upcoming events retrieved successfully", events, nil)
}
