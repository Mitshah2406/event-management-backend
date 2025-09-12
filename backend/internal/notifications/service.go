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

// NotificationService provides a unified interface for the notification system
type NotificationService interface {
	// Core notification methods
	SendNotification(ctx context.Context, notification *UnifiedNotification) error
	SendBatchNotifications(ctx context.Context, notifications []*UnifiedNotification) error

	// Convenience methods for different notification types
	SendWaitlistNotification(ctx context.Context, userID uuid.UUID, email, name string,
		eventID, waitlistEntryID uuid.UUID, notificationType NotificationType,
		templateData map[string]interface{}) error

	SendBookingNotification(ctx context.Context, userID uuid.UUID, email, name string,
		bookingID, eventID uuid.UUID, notificationType NotificationType,
		templateData map[string]interface{}) error

	SendEventNotification(ctx context.Context, userID uuid.UUID, email, name string,
		eventID uuid.UUID, notificationType NotificationType,
		templateData map[string]interface{}) error

	// Service management
	Start(ctx context.Context) error
	Stop() error
	HealthCheck(ctx context.Context) error
	GetMetrics() (*ServiceMetrics, error)
}

// ServiceConfig holds configuration for the notification service
type ServiceConfig struct {
	// Environment
	Environment string // "development", "production", etc.

	// Kafka configuration
	KafkaBrokers       []string
	NotificationTopic  string
	ConsumerGroupID    string
	NumConsumerWorkers int

	// SMTP configuration
	SMTPHost      string
	SMTPPort      int
	SMTPUsername  string
	SMTPPassword  string
	SMTPFromEmail string
	SMTPFromName  string

	// Feature flags
	EnableEmailChannel bool
	EnableSMSChannel   bool
	EnablePushChannel  bool
}

// NewServiceConfigFromEnv creates service config from environment variables
func NewServiceConfigFromEnv() *ServiceConfig {
	config := &ServiceConfig{
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
		EnableEmailChannel: getEnvBool("ENABLE_EMAIL_CHANNEL", true),
		EnableSMSChannel:   getEnvBool("ENABLE_SMS_CHANNEL", false), // Disabled by default until SMS service is implemented
		EnablePushChannel:  getEnvBool("ENABLE_PUSH_CHANNEL", false),
	}

	return config
}

// UnifiedNotificationService is the main implementation of NotificationService
type UnifiedNotificationService struct {
	config    *ServiceConfig
	producer  NotificationProducer
	consumer  NotificationConsumer
	publisher *NotificationPublisher

	// Services
	emailService EmailService
	smsService   SMSService

	// State
	isRunning bool
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewUnifiedNotificationService creates a new unified notification service
func NewUnifiedNotificationService(config *ServiceConfig) (NotificationService, error) {
	if config == nil {
		config = NewServiceConfigFromEnv()
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

	consumer, err := NewKafkaNotificationConsumer(consumerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification consumer: %w", err)
	}

	// Create publisher
	publisher := NewNotificationPublisher(producer)

	// Create email service - only SMTP is supported
	var emailService EmailService
	if config.SMTPHost == "" || config.SMTPUsername == "" {
		return nil, fmt.Errorf("SMTP configuration is required: missing SMTP_HOST or SMTP_USERNAME")
	}

	smtpConfig := &SMTPConfig{
		Host:      config.SMTPHost,
		Port:      config.SMTPPort,
		Username:  config.SMTPUsername,
		Password:  config.SMTPPassword,
		FromEmail: config.SMTPFromEmail,
		FromName:  config.SMTPFromName,
		UseTLS:    true,
	}
	emailService = NewSMTPEmailService(smtpConfig)
	log.Printf("📧 SMTP email service initialized (Host: %s, Port: %d)", config.SMTPHost, config.SMTPPort)

	// Create SMS service (mock for now)
	var smsService SMSService = NewMockSMSServiceImpl()

	ctx, cancel := context.WithCancel(context.Background())

	return &UnifiedNotificationService{
		config:       config,
		producer:     producer,
		consumer:     consumer,
		publisher:    publisher,
		emailService: emailService,
		smsService:   smsService,
		isRunning:    false,
		ctx:          ctx,
		cancel:       cancel,
	}, nil
}

// Start starts the notification service
func (uns *UnifiedNotificationService) Start(ctx context.Context) error {
	uns.mu.Lock()
	defer uns.mu.Unlock()

	if uns.isRunning {
		return fmt.Errorf("notification service is already running")
	}

	log.Printf("🚀 Starting Unified Notification Service...")

	// Register channel handlers
	if uns.config.EnableEmailChannel {
		emailHandler := NewEmailChannelHandler(uns.emailService)
		if err := uns.consumer.RegisterHandler(NotificationChannelEmail, emailHandler); err != nil {
			return fmt.Errorf("failed to register email handler: %w", err)
		}
	}

	if uns.config.EnableSMSChannel {
		smsHandler := NewSMSChannelHandler(uns.smsService)
		if err := uns.consumer.RegisterHandler(NotificationChannelSMS, smsHandler); err != nil {
			return fmt.Errorf("failed to register SMS handler: %w", err)
		}
	}

	// Start consumers
	err := uns.consumer.StartConsumers(uns.ctx, uns.config.NumConsumerWorkers)
	if err != nil {
		return fmt.Errorf("failed to start consumers: %w", err)
	}

	uns.isRunning = true
	log.Printf("✅ Unified Notification Service started successfully")

	return nil
}

// Stop stops the notification service
func (uns *UnifiedNotificationService) Stop() error {
	uns.mu.Lock()
	defer uns.mu.Unlock()

	if !uns.isRunning {
		return fmt.Errorf("notification service is not running")
	}

	log.Printf("🛑 Stopping Unified Notification Service...")

	// Cancel context
	uns.cancel()

	// Stop consumer
	if err := uns.consumer.Stop(); err != nil {
		log.Printf("Error stopping consumer: %v", err)
	}

	// Close producer
	if err := uns.producer.Close(); err != nil {
		log.Printf("Error closing producer: %v", err)
	}

	uns.isRunning = false
	log.Printf("✅ Unified Notification Service stopped")

	return nil
}

// SendNotification sends a single notification
func (uns *UnifiedNotificationService) SendNotification(ctx context.Context, notification *UnifiedNotification) error {
	return uns.producer.PublishNotification(ctx, notification)
}

// SendBatchNotifications sends multiple notifications
func (uns *UnifiedNotificationService) SendBatchNotifications(ctx context.Context, notifications []*UnifiedNotification) error {
	return uns.producer.PublishBatchNotifications(ctx, notifications)
}

// SendWaitlistNotification sends a waitlist-specific notification
func (uns *UnifiedNotificationService) SendWaitlistNotification(ctx context.Context, userID uuid.UUID, email, name string,
	eventID, waitlistEntryID uuid.UUID, notificationType NotificationType,
	templateData map[string]interface{}) error {

	return uns.publisher.PublishWaitlistNotification(ctx, userID, email, name, eventID, waitlistEntryID, notificationType, templateData)
}

// SendBookingNotification sends a booking-specific notification
func (uns *UnifiedNotificationService) SendBookingNotification(ctx context.Context, userID uuid.UUID, email, name string,
	bookingID, eventID uuid.UUID, notificationType NotificationType,
	templateData map[string]interface{}) error {

	return uns.publisher.PublishBookingNotification(ctx, userID, email, name, bookingID, eventID, notificationType, templateData)
}

// SendEventNotification sends an event-specific notification
func (uns *UnifiedNotificationService) SendEventNotification(ctx context.Context, userID uuid.UUID, email, name string,
	eventID uuid.UUID, notificationType NotificationType,
	templateData map[string]interface{}) error {

	return uns.publisher.PublishEventNotification(ctx, userID, email, name, eventID, notificationType, templateData)
}

// HealthCheck performs a health check on the service
func (uns *UnifiedNotificationService) HealthCheck(ctx context.Context) error {
	uns.mu.RLock()
	isRunning := uns.isRunning
	uns.mu.RUnlock()

	if !isRunning {
		return fmt.Errorf("notification service is not running")
	}

	// Check producer health
	if err := uns.producer.HealthCheck(ctx); err != nil {
		return fmt.Errorf("producer health check failed: %w", err)
	}

	// Check consumer health
	if err := uns.consumer.HealthCheck(ctx); err != nil {
		return fmt.Errorf("consumer health check failed: %w", err)
	}

	return nil
}

// GetMetrics returns service metrics
func (uns *UnifiedNotificationService) GetMetrics() (*ServiceMetrics, error) {
	// For now, return basic metrics - can be enhanced later with proper metrics collection
	consumerMetrics := uns.consumer.(*KafkaNotificationConsumer).GetMetrics()

	return &ServiceMetrics{
		ConsumerMetrics: *consumerMetrics,
		IsRunning:       uns.isRunning,
	}, nil
}

// ServiceMetrics contains metrics for the entire notification service
type ServiceMetrics struct {
	ConsumerMetrics ConsumerMetrics `json:"consumer_metrics"`
	IsRunning       bool            `json:"is_running"`
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

func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return boolValue
}

// Global service instance (singleton pattern for easy access)
var (
	globalService NotificationService
	globalOnce    sync.Once
)

// GetGlobalNotificationService returns the global notification service instance
func GetGlobalNotificationService() NotificationService {
	globalOnce.Do(func() {
		config := NewServiceConfigFromEnv()
		service, err := NewUnifiedNotificationService(config)
		if err != nil {
			log.Fatalf("Failed to initialize global notification service: %v", err)
		}
		globalService = service
	})
	return globalService
}

// InitializeGlobalNotificationService initializes and starts the global notification service
func InitializeGlobalNotificationService(ctx context.Context) error {
	service := GetGlobalNotificationService()
	return service.Start(ctx)
}

// StopGlobalNotificationService stops the global notification service
func StopGlobalNotificationService() error {
	if globalService != nil {
		return globalService.Stop()
	}
	return nil
}
