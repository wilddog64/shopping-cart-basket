package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/user/shopping-cart-basket/internal/model"
)

var (
	// ErrCartNotFound is returned when a cart is not found
	ErrCartNotFound = errors.New("cart not found")
)

// CartRepository defines the interface for cart data access
type CartRepository interface {
	Get(ctx context.Context, customerID string) (*model.Cart, error)
	Save(ctx context.Context, cart *model.Cart) error
	Delete(ctx context.Context, customerID string) error
	Exists(ctx context.Context, customerID string) (bool, error)
}

// RedisCartRepository implements CartRepository using Redis
type RedisCartRepository struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisCartRepository creates a new Redis-based cart repository
func NewRedisCartRepository(addr, password string, db int, ttl time.Duration) (*RedisCartRepository, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCartRepository{
		client: client,
		ttl:    ttl,
	}, nil
}

// cartKey returns the Redis key for a customer's cart
func (r *RedisCartRepository) cartKey(customerID string) string {
	return fmt.Sprintf("cart:%s", customerID)
}

// Get retrieves a cart by customer ID
func (r *RedisCartRepository) Get(ctx context.Context, customerID string) (*model.Cart, error) {
	key := r.cartKey(customerID)

	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrCartNotFound
		}
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	var cart model.Cart
	if err := json.Unmarshal(data, &cart); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cart: %w", err)
	}

	return &cart, nil
}

// Save saves or updates a cart
func (r *RedisCartRepository) Save(ctx context.Context, cart *model.Cart) error {
	key := r.cartKey(cart.CustomerID)

	data, err := json.Marshal(cart)
	if err != nil {
		return fmt.Errorf("failed to marshal cart: %w", err)
	}

	// Calculate remaining TTL based on cart expiration
	ttl := time.Until(cart.ExpiresAt)
	if ttl <= 0 {
		ttl = r.ttl
		cart.ExpiresAt = time.Now().Add(ttl)
	}

	if err := r.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to save cart: %w", err)
	}

	return nil
}

// Delete removes a cart
func (r *RedisCartRepository) Delete(ctx context.Context, customerID string) error {
	key := r.cartKey(customerID)

	result, err := r.client.Del(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to delete cart: %w", err)
	}

	if result == 0 {
		return ErrCartNotFound
	}

	return nil
}

// Exists checks if a cart exists
func (r *RedisCartRepository) Exists(ctx context.Context, customerID string) (bool, error) {
	key := r.cartKey(customerID)

	result, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check cart existence: %w", err)
	}

	return result > 0, nil
}

// Close closes the Redis connection
func (r *RedisCartRepository) Close() error {
	return r.client.Close()
}

// Client returns the underlying Redis client for health checks
func (r *RedisCartRepository) Client() *redis.Client {
	return r.client
}
