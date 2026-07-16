package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/user/shopping-cart-basket/internal/model"
	"github.com/user/shopping-cart-basket/internal/repository"
	"go.uber.org/zap"
)

// MockCartRepository is a mock implementation of CartRepository
type MockCartRepository struct {
	mock.Mock
}

func (m *MockCartRepository) Get(ctx context.Context, customerID string) (*model.Cart, error) {
	args := m.Called(ctx, customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Cart), args.Error(1)
}

func (m *MockCartRepository) Save(ctx context.Context, cart *model.Cart) error {
	args := m.Called(ctx, cart)
	return args.Error(0)
}

func (m *MockCartRepository) Delete(ctx context.Context, customerID string) error {
	args := m.Called(ctx, customerID)
	return args.Error(0)
}

func (m *MockCartRepository) Exists(ctx context.Context, customerID string) (bool, error) {
	args := m.Called(ctx, customerID)
	return args.Bool(0), args.Error(1)
}

// MockEventPublisher is a mock implementation of EventPublisher
type MockEventPublisher struct {
	mock.Mock
	events []*model.EventEnvelope
}

func (m *MockEventPublisher) Publish(ctx context.Context, event *model.EventEnvelope) error {
	m.events = append(m.events, event)
	args := m.Called(ctx, event)
	return args.Error(0)
}

func TestCartService_GetCart_ExistingCart(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockCartRepository)
	mockPublisher := new(MockEventPublisher)
	logger := zap.NewNop()

	existingCart := model.NewCart("customer-123", 24*time.Hour)
	existingCart.AddItem(model.CartItem{
		ProductID: "prod-1",
		Name:      "Test Product",
		Quantity:  2,
		UnitPrice: 10.00,
	})

	mockRepo.On("Get", ctx, "customer-123").Return(existingCart, nil)

	service := NewCartService(mockRepo, mockPublisher, 24*time.Hour, 72*time.Hour, logger)
	cart, err := service.GetCart(ctx, "customer-123")

	require.NoError(t, err)
	assert.Equal(t, "customer-123", cart.CustomerID)
	assert.Len(t, cart.Items, 1)
	assert.Equal(t, 20.00, cart.TotalAmount)

	mockRepo.AssertExpectations(t)
}

func TestCartService_GetCart_NewCart(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockCartRepository)
	mockPublisher := new(MockEventPublisher)
	logger := zap.NewNop()

	mockRepo.On("Get", ctx, "new-customer").Return(nil, repository.ErrCartNotFound)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*model.Cart")).Return(nil)
	mockPublisher.On("Publish", ctx, mock.AnythingOfType("*model.EventEnvelope")).Return(nil)

	service := NewCartService(mockRepo, mockPublisher, 24*time.Hour, 72*time.Hour, logger)
	cart, err := service.GetCart(ctx, "new-customer")

	require.NoError(t, err)
	assert.Equal(t, "new-customer", cart.CustomerID)
	assert.Empty(t, cart.Items)

	mockRepo.AssertExpectations(t)
	mockPublisher.AssertExpectations(t)
}

func TestCartService_AddItem(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockCartRepository)
	mockPublisher := new(MockEventPublisher)
	logger := zap.NewNop()

	existingCart := model.NewCart("customer-123", 24*time.Hour)

	mockRepo.On("Get", ctx, "customer-123").Return(existingCart, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*model.Cart")).Return(nil)
	mockPublisher.On("Publish", ctx, mock.AnythingOfType("*model.EventEnvelope")).Return(nil)

	service := NewCartService(mockRepo, mockPublisher, 24*time.Hour, 72*time.Hour, logger)

	req := &model.AddItemRequest{
		ProductID: "prod-1",
		Name:      "Test Product",
		Quantity:  3,
		UnitPrice: 15.00,
	}

	cart, err := service.AddItem(ctx, "customer-123", req)

	require.NoError(t, err)
	assert.Len(t, cart.Items, 1)
	assert.Equal(t, "prod-1", cart.Items[0].ProductID)
	assert.Equal(t, 3, cart.Items[0].Quantity)
	assert.Equal(t, 45.00, cart.TotalAmount)

	mockRepo.AssertExpectations(t)
	mockPublisher.AssertExpectations(t)
}

func TestCartService_AddItem_MaxItemsExceeded(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockCartRepository)
	mockPublisher := new(MockEventPublisher)
	logger := zap.NewNop()

	existingCart := model.NewCart("customer-123", 24*time.Hour)
	for i := 0; i < MaxCartItems; i++ {
		existingCart.AddItem(model.CartItem{
			ProductID: "prod-" + string(rune(i)),
			Name:      "Product",
			Quantity:  1,
			UnitPrice: 1.00,
		})
	}

	mockRepo.On("Get", ctx, "customer-123").Return(existingCart, nil)

	service := NewCartService(mockRepo, mockPublisher, 24*time.Hour, 72*time.Hour, logger)

	req := &model.AddItemRequest{
		ProductID: "prod-new",
		Name:      "New Product",
		Quantity:  1,
		UnitPrice: 10.00,
	}

	_, err := service.AddItem(ctx, "customer-123", req)

	assert.ErrorIs(t, err, ErrMaxItemsExceeded)
	mockRepo.AssertExpectations(t)
}

func TestCartService_UpdateItemQuantity(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockCartRepository)
	mockPublisher := new(MockEventPublisher)
	logger := zap.NewNop()

	existingCart := model.NewCart("customer-123", 24*time.Hour)
	existingCart.AddItem(model.CartItem{
		ProductID: "prod-1",
		Name:      "Test Product",
		Quantity:  2,
		UnitPrice: 10.00,
	})
	itemID := existingCart.Items[0].ID

	mockRepo.On("Get", ctx, "customer-123").Return(existingCart, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*model.Cart")).Return(nil)
	mockPublisher.On("Publish", ctx, mock.AnythingOfType("*model.EventEnvelope")).Return(nil)

	service := NewCartService(mockRepo, mockPublisher, 24*time.Hour, 72*time.Hour, logger)

	cart, err := service.UpdateItemQuantity(ctx, "customer-123", itemID, 5)

	require.NoError(t, err)
	assert.Equal(t, 5, cart.Items[0].Quantity)
	assert.Equal(t, 50.00, cart.TotalAmount)

	mockRepo.AssertExpectations(t)
}

func TestCartService_UpdateItemQuantity_NotFound(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockCartRepository)
	mockPublisher := new(MockEventPublisher)
	logger := zap.NewNop()

	existingCart := model.NewCart("customer-123", 24*time.Hour)

	mockRepo.On("Get", ctx, "customer-123").Return(existingCart, nil)

	service := NewCartService(mockRepo, mockPublisher, 24*time.Hour, 72*time.Hour, logger)

	_, err := service.UpdateItemQuantity(ctx, "customer-123", "non-existent", 5)

	assert.ErrorIs(t, err, ErrItemNotFound)
	mockRepo.AssertExpectations(t)
}

func TestCartService_RemoveItem(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockCartRepository)
	mockPublisher := new(MockEventPublisher)
	logger := zap.NewNop()

	existingCart := model.NewCart("customer-123", 24*time.Hour)
	existingCart.AddItem(model.CartItem{
		ProductID: "prod-1",
		Name:      "Test Product",
		Quantity:  2,
		UnitPrice: 10.00,
	})
	itemID := existingCart.Items[0].ID

	mockRepo.On("Get", ctx, "customer-123").Return(existingCart, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*model.Cart")).Return(nil)
	mockPublisher.On("Publish", ctx, mock.AnythingOfType("*model.EventEnvelope")).Return(nil)

	service := NewCartService(mockRepo, mockPublisher, 24*time.Hour, 72*time.Hour, logger)

	cart, err := service.RemoveItem(ctx, "customer-123", itemID)

	require.NoError(t, err)
	assert.Empty(t, cart.Items)
	assert.Equal(t, 0.00, cart.TotalAmount)

	mockRepo.AssertExpectations(t)
}

func TestCartService_ClearCart(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockCartRepository)
	mockPublisher := new(MockEventPublisher)
	logger := zap.NewNop()

	existingCart := model.NewCart("customer-123", 24*time.Hour)
	existingCart.AddItem(model.CartItem{
		ProductID: "prod-1",
		Name:      "Test Product",
		Quantity:  2,
		UnitPrice: 10.00,
	})

	mockRepo.On("Get", ctx, "customer-123").Return(existingCart, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*model.Cart")).Return(nil)
	mockPublisher.On("Publish", ctx, mock.AnythingOfType("*model.EventEnvelope")).Return(nil)

	service := NewCartService(mockRepo, mockPublisher, 24*time.Hour, 72*time.Hour, logger)

	err := service.ClearCart(ctx, "customer-123")

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestCartService_Checkout(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockCartRepository)
	mockPublisher := new(MockEventPublisher)
	logger := zap.NewNop()

	existingCart := model.NewCart("customer-123", 24*time.Hour)
	existingCart.AddItem(model.CartItem{
		ProductID: "prod-1",
		Name:      "Test Product",
		Quantity:  2,
		UnitPrice: 10.00,
	})

	mockRepo.On("Get", ctx, "customer-123").Return(existingCart, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*model.Cart")).Return(nil)
	mockPublisher.On("Publish", ctx, mock.AnythingOfType("*model.EventEnvelope")).Return(nil)

	service := NewCartService(mockRepo, mockPublisher, 24*time.Hour, 72*time.Hour, logger)

	req := &model.CheckoutRequest{
		ShippingAddress: model.ShippingAddress{
			Street:     "123 Main St",
			City:       "Springfield",
			State:      "IL",
			PostalCode: "62701",
			Country:    "US",
		},
	}

	cart, err := service.Checkout(ctx, "customer-123", req)

	require.NoError(t, err)
	assert.Equal(t, 20.00, cart.TotalAmount)
	assert.Len(t, cart.Items, 1)

	mockRepo.AssertExpectations(t)
	mockPublisher.AssertExpectations(t)
}

func TestCartService_Checkout_EmptyCart(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockCartRepository)
	mockPublisher := new(MockEventPublisher)
	logger := zap.NewNop()

	emptyCart := model.NewCart("customer-123", 24*time.Hour)

	mockRepo.On("Get", ctx, "customer-123").Return(emptyCart, nil)

	service := NewCartService(mockRepo, mockPublisher, 24*time.Hour, 72*time.Hour, logger)

	req := &model.CheckoutRequest{
		ShippingAddress: model.ShippingAddress{
			Street:     "123 Main St",
			City:       "Springfield",
			State:      "IL",
			PostalCode: "62701",
			Country:    "US",
		},
	}

	_, err := service.Checkout(ctx, "customer-123", req)

	assert.ErrorIs(t, err, ErrCartEmpty)
	mockRepo.AssertExpectations(t)
}

func TestCartService_ttlFor(t *testing.T) {
	logger := zap.NewNop()
	service := NewCartService(new(MockCartRepository), new(MockEventPublisher), 168*time.Hour, 72*time.Hour, logger)

	assert.Equal(t, 72*time.Hour, service.ttlFor("guest-123"))
	assert.Equal(t, 168*time.Hour, service.ttlFor("customer-123"))
}

func TestCartService_AddItem_RollingExpiry(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockCartRepository)
	mockPublisher := new(MockEventPublisher)
	logger := zap.NewNop()

	existingCart := model.NewCart("guest-123", time.Hour)
	originalExpiry := existingCart.ExpiresAt

	mockRepo.On("Get", ctx, "guest-123").Return(existingCart, nil)
	mockRepo.On("Save", ctx, mock.MatchedBy(func(cart *model.Cart) bool {
		return cart.ExpiresAt.After(originalExpiry)
	})).Return(nil)
	mockPublisher.On("Publish", ctx, mock.AnythingOfType("*model.EventEnvelope")).Return(nil)

	service := NewCartService(mockRepo, mockPublisher, 24*time.Hour, 72*time.Hour, logger)

	req := &model.AddItemRequest{
		ProductID: "prod-1",
		Name:      "Test Product",
		Quantity:  1,
		UnitPrice: 10.00,
	}

	_, err := service.AddItem(ctx, "guest-123", req)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockPublisher.AssertExpectations(t)
}

func TestCartService_MergeGuestCart(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockCartRepository)
	mockPublisher := new(MockEventPublisher)
	logger := zap.NewNop()

	userCart := model.NewCart("customer-123", 24*time.Hour)
	userCart.AddItem(model.CartItem{
		ProductID: "prod-1",
		Name:      "Product 1",
		Quantity:  1,
		UnitPrice: 10.00,
	})
	guestCart := model.NewCart("guest-123", 72*time.Hour)
	guestCart.AddItem(model.CartItem{
		ProductID: "prod-1",
		Name:      "Product 1",
		Quantity:  2,
		UnitPrice: 10.00,
	})
	guestCart.AddItem(model.CartItem{
		ProductID: "prod-2",
		Name:      "Product 2",
		Quantity:  1,
		UnitPrice: 15.00,
	})

	mockRepo.On("Get", ctx, "customer-123").Return(userCart, nil)
	mockRepo.On("Get", ctx, "guest-123").Return(guestCart, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*model.Cart")).Return(nil)
	mockRepo.On("Delete", ctx, "guest-123").Return(nil)
	mockPublisher.On("Publish", ctx, mock.AnythingOfType("*model.EventEnvelope")).Return(nil)

	service := NewCartService(mockRepo, mockPublisher, 24*time.Hour, 72*time.Hour, logger)

	cart, err := service.MergeGuestCart(ctx, "guest-123", "customer-123")

	require.NoError(t, err)
	require.Len(t, cart.Items, 2)
	assert.Equal(t, "prod-1", cart.Items[0].ProductID)
	assert.Equal(t, 3, cart.Items[0].Quantity)
	assert.Equal(t, "prod-2", cart.Items[1].ProductID)
	assert.Equal(t, 45.00, cart.TotalAmount)
	mockRepo.AssertExpectations(t)
	mockPublisher.AssertExpectations(t)
}

func TestCartService_MergeGuestCart_NoGuestCart(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockCartRepository)
	mockPublisher := new(MockEventPublisher)
	logger := zap.NewNop()

	userCart := model.NewCart("customer-123", 24*time.Hour)
	userCart.AddItem(model.CartItem{
		ProductID: "prod-1",
		Name:      "Product 1",
		Quantity:  1,
		UnitPrice: 10.00,
	})

	mockRepo.On("Get", ctx, "customer-123").Return(userCart, nil)
	mockRepo.On("Get", ctx, "guest-123").Return(nil, repository.ErrCartNotFound)

	service := NewCartService(mockRepo, mockPublisher, 24*time.Hour, 72*time.Hour, logger)

	cart, err := service.MergeGuestCart(ctx, "guest-123", "customer-123")

	require.NoError(t, err)
	assert.Same(t, userCart, cart)
	mockRepo.AssertExpectations(t)
}

func TestCartService_MergeGuestCart_ExistingProductAtCapacity(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockCartRepository)
	mockPublisher := new(MockEventPublisher)
	logger := zap.NewNop()

	userCart := model.NewCart("customer-123", 24*time.Hour)
	for i := 0; i < MaxCartItems; i++ {
		userCart.AddItem(model.CartItem{
			ProductID: fmt.Sprintf("prod-%d", i),
			Name:      "Product",
			Quantity:  1,
			UnitPrice: 1.00,
		})
	}
	guestCart := model.NewCart("guest-123", 72*time.Hour)
	guestCart.AddItem(model.CartItem{
		ProductID: "prod-0",
		Name:      "Product",
		Quantity:  2,
		UnitPrice: 1.00,
	})

	mockRepo.On("Get", ctx, "customer-123").Return(userCart, nil)
	mockRepo.On("Get", ctx, "guest-123").Return(guestCart, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*model.Cart")).Return(nil)
	mockRepo.On("Delete", ctx, "guest-123").Return(nil)
	mockPublisher.On("Publish", ctx, mock.AnythingOfType("*model.EventEnvelope")).Return(nil)

	service := NewCartService(mockRepo, mockPublisher, 24*time.Hour, 72*time.Hour, logger)

	cart, err := service.MergeGuestCart(ctx, "guest-123", "customer-123")

	require.NoError(t, err)
	require.Len(t, cart.Items, MaxCartItems)
	assert.Equal(t, 3, cart.Items[0].Quantity)
	mockRepo.AssertExpectations(t)
}

func TestCartService_MergeGuestCart_NewProductAtCapacity(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockCartRepository)
	mockPublisher := new(MockEventPublisher)
	logger := zap.NewNop()

	userCart := model.NewCart("customer-123", 24*time.Hour)
	for i := 0; i < MaxCartItems; i++ {
		userCart.AddItem(model.CartItem{
			ProductID: fmt.Sprintf("prod-%d", i),
			Name:      "Product",
			Quantity:  1,
			UnitPrice: 1.00,
		})
	}
	guestCart := model.NewCart("guest-123", 72*time.Hour)
	guestCart.AddItem(model.CartItem{
		ProductID: "prod-new",
		Name:      "New Product",
		Quantity:  1,
		UnitPrice: 5.00,
	})

	mockRepo.On("Get", ctx, "customer-123").Return(userCart, nil)
	mockRepo.On("Get", ctx, "guest-123").Return(guestCart, nil)

	service := NewCartService(mockRepo, mockPublisher, 24*time.Hour, 72*time.Hour, logger)

	cart, err := service.MergeGuestCart(ctx, "guest-123", "customer-123")

	assert.Nil(t, cart)
	assert.ErrorIs(t, err, ErrMaxItemsExceeded)
	mockRepo.AssertNotCalled(t, "Save", ctx, mock.AnythingOfType("*model.Cart"))
	mockRepo.AssertNotCalled(t, "Delete", ctx, "guest-123")
	mockRepo.AssertExpectations(t)
}
