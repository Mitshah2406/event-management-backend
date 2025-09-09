package tags

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	// Basic CRUD operations
	Create(tag *Tag) error
	GetByID(id uuid.UUID) (*Tag, error)
	GetBySlug(slug string) (*Tag, error)
	Update(id uuid.UUID, updates map[string]interface{}) (*Tag, error)
	Delete(id uuid.UUID) error
	GetAll(query TagListQuery) ([]Tag, int64, error)
	GetActive() ([]Tag, error)

	// Event-Tag relationship operations
	AssignTagsToEvent(eventID uuid.UUID, tagIDs []uuid.UUID) error
	RemoveTagsFromEvent(eventID uuid.UUID, tagIDs []uuid.UUID) error
	GetTagsByEventID(eventID uuid.UUID) ([]Tag, error)
	GetEventsByTagID(tagID uuid.UUID) ([]uuid.UUID, error)
	ReplaceEventTags(eventID uuid.UUID, tagIDs []uuid.UUID) error

	// Analytics operations
	GetTagAnalytics() (*TagAnalyticsResponse, error)
	GetTagPopularityAnalytics() ([]TagAnalytics, error)
	GetTagTrends(months int) ([]TagTrend, error)
	GetTagComparisons() ([]TagComparison, error)
	GetTagsByNames(names []string) ([]Tag, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// Basic CRUD operations

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

	// First, get the current tag
	if err := r.db.Where("id = ?", id).First(&tag).Error; err != nil {
		return nil, err
	}

	// Update the tag
	if err := r.db.Model(&tag).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Return updated tag
	if err := r.db.Where("id = ?", id).First(&tag).Error; err != nil {
		return nil, err
	}

	return &tag, nil
}

func (r *repository) Delete(id uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// First, remove all event-tag relationships
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

// Event-Tag relationship operations

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

// Analytics operations

func (r *repository) GetTagAnalytics() (*TagAnalyticsResponse, error) {
	analytics := &TagAnalyticsResponse{}

	// Get overview
	overview, err := r.getTagOverview()
	if err != nil {
		return nil, fmt.Errorf("failed to get tag overview: %w", err)
	}
	analytics.Overview = *overview

	// Get top tags by popularity
	topTags, err := r.GetTagPopularityAnalytics()
	if err != nil {
		return nil, fmt.Errorf("failed to get tag popularity: %w", err)
	}
	analytics.TopTags = topTags

	// Get tag trends (last 6 months)
	trends, err := r.GetTagTrends(6)
	if err != nil {
		return nil, fmt.Errorf("failed to get tag trends: %w", err)
	}
	analytics.TagTrends = trends

	// Get tag comparisons
	comparisons, err := r.GetTagComparisons()
	if err != nil {
		return nil, fmt.Errorf("failed to get tag comparisons: %w", err)
	}
	analytics.Comparisons = comparisons

	return analytics, nil
}

func (r *repository) getTagOverview() (*TagOverview, error) {
	var overview TagOverview

	// Total tags
	var totalTags int64
	if err := r.db.Model(&Tag{}).Count(&totalTags).Error; err != nil {
		return nil, err
	}
	overview.TotalTags = int(totalTags)

	// Active tags
	var activeTags int64
	if err := r.db.Model(&Tag{}).Where("is_active = ?", true).Count(&activeTags).Error; err != nil {
		return nil, err
	}
	overview.ActiveTags = int(activeTags)

	// Tags with events
	var tagsWithEvents int64
	if err := r.db.Table("tags").
		Joins("JOIN event_tags ON tags.id = event_tags.tag_id").
		Where("tags.is_active = ?", true).
		Distinct("tags.id").
		Count(&tagsWithEvents).Error; err != nil {
		return nil, err
	}
	overview.TagsWithEvents = int(tagsWithEvents)

	// Average tags per event
	type avgResult struct {
		AvgTags float64 `json:"avg_tags"`
	}
	var avgRes avgResult
	if err := r.db.Raw(`
		SELECT AVG(tag_count) as avg_tags FROM (
			SELECT COUNT(event_tags.tag_id) as tag_count 
			FROM events 
			LEFT JOIN event_tags ON events.id = event_tags.event_id 
			GROUP BY events.id
		) as subquery
	`).Scan(&avgRes).Error; err != nil {
		return nil, err
	}
	overview.AvgTagsPerEvent = math.Round(avgRes.AvgTags*100) / 100

	// Most and least popular tags
	type popularityResult struct {
		TagName    string `json:"tag_name"`
		EventCount int    `json:"event_count"`
	}

	// Most popular
	var mostPopular popularityResult
	if err := r.db.Table("tags").
		Select("tags.name as tag_name, COUNT(event_tags.event_id) as event_count").
		Joins("JOIN event_tags ON tags.id = event_tags.tag_id").
		Where("tags.is_active = ?", true).
		Group("tags.id, tags.name").
		Order("event_count DESC").
		Limit(1).
		Scan(&mostPopular).Error; err == nil && mostPopular.TagName != "" {
		overview.MostPopularTag = mostPopular.TagName
	}

	// Least popular (but has events)
	var leastPopular popularityResult
	if err := r.db.Table("tags").
		Select("tags.name as tag_name, COUNT(event_tags.event_id) as event_count").
		Joins("JOIN event_tags ON tags.id = event_tags.tag_id").
		Where("tags.is_active = ?", true).
		Group("tags.id, tags.name").
		Having("COUNT(event_tags.event_id) > 0").
		Order("event_count ASC").
		Limit(1).
		Scan(&leastPopular).Error; err == nil && leastPopular.TagName != "" {
		overview.LeastUsedTag = leastPopular.TagName
	}

	return &overview, nil
}

func (r *repository) GetTagPopularityAnalytics() ([]TagAnalytics, error) {
	var analytics []TagAnalytics

	query := `
		SELECT 
			t.id as tag_id,
			t.name as tag_name,
			COUNT(DISTINCT et.event_id) as event_count,
			COALESCE(SUM(e.booked_count), 0) as total_bookings,
			COALESCE(SUM(e.booked_count * e.price), 0) as total_revenue,
			COALESCE(AVG(
				CASE 
					WHEN e.total_capacity > 0 
					THEN (e.booked_count * 100.0 / e.total_capacity) 
					ELSE 0 
				END
			), 0) as avg_utilization,
			-- Popularity score: weighted combination of events, bookings, and revenue
			(COUNT(DISTINCT et.event_id) * 0.3 + 
			 COALESCE(SUM(e.booked_count), 0) * 0.4 + 
			 COALESCE(SUM(e.booked_count * e.price), 0) / 1000 * 0.3) as popularity_score
		FROM tags t
		LEFT JOIN event_tags et ON t.id = et.tag_id
		LEFT JOIN events e ON et.event_id = e.id
		WHERE t.is_active = true
		GROUP BY t.id, t.name
		HAVING COUNT(DISTINCT et.event_id) > 0
		ORDER BY popularity_score DESC
		LIMIT 10
	`

	err := r.db.Raw(query).Scan(&analytics).Error
	return analytics, err
}

func (r *repository) GetTagTrends(months int) ([]TagTrend, error) {
	var trends []TagTrend

	query := `
		SELECT 
			t.id as tag_id,
			t.name as tag_name,
			TO_CHAR(e.date_time, 'YYYY-MM') as month,
			COUNT(DISTINCT e.id) as event_count,
			COALESCE(SUM(e.booked_count * e.price), 0) as revenue
		FROM tags t
		JOIN event_tags et ON t.id = et.tag_id
		JOIN events e ON et.event_id = e.id
		WHERE t.is_active = true 
		AND e.date_time >= ?
		GROUP BY t.id, t.name, TO_CHAR(e.date_time, 'YYYY-MM')
		ORDER BY month DESC, revenue DESC
	`

	startDate := time.Now().AddDate(0, -months, 0)
	err := r.db.Raw(query, startDate).Scan(&trends).Error
	return trends, err
}

func (r *repository) GetTagComparisons() ([]TagComparison, error) {
	var comparisons []TagComparison

	query := `
		SELECT 
			t.id as tag_id,
			t.name as tag_name,
			COUNT(DISTINCT et.event_id) as event_count,
			COALESCE(AVG(
				CASE 
					WHEN e.total_capacity > 0 
					THEN (e.booked_count * 100.0 / e.total_capacity) 
					ELSE 0 
				END
			), 0) as avg_capacity_utilization,
			COALESCE(AVG(e.price), 0) as avg_ticket_price,
			COALESCE(SUM(e.booked_count * e.price), 0) as total_revenue,
			-- Booking conversion: average of (booked_count / total_capacity) across events
			COALESCE(AVG(
				CASE 
					WHEN e.total_capacity > 0 
					THEN (e.booked_count * 1.0 / e.total_capacity) 
					ELSE 0 
				END
			) * 100, 0) as booking_conversion
		FROM tags t
		JOIN event_tags et ON t.id = et.tag_id
		JOIN events e ON et.event_id = e.id
		WHERE t.is_active = true
		GROUP BY t.id, t.name
		HAVING COUNT(DISTINCT et.event_id) > 0
		ORDER BY total_revenue DESC
		LIMIT 20
	`

	err := r.db.Raw(query).Scan(&comparisons).Error
	return comparisons, err
}
