package notifications

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
)

type NotificationProducer interface {
	PublishNotification(ctx context.Context, notification *UnifiedNotification) error
	PublishBatchNotifications(ctx context.Context, notifications []*UnifiedNotification) error
	Close() error
	HealthCheck(ctx context.Context) error
}

// KafkaProducerConfig contains configuration for the Kafka notification producer
type KafkaProducerConfig struct {
	Brokers             []string
	NotificationTopic   string
	RetryMax            int
	TimeoutMs           int
	RequiredAcks        sarama.RequiredAcks
	CompressionType     sarama.CompressionCodec
	IdempotentWrites    bool
	MaxMessageBytes     int
	EnableTopicCreation bool
}

// DefaultKafkaProducerConfig returns a default producer configuration
func DefaultKafkaProducerConfig() *KafkaProducerConfig {
	return &KafkaProducerConfig{
		Brokers:             []string{"localhost:9092"},
		NotificationTopic:   "notifications",
		RetryMax:            3,
		TimeoutMs:           10000,             // 10 seconds
		RequiredAcks:        sarama.WaitForAll, // Wait for all in-sync replicas
		CompressionType:     sarama.CompressionSnappy,
		IdempotentWrites:    true,
		MaxMessageBytes:     1000000, // 1MB
		EnableTopicCreation: true,
	}
}

// KafkaNotificationProducer handles publishing notifications to Kafka
type KafkaNotificationProducer struct {
	producer sarama.SyncProducer
	config   *KafkaProducerConfig
}

// NewKafkaNotificationProducer creates a new Kafka notification producer
func NewKafkaNotificationProducer(config *KafkaProducerConfig) (NotificationProducer, error) {
	saramaConfig := sarama.NewConfig()

	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Return.Errors = true
	saramaConfig.Producer.RequiredAcks = config.RequiredAcks
	saramaConfig.Producer.Compression = config.CompressionType
	saramaConfig.Producer.Retry.Max = config.RetryMax
	saramaConfig.Producer.Timeout = time.Duration(config.TimeoutMs) * time.Millisecond
	saramaConfig.Producer.Idempotent = config.IdempotentWrites
	saramaConfig.Producer.MaxMessageBytes = config.MaxMessageBytes

	if config.IdempotentWrites {
		saramaConfig.Net.MaxOpenRequests = 1
	}

	saramaConfig.Producer.Partitioner = sarama.NewHashPartitioner

	// Create the producer
	producer, err := sarama.NewSyncProducer(config.Brokers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	kafkaProducer := &KafkaNotificationProducer{
		producer: producer,
		config:   config,
	}

	log.Printf("📤 Kafka notification producer created successfully")
	return kafkaProducer, nil
}

func (knp *KafkaNotificationProducer) PublishNotification(ctx context.Context, notification *UnifiedNotification) error {
	notification.Status = NotificationStatusQueued
	notification.UpdatedAt = time.Now()

	messageBytes, err := notification.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}
	message := &sarama.ProducerMessage{
		Topic:     knp.config.NotificationTopic,
		Key:       sarama.StringEncoder(notification.GetPartitionKey()),
		Value:     sarama.ByteEncoder(messageBytes),
		Headers:   knp.createHeaders(notification),
		Timestamp: notification.CreatedAt,
	}

	// Send message
	partition, offset, err := knp.producer.SendMessage(message)
	if err != nil {
		notification.Status = NotificationStatusFailed
		errorStr := err.Error()
		notification.LastError = &errorStr
		return fmt.Errorf("failed to send notification to Kafka: %w", err)
	}

	log.Printf("📤 Notification published to Kafka - Topic: %s, Partition: %d, Offset: %d, Type: %s, Recipient: %s",
		knp.config.NotificationTopic, partition, offset, notification.Type, notification.RecipientEmail)

	return nil
}

func (knp *KafkaNotificationProducer) PublishBatchNotifications(ctx context.Context, notifications []*UnifiedNotification) error {
	if len(notifications) == 0 {
		return nil
	}

	messages := make([]*sarama.ProducerMessage, 0, len(notifications))

	for _, notification := range notifications {
		notification.Status = NotificationStatusQueued
		notification.UpdatedAt = time.Now()

		messageBytes, err := notification.ToJSON()
		if err != nil {
			log.Printf("Failed to marshal notification for user %s: %v", notification.RecipientEmail, err)
			continue
		}

		message := &sarama.ProducerMessage{
			Topic:     knp.config.NotificationTopic,
			Key:       sarama.StringEncoder(notification.GetPartitionKey()),
			Value:     sarama.ByteEncoder(messageBytes),
			Headers:   knp.createHeaders(notification),
			Timestamp: notification.CreatedAt,
		}

		messages = append(messages, message)
	}

	err := knp.producer.SendMessages(messages)
	if err != nil {
		for _, notification := range notifications {
			notification.Status = NotificationStatusFailed
			errorStr := err.Error()
			notification.LastError = &errorStr
		}
		return fmt.Errorf("failed to send batch notifications to Kafka: %w", err)
	}

	log.Printf("📤 Batch of %d notifications published to Kafka topic: %s", len(messages), knp.config.NotificationTopic)
	return nil
}

// createHeaders creates Kafka headers for notifications
func (knp *KafkaNotificationProducer) createHeaders(notification *UnifiedNotification) []sarama.RecordHeader {
	headers := []sarama.RecordHeader{
		{Key: []byte("notification_id"), Value: []byte(notification.ID.String())},
		{Key: []byte("notification_type"), Value: []byte(notification.Type)},
		{Key: []byte("priority"), Value: []byte(notification.Priority)},
		{Key: []byte("recipient_id"), Value: []byte(notification.RecipientID.String())},
		{Key: []byte("recipient_email"), Value: []byte(notification.RecipientEmail)},
		{Key: []byte("channels"), Value: []byte(knp.formatChannels(notification.Channels))},
		{Key: []byte("version"), Value: []byte("2.0")},
		{Key: []byte("producer"), Value: []byte("evently-notifications")},
		{Key: []byte("created_at"), Value: []byte(notification.CreatedAt.Format(time.RFC3339))},
	}

	// Add optional context headers
	if notification.EventID != nil {
		headers = append(headers, sarama.RecordHeader{
			Key:   []byte("event_id"),
			Value: []byte(notification.EventID.String()),
		})
	}

	if notification.BookingID != nil {
		headers = append(headers, sarama.RecordHeader{
			Key:   []byte("booking_id"),
			Value: []byte(notification.BookingID.String()),
		})
	}

	if notification.WaitlistEntryID != nil {
		headers = append(headers, sarama.RecordHeader{
			Key:   []byte("waitlist_entry_id"),
			Value: []byte(notification.WaitlistEntryID.String()),
		})
	}

	if notification.ExpiresAt != nil {
		headers = append(headers, sarama.RecordHeader{
			Key:   []byte("expires_at"),
			Value: []byte(notification.ExpiresAt.Format(time.RFC3339)),
		})
	}

	return headers
}

// formatChannels formats notification channels for headers
func (knp *KafkaNotificationProducer) formatChannels(channels []NotificationChannel) string {
	if len(channels) == 0 {
		return string(NotificationChannelEmail) // default
	}

	result := string(channels[0])
	for i := 1; i < len(channels); i++ {
		result += "," + string(channels[i])
	}
	return result
}

func (knp *KafkaNotificationProducer) Close() error {
	if knp.producer != nil {
		err := knp.producer.Close()
		if err != nil {
			return fmt.Errorf("failed to close Kafka producer: %w", err)
		}
		log.Printf("📤 Kafka notification producer closed")
	}
	return nil
}

// HealthCheck performs a health check on the Kafka producerz
func (knp *KafkaNotificationProducer) HealthCheck(ctx context.Context) error {
	// Create a test notification
	testNotification := NewNotificationBuilder().
		WithType(NotificationTypeWelcome).
		WithRecipient(uuid.New(), "health-check@test.com", "Health Check").
		WithChannels(NotificationChannelEmail).
		WithSubject("Health Check").
		Build()

	testNotification.ID = uuid.MustParse("00000000-0000-0000-0000-000000000000") // Use zero UUID for health checks

	// Serialize to test JSON marshaling
	messageBytes, err := testNotification.ToJSON()
	if err != nil {
		return fmt.Errorf("health check failed - JSON marshaling error: %w", err)
	}

	// Create test message (don't actually send to avoid noise)
	message := &sarama.ProducerMessage{
		Topic:   knp.config.NotificationTopic,
		Key:     sarama.StringEncoder("health-check"),
		Value:   sarama.ByteEncoder(messageBytes),
		Headers: knp.createHeaders(testNotification),
	}

	// Validate message is properly formed
	if message.Topic == "" {
		return fmt.Errorf("health check failed - invalid topic configuration")
	}

	if len(message.Headers) == 0 {
		return fmt.Errorf("health check failed - headers not created properly")
	}

	// Validate producer is not nil and configuration is valid
	if knp.producer == nil {
		return fmt.Errorf("health check failed - producer is nil")
	}

	if knp.config.NotificationTopic == "" {
		return fmt.Errorf("health check failed - notification topic not configured")
	}

	// Simple connectivity test - the producer will fail if Kafka is unreachable
	// when we try to send the first actual message

	log.Printf("✅ Kafka notification producer health check passed")
	return nil
}

// NotificationPublisher provides a high-level interface for publishing different types of notifications
type NotificationPublisher struct {
	producer NotificationProducer
}

// NewNotificationPublisher creates a new notification publisher
func NewNotificationPublisher(producer NotificationProducer) *NotificationPublisher {
	return &NotificationPublisher{
		producer: producer,
	}
}

// PublishWaitlistNotification publishes a waitlist-specific notification
func (np *NotificationPublisher) PublishWaitlistNotification(ctx context.Context,
	userID uuid.UUID, email, name string, eventID uuid.UUID, waitlistEntryID uuid.UUID,
	notificationType NotificationType, templateData map[string]interface{}) error {

	notification := NewNotificationBuilder().
		WithType(notificationType).
		WithRecipient(userID, email, name).
		WithChannels(GetDefaultChannels(notificationType)...).
		WithEventContext(eventID).
		WithWaitlistContext(waitlistEntryID).
		WithTemplate(string(notificationType), templateData).
		Build()

	// Generate subject based on type
	notification.Subject = np.generateSubject(notificationType, templateData)

	return np.producer.PublishNotification(ctx, notification)
}

// PublishBookingNotification publishes a booking-specific notification
func (np *NotificationPublisher) PublishBookingNotification(ctx context.Context,
	userID uuid.UUID, email, name string, bookingID uuid.UUID, eventID uuid.UUID,
	notificationType NotificationType, templateData map[string]interface{}) error {

	notification := NewNotificationBuilder().
		WithType(notificationType).
		WithRecipient(userID, email, name).
		WithChannels(GetDefaultChannels(notificationType)...).
		WithBookingContext(bookingID).
		WithEventContext(eventID).
		WithTemplate(string(notificationType), templateData).
		Build()

	// Generate subject based on type
	notification.Subject = np.generateSubject(notificationType, templateData)

	return np.producer.PublishNotification(ctx, notification)
}

// PublishEventNotification publishes an event-specific notification
func (np *NotificationPublisher) PublishEventNotification(ctx context.Context,
	userID uuid.UUID, email, name string, eventID uuid.UUID,
	notificationType NotificationType, templateData map[string]interface{}) error {

	notification := NewNotificationBuilder().
		WithType(notificationType).
		WithRecipient(userID, email, name).
		WithChannels(GetDefaultChannels(notificationType)...).
		WithEventContext(eventID).
		WithTemplate(string(notificationType), templateData).
		Build()

	// Generate subject based on type
	notification.Subject = np.generateSubject(notificationType, templateData)

	return np.producer.PublishNotification(ctx, notification)
}

// generateSubject generates appropriate subjects for different notification types
func (np *NotificationPublisher) generateSubject(notificationType NotificationType, data map[string]interface{}) string {
	switch notificationType {
	case NotificationTypeWaitlistSpotAvailable:
		if eventTitle, ok := data["event_title"]; ok {
			return fmt.Sprintf("🎉 Great News! A spot is available for %s", eventTitle)
		}
		return "🎉 A spot is now available!"

	case NotificationTypeBookingConfirmed:
		if eventTitle, ok := data["event_title"]; ok {
			return fmt.Sprintf("✅ Booking Confirmed for %s", eventTitle)
		}
		return "✅ Your booking is confirmed!"

	default:
		return "📧 Notification from Evently"
	}
}
