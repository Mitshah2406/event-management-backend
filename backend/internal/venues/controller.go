package venues

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

//  VENUE TEMPLATES

func (c *Controller) CreateTemplate(ctx *gin.Context) {
	var req CreateTemplateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Invalid request data", nil, err.Error())
		return
	}

	template, err := c.service.CreateTemplate(ctx.Request.Context(), req)
	if err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Failed to create template", nil, err.Error())
		return
	}

	response.RespondJSON(ctx, "success", http.StatusCreated, "Template created successfully", template, nil)
}

func (c *Controller) GetTemplate(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Template ID is required", nil, "missing template ID")
		return
	}

	template, err := c.service.GetTemplateByID(ctx.Request.Context(), id)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "template not found" {
			statusCode = http.StatusNotFound
		}
		response.RespondJSON(ctx, "error", statusCode, "Failed to get template", nil, err.Error())
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "Template retrieved successfully", template, nil)
}

func (c *Controller) GetTemplates(ctx *gin.Context) {
	var filters TemplateFilters
	if err := ctx.ShouldBindQuery(&filters); err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Invalid query parameters", nil, err.Error())
		return
	}

	result, err := c.service.GetTemplates(ctx.Request.Context(), filters)
	if err != nil {
		response.RespondJSON(ctx, "error", http.StatusInternalServerError, "Failed to get templates", nil, err.Error())
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "Templates retrieved successfully", result, nil)
}

func (c *Controller) UpdateTemplate(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Template ID is required", nil, "missing template ID")
		return
	}

	var req UpdateTemplateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Invalid request data", nil, err.Error())
		return
	}

	template, err := c.service.UpdateTemplate(ctx.Request.Context(), id, req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "template not found" {
			statusCode = http.StatusNotFound
		}
		response.RespondJSON(ctx, "error", statusCode, "Failed to update template", nil, err.Error())
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "Template updated successfully", template, nil)
}

func (c *Controller) DeleteTemplate(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Template ID is required", nil, "missing template ID")
		return
	}

	err := c.service.DeleteTemplate(ctx.Request.Context(), id)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "template not found" {
			statusCode = http.StatusNotFound
		}
		response.RespondJSON(ctx, "error", statusCode, "Failed to delete template", nil, err.Error())
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "Template deleted successfully", nil, nil)
}

// VENUE SECTIONS

func (c *Controller) CreateSection(ctx *gin.Context) {
	templateID := ctx.Param("id")
	if templateID == "" {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Template ID is required", nil, "missing template ID")
		return
	}

	var req CreateSectionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Invalid request data", nil, err.Error())
		return
	}

	section, err := c.service.CreateSection(ctx.Request.Context(), templateID, req)
	if err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Failed to create section", nil, err.Error())
		return
	}

	response.RespondJSON(ctx, "success", http.StatusCreated, "Section created successfully", section, nil)
}

func (c *Controller) GetSectionsByEventID(ctx *gin.Context) {
	eventID := ctx.Param("eventId")
	if eventID == "" {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Event ID is required", nil, "missing event ID")
		return
	}

	sections, err := c.service.GetSectionsByEventID(ctx.Request.Context(), eventID)
	if err != nil {
		response.RespondJSON(ctx, "error", http.StatusInternalServerError, "Failed to get sections", nil, err.Error())
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "Sections retrieved successfully", sections, nil)
}

func (c *Controller) GetSectionsByTemplateID(ctx *gin.Context) {
	templateID := ctx.Param("id")
	if templateID == "" {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Template ID is required", nil, "missing template ID")
		return
	}

	sections, err := c.service.GetSectionsByTemplateID(ctx.Request.Context(), templateID)
	if err != nil {
		response.RespondJSON(ctx, "error", http.StatusInternalServerError, "Failed to get sections", nil, err.Error())
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "Sections retrieved successfully", sections, nil)
}

func (c *Controller) GetVenueLayout(ctx *gin.Context) {
	eventID := ctx.Param("eventId")
	if eventID == "" {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Event ID is required", nil, "missing event ID")
		return
	}

	layout, err := c.service.GetVenueLayout(ctx.Request.Context(), eventID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "no venue sections found for event" {
			statusCode = http.StatusNotFound
		}
		response.RespondJSON(ctx, "error", statusCode, "Failed to get venue layout", nil, err.Error())
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "Venue layout retrieved successfully", layout, nil)
}

func (c *Controller) UpdateSection(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Section ID is required", nil, "missing section ID")
		return
	}

	var req UpdateSectionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Invalid request data", nil, err.Error())
		return
	}

	section, err := c.service.UpdateSection(ctx.Request.Context(), id, req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "section not found" {
			statusCode = http.StatusNotFound
		}
		response.RespondJSON(ctx, "error", statusCode, "Failed to update section", nil, err.Error())
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "Section updated successfully", section, nil)
}

func (c *Controller) DeleteSection(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Section ID is required", nil, "missing section ID")
		return
	}

	err := c.service.DeleteSection(ctx.Request.Context(), id)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "section not found" {
			statusCode = http.StatusNotFound
		}
		response.RespondJSON(ctx, "error", statusCode, "Failed to delete section", nil, err.Error())
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "Section deleted successfully", nil, nil)
}
