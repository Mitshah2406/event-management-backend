package notifications

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// NotificationType represents different types of notifications in the system
type NotificationType string

const (
	// Waitlist notifications
	NotificationTypeWaitlistSpotAvailable  NotificationType = "WAITLIST_SPOT_AVAILABLE"
	NotificationTypeWaitlistPositionUpdate NotificationType = "WAITLIST_POSITION_UPDATE"
	NotificationTypeWaitlistReminder       NotificationType = "WAITLIST_REMINDER"
	NotificationTypeWaitlistExpired        NotificationType = "WAITLIST_EXPIRED"

	// Booking notifications
	NotificationTypeBookingConfirmed NotificationType = "BOOKING_CONFIRMED"
	NotificationTypeBookingReminder  NotificationType = "BOOKING_REMINDER"
	NotificationTypeBookingCancelled NotificationType = "BOOKING_CANCELLED"
	NotificationTypeBookingUpdated   NotificationType = "BOOKING_UPDATED"
	NotificationTypeBookingRefunded  NotificationType = "BOOKING_REFUNDED"

	// Event notifications
	NotificationTypeEventUpdated      NotificationType = "EVENT_UPDATED"
	NotificationTypeEventCancelled    NotificationType = "EVENT_CANCELLED"
	NotificationTypeEventReminder     NotificationType = "EVENT_REMINDER"
	NotificationTypeEventStartingSoon NotificationType = "EVENT_STARTING_SOON"

	// Account notifications
	NotificationTypeWelcome        NotificationType = "WELCOME"
	NotificationTypePasswordReset  NotificationType = "PASSWORD_RESET"
	NotificationTypeAccountVerify  NotificationType = "ACCOUNT_VERIFY"
	NotificationTypeProfileUpdated NotificationType = "PROFILE_UPDATED"

	// Payment notifications
	NotificationTypePaymentSuccess  NotificationType = "PAYMENT_SUCCESS"
	NotificationTypePaymentFailed   NotificationType = "PAYMENT_FAILED"
	NotificationTypeRefundProcessed NotificationType = "REFUND_PROCESSED"
)

// NotificationChannel represents different delivery channels
type NotificationChannel string

const (
	NotificationChannelEmail NotificationChannel = "EMAIL"
	NotificationChannelSMS   NotificationChannel = "SMS"
	NotificationChannelPush  NotificationChannel = "PUSH"
	NotificationChannelInApp NotificationChannel = "IN_APP"
)

// NotificationPriority represents the priority level of notifications
type NotificationPriority string

const (
	NotificationPriorityLow      NotificationPriority = "LOW"
	NotificationPriorityMedium   NotificationPriority = "MEDIUM"
	NotificationPriorityHigh     NotificationPriority = "HIGH"
	NotificationPriorityCritical NotificationPriority = "CRITICAL"
)

// NotificationStatus represents the delivery status of a notification
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

// UnifiedNotification represents a notification message that can be sent through various channels
type UnifiedNotification struct {
	// Metadata
	ID       uuid.UUID             `json:"id"`
	Type     NotificationType      `json:"type"`
	Priority NotificationPriority  `json:"priority"`
	Channels []NotificationChannel `json:"channels"`

	// Recipient information
	RecipientID    uuid.UUID `json:"recipient_id"`
	RecipientEmail string    `json:"recipient_email"`
	RecipientPhone *string   `json:"recipient_phone,omitempty"`
	RecipientName  string    `json:"recipient_name"`

	// Content
	Subject      string                 `json:"subject"`
	TemplateID   string                 `json:"template_id"`
	TemplateData map[string]interface{} `json:"template_data"`

	// Context and relationships
	EventID         *uuid.UUID `json:"event_id,omitempty"`
	BookingID       *uuid.UUID `json:"booking_id,omitempty"`
	WaitlistEntryID *uuid.UUID `json:"waitlist_entry_id,omitempty"`

	// Scheduling and expiry
	ScheduledFor *time.Time `json:"scheduled_for,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`

	// Tracking
	Status           NotificationStatus `json:"status"`
	RetryCount       int                `json:"retry_count"`
	MaxRetries       int                `json:"max_retries"`
	LastError        *string            `json:"last_error,omitempty"`
	DeliveryAttempts []DeliveryAttempt  `json:"delivery_attempts,omitempty"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	SentAt      *time.Time `json:"sent_at,omitempty"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
}

// DeliveryAttempt tracks individual delivery attempts for notifications
type DeliveryAttempt struct {
	Channel     NotificationChannel `json:"channel"`
	AttemptedAt time.Time           `json:"attempted_at"`
	Status      NotificationStatus  `json:"status"`
	Error       *string             `json:"error,omitempty"`
	MessageID   *string             `json:"message_id,omitempty"`
}

// NotificationTemplate represents a template for generating notification content
type NotificationTemplate struct {
	ID        string              `json:"id"`
	Type      NotificationType    `json:"type"`
	Channel   NotificationChannel `json:"channel"`
	Subject   string              `json:"subject"`
	HTMLBody  string              `json:"html_body"`
	TextBody  string              `json:"text_body"`
	SMSBody   string              `json:"sms_body,omitempty"`
	Variables []string            `json:"variables"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
}

// EventData represents common event information used in notifications
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

// BookingData represents booking information used in notifications
type BookingData struct {
	ID            uuid.UUID `json:"id"`
	EventID       uuid.UUID `json:"event_id"`
	UserID        uuid.UUID `json:"user_id"`
	Quantity      int       `json:"quantity"`
	TotalAmount   float64   `json:"total_amount"`
	BookingNumber string    `json:"booking_number"`
	Status        string    `json:"status"`
}

// UserData represents user information used in notifications
type UserData struct {
	ID          uuid.UUID              `json:"id"`
	Email       string                 `json:"email"`
	FirstName   string                 `json:"first_name"`
	LastName    string                 `json:"last_name"`
	PhoneNumber *string                `json:"phone_number,omitempty"`
	Preferences map[string]interface{} `json:"preferences,omitempty"`
}

// WaitlistData represents waitlist information used in notifications
type WaitlistData struct {
	ID        uuid.UUID  `json:"id"`
	Position  int        `json:"position"`
	Quantity  int        `json:"quantity"`
	JoinedAt  time.Time  `json:"joined_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// NotificationBuilder helps build notifications with a fluent interface
type NotificationBuilder struct {
	notification *UnifiedNotification
}

// NewNotificationBuilder creates a new notification builder
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

// WithType sets the notification type
func (nb *NotificationBuilder) WithType(notType NotificationType) *NotificationBuilder {
	nb.notification.Type = notType
	nb.notification.Priority = GetDefaultPriority(notType)
	return nb
}

// WithRecipient sets the recipient information
func (nb *NotificationBuilder) WithRecipient(userID uuid.UUID, email, name string) *NotificationBuilder {
	nb.notification.RecipientID = userID
	nb.notification.RecipientEmail = email
	nb.notification.RecipientName = name
	return nb
}

// WithPhone sets the recipient phone number
func (nb *NotificationBuilder) WithPhone(phone string) *NotificationBuilder {
	nb.notification.RecipientPhone = &phone
	return nb
}

// WithChannels sets the notification channels
func (nb *NotificationBuilder) WithChannels(channels ...NotificationChannel) *NotificationBuilder {
	nb.notification.Channels = channels
	return nb
}

// WithPriority sets the notification priority
func (nb *NotificationBuilder) WithPriority(priority NotificationPriority) *NotificationBuilder {
	nb.notification.Priority = priority
	return nb
}

// WithSubject sets the notification subject
func (nb *NotificationBuilder) WithSubject(subject string) *NotificationBuilder {
	nb.notification.Subject = subject
	return nb
}

// WithTemplate sets the template ID and data
func (nb *NotificationBuilder) WithTemplate(templateID string, data map[string]interface{}) *NotificationBuilder {
	nb.notification.TemplateID = templateID
	nb.notification.TemplateData = data
	return nb
}

// WithEventContext sets event-related context
func (nb *NotificationBuilder) WithEventContext(eventID uuid.UUID) *NotificationBuilder {
	nb.notification.EventID = &eventID
	return nb
}

// WithBookingContext sets booking-related context
func (nb *NotificationBuilder) WithBookingContext(bookingID uuid.UUID) *NotificationBuilder {
	nb.notification.BookingID = &bookingID
	return nb
}

// WithWaitlistContext sets waitlist-related context
func (nb *NotificationBuilder) WithWaitlistContext(waitlistEntryID uuid.UUID) *NotificationBuilder {
	nb.notification.WaitlistEntryID = &waitlistEntryID
	return nb
}

// WithScheduling sets scheduling information
func (nb *NotificationBuilder) WithScheduling(scheduledFor *time.Time, expiresAt *time.Time) *NotificationBuilder {
	nb.notification.ScheduledFor = scheduledFor
	nb.notification.ExpiresAt = expiresAt
	return nb
}

// WithMaxRetries sets the maximum retry attempts
func (nb *NotificationBuilder) WithMaxRetries(maxRetries int) *NotificationBuilder {
	nb.notification.MaxRetries = maxRetries
	return nb
}

// Build returns the built notification
func (nb *NotificationBuilder) Build() *UnifiedNotification {
	return nb.notification
}

// GetDefaultPriority returns the default priority for a notification type
func GetDefaultPriority(notType NotificationType) NotificationPriority {
	switch notType {
	case NotificationTypeWaitlistSpotAvailable,
		NotificationTypeBookingCancelled,
		NotificationTypeEventCancelled,
		NotificationTypePaymentFailed:
		return NotificationPriorityHigh

	case NotificationTypeWaitlistExpired,
		NotificationTypeBookingConfirmed,
		NotificationTypePaymentSuccess,
		NotificationTypeRefundProcessed:
		return NotificationPriorityMedium

	case NotificationTypeWaitlistPositionUpdate,
		NotificationTypeEventReminder,
		NotificationTypeBookingReminder:
		return NotificationPriorityLow

	case NotificationTypePasswordReset,
		NotificationTypeAccountVerify:
		return NotificationPriorityCritical

	default:
		return NotificationPriorityMedium
	}
}

// GetDefaultChannels returns the default channels for a notification type
func GetDefaultChannels(notType NotificationType) []NotificationChannel {
	switch notType {
	case NotificationTypeWaitlistSpotAvailable,
		NotificationTypeBookingCancelled,
		NotificationTypeEventCancelled:
		return []NotificationChannel{NotificationChannelEmail, NotificationChannelSMS}

	case NotificationTypePasswordReset,
		NotificationTypeAccountVerify,
		NotificationTypeBookingConfirmed,
		NotificationTypePaymentSuccess:
		return []NotificationChannel{NotificationChannelEmail}

	case NotificationTypeEventReminder,
		NotificationTypeBookingReminder:
		return []NotificationChannel{NotificationChannelEmail, NotificationChannelInApp}

	default:
		return []NotificationChannel{NotificationChannelEmail}
	}
}

// GetPartitionKey returns the partition key for Kafka (user_id for load balancing)
func (un *UnifiedNotification) GetPartitionKey() string {
	return un.RecipientID.String()
}

// ToJSON serializes the notification to JSON
func (un *UnifiedNotification) ToJSON() ([]byte, error) {
	return json.Marshal(un)
}

// IsExpired checks if the notification has expired
func (un *UnifiedNotification) IsExpired() bool {
	return un.ExpiresAt != nil && time.Now().After(*un.ExpiresAt)
}

// ShouldRetry determines if the notification should be retried
func (un *UnifiedNotification) ShouldRetry() bool {
	return un.RetryCount < un.MaxRetries &&
		un.Status == NotificationStatusFailed &&
		!un.IsExpired()
}

// MarkDelivered updates the notification status to delivered
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

// MarkFailed updates the notification status to failed
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

// IncrementRetry increments the retry count and updates status
func (un *UnifiedNotification) IncrementRetry() {
	un.RetryCount++
	un.UpdatedAt = time.Now()
	if un.ShouldRetry() {
		un.Status = NotificationStatusRetrying
	} else {
		un.Status = NotificationStatusExpired
	}
}
