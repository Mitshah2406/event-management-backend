package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger wraps slog.Logger with additional functionality
type Logger struct {
	*slog.Logger
}

// New creates a new logger instance
func New() *Logger {
	// Get log level from environment
	level := getLogLevel(os.Getenv("LOG_LEVEL"))

	// Create handler options
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug,
	}

	// Create handler based on environment
	var handler slog.Handler
	if gin.Mode() == gin.DebugMode {
		// Use text handler for development (more readable)
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		// Use JSON handler for production (structured)
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	// Create logger
	logger := slog.New(handler)

	return &Logger{
		Logger: logger,
	}
}

// getLogLevel converts string to slog.Level
func getLogLevel(levelStr string) slog.Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithRequestID adds request ID to logger context
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{
		Logger: l.Logger.With(slog.String("request_id", requestID)),
	}
}

// WithUserID adds user ID to logger context
func (l *Logger) WithUserID(userID string) *Logger {
	return &Logger{
		Logger: l.Logger.With(slog.String("user_id", userID)),
	}
}

// WithError adds error to logger context
func (l *Logger) WithError(err error) *Logger {
	return &Logger{
		Logger: l.Logger.With(slog.String("error", err.Error())),
	}
}

// WithFields adds multiple fields to logger context
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, slog.Any(k, v))
	}
	return &Logger{
		Logger: l.Logger.With(args...),
	}
}

// HTTP logging methods

// LogHTTPRequest logs an HTTP request
func (l *Logger) LogHTTPRequest(c *gin.Context, duration time.Duration) {
	l.Logger.InfoContext(c.Request.Context(),
		"HTTP Request",
		slog.String("method", c.Request.Method),
		slog.String("path", c.Request.URL.Path),
		slog.String("query", c.Request.URL.RawQuery),
		slog.Int("status", c.Writer.Status()),
		slog.Duration("duration", duration),
		slog.String("ip", c.ClientIP()),
		slog.String("user_agent", c.Request.UserAgent()),
		slog.Int("size", c.Writer.Size()),
	)
}

// LogHTTPError logs an HTTP error
func (l *Logger) LogHTTPError(c *gin.Context, err error, statusCode int) {
	l.Logger.ErrorContext(c.Request.Context(),
		"HTTP Error",
		slog.String("method", c.Request.Method),
		slog.String("path", c.Request.URL.Path),
		slog.Int("status", statusCode),
		slog.String("error", err.Error()),
		slog.String("ip", c.ClientIP()),
	)
}

// Database logging methods

// LogDBQuery logs a database query
func (l *Logger) LogDBQuery(ctx context.Context, query string, duration time.Duration, err error) {
	if err != nil {
		l.Logger.ErrorContext(ctx,
			"Database Query Error",
			slog.String("query", query),
			slog.Duration("duration", duration),
			slog.String("error", err.Error()),
		)
	} else {
		l.Logger.DebugContext(ctx,
			"Database Query",
			slog.String("query", query),
			slog.Duration("duration", duration),
		)
	}
}

// Business logic logging methods

// LogEventCreated logs when an event is created
func (l *Logger) LogEventCreated(ctx context.Context, eventID, userID string) {
	l.Logger.InfoContext(ctx,
		"Event Created",
		slog.String("event_id", eventID),
		slog.String("user_id", userID),
	)
}

// LogBookingCreated logs when a booking is created
func (l *Logger) LogBookingCreated(ctx context.Context, bookingID, eventID, userID string) {
	l.Logger.InfoContext(ctx,
		"Booking Created",
		slog.String("booking_id", bookingID),
		slog.String("event_id", eventID),
		slog.String("user_id", userID),
	)
}

// LogBookingCancelled logs when a booking is cancelled
func (l *Logger) LogBookingCancelled(ctx context.Context, bookingID, eventID, userID string) {
	l.Logger.InfoContext(ctx,
		"Booking Cancelled",
		slog.String("booking_id", bookingID),
		slog.String("event_id", eventID),
		slog.String("user_id", userID),
	)
}

// Security logging methods

// LogAuthSuccess logs successful authentication
func (l *Logger) LogAuthSuccess(ctx context.Context, userID, method string) {
	l.Logger.InfoContext(ctx,
		"Authentication Success",
		slog.String("user_id", userID),
		slog.String("method", method),
	)
}

// LogAuthFailure logs failed authentication
func (l *Logger) LogAuthFailure(ctx context.Context, reason, ip string) {
	l.Logger.WarnContext(ctx,
		"Authentication Failure",
		slog.String("reason", reason),
		slog.String("ip", ip),
	)
}

// LogRateLimitExceeded logs rate limit exceeded
func (l *Logger) LogRateLimitExceeded(ctx context.Context, ip, endpoint string) {
	l.Logger.WarnContext(ctx,
		"Rate Limit Exceeded",
		slog.String("ip", ip),
		slog.String("endpoint", endpoint),
	)
}

// Performance logging methods

// LogSlowQuery logs slow database queries
func (l *Logger) LogSlowQuery(ctx context.Context, query string, duration time.Duration) {
	l.Logger.WarnContext(ctx,
		"Slow Database Query",
		slog.String("query", query),
		slog.Duration("duration", duration),
	)
}

// LogHighMemoryUsage logs high memory usage
func (l *Logger) LogHighMemoryUsage(ctx context.Context, usage uint64) {
	l.Logger.WarnContext(ctx,
		"High Memory Usage",
		slog.Uint64("memory_usage_mb", usage/1024/1024),
	)
}

// Helper methods for common patterns

// InfoWithContext logs an info message with context
func (l *Logger) InfoWithContext(ctx context.Context, msg string, fields map[string]interface{}) {
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, slog.Any(k, v))
	}
	l.Logger.InfoContext(ctx, msg, args...)
}

// ErrorWithContext logs an error message with context
func (l *Logger) ErrorWithContext(ctx context.Context, msg string, err error, fields map[string]interface{}) {
	args := make([]interface{}, 0, len(fields)*2+2)
	args = append(args, slog.String("error", err.Error()))
	for k, v := range fields {
		args = append(args, slog.Any(k, v))
	}
	l.Logger.ErrorContext(ctx, msg, args...)
}

// DebugWithContext logs a debug message with context
func (l *Logger) DebugWithContext(ctx context.Context, msg string, fields map[string]interface{}) {
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, slog.Any(k, v))
	}
	l.Logger.DebugContext(ctx, msg, args...)
}

// Global logger instance (can be replaced with dependency injection)
var defaultLogger = New()

// GetDefault returns the default logger instance
func GetDefault() *Logger {
	return defaultLogger
}

// SetDefault sets the default logger instance
func SetDefault(logger *Logger) {
	defaultLogger = logger
}
