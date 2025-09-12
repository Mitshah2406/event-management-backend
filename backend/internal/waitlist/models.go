package waitlist

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// JSONMap represents a JSON map type that can be stored in the database
type JSONMap map[string]interface{}

// Value implements the driver.Valuer interface for database storage
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface for database retrieval
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(bytes, j)
}

// GormDataType tells GORM how to handle this type
func (JSONMap) GormDataType() string {
	return "jsonb"
}

// WaitlistStatus represents the status of a waitlist entry
type WaitlistStatus string

const (
	WaitlistStatusActive    WaitlistStatus = "ACTIVE"
	WaitlistStatusNotified  WaitlistStatus = "NOTIFIED"
	WaitlistStatusExpired   WaitlistStatus = "EXPIRED"
	WaitlistStatusConverted WaitlistStatus = "CONVERTED"
	WaitlistStatusCancelled WaitlistStatus = "CANCELLED"
)

// NotificationChannel represents the channel for notifications
type NotificationChannel string

const (
	NotificationChannelEmail NotificationChannel = "EMAIL"
	NotificationChannelSMS   NotificationChannel = "SMS"
	NotificationChannelPush  NotificationChannel = "PUSH"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypeSpotAvailable  NotificationType = "SPOT_AVAILABLE"
	NotificationTypePositionUpdate NotificationType = "POSITION_UPDATE"
	NotificationTypeReminder       NotificationType = "REMINDER"
	NotificationTypeExpired        NotificationType = "EXPIRED"
)

// NotificationStatus represents the status of a notification
type NotificationStatus string

const (
	NotificationStatusPending NotificationStatus = "PENDING"
	NotificationStatusSent    NotificationStatus = "SENT"
	NotificationStatusFailed  NotificationStatus = "FAILED"
	NotificationStatusRetry   NotificationStatus = "RETRY"
)

// WaitlistEntry represents a user's position in an event waitlist
type WaitlistEntry struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()" db:"id"`
	UserID      uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;index" db:"user_id"`
	EventID     uuid.UUID      `json:"event_id" gorm:"type:uuid;not null;index" db:"event_id"`
	Position    int            `json:"position" gorm:"not null;index" db:"position"`
	Quantity    int            `json:"quantity" gorm:"not null" db:"quantity"`
	Status      WaitlistStatus `json:"status" gorm:"type:varchar(20);not null;index" db:"status"`
	Preferences JSONMap        `json:"preferences" gorm:"type:jsonb" db:"preferences"`
	JoinedAt    time.Time      `json:"joined_at" gorm:"not null" db:"joined_at"`
	NotifiedAt  *time.Time     `json:"notified_at,omitempty" db:"notified_at"`
	ExpiresAt   *time.Time     `json:"expires_at,omitempty" db:"expires_at"`
	CreatedAt   time.Time      `json:"created_at" gorm:"autoCreateTime" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"autoUpdateTime" db:"updated_at"`
}

// WaitlistNotification represents a notification sent to a waitlist user
type WaitlistNotification struct {
	ID               uuid.UUID           `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()" db:"id"`
	WaitlistEntryID  uuid.UUID           `json:"waitlist_entry_id" gorm:"type:uuid;not null;index" db:"waitlist_entry_id"`
	NotificationType NotificationType    `json:"notification_type" gorm:"type:varchar(50);not null" db:"notification_type"`
	Channel          NotificationChannel `json:"channel" gorm:"type:varchar(20);not null" db:"channel"`
	Status           NotificationStatus  `json:"status" gorm:"type:varchar(20);not null;index" db:"status"`
	MessageID        *string             `json:"message_id,omitempty" db:"message_id"`
	ErrorMessage     *string             `json:"error_message,omitempty" db:"error_message"`
	SentAt           *time.Time          `json:"sent_at,omitempty" db:"sent_at"`
	CreatedAt        time.Time           `json:"created_at" gorm:"autoCreateTime" db:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at" gorm:"autoUpdateTime" db:"updated_at"`
}

// WaitlistAnalytics represents daily analytics for waitlist operations
type WaitlistAnalytics struct {
	ID                 uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()" db:"id"`
	EventID            uuid.UUID `json:"event_id" gorm:"type:uuid;not null;index" db:"event_id"`
	Date               time.Time `json:"date" gorm:"type:date;not null;uniqueIndex:idx_event_date" db:"date"`
	TotalJoined        int       `json:"total_joined" gorm:"default:0" db:"total_joined"`
	TotalLeft          int       `json:"total_left" gorm:"default:0" db:"total_left"`
	TotalNotified      int       `json:"total_notified" gorm:"default:0" db:"total_notified"`
	TotalConverted     int       `json:"total_converted" gorm:"default:0" db:"total_converted"`
	AvgWaitTimeMinutes *int      `json:"avg_wait_time_minutes,omitempty" db:"avg_wait_time_minutes"`
	PeakQueueLength    int       `json:"peak_queue_length" gorm:"default:0" db:"peak_queue_length"`
	CreatedAt          time.Time `json:"created_at" gorm:"autoCreateTime" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" gorm:"autoUpdateTime" db:"updated_at"`
}

// Note: Kafka-related types have been removed as part of notification system simplification.
// Waitlist notifications now use the unified notification system directly.

// Request/Response Models

// JoinWaitlistRequest represents a request to join a waitlist
type JoinWaitlistRequest struct {
	EventID     uuid.UUID `json:"event_id" validate:"required"`
	Quantity    int       `json:"quantity" validate:"required,min=1,max=10"`
	Preferences JSONMap   `json:"preferences,omitempty"`
}

// WaitlistResponse represents the response when joining/checking waitlist status
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

// WaitlistStatsResponse represents waitlist statistics for an event
type WaitlistStatsResponse struct {
	EventID         uuid.UUID `json:"event_id"`
	TotalInQueue    int       `json:"total_in_queue"`
	ActiveInQueue   int       `json:"active_in_queue"`
	NotifiedCount   int       `json:"notified_count"`
	ConvertedCount  int       `json:"converted_count"`
	AverageWaitTime *int      `json:"average_wait_time_minutes,omitempty"`
}

// Redis Key Helpers

// GetQueueKey returns the Redis key for an event's waitlist queue
func GetQueueKey(eventID uuid.UUID) string {
	return "waitlist:queue:" + eventID.String()
}

// GetPositionKey returns the Redis key for tracking positions
func GetPositionKey(eventID uuid.UUID) string {
	return "waitlist:positions:" + eventID.String()
}

// GetStatsKey returns the Redis key for event waitlist statistics
func GetStatsKey(eventID uuid.UUID) string {
	return "waitlist:stats:" + eventID.String()
}

// GetLockKey returns the Redis key for distributed locking
func GetLockKey(eventID uuid.UUID) string {
	return "waitlist:lock:" + eventID.String()
}

// Kafka message helpers have been removed as part of notification system simplification.

// Validation Methods

// IsValidStatus checks if the waitlist status is valid
func (ws WaitlistStatus) IsValid() bool {
	switch ws {
	case WaitlistStatusActive, WaitlistStatusNotified, WaitlistStatusExpired, WaitlistStatusConverted, WaitlistStatusCancelled:
		return true
	default:
		return false
	}
}

// CanTransitionTo checks if the status can transition to the target status
func (ws WaitlistStatus) CanTransitionTo(target WaitlistStatus) bool {
	validTransitions := map[WaitlistStatus][]WaitlistStatus{
		WaitlistStatusActive:    {WaitlistStatusNotified, WaitlistStatusCancelled},
		WaitlistStatusNotified:  {WaitlistStatusConverted, WaitlistStatusExpired, WaitlistStatusCancelled},
		WaitlistStatusExpired:   {WaitlistStatusCancelled},
		WaitlistStatusConverted: {}, // Terminal state
		WaitlistStatusCancelled: {}, // Terminal state
	}

	allowedTargets := validTransitions[ws]
	for _, allowed := range allowedTargets {
		if allowed == target {
			return true
		}
	}
	return false
}

// IsActive returns true if the waitlist entry is in active status
func (we *WaitlistEntry) IsActive() bool {
	return we.Status == WaitlistStatusActive
}

// IsNotified returns true if the user has been notified of availability
func (we *WaitlistEntry) IsNotified() bool {
	return we.Status == WaitlistStatusNotified
}

// IsExpired returns true if the booking window has expired
func (we *WaitlistEntry) IsExpired() bool {
	return we.Status == WaitlistStatusExpired ||
		(we.ExpiresAt != nil && time.Now().After(*we.ExpiresAt))
}

// TimeRemaining returns the time remaining in the booking window
func (we *WaitlistEntry) TimeRemaining() *time.Duration {
	if we.ExpiresAt == nil {
		return nil
	}
	remaining := time.Until(*we.ExpiresAt)
	if remaining < 0 {
		return nil
	}
	return &remaining
}

// Configuration Constants

const (
	// BookingWindowDuration is the default time window for users to book after notification
	BookingWindowDuration = 15 * time.Minute

	// MaxWaitlistSize is the maximum number of users allowed in a single waitlist
	MaxWaitlistSize = 10000

	// MaxQuantityPerUser is the maximum quantity a single user can request
	MaxQuantityPerUser = 10

	// PositionUpdateBatchSize is the number of positions to update in a single batch
	PositionUpdateBatchSize = 100

	// NotificationRetryLimit is the maximum number of retry attempts for notifications
	NotificationRetryLimit = 3

	// RedisKeyTTL is the TTL for Redis keys (24 hours)
	RedisKeyTTL = 24 * time.Hour
)
