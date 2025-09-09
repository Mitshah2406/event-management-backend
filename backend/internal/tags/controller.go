package tags

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"evently/internal/shared/utils/response"
)

type Controller interface {
	// Admin CRUD operations
	CreateTag(c *gin.Context)
	GetTag(c *gin.Context)
	GetTagBySlug(c *gin.Context)
	UpdateTag(c *gin.Context)
	DeleteTag(c *gin.Context)
	GetAllTags(c *gin.Context)
	GetActiveTags(c *gin.Context)

	// Analytics operations (admin only)
	GetTagAnalytics(c *gin.Context)
	GetTagPopularityAnalytics(c *gin.Context)
	GetTagTrends(c *gin.Context)
	GetTagComparisons(c *gin.Context)
}

type controller struct {
	service Service
}

func NewController(service Service) Controller {
	return &controller{service: service}
}

// Admin CRUD operations

func (ctrl *controller) CreateTag(c *gin.Context) {
	var req CreateTagRequest

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

	tag, err := ctrl.service.CreateTag(adminUUID, req)
	if err != nil {
		statusCode := http.StatusBadRequest
		if err.Error() == "a tag with similar name already exists" {
			statusCode = http.StatusConflict
		}
		response.RespondJSON(c, "error", statusCode, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusCreated, "Tag created successfully", tag, nil)
}

func (ctrl *controller) GetTag(c *gin.Context) {
	tagIDStr := c.Param("id")
	tagID, err := uuid.Parse(tagIDStr)
	if err != nil {
		response.RespondJSON(c, "error", http.StatusBadRequest, "Invalid tag ID", nil, err.Error())
		return
	}

	tag, err := ctrl.service.GetTagByID(tagID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "tag not found" {
			statusCode = http.StatusNotFound
		}
		response.RespondJSON(c, "error", statusCode, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusOK, "Tag retrieved successfully", tag, nil)
}

func (ctrl *controller) GetTagBySlug(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		response.RespondJSON(c, "error", http.StatusBadRequest, "Tag slug is required", nil, nil)
		return
	}

	tag, err := ctrl.service.GetTagBySlug(slug)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "tag not found" {
			statusCode = http.StatusNotFound
		}
		response.RespondJSON(c, "error", statusCode, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusOK, "Tag retrieved successfully", tag, nil)
}

func (ctrl *controller) UpdateTag(c *gin.Context) {
	tagIDStr := c.Param("id")
	tagID, err := uuid.Parse(tagIDStr)
	if err != nil {
		response.RespondJSON(c, "error", http.StatusBadRequest, "Invalid tag ID", nil, err.Error())
		return
	}
	log.Default().Println("Updating tag with ID:", tagID)
	var req UpdateTagRequest
	log.Default().Println("Updating tag with ID:", req)
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

	tag, err := ctrl.service.UpdateTag(tagID, adminUUID, req)
	if err != nil {
		statusCode := http.StatusBadRequest
		if err.Error() == "tag not found" {
			statusCode = http.StatusNotFound
		} else if err.Error() == "a tag with similar name already exists" {
			statusCode = http.StatusConflict
		}
		response.RespondJSON(c, "error", statusCode, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusOK, "Tag updated successfully", tag, nil)
}

func (ctrl *controller) DeleteTag(c *gin.Context) {
	tagIDStr := c.Param("id")
	tagID, err := uuid.Parse(tagIDStr)
	if err != nil {
		response.RespondJSON(c, "error", http.StatusBadRequest, "Invalid tag ID", nil, err.Error())
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

	err = ctrl.service.DeleteTag(tagID, adminUUID)
	if err != nil {
		statusCode := http.StatusBadRequest
		if err.Error() == "tag not found" {
			statusCode = http.StatusNotFound
		}
		response.RespondJSON(c, "error", statusCode, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusOK, "Tag deleted successfully", nil, nil)
}

func (ctrl *controller) GetAllTags(c *gin.Context) {
	var query TagListQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		response.RespondJSON(c, "error", http.StatusBadRequest, "Invalid query parameters", nil, err.Error())
		return
	}

	tags, err := ctrl.service.GetAllTags(query)
	if err != nil {
		response.RespondJSON(c, "error", http.StatusInternalServerError, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusOK, "Tags retrieved successfully", tags, nil)
}

func (ctrl *controller) GetActiveTags(c *gin.Context) {
	tags, err := ctrl.service.GetActiveTags()
	if err != nil {
		response.RespondJSON(c, "error", http.StatusInternalServerError, err.Error(), nil, nil)
		return
	}

	response.RespondJSON(c, "success", http.StatusOK, "Active tags retrieved successfully", tags, nil)
}

// Analytics operations

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
