package tags

import (
	"time"

	"github.com/google/uuid"
)

// Tag represents the normalized tag entity
type Tag struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	Name        string     `json:"name" gorm:"uniqueIndex;not null;size:100"`
	Slug        string     `json:"slug" gorm:"uniqueIndex;not null;size:100"`
	Description string     `json:"description" gorm:"size:500"`
	Color       string     `json:"color" gorm:"size:7;default:'#6B7280'"` // Hex color code
	IsActive    bool       `json:"is_active" gorm:"default:true"`
	CreatedBy   uuid.UUID  `json:"created_by" gorm:"type:uuid;not null"`
	UpdatedBy   *uuid.UUID `json:"updated_by" gorm:"type:uuid"`
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

// EventTag represents the many-to-many relationship between events and tags
type EventTag struct {
	ID      uuid.UUID `json:"id" gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	EventID uuid.UUID `json:"event_id" gorm:"type:uuid;not null;index;uniqueIndex:idx_event_tag_unique"`
	TagID   uuid.UUID `json:"tag_id" gorm:"type:uuid;not null;index;uniqueIndex:idx_event_tag_unique"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// Response DTOs
type TagResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Color       string    `json:"color"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateTagRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=100"`
	Description string `json:"description" binding:"max=500"`
	Color       string `json:"color" binding:"omitempty,len=7"` // Hex color validation
}

type UpdateTagRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=2,max=100"`
	Description *string `json:"description" binding:"omitempty,max=500"`
	Color       *string `json:"color" binding:"omitempty,len=7"`
	IsActive    *bool   `json:"is_active"`
}

type TagListQuery struct {
	Page      int    `form:"page" binding:"omitempty,min=1"`
	Limit     int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Search    string `form:"search"`
	IsActive  *bool  `form:"is_active"`
	SortBy    string `form:"sort_by" binding:"omitempty,oneof=name created_at updated_at"`
	SortOrder string `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

type PaginatedTags struct {
	Tags       []TagResponse `json:"tags"`
	TotalCount int64         `json:"total_count"`
	Page       int           `json:"page"`
	Limit      int           `json:"limit"`
	TotalPages int           `json:"total_pages"`
}

// Tag Analytics
type TagAnalytics struct {
	TagID           string  `json:"tag_id"`
	TagName         string  `json:"tag_name"`
	EventCount      int     `json:"event_count"`
	TotalBookings   int     `json:"total_bookings"`
	TotalRevenue    float64 `json:"total_revenue"`
	AvgUtilization  float64 `json:"avg_utilization"`
	PopularityScore float64 `json:"popularity_score"` // Calculated metric
}

type TagAnalyticsResponse struct {
	Overview    TagOverview     `json:"overview"`
	TopTags     []TagAnalytics  `json:"top_tags"`
	TagTrends   []TagTrend      `json:"tag_trends"`
	Comparisons []TagComparison `json:"comparisons"`
}

type TagOverview struct {
	TotalTags       int     `json:"total_tags"`
	ActiveTags      int     `json:"active_tags"`
	TagsWithEvents  int     `json:"tags_with_events"`
	AvgTagsPerEvent float64 `json:"avg_tags_per_event"`
	MostPopularTag  string  `json:"most_popular_tag"`
	LeastUsedTag    string  `json:"least_used_tag"`
}

type TagTrend struct {
	TagID      string  `json:"tag_id"`
	TagName    string  `json:"tag_name"`
	Month      string  `json:"month"`
	EventCount int     `json:"event_count"`
	Revenue    float64 `json:"revenue"`
}

type TagComparison struct {
	TagID             string  `json:"tag_id"`
	TagName           string  `json:"tag_name"`
	EventCount        int     `json:"event_count"`
	AvgCapacityUtil   float64 `json:"avg_capacity_utilization"`
	AvgTicketPrice    float64 `json:"avg_ticket_price"`
	TotalRevenue      float64 `json:"total_revenue"`
	BookingConversion float64 `json:"booking_conversion"`
}

// Helper methods
func (t *Tag) ToResponse() TagResponse {
	return TagResponse{
		ID:          t.ID.String(),
		Name:        t.Name,
		Slug:        t.Slug,
		Description: t.Description,
		Color:       t.Color,
		IsActive:    t.IsActive,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

// TableName specifies the table name for GORM
func (Tag) TableName() string {
	return "tags"
}

func (EventTag) TableName() string {
	return "event_tags"
}
