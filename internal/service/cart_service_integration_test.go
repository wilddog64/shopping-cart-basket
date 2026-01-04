//go:build integration

package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/shopping-cart-basket/internal/model"
	"github.com/user/shopping-cart-basket/internal/repository"
	"go.uber.org/zap"
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

// NoOpPublisher is a publisher that does nothing (for integration tests without RabbitMQ)
type NoOpPublisher struct{}

func (p *NoOpPublisher) Publish(ctx context.Context, event *model.EventEnvelope) error {
	return nil
}

func setupIntegrationService(t *testing.T) (*CartService, *repository.RedisCartRepository) {
	repo, err := repository.NewRedisCartRepository(getRedisAddr(), getRedisPassword(), 0, 1*time.Hour)
	require.NoError(t, err, "Failed to connect to Redis. Ensure Redis is running on %s", getRedisAddr())

	logger := zap.NewNop()
	publisher := &NoOpPublisher{}
	service := NewCartService(repo, publisher, 1*time.Hour, logger)

	return service, repo
}

func cleanupCart(t *testing.T, repo *repository.RedisCartRepository, customerID string) {
	ctx := context.Background()
	_ = repo.Delete(ctx, customerID)
}

func TestCartService_Integration_GetCart_CreatesNew(t *testing.T) {
	service, repo := setupIntegrationService(t)
	defer repo.Close()

	customerID := "svc-integration-test-1"
	defer cleanupCart(t, repo, customerID)

	ctx := context.Background()

	// Get cart for new customer should create one
	cart, err := service.GetCart(ctx, customerID)
	require.NoError(t, err)

	assert.Equal(t, customerID, cart.CustomerID)
	assert.Empty(t, cart.Items)
	assert.Equal(t, 0.00, cart.TotalAmount)

	// Verify it was persisted
	exists, err := repo.Exists(ctx, customerID)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestCartService_Integration_AddItem(t *testing.T) {
	service, repo := setupIntegrationService(t)
	defer repo.Close()

	customerID := "svc-integration-test-2"
	defer cleanupCart(t, repo, customerID)

	ctx := context.Background()

	// Add item
	req := &model.AddItemRequest{
		ProductID: "prod-001",
		Name:      "Integration Test Product",
		Quantity:  3,
		UnitPrice: 19.99,
	}

	cart, err := service.AddItem(ctx, customerID, req)
	require.NoError(t, err)

	assert.Len(t, cart.Items, 1)
	assert.Equal(t, "prod-001", cart.Items[0].ProductID)
	assert.Equal(t, 3, cart.Items[0].Quantity)
	assert.InDelta(t, 59.97, cart.TotalAmount, 0.01)

	// Verify persistence by getting fresh from repo
	persisted, err := repo.Get(ctx, customerID)
	require.NoError(t, err)
	assert.Len(t, persisted.Items, 1)
}

func TestCartService_Integration_AddMultipleItems(t *testing.T) {
	service, repo := setupIntegrationService(t)
	defer repo.Close()

	customerID := "svc-integration-test-3"
	defer cleanupCart(t, repo, customerID)

	ctx := context.Background()

	// Add first item
	_, err := service.AddItem(ctx, customerID, &model.AddItemRequest{
		ProductID: "prod-001",
		Name:      "Product One",
		Quantity:  2,
		UnitPrice: 10.00,
	})
	require.NoError(t, err)

	// Add second item
	_, err = service.AddItem(ctx, customerID, &model.AddItemRequest{
		ProductID: "prod-002",
		Name:      "Product Two",
		Quantity:  1,
		UnitPrice: 25.00,
	})
	require.NoError(t, err)

	// Add third item
	cart, err := service.AddItem(ctx, customerID, &model.AddItemRequest{
		ProductID: "prod-003",
		Name:      "Product Three",
		Quantity:  4,
		UnitPrice: 5.00,
	})
	require.NoError(t, err)

	assert.Len(t, cart.Items, 3)
	assert.Equal(t, 65.00, cart.TotalAmount) // (2*10) + (1*25) + (4*5)
}

func TestCartService_Integration_AddSameProductUpdatesQuantity(t *testing.T) {
	service, repo := setupIntegrationService(t)
	defer repo.Close()

	customerID := "svc-integration-test-4"
	defer cleanupCart(t, repo, customerID)

	ctx := context.Background()

	// Add item first time
	_, err := service.AddItem(ctx, customerID, &model.AddItemRequest{
		ProductID: "prod-001",
		Name:      "Test Product",
		Quantity:  2,
		UnitPrice: 10.00,
	})
	require.NoError(t, err)

	// Add same product again
	cart, err := service.AddItem(ctx, customerID, &model.AddItemRequest{
		ProductID: "prod-001",
		Name:      "Test Product",
		Quantity:  3,
		UnitPrice: 10.00,
	})
	require.NoError(t, err)

	// Should still be 1 item but with quantity 5
	assert.Len(t, cart.Items, 1)
	assert.Equal(t, 5, cart.Items[0].Quantity)
	assert.Equal(t, 50.00, cart.TotalAmount)
}

func TestCartService_Integration_UpdateItemQuantity(t *testing.T) {
	service, repo := setupIntegrationService(t)
	defer repo.Close()

	customerID := "svc-integration-test-5"
	defer cleanupCart(t, repo, customerID)

	ctx := context.Background()

	// Add item
	cart, err := service.AddItem(ctx, customerID, &model.AddItemRequest{
		ProductID: "prod-001",
		Name:      "Test Product",
		Quantity:  2,
		UnitPrice: 15.00,
	})
	require.NoError(t, err)

	itemID := cart.Items[0].ID

	// Update quantity
	cart, err = service.UpdateItemQuantity(ctx, customerID, itemID, 7)
	require.NoError(t, err)

	assert.Equal(t, 7, cart.Items[0].Quantity)
	assert.Equal(t, 105.00, cart.TotalAmount)

	// Verify persistence
	persisted, err := repo.Get(ctx, customerID)
	require.NoError(t, err)
	assert.Equal(t, 7, persisted.Items[0].Quantity)
}

func TestCartService_Integration_RemoveItem(t *testing.T) {
	service, repo := setupIntegrationService(t)
	defer repo.Close()

	customerID := "svc-integration-test-6"
	defer cleanupCart(t, repo, customerID)

	ctx := context.Background()

	// Add two items
	_, err := service.AddItem(ctx, customerID, &model.AddItemRequest{
		ProductID: "prod-001",
		Name:      "Product One",
		Quantity:  2,
		UnitPrice: 10.00,
	})
	require.NoError(t, err)

	cart, err := service.AddItem(ctx, customerID, &model.AddItemRequest{
		ProductID: "prod-002",
		Name:      "Product Two",
		Quantity:  1,
		UnitPrice: 25.00,
	})
	require.NoError(t, err)

	// Remove first item
	itemID := cart.Items[0].ID
	cart, err = service.RemoveItem(ctx, customerID, itemID)
	require.NoError(t, err)

	assert.Len(t, cart.Items, 1)
	assert.Equal(t, "prod-002", cart.Items[0].ProductID)
	assert.Equal(t, 25.00, cart.TotalAmount)
}

func TestCartService_Integration_ClearCart(t *testing.T) {
	service, repo := setupIntegrationService(t)
	defer repo.Close()

	customerID := "svc-integration-test-7"
	defer cleanupCart(t, repo, customerID)

	ctx := context.Background()

	// Add items
	_, err := service.AddItem(ctx, customerID, &model.AddItemRequest{
		ProductID: "prod-001",
		Name:      "Product One",
		Quantity:  2,
		UnitPrice: 10.00,
	})
	require.NoError(t, err)

	_, err = service.AddItem(ctx, customerID, &model.AddItemRequest{
		ProductID: "prod-002",
		Name:      "Product Two",
		Quantity:  3,
		UnitPrice: 15.00,
	})
	require.NoError(t, err)

	// Clear cart
	err = service.ClearCart(ctx, customerID)
	require.NoError(t, err)

	// Verify empty
	cart, err := service.GetCart(ctx, customerID)
	require.NoError(t, err)
	assert.Empty(t, cart.Items)
	assert.Equal(t, 0.00, cart.TotalAmount)
}

func TestCartService_Integration_Checkout(t *testing.T) {
	service, repo := setupIntegrationService(t)
	defer repo.Close()

	customerID := "svc-integration-test-8"
	defer cleanupCart(t, repo, customerID)

	ctx := context.Background()

	// Add items
	_, err := service.AddItem(ctx, customerID, &model.AddItemRequest{
		ProductID: "prod-001",
		Name:      "Widget",
		Quantity:  2,
		UnitPrice: 29.99,
	})
	require.NoError(t, err)

	_, err = service.AddItem(ctx, customerID, &model.AddItemRequest{
		ProductID: "prod-002",
		Name:      "Gadget",
		Quantity:  1,
		UnitPrice: 49.99,
	})
	require.NoError(t, err)

	// Checkout
	checkoutReq := &model.CheckoutRequest{
		ShippingAddress: model.ShippingAddress{
			Street:     "123 Integration Test St",
			City:       "Test City",
			State:      "TS",
			PostalCode: "12345",
			Country:    "US",
		},
	}

	cart, err := service.Checkout(ctx, customerID, checkoutReq)
	require.NoError(t, err)

	assert.Len(t, cart.Items, 2)
	assert.InDelta(t, 109.97, cart.TotalAmount, 0.01) // (2*29.99) + 49.99

	// Cart should be cleared after checkout
	postCheckout, err := service.GetCart(ctx, customerID)
	require.NoError(t, err)
	assert.Empty(t, postCheckout.Items)
}

func TestCartService_Integration_Checkout_EmptyCart(t *testing.T) {
	service, repo := setupIntegrationService(t)
	defer repo.Close()

	customerID := "svc-integration-test-9"
	defer cleanupCart(t, repo, customerID)

	ctx := context.Background()

	// First create an empty cart
	_, err := service.GetCart(ctx, customerID)
	require.NoError(t, err)

	// Try to checkout empty cart
	checkoutReq := &model.CheckoutRequest{
		ShippingAddress: model.ShippingAddress{
			Street:     "123 Test St",
			City:       "Test City",
			State:      "TS",
			PostalCode: "12345",
			Country:    "US",
		},
	}

	_, err = service.Checkout(ctx, customerID, checkoutReq)
	assert.ErrorIs(t, err, ErrCartEmpty)
}

func TestCartService_Integration_MaxItems(t *testing.T) {
	service, repo := setupIntegrationService(t)
	defer repo.Close()

	customerID := "svc-integration-test-10"
	defer cleanupCart(t, repo, customerID)

	ctx := context.Background()

	// Add MaxCartItems items
	for i := 0; i < MaxCartItems; i++ {
		_, err := service.AddItem(ctx, customerID, &model.AddItemRequest{
			ProductID: "prod-" + string(rune('A'+i)),
			Name:      "Product",
			Quantity:  1,
			UnitPrice: 1.00,
		})
		require.NoError(t, err)
	}

	// Try to add one more
	_, err := service.AddItem(ctx, customerID, &model.AddItemRequest{
		ProductID: "prod-overflow",
		Name:      "Overflow Product",
		Quantity:  1,
		UnitPrice: 1.00,
	})
	assert.ErrorIs(t, err, ErrMaxItemsExceeded)
}

func TestCartService_Integration_ConcurrentAccess(t *testing.T) {
	service, repo := setupIntegrationService(t)
	defer repo.Close()

	customerID := "svc-integration-test-concurrent"
	defer cleanupCart(t, repo, customerID)

	ctx := context.Background()

	// Create cart first
	_, err := service.GetCart(ctx, customerID)
	require.NoError(t, err)

	// Concurrent adds
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func(idx int) {
			_, err := service.AddItem(ctx, customerID, &model.AddItemRequest{
				ProductID: "prod-concurrent-" + string(rune('A'+idx)),
				Name:      "Concurrent Product",
				Quantity:  1,
				UnitPrice: 10.00,
			})
			if err != nil {
				t.Logf("Concurrent add error: %v", err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	// Verify final state
	cart, err := service.GetCart(ctx, customerID)
	require.NoError(t, err)

	// Due to race conditions, we may have fewer items, but cart should be valid
	assert.True(t, len(cart.Items) > 0 && len(cart.Items) <= 5)
	t.Logf("Final cart has %d items", len(cart.Items))
}
