package notifications

import (
	"context"

	"github.com/google/uuid"
)

// WaitlistServiceAdapter implements the waitlist.NotificationService interface
// and adapts calls to the unified notification system
type WaitlistServiceAdapter struct {
	unifiedService NotificationService
}

// NewWaitlistServiceAdapter creates a new adapter for waitlist notifications
func NewWaitlistServiceAdapter(unifiedService NotificationService) *WaitlistServiceAdapter {
	return &WaitlistServiceAdapter{
		unifiedService: unifiedService,
	}
}

// SendWaitlistNotification implements the waitlist.NotificationService interface
func (w *WaitlistServiceAdapter) SendWaitlistNotification(ctx context.Context, userID uuid.UUID, email, name string,
	eventID, waitlistEntryID uuid.UUID, notificationType string,
	templateData map[string]interface{}) error {

	// Map string notification types to unified types
	var unifiedType NotificationType
	switch notificationType {
	case "WAITLIST_SPOT_AVAILABLE":
		unifiedType = NotificationTypeWaitlistSpotAvailable
	case "WAITLIST_POSITION_UPDATE":
		unifiedType = NotificationTypeWaitlistPositionUpdate
	case "WAITLIST_REMINDER":
		unifiedType = NotificationTypeWaitlistReminder
	case "WAITLIST_EXPIRED":
		unifiedType = NotificationTypeWaitlistExpired
	default:
		unifiedType = NotificationTypeWaitlistSpotAvailable
	}

	// Use the unified notification service's waitlist method
	return w.unifiedService.SendWaitlistNotification(ctx, userID, email, name, eventID, waitlistEntryID, unifiedType, templateData)
}

// GetUnifiedService returns the underlying unified notification service
// This can be useful for accessing other notification methods
func (w *WaitlistServiceAdapter) GetUnifiedService() NotificationService {
	return w.unifiedService
}
