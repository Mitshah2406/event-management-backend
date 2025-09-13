package waitlist

import (
	"time"

	"github.com/google/uuid"
)

type WaitlistResponse struct {
	ID            uuid.UUID      `json:"id"`
	EventID       uuid.UUID      `json:"event_id"`
	Position      int            `json:"position"`
	Quantity      int            `json:"quantity"`
	Status        WaitlistStatus `json:"status"`
	EstimatedWait *time.Duration `json:"estimated_wait,omitempty"`
	Preferences   JSONMap        `json:"preferences,omitempty"`
	JoinedAt      time.Time      `json:"joined_at"`
	NotifiedAt    *time.Time     `json:"notified_at,omitempty"`
	ExpiresAt     *time.Time     `json:"expires_at,omitempty"`
}

type WaitlistStatsResponse struct {
	EventID         uuid.UUID `json:"event_id"`
	TotalInQueue    int       `json:"total_in_queue"`
	ActiveInQueue   int       `json:"active_in_queue"`
	NotifiedCount   int       `json:"notified_count"`
	ConvertedCount  int       `json:"converted_count"`
	AverageWaitTime *int      `json:"average_wait_time_minutes,omitempty"`
}
