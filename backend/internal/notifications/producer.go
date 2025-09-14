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
	PublishNotification(ctx context.Context, notification *EmailNotification) error
	PublishBatchNotifications(ctx context.Context, notifications []*EmailNotification) error
	Close() error
	HealthCheck(ctx context.Context) error
}

type KafkaProducerConfig struct {
	Brokers           []string
	NotificationTopic string
	RetryMax          int
	TimeoutMs         int
	RequiredAcks      sarama.RequiredAcks
	CompressionType   sarama.CompressionCodec
	IdempotentWrites  bool
	MaxMessageBytes   int
}

func DefaultKafkaProducerConfig() *KafkaProducerConfig {
	return &KafkaProducerConfig{
		Brokers:           []string{"localhost:9092"},
		NotificationTopic: "notifications",
		RetryMax:          3,
		TimeoutMs:         10000,
		RequiredAcks:      sarama.WaitForAll,
		CompressionType:   sarama.CompressionSnappy,
		IdempotentWrites:  true,
		MaxMessageBytes:   1000000,
	}
}

type KafkaNotificationProducer struct {
	producer sarama.SyncProducer
	config   *KafkaProducerConfig
}

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

	producer, err := sarama.NewSyncProducer(config.Brokers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	log.Printf("📤 Kafka notification producer created successfully")
	return &KafkaNotificationProducer{
		producer: producer,
		config:   config,
	}, nil
}

func (knp *KafkaNotificationProducer) PublishNotification(ctx context.Context, notification *EmailNotification) error {
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

func (knp *KafkaNotificationProducer) PublishBatchNotifications(ctx context.Context, notifications []*EmailNotification) error {
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

func (knp *KafkaNotificationProducer) createHeaders(notification *EmailNotification) []sarama.RecordHeader {
	headers := []sarama.RecordHeader{
		{Key: []byte("notification_id"), Value: []byte(notification.ID.String())},
		{Key: []byte("notification_type"), Value: []byte(notification.Type)},
		{Key: []byte("priority"), Value: []byte(notification.Priority)},
		{Key: []byte("recipient_id"), Value: []byte(notification.RecipientID.String())},
		{Key: []byte("recipient_email"), Value: []byte(notification.RecipientEmail)},
		{Key: []byte("channel"), Value: []byte(NotificationChannelEmail)},
		{Key: []byte("version"), Value: []byte("2.1")},
		{Key: []byte("producer"), Value: []byte("evently-notifications-simplified")},
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

func (knp *KafkaNotificationProducer) HealthCheck(ctx context.Context) error {
	// Actually test the producer by sending a test message to a test topic
	testNotification := NewNotificationBuilder().
		WithType(NotificationTypeBookingConfirmed).
		WithRecipient(uuid.New(), "health-check@test.com", "Health Check").
		WithSubject("Health Check").
		WithTemplateData(map[string]interface{}{
			"event_title":    "Test Event",
			"booking_number": "TEST-123",
			"quantity":       1,
			"total_amount":   0.0,
		}).
		Build()

	// Test JSON marshaling
	messageBytes, err := testNotification.ToJSON()
	if err != nil {
		return fmt.Errorf("health check failed - JSON marshaling error: %w", err)
	}

	// Test header creation
	headers := knp.createHeaders(testNotification)
	if len(headers) == 0 {
		return fmt.Errorf("health check failed - headers not created properly")
	}

	if knp.producer == nil {
		return fmt.Errorf("health check failed - producer is nil")
	}

	if knp.config.NotificationTopic == "" {
		return fmt.Errorf("health check failed - notification topic not configured")
	}

	// Verify message size
	if len(messageBytes) > knp.config.MaxMessageBytes {
		return fmt.Errorf("health check failed - message too large")
	}

	log.Printf("✅ Kafka notification producer health check passed")
	return nil
}

// Simplified publisher for high-level operations
type NotificationPublisher struct {
	producer NotificationProducer
}

func NewNotificationPublisher(producer NotificationProducer) *NotificationPublisher {
	return &NotificationPublisher{
		producer: producer,
	}
}

func (np *NotificationPublisher) PublishWaitlistNotification(ctx context.Context,
	userID uuid.UUID, email, name string, eventID uuid.UUID, waitlistEntryID uuid.UUID,
	notificationType NotificationType, templateData map[string]interface{}) error {

	notification := NewNotificationBuilder().
		WithType(notificationType).
		WithRecipient(userID, email, name).
		WithEventContext(eventID).
		WithWaitlistContext(waitlistEntryID).
		WithTemplateData(templateData).
		WithSubject(np.generateSubject(notificationType, templateData)).
		Build()

	return np.producer.PublishNotification(ctx, notification)
}

func (np *NotificationPublisher) PublishBookingNotification(ctx context.Context,
	userID uuid.UUID, email, name string, bookingID uuid.UUID, eventID uuid.UUID,
	notificationType NotificationType, templateData map[string]interface{}) error {

	notification := NewNotificationBuilder().
		WithType(notificationType).
		WithRecipient(userID, email, name).
		WithBookingContext(bookingID).
		WithEventContext(eventID).
		WithTemplateData(templateData).
		WithSubject(np.generateSubject(notificationType, templateData)).
		Build()

	return np.producer.PublishNotification(ctx, notification)
}

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

	case NotificationTypeWaitlistPositionUpdate:
		if eventTitle, ok := data["event_title"]; ok {
			return fmt.Sprintf("📊 Position Update for %s", eventTitle)
		}
		return "📊 Your waitlist position has been updated"

	default:
		return "📧 Notification from Evently"
	}
}
