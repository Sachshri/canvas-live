package redis

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisClient struct holds the client connection
type RedisClient struct {
	Client *redis.Client
}

// NewRedisClient creates and tests the connection to Redis
func NewRedisClient(addr string) *RedisClient {
	// Initialize the client connection
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr, // e.g., "redis:6379" from Docker Compose
		Password: "",   // No password by default
		DB:       0,    // Default DB
	})

	// Use a context to test the connection (Ping)
	ctx := context.Background()
	status := rdb.Ping(ctx)

	if status.Err() != nil {
		log.Fatalf("Failed to connect to Redis at %s: %v", addr, status.Err())
	}

	fmt.Printf("Successfully connected to Redis at %s\n", addr)
	return &RedisClient{
		Client: rdb,
	}
}

// Example method to set a value (e.g., setting an exclusive lock)
func (r *RedisClient) SetExclusiveLock(ctx context.Context, objectId string, userId string, duration time.Duration) error {
	// SET key value NX EX duration
	// NX: Only set the key if it does NOT EXIST
	// EX: Set an expiration time

	// If the key (lock) is already set, this command returns false (no modification)
	ok, err := r.Client.SetNX(ctx, objectId, userId, duration).Result()

	if err != nil {
		return fmt.Errorf("redis SETNX failed: %w", err)
	}

	if !ok {
		// Lock failed because the key already exists
		return fmt.Errorf("element %s is already locked by another user", objectId)
	}

	return nil // Lock acquired successfully
}

// Example method to release a value (e.g., releasing an exclusive lock)
func (r *RedisClient) ReleaseLock(ctx context.Context, objectId string) (bool, error) {
	// DEL key
	// This command removes the lock
	count, err := r.Client.Del(ctx, objectId).Result()
	if err != nil {
		return false, fmt.Errorf("redis DEL failed: %w", err)
	}

	return count > 0, err
}
