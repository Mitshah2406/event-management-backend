package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os" // Added for file handling
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// TestResponse represents a simple test response structure
type TestResponse struct {
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	ID        string    `json:"id"`
}

// CacheTestClient simulates cache testing
type CacheTestClient struct {
	baseURL     string
	client      *http.Client
	redisClient *redis.Client
	logFile     *os.File    // Added for log file
	logger      *log.Logger // Added for file logger
}

// NewCacheTestClient initializes the test client with Redis and file logger
func NewCacheTestClient(baseURL string, logFile *os.File) *CacheTestClient {
	return &CacheTestClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		redisClient: redis.NewClient(&redis.Options{
			Addr:     "localhost:6379", // Adjust Redis address as needed
			Password: "",               // Set password if required
			DB:       0,                // Default DB
		}),
		logFile: logFile,
		logger:  log.New(logFile, "", log.LstdFlags), // Initialize logger to write to file
	}
}

// listRedisKeys logs all Redis keys matching a pattern to the log file
func (c *CacheTestClient) listRedisKeys(ctx context.Context, pattern string) {
	keys, err := c.redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		c.logger.Printf("Error fetching Redis keys: %v", err) // Log to file
		return
	}
	if len(keys) == 0 {
		c.logger.Println("No Redis keys found") // Log to file
		return
	}
	c.logger.Println("Redis keys:", keys) // Log to file
}

// invalidateCache simulates cache invalidation for a given key
func (c *CacheTestClient) invalidateCache(ctx context.Context, key string) error {
	err := c.redisClient.Del(ctx, key).Err()
	if err != nil {
		c.logger.Printf("Failed to invalidate cache key %s: %v", key, err) // Log to file
		return fmt.Errorf("failed to invalidate cache key %s: %v", key, err)
	}
	c.logger.Printf("Invalidated cache key %s", key) // Log to file
	return nil
}

func (c *CacheTestClient) makeRequest(endpoint string) (*TestResponse, time.Duration, error) {
	ctx := context.Background()

	// Generate a cache key based on the endpoint
	cacheKey := "cache:" + endpoint

	// Log Redis keys before operation
	fmt.Printf("    📜 Logging Redis keys before request (%s) to cache_test.log\n", endpoint)
	c.listRedisKeys(ctx, "*")

	start := time.Now()

	// Simulate cache set by making the HTTP request
	resp, err := c.client.Get(c.baseURL + endpoint)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	duration := time.Since(start)

	var result TestResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, duration, err
	}

	// Simulate setting cache
	err = c.redisClient.Set(ctx, cacheKey, "cached_response", 1*time.Hour).Err()
	if err != nil {
		log.Printf("    ❌ Error setting cache for %s: %v", cacheKey, err)
	}

	// Log Redis keys after setting cache
	fmt.Printf("    📜 Logging Redis keys after setting cache (%s) to cache_test.log\n", endpoint)
	c.listRedisKeys(ctx, "*")

	return &result, duration, nil
}

func (c *CacheTestClient) testEventListingCache() {
	ctx := context.Background()
	fmt.Println("🎯 Testing Event Listing Cache (Phase 2)")

	endpoints := []string{
		"/api/v1/events?page=1&limit=10",
		"/api/v1/events?page=1&limit=10&status=published",
		"/api/v1/events/upcoming?limit=5",
	}

	for _, endpoint := range endpoints {
		fmt.Printf("  📋 Testing: %s\n", endpoint)

		// Invalidate cache before first request
		cacheKey := "cache:" + endpoint
		fmt.Printf("    🗑️ Logging invalidation for %s to cache_test.log\n", cacheKey)
		if err := c.invalidateCache(ctx, cacheKey); err != nil {
			fmt.Printf("    ❌ Error invalidating cache: %v\n", err)
		}
		fmt.Printf("    📜 Logging Redis keys after invalidation (%s) to cache_test.log\n", endpoint)
		c.listRedisKeys(ctx, "*")

		// First request (cache miss)
		_, duration1, err := c.makeRequest(endpoint)
		if err != nil {
			fmt.Printf("    ❌ Error: %v\n", err)
			continue
		}

		// Second request (cache hit)
		_, duration2, err := c.makeRequest(endpoint)
		if err != nil {
			fmt.Printf("    ❌ Error: %v\n", err)
			continue
		}

		// Third request (cache hit)
		_, duration3, err := c.makeRequest(endpoint)
		if err != nil {
			fmt.Printf("    ❌ Error: %v\n", err)
			continue
		}

		fmt.Printf("    ⏱️  Request 1 (miss): %v\n", duration1)
		fmt.Printf("    ⏱️  Request 2 (hit):  %v\n", duration2)
		fmt.Printf("    ⏱️  Request 3 (hit):  %v\n", duration3)

		// Check if subsequent requests are faster
		if duration2 < duration1 && duration3 < duration1 {
			fmt.Printf("    ✅ Cache working - subsequent requests faster\n")
		} else {
			fmt.Printf("    ⚠️  Cache might not be working - no speed improvement\n")
		}

		fmt.Println()
	}
}

func (c *CacheTestClient) testAnalyticsCache() {
	ctx := context.Background()
	fmt.Println("📊 Testing Analytics Cache (Phase 2)")

	endpoints := []string{
		"/api/v1/analytics/admin/dashboard",
		"/api/v1/analytics/admin/events/global",
		"/api/v1/analytics/admin/tags/popularity",
	}

	for _, endpoint := range endpoints {
		fmt.Printf("  📈 Testing: %s\n", endpoint)

		// Invalidate cache before first request
		cacheKey := "cache:" + endpoint
		fmt.Printf("    🗑️ Logging invalidation for %s to cache_test.log\n", cacheKey)
		if err := c.invalidateCache(ctx, cacheKey); err != nil {
			fmt.Printf("    ❌ Error invalidating cache: %v\n", err)
		}
		fmt.Printf("    📜 Logging Redis keys after invalidation (%s) to cache_test.log\n", endpoint)
		c.listRedisKeys(ctx, "*")

		// First request (cache miss)
		_, duration1, err := c.makeRequest(endpoint)
		if err != nil {
			fmt.Printf("    ❌ Error: %v\n", err)
			continue
		}

		// Second request (cache hit)
		_, duration2, err := c.makeRequest(endpoint)
		if err != nil {
			fmt.Printf("    ❌ Error: %v\n", err)
			continue
		}

		fmt.Printf("    ⏱️  Request 1 (miss): %v\n", duration1)
		fmt.Printf("    ⏱️  Request 2 (hit):  %v\n", duration2)

		// Analytics should show significant improvement
		improvement := float64(duration1-duration2) / float64(duration1) * 100
		if improvement > 10 {
			fmt.Printf("    ✅ Strong cache improvement: %.1f%%\n", improvement)
		} else if improvement > 0 {
			fmt.Printf("    ⚠️  Moderate cache improvement: %.1f%%\n", improvement)
		} else {
			fmt.Printf("    ❌ No cache improvement detected\n")
		}

		fmt.Println()
	}
}

func (c *CacheTestClient) testSeatAvailabilityCache() {
	ctx := context.Background()
	fmt.Println("💺 Testing Seat Availability Cache (Phase 2)")

	sampleEventID := uuid.New().String()
	sampleSectionID := uuid.New().String()

	endpoints := []string{
		fmt.Sprintf("/api/v1/events/%s/sections", sampleEventID),
		fmt.Sprintf("/api/v1/sections/%s/seats/available", sampleSectionID),
	}

	for _, endpoint := range endpoints {
		fmt.Printf("  🎫 Testing: %s\n", endpoint)

		// Invalidate cache before request
		cacheKey := "cache:" + endpoint
		fmt.Printf("    🗑️ Logging invalidation for %s to cache_test.log\n", cacheKey)
		if err := c.invalidateCache(ctx, cacheKey); err != nil {
			fmt.Printf("    ❌ Error invalidating cache: %v\n", err)
		}
		fmt.Printf("    📜 Logging Redis keys after invalidation (%s) to cache_test.log\n", endpoint)
		c.listRedisKeys(ctx, "*")

		start := time.Now()
		resp, err := c.client.Get(c.baseURL + endpoint)
		duration := time.Since(start)

		if err != nil {
			fmt.Printf("    ❌ Error: %v\n", err)
		} else {
			defer resp.Body.Close()
			// Simulate setting cache
			err = c.redisClient.Set(ctx, cacheKey, "cached_response", 1*time.Hour).Err()
			if err != nil {
				log.Printf("    ❌ Error setting cache for %s: %v", cacheKey, err)
			}
			fmt.Printf("    ⏱️  Response time: %v (Status: %d)\n", duration, resp.StatusCode)
		}

		// Log Redis keys after setting cache
		fmt.Printf("    📜 Logging Redis keys after setting cache (%s) to cache_test.log\n", endpoint)
		c.listRedisKeys(ctx, "*")

		fmt.Println()
	}
}

func (c *CacheTestClient) runCacheLoadTest(endpoint string, concurrent int, requests int) {
	ctx := context.Background()
	fmt.Printf("🚀 Load Testing Cache: %s (%d concurrent, %d total requests)\n", endpoint, concurrent, requests)

	cacheKey := "cache:" + endpoint

	// Log Redis keys before load test
	fmt.Printf("    📜 Logging Redis keys before load test (%s) to cache_test.log\n", endpoint)
	c.listRedisKeys(ctx, "*")

	// Invalidate cache before load test
	fmt.Printf("    🗑️ Logging invalidation for %s to cache_test.log\n", cacheKey)
	if err := c.invalidateCache(ctx, cacheKey); err != nil {
		fmt.Printf("    ❌ Error invalidating cache: %v\n", err)
	}
	fmt.Printf("    📜 Logging Redis keys after invalidation (%s) to cache_test.log\n", endpoint)
	c.listRedisKeys(ctx, "*")

	results := make(chan time.Duration, requests)

	// Warm up cache
	c.makeRequest(endpoint)

	start := time.Now()

	// Launch concurrent requests
	for i := 0; i < concurrent; i++ {
		go func() {
			for j := 0; j < requests/concurrent; j++ {
				_, duration, _ := c.makeRequest(endpoint)
				results <- duration
			}
		}()
	}

	// Collect results
	var durations []time.Duration
	for i := 0; i < requests; i++ {
		durations = append(durations, <-results)
	}

	totalTime := time.Since(start)

	// Log Redis keys after load test
	fmt.Printf("    📜 Logging Redis keys after load test (%s) to cache_test.log\n", endpoint)
	c.listRedisKeys(ctx, "*")

	// Calculate statistics
	var total time.Duration
	min := durations[0]
	max := durations[0]

	for _, d := range durations {
		total += d
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
	}

	avg := total / time.Duration(len(durations))
	rps := float64(requests) / totalTime.Seconds()

	fmt.Printf("  📊 Results:\n")
	fmt.Printf("    Total time: %v\n", totalTime)
	fmt.Printf("    Requests/sec: %.2f\n", rps)
	fmt.Printf("    Avg response: %v\n", avg)
	fmt.Printf("    Min response: %v\n", min)
	fmt.Printf("    Max response: %v\n", max)
	fmt.Println()
}

func main() {
	fmt.Println("🧪 Evently Backend - Cache Testing Tool")
	fmt.Println("=====================================")

	// Create or open log file
	logFile, err := os.OpenFile("cache_test.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("❌ Failed to open log file: %v", err)
	}
	defer logFile.Close()

	// Configuration
	baseURL := "http://localhost:8080"

	// Create test client with log file
	client := NewCacheTestClient(baseURL, logFile)

	// Test server availability
	fmt.Println("🔗 Testing server connection...")
	_, _, err = client.makeRequest("/ping")
	if err != nil {
		log.Fatalf("❌ Server not available at %s: %v", baseURL, err)
	}
	fmt.Println("✅ Server is running")
	fmt.Println()

	// Phase 2 Cache Tests
	fmt.Println("🔍 Running Phase 2 Cache Verification Tests")
	fmt.Println("============================================")

	// Test 1: Event Listings Cache
	client.testEventListingCache()

	// Test 2: Analytics Cache
	client.testAnalyticsCache()

	// Test 3: Seat Availability Cache
	client.testSeatAvailabilityCache()

	// Load Tests
	fmt.Println("⚡ Running Load Tests")
	fmt.Println("====================")

	client.runCacheLoadTest("/api/v1/events?page=1&limit=10", 10, 100)
	client.runCacheLoadTest("/api/v1/tags/active", 5, 50)

	fmt.Println("🎉 Cache testing complete!")
}
