package tags

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"evently/internal/shared/utils/constants"
	"evently/pkg/cache"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Service interface {
	CreateTag(adminID uuid.UUID, req CreateTagRequest) (*TagResponse, error)
	GetTagByID(id uuid.UUID) (*TagResponse, error)
	GetTagBySlug(slug string) (*TagResponse, error)
	UpdateTag(id uuid.UUID, adminID uuid.UUID, req UpdateTagRequest) (*TagResponse, error)
	DeleteTag(id uuid.UUID, adminID uuid.UUID) error
	GetAllTags(query TagListQuery) (*PaginatedTags, error)
	GetActiveTags() ([]TagResponse, error)

	AssignTagsToEvent(eventID uuid.UUID, tagNames []string) error
	RemoveTagsFromEvent(eventID uuid.UUID, tagNames []string) error
	GetTagsByEventID(eventID uuid.UUID) ([]TagResponse, error)
	GetTagsByNames(tagNames []string) ([]TagResponse, error)
	ReplaceEventTags(eventID uuid.UUID, tagNames []string) error
}

type service struct {
	repo        Repository
	redisClient *redis.Client
}

func NewService(repo Repository) Service {
	return &service{
		repo:        repo,
		redisClient: cache.Client(),
	}
}

func (s *service) CreateTag(adminID uuid.UUID, req CreateTagRequest) (*TagResponse, error) {

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("tag name cannot be empty")
	}

	slug := GenerateSlug(name)
	if slug == "" {
		return nil, errors.New("tag name must contain at least one alphanumeric character")
	}

	existingTag, err := s.repo.GetBySlug(slug)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check existing tag: %w", err)
	}
	if existingTag != nil {
		return nil, errors.New("a tag with similar name already exists")
	}

	color := req.Color
	if color == "" || !IsValidHexColor(color) {
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

	// Invalidate tag cache
	ctx := context.Background()
	if err := InvalidateTagCache(ctx, s.redisClient); err != nil {
		fmt.Printf("Warning: failed to invalidate tag cache after creation: %v\n", err)
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
	ctx := context.Background()
	cacheKey := constants.BuildTagBySlugKey(slug)

	// Trying cache
	var cachedTag TagResponse
	if err := GetCache(ctx, s.redisClient, cacheKey, &cachedTag); err == nil {
		return &cachedTag, nil
	}

	// Cache miss
	tag, err := s.repo.GetBySlug(slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("tag not found")
		}
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}

	response := tag.ToResponse()

	if err := SetCache(ctx, s.redisClient, cacheKey, response, constants.TTL_TAG_DETAIL); err != nil {
		fmt.Printf("Warning: failed to cache tag by slug: %v\n", err)
	}

	return &response, nil
}

func (s *service) UpdateTag(id uuid.UUID, adminID uuid.UUID, req UpdateTagRequest) (*TagResponse, error) {

	currentTag, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("tag not found")
		}
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}

	updates := make(map[string]interface{})

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, errors.New("tag name cannot be empty")
		}

		slug := GenerateSlug(name)
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
		if color != "" && !IsValidHexColor(color) {
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

	updates["updated_at"] = time.Now()
	updates["updated_by"] = adminID

	updatedTag, err := s.repo.Update(id, updates)
	if err != nil {
		return nil, fmt.Errorf("failed to update tag: %w", err)
	}

	// Invalidate tag cache
	ctx := context.Background()
	if err := InvalidateTagCache(ctx, s.redisClient); err != nil {

		fmt.Printf("Warning: failed to invalidate tag cache after update: %v\n", err)
	}

	response := updatedTag.ToResponse()
	return &response, nil
}

func (s *service) DeleteTag(id uuid.UUID, adminID uuid.UUID) error {

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

	// Invalidate tag cache
	ctx := context.Background()
	if err := InvalidateTagCache(ctx, s.redisClient); err != nil {

		fmt.Printf("Warning: failed to invalidate tag cache after deletion: %v\n", err)
	}

	return nil
}

func (s *service) GetAllTags(query TagListQuery) (*PaginatedTags, error) {

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
	ctx := context.Background()
	cacheKey := constants.CACHE_KEY_TAGS_ACTIVE

	var cachedTags []TagResponse
	if err := GetCache(ctx, s.redisClient, cacheKey, &cachedTags); err == nil {
		return cachedTags, nil
	}

	// Cache miss
	tags, err := s.repo.GetActive()
	if err != nil {
		return nil, fmt.Errorf("failed to get active tags: %w", err)
	}

	responses := make([]TagResponse, len(tags))
	for i, tag := range tags {
		responses[i] = tag.ToResponse()
	}

	if err := SetCache(ctx, s.redisClient, cacheKey, responses, constants.TTL_TAGS_ACTIVE); err != nil {
		fmt.Printf("Warning: failed to cache active tags: %v\n", err)
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
