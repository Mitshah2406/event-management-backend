package seats

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Repository interface {
	// Seat CRUD
	CreateSeats(ctx context.Context, seats []Seat) error
	GetSeatByID(ctx context.Context, id uuid.UUID) (*Seat, error)
	GetSeatsBySectionID(ctx context.Context, sectionID uuid.UUID) ([]Seat, error)
	GetSeatsByIDs(ctx context.Context, seatIDs []uuid.UUID) ([]Seat, error)
	UpdateSeat(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error
	UpdateSeatStatus(ctx context.Context, id uuid.UUID, status string) error
	UpdateSeatsStatus(ctx context.Context, seatIDs []uuid.UUID, status string) error
	DeleteSeat(ctx context.Context, id uuid.UUID) error
	DeleteSeatsBySectionID(ctx context.Context, sectionID uuid.UUID) error

	// Availability checks
	CheckSeatsAvailability(ctx context.Context, seatIDs []uuid.UUID) (map[string]bool, error)
	GetAvailableSeatsInSection(ctx context.Context, sectionID uuid.UUID) ([]Seat, error)

	// Redis seat holding operations
	HoldSeats(ctx context.Context, seatIDs []uuid.UUID, userID, holdID, eventID string, ttl time.Duration) error
	ReleaseHold(ctx context.Context, holdID string) error
	CheckSeatHolds(ctx context.Context, seatIDs []uuid.UUID) (map[string]string, error) // seatID -> holdID
	GetUserHolds(ctx context.Context, userID string) ([]string, error)                  // returns holdIDs
	IsHoldValid(ctx context.Context, holdID string) (bool, error)
	GetHoldDetails(ctx context.Context, holdID string) (*SeatHoldDetails, error)
}

type repository struct {
	db    *gorm.DB
	redis *redis.Client
}

func NewRepository(db *gorm.DB, redisClient *redis.Client) Repository {
	return &repository{
		db:    db,
		redis: redisClient,
	}
}

// SEAT CRUD

func (r *repository) CreateSeats(ctx context.Context, seats []Seat) error {
	return r.db.WithContext(ctx).Create(&seats).Error
}

func (r *repository) GetSeatByID(ctx context.Context, id uuid.UUID) (*Seat, error) {
	var seat Seat
	err := r.db.WithContext(ctx).Preload("Section").First(&seat, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &seat, nil
}

func (r *repository) GetSeatsBySectionID(ctx context.Context, sectionID uuid.UUID) ([]Seat, error) {
	var seats []Seat
	err := r.db.WithContext(ctx).
		Where("section_id = ?", sectionID).
		Order("row ASC, position ASC").
		Find(&seats).Error
	return seats, err
}

func (r *repository) GetSeatsByIDs(ctx context.Context, seatIDs []uuid.UUID) ([]Seat, error) {
	var seats []Seat
	err := r.db.WithContext(ctx).
		Preload("Section").
		Where("id IN ?", seatIDs).
		Find(&seats).Error
	return seats, err
}

func (r *repository) UpdateSeat(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&Seat{}).Where("id = ?", id).Updates(updates).Error
}

func (r *repository) UpdateSeatStatus(ctx context.Context, id uuid.UUID, status string) error {
	return r.db.WithContext(ctx).Model(&Seat{}).Where("id = ?", id).Update("status", status).Error
}

func (r *repository) UpdateSeatsStatus(ctx context.Context, seatIDs []uuid.UUID, status string) error {
	return r.db.WithContext(ctx).Model(&Seat{}).Where("id IN ?", seatIDs).Update("status", status).Error
}

func (r *repository) DeleteSeat(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&Seat{}, "id = ?", id).Error
}

func (r *repository) DeleteSeatsBySectionID(ctx context.Context, sectionID uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&Seat{}, "section_id = ?", sectionID).Error
}

// AVAILABILITY CHECKS

func (r *repository) CheckSeatsAvailability(ctx context.Context, seatIDs []uuid.UUID) (map[string]bool, error) {
	var seats []Seat
	err := r.db.WithContext(ctx).
		Select("id, status").
		Where("id IN ?", seatIDs).
		Find(&seats).Error

	if err != nil {
		return nil, err
	}

	availability := make(map[string]bool)
	for _, seat := range seats {
		availability[seat.ID.String()] = seat.Status == "AVAILABLE"
	}

	return availability, nil
}

func (r *repository) GetAvailableSeatsInSection(ctx context.Context, sectionID uuid.UUID) ([]Seat, error) {
	var seats []Seat
	err := r.db.WithContext(ctx).
		Where("section_id = ? AND status = ?", sectionID, "AVAILABLE").
		Order("row ASC, position ASC").
		Find(&seats).Error
	return seats, err
}

// REDIS SEAT HOLDING

func (r *repository) HoldSeats(ctx context.Context, seatIDs []uuid.UUID, userID, holdID, eventID string, ttl time.Duration) error {
	// If Redis is not available, skip holding for now
	if r.redis == nil {
		return fmt.Errorf("redis client not available - seat holding disabled")
	}

	pipe := r.redis.TxPipeline()

	// Create hold metadata
	holdKey := fmt.Sprintf("hold:%s", holdID)
	holdData := map[string]interface{}{
		"user_id":    userID,
		"event_id":   eventID,
		"seat_count": len(seatIDs),
		"created_at": time.Now().Unix(),
		"expires_at": time.Now().Add(ttl).Unix(),
	}
	pipe.HMSet(ctx, holdKey, holdData)
	pipe.Expire(ctx, holdKey, ttl)

	// Hold individual seats
	for _, seatID := range seatIDs {
		seatHoldKey := fmt.Sprintf("seat_hold:%s", seatID.String())
		holdValue := fmt.Sprintf("%s:%s", userID, holdID)

		// Use SETNX to ensure atomic seat holding
		pipe.SetNX(ctx, seatHoldKey, holdValue, ttl)

		// Add seat to hold set
		pipe.SAdd(ctx, fmt.Sprintf("hold_seats:%s", holdID), seatID.String())
		pipe.Expire(ctx, fmt.Sprintf("hold_seats:%s", holdID), ttl)
	}

	// Add to user's holds
	userHoldsKey := fmt.Sprintf("user_holds:%s", userID)
	pipe.SAdd(ctx, userHoldsKey, holdID)
	pipe.Expire(ctx, userHoldsKey, ttl)

	_, err := pipe.Exec(ctx)
	return err
}

func (r *repository) ReleaseHold(ctx context.Context, holdID string) error {
	if r.redis == nil {
		return fmt.Errorf("redis client not available - seat holding disabled")
	}

	pipe := r.redis.TxPipeline()

	// Get seats in this hold
	holdSeatsKey := fmt.Sprintf("hold_seats:%s", holdID)
	seatIDs, err := r.redis.SMembers(ctx, holdSeatsKey).Result()
	if err != nil && err != redis.Nil {
		return err
	}

	// Get hold metadata to find user
	holdKey := fmt.Sprintf("hold:%s", holdID)
	holdData, err := r.redis.HGetAll(ctx, holdKey).Result()
	if err != nil && err != redis.Nil {
		return err
	}

	// Release individual seat holds
	for _, seatID := range seatIDs {
		seatHoldKey := fmt.Sprintf("seat_hold:%s", seatID)
		pipe.Del(ctx, seatHoldKey)
	}

	// Remove from user's holds if user data available
	if userID, exists := holdData["user_id"]; exists {
		userHoldsKey := fmt.Sprintf("user_holds:%s", userID)
		pipe.SRem(ctx, userHoldsKey, holdID)
	}

	// Clean up hold metadata
	pipe.Del(ctx, holdKey)
	pipe.Del(ctx, holdSeatsKey)

	_, err = pipe.Exec(ctx)
	return err
}

func (r *repository) CheckSeatHolds(ctx context.Context, seatIDs []uuid.UUID) (map[string]string, error) {
	holds := make(map[string]string)

	if r.redis == nil {
		return holds, nil // Return empty map if Redis not available
	}

	for _, seatID := range seatIDs {
		seatHoldKey := fmt.Sprintf("seat_hold:%s", seatID.String())
		holdValue, err := r.redis.Get(ctx, seatHoldKey).Result()

		if err == redis.Nil {
			// No hold on this seat
			continue
		} else if err != nil {
			return nil, err
		}

		holds[seatID.String()] = holdValue
	}

	return holds, nil
}

func (r *repository) GetUserHolds(ctx context.Context, userID string) ([]string, error) {
	if r.redis == nil {
		return []string{}, nil // Return empty slice if Redis not available
	}

	userHoldsKey := fmt.Sprintf("user_holds:%s", userID)
	holdIDs, err := r.redis.SMembers(ctx, userHoldsKey).Result()
	if err == redis.Nil {
		return []string{}, nil
	}
	return holdIDs, err
}

func (r *repository) IsHoldValid(ctx context.Context, holdID string) (bool, error) {
	if r.redis == nil {
		return false, fmt.Errorf("Redis client not available - seat holding disabled")
	}

	holdKey := fmt.Sprintf("hold:%s", holdID)
	exists, err := r.redis.Exists(ctx, holdKey).Result()
	return exists > 0, err
}

func (r *repository) GetHoldDetails(ctx context.Context, holdID string) (*SeatHoldDetails, error) {
	if r.redis == nil {
		return nil, fmt.Errorf("redis client not available - seat holding disabled")
	}

	holdKey := fmt.Sprintf("hold:%s", holdID)
	holdData, err := r.redis.HGetAll(ctx, holdKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("hold not found")
		}
		return nil, err
	}

	if len(holdData) == 0 {
		return nil, fmt.Errorf("hold not found")
	}

	// Get seat IDs
	holdSeatsKey := fmt.Sprintf("hold_seats:%s", holdID)
	seatIDs, err := r.redis.SMembers(ctx, holdSeatsKey).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}

	ttl, err := r.redis.TTL(ctx, holdKey).Result()
	if err != nil {
		ttl = 0
	}

	details := &SeatHoldDetails{
		HoldID:  holdID,
		UserID:  holdData["user_id"],
		EventID: holdData["event_id"],
		SeatIDs: seatIDs,
		TTL:     int(ttl.Seconds()),
	}

	return details, nil
}

// Helper struct

type SeatHoldDetails struct {
	HoldID  string   `json:"hold_id"`
	UserID  string   `json:"user_id"`
	EventID string   `json:"event_id"`
	SeatIDs []string `json:"seat_ids"`
	TTL     int      `json:"ttl_seconds"`
}
