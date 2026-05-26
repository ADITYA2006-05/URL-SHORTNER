package repository

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRepo handles all Redis operations
type RedisRepo struct {
	client *redis.Client
}

// NewRedisRepo creates a new Redis repository
func NewRedisRepo(addr, password string, db int) (*RedisRepo, error) {
	opts := &redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	}

	// Dynamically enable TLS/SSL for secure cloud hosts (like Upstash)
	if strings.Contains(addr, "upstash.io") || strings.Contains(addr, "rediss://") {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("unable to connect to Redis: %w", err)
	}

	return &RedisRepo{client: client}, nil
}

// Close closes the Redis connection
func (r *RedisRepo) Close() error {
	return r.client.Close()
}

// CacheURL caches a URL mapping with optional TTL
func (r *RedisRepo) CacheURL(ctx context.Context, shortCode, originalURL string, expiresAt *time.Time) error {
	key := "url:" + shortCode

	var ttl time.Duration
	if expiresAt != nil {
		ttl = time.Until(*expiresAt)
		if ttl <= 0 {
			return fmt.Errorf("URL has already expired")
		}
	} else {
		ttl = 24 * time.Hour // Default cache for 24 hours
	}

	return r.client.Set(ctx, key, originalURL, ttl).Err()
}

// GetCachedURL retrieves a cached URL
func (r *RedisRepo) GetCachedURL(ctx context.Context, shortCode string) (string, error) {
	key := "url:" + shortCode
	return r.client.Get(ctx, key).Result()
}

// DeleteCachedURL removes a URL from cache
func (r *RedisRepo) DeleteCachedURL(ctx context.Context, shortCode string) error {
	key := "url:" + shortCode
	return r.client.Del(ctx, key).Err()
}

// IncrementClickCount increments the real-time click counter
func (r *RedisRepo) IncrementClickCount(ctx context.Context, shortCode string) (int64, error) {
	key := "clicks:" + shortCode
	return r.client.Incr(ctx, key).Result()
}

// GetClickCount gets the cached click count
func (r *RedisRepo) GetClickCount(ctx context.Context, shortCode string) (int64, error) {
	key := "clicks:" + shortCode
	val, err := r.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

// RateLimitCheck checks and increments the rate limit counter for an IP
// Returns true if the request is allowed, false if rate limited
func (r *RedisRepo) RateLimitCheck(ctx context.Context, ip string, maxRequests int, window time.Duration) (bool, error) {
	key := "ratelimit:" + ip

	// Use a pipeline for atomic operations
	pipe := r.client.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return true, err // Allow on error
	}

	count := incr.Val()
	return count <= int64(maxRequests), nil
}

// GetRateLimitRemaining returns remaining requests for an IP
func (r *RedisRepo) GetRateLimitRemaining(ctx context.Context, ip string, maxRequests int) (int, error) {
	key := "ratelimit:" + ip
	count, err := r.client.Get(ctx, key).Int()
	if err == redis.Nil {
		return maxRequests, nil
	}
	if err != nil {
		return maxRequests, err
	}
	remaining := maxRequests - count
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}
