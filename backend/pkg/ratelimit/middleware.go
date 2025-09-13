package ratelimit

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"evently/internal/shared/utils/response"

	"github.com/gin-gonic/gin"
)

// rate limiting middleware
func Middleware(rateLimiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client IP
		clientIP := getClientIP(c)

		// Determine rate limit type from route
		limitType := getRateLimitType(c.FullPath())

		// Check rate limit
		result, err := rateLimiter.IsAllowed(c.Request.Context(), clientIP, limitType)
		if err != nil {
			response.RespondJSON(c, "error", http.StatusInternalServerError,
				"Rate limit check failed", nil, nil)
			c.Abort()
			return
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", result.Limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", result.Remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", result.ResetTime))

		// Check if rate limited
		if !result.Allowed {
			response.RespondJSON(c, "error", http.StatusTooManyRequests,
				"Rate limit exceeded", nil, map[string]interface{}{
					"limit":      result.Limit,
					"reset_time": result.ResetTime,
				})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Alternative approach: More specific route matching
func getRateLimitType(path string) RateLimitType {
	switch {
	// Health/monitoring endpoints
	case strings.HasPrefix(path, "/health"),
		strings.HasPrefix(path, "/ping"),
		strings.HasPrefix(path, "/status"):
		return RateLimitTypeHealth

	// Admin endpoints (catch-all for admin)
	case strings.Contains(path, "/admin/"):
		return RateLimitTypeAdmin

	// Analytics endpoints
	case strings.Contains(path, "/analytics"):
		return RateLimitTypeAnalytics

	// Authentication endpoints
	case strings.Contains(path, "/auth/"):
		return RateLimitTypeAuth

	// Critical booking flow endpoints
	case strings.Contains(path, "/seats/hold"),
		strings.Contains(path, "/seats/availability"),
		strings.Contains(path, "/bookings/confirm"),
		strings.Contains(path, "/bookings/") && (strings.Contains(path, "/cancel") || strings.Contains(path, "request-cancel")):
		return RateLimitTypeBookingCritical // New stricter type

	// Other booking-related endpoints
	case strings.Contains(path, "/booking") ||
		strings.Contains(path, "/seats") ||
		strings.Contains(path, "/waitlist") ||
		strings.Contains(path, "/cancellation"):
		return RateLimitTypeBooking

	// Public browsing endpoints
	case strings.Contains(path, "/events"),
		strings.Contains(path, "/tags"),
		strings.Contains(path, "/sections"):
		return RateLimitTypePublic

	// User-specific endpoints
	case strings.Contains(path, "/users/"):
		return RateLimitTypeUser // New type for user operations

	default:
		return RateLimitTypeDefault
	}
}

// extracts real client IP
func getClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header
	xForwardedFor := c.GetHeader("X-Forwarded-For")
	if xForwardedFor != "" {
		ips := strings.Split(xForwardedFor, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	// Check X-Real-IP header
	xRealIP := c.GetHeader("X-Real-IP")
	if xRealIP != "" {
		if net.ParseIP(xRealIP) != nil {
			return xRealIP
		}
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}

	return ip
}
