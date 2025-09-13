package seats

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// AtomicRedisOperations handles atomic Redis operations for seat holding
type AtomicRedisOperations struct {
	redis *redis.Client
}

// NewAtomicRedisOperations creates a new atomic Redis operations handler
func NewAtomicRedisOperations(redisClient *redis.Client) *AtomicRedisOperations {
	return &AtomicRedisOperations{
		redis: redisClient,
	}
}

// Lua script for atomic seat holding - prevents race conditions
const luaAtomicSeatHold = `
-- KEYS[1] = hold_id
-- ARGV[1] = user_id
-- ARGV[2] = event_id  
-- ARGV[3] = ttl_seconds
-- ARGV[4..N] = seat_ids

local hold_id = KEYS[1]
local user_id = ARGV[1]
local event_id = ARGV[2]
local ttl = tonumber(ARGV[3])

-- Check if all seats are available (not held)
for i = 4, #ARGV do
    local seat_id = ARGV[i]
    local seat_hold_key = "seat_hold:" .. seat_id
    
    if redis.call("EXISTS", seat_hold_key) == 1 then
        -- Seat is already held, return failure with the seat that's held
        return {0, seat_id}
    end
end

-- All seats are available, hold them atomically
local hold_key = "hold:" .. hold_id
local hold_seats_key = "hold_seats:" .. hold_id
local user_holds_key = "user_holds:" .. user_id
local created_at = redis.call("TIME")[1]

-- Create hold metadata
redis.call("HMSET", hold_key,
    "user_id", user_id,
    "event_id", event_id,
    "seat_count", #ARGV - 3,
    "created_at", created_at
)
redis.call("EXPIRE", hold_key, ttl)

-- Hold individual seats and add to hold set
for i = 4, #ARGV do
    local seat_id = ARGV[i]
    local seat_hold_key = "seat_hold:" .. seat_id
    local hold_value = user_id .. ":" .. hold_id
    
    redis.call("SETEX", seat_hold_key, ttl, hold_value)
    redis.call("SADD", hold_seats_key, seat_id)
end

-- Set expiry for hold seats set
redis.call("EXPIRE", hold_seats_key, ttl)

-- Add to user's holds
redis.call("SADD", user_holds_key, hold_id)
redis.call("EXPIRE", user_holds_key, ttl)

-- Return success
return {1, "success"}
`

// Lua script for atomic seat release
const luaAtomicSeatRelease = `
-- KEYS[1] = hold_id
local hold_id = KEYS[1]

local hold_key = "hold:" .. hold_id
local hold_seats_key = "hold_seats:" .. hold_id

-- Get hold metadata
local hold_data = redis.call("HGETALL", hold_key)
if #hold_data == 0 then
    return {0, "hold_not_found"}
end

local user_id = nil
for i = 1, #hold_data, 2 do
    if hold_data[i] == "user_id" then
        user_id = hold_data[i + 1]
        break
    end
end

if not user_id then
    return {0, "invalid_hold_data"}
end

-- Get all seats in this hold
local seat_ids = redis.call("SMEMBERS", hold_seats_key)

-- Release individual seat holds
for i = 1, #seat_ids do
    local seat_hold_key = "seat_hold:" .. seat_ids[i]
    redis.call("DEL", seat_hold_key)
end

-- Remove from user's holds
local user_holds_key = "user_holds:" .. user_id
redis.call("SREM", user_holds_key, hold_id)

-- Clean up hold metadata
redis.call("DEL", hold_key)
redis.call("DEL", hold_seats_key)

return {1, #seat_ids}
`

// AtomicHoldSeats atomically holds multiple seats using Lua script
func (a *AtomicRedisOperations) AtomicHoldSeats(ctx context.Context, seatIDs []uuid.UUID, userID, holdID, eventID string, ttl time.Duration) error {
	if a.redis == nil {
		return fmt.Errorf("redis client not available")
	}

	// Prepare arguments for Lua script
	keys := []string{holdID}
	args := []interface{}{
		userID,
		eventID,
		strconv.Itoa(int(ttl.Seconds())),
	}

	// Add seat IDs to arguments
	for _, seatID := range seatIDs {
		args = append(args, seatID.String())
	}

	// Execute Lua script
	result, err := a.redis.EvalSha(ctx, luaAtomicSeatHold, keys, args...).Result()
	if err != nil {
		// If script is not loaded, try to load and execute
		result, err = a.redis.Eval(ctx, luaAtomicSeatHold, keys, args...).Result()
		if err != nil {
			return fmt.Errorf("failed to execute atomic seat hold: %w", err)
		}
	}

	// Parse result
	resultArray, ok := result.([]interface{})
	if !ok || len(resultArray) != 2 {
		return fmt.Errorf("unexpected result format from Lua script")
	}

	success, ok := resultArray[0].(int64)
	if !ok {
		return fmt.Errorf("invalid success flag in Lua script result")
	}

	if success == 0 {
		conflictSeat, ok := resultArray[1].(string)
		if ok {
			return fmt.Errorf("seat already held: %s", conflictSeat)
		}
		return fmt.Errorf("failed to hold seats")
	}

	return nil
}

// AtomicReleaseHold atomically releases a hold using Lua script
func (a *AtomicRedisOperations) AtomicReleaseHold(ctx context.Context, holdID string) (int, error) {
	if a.redis == nil {
		return 0, fmt.Errorf("redis client not available")
	}

	// Execute Lua script
	result, err := a.redis.EvalSha(ctx, luaAtomicSeatRelease, []string{holdID}).Result()
	if err != nil {
		// If script is not loaded, try to load and execute
		result, err = a.redis.Eval(ctx, luaAtomicSeatRelease, []string{holdID}).Result()
		if err != nil {
			return 0, fmt.Errorf("failed to execute atomic seat release: %w", err)
		}
	}

	// Parse result
	resultArray, ok := result.([]interface{})
	if !ok || len(resultArray) != 2 {
		return 0, fmt.Errorf("unexpected result format from Lua script")
	}

	success, ok := resultArray[0].(int64)
	if !ok {
		return 0, fmt.Errorf("invalid success flag in Lua script result")
	}

	if success == 0 {
		reason, ok := resultArray[1].(string)
		if ok {
			return 0, fmt.Errorf("failed to release hold: %s", reason)
		}
		return 0, fmt.Errorf("failed to release hold")
	}

	releasedCount, ok := resultArray[1].(int64)
	if !ok {
		return 0, fmt.Errorf("invalid released count in Lua script result")
	}

	return int(releasedCount), nil
}

// PreloadScripts loads Lua scripts into Redis for better performance
func (a *AtomicRedisOperations) PreloadScripts(ctx context.Context) error {
	if a.redis == nil {
		return fmt.Errorf("redis client not available")
	}

	// Load seat hold script
	_, err := a.redis.ScriptLoad(ctx, luaAtomicSeatHold).Result()
	if err != nil {
		return fmt.Errorf("failed to load seat hold script: %w", err)
	}

	// Load seat release script
	_, err = a.redis.ScriptLoad(ctx, luaAtomicSeatRelease).Result()
	if err != nil {
		return fmt.Errorf("failed to load seat release script: %w", err)
	}

	return nil
}
