package tags

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"evently/internal/shared/constants"

	"github.com/redis/go-redis/v9"
)

// Cache Helper Methods

func SetCache(ctx context.Context, client *redis.Client, key string, value interface{}, ttl time.Duration) error {
	if client == nil {
		return nil // skip caching if Redis is not available
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}

	return client.Set(ctx, key, data, ttl).Err()
}

func GetCache(ctx context.Context, client *redis.Client, key string, dest interface{}) error {
	if client == nil {
		return fmt.Errorf("redis client not available")
	}

	data, err := client.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(data), dest)
}

func DeleteCache(ctx context.Context, client *redis.Client, keys ...string) error {
	if client == nil || len(keys) == 0 {
		return nil
	}

	return client.Del(ctx, keys...).Err()
}

func InvalidateTagCache(ctx context.Context, client *redis.Client) error {
	if client == nil {
		return nil
	}

	keys, err := client.Keys(ctx, constants.PATTERN_INVALIDATE_TAGS_ALL).Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return client.Del(ctx, keys...).Err()
	}

	return nil
}

// Tag Helper Methods

// converts a tag name to a URL-friendly slug
func GenerateSlug(name string) string {
	slug := strings.ToLower(name)

	reg := regexp.MustCompile(`[^\w\s-]`)
	slug = reg.ReplaceAllString(slug, "")

	reg = regexp.MustCompile(`[\s-]+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Trim hyphens from start and end
	return strings.Trim(slug, "-")
}

// validates hex color codes
func IsValidHexColor(color string) bool {
	if len(color) != 7 || color[0] != '#' {
		return false
	}
	match, _ := regexp.MatchString("^#[0-9A-Fa-f]{6}$", color)
	return match
}
