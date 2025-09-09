package tags

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service interface {
	// Admin CRUD operations
	CreateTag(adminID uuid.UUID, req CreateTagRequest) (*TagResponse, error)
	GetTagByID(id uuid.UUID) (*TagResponse, error)
	GetTagBySlug(slug string) (*TagResponse, error)
	UpdateTag(id uuid.UUID, adminID uuid.UUID, req UpdateTagRequest) (*TagResponse, error)
	DeleteTag(id uuid.UUID, adminID uuid.UUID) error
	GetAllTags(query TagListQuery) (*PaginatedTags, error)
	GetActiveTags() ([]TagResponse, error)

	// Tag assignment operations (called by event service)
	AssignTagsToEvent(eventID uuid.UUID, tagNames []string) error
	RemoveTagsFromEvent(eventID uuid.UUID, tagNames []string) error
	GetTagsByEventID(eventID uuid.UUID) ([]TagResponse, error)
	GetTagsByNames(tagNames []string) ([]TagResponse, error)
	ReplaceEventTags(eventID uuid.UUID, tagNames []string) error

	// Analytics operations (admin only)
	GetTagAnalytics() (*TagAnalyticsResponse, error)
	GetTagPopularityAnalytics() ([]TagAnalytics, error)
	GetTagTrends(months int) ([]TagTrend, error)
	GetTagComparisons() ([]TagComparison, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// Helper function to generate slug from name
func generateSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)

	// Replace spaces and special characters with hyphens
	reg := regexp.MustCompile(`[^\w\s-]`)
	slug = reg.ReplaceAllString(slug, "")

	// Replace multiple spaces/hyphens with single hyphen
	reg = regexp.MustCompile(`[\s-]+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")

	return slug
}

// Helper function to validate hex color
func isValidHexColor(color string) bool {
	if len(color) != 7 || color[0] != '#' {
		return false
	}
	match, _ := regexp.MatchString("^#[0-9A-Fa-f]{6}$", color)
	return match
}

// Admin CRUD operations

func (s *service) CreateTag(adminID uuid.UUID, req CreateTagRequest) (*TagResponse, error) {
	// Validate and clean name
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("tag name cannot be empty")
	}

	// Generate slug
	slug := generateSlug(name)
	if slug == "" {
		return nil, errors.New("tag name must contain at least one alphanumeric character")
	}

	// Check if tag with same slug already exists
	existingTag, err := s.repo.GetBySlug(slug)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check existing tag: %w", err)
	}
	if existingTag != nil {
		return nil, errors.New("a tag with similar name already exists")
	}

	// Set default color if not provided or invalid
	color := req.Color
	if color == "" || !isValidHexColor(color) {
		color = "#6B7280" // Default gray color
	}

	tag := &Tag{
		Name:        name,
		Slug:        slug,
		Description: strings.TrimSpace(req.Description),
		Color:       color,
		IsActive:    true,
		CreatedBy:   adminID,
	}

	if err := s.repo.Create(tag); err != nil {
		return nil, fmt.Errorf("failed to create tag: %w", err)
	}

	response := tag.ToResponse()
	return &response, nil
}

func (s *service) GetTagByID(id uuid.UUID) (*TagResponse, error) {
	tag, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("tag not found")
		}
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}

	response := tag.ToResponse()
	return &response, nil
}

func (s *service) GetTagBySlug(slug string) (*TagResponse, error) {
	tag, err := s.repo.GetBySlug(slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("tag not found")
		}
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}

	response := tag.ToResponse()
	return &response, nil
}

func (s *service) UpdateTag(id uuid.UUID, adminID uuid.UUID, req UpdateTagRequest) (*TagResponse, error) {
	// Get current tag
	currentTag, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("tag not found")
		}
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}

	// Build updates map
	updates := make(map[string]interface{})

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, errors.New("tag name cannot be empty")
		}

		// Generate new slug
		slug := generateSlug(name)
		if slug == "" {
			return nil, errors.New("tag name must contain at least one alphanumeric character")
		}

		// Check if another tag with same slug exists (excluding current tag)
		if slug != currentTag.Slug {
			existingTag, err := s.repo.GetBySlug(slug)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("failed to check existing tag: %w", err)
			}
			if existingTag != nil && existingTag.ID != currentTag.ID {
				return nil, errors.New("a tag with similar name already exists")
			}
		}

		updates["name"] = name
		updates["slug"] = slug
	}

	if req.Description != nil {
		updates["description"] = strings.TrimSpace(*req.Description)
	}

	if req.Color != nil {
		color := *req.Color
		if color != "" && !isValidHexColor(color) {
			return nil, errors.New("invalid color format. Use hex format like #FF0000")
		}
		if color == "" {
			color = "#6B7280" // Default color
		}
		updates["color"] = color
	}

	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	// Update timestamp and admin info
	updates["updated_at"] = time.Now()
	updates["updated_by"] = adminID

	updatedTag, err := s.repo.Update(id, updates)
	if err != nil {
		return nil, fmt.Errorf("failed to update tag: %w", err)
	}

	response := updatedTag.ToResponse()
	return &response, nil
}

func (s *service) DeleteTag(id uuid.UUID, adminID uuid.UUID) error {
	// Check if tag exists
	_, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("tag not found")
		}
		return fmt.Errorf("failed to get tag: %w", err)
	}

	// Check if tag is being used by any events
	eventIDs, err := s.repo.GetEventsByTagID(id)
	if err != nil {
		return fmt.Errorf("failed to check tag usage: %w", err)
	}

	if len(eventIDs) > 0 {
		return fmt.Errorf("cannot delete tag as it is being used by %d event(s). Consider deactivating it instead", len(eventIDs))
	}

	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}

	return nil
}

func (s *service) GetAllTags(query TagListQuery) (*PaginatedTags, error) {
	// Set defaults
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Limit <= 0 {
		query.Limit = 10
	}

	tags, totalCount, err := s.repo.GetAll(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}

	// Convert to response format
	tagResponses := make([]TagResponse, len(tags))
	for i, tag := range tags {
		tagResponses[i] = tag.ToResponse()
	}

	// Calculate total pages
	totalPages := int(math.Ceil(float64(totalCount) / float64(query.Limit)))

	return &PaginatedTags{
		Tags:       tagResponses,
		TotalCount: totalCount,
		Page:       query.Page,
		Limit:      query.Limit,
		TotalPages: totalPages,
	}, nil
}

func (s *service) GetActiveTags() ([]TagResponse, error) {
	tags, err := s.repo.GetActive()
	if err != nil {
		return nil, fmt.Errorf("failed to get active tags: %w", err)
	}

	responses := make([]TagResponse, len(tags))
	for i, tag := range tags {
		responses[i] = tag.ToResponse()
	}

	return responses, nil
}

// Tag assignment operations

func (s *service) AssignTagsToEvent(eventID uuid.UUID, tagNames []string) error {
	if len(tagNames) == 0 {
		return nil
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

	if len(cleanNames) == 0 {
		return nil
	}

	// Get existing tags
	existingTags, err := s.repo.GetTagsByNames(cleanNames)
	if err != nil {
		return fmt.Errorf("failed to get existing tags: %w", err)
	}

	// Map existing tag names to IDs
	existingTagMap := make(map[string]uuid.UUID)
	for _, tag := range existingTags {
		existingTagMap[tag.Name] = tag.ID
	}

	// Collect tag IDs to assign
	var tagIDs []uuid.UUID
	for _, name := range cleanNames {
		if tagID, exists := existingTagMap[name]; exists {
			tagIDs = append(tagIDs, tagID)
		}
		// Skip non-existing tags - only assign existing ones
	}

	if len(tagIDs) == 0 {
		return nil
	}

	return s.repo.AssignTagsToEvent(eventID, tagIDs)
}

func (s *service) RemoveTagsFromEvent(eventID uuid.UUID, tagNames []string) error {
	if len(tagNames) == 0 {
		return nil
	}

	// Get tag IDs from names
	existingTags, err := s.repo.GetTagsByNames(tagNames)
	if err != nil {
		return fmt.Errorf("failed to get existing tags: %w", err)
	}

	var tagIDs []uuid.UUID
	for _, tag := range existingTags {
		tagIDs = append(tagIDs, tag.ID)
	}

	if len(tagIDs) == 0 {
		return nil // No tags to remove
	}

	return s.repo.RemoveTagsFromEvent(eventID, tagIDs)
}

func (s *service) GetTagsByEventID(eventID uuid.UUID) ([]TagResponse, error) {
	tags, err := s.repo.GetTagsByEventID(eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get event tags: %w", err)
	}

	responses := make([]TagResponse, len(tags))
	for i, tag := range tags {
		responses[i] = tag.ToResponse()
	}

	return responses, nil
}

func (s *service) GetTagsByNames(tagNames []string) ([]TagResponse, error) {
	if len(tagNames) == 0 {
		return []TagResponse{}, nil
	}

	tags, err := s.repo.GetTagsByNames(tagNames)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags by names: %w", err)
	}

	responses := make([]TagResponse, len(tags))
	for i, tag := range tags {
		responses[i] = tag.ToResponse()
	}

	return responses, nil
}

func (s *service) ReplaceEventTags(eventID uuid.UUID, tagNames []string) error {
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

	// If no valid tag names, remove all tags
	if len(cleanNames) == 0 {
		return s.repo.ReplaceEventTags(eventID, []uuid.UUID{})
	}

	// Get existing tags
	existingTags, err := s.repo.GetTagsByNames(cleanNames)
	if err != nil {
		return fmt.Errorf("failed to get existing tags: %w", err)
	}

	// Collect tag IDs
	var tagIDs []uuid.UUID
	for _, tag := range existingTags {
		tagIDs = append(tagIDs, tag.ID)
	}

	return s.repo.ReplaceEventTags(eventID, tagIDs)
}

// Analytics operations

func (s *service) GetTagAnalytics() (*TagAnalyticsResponse, error) {
	analytics, err := s.repo.GetTagAnalytics()
	if err != nil {
		return nil, fmt.Errorf("failed to get tag analytics: %w", err)
	}

	return analytics, nil
}

func (s *service) GetTagPopularityAnalytics() ([]TagAnalytics, error) {
	analytics, err := s.repo.GetTagPopularityAnalytics()
	if err != nil {
		return nil, fmt.Errorf("failed to get tag popularity analytics: %w", err)
	}

	return analytics, nil
}

func (s *service) GetTagTrends(months int) ([]TagTrend, error) {
	if months <= 0 {
		months = 6 // Default to 6 months
	}
	if months > 24 {
		months = 24 // Max 24 months
	}

	trends, err := s.repo.GetTagTrends(months)
	if err != nil {
		return nil, fmt.Errorf("failed to get tag trends: %w", err)
	}

	return trends, nil
}

func (s *service) GetTagComparisons() ([]TagComparison, error) {
	comparisons, err := s.repo.GetTagComparisons()
	if err != nil {
		return nil, fmt.Errorf("failed to get tag comparisons: %w", err)
	}

	return comparisons, nil
}
