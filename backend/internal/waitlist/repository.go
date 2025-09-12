package waitlist

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Repository interface defines the contract for waitlist data operations
type Repository interface {
	// Waitlist Queue Operations
	AddToQueue(ctx context.Context, entry *WaitlistEntry) error
	RemoveFromQueue(ctx context.Context, userID, eventID uuid.UUID) error
	GetPosition(ctx context.Context, userID, eventID uuid.UUID) (int, error)
	GetQueueLength(ctx context.Context, eventID uuid.UUID) (int, error)
	GetNextInQueue(ctx context.Context, eventID uuid.UUID, count int) ([]WaitlistEntry, error)
	UpdatePositions(ctx context.Context, eventID uuid.UUID) error

	// Database Operations
	CreateEntry(ctx context.Context, entry *WaitlistEntry) error
	UpdateEntry(ctx context.Context, entry *WaitlistEntry) error
	GetEntry(ctx context.Context, userID, eventID uuid.UUID) (*WaitlistEntry, error)
	GetEntryByID(ctx context.Context, id uuid.UUID) (*WaitlistEntry, error)
	ListEntries(ctx context.Context, eventID uuid.UUID, status WaitlistStatus) ([]WaitlistEntry, error)
	DeleteEntry(ctx context.Context, id uuid.UUID) error

	// Batch Operations
	GetExpiredEntries(ctx context.Context, limit int) ([]WaitlistEntry, error)
	UpdateEntriesStatus(ctx context.Context, ids []uuid.UUID, status WaitlistStatus) error

	// Analytics
	GetWaitlistStats(ctx context.Context, eventID uuid.UUID) (*WaitlistStatsResponse, error)
	CreateAnalytics(ctx context.Context, analytics *WaitlistAnalytics) error

	// Notifications
	CreateNotification(ctx context.Context, notification *WaitlistNotification) error
	UpdateNotification(ctx context.Context, notification *WaitlistNotification) error
	GetPendingNotifications(ctx context.Context, limit int) ([]WaitlistNotification, error)
}

// repository implements the Repository interface
type repository struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewRepository creates a new waitlist repository
func NewRepository(db *gorm.DB, redisClient *redis.Client) Repository {
	return &repository{
		db:    db,
		redis: redisClient,
	}
}

// AddToQueue adds a user to the event waitlist queue using Redis ZSet
func (r *repository) AddToQueue(ctx context.Context, entry *WaitlistEntry) error {
	queueKey := GetQueueKey(entry.EventID)
	lockKey := GetLockKey(entry.EventID)

	// Acquire distributed lock for queue operations
	lock := r.redis.SetNX(ctx, lockKey, "locked", 5*time.Second)
	if !lock.Val() {
		return fmt.Errorf("could not acquire lock for event %s", entry.EventID)
	}
	defer r.redis.Del(ctx, lockKey)

	// Get current queue length to determine position
	queueLength := r.redis.ZCard(ctx, queueKey).Val()

	// Check if user is already in queue
	userKey := entry.UserID.String()
	score := r.redis.ZScore(ctx, queueKey, userKey).Val()
	if score != 0 { // User already exists in queue
		return fmt.Errorf("user already in waitlist for event %s", entry.EventID)
	}

	// Add user to queue with position as score
	position := int(queueLength) + 1
	entry.Position = position

	err := r.redis.ZAdd(ctx, queueKey, redis.Z{
		Score:  float64(position),
		Member: userKey,
	}).Err()
	if err != nil {
		return fmt.Errorf("failed to add user to queue: %w", err)
	}

	// Set TTL for the queue key
	r.redis.Expire(ctx, queueKey, RedisKeyTTL)

	// Update position tracking
	positionKey := GetPositionKey(entry.EventID)
	r.redis.HSet(ctx, positionKey, userKey, position)
	r.redis.Expire(ctx, positionKey, RedisKeyTTL)

	return nil
}

// RemoveFromQueue removes a user from the waitlist queue
func (r *repository) RemoveFromQueue(ctx context.Context, userID, eventID uuid.UUID) error {
	queueKey := GetQueueKey(eventID)
	lockKey := GetLockKey(eventID)

	// Acquire distributed lock
	lock := r.redis.SetNX(ctx, lockKey, "locked", 5*time.Second)
	if !lock.Val() {
		return fmt.Errorf("could not acquire lock for event %s", eventID)
	}
	defer r.redis.Del(ctx, lockKey)

	userKey := userID.String()

	// Get user's current position
	position := r.redis.ZScore(ctx, queueKey, userKey).Val()
	if position == 0 {
		return fmt.Errorf("user not found in waitlist for event %s", eventID)
	}

	// Remove user from queue
	err := r.redis.ZRem(ctx, queueKey, userKey).Err()
	if err != nil {
		return fmt.Errorf("failed to remove user from queue: %w", err)
	}

	// Update positions of users behind the removed user
	err = r.redis.ZIncrBy(ctx, queueKey, -1, userKey).Err()
	if err == nil { // If the operation succeeded, adjust all positions
		// Get all members with score greater than the removed user's position
		members := r.redis.ZRangeByScore(ctx, queueKey, &redis.ZRangeBy{
			Min: fmt.Sprintf("(%f", position),
			Max: "+inf",
		}).Val()

		// Decrement each position by 1
		for _, member := range members {
			r.redis.ZIncrBy(ctx, queueKey, -1, member)
		}
	}

	// Remove from position tracking
	positionKey := GetPositionKey(eventID)
	r.redis.HDel(ctx, positionKey, userKey)

	return nil
}

// GetPosition gets a user's position in the waitlist queue
func (r *repository) GetPosition(ctx context.Context, userID, eventID uuid.UUID) (int, error) {
	queueKey := GetQueueKey(eventID)
	userKey := userID.String()

	// Get rank (0-based) and convert to position (1-based)
	rank := r.redis.ZRank(ctx, queueKey, userKey).Val()
	if rank == 0 {
		// Check if user exists in queue (rank 0 could mean first position or not found)
		score := r.redis.ZScore(ctx, queueKey, userKey).Val()
		if score == 0 {
			return 0, fmt.Errorf("user not found in waitlist")
		}
	}

	return int(rank) + 1, nil
}

// GetQueueLength gets the total number of users in the waitlist
func (r *repository) GetQueueLength(ctx context.Context, eventID uuid.UUID) (int, error) {
	queueKey := GetQueueKey(eventID)
	length := r.redis.ZCard(ctx, queueKey).Val()
	return int(length), nil
}

// GetNextInQueue gets the next N users in the queue
func (r *repository) GetNextInQueue(ctx context.Context, eventID uuid.UUID, count int) ([]WaitlistEntry, error) {
	queueKey := GetQueueKey(eventID)

	// Get the first 'count' users from the queue (lowest scores)
	members := r.redis.ZRangeWithScores(ctx, queueKey, 0, int64(count-1)).Val()

	if len(members) == 0 {
		return []WaitlistEntry{}, nil
	}

	// Convert Redis members to user IDs and fetch full entries from database
	var userIDs []uuid.UUID
	for _, member := range members {
		userID, err := uuid.Parse(member.Member.(string))
		if err != nil {
			continue
		}
		userIDs = append(userIDs, userID)
	}

	// Fetch entries from database
	var entries []WaitlistEntry
	err := r.db.WithContext(ctx).
		Where("user_id IN ? AND event_id = ? AND status = ?", userIDs, eventID, WaitlistStatusActive).
		Order("position ASC").
		Find(&entries).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch waitlist entries: %w", err)
	}

	return entries, nil
}

// UpdatePositions recalculates and updates positions for all users in a queue
func (r *repository) UpdatePositions(ctx context.Context, eventID uuid.UUID) error {
	queueKey := GetQueueKey(eventID)
	lockKey := GetLockKey(eventID)

	// Acquire distributed lock
	lock := r.redis.SetNX(ctx, lockKey, "locked", 10*time.Second)
	if !lock.Val() {
		return fmt.Errorf("could not acquire lock for event %s", eventID)
	}
	defer r.redis.Del(ctx, lockKey)

	// Get all members in order
	members := r.redis.ZRange(ctx, queueKey, 0, -1).Val()

	// Rebuild the queue with correct positions
	pipe := r.redis.Pipeline()
	pipe.Del(ctx, queueKey)

	for i, member := range members {
		position := i + 1
		pipe.ZAdd(ctx, queueKey, redis.Z{
			Score:  float64(position),
			Member: member,
		})
	}

	pipe.Expire(ctx, queueKey, RedisKeyTTL)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update positions: %w", err)
	}

	return nil
}

// CreateEntry creates a new waitlist entry in the database
func (r *repository) CreateEntry(ctx context.Context, entry *WaitlistEntry) error {
	entry.ID = uuid.New()
	entry.CreatedAt = time.Now()
	entry.UpdatedAt = time.Now()

	err := r.db.WithContext(ctx).Create(entry).Error
	if err != nil {
		return fmt.Errorf("failed to create waitlist entry: %w", err)
	}

	return nil
}

// UpdateEntry updates an existing waitlist entry
func (r *repository) UpdateEntry(ctx context.Context, entry *WaitlistEntry) error {
	entry.UpdatedAt = time.Now()

	err := r.db.WithContext(ctx).
		Model(entry).
		Where("id = ?", entry.ID).
		Updates(entry).Error

	if err != nil {
		return fmt.Errorf("failed to update waitlist entry: %w", err)
	}

	return nil
}

// GetEntry gets a waitlist entry by user and event ID
func (r *repository) GetEntry(ctx context.Context, userID, eventID uuid.UUID) (*WaitlistEntry, error) {
	var entry WaitlistEntry
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND event_id = ?", userID, eventID).
		First(&entry).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("waitlist entry not found")
		}
		return nil, fmt.Errorf("failed to get waitlist entry: %w", err)
	}

	return &entry, nil
}

// GetEntryByID gets a waitlist entry by ID
func (r *repository) GetEntryByID(ctx context.Context, id uuid.UUID) (*WaitlistEntry, error) {
	var entry WaitlistEntry
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&entry).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("waitlist entry not found")
		}
		return nil, fmt.Errorf("failed to get waitlist entry: %w", err)
	}

	return &entry, nil
}

// ListEntries lists waitlist entries for an event with optional status filter
func (r *repository) ListEntries(ctx context.Context, eventID uuid.UUID, status WaitlistStatus) ([]WaitlistEntry, error) {
	var entries []WaitlistEntry
	query := r.db.WithContext(ctx).Where("event_id = ?", eventID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	err := query.Order("position ASC").Find(&entries).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list waitlist entries: %w", err)
	}

	return entries, nil
}

// DeleteEntry deletes a waitlist entry
func (r *repository) DeleteEntry(ctx context.Context, id uuid.UUID) error {
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&WaitlistEntry{}).Error

	if err != nil {
		return fmt.Errorf("failed to delete waitlist entry: %w", err)
	}

	return nil
}

// GetExpiredEntries gets waitlist entries with expired booking windows
func (r *repository) GetExpiredEntries(ctx context.Context, limit int) ([]WaitlistEntry, error) {
	var entries []WaitlistEntry
	err := r.db.WithContext(ctx).
		Where("status = ? AND expires_at IS NOT NULL AND expires_at < ?",
			WaitlistStatusNotified, time.Now()).
		Limit(limit).
		Find(&entries).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get expired entries: %w", err)
	}

	return entries, nil
}

// UpdateEntriesStatus updates the status of multiple entries
func (r *repository) UpdateEntriesStatus(ctx context.Context, ids []uuid.UUID, status WaitlistStatus) error {
	err := r.db.WithContext(ctx).
		Model(&WaitlistEntry{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error

	if err != nil {
		return fmt.Errorf("failed to update entries status: %w", err)
	}

	return nil
}

// GetWaitlistStats gets statistics for a waitlist
func (r *repository) GetWaitlistStats(ctx context.Context, eventID uuid.UUID) (*WaitlistStatsResponse, error) {
	var stats WaitlistStatsResponse
	stats.EventID = eventID

	// Get counts by status
	type StatusCount struct {
		Status WaitlistStatus
		Count  int
	}

	var statusCounts []StatusCount
	err := r.db.WithContext(ctx).
		Model(&WaitlistEntry{}).
		Select("status, COUNT(*) as count").
		Where("event_id = ?", eventID).
		Group("status").
		Scan(&statusCounts).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get waitlist stats: %w", err)
	}

	// Calculate totals
	for _, sc := range statusCounts {
		switch sc.Status {
		case WaitlistStatusActive:
			stats.ActiveInQueue = sc.Count
		case WaitlistStatusNotified:
			stats.NotifiedCount = sc.Count
		case WaitlistStatusConverted:
			stats.ConvertedCount = sc.Count
		}
		stats.TotalInQueue += sc.Count
	}

	// Get average wait time from analytics
	var avgWaitTime sql.NullInt64
	err = r.db.WithContext(ctx).
		Model(&WaitlistAnalytics{}).
		Select("AVG(avg_wait_time_minutes) as avg_wait_time").
		Where("event_id = ? AND avg_wait_time_minutes IS NOT NULL", eventID).
		Scan(&avgWaitTime).Error

	if err == nil && avgWaitTime.Valid {
		avgTime := int(avgWaitTime.Int64)
		stats.AverageWaitTime = &avgTime
	}

	return &stats, nil
}

// CreateAnalytics creates waitlist analytics entry
func (r *repository) CreateAnalytics(ctx context.Context, analytics *WaitlistAnalytics) error {
	analytics.ID = uuid.New()
	analytics.CreatedAt = time.Now()
	analytics.UpdatedAt = time.Now()

	err := r.db.WithContext(ctx).Create(analytics).Error
	if err != nil {
		return fmt.Errorf("failed to create analytics entry: %w", err)
	}

	return nil
}

// CreateNotification creates a new notification record
func (r *repository) CreateNotification(ctx context.Context, notification *WaitlistNotification) error {
	notification.ID = uuid.New()
	notification.CreatedAt = time.Now()
	notification.UpdatedAt = time.Now()

	err := r.db.WithContext(ctx).Create(notification).Error
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	return nil
}

// UpdateNotification updates a notification record
func (r *repository) UpdateNotification(ctx context.Context, notification *WaitlistNotification) error {
	notification.UpdatedAt = time.Now()

	err := r.db.WithContext(ctx).
		Model(notification).
		Where("id = ?", notification.ID).
		Updates(notification).Error

	if err != nil {
		return fmt.Errorf("failed to update notification: %w", err)
	}

	return nil
}

// GetPendingNotifications gets notifications that need to be processed
func (r *repository) GetPendingNotifications(ctx context.Context, limit int) ([]WaitlistNotification, error) {
	var notifications []WaitlistNotification
	err := r.db.WithContext(ctx).
		Where("status IN ?", []NotificationStatus{NotificationStatusPending, NotificationStatusRetry}).
		Order("created_at ASC").
		Limit(limit).
		Find(&notifications).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get pending notifications: %w", err)
	}

	return notifications, nil
}
