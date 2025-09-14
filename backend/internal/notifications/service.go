package notifications

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/google/uuid"
)

type NotificationService interface {
	SendNotification(ctx context.Context, notification *EmailNotification) error
	SendBatchNotifications(ctx context.Context, notifications []*EmailNotification) error

	SendWaitlistNotification(ctx context.Context, userID uuid.UUID, email, name string,
		eventID, waitlistEntryID uuid.UUID, notificationType NotificationType,
		templateData map[string]interface{}) error

	SendBookingNotification(ctx context.Context, userID uuid.UUID, email, name string,
		bookingID, eventID uuid.UUID, notificationType NotificationType,
		templateData map[string]interface{}) error

	Start(ctx context.Context) error
	Stop() error
	HealthCheck(ctx context.Context) error
}

type ServiceConfig struct {
	Environment        string
	KafkaBrokers       []string
	NotificationTopic  string
	ConsumerGroupID    string
	NumConsumerWorkers int
	SMTPHost           string
	SMTPPort           int
	SMTPUsername       string
	SMTPPassword       string
	SMTPFromEmail      string
	SMTPFromName       string
}

func NewServiceConfigFromEnv() *ServiceConfig {
	return &ServiceConfig{
		Environment:        getEnvString("GIN_MODE", "development"),
		KafkaBrokers:       []string{getEnvString("KAFKA_BROKERS", "localhost:9092")},
		NotificationTopic:  getEnvString("NOTIFICATION_TOPIC", "notifications"),
		ConsumerGroupID:    getEnvString("CONSUMER_GROUP_ID", "evently-notification-workers"),
		NumConsumerWorkers: getEnvInt("NUM_CONSUMER_WORKERS", 3),
		SMTPHost:           getEnvString("SMTP_HOST", ""),
		SMTPPort:           getEnvInt("SMTP_PORT", 587),
		SMTPUsername:       getEnvString("SMTP_USERNAME", ""),
		SMTPPassword:       getEnvString("SMTP_PASSWORD", ""),
		SMTPFromEmail:      getEnvString("FROM_EMAIL", ""),
		SMTPFromName:       getEnvString("SMTP_FROM_NAME", "Evently"),
	}
}

type EmailNotificationService struct {
	config       *ServiceConfig
	producer     NotificationProducer
	consumer     NotificationConsumer
	publisher    *NotificationPublisher
	emailService EmailService

	// State
	isRunning bool
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewEmailNotificationService(config *ServiceConfig) (NotificationService, error) {
	if config == nil {
		config = NewServiceConfigFromEnv()
	}

	// Validate SMTP configuration
	if config.SMTPHost == "" || config.SMTPUsername == "" {
		return nil, fmt.Errorf("SMTP configuration is required: missing SMTP_HOST or SMTP_USERNAME")
	}

	// Create SMTP email service
	smtpConfig := &SMTPConfig{
		Host:      config.SMTPHost,
		Port:      config.SMTPPort,
		Username:  config.SMTPUsername,
		Password:  config.SMTPPassword,
		FromEmail: config.SMTPFromEmail,
		FromName:  config.SMTPFromName,
		UseTLS:    true,
	}
	emailService := NewSMTPEmailService(smtpConfig)
	if emailService == nil {
		return nil, fmt.Errorf("failed to create SMTP email service")
	}

	// Create producer
	producerConfig := DefaultKafkaProducerConfig()
	producerConfig.Brokers = config.KafkaBrokers
	producerConfig.NotificationTopic = config.NotificationTopic

	producer, err := NewKafkaNotificationProducer(producerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification producer: %w", err)
	}

	// Create consumer
	consumerConfig := DefaultConsumerConfig()
	consumerConfig.Brokers = config.KafkaBrokers
	consumerConfig.Topics = []string{config.NotificationTopic}
	consumerConfig.GroupID = config.ConsumerGroupID

	consumer, err := NewKafkaNotificationConsumer(consumerConfig, emailService)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification consumer: %w", err)
	}

	publisher := NewNotificationPublisher(producer)

	ctx, cancel := context.WithCancel(context.Background())

	log.Printf("📧 Email notification service initialized (Host: %s, Port: %d)", config.SMTPHost, config.SMTPPort)

	return &EmailNotificationService{
		config:       config,
		producer:     producer,
		consumer:     consumer,
		publisher:    publisher,
		emailService: emailService,
		isRunning:    false,
		ctx:          ctx,
		cancel:       cancel,
	}, nil
}

func (ens *EmailNotificationService) Start(ctx context.Context) error {
	ens.mu.Lock()
	defer ens.mu.Unlock()

	if ens.isRunning {
		return fmt.Errorf("notification service is already running")
	}

	log.Printf("🚀 Starting Email Notification Service...")

	err := ens.consumer.StartConsumers(ens.ctx, ens.config.NumConsumerWorkers)
	if err != nil {
		return fmt.Errorf("failed to start consumers: %w", err)
	}

	ens.isRunning = true
	log.Printf("✅ Email Notification Service started successfully")

	return nil
}

func (ens *EmailNotificationService) Stop() error {
	ens.mu.Lock()
	defer ens.mu.Unlock()

	if !ens.isRunning {
		return fmt.Errorf("notification service is not running")
	}

	log.Printf("🛑 Stopping Email Notification Service...")

	ens.cancel()

	if err := ens.consumer.Stop(); err != nil {
		log.Printf("Error stopping consumer: %v", err)
	}

	if err := ens.producer.Close(); err != nil {
		log.Printf("Error closing producer: %v", err)
	}

	ens.isRunning = false
	log.Printf("✅ Email Notification Service stopped")

	return nil
}

func (ens *EmailNotificationService) SendNotification(ctx context.Context, notification *EmailNotification) error {
	return ens.producer.PublishNotification(ctx, notification)
}

func (ens *EmailNotificationService) SendBatchNotifications(ctx context.Context, notifications []*EmailNotification) error {
	return ens.producer.PublishBatchNotifications(ctx, notifications)
}

func (ens *EmailNotificationService) SendWaitlistNotification(ctx context.Context, userID uuid.UUID, email, name string,
	eventID, waitlistEntryID uuid.UUID, notificationType NotificationType,
	templateData map[string]interface{}) error {

	return ens.publisher.PublishWaitlistNotification(ctx, userID, email, name, eventID, waitlistEntryID, notificationType, templateData)
}

func (ens *EmailNotificationService) SendBookingNotification(ctx context.Context, userID uuid.UUID, email, name string,
	bookingID, eventID uuid.UUID, notificationType NotificationType,
	templateData map[string]interface{}) error {

	return ens.publisher.PublishBookingNotification(ctx, userID, email, name, bookingID, eventID, notificationType, templateData)
}

func (ens *EmailNotificationService) HealthCheck(ctx context.Context) error {
	ens.mu.RLock()
	isRunning := ens.isRunning
	ens.mu.RUnlock()

	if !isRunning {
		return fmt.Errorf("notification service is not running")
	}

	if err := ens.producer.HealthCheck(ctx); err != nil {
		return fmt.Errorf("producer health check failed: %w", err)
	}

	if err := ens.consumer.HealthCheck(ctx); err != nil {
		return fmt.Errorf("consumer health check failed: %w", err)
	}

	return nil
}

// Helper functions for environment variables
func getEnvString(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}
