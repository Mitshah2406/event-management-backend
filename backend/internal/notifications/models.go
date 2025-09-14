package notifications

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Simplified notification types - only the ones actually used
type NotificationType string

const (
	NotificationTypeWaitlistSpotAvailable  NotificationType = "WAITLIST_SPOT_AVAILABLE"
	NotificationTypeBookingConfirmed       NotificationType = "BOOKING_CONFIRMED"
	NotificationTypeWaitlistPositionUpdate NotificationType = "WAITLIST_POSITION_UPDATE"
)

// Only email channel since that's all that's implemented
type NotificationChannel string

const (
	NotificationChannelEmail NotificationChannel = "EMAIL"
)

type NotificationPriority string

const (
	NotificationPriorityLow      NotificationPriority = "LOW"
	NotificationPriorityMedium   NotificationPriority = "MEDIUM"
	NotificationPriorityHigh     NotificationPriority = "HIGH"
	NotificationPriorityCritical NotificationPriority = "CRITICAL"
)

type NotificationStatus string

const (
	NotificationStatusPending   NotificationStatus = "PENDING"
	NotificationStatusQueued    NotificationStatus = "QUEUED"
	NotificationStatusSending   NotificationStatus = "SENDING"
	NotificationStatusSent      NotificationStatus = "SENT"
	NotificationStatusDelivered NotificationStatus = "DELIVERED"
	NotificationStatusFailed    NotificationStatus = "FAILED"
	NotificationStatusRetrying  NotificationStatus = "RETRYING"
	NotificationStatusExpired   NotificationStatus = "EXPIRED"
)

// Simplified notification struct - removed unused fields
type EmailNotification struct {
	ID       uuid.UUID            `json:"id"`
	Type     NotificationType     `json:"type"`
	Priority NotificationPriority `json:"priority"`

	// Recipient info - removed phone since only email is supported
	RecipientID    uuid.UUID `json:"recipient_id"`
	RecipientEmail string    `json:"recipient_email"`
	RecipientName  string    `json:"recipient_name"`

	// Content
	Subject      string                 `json:"subject"`
	TemplateData map[string]interface{} `json:"template_data"`

	// Context - kept only the ones actually used
	EventID         *uuid.UUID `json:"event_id,omitempty"`
	BookingID       *uuid.UUID `json:"booking_id,omitempty"`
	WaitlistEntryID *uuid.UUID `json:"waitlist_entry_id,omitempty"`

	// Timing
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// Status tracking
	Status     NotificationStatus `json:"status"`
	RetryCount int                `json:"retry_count"`
	MaxRetries int                `json:"max_retries"`
	LastError  *string            `json:"last_error,omitempty"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
	SentAt     *time.Time         `json:"sent_at,omitempty"`
}

// Simplified builder pattern
type NotificationBuilder struct {
	notification *EmailNotification
}

func NewNotificationBuilder() *NotificationBuilder {
	return &NotificationBuilder{
		notification: &EmailNotification{
			ID:           uuid.New(),
			Status:       NotificationStatusPending,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			MaxRetries:   3,
			TemplateData: make(map[string]interface{}),
		},
	}
}

func (nb *NotificationBuilder) WithType(notType NotificationType) *NotificationBuilder {
	nb.notification.Type = notType
	nb.notification.Priority = GetDefaultPriority(notType)
	return nb
}

func (nb *NotificationBuilder) WithRecipient(userID uuid.UUID, email, name string) *NotificationBuilder {
	nb.notification.RecipientID = userID
	nb.notification.RecipientEmail = email
	nb.notification.RecipientName = name
	return nb
}

func (nb *NotificationBuilder) WithPriority(priority NotificationPriority) *NotificationBuilder {
	nb.notification.Priority = priority
	return nb
}

func (nb *NotificationBuilder) WithSubject(subject string) *NotificationBuilder {
	nb.notification.Subject = subject
	return nb
}

func (nb *NotificationBuilder) WithTemplateData(data map[string]interface{}) *NotificationBuilder {
	nb.notification.TemplateData = data
	return nb
}

func (nb *NotificationBuilder) WithEventContext(eventID uuid.UUID) *NotificationBuilder {
	nb.notification.EventID = &eventID
	return nb
}

func (nb *NotificationBuilder) WithBookingContext(bookingID uuid.UUID) *NotificationBuilder {
	nb.notification.BookingID = &bookingID
	return nb
}

func (nb *NotificationBuilder) WithWaitlistContext(waitlistEntryID uuid.UUID) *NotificationBuilder {
	nb.notification.WaitlistEntryID = &waitlistEntryID
	return nb
}

func (nb *NotificationBuilder) WithExpiration(expiresAt *time.Time) *NotificationBuilder {
	nb.notification.ExpiresAt = expiresAt
	return nb
}

func (nb *NotificationBuilder) WithMaxRetries(maxRetries int) *NotificationBuilder {
	nb.notification.MaxRetries = maxRetries
	return nb
}

func (nb *NotificationBuilder) Build() *EmailNotification {
	return nb.notification
}

// Helper functions
func GetDefaultPriority(notType NotificationType) NotificationPriority {
	switch notType {
	case NotificationTypeWaitlistSpotAvailable:
		return NotificationPriorityHigh
	case NotificationTypeBookingConfirmed:
		return NotificationPriorityMedium
	case NotificationTypeWaitlistPositionUpdate:
		return NotificationPriorityLow
	default:
		return NotificationPriorityMedium
	}
}

// Utility methods
func (en *EmailNotification) GetPartitionKey() string {
	return en.RecipientID.String()
}

func (en *EmailNotification) ToJSON() ([]byte, error) {
	return json.Marshal(en)
}

func (en *EmailNotification) IsExpired() bool {
	return en.ExpiresAt != nil && time.Now().After(*en.ExpiresAt)
}

func (en *EmailNotification) ShouldRetry() bool {
	return en.RetryCount < en.MaxRetries &&
		en.Status == NotificationStatusFailed &&
		!en.IsExpired()
}

func (en *EmailNotification) MarkSent() {
	now := time.Now()
	en.Status = NotificationStatusSent
	en.SentAt = &now
	en.UpdatedAt = now
}

func (en *EmailNotification) MarkFailed(err error) {
	now := time.Now()
	en.Status = NotificationStatusFailed
	en.UpdatedAt = now

	errorStr := err.Error()
	en.LastError = &errorStr
}

func (en *EmailNotification) IncrementRetry() {
	en.RetryCount++
	en.UpdatedAt = time.Now()
	if en.ShouldRetry() {
		en.Status = NotificationStatusRetrying
	} else {
		en.Status = NotificationStatusExpired
	}
}
