package notifications

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type NotificationType string

const (
	NotificationTypeWaitlistSpotAvailable  NotificationType = "WAITLIST_SPOT_AVAILABLE"
	NotificationTypeWaitlistPositionUpdate NotificationType = "WAITLIST_POSITION_UPDATE"
	NotificationTypeWaitlistReminder       NotificationType = "WAITLIST_REMINDER"
	NotificationTypeWaitlistExpired        NotificationType = "WAITLIST_EXPIRED"

	NotificationTypeBookingConfirmed NotificationType = "BOOKING_CONFIRMED"
	NotificationTypeWelcome          NotificationType = "WELCOME"
)

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
	NotificationStatusCancelled NotificationStatus = "CANCELLED"
)

type UnifiedNotification struct {
	ID       uuid.UUID             `json:"id"`
	Type     NotificationType      `json:"type"`
	Priority NotificationPriority  `json:"priority"`
	Channels []NotificationChannel `json:"channels"`

	RecipientID    uuid.UUID `json:"recipient_id"`
	RecipientEmail string    `json:"recipient_email"`
	RecipientPhone *string   `json:"recipient_phone,omitempty"`
	RecipientName  string    `json:"recipient_name"`

	Subject      string                 `json:"subject"`
	TemplateID   string                 `json:"template_id"`
	TemplateData map[string]interface{} `json:"template_data"`

	EventID         *uuid.UUID `json:"event_id,omitempty"`
	BookingID       *uuid.UUID `json:"booking_id,omitempty"`
	WaitlistEntryID *uuid.UUID `json:"waitlist_entry_id,omitempty"`

	ScheduledFor *time.Time `json:"scheduled_for,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`

	Status           NotificationStatus `json:"status"`
	RetryCount       int                `json:"retry_count"`
	MaxRetries       int                `json:"max_retries"`
	LastError        *string            `json:"last_error,omitempty"`
	DeliveryAttempts []DeliveryAttempt  `json:"delivery_attempts,omitempty"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
	SentAt           *time.Time         `json:"sent_at,omitempty"`
	DeliveredAt      *time.Time         `json:"delivered_at,omitempty"`
}

type DeliveryAttempt struct {
	Channel     NotificationChannel `json:"channel"`
	AttemptedAt time.Time           `json:"attempted_at"`
	Status      NotificationStatus  `json:"status"`
	Error       *string             `json:"error,omitempty"`
	MessageID   *string             `json:"message_id,omitempty"`
}

type NotificationTemplate struct {
	ID        string              `json:"id"`
	Type      NotificationType    `json:"type"`
	Channel   NotificationChannel `json:"channel"`
	Subject   string              `json:"subject"`
	HTMLBody  string              `json:"html_body"`
	TextBody  string              `json:"text_body"`
	Variables []string            `json:"variables"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
}

type EventData struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	VenueID     uuid.UUID `json:"venue_id"`
	VenueName   string    `json:"venue_name"`
	Price       float64   `json:"price"`
}

type BookingData struct {
	ID            uuid.UUID `json:"id"`
	EventID       uuid.UUID `json:"event_id"`
	UserID        uuid.UUID `json:"user_id"`
	Quantity      int       `json:"quantity"`
	TotalAmount   float64   `json:"total_amount"`
	BookingNumber string    `json:"booking_number"`
	Status        string    `json:"status"`
}

type UserData struct {
	ID          uuid.UUID              `json:"id"`
	Email       string                 `json:"email"`
	FirstName   string                 `json:"first_name"`
	LastName    string                 `json:"last_name"`
	PhoneNumber *string                `json:"phone_number,omitempty"`
	Preferences map[string]interface{} `json:"preferences,omitempty"`
}

type WaitlistData struct {
	ID        uuid.UUID  `json:"id"`
	Position  int        `json:"position"`
	Quantity  int        `json:"quantity"`
	JoinedAt  time.Time  `json:"joined_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type NotificationBuilder struct {
	notification *UnifiedNotification
}

func NewNotificationBuilder() *NotificationBuilder {
	return &NotificationBuilder{
		notification: &UnifiedNotification{
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

func (nb *NotificationBuilder) WithPhone(phone string) *NotificationBuilder {
	nb.notification.RecipientPhone = &phone
	return nb
}
func (nb *NotificationBuilder) WithChannels(channels ...NotificationChannel) *NotificationBuilder {
	nb.notification.Channels = channels
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
func (nb *NotificationBuilder) WithTemplate(templateID string, data map[string]interface{}) *NotificationBuilder {
	nb.notification.TemplateID = templateID
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

func (nb *NotificationBuilder) WithScheduling(scheduledFor *time.Time, expiresAt *time.Time) *NotificationBuilder {
	nb.notification.ScheduledFor = scheduledFor
	nb.notification.ExpiresAt = expiresAt
	return nb
}

func (nb *NotificationBuilder) WithMaxRetries(maxRetries int) *NotificationBuilder {
	nb.notification.MaxRetries = maxRetries
	return nb
}

func (nb *NotificationBuilder) Build() *UnifiedNotification {
	return nb.notification
}

func GetDefaultPriority(notType NotificationType) NotificationPriority {
	switch notType {
	case NotificationTypeWaitlistSpotAvailable:
		return NotificationPriorityHigh

	case NotificationTypeWaitlistExpired,
		NotificationTypeBookingConfirmed:
		return NotificationPriorityMedium

	case NotificationTypeWaitlistPositionUpdate:
		return NotificationPriorityLow

	default:
		return NotificationPriorityMedium
	}
}

func GetDefaultChannels(notType NotificationType) []NotificationChannel {
	return []NotificationChannel{NotificationChannelEmail}
}

func (un *UnifiedNotification) GetPartitionKey() string {
	return un.RecipientID.String()
}

func (un *UnifiedNotification) ToJSON() ([]byte, error) {
	return json.Marshal(un)
}

func (un *UnifiedNotification) IsExpired() bool {
	return un.ExpiresAt != nil && time.Now().After(*un.ExpiresAt)
}

func (un *UnifiedNotification) ShouldRetry() bool {
	return un.RetryCount < un.MaxRetries &&
		un.Status == NotificationStatusFailed &&
		!un.IsExpired()
}

func (un *UnifiedNotification) MarkDelivered(channel NotificationChannel, messageID *string) {
	now := time.Now()
	un.Status = NotificationStatusDelivered
	un.DeliveredAt = &now
	un.UpdatedAt = now

	attempt := DeliveryAttempt{
		Channel:     channel,
		AttemptedAt: now,
		Status:      NotificationStatusDelivered,
		MessageID:   messageID,
	}
	un.DeliveryAttempts = append(un.DeliveryAttempts, attempt)
}

func (un *UnifiedNotification) MarkFailed(channel NotificationChannel, err error) {
	now := time.Now()
	un.Status = NotificationStatusFailed
	un.UpdatedAt = now

	errorStr := err.Error()
	un.LastError = &errorStr

	attempt := DeliveryAttempt{
		Channel:     channel,
		AttemptedAt: now,
		Status:      NotificationStatusFailed,
		Error:       &errorStr,
	}
	un.DeliveryAttempts = append(un.DeliveryAttempts, attempt)
}

func (un *UnifiedNotification) IncrementRetry() {
	un.RetryCount++
	un.UpdatedAt = time.Now()
	if un.ShouldRetry() {
		un.Status = NotificationStatusRetrying
	} else {
		un.Status = NotificationStatusExpired
	}
}
