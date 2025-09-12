package events

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"reflect"
	"strings"
	"time"

	"evently/internal/shared/constants"
	"evently/pkg/cache"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service interface {
	// Service dependency injection
	SetTagService(tagService TagService)
	SetVenueService(venueService VenueService)
	SetCacheService(cacheService cache.Service)
	CreateEvent(userID uuid.UUID, req CreateEventRequest) (*EventResponse, error)
	GetEventByID(id uuid.UUID) (*EventResponse, error)
	// Original methods for backward compatibility
	UpdateEvent(id uuid.UUID, userID uuid.UUID, req UpdateEventRequest) (*EventResponse, error)
	DeleteEvent(id uuid.UUID, userID uuid.UUID) error
	GetEventAnalytics(eventID uuid.UUID, userID uuid.UUID) (*EventAnalytics, error)
	// New admin methods
	UpdateEventAsAdmin(id uuid.UUID, adminID uuid.UUID, req UpdateEventRequest) (*EventResponse, error)
	DeleteEventAsAdmin(id uuid.UUID, adminID uuid.UUID) error
	GetEventAnalyticsAsAdmin(eventID uuid.UUID) (*EventAnalytics, error)
	GetAllEventAnalyticsAsAdmin() (*GlobalAnalytics, error)
	// Common methods
	GetAllEvents(query EventListQuery) (*PaginatedEvents, error)
	GetUpcomingEvents(limit int) ([]EventResponse, error)
	CheckEventAvailability(eventID uuid.UUID, seatCount int) (bool, error)
	IsEventInFuture(eventID uuid.UUID) (bool, error)
	GetEventCapacityData(eventID uuid.UUID) (totalCapacity, bookedCount, availableSeats int, err error)
}

type service struct {
	repo         Repository
	tagService   TagService
	venueService VenueService
	cacheService cache.Service
}

// TagService interface to avoid circular dependencies
type TagService interface {
	ReplaceEventTags(eventID uuid.UUID, tagNames []string) error
	GetTagsByEventID(eventID uuid.UUID) ([]TagResponse, error)
	GetTagsByNames(tagNames []string) ([]TagResponse, error) // Add this method
}

// VenueService interface to validate venue sections
// We use interface{} and type assertions to avoid circular dependencies
type VenueService interface {
	GetSectionsByTemplateID(ctx context.Context, templateID string) (interface{}, error)
}

func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) SetTagService(tagService TagService) {
	s.tagService = tagService
}

func (s *service) SetVenueService(venueService VenueService) {
	s.venueService = venueService
}

// SetCacheService injects the cache service dependency
func (s *service) SetCacheService(cacheService cache.Service) {
	s.cacheService = cacheService
}

// Cache helper methods
func (s *service) setCache(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if s.cacheService == nil {
		return nil // Skip caching if cache service is not available
	}

	return s.cacheService.Set(ctx, key, value, ttl)
}

func (s *service) getCache(ctx context.Context, key string, dest interface{}) error {
	if s.cacheService == nil {
		return fmt.Errorf("cache service not available")
	}

	return s.cacheService.Get(ctx, key, dest)
}

func (s *service) deleteCache(ctx context.Context, keys ...string) error {
	if s.cacheService == nil || len(keys) == 0 {
		return nil
	}

	for _, key := range keys {
		if err := s.cacheService.Delete(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

func (s *service) invalidateEventCache(ctx context.Context, eventID *uuid.UUID) error {
	if s.cacheService == nil {
		return nil
	}

	// Build patterns for cache invalidation
	patterns := []string{
		constants.PATTERN_INVALIDATE_EVENT_ALL,
	}

	// If specific event ID provided, also invalidate its specific caches
	if eventID != nil {
		patterns = append(patterns, constants.PATTERN_INVALIDATE_EVENT_DETAIL+eventID.String()+"*")
	}

	// Delete matching keys using pattern
	for _, pattern := range patterns {
		if err := s.cacheService.DeletePattern(ctx, pattern); err != nil {
			return err
		}
	}

	return nil
}

// Helper function to populate tags in event response
func (s *service) populateEventTags(response *EventResponse) error {
	if s.tagService == nil {
		return nil // No tag service available
	}

	eventID, err := uuid.Parse(response.ID)
	if err != nil {
		return err
	}

	tagResponses, err := s.tagService.GetTagsByEventID(eventID)
	if err != nil {
		return err
	}

	// Convert TagResponse to TagInfo
	tags := make([]TagInfo, len(tagResponses))
	for i, tag := range tagResponses {
		tags[i] = TagInfo{
			ID:    tag.ID,
			Name:  tag.Name,
			Slug:  tag.Slug,
			Color: tag.Color,
		}
	}

	response.Tags = tags
	return nil
}

// Helper function to populate capacity data in event response
func (s *service) populateEventCapacity(response *EventResponse) error {
	eventID, err := uuid.Parse(response.ID)
	if err != nil {
		return err
	}

	totalCapacity, bookedCount, availableSeats, err := s.GetEventCapacityData(eventID)
	if err != nil {
		// Don't fail the entire request if capacity data is unavailable
		// Just leave the fields as 0
		return nil
	}

	response.TotalCapacity = totalCapacity
	response.BookedCount = bookedCount
	response.AvailableTickets = availableSeats

	return nil
}

// validateTagsExist checks if all provided tag names exist in the database
func (s *service) validateTagsExist(tagNames []string) error {
	if s.tagService == nil {
		return errors.New("tag service not available")
	}

	// Clean and filter unique tag names
	uniqueNames := make(map[string]bool)
	var cleanNames []string

	for _, name := range tagNames {
		cleanName := strings.TrimSpace(name)
		if cleanName != "" && !uniqueNames[cleanName] {
			uniqueNames[cleanName] = true
			cleanNames = append(cleanNames, cleanName)
		}
	}
	// log
	log.Printf("Validating tags: %v", cleanNames)
	if len(cleanNames) == 0 {
		return nil // No tags to validate
	}

	// Get existing tags by names
	existingTags, err := s.tagService.GetTagsByNames(cleanNames)
	log.Printf("Existing tags found: %v", existingTags)
	if err != nil {
		return fmt.Errorf("failed to fetch tags: %w", err)
	}

	// Create a map of existing tag names
	existingTagNames := make(map[string]bool)
	for _, tag := range existingTags {
		existingTagNames[tag.Slug] = true
	}

	// Find missing tags
	var missingTags []string
	for _, name := range cleanNames {
		if !existingTagNames[name] {
			missingTags = append(missingTags, name)
		}
	}

	if len(missingTags) > 0 {
		return fmt.Errorf("the following tags do not exist: %v", missingTags)
	}

	return nil
}

// validateSectionsExist checks if all provided section IDs exist and belong to the venue template
func (s *service) validateSectionsExist(venueTemplateID uuid.UUID, sectionPricing []CreateEventSectionPricing) error {
	if s.venueService == nil {
		return errors.New("venue service not available")
	}

	if len(sectionPricing) == 0 {
		return errors.New("at least one section pricing must be provided")
	}

	// Get all sections for the venue template
	sectionsInterface, err := s.venueService.GetSectionsByTemplateID(context.TODO(), venueTemplateID.String())
	if err != nil {
		return fmt.Errorf("failed to fetch venue sections: %w", err)
	}

	// Use reflection to handle the interface{} response
	sectionsValue := reflect.ValueOf(sectionsInterface)
	if sectionsValue.Kind() != reflect.Slice {
		return fmt.Errorf("expected slice of venue sections, got %T", sectionsInterface)
	}

	if sectionsValue.Len() == 0 {
		return fmt.Errorf("venue template has no sections defined")
	}

	// Create a map of valid section IDs for quick lookup
	validSectionIDs := make(map[string]bool)
	for i := 0; i < sectionsValue.Len(); i++ {
		section := sectionsValue.Index(i)

		// Try to get the ID field using reflection
		if section.Kind() == reflect.Struct {
			idField := section.FieldByName("ID")
			if idField.IsValid() && idField.Type() == reflect.TypeOf(uuid.UUID{}) {
				id := idField.Interface().(uuid.UUID)
				validSectionIDs[id.String()] = true
			}
		}
	}

	// Validate each section ID in the request
	var invalidSections []string
	requestedSections := make(map[string]bool) // To check for duplicates

	for _, pricing := range sectionPricing {
		// Check for duplicates
		if requestedSections[pricing.SectionID] {
			return fmt.Errorf("duplicate section ID in pricing: %s", pricing.SectionID)
		}
		requestedSections[pricing.SectionID] = true

		// Validate section ID format
		if _, err := uuid.Parse(pricing.SectionID); err != nil {
			return fmt.Errorf("invalid section ID format: %s", pricing.SectionID)
		}

		// Check if section exists in the venue template
		if !validSectionIDs[pricing.SectionID] {
			invalidSections = append(invalidSections, pricing.SectionID)
		}
	}

	if len(invalidSections) > 0 {
		return fmt.Errorf("the following section IDs do not exist in the venue template: %v", invalidSections)
	}

	return nil
}

func (s *service) CreateEvent(userID uuid.UUID, req CreateEventRequest) (*EventResponse, error) {
	// Validate date is in the future
	if req.DateTime.Before(time.Now()) {
		return nil, errors.New("event date must be in the future")
	}

	// VALIDATE TAGS FIRST - before creating event
	if len(req.Tags) > 0 && s.tagService != nil {
		if err := s.validateTagsExist(req.Tags); err != nil {
			return nil, fmt.Errorf("tag validation failed: %w", err)
		}
	}

	// Parse venue template ID
	venueTemplateID, err := uuid.Parse(req.VenueTemplateID)
	if err != nil {
		return nil, fmt.Errorf("invalid venue template ID: %w", err)
	}

	// VALIDATE SECTION IDs - ensure they exist and belong to the venue template
	if len(req.SectionPricing) > 0 && s.venueService != nil {
		if err := s.validateSectionsExist(venueTemplateID, req.SectionPricing); err != nil {
			return nil, fmt.Errorf("section validation failed: %w", err)
		}
	}

	event := &Event{
		Name:            req.Name,
		Description:     req.Description,
		Venue:           req.Venue,
		VenueTemplateID: venueTemplateID,
		DateTime:        req.DateTime,
		BasePrice:       req.BasePrice,
		Status:          EventStatusPublished,
		ImageURL:        req.ImageURL,
		CreatedBy:       userID,
	}

	if err := s.repo.Create(event); err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	// Create event pricing for each section
	if err := s.createEventPricing(event.ID, req.SectionPricing); err != nil {
		// If pricing creation fails, we should delete the created event
		s.repo.Delete(event.ID) // Best effort cleanup
		return nil, fmt.Errorf("failed to create event pricing: %w", err)
	}

	response := event.ToResponse()

	// Handle tags if provided (we already validated they exist)
	if len(req.Tags) > 0 && s.tagService != nil {
		if err := s.tagService.ReplaceEventTags(event.ID, req.Tags); err != nil {
			// If tag assignment fails, we should delete the created event
			s.repo.Delete(event.ID) // Best effort cleanup
			return nil, fmt.Errorf("failed to assign tags: %w", err)
		}
	}

	// Populate capacity data in response
	if err := s.populateEventCapacity(&response); err != nil {
		return nil, fmt.Errorf("failed to populate capacity data: %w", err)
	}

	// Populate tags in response
	if err := s.populateEventTags(&response); err != nil {
		return nil, fmt.Errorf("failed to populate tags: %w", err)
	}

	// Invalidate event cache after creation
	ctx := context.Background()
	if err := s.invalidateEventCache(ctx, nil); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: failed to invalidate event cache after creation: %v\n", err)
	}

	return &response, nil
}

func (s *service) GetEventByID(id uuid.UUID) (*EventResponse, error) {
	ctx := context.Background()
	cacheKey := constants.BuildEventDetailKey(id.String())

	// Try to get from cache first
	var cachedEvent EventResponse
	if err := s.getCache(ctx, cacheKey, &cachedEvent); err == nil {
		return &cachedEvent, nil
	}

	// Cache miss - get from database
	event, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("event not found")
		}
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	response := event.ToResponse()

	// Populate capacity data in response
	if err := s.populateEventCapacity(&response); err != nil {
		return nil, fmt.Errorf("failed to populate capacity data: %w", err)
	}

	// Populate tags in response
	if err := s.populateEventTags(&response); err != nil {
		return nil, fmt.Errorf("failed to populate tags: %w", err)
	}

	// Cache the result
	if err := s.setCache(ctx, cacheKey, response, constants.TTL_EVENT_DETAIL); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: failed to cache event detail: %v\n", err)
	}

	return &response, nil
}

func (s *service) UpdateEvent(id uuid.UUID, userID uuid.UUID, req UpdateEventRequest) (*EventResponse, error) {
	// Get current event
	currentEvent, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("event not found")
		}
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	// Check if user has permission to update (for now, only creator can update)
	if currentEvent.CreatedBy != userID {
		return nil, errors.New("unauthorized: you can only update events you created")
	}

	// Check if event can be updated based on its status
	if !currentEvent.Status.CanBeUpdated() {
		return nil, fmt.Errorf("cannot update event with status: %s", currentEvent.Status)
	}

	// Build updates map
	updates := make(map[string]interface{})

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Venue != nil {
		updates["venue"] = *req.Venue
	}
	if req.DateTime != nil {
		if req.DateTime.Before(time.Now()) {
			return nil, errors.New("event date must be in the future")
		}
		updates["date_time"] = *req.DateTime
	}
	if req.BasePrice != nil {
		updates["base_price"] = *req.BasePrice
	}
	if req.Status != nil {
		status := EventStatus(*req.Status)
		if !status.IsValid() {
			return nil, errors.New("invalid event status")
		}
		updates["status"] = status
	}
	if req.ImageURL != nil {
		updates["image_url"] = *req.ImageURL
	}

	// Update timestamp
	updates["updated_at"] = time.Now()

	updatedEvent, err := s.repo.Update(id, updates)
	if err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	// Handle tags if provided - validate first
	if req.Tags != nil && s.tagService != nil {
		if len(req.Tags) > 0 {
			if err := s.validateTagsExist(req.Tags); err != nil {
				return nil, fmt.Errorf("tag validation failed: %w", err)
			}
		}
		if err := s.tagService.ReplaceEventTags(id, req.Tags); err != nil {
			return nil, fmt.Errorf("failed to update tags: %w", err)
		}
	}

	response := updatedEvent.ToResponse()

	// Populate capacity data in response
	if err := s.populateEventCapacity(&response); err != nil {
		return nil, fmt.Errorf("failed to populate capacity data: %w", err)
	}

	// Populate tags in response
	if err := s.populateEventTags(&response); err != nil {
		return nil, fmt.Errorf("failed to populate tags: %w", err)
	}

	// Invalidate event cache after update
	ctx := context.Background()
	if err := s.invalidateEventCache(ctx, &id); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: failed to invalidate event cache after update: %v\n", err)
	}

	return &response, nil
}

func (s *service) DeleteEvent(id uuid.UUID, userID uuid.UUID) error {
	// Get current event
	currentEvent, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("event not found")
		}
		return fmt.Errorf("failed to get event: %w", err)
	}

	// Check if user has permission to delete
	if currentEvent.CreatedBy != userID {
		return errors.New("unauthorized: you can only delete events you created")
	}

	// Check if event can be deleted based on its status
	if !currentEvent.Status.CanBeDeleted() {
		return fmt.Errorf("cannot delete event with status: %s. Only draft events can be deleted", currentEvent.Status)
	}

	// If event has bookings, don't allow deletion
	_, bookedCount, err := s.repo.GetEventCapacityAndBookings(id)
	if err != nil {
		return fmt.Errorf("failed to check event bookings: %w", err)
	}
	if bookedCount > 0 {
		return errors.New("cannot delete event with existing bookings")
	}

	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	return nil
}

func (s *service) GetAllEvents(query EventListQuery) (*PaginatedEvents, error) {
	// Set defaults
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Limit <= 0 {
		query.Limit = 10
	}

	ctx := context.Background()
	cacheKey := constants.BuildEventListKey(query.Page, query.Limit, query.Status)

	// Try to get from cache first
	var cachedResult PaginatedEvents
	if err := s.getCache(ctx, cacheKey, &cachedResult); err == nil {
		log.Printf("Cache HIT for event list: %s", cacheKey)
		return &cachedResult, nil
	} else {
		log.Printf("Cache MISS for event list: %s (error: %v)", cacheKey, err)
	}

	// Cache miss - get from database
	events, totalCount, err := s.repo.GetAll(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	// Convert to response format and populate capacity + tags
	eventResponses := make([]EventResponse, len(events))
	for i, event := range events {
		response := event.ToResponse()

		// Populate capacity data for each event
		if err := s.populateEventCapacity(&response); err != nil {
			// Log error but don't fail the entire request
		}

		// Populate tags for each event
		if err := s.populateEventTags(&response); err != nil {
			// Log error but don't fail the entire request
			// In production, you might want to log this properly
		}
		eventResponses[i] = response
	}

	// Calculate total pages
	totalPages := int(math.Ceil(float64(totalCount) / float64(query.Limit)))

	result := &PaginatedEvents{
		Events:     eventResponses,
		TotalCount: totalCount,
		Page:       query.Page,
		Limit:      query.Limit,
		TotalPages: totalPages,
	}

	// Cache the result
	if err := s.setCache(ctx, cacheKey, result, constants.TTL_EVENT_LIST); err != nil {
		// Log error but don't fail the request
		log.Printf("Warning: failed to cache event list: %v", err)
	} else {
		log.Printf("Cached event list: %s", cacheKey)
	}

	return result, nil
}

func (s *service) GetEventAnalytics(eventID uuid.UUID, userID uuid.UUID) (*EventAnalytics, error) {
	// Get event to check ownership
	event, err := s.repo.GetByID(eventID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("event not found")
		}
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	// Check if user has permission to view analytics
	if event.CreatedBy != userID {
		return nil, errors.New("unauthorized: you can only view analytics for events you created")
	}

	analytics, err := s.repo.GetEventAnalytics(eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get event analytics: %w", err)
	}

	return analytics, nil
}

func (s *service) GetUpcomingEvents(limit int) ([]EventResponse, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	ctx := context.Background()
	cacheKey := constants.CACHE_KEY_EVENTS_UPCOMING + ":limit:" + fmt.Sprintf("%d", limit)

	// Try to get from cache first
	var cachedResult []EventResponse
	if err := s.getCache(ctx, cacheKey, &cachedResult); err == nil {
		log.Printf("Cache HIT for upcoming events: %s", cacheKey)
		return cachedResult, nil
	} else {
		log.Printf("Cache MISS for upcoming events: %s (error: %v)", cacheKey, err)
	}

	// Cache miss - get from database
	events, err := s.repo.GetUpcomingEvents(limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get upcoming events: %w", err)
	}

	responses := make([]EventResponse, len(events))
	for i, event := range events {
		response := event.ToResponse()

		// Populate capacity data for each event
		if err := s.populateEventCapacity(&response); err != nil {
			// Log error but don't fail the entire request
		}

		// Populate tags for each event
		if err := s.populateEventTags(&response); err != nil {
			// Log error but don't fail the entire request
		}
		responses[i] = response
	}

	// Cache the result
	if err := s.setCache(ctx, cacheKey, responses, constants.TTL_EVENT_UPCOMING); err != nil {
		// Log error but don't fail the request
		log.Printf("Warning: failed to cache upcoming events: %v", err)
	} else {
		log.Printf("Cached upcoming events: %s", cacheKey)
	}

	return responses, nil
}

func (s *service) CheckEventAvailability(eventID uuid.UUID, seatCount int) (bool, error) {
	if seatCount <= 0 {
		return false, errors.New("seat count must be positive")
	}

	// Check if event exists and can be booked
	event, err := s.repo.GetByID(eventID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, errors.New("event not found")
		}
		return false, fmt.Errorf("failed to get event: %w", err)
	}

	// Check if event allows bookings
	if !event.Status.CanBeBooked() {
		return false, fmt.Errorf("event is not available for booking (status: %s)", event.Status)
	}

	// Check if event is in the future
	if event.DateTime.Before(time.Now()) {
		return false, errors.New("cannot book tickets for past events")
	}

	// Check seat availability
	return s.repo.CheckSeatAvailability(eventID, seatCount)
}

func (s *service) GetEventCapacityData(eventID uuid.UUID) (totalCapacity, bookedCount, availableSeats int, err error) {
	totalCapacity, bookedCount, err = s.repo.GetEventCapacityAndBookings(eventID)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get event capacity data: %w", err)
	}

	availableSeats = totalCapacity - bookedCount
	if availableSeats < 0 {
		availableSeats = 0
	}

	return totalCapacity, bookedCount, availableSeats, nil
}

func (s *service) IsEventInFuture(eventID uuid.UUID) (bool, error) {
	// Get the event to check its date
	event, err := s.repo.GetByID(eventID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, errors.New("event not found")
		}
		return false, fmt.Errorf("failed to get event: %w", err)
	}

	// Check if event is in the future
	return event.DateTime.After(time.Now()), nil
}

// Admin methods - allow admins to manage any event without ownership checks

func (s *service) UpdateEventAsAdmin(id uuid.UUID, adminID uuid.UUID, req UpdateEventRequest) (*EventResponse, error) {
	// Get current event
	currentEvent, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("event not found")
		}
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	// Admins can update any event, but still respect status constraints
	if !currentEvent.Status.CanBeUpdated() {
		return nil, fmt.Errorf("cannot update event with status: %s", currentEvent.Status)
	}

	// Build updates map (same logic as regular update)
	updates := make(map[string]interface{})

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Venue != nil {
		updates["venue"] = *req.Venue
	}
	if req.DateTime != nil {
		if req.DateTime.Before(time.Now()) {
			return nil, errors.New("event date must be in the future")
		}
		updates["date_time"] = *req.DateTime
	}
	if req.BasePrice != nil {
		updates["base_price"] = *req.BasePrice
	}
	if req.Status != nil {
		status := EventStatus(*req.Status)
		if !status.IsValid() {
			return nil, errors.New("invalid event status")
		}
		updates["status"] = status
	}
	if req.ImageURL != nil {
		updates["image_url"] = *req.ImageURL
	}
	// Update timestamp
	updates["updated_at"] = time.Now()
	// Track who updated it
	updates["updated_by"] = adminID

	updatedEvent, err := s.repo.Update(id, updates)
	if err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	// Handle tags if provided - validate first
	if req.Tags != nil && s.tagService != nil {
		if len(req.Tags) > 0 {
			if err := s.validateTagsExist(req.Tags); err != nil {
				return nil, fmt.Errorf("tag validation failed: %w", err)
			}
		}
		if err := s.tagService.ReplaceEventTags(id, req.Tags); err != nil {
			return nil, fmt.Errorf("failed to update tags: %w", err)
		}
	}

	response := updatedEvent.ToResponse()

	// Populate capacity data in response
	if err := s.populateEventCapacity(&response); err != nil {
		return nil, fmt.Errorf("failed to populate capacity data: %w", err)
	}

	// Populate tags in response
	if err := s.populateEventTags(&response); err != nil {
		return nil, fmt.Errorf("failed to populate tags: %w", err)
	}

	return &response, nil
}

func (s *service) DeleteEventAsAdmin(id uuid.UUID, adminID uuid.UUID) error {
	// Check if event exists
	_, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("event not found")
		}
		return fmt.Errorf("failed to get event: %w", err)
	}

	// Admin can delete events with more flexibility than regular users
	// But still respect business logic for events with bookings
	_, bookedCount, err := s.repo.GetEventCapacityAndBookings(id)
	if err != nil {
		return fmt.Errorf("failed to check event bookings: %w", err)
	}
	if bookedCount > 0 {
		return errors.New("cannot delete event with existing bookings. Consider canceling the event instead")
	}

	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	return nil
}

func (s *service) GetEventAnalyticsAsAdmin(eventID uuid.UUID) (*EventAnalytics, error) {
	// Admin can view analytics for any event without ownership check
	_, err := s.repo.GetByID(eventID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("event not found")
		}
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	analytics, err := s.repo.GetEventAnalytics(eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get event analytics: %w", err)
	}

	return analytics, nil
}

func (s *service) GetAllEventAnalyticsAsAdmin() (*GlobalAnalytics, error) {
	// Get overall analytics across all events
	analytics, err := s.repo.GetGlobalAnalytics()
	if err != nil {
		return nil, fmt.Errorf("failed to get global analytics: %w", err)
	}

	return analytics, nil
}

// createEventPricing creates event pricing entries for the given event and sections
func (s *service) createEventPricing(eventID uuid.UUID, sectionPricing []CreateEventSectionPricing) error {
	// Create a temporary struct to match the event_pricing table
	type EventPricing struct {
		ID              uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
		EventID         uuid.UUID `gorm:"type:uuid;not null;index"`
		SectionID       uuid.UUID `gorm:"type:uuid;not null;index"`
		PriceMultiplier float64   `gorm:"not null;default:1.0"`
		IsActive        bool      `gorm:"default:true"`
	}

	db := s.repo.(*repository).db // Access the underlying DB

	for _, pricing := range sectionPricing {
		sectionID, err := uuid.Parse(pricing.SectionID)
		if err != nil {
			return fmt.Errorf("invalid section ID %s: %w", pricing.SectionID, err)
		}

		eventPricing := EventPricing{
			ID:              uuid.New(),
			EventID:         eventID,
			SectionID:       sectionID,
			PriceMultiplier: pricing.PriceMultiplier,
			IsActive:        true,
		}

		if err := db.Table("event_pricing").Create(&eventPricing).Error; err != nil {
			return fmt.Errorf("failed to create pricing for section %s: %w", pricing.SectionID, err)
		}
	}

	return nil
}
