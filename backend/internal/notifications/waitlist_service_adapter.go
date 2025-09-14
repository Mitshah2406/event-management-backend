package notifications

import (
	"context"

	"github.com/google/uuid"
)

// Simplified adapter for waitlist service integration
type WaitlistServiceAdapter struct {
	emailService NotificationService
}

func NewWaitlistServiceAdapter(emailService NotificationService) *WaitlistServiceAdapter {
	return &WaitlistServiceAdapter{
		emailService: emailService,
	}
}

func (w *WaitlistServiceAdapter) SendWaitlistNotification(ctx context.Context, userID uuid.UUID, email, name string,
	eventID, waitlistEntryID uuid.UUID, notificationType string,
	templateData map[string]interface{}) error {

	// Convert string notification type to enum
	var unifiedType NotificationType
	switch notificationType {
	case "WAITLIST_SPOT_AVAILABLE":
		unifiedType = NotificationTypeWaitlistSpotAvailable
	case "WAITLIST_POSITION_UPDATE":
		unifiedType = NotificationTypeWaitlistPositionUpdate
	default:
		unifiedType = NotificationTypeWaitlistSpotAvailable
	}

	return w.emailService.SendWaitlistNotification(ctx, userID, email, name, eventID, waitlistEntryID, unifiedType, templateData)
}

func (w *WaitlistServiceAdapter) GetEmailService() NotificationService {
	return w.emailService
}
