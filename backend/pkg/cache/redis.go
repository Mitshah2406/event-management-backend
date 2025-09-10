package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config holds Redis connection configuration
type Config struct {
	Address  string // Redis server address (host:port)
	Password string // Redis password (empty if no password)
	DB       int    // Redis database number (0-15)
}

// RedisConfig is an alias for compatibility with the main config package
// This allows using either cache.Config or the main RedisConfig structure
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	Addr     string
}

// RedisClient wraps the Redis client with additional functionality
type RedisClient struct {
	client *redis.Client
	ctx    context.Context
}

var redisClient *RedisClient

// NewConfigFromRedisConfig creates a cache.Config from a RedisConfig
// This provides compatibility with the main application's config structure
func NewConfigFromRedisConfig(rc RedisConfig) Config {
	address := rc.Addr
	if address == "" {
		address = rc.Host + ":" + rc.Port
	}

	return Config{
		Address:  address,
		Password: rc.Password,
		DB:       rc.DB,
	}
}

// Init initializes the Redis client with the provided configuration
func Init(cfg Config) error {
	if cfg.Address == "" {
		return fmt.Errorf("redis address cannot be empty")
	}

	// Create Redis client options
	opts := &redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
	}

	// Create new Redis client
	client := redis.NewClient(opts)

	// Create context with timeout for connection test
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test the connection
	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis at %s: %w", cfg.Address, err)
	}

	// Initialize the global client
	redisClient = &RedisClient{
		client: client,
		ctx:    context.Background(),
	}

	return nil
}

// InitWithRedisConfig initializes the Redis client using a RedisConfig struct
// This provides a convenient way to use the main application's config structure
func InitWithRedisConfig(rc RedisConfig) error {
	cfg := NewConfigFromRedisConfig(rc)
	return Init(cfg)
}

// Client returns the Redis client instance
// Returns nil if Init() hasn't been called successfully
func Client() *redis.Client {
	if redisClient == nil {
		return nil
	}
	return redisClient.client
}

// Close closes the Redis connection
func Close() error {
	if redisClient == nil {
		return fmt.Errorf("redis client is not initialized")
	}

	if err := redisClient.client.Close(); err != nil {
		return fmt.Errorf("failed to close Redis connection: %w", err)
	}

	redisClient = nil
	return nil
}

// GetContext returns the background context used by the Redis client
func GetContext() context.Context {
	if redisClient == nil {
		return context.Background()
	}
	return redisClient.ctx
}

// IsInitialized checks if the Redis client has been initialized
func IsInitialized() bool {
	return redisClient != nil && redisClient.client != nil
}

// Ping tests the Redis connection
func Ping() error {
	if redisClient == nil {
		return fmt.Errorf("redis client is not initialized")
	}

	ctx, cancel := context.WithTimeout(redisClient.ctx, 5*time.Second)
	defer cancel()

	if err := redisClient.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	return nil
}
