package waitlist

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// NotificationService defines the interface for sending notifications (to avoid import cycles)
type NotificationService interface {
	SendWaitlistNotification(ctx context.Context, userID uuid.UUID, email, name string,
		eventID, waitlistEntryID uuid.UUID, notificationType string,
		templateData map[string]interface{}) error
}

// UserService defines the interface for fetching user details (to avoid import cycles)
type UserService interface {
	GetUserByID(ctx context.Context, userID uuid.UUID) (email, firstName, lastName string, err error)
}

// Service interface defines the contract for waitlist business operations
type Service interface {
	// Core waitlist operations
	JoinWaitlist(ctx context.Context, userID uuid.UUID, request *JoinWaitlistRequest) (*WaitlistResponse, error)
	LeaveWaitlist(ctx context.Context, userID, eventID uuid.UUID) error
	GetWaitlistStatus(ctx context.Context, userID, eventID uuid.UUID) (*WaitlistResponse, error)

	// Event-triggered operations
	ProcessCancellation(ctx context.Context, eventID uuid.UUID, freedTickets int) error
	ProcessBookingExpiry(ctx context.Context, userID, eventID uuid.UUID) error

	// Notification operations
	NotifyNextInLine(ctx context.Context, eventID uuid.UUID, availableTickets int) error
	NotifyPositionUpdate(ctx context.Context, eventID uuid.UUID) error

	// Admin operations
	GetWaitlistStats(ctx context.Context, eventID uuid.UUID) (*WaitlistStatsResponse, error)
	GetWaitlistEntries(ctx context.Context, eventID uuid.UUID, status WaitlistStatus) ([]WaitlistEntry, error)

	// Background job operations
	ProcessExpiredBookingWindows(ctx context.Context) (int, error)
	UpdateDailyAnalytics(ctx context.Context) error

	// Booking operations
	MarkAsConverted(ctx context.Context, userID, eventID, bookingID uuid.UUID) error
	GetWaitlistStatusForBooking(ctx context.Context, userID, eventID uuid.UUID) (*WaitlistStatusForBooking, error)
}

// service implements the Service interface
type service struct {
	repo                Repository
	notificationService NotificationService
	userService         UserService
	config              *ServiceConfig
}

// ServiceConfig contains configuration for the waitlist service
type ServiceConfig struct {
	BookingWindowDuration time.Duration
	MaxWaitlistSize       int
	MaxQuantityPerUser    int
	NotificationTimeout   time.Duration
}

// DefaultServiceConfig returns default service configuration
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		BookingWindowDuration: BookingWindowDuration,
		MaxWaitlistSize:       MaxWaitlistSize,
		MaxQuantityPerUser:    MaxQuantityPerUser,
		NotificationTimeout:   5 * time.Second,
	}
}

// NewService creates a new waitlist service
func NewService(repo Repository, notificationService NotificationService, userService UserService, config *ServiceConfig) Service {
	if config == nil {
		config = DefaultServiceConfig()
	}

	return &service{
		repo:                repo,
		notificationService: notificationService,
		userService:         userService,
		config:              config,
	}
}

// JoinWaitlist adds a user to an event's waitlist
func (s *service) JoinWaitlist(ctx context.Context, userID uuid.UUID, request *JoinWaitlistRequest) (*WaitlistResponse, error) {
	// Validate request
	if err := s.validateJoinRequest(request); err != nil {
		return nil, fmt.Errorf("invalid join request: %w", err)
	}

	// Check if user is already in waitlist
	existingEntry, err := s.repo.GetEntry(ctx, userID, request.EventID)
	if err == nil && existingEntry != nil {
		return nil, fmt.Errorf("user already in waitlist for event %s", request.EventID)
	}

	// Check waitlist capacity
	queueLength, err := s.repo.GetQueueLength(ctx, request.EventID)
	if err != nil {
		return nil, fmt.Errorf("failed to check queue length: %w", err)
	}

	if queueLength >= s.config.MaxWaitlistSize {
		return nil, fmt.Errorf("waitlist is full (max %d users)", s.config.MaxWaitlistSize)
	}

	// Create waitlist entry
	entry := &WaitlistEntry{
		UserID:      userID,
		EventID:     request.EventID,
		Quantity:    request.Quantity,
		Status:      WaitlistStatusActive,
		Preferences: request.Preferences,
		JoinedAt:    time.Now(),
	}

	// Add to Redis queue first to get position
	err = s.repo.AddToQueue(ctx, entry)
	if err != nil {
		return nil, fmt.Errorf("failed to add to queue: %w", err)
	}

	// Save to database
	err = s.repo.CreateEntry(ctx, entry)
	if err != nil {
		// Rollback Redis operation
		s.repo.RemoveFromQueue(ctx, userID, request.EventID)
		return nil, fmt.Errorf("failed to create waitlist entry: %w", err)
	}

	log.Printf("User %s joined waitlist for event %s at position %d", userID, request.EventID, entry.Position)

	// Return response
	response := &WaitlistResponse{
		ID:          entry.ID,
		EventID:     entry.EventID,
		Position:    entry.Position,
		Quantity:    entry.Quantity,
		Status:      entry.Status,
		Preferences: entry.Preferences,
		JoinedAt:    entry.JoinedAt,
	}

	// Estimate wait time (this is a simple heuristic)
	if entry.Position > 1 {
		estimatedWait := time.Duration(entry.Position-1) * 30 * time.Minute // Assume 30 min per position
		response.EstimatedWait = &estimatedWait
	}

	return response, nil
}

// LeaveWaitlist removes a user from an event's waitlist
func (s *service) LeaveWaitlist(ctx context.Context, userID, eventID uuid.UUID) error {
	// Get existing entry
	entry, err := s.repo.GetEntry(ctx, userID, eventID)
	if err != nil {
		return fmt.Errorf("waitlist entry not found: %w", err)
	}

	if entry.Status != WaitlistStatusActive {
		return fmt.Errorf("cannot leave waitlist in status %s", entry.Status)
	}

	// Remove from Redis queue
	err = s.repo.RemoveFromQueue(ctx, userID, eventID)
	if err != nil {
		return fmt.Errorf("failed to remove from queue: %w", err)
	}

	// Update database entry status
	entry.Status = WaitlistStatusCancelled
	err = s.repo.UpdateEntry(ctx, entry)
	if err != nil {
		return fmt.Errorf("failed to update entry status: %w", err)
	}

	log.Printf("User %s left waitlist for event %s", userID, eventID)

	// Update positions for remaining users
	go func() {
		if err := s.repo.UpdatePositions(context.Background(), eventID); err != nil {
			log.Printf("Failed to update positions after user left: %v", err)
		}
	}()

	return nil
}

// GetWaitlistStatus gets a user's current waitlist status
func (s *service) GetWaitlistStatus(ctx context.Context, userID, eventID uuid.UUID) (*WaitlistResponse, error) {
	// Get entry from database
	entry, err := s.repo.GetEntry(ctx, userID, eventID)
	if err != nil {
		return nil, fmt.Errorf("waitlist entry not found: %w", err)
	}

	// Get current position from Redis if active
	var currentPosition int
	if entry.Status == WaitlistStatusActive {
		pos, err := s.repo.GetPosition(ctx, userID, eventID)
		if err == nil {
			currentPosition = pos
		} else {
			currentPosition = entry.Position // Fallback to stored position
		}
	} else {
		currentPosition = entry.Position
	}

	response := &WaitlistResponse{
		ID:          entry.ID,
		EventID:     entry.EventID,
		Position:    currentPosition,
		Quantity:    entry.Quantity,
		Status:      entry.Status,
		Preferences: entry.Preferences,
		JoinedAt:    entry.JoinedAt,
		NotifiedAt:  entry.NotifiedAt,
		ExpiresAt:   entry.ExpiresAt,
	}

	// Calculate time remaining if notified
	if entry.Status == WaitlistStatusNotified && entry.ExpiresAt != nil {
		timeRemaining := entry.TimeRemaining()
		if timeRemaining != nil {
			response.EstimatedWait = timeRemaining
		}
	}

	return response, nil
}

// ProcessCancellation handles event cancellations that free up tickets
func (s *service) ProcessCancellation(ctx context.Context, eventID uuid.UUID, freedTickets int) error {
	log.Printf("🎫 WAITLIST: Processing cancellation for event %s, freed tickets: %d", eventID, freedTickets)

	// Get next users in queue
	nextInQueue, err := s.repo.GetNextInQueue(ctx, eventID, freedTickets)
	if err != nil {
		log.Printf("❌ WAITLIST ERROR: Failed to get next in queue for event %s: %v", eventID, err)
		return fmt.Errorf("failed to get next in queue: %w", err)
	}

	if len(nextInQueue) == 0 {
		log.Printf("📭 WAITLIST: No users in waitlist for event %s - no notifications sent", eventID)
		return nil
	}

	log.Printf("👥 WAITLIST: Found %d users in queue for event %s, will notify up to %d users",
		len(nextInQueue), eventID, freedTickets)

	// Notify users and update their status
	var notifiedUsers []uuid.UUID
	for i, entry := range nextInQueue {
		if i >= freedTickets {
			break // Don't notify more users than available tickets
		}

		// Update entry status to notified
		entry.Status = WaitlistStatusNotified
		entry.NotifiedAt = &time.Time{}
		*entry.NotifiedAt = time.Now()
		expiresAt := time.Now().Add(s.config.BookingWindowDuration)
		entry.ExpiresAt = &expiresAt

		err = s.repo.UpdateEntry(ctx, &entry)
		if err != nil {
			log.Printf("Failed to update entry %s: %v", entry.ID, err)
			continue
		}

		// Send notification
		log.Printf("📧 SENDING: Notification to user %s (position %d) for event %s - expires at %s",
			entry.UserID, entry.Position, eventID, expiresAt.Format("15:04:05"))

		err = s.sendSpotAvailableNotification(ctx, &entry)
		if err != nil {
			log.Printf("❌ NOTIFICATION FAILED: User %s for event %s - Error: %v", entry.UserID, eventID, err)
		} else {
			log.Printf("✅ NOTIFICATION SENT: User %s for event %s via Kafka", entry.UserID, eventID)
		}

		notifiedUsers = append(notifiedUsers, entry.UserID)
	}

	log.Printf("🎉 WAITLIST COMPLETE: Notified %d users from waitlist for event %s", len(notifiedUsers), eventID)

	return nil
}

// NotifyNextInLine notifies the next users in line when tickets become available
func (s *service) NotifyNextInLine(ctx context.Context, eventID uuid.UUID, availableTickets int) error {
	return s.ProcessCancellation(ctx, eventID, availableTickets)
}

// ProcessBookingExpiry handles expired booking windows
func (s *service) ProcessBookingExpiry(ctx context.Context, userID, eventID uuid.UUID) error {
	entry, err := s.repo.GetEntry(ctx, userID, eventID)
	if err != nil {
		return fmt.Errorf("entry not found: %w", err)
	}

	if entry.Status != WaitlistStatusNotified {
		return fmt.Errorf("entry is not in notified status")
	}

	// Update status to expired
	entry.Status = WaitlistStatusExpired
	err = s.repo.UpdateEntry(ctx, entry)
	if err != nil {
		return fmt.Errorf("failed to update entry: %w", err)
	}

	// Remove from Redis queue
	err = s.repo.RemoveFromQueue(ctx, userID, eventID)
	if err != nil {
		log.Printf("Failed to remove expired user from queue: %v", err)
	}

	log.Printf("Booking window expired for user %s, event %s", userID, eventID)

	// Notify next user in line
	go func() {
		if err := s.NotifyNextInLine(context.Background(), eventID, entry.Quantity); err != nil {
			log.Printf("Failed to notify next in line: %v", err)
		}
	}()

	return nil
}

// sendSpotAvailableNotification sends a spot available notification
func (s *service) sendSpotAvailableNotification(ctx context.Context, entry *WaitlistEntry) error {
	// Get real user details from user service
	userEmail, firstName, lastName, err := s.userService.GetUserByID(ctx, entry.UserID)
	if err != nil {
		log.Printf("❌ USER FETCH ERROR: Failed to get user details for %s: %v", entry.UserID, err)
		return fmt.Errorf("failed to get user details: %w", err)
	}

	userName := firstName
	if lastName != "" {
		userName = firstName + " " + lastName
	}
	if userName == "" {
		userName = "User" // Fallback if no name is available
	}

	// Prepare template data
	templateData := map[string]interface{}{
		"event_id":       entry.EventID.String(),
		"position":       entry.Position,
		"quantity":       entry.Quantity,
		"expires_at":     entry.ExpiresAt,
		"event_title":    "Event Title", // TODO: Fetch from event service
		"venue_name":     "Venue Name",  // TODO: Fetch from venue service
		"booking_window": s.config.BookingWindowDuration.Minutes(),
	}

	// Send via unified notification service
	log.Printf("� UNIFIED: Sending spot available notification to user %s for event %s", entry.UserID, entry.EventID)
	notificationErr := s.notificationService.SendWaitlistNotification(ctx,
		entry.UserID,
		userEmail,
		userName,
		entry.EventID,
		entry.ID,
		"WAITLIST_SPOT_AVAILABLE", // Notification type string
		templateData,
	)
	if notificationErr != nil {
		log.Printf("❌ NOTIFICATION FAILED: Could not send notification for user %s: %v", entry.UserID, notificationErr)
		return fmt.Errorf("failed to send notification: %w", notificationErr)
	}
	log.Printf("✅ NOTIFICATION SUCCESS: Spot available notification sent for user %s", entry.UserID)

	// Create notification record
	notificationRecord := &WaitlistNotification{
		WaitlistEntryID:  entry.ID,
		NotificationType: NotificationTypeSpotAvailable,
		Channel:          NotificationChannelEmail,
		Status:           NotificationStatusPending,
	}

	err = s.repo.CreateNotification(ctx, notificationRecord)
	if err != nil {
		log.Printf("⚠️ DB WARNING: Failed to create notification record for user %s: %v", entry.UserID, err)
	} else {
		log.Printf("💾 DB SUCCESS: Notification record created for user %s", entry.UserID)
	}

	return nil
}

// NotifyPositionUpdate sends position updates to all users in waitlist
func (s *service) NotifyPositionUpdate(ctx context.Context, eventID uuid.UUID) error {
	entries, err := s.repo.ListEntries(ctx, eventID, WaitlistStatusActive)
	if err != nil {
		return fmt.Errorf("failed to get waitlist entries: %w", err)
	}

	if len(entries) == 0 {
		return nil
	}

	log.Printf("📊 POSITION UPDATE: Sending position updates to %d users for event %s", len(entries), eventID)

	// Send individual notifications via unified service
	for _, entry := range entries {
		// Get real user details from user service
		userEmail, firstName, lastName, err := s.userService.GetUserByID(ctx, entry.UserID)
		if err != nil {
			log.Printf("❌ USER FETCH ERROR: Failed to get user details for %s: %v", entry.UserID, err)
			continue // Skip this notification but continue with others
		}

		userName := firstName
		if lastName != "" {
			userName = firstName + " " + lastName
		}
		if userName == "" {
			userName = "User" // Fallback if no name is available
		}

		templateData := map[string]interface{}{
			"event_id":    entry.EventID.String(),
			"position":    entry.Position,
			"quantity":    entry.Quantity,
			"event_title": "Event Title", // TODO: Fetch from event service
			"venue_name":  "Venue Name",  // TODO: Fetch from venue service
		}

		notificationErr := s.notificationService.SendWaitlistNotification(ctx,
			entry.UserID,
			userEmail,
			userName,
			entry.EventID,
			entry.ID,
			"WAITLIST_POSITION_UPDATE", // Notification type string
			templateData,
		)
		if notificationErr != nil {
			log.Printf("❌ Position update failed for user %s: %v", entry.UserID, notificationErr)
			continue // Continue with other notifications even if one fails
		}
	}

	log.Printf("✅ POSITION UPDATE: Completed sending position updates for event %s", eventID)
	return nil
}

// GetWaitlistStats gets statistics for a waitlist
func (s *service) GetWaitlistStats(ctx context.Context, eventID uuid.UUID) (*WaitlistStatsResponse, error) {
	return s.repo.GetWaitlistStats(ctx, eventID)
}

// GetWaitlistEntries gets waitlist entries for an event
func (s *service) GetWaitlistEntries(ctx context.Context, eventID uuid.UUID, status WaitlistStatus) ([]WaitlistEntry, error) {
	return s.repo.ListEntries(ctx, eventID, status)
}

// ProcessExpiredBookingWindows processes all expired booking windows
func (s *service) ProcessExpiredBookingWindows(ctx context.Context) (int, error) {
	expiredEntries, err := s.repo.GetExpiredEntries(ctx, 100) // Process 100 at a time
	if err != nil {
		return 0, fmt.Errorf("failed to get expired entries: %w", err)
	}

	if len(expiredEntries) == 0 {
		return 0, nil
	}

	// Re-queue expired users instead of removing them permanently
	eventTickets := make(map[uuid.UUID]int)
	var requeuedUsers []uuid.UUID

	for _, entry := range expiredEntries {
		// Requeue the user at the end of the waitlist
		err := s.repo.RequeueExpiredUser(ctx, entry.UserID, entry.EventID)
		if err != nil {
			log.Printf("Failed to requeue expired user %s for event %s: %v", entry.UserID, entry.EventID, err)
			continue
		}

		log.Printf("🔄 RE-QUEUED: User %s moved back to end of waitlist for event %s (missed booking window)",
			entry.UserID, entry.EventID)

		requeuedUsers = append(requeuedUsers, entry.UserID)
		eventTickets[entry.EventID] += entry.Quantity
	}

	// Notify next users for each event (the tickets are still available)
	for eventID, freedTickets := range eventTickets {
		go func(eID uuid.UUID, tickets int) {
			if err := s.NotifyNextInLine(context.Background(), eID, tickets); err != nil {
				log.Printf("Failed to notify next in line for event %s: %v", eID, err)
			}
		}(eventID, freedTickets)
	}

	log.Printf("✅ REQUEUE COMPLETE: Re-queued %d expired users instead of removing them", len(requeuedUsers))
	return len(expiredEntries), nil
}

// UpdateDailyAnalytics updates daily analytics for all events
func (s *service) UpdateDailyAnalytics(ctx context.Context) error {
	// This would typically query aggregated data and update analytics tables
	// For now, we'll just log that the function was called
	log.Println("Updating daily waitlist analytics...")

	// TODO: Implement actual analytics aggregation
	// This would involve:
	// 1. Querying waitlist entries by date
	// 2. Calculating metrics (conversions, wait times, etc.)
	// 3. Storing in analytics tables

	return nil
}

// validateJoinRequest validates a join waitlist request
func (s *service) validateJoinRequest(request *JoinWaitlistRequest) error {
	if request.EventID == uuid.Nil {
		return fmt.Errorf("event ID is required")
	}

	if request.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}

	if request.Quantity > s.config.MaxQuantityPerUser {
		return fmt.Errorf("quantity exceeds maximum allowed (%d)", s.config.MaxQuantityPerUser)
	}

	return nil
}

// MarkAsConverted marks a waitlist entry as converted after successful booking
func (s *service) MarkAsConverted(ctx context.Context, userID, eventID, bookingID uuid.UUID) error {
	log.Printf("🔄 MARK AS CONVERTED: Starting conversion for user %s, event %s, booking %s", userID, eventID, bookingID)

	// Get the waitlist entry
	entry, err := s.repo.GetEntry(ctx, userID, eventID)
	if err != nil {
		// No waitlist entry found - user wasn't on waitlist, which is fine
		log.Printf("ℹ️  MARK AS CONVERTED: No waitlist entry found for user %s, event %s (user wasn't on waitlist)", userID, eventID)
		return nil
	}

	log.Printf("📊 MARK AS CONVERTED: Found waitlist entry with status %s for user %s, event %s", entry.Status, userID, eventID)

	// Only update if user was notified (allowing conversion)
	if entry.Status != WaitlistStatusNotified {
		// User wasn't in notified status, no need to update
		log.Printf("⚠️  MARK AS CONVERTED: User %s is in status %s, not NOTIFIED - skipping conversion", userID, entry.Status)
		return nil
	}

	log.Printf("📝 MARK AS CONVERTED: Updating database status to CONVERTED for user %s", userID)
	// Update status to converted
	entry.Status = WaitlistStatusConverted
	err = s.repo.UpdateEntry(ctx, entry)
	if err != nil {
		log.Printf("❌ MARK AS CONVERTED: Database update failed for user %s: %v", userID, err)
		return fmt.Errorf("failed to mark waitlist entry as converted: %w", err)
	}
	log.Printf("✅ MARK AS CONVERTED: Database status updated to CONVERTED for user %s", userID)

	log.Printf("🗑️  MARK AS CONVERTED: Removing user %s from Redis queue for event %s", userID, eventID)
	// Remove from Redis queue since they've successfully booked
	err = s.repo.RemoveFromQueue(ctx, userID, eventID)
	if err != nil {
		log.Printf("❌ MARK AS CONVERTED: Failed to remove user %s from Redis queue: %v", userID, err)
		// Don't return error as the main goal (marking as converted) succeeded
	} else {
		log.Printf("✅ MARK AS CONVERTED: Successfully removed user %s from Redis queue", userID)
	}

	log.Printf("✅ WAITLIST CONVERTED: User %s successfully booked from waitlist for event %s (booking %s)",
		userID, eventID, bookingID)

	return nil
}

// GetWaitlistStatusForBooking returns simplified waitlist status for booking validation
func (s *service) GetWaitlistStatusForBooking(ctx context.Context, userID, eventID uuid.UUID) (*WaitlistStatusForBooking, error) {
	entry, err := s.repo.GetEntry(ctx, userID, eventID)
	if err != nil {
		// No waitlist entry found
		return nil, nil
	}

	return &WaitlistStatusForBooking{
		Status:    string(entry.Status),
		IsExpired: entry.IsExpired(),
	}, nil
}

// WaitlistStatusForBooking represents simplified waitlist status for booking service
type WaitlistStatusForBooking struct {
	Status    string `json:"status"`
	IsExpired bool   `json:"is_expired"`
}
