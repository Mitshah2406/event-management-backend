package venues

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository interface for venue operations
type Repository interface {
	// Venue Templates
	CreateTemplate(ctx context.Context, template *VenueTemplate) error
	GetTemplateByID(ctx context.Context, id uuid.UUID) (*VenueTemplate, error)
	GetTemplates(ctx context.Context, filters TemplateFilters) (*PaginatedTemplates, error)
	UpdateTemplate(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error
	DeleteTemplate(ctx context.Context, id uuid.UUID) error
	GetTemplateByName(ctx context.Context, name string) (*VenueTemplate, error)

	// Venue Sections (Fixed per template)
	CreateSection(ctx context.Context, section *VenueSection) error
	GetSectionByID(ctx context.Context, id uuid.UUID) (*VenueSection, error)
	GetSectionsByTemplateID(ctx context.Context, templateID uuid.UUID) ([]VenueSection, error)
	GetSectionsWithSeats(ctx context.Context, templateID uuid.UUID) ([]VenueSection, error)
	UpdateSection(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error
	DeleteSection(ctx context.Context, id uuid.UUID) error
	DeleteSectionsByTemplateID(ctx context.Context, templateID uuid.UUID) error

	// Event Pricing (Per event-section combination)
	CreateEventPricing(ctx context.Context, pricing *EventPricing) error
	GetEventPricing(ctx context.Context, eventID uuid.UUID, sectionID uuid.UUID) (*EventPricing, error)
	GetEventPricingByEventID(ctx context.Context, eventID uuid.UUID) ([]EventPricing, error)
	UpdateEventPricing(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error
	DeleteEventPricing(ctx context.Context, id uuid.UUID) error
	DeleteEventPricingByEventID(ctx context.Context, eventID uuid.UUID) error

	// Get venue layout for an event (sections + pricing + seats)
	GetVenueLayoutForEvent(ctx context.Context, eventID uuid.UUID) (*VenueLayoutResponse, error)
}

// repository implements Repository interface
type repository struct {
	db *gorm.DB
}

// NewRepository creates a new venue repository
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// ============= VENUE TEMPLATES =============

func (r *repository) CreateTemplate(ctx context.Context, template *VenueTemplate) error {
	return r.db.WithContext(ctx).Create(template).Error
}

func (r *repository) GetTemplateByID(ctx context.Context, id uuid.UUID) (*VenueTemplate, error) {
	var template VenueTemplate
	err := r.db.WithContext(ctx).First(&template, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

func (r *repository) GetTemplateByName(ctx context.Context, name string) (*VenueTemplate, error) {
	var template VenueTemplate
	err := r.db.WithContext(ctx).First(&template, "name = ?", name).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

func (r *repository) GetTemplates(ctx context.Context, filters TemplateFilters) (*PaginatedTemplates, error) {
	var templates []VenueTemplate
	var total int64

	query := r.db.WithContext(ctx).Model(&VenueTemplate{})

	// Apply filters
	if filters.Search != "" {
		searchPattern := fmt.Sprintf("%%%s%%", filters.Search)
		query = query.Where("name ILIKE ? OR description ILIKE ?", searchPattern, searchPattern)
	}

	if filters.LayoutType != "" {
		query = query.Where("layout_type = ?", filters.LayoutType)
	}

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply sorting
	sortBy := filters.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortOrder := filters.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}
	query = query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))

	// Apply pagination
	offset := (filters.Page - 1) * filters.Limit
	if err := query.Offset(offset).Limit(filters.Limit).Find(&templates).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filters.Limit) - 1) / int64(filters.Limit))

	return &PaginatedTemplates{
		Templates:  templates,
		TotalCount: total,
		Page:       filters.Page,
		Limit:      filters.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *repository) UpdateTemplate(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&VenueTemplate{}).Where("id = ?", id).Updates(updates).Error
}

func (r *repository) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&VenueTemplate{}, "id = ?", id).Error
}

// ============= VENUE SECTIONS =============

func (r *repository) CreateSection(ctx context.Context, section *VenueSection) error {
	return r.db.WithContext(ctx).Create(section).Error
}

func (r *repository) GetSectionByID(ctx context.Context, id uuid.UUID) (*VenueSection, error) {
	var section VenueSection
	err := r.db.WithContext(ctx).Preload("Template").First(&section, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &section, nil
}

func (r *repository) GetSectionsByTemplateID(ctx context.Context, templateID uuid.UUID) ([]VenueSection, error) {
	var sections []VenueSection
	err := r.db.WithContext(ctx).
		Preload("Template").
		Where("template_id = ?", templateID).
		Order("name ASC").
		Find(&sections).Error
	return sections, err
}

func (r *repository) GetSectionsWithSeats(ctx context.Context, templateID uuid.UUID) ([]VenueSection, error) {
	var sections []VenueSection
	err := r.db.WithContext(ctx).
		Preload("Template").
		Preload("Seats", func(db *gorm.DB) *gorm.DB {
			return db.Order("row ASC, position ASC")
		}).
		Where("template_id = ?", templateID).
		Order("name ASC").
		Find(&sections).Error
	return sections, err
}

func (r *repository) UpdateSection(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&VenueSection{}).Where("id = ?", id).Updates(updates).Error
}

func (r *repository) DeleteSection(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&VenueSection{}, "id = ?", id).Error
}

func (r *repository) DeleteSectionsByTemplateID(ctx context.Context, templateID uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&VenueSection{}, "template_id = ?", templateID).Error
}

// ============= EVENT PRICING =============

func (r *repository) CreateEventPricing(ctx context.Context, pricing *EventPricing) error {
	return r.db.WithContext(ctx).Create(pricing).Error
}

func (r *repository) GetEventPricing(ctx context.Context, eventID uuid.UUID, sectionID uuid.UUID) (*EventPricing, error) {
	var pricing EventPricing
	err := r.db.WithContext(ctx).
		Preload("Section").
		Where("event_id = ? AND section_id = ?", eventID, sectionID).
		First(&pricing).Error
	if err != nil {
		return nil, err
	}
	return &pricing, nil
}

func (r *repository) GetEventPricingByEventID(ctx context.Context, eventID uuid.UUID) ([]EventPricing, error) {
	var pricing []EventPricing
	err := r.db.WithContext(ctx).
		Preload("Section").
		Where("event_id = ? AND is_active = true", eventID).
		Find(&pricing).Error
	return pricing, err
}

func (r *repository) UpdateEventPricing(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&EventPricing{}).Where("id = ?", id).Updates(updates).Error
}

func (r *repository) DeleteEventPricing(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&EventPricing{}, "id = ?", id).Error
}

func (r *repository) DeleteEventPricingByEventID(ctx context.Context, eventID uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&EventPricing{}, "event_id = ?", eventID).Error
}

// GetVenueLayoutForEvent returns the complete venue layout for an event
func (r *repository) GetVenueLayoutForEvent(ctx context.Context, eventID uuid.UUID) (*VenueLayoutResponse, error) {
	// First get the event details
	var event struct {
		ID              uuid.UUID `json:"id"`
		Name            string    `json:"name"`
		VenueTemplateID uuid.UUID `json:"venue_template_id"`
		BasePrice       float64   `json:"base_price"`
	}
	
	err := r.db.WithContext(ctx).
		Table("events").
		Select("id, name, venue_template_id, base_price").
		Where("id = ?", eventID).
		First(&event).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	// Get the venue template
	template, err := r.GetTemplateByID(ctx, event.VenueTemplateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get venue template: %w", err)
	}

	// Get sections for the template
	sections, err := r.GetSectionsWithSeats(ctx, event.VenueTemplateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sections: %w", err)
	}

	// Get event pricing for all sections
	eventPricing, err := r.GetEventPricingByEventID(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get event pricing: %w", err)
	}

	// Create pricing map for quick lookup
	pricingMap := make(map[uuid.UUID]float64)
	for _, pricing := range eventPricing {
		pricingMap[pricing.SectionID] = pricing.PriceMultiplier
	}

	// Build response
	layout := &VenueLayoutResponse{
		EventID:   event.ID.String(),
		EventName: event.Name,
		VenueInfo: VenueInfo{
			TemplateID:   template.ID.String(),
			TemplateName: template.Name,
			LayoutType:   template.LayoutType,
			Description:  template.Description,
		},
		BasePrice:      event.BasePrice,
		Sections:       []VenueSectionResponse{},
		TotalSeats:     0,
		AvailableSeats: 0,
	}

	// Process each section
	for _, section := range sections {
		priceMultiplier := pricingMap[section.ID]
		if priceMultiplier == 0 {
			priceMultiplier = 1.0 // Default if no pricing set
		}

		// Get booked seat IDs for this event and section
		bookedSeatIDs, err := r.getBookedSeatIDs(ctx, eventID, section.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get booked seats for section %s: %w", section.ID, err)
		}
		
		// Convert seats to response format using event-specific status
		seatResponses := make([]SeatResponse, len(section.Seats))
		availableInSection := 0
		
		for i, seat := range section.Seats {
			isHeld := false // TODO: Check Redis for holds when needed
			
			// Calculate event-specific effective status
		effectiveStatus := r.calculateEffectiveStatus(seat, bookedSeatIDs, isHeld)
			
			seatResponses[i] = SeatResponse{
				ID:         seat.ID.String(),
				SeatNumber: seat.SeatNumber,
				Row:        seat.Row,
				Position:   seat.Position,
				Status:     effectiveStatus, // Use event-specific status
				Price:      event.BasePrice * priceMultiplier,
				IsHeld:     isHeld,
			}

			if effectiveStatus == "AVAILABLE" {
				availableInSection++
			}
		}

		sectionResponse := section.ToResponseWithPricing(event.BasePrice, priceMultiplier, seatResponses)
		layout.Sections = append(layout.Sections, sectionResponse)
		layout.TotalSeats += section.TotalSeats
		layout.AvailableSeats += availableInSection
	}

	return layout, nil
}

// getBookedSeatIDs retrieves booked seat IDs for a specific event and section
func (r *repository) getBookedSeatIDs(ctx context.Context, eventID uuid.UUID, sectionID uuid.UUID) (map[uuid.UUID]bool, error) {
	var seatIDs []uuid.UUID
	
	// Query seat_bookings table for this event and section
	if err := r.db.WithContext(ctx).
		Table("seat_bookings sb").
		Joins("JOIN bookings b ON b.id = sb.booking_id").
		Where("b.event_id = ? AND sb.section_id = ? AND b.status != 'CANCELLED'", eventID, sectionID).
		Select("sb.seat_id").
		Find(&seatIDs).Error; err != nil {
		return nil, fmt.Errorf("failed to query booked seats: %w", err)
	}
	
	// Convert to map for efficient lookup
	bookedMap := make(map[uuid.UUID]bool)
	for _, seatID := range seatIDs {
		bookedMap[seatID] = true
	}
	
	return bookedMap, nil
}

// calculateEffectiveStatus determines the effective status of a seat for an event
func (r *repository) calculateEffectiveStatus(seat Seat, bookedSeatIDs map[uuid.UUID]bool, isHeld bool) string {
	// Check permanent seat status first
	if seat.Status == "BLOCKED" {
		return "BLOCKED"
	}
	
	// Check if held
	if isHeld {
		return "HELD"
	}
	
	// Check if booked for this event
	if bookedSeatIDs[seat.ID] {
		return "BOOKED"
	}
	
	// Default to available
	return "AVAILABLE"
}

// ============= FILTER STRUCTS =============

type TemplateFilters struct {
	Page       int    `form:"page" binding:"omitempty,min=1"`
	Limit      int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Search     string `form:"search"`
	LayoutType string `form:"layout_type" binding:"omitempty,oneof=THEATER STADIUM CONFERENCE GENERAL"`
	SortBy     string `form:"sort_by" binding:"omitempty,oneof=name created_at updated_at"`
	SortOrder  string `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

type PaginatedTemplates struct {
	Templates  []VenueTemplate `json:"templates"`
	TotalCount int64           `json:"total_count"`
	Page       int             `json:"page"`
	Limit      int             `json:"limit"`
	TotalPages int             `json:"total_pages"`
}
