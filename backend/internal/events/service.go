package events

import (
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service interface {
	// Tag service dependency injection
	SetTagService(tagService TagService)
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
	CheckEventAvailability(eventID uuid.UUID, ticketCount int) (bool, error)
	IsEventInFuture(eventID uuid.UUID) (bool, error)
	IncrementBookedCount(eventID uuid.UUID, increment int) error
}

type service struct {
	repo       Repository
	tagService TagService
}

// TagService interface to avoid circular dependencies
type TagService interface {
	ReplaceEventTags(eventID uuid.UUID, tagNames []string) error
	GetTagsByEventID(eventID uuid.UUID) ([]TagResponse, error)
	GetTagsByNames(tagNames []string) ([]TagResponse, error) // Add this method
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) SetTagService(tagService TagService) {
	s.tagService = tagService
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

	event := &Event{
		Name:          req.Name,
		Description:   req.Description,
		Venue:         req.Venue,
		DateTime:      req.DateTime,
		TotalCapacity: req.TotalCapacity,
		Price:         req.Price,
		Status:        EventStatusPublished,
		ImageURL:      req.ImageURL,
		CreatedBy:     userID,
		BookedCount:   0,
	}

	if err := s.repo.Create(event); err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
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

	// Populate tags in response
	if err := s.populateEventTags(&response); err != nil {
		return nil, fmt.Errorf("failed to populate tags: %w", err)
	}

	return &response, nil
}

func (s *service) GetEventByID(id uuid.UUID) (*EventResponse, error) {
	event, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("event not found")
		}
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	response := event.ToResponse()

	// Populate tags in response
	if err := s.populateEventTags(&response); err != nil {
		return nil, fmt.Errorf("failed to populate tags: %w", err)
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
	if req.TotalCapacity != nil {
		// Ensure new capacity is not less than current bookings
		if *req.TotalCapacity < currentEvent.BookedCount {
			return nil, fmt.Errorf("cannot reduce capacity below current bookings (%d)", currentEvent.BookedCount)
		}
		updates["total_capacity"] = *req.TotalCapacity
	}
	if req.Price != nil {
		updates["price"] = *req.Price
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

	// Populate tags in response
	if err := s.populateEventTags(&response); err != nil {
		return nil, fmt.Errorf("failed to populate tags: %w", err)
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
	if currentEvent.BookedCount > 0 {
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

	events, totalCount, err := s.repo.GetAll(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	// Convert to response format and populate tags
	eventResponses := make([]EventResponse, len(events))
	for i, event := range events {
		response := event.ToResponse()
		// Populate tags for each event
		if err := s.populateEventTags(&response); err != nil {
			// Log error but don't fail the entire request
			// In production, you might want to log this properly
		}
		eventResponses[i] = response
	}

	// Calculate total pages
	totalPages := int(math.Ceil(float64(totalCount) / float64(query.Limit)))

	return &PaginatedEvents{
		Events:     eventResponses,
		TotalCount: totalCount,
		Page:       query.Page,
		Limit:      query.Limit,
		TotalPages: totalPages,
	}, nil
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

	events, err := s.repo.GetUpcomingEvents(limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get upcoming events: %w", err)
	}

	responses := make([]EventResponse, len(events))
	for i, event := range events {
		response := event.ToResponse()
		// Populate tags for each event
		if err := s.populateEventTags(&response); err != nil {
			// Log error but don't fail the entire request
		}
		responses[i] = response
	}

	return responses, nil
}

func (s *service) CheckEventAvailability(eventID uuid.UUID, ticketCount int) (bool, error) {
	if ticketCount <= 0 {
		return false, errors.New("ticket count must be positive")
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

	// Check capacity
	return s.repo.CheckCapacityAvailability(eventID, ticketCount)
}

func (s *service) IncrementBookedCount(eventID uuid.UUID, increment int) error {
	return s.repo.UpdateBookedCount(eventID, increment)
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
	if req.TotalCapacity != nil {
		// Ensure new capacity is not less than current bookings
		if *req.TotalCapacity < currentEvent.BookedCount {
			return nil, fmt.Errorf("cannot reduce capacity below current bookings (%d)", currentEvent.BookedCount)
		}
		updates["total_capacity"] = *req.TotalCapacity
	}
	if req.Price != nil {
		updates["price"] = *req.Price
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

	// Populate tags in response
	if err := s.populateEventTags(&response); err != nil {
		return nil, fmt.Errorf("failed to populate tags: %w", err)
	}

	return &response, nil
}

func (s *service) DeleteEventAsAdmin(id uuid.UUID, adminID uuid.UUID) error {
	// Get current event
	currentEvent, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("event not found")
		}
		return fmt.Errorf("failed to get event: %w", err)
	}

	// Admin can delete events with more flexibility than regular users
	// But still respect business logic for events with bookings
	if currentEvent.BookedCount > 0 {
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
