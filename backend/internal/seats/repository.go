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
	AtomicHoldSeats(ctx context.Context, seatIDs []uuid.UUID, userID, holdID, eventID string, ttl time.Duration) error
	ReleaseHold(ctx context.Context, holdID string) error
	AtomicReleaseHold(ctx context.Context, holdID string) (int, error)
	CheckSeatHolds(ctx context.Context, seatIDs []uuid.UUID) (map[string]string, error) // seatID -> holdID
	GetUserHolds(ctx context.Context, userID string) ([]string, error)                  // returns holdIDs
	IsHoldValid(ctx context.Context, holdID string) (bool, error)
	GetHoldDetails(ctx context.Context, holdID string) (*SeatHoldDetails, error)
}

type repository struct {
	db          *gorm.DB
	redis       *redis.Client
	atomicRedis *AtomicRedisOperations
}

func NewRepository(db *gorm.DB, redisClient *redis.Client) Repository {
	atomicRedis := NewAtomicRedisOperations(redisClient)
	return &repository{
		db:          db,
		redis:       redisClient,
		atomicRedis: atomicRedis,
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
	// Use atomic version for better concurrency control
	return r.AtomicHoldSeats(ctx, seatIDs, userID, holdID, eventID, ttl)
}

// AtomicHoldSeats provides atomic seat holding using Lua scripts
func (r *repository) AtomicHoldSeats(ctx context.Context, seatIDs []uuid.UUID, userID, holdID, eventID string, ttl time.Duration) error {
	if r.atomicRedis == nil {
		return fmt.Errorf("atomic redis operations not available - seat holding disabled")
	}

	return r.atomicRedis.AtomicHoldSeats(ctx, seatIDs, userID, holdID, eventID, ttl)
}

func (r *repository) ReleaseHold(ctx context.Context, holdID string) error {
	// Use atomic version for better consistency
	_, err := r.AtomicReleaseHold(ctx, holdID)
	return err
}

// AtomicReleaseHold provides atomic hold release using Lua scripts
func (r *repository) AtomicReleaseHold(ctx context.Context, holdID string) (int, error) {
	if r.atomicRedis == nil {
		return 0, fmt.Errorf("atomic redis operations not available - seat holding disabled")
	}

	return r.atomicRedis.AtomicReleaseHold(ctx, holdID)
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
