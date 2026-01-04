//go:build integration

package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/shopping-cart-basket/internal/model"
)

func getRedisAddr() string {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	return addr
}

func getRedisPassword() string {
	return os.Getenv("REDIS_PASSWORD")
}

func setupTestRepository(t *testing.T) *RedisCartRepository {
	repo, err := NewRedisCartRepository(getRedisAddr(), getRedisPassword(), 0, 1*time.Hour)
	require.NoError(t, err, "Failed to connect to Redis. Ensure Redis is running on %s", getRedisAddr())
	return repo
}

func cleanupCart(t *testing.T, repo *RedisCartRepository, customerID string) {
	ctx := context.Background()
	_ = repo.Delete(ctx, customerID) // Ignore error, cart may not exist
}

func TestRedisCartRepository_Integration_SaveAndGet(t *testing.T) {
	repo := setupTestRepository(t)
	defer repo.Close()

	customerID := "integration-test-customer-1"
	defer cleanupCart(t, repo, customerID)

	ctx := context.Background()

	// Create and save a cart
	cart := model.NewCart(customerID, 1*time.Hour)
	cart.AddItem(model.CartItem{
		ProductID: "prod-001",
		Name:      "Test Product",
		Quantity:  2,
		UnitPrice: 29.99,
	})

	err := repo.Save(ctx, cart)
	require.NoError(t, err)

	// Retrieve the cart
	retrieved, err := repo.Get(ctx, customerID)
	require.NoError(t, err)

	assert.Equal(t, customerID, retrieved.CustomerID)
	assert.Len(t, retrieved.Items, 1)
	assert.Equal(t, "prod-001", retrieved.Items[0].ProductID)
	assert.Equal(t, "Test Product", retrieved.Items[0].Name)
	assert.Equal(t, 2, retrieved.Items[0].Quantity)
	assert.Equal(t, 29.99, retrieved.Items[0].UnitPrice)
	assert.Equal(t, 59.98, retrieved.TotalAmount)
}

func TestRedisCartRepository_Integration_Update(t *testing.T) {
	repo := setupTestRepository(t)
	defer repo.Close()

	customerID := "integration-test-customer-2"
	defer cleanupCart(t, repo, customerID)

	ctx := context.Background()

	// Create and save initial cart
	cart := model.NewCart(customerID, 1*time.Hour)
	cart.AddItem(model.CartItem{
		ProductID: "prod-001",
		Name:      "Product One",
		Quantity:  1,
		UnitPrice: 10.00,
	})

	err := repo.Save(ctx, cart)
	require.NoError(t, err)

	// Update the cart
	cart.AddItem(model.CartItem{
		ProductID: "prod-002",
		Name:      "Product Two",
		Quantity:  3,
		UnitPrice: 15.00,
	})

	err = repo.Save(ctx, cart)
	require.NoError(t, err)

	// Verify update
	retrieved, err := repo.Get(ctx, customerID)
	require.NoError(t, err)

	assert.Len(t, retrieved.Items, 2)
	assert.Equal(t, 55.00, retrieved.TotalAmount) // 10 + (3 * 15)
}

func TestRedisCartRepository_Integration_Delete(t *testing.T) {
	repo := setupTestRepository(t)
	defer repo.Close()

	customerID := "integration-test-customer-3"

	ctx := context.Background()

	// Create and save a cart
	cart := model.NewCart(customerID, 1*time.Hour)
	cart.AddItem(model.CartItem{
		ProductID: "prod-001",
		Name:      "Test Product",
		Quantity:  1,
		UnitPrice: 10.00,
	})

	err := repo.Save(ctx, cart)
	require.NoError(t, err)

	// Verify it exists
	exists, err := repo.Exists(ctx, customerID)
	require.NoError(t, err)
	assert.True(t, exists)

	// Delete the cart
	err = repo.Delete(ctx, customerID)
	require.NoError(t, err)

	// Verify it's gone
	exists, err = repo.Exists(ctx, customerID)
	require.NoError(t, err)
	assert.False(t, exists)

	// Get should return ErrCartNotFound
	_, err = repo.Get(ctx, customerID)
	assert.ErrorIs(t, err, ErrCartNotFound)
}

func TestRedisCartRepository_Integration_GetNotFound(t *testing.T) {
	repo := setupTestRepository(t)
	defer repo.Close()

	ctx := context.Background()

	_, err := repo.Get(ctx, "non-existent-customer")
	assert.ErrorIs(t, err, ErrCartNotFound)
}

func TestRedisCartRepository_Integration_DeleteNotFound(t *testing.T) {
	repo := setupTestRepository(t)
	defer repo.Close()

	ctx := context.Background()

	err := repo.Delete(ctx, "non-existent-customer")
	assert.ErrorIs(t, err, ErrCartNotFound)
}

func TestRedisCartRepository_Integration_Exists(t *testing.T) {
	repo := setupTestRepository(t)
	defer repo.Close()

	customerID := "integration-test-customer-4"
	defer cleanupCart(t, repo, customerID)

	ctx := context.Background()

	// Should not exist initially
	exists, err := repo.Exists(ctx, customerID)
	require.NoError(t, err)
	assert.False(t, exists)

	// Create cart
	cart := model.NewCart(customerID, 1*time.Hour)
	err = repo.Save(ctx, cart)
	require.NoError(t, err)

	// Should exist now
	exists, err = repo.Exists(ctx, customerID)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestRedisCartRepository_Integration_TTL(t *testing.T) {
	repo, err := NewRedisCartRepository(getRedisAddr(), getRedisPassword(), 0, 2*time.Second)
	require.NoError(t, err)
	defer repo.Close()

	customerID := "integration-test-customer-ttl"
	defer cleanupCart(t, repo, customerID)

	ctx := context.Background()

	// Create cart with short TTL
	cart := model.NewCart(customerID, 2*time.Second)
	err = repo.Save(ctx, cart)
	require.NoError(t, err)

	// Should exist immediately
	exists, err := repo.Exists(ctx, customerID)
	require.NoError(t, err)
	assert.True(t, exists)

	// Wait for TTL to expire
	time.Sleep(3 * time.Second)

	// Should be gone
	exists, err = repo.Exists(ctx, customerID)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRedisCartRepository_Integration_MultipleItems(t *testing.T) {
	repo := setupTestRepository(t)
	defer repo.Close()

	customerID := "integration-test-customer-5"
	defer cleanupCart(t, repo, customerID)

	ctx := context.Background()

	// Create cart with multiple items
	cart := model.NewCart(customerID, 1*time.Hour)

	items := []model.CartItem{
		{ProductID: "prod-001", Name: "Widget", Quantity: 2, UnitPrice: 10.00},
		{ProductID: "prod-002", Name: "Gadget", Quantity: 1, UnitPrice: 25.00},
		{ProductID: "prod-003", Name: "Gizmo", Quantity: 5, UnitPrice: 5.00},
	}

	for _, item := range items {
		cart.AddItem(item)
	}

	err := repo.Save(ctx, cart)
	require.NoError(t, err)

	// Retrieve and verify
	retrieved, err := repo.Get(ctx, customerID)
	require.NoError(t, err)

	assert.Len(t, retrieved.Items, 3)
	assert.Equal(t, 70.00, retrieved.TotalAmount) // (2*10) + (1*25) + (5*5)
}

func TestRedisCartRepository_Integration_UpdateQuantity(t *testing.T) {
	repo := setupTestRepository(t)
	defer repo.Close()

	customerID := "integration-test-customer-6"
	defer cleanupCart(t, repo, customerID)

	ctx := context.Background()

	// Create cart
	cart := model.NewCart(customerID, 1*time.Hour)
	cart.AddItem(model.CartItem{
		ProductID: "prod-001",
		Name:      "Test Product",
		Quantity:  2,
		UnitPrice: 10.00,
	})

	err := repo.Save(ctx, cart)
	require.NoError(t, err)

	// Get, update quantity, save
	retrieved, err := repo.Get(ctx, customerID)
	require.NoError(t, err)

	itemID := retrieved.Items[0].ID
	retrieved.UpdateItemQuantity(itemID, 5)

	err = repo.Save(ctx, retrieved)
	require.NoError(t, err)

	// Verify
	final, err := repo.Get(ctx, customerID)
	require.NoError(t, err)

	assert.Equal(t, 5, final.Items[0].Quantity)
	assert.Equal(t, 50.00, final.TotalAmount)
}

func TestRedisCartRepository_Integration_ClearCart(t *testing.T) {
	repo := setupTestRepository(t)
	defer repo.Close()

	customerID := "integration-test-customer-7"
	defer cleanupCart(t, repo, customerID)

	ctx := context.Background()

	// Create cart with items
	cart := model.NewCart(customerID, 1*time.Hour)
	cart.AddItem(model.CartItem{
		ProductID: "prod-001",
		Name:      "Test Product",
		Quantity:  3,
		UnitPrice: 15.00,
	})

	err := repo.Save(ctx, cart)
	require.NoError(t, err)

	// Clear and save
	retrieved, err := repo.Get(ctx, customerID)
	require.NoError(t, err)

	retrieved.Clear()
	err = repo.Save(ctx, retrieved)
	require.NoError(t, err)

	// Verify
	final, err := repo.Get(ctx, customerID)
	require.NoError(t, err)

	assert.Empty(t, final.Items)
	assert.Equal(t, 0.00, final.TotalAmount)
}
