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
	SendNotification(ctx context.Context, notification *UnifiedNotification) error
	SendBatchNotifications(ctx context.Context, notifications []*UnifiedNotification) error

	SendWaitlistNotification(ctx context.Context, userID uuid.UUID, email, name string,
		eventID, waitlistEntryID uuid.UUID, notificationType NotificationType,
		templateData map[string]interface{}) error

	SendBookingNotification(ctx context.Context, userID uuid.UUID, email, name string,
		bookingID, eventID uuid.UUID, notificationType NotificationType,
		templateData map[string]interface{}) error

	SendEventNotification(ctx context.Context, userID uuid.UUID, email, name string,
		eventID uuid.UUID, notificationType NotificationType,
		templateData map[string]interface{}) error

	Start(ctx context.Context) error
	Stop() error
	HealthCheck(ctx context.Context) error
}

type ServiceConfig struct {
	Environment string

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
	EnableEmailChannel bool
	EnableSMSChannel   bool
	EnablePushChannel  bool
}

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
	}

	return config
}

type UnifiedNotificationService struct {
	config    *ServiceConfig
	producer  NotificationProducer
	consumer  NotificationConsumer
	publisher *NotificationPublisher

	// Services
	emailService EmailService

	// State
	isRunning bool
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewUnifiedNotificationService(config *ServiceConfig) (NotificationService, error) {
	if config == nil {
		config = NewServiceConfigFromEnv()
	}

	producerConfig := DefaultKafkaProducerConfig()
	producerConfig.Brokers = config.KafkaBrokers
	producerConfig.NotificationTopic = config.NotificationTopic

	producer, err := NewKafkaNotificationProducer(producerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification producer: %w", err)
	}

	consumerConfig := DefaultConsumerConfig()
	consumerConfig.Brokers = config.KafkaBrokers
	consumerConfig.Topics = []string{config.NotificationTopic}
	consumerConfig.GroupID = config.ConsumerGroupID

	consumer, err := NewKafkaNotificationConsumer(consumerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification consumer: %w", err)
	}

	publisher := NewNotificationPublisher(producer)

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

	ctx, cancel := context.WithCancel(context.Background())

	return &UnifiedNotificationService{
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

func (uns *UnifiedNotificationService) Start(ctx context.Context) error {
	uns.mu.Lock()
	defer uns.mu.Unlock()

	if uns.isRunning {
		return fmt.Errorf("notification service is already running")
	}

	log.Printf("🚀 Starting Unified Notification Service...")

	if uns.config.EnableEmailChannel {
		emailHandler := NewEmailChannelHandler(uns.emailService)
		if err := uns.consumer.RegisterHandler(NotificationChannelEmail, emailHandler); err != nil {
			return fmt.Errorf("failed to register email handler: %w", err)
		}
	}

	err := uns.consumer.StartConsumers(uns.ctx, uns.config.NumConsumerWorkers)
	if err != nil {
		return fmt.Errorf("failed to start consumers: %w", err)
	}

	uns.isRunning = true
	log.Printf("✅ Unified Notification Service started successfully")

	return nil
}

func (uns *UnifiedNotificationService) Stop() error {
	uns.mu.Lock()
	defer uns.mu.Unlock()

	if !uns.isRunning {
		return fmt.Errorf("notification service is not running")
	}

	log.Printf("🛑 Stopping Unified Notification Service...")

	uns.cancel()

	if err := uns.consumer.Stop(); err != nil {
		log.Printf("Error stopping consumer: %v", err)
	}

	if err := uns.producer.Close(); err != nil {
		log.Printf("Error closing producer: %v", err)
	}

	uns.isRunning = false
	log.Printf("✅ Unified Notification Service stopped")

	return nil
}

func (uns *UnifiedNotificationService) SendNotification(ctx context.Context, notification *UnifiedNotification) error {
	return uns.producer.PublishNotification(ctx, notification)
}

func (uns *UnifiedNotificationService) SendBatchNotifications(ctx context.Context, notifications []*UnifiedNotification) error {
	return uns.producer.PublishBatchNotifications(ctx, notifications)
}

func (uns *UnifiedNotificationService) SendWaitlistNotification(ctx context.Context, userID uuid.UUID, email, name string,
	eventID, waitlistEntryID uuid.UUID, notificationType NotificationType,
	templateData map[string]interface{}) error {

	return uns.publisher.PublishWaitlistNotification(ctx, userID, email, name, eventID, waitlistEntryID, notificationType, templateData)
}

func (uns *UnifiedNotificationService) SendBookingNotification(ctx context.Context, userID uuid.UUID, email, name string,
	bookingID, eventID uuid.UUID, notificationType NotificationType,
	templateData map[string]interface{}) error {

	return uns.publisher.PublishBookingNotification(ctx, userID, email, name, bookingID, eventID, notificationType, templateData)
}

func (uns *UnifiedNotificationService) SendEventNotification(ctx context.Context, userID uuid.UUID, email, name string,
	eventID uuid.UUID, notificationType NotificationType,
	templateData map[string]interface{}) error {

	return uns.publisher.PublishEventNotification(ctx, userID, email, name, eventID, notificationType, templateData)
}

func (uns *UnifiedNotificationService) HealthCheck(ctx context.Context) error {
	uns.mu.RLock()
	isRunning := uns.isRunning
	uns.mu.RUnlock()

	if !isRunning {
		return fmt.Errorf("notification service is not running")
	}

	if err := uns.producer.HealthCheck(ctx); err != nil {
		return fmt.Errorf("producer health check failed: %w", err)
	}

	if err := uns.consumer.HealthCheck(ctx); err != nil {
		return fmt.Errorf("consumer health check failed: %w", err)
	}

	return nil
}

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

var (
	globalService NotificationService
	globalOnce    sync.Once
)

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

func InitializeGlobalNotificationService(ctx context.Context) error {
	service := GetGlobalNotificationService()
	return service.Start(ctx)
}

func StopGlobalNotificationService() error {
	if globalService != nil {
		return globalService.Stop()
	}
	return nil
}
