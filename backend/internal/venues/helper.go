package venues

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"evently/internal/shared/constants"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func SetCache(ctx context.Context, redisClient *redis.Client, key string, value interface{}, ttl time.Duration) error {
	if redisClient == nil {
		return nil // Skip caching if Redis not available
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}

	return redisClient.Set(ctx, key, data, ttl).Err()
}

func GetCache(ctx context.Context, redisClient *redis.Client, key string, dest interface{}) error {
	if redisClient == nil {
		return fmt.Errorf("redis client not available")
	}

	data, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(data), dest)
}

func DeleteCache(ctx context.Context, redisClient *redis.Client, keys ...string) error {
	if redisClient == nil || len(keys) == 0 {
		return nil
	}

	return redisClient.Del(ctx, keys...).Err()
}

func InvalidateVenueCache(ctx context.Context, redisClient *redis.Client, templateID *uuid.UUID) error {
	if redisClient == nil {
		return nil
	}

	patterns := []string{
		constants.PATTERN_INVALIDATE_VENUES_ALL,
	}

	if templateID != nil {
		patterns = append(patterns, constants.CACHE_KEY_VENUE_TEMPLATE+templateID.String()+"*")
		patterns = append(patterns, constants.CACHE_KEY_VENUE_SECTIONS+templateID.String()+"*")
	}

	for _, pattern := range patterns {
		keys, err := redisClient.Keys(ctx, pattern).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := redisClient.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}
	}

	return nil
}
