package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/IBM/sarama"
)

type NotificationConsumer interface {
	StartConsumers(ctx context.Context, numWorkers int) error
	Stop() error
	HealthCheck(ctx context.Context) error
}

type ConsumerConfig struct {
	Brokers              []string
	GroupID              string
	Topics               []string
	SessionTimeoutMs     int
	HeartbeatMs          int
	RetryBackoffMs       int
	MaxProcessingTime    time.Duration
	AutoCommit           bool
	OffsetOldest         bool
	MaxRetries           int
	RetryBackoffDuration time.Duration
}

func DefaultConsumerConfig() *ConsumerConfig {
	return &ConsumerConfig{
		Brokers:              []string{"localhost:9092"},
		GroupID:              "evently-notification-workers",
		Topics:               []string{"notifications"},
		SessionTimeoutMs:     30000,
		HeartbeatMs:          3000,
		RetryBackoffMs:       100,
		MaxProcessingTime:    5 * time.Minute,
		AutoCommit:           true,
		OffsetOldest:         false,
		MaxRetries:           3,
		RetryBackoffDuration: time.Second,
	}
}

type KafkaNotificationConsumer struct {
	consumerGroup sarama.ConsumerGroup
	config        *ConsumerConfig
	emailService  EmailService
	topics        []string
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewKafkaNotificationConsumer(config *ConsumerConfig, emailService EmailService) (NotificationConsumer, error) {
	saramaConfig := sarama.NewConfig()

	saramaConfig.Consumer.Group.Session.Timeout = time.Duration(config.SessionTimeoutMs) * time.Millisecond
	saramaConfig.Consumer.Group.Heartbeat.Interval = time.Duration(config.HeartbeatMs) * time.Millisecond
	saramaConfig.Consumer.Retry.Backoff = time.Duration(config.RetryBackoffMs) * time.Millisecond
	saramaConfig.Consumer.MaxProcessingTime = config.MaxProcessingTime
	saramaConfig.Consumer.Return.Errors = true

	if config.OffsetOldest {
		saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	} else {
		saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	}

	if config.AutoCommit {
		saramaConfig.Consumer.Offsets.AutoCommit.Enable = true
		saramaConfig.Consumer.Offsets.AutoCommit.Interval = 1 * time.Second
	}

	consumerGroup, err := sarama.NewConsumerGroup(config.Brokers, config.GroupID, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &KafkaNotificationConsumer{
		consumerGroup: consumerGroup,
		config:        config,
		emailService:  emailService,
		topics:        config.Topics,
		ctx:           ctx,
		cancel:        cancel,
	}, nil
}

func (knc *KafkaNotificationConsumer) StartConsumers(ctx context.Context, numWorkers int) error {
	log.Printf("📥 Starting %d notification consumer workers for topics: %v", numWorkers, knc.topics)

	// Start error handler goroutine
	go knc.handleErrors()

	// Start consumer workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			knc.runWorker(ctx, workerID)
		}(i)
	}

	log.Printf("📥 All %d notification consumer workers started", numWorkers)
	return nil
}

func (knc *KafkaNotificationConsumer) runWorker(ctx context.Context, workerID int) {
	consumer := &ConsumerGroupHandler{
		consumer:     knc,
		workerID:     workerID,
		emailService: knc.emailService,
	}

	for {
		select {
		case <-ctx.Done():
			log.Printf("📥 Worker %d shutting down", workerID)
			return
		default:
			err := knc.consumerGroup.Consume(ctx, knc.topics, consumer)
			if err != nil {
				log.Printf("📥 Worker %d error consuming messages: %v", workerID, err)
				time.Sleep(time.Second)
			}
		}
	}
}

func (knc *KafkaNotificationConsumer) handleErrors() {
	for err := range knc.consumerGroup.Errors() {
		log.Printf("📥 Consumer group error: %v", err)
	}
}

func (knc *KafkaNotificationConsumer) Stop() error {
	log.Println("📥 Stopping notification consumer...")
	knc.cancel()

	err := knc.consumerGroup.Close()
	if err != nil {
		return fmt.Errorf("failed to close consumer group: %w", err)
	}

	log.Println("📥 Notification consumer stopped")
	return nil
}

func (knc *KafkaNotificationConsumer) HealthCheck(ctx context.Context) error {
	select {
	case <-knc.ctx.Done():
		return fmt.Errorf("consumer context is cancelled")
	default:
		if knc.emailService == nil {
			return fmt.Errorf("email service not configured")
		}
		return nil
	}
}

type ConsumerGroupHandler struct {
	consumer     *KafkaNotificationConsumer
	workerID     int
	emailService EmailService
}

func (h *ConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	log.Printf("📥 Worker %d: Consumer group session started", h.workerID)
	return nil
}

func (h *ConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	log.Printf("📥 Worker %d: Consumer group session ended", h.workerID)
	return nil
}

func (h *ConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			err := h.processMessage(session.Context(), message)
			if err != nil {
				log.Printf("📥 Worker %d: Error processing message: %v", h.workerID, err)
			} else {
				session.MarkMessage(message, "")
			}

		case <-session.Context().Done():
			return nil
		}
	}
}

func (h *ConsumerGroupHandler) processMessage(ctx context.Context, message *sarama.ConsumerMessage) error {
	log.Printf("📥 Worker %d: Processing notification from topic %s, partition %d, offset %d",
		h.workerID, message.Topic, message.Partition, message.Offset)

	// Parse the notification from message value
	var notification EmailNotification
	if err := json.Unmarshal(message.Value, &notification); err != nil {
		return fmt.Errorf("failed to unmarshal notification: %w", err)
	}

	// Check if notification is expired
	if notification.IsExpired() {
		log.Printf("📥 Worker %d: Notification %s expired, skipping", h.workerID, notification.ID)
		return nil
	}

	// Update status to sending
	notification.Status = NotificationStatusSending

	// Process email notification with retry logic
	err := h.executeWithRetry(ctx, &notification)
	if err != nil {
		notification.MarkFailed(err)
		return err
	}

	notification.MarkSent()
	log.Printf("📧 Worker %d: Email notification sent successfully to %s", h.workerID, notification.RecipientEmail)
	return nil
}

func (h *ConsumerGroupHandler) executeWithRetry(ctx context.Context, notification *EmailNotification) error {
	maxRetries := h.consumer.config.MaxRetries
	backoff := h.consumer.config.RetryBackoffDuration

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := h.emailService.SendNotification(ctx, notification)
		if err == nil {
			if attempt > 0 {
				log.Printf("📥 Worker %d: Successfully processed notification after %d retries", h.workerID, attempt)
			}
			return nil
		}

		if attempt == maxRetries {
			log.Printf("📥 Worker %d: Failed to process notification after %d attempts: %v", h.workerID, maxRetries, err)
			return err
		}

		// Exponential backoff
		delay := backoff * time.Duration(1<<attempt)
		log.Printf("📥 Worker %d: Retry %d for notification processing after %v", h.workerID, attempt+1, delay)

		select {
		case <-time.After(delay):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}
