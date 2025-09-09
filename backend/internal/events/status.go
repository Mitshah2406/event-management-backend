package events

type EventStatus string

const (
	EventStatusDraft     EventStatus = "draft"
	EventStatusPublished EventStatus = "published"
	EventStatusCancelled EventStatus = "cancelled"
	EventStatusCompleted EventStatus = "completed"
)

// IsValid checks if the event status is valid
func (es EventStatus) IsValid() bool {
	switch es {
	case EventStatusDraft, EventStatusPublished, EventStatusCancelled, EventStatusCompleted:
		return true
	}
	return false
}

// String returns the string representation of EventStatus
func (es EventStatus) String() string {
	return string(es)
}

// CanBeUpdated checks if an event with this status can be updated
func (es EventStatus) CanBeUpdated() bool {
	return es == EventStatusDraft || es == EventStatusPublished
}

// CanBeDeleted checks if an event with this status can be deleted
func (es EventStatus) CanBeDeleted() bool {
	return es == EventStatusDraft
}

// CanBeBooked checks if an event with this status allows new bookings
func (es EventStatus) CanBeBooked() bool {
	return es == EventStatusPublished
}
