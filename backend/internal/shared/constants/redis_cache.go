package constants

import (
	"fmt"
	"time"
)

// Redis Cache Configuration
// This file centralizes all Redis cache keys and TTL values for the Evently application
// Pattern: evently:{module}:{operation}:{identifier}:{params?}

// ================== CACHE TTL DURATIONS ==================

// Static Data (Long TTL: rarely changes)
const (
	TTL_STATIC_LONG   = 24 * time.Hour // 24 hours - for very stable data
	TTL_STATIC_MEDIUM = 12 * time.Hour // 12 hours - for architectural data
	TTL_STATIC_SHORT  = 6 * time.Hour  // 6 hours - for user profiles
)

// Semi-Static Data (Medium TTL: changes occasionally)
const (
	TTL_SEMI_STATIC_LONG   = 4 * time.Hour    // 4 hours - for venue layouts
	TTL_SEMI_STATIC_MEDIUM = 2 * time.Hour    // 2 hours - for event details
	TTL_SEMI_STATIC_SHORT  = 1 * time.Hour    // 1 hour - for event listings
	TTL_SEMI_STATIC_QUICK  = 15 * time.Minute // 15 minutes - for upcoming events
)

// Dynamic Data (Short TTL: changes frequently)
const (
	TTL_DYNAMIC_MEDIUM = 10 * time.Minute // 10 minutes - for analytics
	TTL_DYNAMIC_SHORT  = 5 * time.Minute  // 5 minutes - for seat availability
	TTL_DYNAMIC_QUICK  = 2 * time.Minute  // 2 minutes - for booking availability
)

// Highly Dynamic (Micro TTL: real-time sensitive)
const (
	TTL_REALTIME_MEDIUM = 1 * time.Minute  // 1 minute - for waitlist positions
	TTL_REALTIME_SHORT  = 30 * time.Second // 30 seconds - for live seat counts
)

// ================== REDIS KEY PREFIXES ==================

const (
	CACHE_PREFIX = "evently"
)

// ================== EVENTS MODULE ==================

// Event Cache Keys
const (
	// Event listings and searches
	CACHE_KEY_EVENTS_LIST     = CACHE_PREFIX + ":events:list"     // + :page:X:limit:Y:status:Z
	CACHE_KEY_EVENTS_UPCOMING = CACHE_PREFIX + ":events:upcoming" // + :page:X:limit:Y
	CACHE_KEY_EVENTS_BY_TAG   = CACHE_PREFIX + ":events:by_tag"   // + :slug:X:page:Y
	CACHE_KEY_EVENTS_SEARCH   = CACHE_PREFIX + ":events:search"   // + :query:X:page:Y

	// Individual event details
	CACHE_KEY_EVENT_DETAIL      = CACHE_PREFIX + ":events:detail:uuid:"      // + event-id
	CACHE_KEY_EVENT_WITH_TAGS   = CACHE_PREFIX + ":events:with_tags:uuid:"   // + event-id
	CACHE_KEY_EVENT_FULL_DETAIL = CACHE_PREFIX + ":events:full_detail:uuid:" // + event-id (with venue info)
)

// Event Cache TTLs
const (
	TTL_EVENT_LIST     = TTL_SEMI_STATIC_SHORT  // 1 hour
	TTL_EVENT_UPCOMING = TTL_SEMI_STATIC_QUICK  // 15 minutes
	TTL_EVENT_DETAIL   = TTL_SEMI_STATIC_MEDIUM // 2 hours
	TTL_EVENT_SEARCH   = TTL_SEMI_STATIC_QUICK  // 15 minutes
)

// ================== TAGS MODULE ==================

// Tag Cache Keys
const (
	CACHE_KEY_TAGS_ACTIVE   = CACHE_PREFIX + ":tags:active:all"     // Active tags list
	CACHE_KEY_TAGS_ALL      = CACHE_PREFIX + ":tags:list"           // + :page:X:limit:Y:status:Z
	CACHE_KEY_TAG_BY_SLUG   = CACHE_PREFIX + ":tags:detail:slug:"   // + tag-slug
	CACHE_KEY_TAG_BY_ID     = CACHE_PREFIX + ":tags:detail:uuid:"   // + tag-id
	CACHE_KEY_TAGS_BY_EVENT = CACHE_PREFIX + ":tags:by_event:uuid:" // + event-id
)

// Tag Cache TTLs
const (
	TTL_TAGS_ACTIVE = TTL_STATIC_LONG  // 24 hours
	TTL_TAGS_LIST   = TTL_STATIC_SHORT // 6 hours
	TTL_TAG_DETAIL  = TTL_STATIC_LONG  // 24 hours
)

// ================== VENUES MODULE ==================

// Venue Cache Keys
const (
	// Venue templates
	CACHE_KEY_VENUE_TEMPLATES = CACHE_PREFIX + ":venues:templates"      // + :page:X:limit:Y
	CACHE_KEY_VENUE_TEMPLATE  = CACHE_PREFIX + ":venues:template:uuid:" // + template-id

	// Venue sections
	CACHE_KEY_VENUE_SECTIONS = CACHE_PREFIX + ":venues:sections:template:" // + template-id
	CACHE_KEY_EVENT_SECTIONS = CACHE_PREFIX + ":venues:sections:event:"    // + event-id

	// Venue layouts (complex data)
	CACHE_KEY_VENUE_LAYOUT   = CACHE_PREFIX + ":venues:layout:event:" // + event-id
	CACHE_KEY_SECTION_DETAIL = CACHE_PREFIX + ":venues:section:uuid:" // + section-id
)

// Venue Cache TTLs
const (
	TTL_VENUE_TEMPLATES = TTL_STATIC_MEDIUM    // 12 hours
	TTL_VENUE_TEMPLATE  = TTL_STATIC_MEDIUM    // 12 hours
	TTL_VENUE_SECTIONS  = TTL_STATIC_MEDIUM    // 12 hours
	TTL_VENUE_LAYOUT    = TTL_SEMI_STATIC_LONG // 4 hours
)

// ================== SEATS MODULE ==================

// Seat Cache Keys
const (
	// Seat listings
	CACHE_KEY_SEATS_BY_SECTION = CACHE_PREFIX + ":seats:section:uuid:"      // + section-id:all
	CACHE_KEY_SEATS_AVAILABLE  = CACHE_PREFIX + ":seats:available:section:" // + section-id:event:event-id

	// Seat details
	CACHE_KEY_SEAT_DETAIL = CACHE_PREFIX + ":seats:detail:uuid:" // + seat-id

	// Availability checks (short-lived)
	CACHE_KEY_SEAT_AVAILABILITY = CACHE_PREFIX + ":seats:availability:" // + event-id:section-id
	CACHE_KEY_BULK_AVAILABILITY = CACHE_PREFIX + ":seats:bulk_check:"   // + event-id:hash
)

// Seat Cache TTLs
const (
	TTL_SEATS_BY_SECTION  = TTL_DYNAMIC_SHORT  // 5 minutes
	TTL_SEATS_AVAILABLE   = TTL_DYNAMIC_QUICK  // 2 minutes
	TTL_SEAT_DETAIL       = TTL_DYNAMIC_MEDIUM // 10 minutes
	TTL_SEAT_AVAILABILITY = TTL_REALTIME_SHORT // 30 seconds
)

// ================== ANALYTICS MODULE ==================

// Analytics Cache Keys
const (
	// Dashboard analytics
	CACHE_KEY_ANALYTICS_DASHBOARD     = CACHE_PREFIX + ":analytics:dashboard:admin"
	CACHE_KEY_ANALYTICS_USER_PERSONAL = CACHE_PREFIX + ":analytics:user:personal:uuid:" // + user-id

	// Event analytics
	CACHE_KEY_ANALYTICS_EVENT_GLOBAL = CACHE_PREFIX + ":analytics:events:global"
	CACHE_KEY_ANALYTICS_EVENT_DETAIL = CACHE_PREFIX + ":analytics:event:uuid:" // + event-id

	// Tag analytics
	CACHE_KEY_ANALYTICS_TAGS           = CACHE_PREFIX + ":analytics:tags:overview"
	CACHE_KEY_ANALYTICS_TAG_POPULARITY = CACHE_PREFIX + ":analytics:tags:popularity"
	CACHE_KEY_ANALYTICS_TAG_TRENDS     = CACHE_PREFIX + ":analytics:tags:trends:months:" // + month-count
	CACHE_KEY_ANALYTICS_TAG_COMPARE    = CACHE_PREFIX + ":analytics:tags:comparisons"

	// Booking analytics
	CACHE_KEY_ANALYTICS_BOOKINGS      = CACHE_PREFIX + ":analytics:bookings:overview"
	CACHE_KEY_ANALYTICS_BOOKING_DAILY = CACHE_PREFIX + ":analytics:bookings:daily"
	CACHE_KEY_ANALYTICS_CANCELLATION  = CACHE_PREFIX + ":analytics:cancellations"

	// User analytics
	CACHE_KEY_ANALYTICS_USERS             = CACHE_PREFIX + ":analytics:users:overview"
	CACHE_KEY_ANALYTICS_USER_RETENTION    = CACHE_PREFIX + ":analytics:users:retention"
	CACHE_KEY_ANALYTICS_USER_DEMOGRAPHICS = CACHE_PREFIX + ":analytics:users:demographics"
)

// Analytics Cache TTLs
const (
	TTL_ANALYTICS_DASHBOARD = TTL_SEMI_STATIC_SHORT // 1 hour
	TTL_ANALYTICS_EVENT     = TTL_SEMI_STATIC_SHORT // 1 hour
	TTL_ANALYTICS_TAGS      = TTL_DYNAMIC_MEDIUM    // 10 minutes
	TTL_ANALYTICS_BOOKINGS  = TTL_DYNAMIC_MEDIUM    // 10 minutes
	TTL_ANALYTICS_USERS     = TTL_SEMI_STATIC_SHORT // 1 hour
	TTL_ANALYTICS_PERSONAL  = TTL_SEMI_STATIC_SHORT // 1 hour
)

// ================== AUTH MODULE ==================

// Auth Cache Keys
const (
	CACHE_KEY_USER_PROFILE = CACHE_PREFIX + ":auth:user:profile:uuid:" // + user-id
	CACHE_KEY_USER_ROLES   = CACHE_PREFIX + ":auth:user:roles:uuid:"   // + user-id
)

// Auth Cache TTLs
const (
	TTL_USER_PROFILE = TTL_STATIC_SHORT // 6 hours
	TTL_USER_ROLES   = TTL_STATIC_SHORT // 6 hours
)

// ================== BOOKINGS MODULE ==================

// Booking Cache Keys
const (
	CACHE_KEY_USER_BOOKINGS   = CACHE_PREFIX + ":bookings:user:uuid:"    // + user-id:page:X
	CACHE_KEY_BOOKING_DETAIL  = CACHE_PREFIX + ":bookings:detail:uuid:"  // + booking-id
	CACHE_KEY_BOOKING_HISTORY = CACHE_PREFIX + ":bookings:history:user:" // + user-id
)

// Booking Cache TTLs
const (
	TTL_USER_BOOKINGS  = TTL_DYNAMIC_MEDIUM // 10 minutes
	TTL_BOOKING_DETAIL = TTL_DYNAMIC_MEDIUM // 10 minutes
)

// ================== WAITLIST MODULE ==================

// Waitlist Cache Keys
const (
	CACHE_KEY_WAITLIST_STATUS   = CACHE_PREFIX + ":waitlist:status:event:"   // + event-id:user:user-id
	CACHE_KEY_WAITLIST_STATS    = CACHE_PREFIX + ":waitlist:stats:event:"    // + event-id
	CACHE_KEY_WAITLIST_ENTRIES  = CACHE_PREFIX + ":waitlist:entries:event:"  // + event-id
	CACHE_KEY_WAITLIST_POSITION = CACHE_PREFIX + ":waitlist:position:event:" // + event-id:user:user-id
)

// Waitlist Cache TTLs
const (
	TTL_WAITLIST_STATUS   = TTL_REALTIME_MEDIUM // 1 minute
	TTL_WAITLIST_STATS    = TTL_DYNAMIC_SHORT   // 5 minutes
	TTL_WAITLIST_POSITION = TTL_REALTIME_MEDIUM // 1 minute
)

// ================== CANCELLATION MODULE ==================

// Cancellation Cache Keys
const (
	CACHE_KEY_CANCELLATION_POLICY = CACHE_PREFIX + ":cancellation:policy:event:" // + event-id
	CACHE_KEY_USER_CANCELLATIONS  = CACHE_PREFIX + ":cancellation:user:uuid:"    // + user-id
	CACHE_KEY_CANCELLATION_DETAIL = CACHE_PREFIX + ":cancellation:detail:uuid:"  // + cancellation-id
)

// Cancellation Cache TTLs
const (
	TTL_CANCELLATION_POLICY = TTL_SEMI_STATIC_MEDIUM // 2 hours
	TTL_USER_CANCELLATIONS  = TTL_DYNAMIC_MEDIUM     // 10 minutes
	TTL_CANCELLATION_DETAIL = TTL_DYNAMIC_MEDIUM     // 10 minutes
)

// ================== CACHE INVALIDATION PATTERNS ==================

// Patterns for cache invalidation (used with Redis KEYS command or manual invalidation)
const (
	// Event-related invalidation patterns
	PATTERN_INVALIDATE_EVENT_ALL    = CACHE_PREFIX + ":events:*"
	PATTERN_INVALIDATE_EVENT_DETAIL = CACHE_PREFIX + ":events:*:uuid:" // + event-id + *

	// Tag-related invalidation patterns
	PATTERN_INVALIDATE_TAGS_ALL = CACHE_PREFIX + ":tags:*"

	// Venue-related invalidation patterns
	PATTERN_INVALIDATE_VENUES_ALL = CACHE_PREFIX + ":venues:*"

	// User-related invalidation patterns
	PATTERN_INVALIDATE_USER_ALL = CACHE_PREFIX + ":*:user:*" // + user-id + *

	// Analytics invalidation patterns
	PATTERN_INVALIDATE_ANALYTICS = CACHE_PREFIX + ":analytics:*"
)

// ================== HELPER FUNCTIONS ==================

// BuildCacheKey constructs cache keys with parameters
// Example: BuildEventListKey(page, limit, status) -> "evently:events:list:page:1:limit:10:status:active"
func BuildEventListKey(page, limit int, status string) string {
	if status != "" {
		return CACHE_KEY_EVENTS_LIST + ":page:" + fmt.Sprintf("%d", page) + ":limit:" + fmt.Sprintf("%d", limit) + ":status:" + status
	}
	return CACHE_KEY_EVENTS_LIST + ":page:" + fmt.Sprintf("%d", page) + ":limit:" + fmt.Sprintf("%d", limit)
}

func BuildEventDetailKey(eventID string) string {
	return CACHE_KEY_EVENT_DETAIL + eventID
}

func BuildTagBySlugKey(slug string) string {
	return CACHE_KEY_TAG_BY_SLUG + slug
}

func BuildVenueLayoutKey(eventID string) string {
	return CACHE_KEY_VENUE_LAYOUT + eventID
}

func BuildUserBookingsKey(userID string, page int) string {
	return CACHE_KEY_USER_BOOKINGS + userID + ":page:" + fmt.Sprintf("%d", page)
}

func BuildSeatAvailabilityKey(sectionID, eventID string) string {
	return CACHE_KEY_SEATS_AVAILABLE + sectionID + ":event:" + eventID
}

func BuildAnalyticsEventKey(eventID string) string {
	return CACHE_KEY_ANALYTICS_EVENT_DETAIL + eventID
}

func BuildWaitlistStatusKey(eventID, userID string) string {
	return CACHE_KEY_WAITLIST_STATUS + eventID + ":user:" + userID
}

// ================== USAGE EXAMPLES ==================

/*
IMPLEMENTATION EXAMPLES:

1. Cache Event Details:
   key := BuildEventDetailKey(eventID)
   ttl := TTL_EVENT_DETAIL

2. Cache Active Tags:
   key := CACHE_KEY_TAGS_ACTIVE
   ttl := TTL_TAGS_ACTIVE

3. Cache Seat Availability:
   key := BuildSeatAvailabilityKey(sectionID, eventID)
   ttl := TTL_SEATS_AVAILABLE

4. Cache User Bookings:
   key := BuildUserBookingsKey(userID, page)
   ttl := TTL_USER_BOOKINGS

INVALIDATION EXAMPLES:

1. When event is updated:
   - Invalidate: evently:events:*:uuid:eventID*
   - Invalidate: evently:analytics:event:uuid:eventID

2. When tag is updated:
   - Invalidate: evently:tags:*
   - Invalidate: evently:events:* (if events cache tags)

3. When seat is booked:
   - Invalidate: evently:seats:available:*
   - Invalidate: evently:seats:section:*
*/
