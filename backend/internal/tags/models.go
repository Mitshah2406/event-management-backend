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
