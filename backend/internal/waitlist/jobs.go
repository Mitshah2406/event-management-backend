package waitlist

import (
	"context"
	"log"
	"time"
)

// JobProcessor handles background jobs for waitlist operations
type JobProcessor struct {
	service Service
	config  *JobConfig
	done    chan struct{}
}

// JobConfig contains configuration for background jobs
type JobConfig struct {
	ExpiryCheckInterval time.Duration
	AnalyticsInterval   time.Duration
	BatchSize           int
}

// DefaultJobConfig returns default job configuration
func DefaultJobConfig() *JobConfig {
	return &JobConfig{
		ExpiryCheckInterval: 1 * time.Minute, // Check for expired bookings every minute
		AnalyticsInterval:   24 * time.Hour,  // Update analytics daily
		BatchSize:           100,             // Process 100 expired entries at a time
	}
}

// NewJobProcessor creates a new job processor
func NewJobProcessor(service Service, config *JobConfig) *JobProcessor {
	if config == nil {
		config = DefaultJobConfig()
	}

	return &JobProcessor{
		service: service,
		config:  config,
		done:    make(chan struct{}),
	}
}

// Start starts all background jobs
func (jp *JobProcessor) Start(ctx context.Context) {
	log.Println("Starting waitlist background jobs...")

	// Start expired booking processor
	go jp.startExpiryProcessor(ctx)

	// Start analytics updater
	go jp.startAnalyticsUpdater(ctx)

	log.Println("Waitlist background jobs started")
}

// Stop stops all background jobs
func (jp *JobProcessor) Stop() {
	log.Println("Stopping waitlist background jobs...")
	close(jp.done)
	log.Println("Waitlist background jobs stopped")
}

// startExpiryProcessor starts the expired booking window processor
func (jp *JobProcessor) startExpiryProcessor(ctx context.Context) {
	ticker := time.NewTicker(jp.config.ExpiryCheckInterval)
	defer ticker.Stop()

	log.Printf("Started expired booking processor with %v interval", jp.config.ExpiryCheckInterval)

	for {
		select {
		case <-ticker.C:
			jp.processExpiredBookings(ctx)
		case <-jp.done:
			return
		case <-ctx.Done():
			return
		}
	}
}

// processExpiredBookings processes expired booking windows
func (jp *JobProcessor) processExpiredBookings(ctx context.Context) {
	processed, err := jp.service.ProcessExpiredBookingWindows(ctx)
	if err != nil {
		log.Printf("Error processing expired bookings: %v", err)
		return
	}

	if processed > 0 {
		log.Printf("Processed %d expired booking windows", processed)
	}
}

// startAnalyticsUpdater starts the daily analytics updater
func (jp *JobProcessor) startAnalyticsUpdater(ctx context.Context) {
	ticker := time.NewTicker(jp.config.AnalyticsInterval)
	defer ticker.Stop()

	log.Printf("Started analytics updater with %v interval", jp.config.AnalyticsInterval)

	// Run immediately on startup
	jp.updateAnalytics(ctx)

	for {
		select {
		case <-ticker.C:
			jp.updateAnalytics(ctx)
		case <-jp.done:
			return
		case <-ctx.Done():
			return
		}
	}
}

// updateAnalytics updates daily analytics
func (jp *JobProcessor) updateAnalytics(ctx context.Context) {
	err := jp.service.UpdateDailyAnalytics(ctx)
	if err != nil {
		log.Printf("Error updating analytics: %v", err)
		return
	}

	log.Println("Updated daily waitlist analytics")
}

// GetJobStatus returns the status of background jobs
func (jp *JobProcessor) GetJobStatus() map[string]interface{} {
	return map[string]interface{}{
		"expiry_check_interval": jp.config.ExpiryCheckInterval.String(),
		"analytics_interval":    jp.config.AnalyticsInterval.String(),
		"batch_size":            jp.config.BatchSize,
		"status":                "running",
	}
}
