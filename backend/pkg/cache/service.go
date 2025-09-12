package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Service interface {
	// Generic cache operations
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	DeletePattern(ctx context.Context, pattern string) error
	Exists(ctx context.Context, key string) bool

	// Batch operations
	MGet(ctx context.Context, keys []string, dest interface{}) error
	MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error

	// Cache-aside pattern helpers
	GetOrSet(ctx context.Context, key string, ttl time.Duration, fetcher func() (interface{}, error), dest interface{}) error

	// Health check
	Ping(ctx context.Context) error
}

type service struct {
	client *redis.Client
}

func NewService(client *redis.Client) Service {
	return &service{client: client}
}

func (s *service) Get(ctx context.Context, key string, dest interface{}) error {
	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return ErrCacheMiss
		}
		return fmt.Errorf("cache get error: %w", err)
	}

	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return fmt.Errorf("cache unmarshal error: %w", err)
	}

	return nil
}

func (s *service) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache marshal error: %w", err)
	}

	if err := s.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("cache set error: %w", err)
	}

	return nil
}

func (s *service) Delete(ctx context.Context, key string) error {
	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("cache delete error: %w", err)
	}
	return nil
}

func (s *service) DeletePattern(ctx context.Context, pattern string) error {
	keys, err := s.client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("cache keys error: %w", err)
	}

	if len(keys) > 0 {
		if err := s.client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("cache delete pattern error: %w", err)
		}
	}

	return nil
}

func (s *service) Exists(ctx context.Context, key string) bool {
	result, err := s.client.Exists(ctx, key).Result()
	return err == nil && result > 0
}

func (s *service) MGet(ctx context.Context, keys []string, dest interface{}) error {
	values, err := s.client.MGet(ctx, keys...).Result()
	if err != nil {
		return fmt.Errorf("cache mget error: %w", err)
	}

	results := make([]interface{}, len(values))
	for i, val := range values {
		if val != nil {
			var item interface{}
			if err := json.Unmarshal([]byte(val.(string)), &item); err != nil {
				return fmt.Errorf("cache unmarshal error: %w", err)
			}
			results[i] = item
		}
	}

	data, err := json.Marshal(results)
	if err != nil {
		return fmt.Errorf("cache marshal results error: %w", err)
	}

	return json.Unmarshal(data, dest)
}

func (s *service) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	pipe := s.client.Pipeline()

	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("cache marshal error for key %s: %w", key, err)
		}
		pipe.Set(ctx, key, data, ttl)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("cache mset error: %w", err)
	}

	return nil
}

func (s *service) GetOrSet(ctx context.Context, key string, ttl time.Duration, fetcher func() (interface{}, error), dest interface{}) error {
	// Try to get from cache first
	err := s.Get(ctx, key, dest)
	if err == nil {
		return nil // Cache hit
	}

	if err != ErrCacheMiss {
		// Some other error occurred, log it but continue to fetch
		// In production, you might want to use structured logging
		fmt.Printf("Cache get error (continuing to fetch): %v\n", err)
	}

	// Cache miss, fetch data
	data, err := fetcher()
	if err != nil {
		return fmt.Errorf("fetcher error: %w", err)
	}

	// Store in cache (fire and forget - don't fail the request if cache set fails)
	go func() {
		if setErr := s.Set(context.Background(), key, data, ttl); setErr != nil {
			fmt.Printf("Cache set error (non-blocking): %v\n", setErr)
		}
	}()

	// Marshal and unmarshal to ensure dest gets the right data structure
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal fetched data error: %w", err)
	}

	return json.Unmarshal(jsonData, dest)
}

func (s *service) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

// Error definitions
var (
	ErrCacheMiss = fmt.Errorf("cache miss")
)
