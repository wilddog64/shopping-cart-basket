package service

import (
	"context"
	"errors"
	"time"

	"github.com/user/shopping-cart-basket/internal/model"
	"github.com/user/shopping-cart-basket/internal/repository"
	"go.uber.org/zap"
)

var (
	// ErrCartEmpty is returned when trying to checkout an empty cart
	ErrCartEmpty = errors.New("cart is empty")
	// ErrItemNotFound is returned when an item is not found in the cart
	ErrItemNotFound = errors.New("item not found in cart")
	// ErrMaxItemsExceeded is returned when cart item limit is exceeded
	ErrMaxItemsExceeded = errors.New("maximum cart items exceeded")
)

const (
	// MaxCartItems is the maximum number of items allowed in a cart
	MaxCartItems = 100
)

// EventPublisher defines the interface for publishing cart events
type EventPublisher interface {
	Publish(ctx context.Context, event *model.EventEnvelope) error
}

// CartService handles cart business logic
type CartService struct {
	repo      repository.CartRepository
	publisher EventPublisher
	cartTTL   time.Duration
	logger    *zap.Logger
}

// NewCartService creates a new cart service
func NewCartService(repo repository.CartRepository, publisher EventPublisher, cartTTL time.Duration, logger *zap.Logger) *CartService {
	return &CartService{
		repo:      repo,
		publisher: publisher,
		cartTTL:   cartTTL,
		logger:    logger,
	}
}

// GetCart retrieves or creates a cart for a customer
func (s *CartService) GetCart(ctx context.Context, customerID string) (*model.Cart, error) {
	cart, err := s.repo.Get(ctx, customerID)
	if err != nil {
		if errors.Is(err, repository.ErrCartNotFound) {
			// Create a new cart
			cart = model.NewCart(customerID, s.cartTTL)
			if err := s.repo.Save(ctx, cart); err != nil {
				return nil, err
			}

			// Publish cart created event
			if s.publisher != nil {
				correlationID := getCorrelationID(ctx)
				event := model.NewCartCreatedEvent(cart, correlationID)
				if err := s.publisher.Publish(ctx, event); err != nil {
					s.logger.Warn("failed to publish cart created event",
						zap.String("cartId", cart.ID),
						zap.Error(err),
					)
				}
			}

			return cart, nil
		}
		return nil, err
	}
	return cart, nil
}

// AddItem adds an item to the cart
func (s *CartService) AddItem(ctx context.Context, customerID string, req *model.AddItemRequest) (*model.Cart, error) {
	cart, err := s.GetCart(ctx, customerID)
	if err != nil {
		return nil, err
	}

	// Check item limit
	if len(cart.Items) >= MaxCartItems {
		return nil, ErrMaxItemsExceeded
	}

	// Add item
	item := model.CartItem{
		ProductID: req.ProductID,
		Name:      req.Name,
		Quantity:  req.Quantity,
		UnitPrice: req.UnitPrice,
	}
	cart.AddItem(item)

	// Save cart
	if err := s.repo.Save(ctx, cart); err != nil {
		return nil, err
	}

	// Publish event
	if s.publisher != nil {
		correlationID := getCorrelationID(ctx)
		event := model.NewCartUpdatedEvent(cart, "item_added", correlationID)
		if err := s.publisher.Publish(ctx, event); err != nil {
			s.logger.Warn("failed to publish cart updated event",
				zap.String("cartId", cart.ID),
				zap.Error(err),
			)
		}
	}

	s.logger.Info("item added to cart",
		zap.String("cartId", cart.ID),
		zap.String("productId", req.ProductID),
		zap.Int("quantity", req.Quantity),
	)

	return cart, nil
}

// UpdateItemQuantity updates the quantity of an item
func (s *CartService) UpdateItemQuantity(ctx context.Context, customerID, itemID string, quantity int) (*model.Cart, error) {
	cart, err := s.repo.Get(ctx, customerID)
	if err != nil {
		return nil, err
	}

	if !cart.UpdateItemQuantity(itemID, quantity) {
		return nil, ErrItemNotFound
	}

	// Save cart
	if err := s.repo.Save(ctx, cart); err != nil {
		return nil, err
	}

	// Publish event
	if s.publisher != nil {
		correlationID := getCorrelationID(ctx)
		event := model.NewCartUpdatedEvent(cart, "item_updated", correlationID)
		if err := s.publisher.Publish(ctx, event); err != nil {
			s.logger.Warn("failed to publish cart updated event",
				zap.String("cartId", cart.ID),
				zap.Error(err),
			)
		}
	}

	s.logger.Info("cart item updated",
		zap.String("cartId", cart.ID),
		zap.String("itemId", itemID),
		zap.Int("quantity", quantity),
	)

	return cart, nil
}

// RemoveItem removes an item from the cart
func (s *CartService) RemoveItem(ctx context.Context, customerID, itemID string) (*model.Cart, error) {
	cart, err := s.repo.Get(ctx, customerID)
	if err != nil {
		return nil, err
	}

	if !cart.RemoveItem(itemID) {
		return nil, ErrItemNotFound
	}

	// Save cart
	if err := s.repo.Save(ctx, cart); err != nil {
		return nil, err
	}

	// Publish event
	if s.publisher != nil {
		correlationID := getCorrelationID(ctx)
		event := model.NewCartUpdatedEvent(cart, "item_removed", correlationID)
		if err := s.publisher.Publish(ctx, event); err != nil {
			s.logger.Warn("failed to publish cart updated event",
				zap.String("cartId", cart.ID),
				zap.Error(err),
			)
		}
	}

	s.logger.Info("item removed from cart",
		zap.String("cartId", cart.ID),
		zap.String("itemId", itemID),
	)

	return cart, nil
}

// ClearCart removes all items from the cart
func (s *CartService) ClearCart(ctx context.Context, customerID string) error {
	cart, err := s.repo.Get(ctx, customerID)
	if err != nil {
		if errors.Is(err, repository.ErrCartNotFound) {
			return nil // Nothing to clear
		}
		return err
	}

	cart.Clear()

	// Save cart
	if err := s.repo.Save(ctx, cart); err != nil {
		return err
	}

	// Publish event
	if s.publisher != nil {
		correlationID := getCorrelationID(ctx)
		event := model.NewCartClearedEvent(cart, correlationID)
		if err := s.publisher.Publish(ctx, event); err != nil {
			s.logger.Warn("failed to publish cart cleared event",
				zap.String("cartId", cart.ID),
				zap.Error(err),
			)
		}
	}

	s.logger.Info("cart cleared",
		zap.String("cartId", cart.ID),
		zap.String("customerId", customerID),
	)

	return nil
}

// Checkout processes the cart for checkout
func (s *CartService) Checkout(ctx context.Context, customerID string, req *model.CheckoutRequest) (*model.Cart, error) {
	cart, err := s.repo.Get(ctx, customerID)
	if err != nil {
		return nil, err
	}

	if cart.IsEmpty() {
		return nil, ErrCartEmpty
	}

	// Publish checkout event
	if s.publisher != nil {
		correlationID := getCorrelationID(ctx)
		event := model.NewCartCheckoutEvent(cart, req.ShippingAddress, correlationID)
		if err := s.publisher.Publish(ctx, event); err != nil {
			s.logger.Error("failed to publish checkout event",
				zap.String("cartId", cart.ID),
				zap.Error(err),
			)
			return nil, err
		}
	}

	// Clear the cart after successful checkout
	cartCopy := *cart // Keep a copy for the response
	cart.Clear()
	if err := s.repo.Save(ctx, cart); err != nil {
		s.logger.Warn("failed to clear cart after checkout",
			zap.String("cartId", cart.ID),
			zap.Error(err),
		)
	}

	s.logger.Info("cart checkout completed",
		zap.String("cartId", cartCopy.ID),
		zap.String("customerId", customerID),
		zap.Float64("totalAmount", cartCopy.TotalAmount),
	)

	return &cartCopy, nil
}

// Context key for correlation ID
type contextKey string

const correlationIDKey contextKey = "correlationID"

// SetCorrelationID sets the correlation ID in the context
func SetCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey, id)
}

// getCorrelationID gets the correlation ID from the context
func getCorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(correlationIDKey).(string); ok {
		return id
	}
	return ""
}
