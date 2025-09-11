package venues

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"evently/internal/seats"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Service interface for venue business logic
type Service interface {
	// Venue Templates
	CreateTemplate(ctx context.Context, req CreateTemplateRequest) (*VenueTemplate, error)
	GetTemplateByID(ctx context.Context, id string) (*VenueTemplate, error)
	GetTemplates(ctx context.Context, filters TemplateFilters) (*PaginatedTemplates, error)
	UpdateTemplate(ctx context.Context, id string, req UpdateTemplateRequest) (*VenueTemplate, error)
	DeleteTemplate(ctx context.Context, id string) error

	// Venue Sections (Fixed per template)
	CreateSection(ctx context.Context, templateID string, req CreateSectionRequest) (*VenueSection, error)
	GetSectionsByTemplateID(ctx context.Context, templateID string) ([]VenueSection, error)
	GetSectionsByEventID(ctx context.Context, eventID string) ([]VenueSection, error)
	UpdateSection(ctx context.Context, id string, req UpdateSectionRequest) (*VenueSection, error)
	DeleteSection(ctx context.Context, id string) error

	// Event Pricing (Per event-section combination)
	CreateEventPricing(ctx context.Context, req CreateEventPricingRequest) (*EventPricingResponse, error)
	GetEventPricingByEventID(ctx context.Context, eventID string) ([]EventPricingResponse, error)
	UpdateEventPricing(ctx context.Context, id string, req UpdateEventPricingRequest) (*EventPricingResponse, error)
	DeleteEventPricing(ctx context.Context, id string) error
	DeleteEventPricingByEventID(ctx context.Context, eventID string) error

	// Venue Layout for Events
	GetVenueLayout(ctx context.Context, eventID string) (*VenueLayoutResponse, error)
}

// service implements Service interface
type service struct {
	repo     Repository
	seatRepo seats.Repository
}

// NewService creates a new venue service
func NewService(repo Repository, seatRepo seats.Repository) Service {
	return &service{
		repo:     repo,
		seatRepo: seatRepo,
	}
}

// ============= VENUE TEMPLATES =============

func (s *service) CreateTemplate(ctx context.Context, req CreateTemplateRequest) (*VenueTemplate, error) {
	// Validate template name uniqueness
	existing, err := s.repo.GetTemplateByName(ctx, req.Name)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check template name: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("template with name '%s' already exists", req.Name)
	}

	// Validate layout type
	validLayouts := map[string]bool{
		"THEATER":    true,
		"STADIUM":    true,
		"CONFERENCE": true,
		"GENERAL":    true,
	}
	if !validLayouts[req.LayoutType] {
		return nil, fmt.Errorf("invalid layout type: %s", req.LayoutType)
	}

	template := &VenueTemplate{
		ID:                 uuid.New(),
		Name:               req.Name,
		Description:        req.Description,
		DefaultRows:        req.DefaultRows,
		DefaultSeatsPerRow: req.DefaultSeatsPerRow,
		LayoutType:         req.LayoutType,
	}

	if err := s.repo.CreateTemplate(ctx, template); err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	return template, nil
}

func (s *service) GetTemplateByID(ctx context.Context, id string) (*VenueTemplate, error) {
	templateID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid template ID: %w", err)
	}

	template, err := s.repo.GetTemplateByID(ctx, templateID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("template not found")
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	return template, nil
}

func (s *service) GetTemplates(ctx context.Context, filters TemplateFilters) (*PaginatedTemplates, error) {
	// Set default pagination
	if filters.Page <= 0 {
		filters.Page = 1
	}
	if filters.Limit <= 0 {
		filters.Limit = 20
	}
	if filters.Limit > 100 {
		filters.Limit = 100
	}

	return s.repo.GetTemplates(ctx, filters)
}

func (s *service) UpdateTemplate(ctx context.Context, id string, req UpdateTemplateRequest) (*VenueTemplate, error) {
	templateID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid template ID: %w", err)
	}

	// Check if template exists
	existing, err := s.repo.GetTemplateByID(ctx, templateID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("template not found")
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	updates := make(map[string]interface{})

	if req.Name != nil {
		// Check name uniqueness if changing name
		if *req.Name != existing.Name {
			nameExists, err := s.repo.GetTemplateByName(ctx, *req.Name)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("failed to check template name: %w", err)
			}
			if nameExists != nil {
				return nil, fmt.Errorf("template with name '%s' already exists", *req.Name)
			}
		}
		updates["name"] = *req.Name
	}

	if req.Description != nil {
		updates["description"] = *req.Description
	}

	if req.DefaultRows != nil {
		updates["default_rows"] = *req.DefaultRows
	}

	if req.DefaultSeatsPerRow != nil {
		updates["default_seats_per_row"] = *req.DefaultSeatsPerRow
	}

	if req.LayoutType != nil {
		validLayouts := map[string]bool{
			"THEATER":    true,
			"STADIUM":    true,
			"CONFERENCE": true,
			"GENERAL":    true,
		}
		if !validLayouts[*req.LayoutType] {
			return nil, fmt.Errorf("invalid layout type: %s", *req.LayoutType)
		}
		updates["layout_type"] = *req.LayoutType
	}

	if len(updates) > 0 {
		if err := s.repo.UpdateTemplate(ctx, templateID, updates); err != nil {
			return nil, fmt.Errorf("failed to update template: %w", err)
		}
	}

	// Return updated template
	return s.repo.GetTemplateByID(ctx, templateID)
}

func (s *service) DeleteTemplate(ctx context.Context, id string) error {
	templateID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid template ID: %w", err)
	}

	// Check if template exists
	_, err = s.repo.GetTemplateByID(ctx, templateID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("template not found")
		}
		return fmt.Errorf("failed to get template: %w", err)
	}

	if err := s.repo.DeleteTemplate(ctx, templateID); err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	return nil
}

// ============= VENUE SECTIONS =============

func (s *service) CreateSection(ctx context.Context, templateID string, req CreateSectionRequest) (*VenueSection, error) {
	templateUUID, err := uuid.Parse(templateID)
	if err != nil {
		return nil, fmt.Errorf("invalid template ID: %w", err)
	}

	// Validate template exists
	_, err = s.repo.GetTemplateByID(ctx, templateUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("template not found")
		}
		return nil, fmt.Errorf("failed to validate template: %w", err)
	}

	section := &VenueSection{
		TemplateID:  templateUUID,
		Name:        req.Name,
		Description: req.Description,
		RowStart:    req.RowStart,
		RowEnd:      req.RowEnd,
		SeatsPerRow: req.SeatsPerRow,
		TotalSeats:  req.TotalSeats,
	}

	if err := s.repo.CreateSection(ctx, section); err != nil {
		return nil, fmt.Errorf("failed to create section: %w", err)
	}

	// Auto-generate seats for the section
	if err := s.generateSeatsForSection(ctx, section); err != nil {
		// If seat generation fails, we might want to rollback the section creation
		// For now, just log the error and continue
		return nil, fmt.Errorf("failed to generate seats for section: %w", err)
	}

	return section, nil
}

func (s *service) GetSectionsByTemplateID(ctx context.Context, templateID string) ([]VenueSection, error) {
	templateUUID, err := uuid.Parse(templateID)
	if err != nil {
		return nil, fmt.Errorf("invalid template ID: %w", err)
	}

	return s.repo.GetSectionsByTemplateID(ctx, templateUUID)
}

func (s *service) GetSectionsByEventID(ctx context.Context, eventID string) ([]VenueSection, error) {
	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		return nil, fmt.Errorf("invalid event ID: %w", err)
	}

	// Get the venue layout which contains the sections (this internally gets the template ID)
	layout, err := s.repo.GetVenueLayoutForEvent(ctx, eventUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get venue layout: %w", err)
	}

	// Extract sections from the layout and convert to VenueSection format
	sections := make([]VenueSection, len(layout.Sections))
	for i, sectionResp := range layout.Sections {
		sectionUUID, err := uuid.Parse(sectionResp.ID)
		if err != nil {
			return nil, fmt.Errorf("invalid section ID in layout: %w", err)
		}

		// Get the full section details from repository
		section, err := s.repo.GetSectionByID(ctx, sectionUUID)
		if err != nil {
			return nil, fmt.Errorf("failed to get section details: %w", err)
		}

		sections[i] = *section
	}

	return sections, nil
}

func (s *service) GetVenueLayout(ctx context.Context, eventID string) (*VenueLayoutResponse, error) {
	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		return nil, fmt.Errorf("invalid event ID: %w", err)
	}

	return s.repo.GetVenueLayoutForEvent(ctx, eventUUID)
}

func (s *service) UpdateSection(ctx context.Context, id string, req UpdateSectionRequest) (*VenueSection, error) {
	sectionID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid section ID: %w", err)
	}

	// Check if section exists
	_, err = s.repo.GetSectionByID(ctx, sectionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("section not found")
		}
		return nil, fmt.Errorf("failed to get section: %w", err)
	}

	updates := make(map[string]interface{})

	if req.Name != nil {
		updates["name"] = *req.Name
	}

	if req.Description != nil {
		updates["description"] = *req.Description
	}

	if req.RowStart != nil {
		updates["row_start"] = *req.RowStart
	}

	if req.RowEnd != nil {
		updates["row_end"] = *req.RowEnd
	}

	if req.SeatsPerRow != nil {
		if *req.SeatsPerRow <= 0 {
			return nil, fmt.Errorf("seats per row must be greater than 0")
		}
		updates["seats_per_row"] = *req.SeatsPerRow
	}

	if req.TotalSeats != nil {
		if *req.TotalSeats <= 0 {
			return nil, fmt.Errorf("total seats must be greater than 0")
		}
		updates["total_seats"] = *req.TotalSeats
	}

	if len(updates) > 0 {
		if err := s.repo.UpdateSection(ctx, sectionID, updates); err != nil {
			return nil, fmt.Errorf("failed to update section: %w", err)
		}
	}

	// Return updated section
	return s.repo.GetSectionByID(ctx, sectionID)
}

func (s *service) DeleteSection(ctx context.Context, id string) error {
	sectionID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid section ID: %w", err)
	}

	// Check if section exists
	_, err = s.repo.GetSectionByID(ctx, sectionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("section not found")
		}
		return fmt.Errorf("failed to get section: %w", err)
	}

	if err := s.repo.DeleteSection(ctx, sectionID); err != nil {
		return fmt.Errorf("failed to delete section: %w", err)
	}

	return nil
}

// ============= EVENT PRICING =============

func (s *service) CreateEventPricing(ctx context.Context, req CreateEventPricingRequest) (*EventPricingResponse, error) {
	eventID, err := uuid.Parse(req.EventID)
	if err != nil {
		return nil, fmt.Errorf("invalid event ID: %w", err)
	}

	sectionID, err := uuid.Parse(req.SectionID)
	if err != nil {
		return nil, fmt.Errorf("invalid section ID: %w", err)
	}

	// Check if pricing already exists for this event-section combination
	existing, err := s.repo.GetEventPricing(ctx, eventID, sectionID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check existing pricing: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("pricing already exists for this event-section combination")
	}

	pricing := &EventPricing{
		ID:              uuid.New(),
		EventID:         eventID,
		SectionID:       sectionID,
		PriceMultiplier: req.PriceMultiplier,
		IsActive:        true,
	}

	if err := s.repo.CreateEventPricing(ctx, pricing); err != nil {
		return nil, fmt.Errorf("failed to create event pricing: %w", err)
	}

	// Get section name for response
	section, err := s.repo.GetSectionByID(ctx, sectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get section: %w", err)
	}

	response := pricing.ToResponse(section.Name, 0) // Base price will be set by caller
	return &response, nil
}

func (s *service) GetEventPricingByEventID(ctx context.Context, eventID string) ([]EventPricingResponse, error) {
	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		return nil, fmt.Errorf("invalid event ID: %w", err)
	}

	pricings, err := s.repo.GetEventPricingByEventID(ctx, eventUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get event pricing: %w", err)
	}

	responses := make([]EventPricingResponse, len(pricings))
	for i, pricing := range pricings {
		sectionName := ""
		if pricing.Section != nil {
			sectionName = pricing.Section.Name
		}
		responses[i] = pricing.ToResponse(sectionName, 0) // Base price will be set by caller
	}

	return responses, nil
}

func (s *service) UpdateEventPricing(ctx context.Context, id string, req UpdateEventPricingRequest) (*EventPricingResponse, error) {
	pricingID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid pricing ID: %w", err)
	}

	updates := make(map[string]interface{})

	if req.PriceMultiplier != nil {
		updates["price_multiplier"] = *req.PriceMultiplier
	}

	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if len(updates) > 0 {
		if err := s.repo.UpdateEventPricing(ctx, pricingID, updates); err != nil {
			return nil, fmt.Errorf("failed to update pricing: %w", err)
		}
	}

	// Get updated pricing for response
	eventUUID := uuid.Nil // We need to get this from the pricing record
	pricings, err := s.repo.GetEventPricingByEventID(ctx, eventUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated pricing: %w", err)
	}

	// Find the updated pricing (simplified for now)
	for _, pricing := range pricings {
		if pricing.ID.String() == id {
			sectionName := ""
			if pricing.Section != nil {
				sectionName = pricing.Section.Name
			}
			response := pricing.ToResponse(sectionName, 0)
			return &response, nil
		}
	}

	return nil, fmt.Errorf("updated pricing not found")
}

func (s *service) DeleteEventPricing(ctx context.Context, id string) error {
	pricingID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid pricing ID: %w", err)
	}

	if err := s.repo.DeleteEventPricing(ctx, pricingID); err != nil {
		return fmt.Errorf("failed to delete pricing: %w", err)
	}

	return nil
}

func (s *service) DeleteEventPricingByEventID(ctx context.Context, eventID string) error {
	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		return fmt.Errorf("invalid event ID: %w", err)
	}

	if err := s.repo.DeleteEventPricingByEventID(ctx, eventUUID); err != nil {
		return fmt.Errorf("failed to delete event pricing: %w", err)
	}

	return nil
}

// ============= HELPER FUNCTIONS =============

// generateSeatsForSection automatically creates seats for a venue section
func (s *service) generateSeatsForSection(ctx context.Context, section *VenueSection) error {
	if section.RowStart == "" || section.RowEnd == "" {
		return fmt.Errorf("row start and end must be specified for seat generation")
	}

	// Generate row labels (A-Z or numeric)
	rows, err := s.generateRowLabels(section.RowStart, section.RowEnd)
	if err != nil {
		return fmt.Errorf("failed to generate row labels: %w", err)
	}

	var seatsToCreate []seats.Seat
	position := 1

	// Generate seats for each row
	for _, row := range rows {
		for seatNum := 1; seatNum <= section.SeatsPerRow; seatNum++ {
			seat := seats.Seat{
				ID:         uuid.New(),
				SectionID:  section.ID,
				SeatNumber: fmt.Sprintf("%s%d", row, seatNum),
				Row:        row,
				Position:   position,
				Status:     "AVAILABLE",
			}
			seatsToCreate = append(seatsToCreate, seat)
			position++
		}
	}

	// Validate total seats match
	if len(seatsToCreate) != section.TotalSeats {
		return fmt.Errorf("generated seat count (%d) doesn't match section total (%d)", 
			len(seatsToCreate), section.TotalSeats)
	}

	// Create all seats in batch
	return s.seatRepo.CreateSeats(ctx, seatsToCreate)
}

// generateRowLabels creates row labels between start and end
func (s *service) generateRowLabels(start, end string) ([]string, error) {
	var rows []string

	// Check if numeric rows (1, 2, 3...) or alphabetic (A, B, C...)
	if startNum, err := strconv.Atoi(start); err == nil {
		// Numeric rows
		endNum, err := strconv.Atoi(end)
		if err != nil {
			return nil, fmt.Errorf("inconsistent row format: start is numeric but end is not")
		}
		
		if startNum > endNum {
			return nil, fmt.Errorf("start row (%d) cannot be greater than end row (%d)", startNum, endNum)
		}

		for i := startNum; i <= endNum; i++ {
			rows = append(rows, strconv.Itoa(i))
		}
	} else {
		// Alphabetic rows
		if len(start) != 1 || len(end) != 1 {
			return nil, fmt.Errorf("alphabetic rows must be single characters")
		}

		startChar := start[0]
		endChar := end[0]

		if startChar > endChar {
			return nil, fmt.Errorf("start row (%s) cannot be greater than end row (%s)", start, end)
		}

		for c := startChar; c <= endChar; c++ {
			rows = append(rows, string(c))
		}
	}

	return rows, nil
}
