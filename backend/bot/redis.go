package main

import (
	"context"
	"encoding/json"
	"fmt"
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

// RedisUserCache handles Redis operations for user data caching
type RedisUserCache struct {
	client *redis.Client
	logger zerolog.Logger
	ttl    time.Duration
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
		client: client,
		logger: logger,
		ttl:    7 * 24 * time.Hour, // 7 days to match frontend cache
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
