package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// configuration
type Config struct {
	// Server configuration
	Port           string
	GinMode        string
	APIVersion     string
	APIPrefix      string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	MaxHeaderBytes int

	// Database configuration
	Database DatabaseConfig

	// Redis configuration
	Redis RedisConfig

	// JWT configuration
	JWT JWTConfig

	// Rate limiting
	RateLimit RateLimitConfig

	// File upload
	Upload UploadConfig

	// Logging
	LogLevel string

	// External services
	AWS   AWSConfig
	Email EmailConfig
}

// database configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSLMode  string
	DSN      string
}

// Redis configuration
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	Addr     string

	SeatHoldTTL time.Duration
	SessionTTL  time.Duration
	CacheTTL    time.Duration
	TempDataTTL time.Duration
}

// JWT configuration
type JWTConfig struct {
	Secret           string
	JWTExpiresIn     time.Duration
	RefreshExpiresIn time.Duration
}

// rate limiting configuration
type RateLimitConfig struct {
	Enabled                 bool          `json:"enabled"`
	WindowDuration          time.Duration `json:"window_duration"`
	DefaultRequests         int           `json:"default_requests"`
	PublicRequests          int           `json:"public_requests"`
	AuthRequests            int           `json:"auth_requests"`
	BookingRequests         int           `json:"booking_requests"`
	BookingCriticalRequests int           `json:"booking_critical_requests"` // NEW
	AdminRequests           int           `json:"admin_requests"`
	AnalyticsRequests       int           `json:"analytics_requests"`
	UserRequests            int           `json:"user_requests"`   // NEW
	HealthRequests          int           `json:"health_requests"` // NEW
	WhitelistedIPs          []string      `json:"whitelisted_ips"`
}

type UploadConfig struct {
	MaxSize int64
	Path    string
}

type AWSConfig struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	S3Bucket        string
}

type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
}

func Load() *Config {
	cfg := &Config{
		// Server configuration
		Port:           getEnv("PORT", "8080"),
		GinMode:        getEnv("GIN_MODE", "debug"),
		APIVersion:     getEnv("API_VERSION", "v1"),
		APIPrefix:      getEnv("API_PREFIX", "/api"),
		ReadTimeout:    getDurationEnv("READ_TIMEOUT", 15*time.Second),
		WriteTimeout:   getDurationEnv("WRITE_TIMEOUT", 15*time.Second),
		IdleTimeout:    getDurationEnv("IDLE_TIMEOUT", 60*time.Second),
		MaxHeaderBytes: getIntEnv("MAX_HEADER_BYTES", 1<<20), // 1 MB

		// Database configuration
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			Name:     getEnv("DB_NAME", "evently_db"),
			User:     getEnv("DB_USER", "evently_user"),
			Password: getEnv("DB_PASSWORD", "evently_password"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},

		// Redis configuration
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getIntEnv("REDIS_DB", 0),

			// TTL configurations with defaults
			SeatHoldTTL: getDurationEnv("REDIS_SEAT_HOLD_TTL", 10*time.Minute),
			SessionTTL:  getDurationEnv("REDIS_SESSION_TTL", 24*time.Hour),
			CacheTTL:    getDurationEnv("REDIS_CACHE_TTL", 1*time.Hour),
			TempDataTTL: getDurationEnv("REDIS_TEMP_DATA_TTL", 5*time.Minute),
		},

		// JWT configuration
		JWT: JWTConfig{
			Secret:           getEnv("JWT_SECRET", "your-super-secret-jwt-key"),
			JWTExpiresIn:     getDurationEnvSeconds("JWT_EXPIRES_IN", 15*time.Minute),
			RefreshExpiresIn: getDurationEnvSeconds("JWT_REFRESH_EXPIRES_IN", 24*time.Hour),
		},

		// Rate limiting
		RateLimit: RateLimitConfig{
			Enabled:                 getBoolEnv("RATE_LIMIT_ENABLED", true),
			WindowDuration:          getDurationEnv("RATE_LIMIT_WINDOW_DURATION", 60*time.Second),
			DefaultRequests:         getIntEnv("RATE_LIMIT_DEFAULT_REQUESTS", 100),
			PublicRequests:          getIntEnv("RATE_LIMIT_PUBLIC_REQUESTS", 300),
			AuthRequests:            getIntEnv("RATE_LIMIT_AUTH_REQUESTS", 30),
			BookingRequests:         getIntEnv("RATE_LIMIT_BOOKING_REQUESTS", 80),
			BookingCriticalRequests: getIntEnv("RATE_LIMIT_BOOKING_CRITICAL_REQUESTS", 20),
			AdminRequests:           getIntEnv("RATE_LIMIT_ADMIN_REQUESTS", 600),
			AnalyticsRequests:       getIntEnv("RATE_LIMIT_ANALYTICS_REQUESTS", 120),
			UserRequests:            getIntEnv("RATE_LIMIT_USER_REQUESTS", 150),
			HealthRequests:          getIntEnv("RATE_LIMIT_HEALTH_REQUESTS", 1000),
			WhitelistedIPs:          getStringSliceEnv("RATE_LIMIT_WHITELISTED_IPS", []string{}),
		},

		// File upload
		Upload: UploadConfig{
			MaxSize: getInt64Env("MAX_UPLOAD_SIZE", 10*1024*1024), // 10 MB
			Path:    getEnv("UPLOAD_PATH", "./uploads"),
		},

		// Logging
		LogLevel: getEnv("LOG_LEVEL", "debug"),

		AWS: AWSConfig{
			Region:          getEnv("AWS_REGION", ""),
			AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
			S3Bucket:        getEnv("S3_BUCKET", ""),
		},

		Email: EmailConfig{
			SMTPHost:     getEnv("SMTP_HOST", ""),
			SMTPPort:     getIntEnv("SMTP_PORT", 587),
			SMTPUsername: getEnv("SMTP_USERNAME", ""),
			SMTPPassword: getEnv("SMTP_PASSWORD", ""),
			FromEmail:    getEnv("FROM_EMAIL", "noreply@evently.com"),
		},
	}

	cfg.Database.DSN = buildDatabaseDSN(cfg.Database)
	cfg.Redis.Addr = cfg.Redis.Host + ":" + cfg.Redis.Port

	return cfg
}

// builds the database connection string
func buildDatabaseDSN(db DatabaseConfig) string {
	return "host=" + db.Host +
		" port=" + db.Port +
		" user=" + db.User +
		" password=" + db.Password +
		" dbname=" + db.Name +
		" sslmode=" + db.SSLMode
}

// gets an environment variable with a fallback value
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// gets an integer environment variable with a fallback value
func getIntEnv(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return fallback
}

// gets an int64 environment variable with a fallback value
func getInt64Env(key string, fallback int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return fallback
}

// gets a duration environment variable with a fallback value
func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return fallback
}

// gets an environment variable as seconds (int) and converts to time.Duration
func getDurationEnvSeconds(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if seconds, err := strconv.Atoi(value); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	return fallback
}

// gets a boolean environment variable with a fallback value
func getBoolEnv(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return fallback
}

// gets a comma-separated string environment variable as a slice
func getStringSliceEnv(key string, fallback []string) []string {
	if value := os.Getenv(key); value != "" {
		parts := strings.Split(value, ",")
		var result []string
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return fallback
}

func (c *Config) IsProduction() bool {
	return c.GinMode == "release"
}

func (c *Config) IsDevelopment() bool {
	return c.GinMode == "debug"
}

func (c *Config) GetServerAddress() string {
	return ":" + c.Port
}

func (c *Config) GetAPIBasePath() string {
	return c.APIPrefix + "/" + c.APIVersion
}
