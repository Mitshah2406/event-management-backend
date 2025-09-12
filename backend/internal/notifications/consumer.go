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

// NotificationConsumer interface defines the contract for consuming notifications
type NotificationConsumer interface {
	StartConsumers(ctx context.Context, numWorkers int) error
	Stop() error
	RegisterHandler(channel NotificationChannel, handler ChannelHandler) error
	HealthCheck(ctx context.Context) error
}

// ChannelHandler processes notifications for a specific delivery channel
type ChannelHandler interface {
	Handle(ctx context.Context, notification *UnifiedNotification) error
	GetChannel() NotificationChannel
}

// ConsumerConfig contains configuration for the notification consumer
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

// DefaultConsumerConfig returns a default consumer configuration
func DefaultConsumerConfig() *ConsumerConfig {
	return &ConsumerConfig{
		Brokers:              []string{"localhost:9092"},
		GroupID:              "evently-notification-workers",
		Topics:               []string{"notifications"},
		SessionTimeoutMs:     30000, // 30 seconds
		HeartbeatMs:          3000,  // 3 seconds
		RetryBackoffMs:       100,   // 100ms
		MaxProcessingTime:    5 * time.Minute,
		AutoCommit:           true,
		OffsetOldest:         false, // Start from latest
		MaxRetries:           3,
		RetryBackoffDuration: time.Second,
	}
}

// KafkaNotificationConsumer handles consuming notifications from Kafka
type KafkaNotificationConsumer struct {
	consumerGroup sarama.ConsumerGroup
	config        *ConsumerConfig
	handlers      map[NotificationChannel]ChannelHandler
	handlersMu    sync.RWMutex
	topics        []string
	ready         chan bool
	ctx           context.Context
	cancel        context.CancelFunc
	metrics       *ConsumerMetrics
	metricsMu     sync.RWMutex
}

// NewKafkaNotificationConsumer creates a new Kafka notification consumer
func NewKafkaNotificationConsumer(config *ConsumerConfig) (NotificationConsumer, error) {
	saramaConfig := sarama.NewConfig()

	// Consumer configuration
	saramaConfig.Consumer.Group.Session.Timeout = time.Duration(config.SessionTimeoutMs) * time.Millisecond
	saramaConfig.Consumer.Group.Heartbeat.Interval = time.Duration(config.HeartbeatMs) * time.Millisecond
	saramaConfig.Consumer.Retry.Backoff = time.Duration(config.RetryBackoffMs) * time.Millisecond
	saramaConfig.Consumer.MaxProcessingTime = config.MaxProcessingTime
	saramaConfig.Consumer.Return.Errors = true

	// Offset configuration
	if config.OffsetOldest {
		saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	} else {
		saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	}

	// Auto-commit configuration
	if config.AutoCommit {
		saramaConfig.Consumer.Offsets.AutoCommit.Enable = true
		saramaConfig.Consumer.Offsets.AutoCommit.Interval = 1 * time.Second
	}

	// Create consumer group
	consumerGroup, err := sarama.NewConsumerGroup(config.Brokers, config.GroupID, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &KafkaNotificationConsumer{
		consumerGroup: consumerGroup,
		config:        config,
		handlers:      make(map[NotificationChannel]ChannelHandler),
		topics:        config.Topics,
		ready:         make(chan bool),
		ctx:           ctx,
		cancel:        cancel,
		metrics:       &ConsumerMetrics{},
	}, nil
}

// RegisterHandler registers a handler for a specific notification channel
func (knc *KafkaNotificationConsumer) RegisterHandler(channel NotificationChannel, handler ChannelHandler) error {
	knc.handlersMu.Lock()
	defer knc.handlersMu.Unlock()

	if handler.GetChannel() != channel {
		return fmt.Errorf("handler channel mismatch: expected %s, got %s", channel, handler.GetChannel())
	}

	knc.handlers[channel] = handler
	log.Printf("📥 Registered handler for notification channel: %s", channel)
	return nil
}

// StartConsumers starts the consumer group with specified number of workers
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

	// Wait for all workers to be ready
	for i := 0; i < numWorkers; i++ {
		<-knc.ready
	}

	knc.updateMetrics(func(m *ConsumerMetrics) {
		m.ActiveWorkers = numWorkers
	})

	log.Printf("📥 All %d notification consumer workers are ready and consuming messages", numWorkers)

	// Wait for context cancellation or all workers to finish
	go func() {
		wg.Wait()
		knc.cancel()
	}()

	return nil
}

// runWorker runs a single consumer worker
func (knc *KafkaNotificationConsumer) runWorker(ctx context.Context, workerID int) {
	consumer := &ConsumerGroupHandler{
		consumer: knc,
		workerID: workerID,
		ready:    knc.ready,
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
				time.Sleep(time.Second) // Brief pause before retry
			}
		}
	}
}

// handleErrors handles consumer errors
func (knc *KafkaNotificationConsumer) handleErrors() {
	for err := range knc.consumerGroup.Errors() {
		log.Printf("📥 Consumer group error: %v", err)
		knc.updateMetrics(func(m *ConsumerMetrics) {
			m.ErrorCount++
		})
		// Here you could implement error metrics, alerting, etc.
	}
}

// Stop stops the consumer
func (knc *KafkaNotificationConsumer) Stop() error {
	log.Println("📥 Stopping notification consumer...")
	knc.cancel()

	err := knc.consumerGroup.Close()
	if err != nil {
		return fmt.Errorf("failed to close consumer group: %w", err)
	}

	knc.updateMetrics(func(m *ConsumerMetrics) {
		m.ActiveWorkers = 0
	})

	log.Println("📥 Notification consumer stopped")
	return nil
}

// HealthCheck performs a health check on the consumer
func (knc *KafkaNotificationConsumer) HealthCheck(ctx context.Context) error {
	select {
	case <-knc.ctx.Done():
		return fmt.Errorf("consumer context is cancelled")
	default:
		// Check if we have registered handlers
		knc.handlersMu.RLock()
		handlerCount := len(knc.handlers)
		knc.handlersMu.RUnlock()

		if handlerCount == 0 {
			return fmt.Errorf("no handlers registered")
		}

		return nil
	}
}

// updateMetrics safely updates consumer metrics
func (knc *KafkaNotificationConsumer) updateMetrics(updateFn func(*ConsumerMetrics)) {
	knc.metricsMu.Lock()
	defer knc.metricsMu.Unlock()
	updateFn(knc.metrics)
}

// GetMetrics returns current consumer metrics
func (knc *KafkaNotificationConsumer) GetMetrics() *ConsumerMetrics {
	knc.metricsMu.RLock()
	defer knc.metricsMu.RUnlock()

	// Return a copy to avoid race conditions
	return &ConsumerMetrics{
		MessagesProcessed: knc.metrics.MessagesProcessed,
		MessagesSucceeded: knc.metrics.MessagesSucceeded,
		MessagesFailed:    knc.metrics.MessagesFailed,
		LastMessageTime:   knc.metrics.LastMessageTime,
		ProcessingLatency: knc.metrics.ProcessingLatency,
		ActiveWorkers:     knc.metrics.ActiveWorkers,
		ErrorCount:        knc.metrics.ErrorCount,
	}
}

// ConsumerGroupHandler implements sarama.ConsumerGroupHandler
type ConsumerGroupHandler struct {
	consumer *KafkaNotificationConsumer
	workerID int
	ready    chan bool
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (h *ConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	log.Printf("📥 Worker %d: Consumer group session started", h.workerID)
	close(h.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (h *ConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	log.Printf("📥 Worker %d: Consumer group session ended", h.workerID)
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages()
func (h *ConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			startTime := time.Now()
			err := h.processMessage(session.Context(), message)
			processingTime := time.Since(startTime)

			h.consumer.updateMetrics(func(m *ConsumerMetrics) {
				m.MessagesProcessed++
				m.LastMessageTime = startTime
				m.ProcessingLatency = processingTime
				if err != nil {
					m.MessagesFailed++
				} else {
					m.MessagesSucceeded++
				}
			})

			if err != nil {
				log.Printf("📥 Worker %d: Error processing message: %v", h.workerID, err)
				// Continue processing other messages even if one fails
			} else {
				session.MarkMessage(message, "")
			}

		case <-session.Context().Done():
			return nil
		}
	}
}

// processMessage processes a single Kafka message
func (h *ConsumerGroupHandler) processMessage(ctx context.Context, message *sarama.ConsumerMessage) error {
	log.Printf("📥 Worker %d: Processing notification from topic %s, partition %d, offset %d",
		h.workerID, message.Topic, message.Partition, message.Offset)

	// Parse the notification from message value
	var notification UnifiedNotification
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

	// Process each channel
	var lastErr error
	successCount := 0

	for _, channel := range notification.Channels {
		err := h.processChannel(ctx, &notification, channel)
		if err != nil {
			log.Printf("📥 Worker %d: Failed to process channel %s: %v", h.workerID, channel, err)
			lastErr = err
		} else {
			successCount++
		}
	}

	// Update final status
	if successCount > 0 {
		if successCount == len(notification.Channels) {
			notification.Status = NotificationStatusSent
			now := time.Now()
			notification.SentAt = &now
		} else {
			notification.Status = NotificationStatusFailed // Partial failure
		}
	} else {
		notification.Status = NotificationStatusFailed
		if lastErr != nil {
			notification.MarkFailed(NotificationChannelEmail, lastErr) // Use email as default for error tracking
		}
	}

	// If all channels failed and we should retry, send to retry queue
	if successCount == 0 && notification.ShouldRetry() {
		return h.scheduleRetry(ctx, &notification)
	}

	return lastErr
}

// processChannel processes a notification for a specific channel
func (h *ConsumerGroupHandler) processChannel(ctx context.Context, notification *UnifiedNotification, channel NotificationChannel) error {
	h.consumer.handlersMu.RLock()
	handler, exists := h.consumer.handlers[channel]
	h.consumer.handlersMu.RUnlock()

	if !exists {
		log.Printf("📥 Worker %d: No handler registered for channel: %s, skipping", h.workerID, channel)
		return nil // Not an error, just no handler available
	}

	// Process with timeout
	processCtx, cancel := context.WithTimeout(ctx, h.consumer.config.MaxProcessingTime)
	defer cancel()

	// Execute handler with retry logic
	return h.executeWithRetry(processCtx, handler, notification, channel)
}

// executeWithRetry executes a handler with retry logic
func (h *ConsumerGroupHandler) executeWithRetry(ctx context.Context, handler ChannelHandler, notification *UnifiedNotification, channel NotificationChannel) error {
	maxRetries := h.consumer.config.MaxRetries
	backoff := h.consumer.config.RetryBackoffDuration

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := handler.Handle(ctx, notification)
		if err == nil {
			if attempt > 0 {
				log.Printf("📥 Worker %d: Successfully processed notification after %d retries", h.workerID, attempt)
			}
			// Mark as delivered for this channel
			notification.MarkDelivered(channel, nil)
			return nil
		}

		if attempt == maxRetries {
			log.Printf("📥 Worker %d: Failed to process notification after %d attempts: %v", h.workerID, maxRetries, err)
			notification.MarkFailed(channel, err)
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

// scheduleRetry schedules a notification for retry
func (h *ConsumerGroupHandler) scheduleRetry(ctx context.Context, notification *UnifiedNotification) error {
	notification.IncrementRetry()

	// In a more sophisticated implementation, you would republish to a retry topic
	// or delay queue. For now, we'll just log the retry attempt.
	log.Printf("📥 Worker %d: Scheduling retry for notification %s (attempt %d/%d)",
		h.workerID, notification.ID, notification.RetryCount, notification.MaxRetries)

	// TODO: Implement retry queue or delay mechanism
	return fmt.Errorf("retry scheduled for notification %s", notification.ID)
}

// EmailChannelHandler handles email notifications
type EmailChannelHandler struct {
	emailService EmailService
}

// NewEmailChannelHandler creates a new email channel handler
func NewEmailChannelHandler(emailService EmailService) ChannelHandler {
	return &EmailChannelHandler{
		emailService: emailService,
	}
}

// Handle processes email notifications
func (e *EmailChannelHandler) Handle(ctx context.Context, notification *UnifiedNotification) error {
	log.Printf("📧 Processing email notification for %s (ID: %s)", notification.RecipientEmail, notification.ID)

	err := e.emailService.SendNotification(ctx, notification)
	if err != nil {
		return fmt.Errorf("failed to send email notification: %w", err)
	}

	log.Printf("📧 Email notification sent successfully to %s", notification.RecipientEmail)
	return nil
}

// GetChannel returns the channel this handler processes
func (e *EmailChannelHandler) GetChannel() NotificationChannel {
	return NotificationChannelEmail
}

// SMSChannelHandler handles SMS notifications
type SMSChannelHandler struct {
	smsService SMSService
}

// SMSService interface for SMS sending (placeholder)
type SMSService interface {
	SendSMS(ctx context.Context, to, message string) error
}

// NewSMSChannelHandler creates a new SMS channel handler
func NewSMSChannelHandler(smsService SMSService) ChannelHandler {
	return &SMSChannelHandler{
		smsService: smsService,
	}
}

// Handle processes SMS notifications
func (s *SMSChannelHandler) Handle(ctx context.Context, notification *UnifiedNotification) error {
	if notification.RecipientPhone == nil {
		log.Printf("📱 Skipping SMS - no phone number for user %s", notification.RecipientEmail)
		return nil // Not an error, just no phone number
	}

	log.Printf("📱 Processing SMS notification for %s (ID: %s)", *notification.RecipientPhone, notification.ID)

	// Generate SMS message (simplified)
	message := fmt.Sprintf("%s - Evently", notification.Subject)
	if len(message) > 160 { // SMS length limit
		message = message[:157] + "..."
	}

	err := s.smsService.SendSMS(ctx, *notification.RecipientPhone, message)
	if err != nil {
		return fmt.Errorf("failed to send SMS notification: %w", err)
	}

	log.Printf("📱 SMS notification sent successfully to %s", *notification.RecipientPhone)
	return nil
}

// GetChannel returns the channel this handler processes
func (s *SMSChannelHandler) GetChannel() NotificationChannel {
	return NotificationChannelSMS
}

// ConsumerMetrics contains metrics for monitoring consumer performance
type ConsumerMetrics struct {
	MessagesProcessed int64
	MessagesSucceeded int64
	MessagesFailed    int64
	LastMessageTime   time.Time
	ProcessingLatency time.Duration
	ActiveWorkers     int
	ErrorCount        int64
}

// MockSMSService is a mock implementation for SMS sending
type MockSMSServiceImpl struct{}

// NewMockSMSServiceImpl creates a new mock SMS service
func NewMockSMSServiceImpl() *MockSMSServiceImpl {
	return &MockSMSServiceImpl{}
}

// SendSMS sends a mock SMS
func (m *MockSMSServiceImpl) SendSMS(ctx context.Context, to, message string) error {
	log.Printf("📱 [MOCK SMS] To: %s, Message: %s", to, message)
	return nil
}
