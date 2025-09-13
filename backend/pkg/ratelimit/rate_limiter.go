package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimitType string

const (
	RateLimitTypeDefault         RateLimitType = "default"
	RateLimitTypePublic          RateLimitType = "public"
	RateLimitTypeAuth            RateLimitType = "auth"
	RateLimitTypeBooking         RateLimitType = "booking"
	RateLimitTypeBookingCritical RateLimitType = "booking_critical"
	RateLimitTypeAdmin           RateLimitType = "admin"
	RateLimitTypeAnalytics       RateLimitType = "analytics"
	RateLimitTypeUser            RateLimitType = "user"
	RateLimitTypeHealth          RateLimitType = "health"
)

// Enhanced Config with new rate limit types
type Config struct {
	Enabled                 bool          `json:"enabled"`
	WindowDuration          time.Duration `json:"window_duration"`
	DefaultRequests         int           `json:"default_requests"`
	PublicRequests          int           `json:"public_requests"`
	AuthRequests            int           `json:"auth_requests"`
	BookingRequests         int           `json:"booking_requests"`
	BookingCriticalRequests int           `json:"booking_critical_requests"`
	AdminRequests           int           `json:"admin_requests"`
	AnalyticsRequests       int           `json:"analytics_requests"`
	UserRequests            int           `json:"user_requests"`
	HealthRequests          int           `json:"health_requests"`
	WhitelistedIPs          []string      `json:"whitelisted_ips"`
}

// Result represents rate limit check result
type Result struct {
	Allowed   bool  `json:"allowed"`
	Limit     int   `json:"limit"`
	Remaining int   `json:"remaining"`
	ResetTime int64 `json:"reset_time"`
}

// RateLimiter handles rate limiting using Redis
type RateLimiter struct {
	client *redis.Client
	config *Config
}

func NewRateLimiter(client *redis.Client, config *Config) *RateLimiter {
	return &RateLimiter{
		client: client,
		config: config,
	}
}

// checks if request is allowed
func (r *RateLimiter) IsAllowed(ctx context.Context, clientIP string, limitType RateLimitType) (*Result, error) {
	if !r.config.Enabled {
		limit := r.getLimit(limitType)
		return &Result{
			Allowed:   true,
			Limit:     limit,
			Remaining: limit,
			ResetTime: time.Now().Add(r.config.WindowDuration).Unix(),
		}, nil
	}

	// Check if IP is whitelisted
	if r.isWhitelisted(clientIP) {
		limit := r.getLimit(limitType)
		return &Result{
			Allowed:   true,
			Limit:     limit,
			Remaining: limit,
			ResetTime: time.Now().Add(r.config.WindowDuration).Unix(),
		}, nil
	}

	// Create Redis key
	key := fmt.Sprintf("evently:ratelimit:%s:%s", clientIP, limitType)
	limit := r.getLimit(limitType)

	return r.checkLimit(ctx, key, limit)
}

// performs the actual rate limit check using sliding window
func (r *RateLimiter) checkLimit(ctx context.Context, key string, limit int) (*Result, error) {
	now := time.Now()
	windowStart := now.Add(-r.config.WindowDuration)

	// Lua script for atomic sliding window rate limiting
	luaScript := `
		local key = KEYS[1]
		local window_start = tonumber(ARGV[1])
		local now = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])
		local window_seconds = tonumber(ARGV[4])

		-- Remove old entries
		redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)

		-- Count current requests
		local current_count = redis.call('ZCARD', key)

		-- Check if limit exceeded
		if current_count >= limit then
			redis.call('EXPIRE', key, window_seconds)
			return {current_count, limit - current_count}
		end

		-- Add current request
		redis.call('ZADD', key, now, now)
		redis.call('EXPIRE', key, window_seconds)
		
		return {current_count + 1, limit - current_count - 1}
	`

	result, err := r.client.Eval(ctx, luaScript, []string{key},
		windowStart.Unix(),
		now.Unix(),
		limit,
		int(r.config.WindowDuration.Seconds())).Result()

	if err != nil {
		return nil, fmt.Errorf("redis eval failed: %w", err)
	}

	values, ok := result.([]interface{})
	if !ok || len(values) != 2 {
		return nil, fmt.Errorf("unexpected redis response")
	}

	currentCount, _ := strconv.Atoi(fmt.Sprintf("%.0f", values[0]))
	remaining, _ := strconv.Atoi(fmt.Sprintf("%.0f", values[1]))

	return &Result{
		Allowed:   currentCount <= limit,
		Limit:     limit,
		Remaining: remaining,
		ResetTime: now.Add(r.config.WindowDuration).Unix(),
	}, nil
}

func (r *RateLimiter) getLimit(limitType RateLimitType) int {
	switch limitType {
	case RateLimitTypePublic:
		return r.config.PublicRequests
	case RateLimitTypeAuth:
		return r.config.AuthRequests
	case RateLimitTypeBooking:
		return r.config.BookingRequests
	case RateLimitTypeBookingCritical:
		return r.config.BookingCriticalRequests
	case RateLimitTypeAdmin:
		return r.config.AdminRequests
	case RateLimitTypeAnalytics:
		return r.config.AnalyticsRequests
	case RateLimitTypeUser:
		return r.config.UserRequests
	case RateLimitTypeHealth:
		return r.config.HealthRequests
	default:
		return r.config.DefaultRequests
	}
}

func (r *RateLimiter) isWhitelisted(ip string) bool {
	// for _, whitelistedIP := range r.config.WhitelistedIPs {
	// 	if ip == whitelistedIP {
	// 		return true
	// 	}
	// }
	return false // whitelisting disabled for task
}
