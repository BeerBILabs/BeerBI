package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// UserCacheData represents cached user information
type UserCacheData struct {
	RealName     string    `json:"real_name"`
	ProfileImage string    `json:"profile_image"`
	CachedAt     time.Time `json:"cached_at"`
}

// CircuitBreaker protects against cascading failures when Redis is down
type CircuitBreaker struct {
	mu            sync.RWMutex
	failureCount  int
	disabledUntil time.Time
	threshold     int           // Number of failures before opening circuit
	timeout       time.Duration // How long to keep circuit open
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		threshold: threshold,
		timeout:   timeout,
	}
}

// IsOpen returns true if the circuit is open (Redis should be bypassed)
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return time.Now().Before(cb.disabledUntil)
}

// RecordFailure increments failure count and may open the circuit
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failureCount++
	if cb.failureCount >= cb.threshold {
		cb.disabledUntil = time.Now().Add(cb.timeout)
		cb.failureCount = 0 // Reset for next cycle
	}
}

// RecordSuccess resets the failure count
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failureCount = 0
}

// Reset manually resets the circuit breaker
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failureCount = 0
	cb.disabledUntil = time.Time{}
}

// RedisUserCache handles Redis operations for user data caching
type RedisUserCache struct {
	client         *redis.Client
	logger         zerolog.Logger
	ttl            time.Duration
	circuitBreaker *CircuitBreaker
}

// NewRedisUserCache creates a new Redis user cache instance
func NewRedisUserCache(addr string, logger zerolog.Logger) (*RedisUserCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     "", // No password for now
		DB:           0,  // Default DB
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MaxRetries:   3,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	logger.Info().Str("addr", addr).Msg("redis connection established")

	return &RedisUserCache{
		client:         client,
		logger:         logger,
		ttl:            7 * 24 * time.Hour, // 7 days to match frontend cache
		circuitBreaker: NewCircuitBreaker(3, 10*time.Minute),
	}, nil
}

// GetUser retrieves user data from Redis cache
func (r *RedisUserCache) GetUser(ctx context.Context, userID string) (*UserCacheData, error) {
	key := fmt.Sprintf("user:%s", userID)

	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		// Cache miss - not an error
		return nil, nil
	}
	if err != nil {
		r.logger.Warn().Str("userID", userID).Err(err).Msg("redis get error")
		return nil, err
	}

	var data UserCacheData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		r.logger.Warn().Str("userID", userID).Err(err).Msg("failed to unmarshal cached user data")
		return nil, err
	}

	r.logger.Debug().Str("userID", userID).Msg("cache hit")
	return &data, nil
}

// SetUser stores user data in Redis cache
func (r *RedisUserCache) SetUser(ctx context.Context, userID, realName, profileImage string) error {
	key := fmt.Sprintf("user:%s", userID)

	data := UserCacheData{
		RealName:     realName,
		ProfileImage: profileImage,
		CachedAt:     time.Now(),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		r.logger.Warn().Str("userID", userID).Err(err).Msg("failed to marshal user data")
		return err
	}

	if err := r.client.Set(ctx, key, jsonData, r.ttl).Err(); err != nil {
		r.logger.Warn().Str("userID", userID).Err(err).Msg("redis set error")
		return err
	}

	r.logger.Debug().Str("userID", userID).Msg("user cached")
	return nil
}

// DeleteUser removes user data from Redis cache (for invalidation)
func (r *RedisUserCache) DeleteUser(ctx context.Context, userID string) error {
	key := fmt.Sprintf("user:%s", userID)

	if err := r.client.Del(ctx, key).Err(); err != nil {
		r.logger.Warn().Str("userID", userID).Err(err).Msg("redis del error")
		return err
	}

	r.logger.Debug().Str("userID", userID).Msg("user cache invalidated")
	return nil
}

// GetUsers retrieves multiple users from Redis cache in a single pipeline
func (r *RedisUserCache) GetUsers(ctx context.Context, userIDs []string) (map[string]*UserCacheData, error) {
	if len(userIDs) == 0 {
		return make(map[string]*UserCacheData), nil
	}

	// Build keys
	keys := make([]string, len(userIDs))
	for i, userID := range userIDs {
		keys[i] = fmt.Sprintf("user:%s", userID)
	}

	// Pipeline GET commands
	pipe := r.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(keys))
	for i, key := range keys {
		cmds[i] = pipe.Get(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	// Ignore redis.Nil errors - they're expected for cache misses
	if err != nil && err != redis.Nil {
		r.logger.Warn().Err(err).Msg("redis pipeline error")
		// Continue processing partial results
	}

	// Parse results
	results := make(map[string]*UserCacheData)
	for i, cmd := range cmds {
		val, err := cmd.Result()
		if err == redis.Nil {
			// Cache miss - skip
			continue
		}
		if err != nil {
			r.logger.Warn().Str("userID", userIDs[i]).Err(err).Msg("failed to get user from pipeline")
			continue
		}

		var data UserCacheData
		if err := json.Unmarshal([]byte(val), &data); err != nil {
			r.logger.Warn().Str("userID", userIDs[i]).Err(err).Msg("failed to unmarshal cached user")
			continue
		}

		results[userIDs[i]] = &data
	}

	r.logger.Debug().Int("requested", len(userIDs)).Int("hits", len(results)).Msg("batch cache lookup")
	return results, nil
}

// SetUsers stores multiple users in Redis cache using a pipeline
func (r *RedisUserCache) SetUsers(ctx context.Context, users map[string]UserCacheData) error {
	if len(users) == 0 {
		return nil
	}

	pipe := r.client.Pipeline()
	for userID, data := range users {
		key := fmt.Sprintf("user:%s", userID)
		data.CachedAt = time.Now()

		jsonData, err := json.Marshal(data)
		if err != nil {
			r.logger.Warn().Str("userID", userID).Err(err).Msg("failed to marshal user data")
			continue
		}

		pipe.Set(ctx, key, jsonData, r.ttl)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		r.logger.Warn().Err(err).Msg("redis pipeline set error")
		return err
	}

	r.logger.Debug().Int("count", len(users)).Msg("batch users cached")
	return nil
}

// Close closes the Redis connection
func (r *RedisUserCache) Close() error {
	return r.client.Close()
}

// ============================================================================
// Beer Stats Caching
// ============================================================================

// Common date ranges for caching
const (
	RangeAllTime        = "all-time"
	RangeCurrentQuarter = "current-quarter"
	RangeLastQuarter    = "last-quarter"
	RangeCurrentMonth   = "current-month"
	RangeLastMonth      = "last-month"
	RangeLast7Days      = "last-7-days"
	RangeToday          = "today"
)

// IncrementGivenStats performs a write-through cache update for beers given by a user.
func (r *RedisUserCache) IncrementGivenStats(ctx context.Context, userID string, count int) error {
	if r.circuitBreaker.IsOpen() {
		r.logger.Debug().Msg("circuit breaker open, skipping redis write")
		return nil // Gracefully skip
	}

	ranges := r.calculateActiveRanges()

	pipe := r.client.Pipeline()
	for _, rangeKey := range ranges {
		// Increment leaderboard sorted set (score = total count)
		pipe.ZIncrBy(ctx, fmt.Sprintf("leaderboard:givers:%s", rangeKey), float64(count), userID)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		r.logger.Warn().Err(err).Str("userID", userID).Msg("failed to increment given stats")
		r.circuitBreaker.RecordFailure()
		return err
	}

	r.circuitBreaker.RecordSuccess()
	return nil
}

// IncrementReceivedStats increments beer received count for a user in common date ranges
func (r *RedisUserCache) IncrementReceivedStats(ctx context.Context, userID string, count int) error {
	if r.circuitBreaker.IsOpen() {
		r.logger.Debug().Msg("circuit breaker open, skipping redis write")
		return nil
	}

	ranges := r.calculateActiveRanges()

	pipe := r.client.Pipeline()
	for _, rangeKey := range ranges {
		// Increment leaderboard sorted set
		pipe.ZIncrBy(ctx, fmt.Sprintf("leaderboard:recipients:%s", rangeKey), float64(count), userID)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		r.logger.Warn().Err(err).Str("userID", userID).Msg("failed to increment received stats")
		r.circuitBreaker.RecordFailure()
		return err
	}

	r.circuitBreaker.RecordSuccess()
	return nil
}

// GetTopGivers retrieves top givers from Redis cache for a specific range
func (r *RedisUserCache) GetTopGivers(ctx context.Context, rangeKey string, limit int) ([]TopUserStats, error) {
	if r.circuitBreaker.IsOpen() {
		r.logger.Debug().Msg("circuit breaker open, skipping redis read")
		return nil, fmt.Errorf("circuit breaker open")
	}

	key := fmt.Sprintf("leaderboard:givers:%s", rangeKey)

	// ZREVRANGE with scores (highest first)
	results, err := r.client.ZRevRangeWithScores(ctx, key, 0, int64(limit-1)).Result()
	if err == redis.Nil {
		return nil, nil // Cache miss
	}
	if err != nil {
		r.circuitBreaker.RecordFailure()
		return nil, err
	}

	r.circuitBreaker.RecordSuccess()

	stats := make([]TopUserStats, 0, len(results))
	for _, z := range results {
		stats = append(stats, TopUserStats{
			UserID: z.Member.(string),
			Count:  int(z.Score),
		})
	}

	return stats, nil
}

// GetTopRecipients retrieves top recipients from Redis cache for a specific range
func (r *RedisUserCache) GetTopRecipients(ctx context.Context, rangeKey string, limit int) ([]TopUserStats, error) {
	if r.circuitBreaker.IsOpen() {
		r.logger.Debug().Msg("circuit breaker open, skipping redis read")
		return nil, fmt.Errorf("circuit breaker open")
	}

	key := fmt.Sprintf("leaderboard:recipients:%s", rangeKey)

	results, err := r.client.ZRevRangeWithScores(ctx, key, 0, int64(limit-1)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		r.circuitBreaker.RecordFailure()
		return nil, err
	}

	r.circuitBreaker.RecordSuccess()

	stats := make([]TopUserStats, 0, len(results))
	for _, z := range results {
		stats = append(stats, TopUserStats{
			UserID: z.Member.(string),
			Count:  int(z.Score),
		})
	}

	return stats, nil
}

// calculateActiveRanges returns which date range keys should be updated for the current date
func (r *RedisUserCache) calculateActiveRanges() []string {
	now := time.Now()
	ranges := []string{RangeAllTime, RangeToday}

	// Current quarter
	ranges = append(ranges, RangeCurrentQuarter)

	// Current month
	ranges = append(ranges, RangeCurrentMonth)

	// Last 7 days
	ranges = append(ranges, RangeLast7Days)

	// Last quarter (if we're in first month of a quarter, also update last quarter)
	if now.Month()%3 == 1 && now.Day() <= 7 {
		ranges = append(ranges, RangeLastQuarter)
	}

	// Last month (if we're in first week of month, also update last month)
	if now.Day() <= 7 {
		ranges = append(ranges, RangeLastMonth)
	}

	return ranges
}

// PopulateFromDB rebuilds Redis cache from SQLite database
func (r *RedisUserCache) PopulateFromDB(ctx context.Context, store *SQLiteStore) error {
	r.logger.Info().Msg("starting redis cache population from database")

	// Define date ranges
	now := time.Now()

	ranges := map[string]struct{ start, end time.Time }{
		RangeAllTime: {
			start: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			end:   now,
		},
		RangeToday: {
			start: now,
			end:   now,
		},
		RangeLast7Days: {
			start: now.AddDate(0, 0, -6),
			end:   now,
		},
		RangeCurrentMonth: {
			start: time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()),
			end:   now,
		},
		RangeLastMonth: {
			start: time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -1, 0),
			end:   time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, 0, -1),
		},
	}

	// Current quarter
	quarter := (int(now.Month())-1)/3 + 1
	quarterStart := time.Date(now.Year(), time.Month((quarter-1)*3+1), 1, 0, 0, 0, 0, now.Location())
	ranges[RangeCurrentQuarter] = struct{ start, end time.Time }{start: quarterStart, end: now}

	// Last quarter
	lastQuarterNum := quarter - 1
	lastQuarterYear := now.Year()
	if lastQuarterNum == 0 {
		lastQuarterNum = 4
		lastQuarterYear--
	}
	lastQuarterStart := time.Date(lastQuarterYear, time.Month((lastQuarterNum-1)*3+1), 1, 0, 0, 0, 0, now.Location())
	lastQuarterEnd := quarterStart.AddDate(0, 0, -1)
	ranges[RangeLastQuarter] = struct{ start, end time.Time }{start: lastQuarterStart, end: lastQuarterEnd}

	// Populate each range
	for rangeKey, dates := range ranges {
		if err := r.populateRange(ctx, store, rangeKey, dates.start, dates.end); err != nil {
			r.logger.Error().Err(err).Str("range", rangeKey).Msg("failed to populate range")
			// Continue with other ranges even if one fails
		}
	}

	r.logger.Info().Msg("redis cache population completed")
	return nil
}

// populateRange populates a specific date range in Redis from database
func (r *RedisUserCache) populateRange(ctx context.Context, store *SQLiteStore, rangeKey string, start, end time.Time) error {
	r.logger.Debug().Str("range", rangeKey).Time("start", start).Time("end", end).Msg("populating range")

	// Get top users from database
	topUsers, err := store.GetTopUsers(start, end, 100) // Get top 100 for each range
	if err != nil {
		return fmt.Errorf("failed to get top users: %w", err)
	}

	// Clear existing data for this range
	pipe := r.client.Pipeline()
	pipe.Del(ctx, fmt.Sprintf("leaderboard:givers:%s", rangeKey))
	pipe.Del(ctx, fmt.Sprintf("leaderboard:recipients:%s", rangeKey))

	// Populate givers
	for _, giver := range topUsers.Givers {
		pipe.ZAdd(ctx, fmt.Sprintf("leaderboard:givers:%s", rangeKey), redis.Z{
			Score:  float64(giver.Count),
			Member: giver.UserID,
		})
	}

	// Populate recipients
	for _, recipient := range topUsers.Recipients {
		pipe.ZAdd(ctx, fmt.Sprintf("leaderboard:recipients:%s", rangeKey), redis.Z{
			Score:  float64(recipient.Count),
			Member: recipient.UserID,
		})
	}

	if _, err := pipe.Exec(ctx); err != nil {
		r.circuitBreaker.RecordFailure()
		return fmt.Errorf("failed to execute pipeline: %w", err)
	}

	r.circuitBreaker.RecordSuccess()
	r.logger.Debug().Str("range", rangeKey).Int("givers", len(topUsers.Givers)).Int("recipients", len(topUsers.Recipients)).Msg("range populated")
	return nil
}

// StartSyncWorker starts a background goroutine that periodically syncs Redis from database
func (r *RedisUserCache) StartSyncWorker(ctx context.Context, store *SQLiteStore, interval time.Duration) {
	r.logger.Info().Dur("interval", interval).Msg("starting redis sync worker")

	// Initial population
	if err := r.PopulateFromDB(ctx, store); err != nil {
		r.logger.Error().Err(err).Msg("initial redis population failed")
	}

	// Periodic sync
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.logger.Debug().Msg("running periodic redis sync")
			if err := r.PopulateFromDB(ctx, store); err != nil {
				r.logger.Error().Err(err).Msg("periodic redis sync failed")
			}
		case <-ctx.Done():
			r.logger.Info().Msg("redis sync worker stopping")
			return
		}
	}
}
