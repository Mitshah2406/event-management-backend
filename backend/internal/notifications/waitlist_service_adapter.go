package notifications

import (
	"context"

	"github.com/google/uuid"
)

type WaitlistServiceAdapter struct {
	unifiedService NotificationService
}

func NewWaitlistServiceAdapter(unifiedService NotificationService) *WaitlistServiceAdapter {
	return &WaitlistServiceAdapter{
		unifiedService: unifiedService,
	}
}

func (w *WaitlistServiceAdapter) SendWaitlistNotification(ctx context.Context, userID uuid.UUID, email, name string,
	eventID, waitlistEntryID uuid.UUID, notificationType string,
	templateData map[string]interface{}) error {

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

	return w.unifiedService.SendWaitlistNotification(ctx, userID, email, name, eventID, waitlistEntryID, unifiedType, templateData)
}

func (w *WaitlistServiceAdapter) GetUnifiedService() NotificationService {
	return w.unifiedService
}
