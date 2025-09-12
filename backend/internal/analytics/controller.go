package analytics

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"evently/internal/shared/utils/response"
)

// Controller defines the analytics controller interface
type Controller interface {
	// Dashboard Analytics
	GetDashboardAnalytics(c *gin.Context)

	// Event Analytics (migrated from events package)
	GetEventAnalytics(c *gin.Context)
	GetGlobalEventAnalytics(c *gin.Context)

	// Tag Analytics (migrated from tags package)
	GetTagAnalytics(c *gin.Context)
	GetTagPopularityAnalytics(c *gin.Context)
	GetTagTrends(c *gin.Context)
	GetTagComparisons(c *gin.Context)

	// Booking Analytics (new)
	GetBookingAnalytics(c *gin.Context)
	GetBookingDailyStats(c *gin.Context)
	GetCancellationAnalytics(c *gin.Context)

	// User Analytics (new)
	GetUserAnalytics(c *gin.Context)
	GetUserRetentionMetrics(c *gin.Context)
	GetUserDemographics(c *gin.Context)

	// User-facing Analytics
	GetUserBookingHistory(c *gin.Context)
	GetPersonalAnalytics(c *gin.Context)
}

// controller implements the Controller interface
type controller struct {
	service Service
}

// NewController creates a new analytics controller instance
func NewController(service Service) Controller {
	return &controller{service: service}
}

// Dashboard Analytics Implementation

func (ctrl *controller) GetDashboardAnalytics(c *gin.Context) {
	dashboard, err := ctrl.service.GetDashboardAnalytics()
	if err != nil {
		response.RespondJSON(c, "error", http.StatusInternalServerError, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusOK, "Dashboard analytics retrieved successfully", dashboard, nil)
}

// Event Analytics Implementation

func (ctrl *controller) GetEventAnalytics(c *gin.Context) {
	eventIDStr := c.Param("id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		response.RespondJSON(c, "error", http.StatusBadRequest, "Invalid event ID", nil, err.Error())
		return
	}

	analytics, err := ctrl.service.GetEventAnalytics(eventID)
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

func (ctrl *controller) GetGlobalEventAnalytics(c *gin.Context) {
	analytics, err := ctrl.service.GetGlobalEventAnalytics()
	if err != nil {
		response.RespondJSON(c, "error", http.StatusInternalServerError, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusOK, "Global event analytics retrieved successfully", analytics, nil)
}

// Tag Analytics Implementation

func (ctrl *controller) GetTagAnalytics(c *gin.Context) {
	analytics, err := ctrl.service.GetTagAnalytics()
	if err != nil {
		response.RespondJSON(c, "error", http.StatusInternalServerError, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusOK, "Tag analytics retrieved successfully", analytics, nil)
}

func (ctrl *controller) GetTagPopularityAnalytics(c *gin.Context) {
	analytics, err := ctrl.service.GetTagPopularityAnalytics()
	if err != nil {
		response.RespondJSON(c, "error", http.StatusInternalServerError, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusOK, "Tag popularity analytics retrieved successfully", analytics, nil)
}

func (ctrl *controller) GetTagTrends(c *gin.Context) {
	monthsStr := c.DefaultQuery("months", "6")
	months, err := strconv.Atoi(monthsStr)
	if err != nil || months <= 0 {
		months = 6 // Default to 6 months
	}

	trends, err := ctrl.service.GetTagTrends(months)
	if err != nil {
		response.RespondJSON(c, "error", http.StatusInternalServerError, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusOK, "Tag trends retrieved successfully", trends, nil)
}

func (ctrl *controller) GetTagComparisons(c *gin.Context) {
	comparisons, err := ctrl.service.GetTagComparisons()
	if err != nil {
		response.RespondJSON(c, "error", http.StatusInternalServerError, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusOK, "Tag comparisons retrieved successfully", comparisons, nil)
}

// Booking Analytics Implementation

func (ctrl *controller) GetBookingAnalytics(c *gin.Context) {
	analytics, err := ctrl.service.GetBookingAnalytics()
	if err != nil {
		response.RespondJSON(c, "error", http.StatusInternalServerError, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusOK, "Booking analytics retrieved successfully", analytics, nil)
}

func (ctrl *controller) GetBookingDailyStats(c *gin.Context) {
	stats, err := ctrl.service.GetBookingDailyStats()
	if err != nil {
		response.RespondJSON(c, "error", http.StatusInternalServerError, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusOK, "Daily booking statistics retrieved successfully", stats, nil)
}

func (ctrl *controller) GetCancellationAnalytics(c *gin.Context) {
	analytics, err := ctrl.service.GetCancellationAnalytics()
	if err != nil {
		response.RespondJSON(c, "error", http.StatusInternalServerError, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusOK, "Cancellation analytics retrieved successfully", analytics, nil)
}

// User Analytics Implementation

func (ctrl *controller) GetUserAnalytics(c *gin.Context) {
	analytics, err := ctrl.service.GetUserAnalytics()
	if err != nil {
		response.RespondJSON(c, "error", http.StatusInternalServerError, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusOK, "User analytics retrieved successfully", analytics, nil)
}

func (ctrl *controller) GetUserRetentionMetrics(c *gin.Context) {
	retention, err := ctrl.service.GetUserRetentionMetrics()
	if err != nil {
		response.RespondJSON(c, "error", http.StatusInternalServerError, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusOK, "User retention metrics retrieved successfully", retention, nil)
}

func (ctrl *controller) GetUserDemographics(c *gin.Context) {
	demographics, err := ctrl.service.GetUserDemographics()
	if err != nil {
		response.RespondJSON(c, "error", http.StatusInternalServerError, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusOK, "User demographics retrieved successfully", demographics, nil)
}

// User-facing Analytics Implementation

func (ctrl *controller) GetUserBookingHistory(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		response.RespondJSON(c, "error", http.StatusUnauthorized, "User not authenticated", nil, nil)
		return
	}

	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		response.RespondJSON(c, "error", http.StatusInternalServerError, "Invalid user ID format", nil, nil)
		return
	}

	history, err := ctrl.service.GetUserBookingHistory(userUUID)
	if err != nil {
		response.RespondJSON(c, "error", http.StatusInternalServerError, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusOK, "User booking history retrieved successfully", history, nil)
}

func (ctrl *controller) GetPersonalAnalytics(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		response.RespondJSON(c, "error", http.StatusUnauthorized, "User not authenticated", nil, nil)
		return
	}

	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		response.RespondJSON(c, "error", http.StatusInternalServerError, "Invalid user ID format", nil, nil)
		return
	}

	analytics, err := ctrl.service.GetPersonalAnalytics(userUUID)
	if err != nil {
		response.RespondJSON(c, "error", http.StatusInternalServerError, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusOK, "Personal analytics retrieved successfully", analytics, nil)
}

// Helper methods for validation and error handling

func (ctrl *controller) validateAdminAccess(c *gin.Context) bool {
	// Check if user has admin role (this would depend on your auth implementation)
	userRole, exists := c.Get("user_role")
	if !exists {
		response.RespondJSON(c, "error", http.StatusUnauthorized, "User role not found", nil, nil)
		return false
	}

	if userRole.(string) != "admin" {
		response.RespondJSON(c, "error", http.StatusForbidden, "Admin access required", nil, nil)
		return false
	}

	return true
}

func (ctrl *controller) validateUserAccess(c *gin.Context) (*uuid.UUID, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.RespondJSON(c, "error", http.StatusUnauthorized, "User not authenticated", nil, nil)
		return nil, false
	}

	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		response.RespondJSON(c, "error", http.StatusInternalServerError, "Invalid user ID format", nil, nil)
		return nil, false
	}

	return &userUUID, true
}

// Additional helper methods for query parameter validation

func (ctrl *controller) parsePositiveInt(value string, defaultValue int, maxValue int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return defaultValue
	}
	if maxValue > 0 && parsed > maxValue {
		return maxValue
	}
	return parsed
}

func (ctrl *controller) parseDateRange(c *gin.Context) (string, string, error) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// Add validation logic for date format if needed
	// For now, return as-is
	return startDate, endDate, nil
}
