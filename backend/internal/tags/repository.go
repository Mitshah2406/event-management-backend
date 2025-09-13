package tags

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	Create(tag *Tag) error
	GetByID(id uuid.UUID) (*Tag, error)
	GetBySlug(slug string) (*Tag, error)
	Update(id uuid.UUID, updates map[string]interface{}) (*Tag, error)
	Delete(id uuid.UUID) error
	GetAll(query TagListQuery) ([]Tag, int64, error)
	GetActive() ([]Tag, error)

	AssignTagsToEvent(eventID uuid.UUID, tagIDs []uuid.UUID) error
	RemoveTagsFromEvent(eventID uuid.UUID, tagIDs []uuid.UUID) error
	GetTagsByEventID(eventID uuid.UUID) ([]Tag, error)
	GetEventsByTagID(tagID uuid.UUID) ([]uuid.UUID, error)
	ReplaceEventTags(eventID uuid.UUID, tagIDs []uuid.UUID) error

	GetTagsByNames(names []string) ([]Tag, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(tag *Tag) error {
	return r.db.Create(tag).Error
}

func (r *repository) GetByID(id uuid.UUID) (*Tag, error) {
	var tag Tag
	err := r.db.Where("id = ?", id).First(&tag).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

func (r *repository) GetBySlug(slug string) (*Tag, error) {
	var tag Tag
	err := r.db.Where("slug = ?", slug).First(&tag).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

func (r *repository) Update(id uuid.UUID, updates map[string]interface{}) (*Tag, error) {
	var tag Tag

	if err := r.db.Where("id = ?", id).First(&tag).Error; err != nil {
		return nil, err
	}

	if err := r.db.Model(&tag).Updates(updates).Error; err != nil {
		return nil, err
	}

	if err := r.db.Where("id = ?", id).First(&tag).Error; err != nil {
		return nil, err
	}

	return &tag, nil
}

func (r *repository) Delete(id uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// remove all event-tag relationships
		if err := tx.Where("tag_id = ?", id).Delete(&EventTag{}).Error; err != nil {
			return err
		}

		// Then delete the tag
		return tx.Where("id = ?", id).Delete(&Tag{}).Error
	})
}

func (r *repository) GetAll(query TagListQuery) ([]Tag, int64, error) {
	var tags []Tag
	var totalCount int64

	// Build the query
	db := r.db.Model(&Tag{})

	// Apply filters
	if query.Search != "" {
		searchTerm := "%" + strings.ToLower(query.Search) + "%"
		db = db.Where("LOWER(name) LIKE ? OR LOWER(description) LIKE ?", searchTerm, searchTerm)
	}

	if query.IsActive != nil {
		db = db.Where("is_active = ?", *query.IsActive)
	}

	// Count total records
	if err := db.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	sortBy := "created_at"
	sortOrder := "desc"

	if query.SortBy != "" {
		sortBy = query.SortBy
	}
	if query.SortOrder != "" {
		sortOrder = query.SortOrder
	}

	// Set defaults for pagination
	if query.Page == 0 {
		query.Page = 1
	}
	if query.Limit == 0 {
		query.Limit = 10
	}

	offset := (query.Page - 1) * query.Limit

	// Get paginated results
	err := db.Order(fmt.Sprintf("%s %s", sortBy, sortOrder)).
		Offset(offset).
		Limit(query.Limit).
		Find(&tags).Error

	return tags, totalCount, err
}

func (r *repository) GetActive() ([]Tag, error) {
	var tags []Tag
	err := r.db.Where("is_active = ?", true).Order("name ASC").Find(&tags).Error
	return tags, err
}

func (r *repository) AssignTagsToEvent(eventID uuid.UUID, tagIDs []uuid.UUID) error {
	if len(tagIDs) == 0 {
		return nil
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, tagID := range tagIDs {
			// Check if relationship already exists
			var existing EventTag
			err := tx.Where("event_id = ? AND tag_id = ?", eventID, tagID).First(&existing).Error
			if err == nil {
				// Relationship already exists, skip
				continue
			}
			if err != gorm.ErrRecordNotFound {
				return err
			}

			// Create new relationship
			eventTag := EventTag{
				EventID: eventID,
				TagID:   tagID,
			}
			if err := tx.Create(&eventTag).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *repository) RemoveTagsFromEvent(eventID uuid.UUID, tagIDs []uuid.UUID) error {
	if len(tagIDs) == 0 {
		return nil
	}

	return r.db.Where("event_id = ? AND tag_id IN ?", eventID, tagIDs).Delete(&EventTag{}).Error
}

func (r *repository) GetTagsByEventID(eventID uuid.UUID) ([]Tag, error) {
	var tags []Tag

	err := r.db.Table("tags").
		Joins("JOIN event_tags ON tags.id = event_tags.tag_id").
		Where("event_tags.event_id = ? AND tags.is_active = ?", eventID, true).
		Find(&tags).Error

	return tags, err
}

func (r *repository) GetEventsByTagID(tagID uuid.UUID) ([]uuid.UUID, error) {
	var eventIDs []uuid.UUID

	err := r.db.Table("event_tags").
		Select("event_id").
		Where("tag_id = ?", tagID).
		Pluck("event_id", &eventIDs).Error

	return eventIDs, err
}

func (r *repository) ReplaceEventTags(eventID uuid.UUID, tagIDs []uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Remove all existing tags for this event
		if err := tx.Where("event_id = ?", eventID).Delete(&EventTag{}).Error; err != nil {
			return err
		}

		// Add new tags if any
		if len(tagIDs) > 0 {
			for _, tagID := range tagIDs {
				eventTag := EventTag{
					EventID: eventID,
					TagID:   tagID,
				}
				if err := tx.Create(&eventTag).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func (r *repository) GetTagsByNames(names []string) ([]Tag, error) {
	var tags []Tag
	if len(names) == 0 {
		return tags, nil
	}

	err := r.db.Where("slug IN ? AND is_active = ?", names, true).Find(&tags).Error
	return tags, err
}
