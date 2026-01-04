package model

import (
	"time"

	"github.com/google/uuid"
)

// Event types
const (
	EventTypeCartCreated  = "cart.created"
	EventTypeCartUpdated  = "cart.updated"
	EventTypeCartCleared  = "cart.cleared"
	EventTypeCartCheckout = "cart.checkout"
)

// EventEnvelope wraps all events with standard metadata
type EventEnvelope struct {
	ID            string      `json:"id"`
	Type          string      `json:"type"`
	Version       string      `json:"version"`
	Timestamp     time.Time   `json:"timestamp"`
	Source        string      `json:"source"`
	CorrelationID string      `json:"correlationId"`
	Data          interface{} `json:"data"`
}

// NewEventEnvelope creates a new event envelope
func NewEventEnvelope(eventType string, correlationID string, data interface{}) *EventEnvelope {
	if correlationID == "" {
		correlationID = uuid.New().String()
	}
	return &EventEnvelope{
		ID:            uuid.New().String(),
		Type:          eventType,
		Version:       "1.0",
		Timestamp:     time.Now(),
		Source:        "cart-service",
		CorrelationID: correlationID,
		Data:          data,
	}
}

// CartCreatedEvent represents cart creation event data
type CartCreatedEvent struct {
	CartID     string `json:"cartId"`
	CustomerID string `json:"customerId"`
}

// CartUpdatedEvent represents cart update event data
type CartUpdatedEvent struct {
	CartID      string     `json:"cartId"`
	CustomerID  string     `json:"customerId"`
	Items       []CartItem `json:"items"`
	ItemCount   int        `json:"itemCount"`
	TotalAmount float64    `json:"totalAmount"`
	Currency    string     `json:"currency"`
	Action      string     `json:"action"` // "item_added", "item_updated", "item_removed"
}

// CartClearedEvent represents cart cleared event data
type CartClearedEvent struct {
	CartID     string `json:"cartId"`
	CustomerID string `json:"customerId"`
}

// CartCheckoutEvent represents checkout event data
type CartCheckoutEvent struct {
	CartID          string          `json:"cartId"`
	CustomerID      string          `json:"customerId"`
	Items           []CartItem      `json:"items"`
	TotalAmount     float64         `json:"totalAmount"`
	Currency        string          `json:"currency"`
	ShippingAddress ShippingAddress `json:"shippingAddress"`
}

// NewCartCreatedEvent creates a cart created event
func NewCartCreatedEvent(cart *Cart, correlationID string) *EventEnvelope {
	return NewEventEnvelope(EventTypeCartCreated, correlationID, &CartCreatedEvent{
		CartID:     cart.ID,
		CustomerID: cart.CustomerID,
	})
}

// NewCartUpdatedEvent creates a cart updated event
func NewCartUpdatedEvent(cart *Cart, action string, correlationID string) *EventEnvelope {
	return NewEventEnvelope(EventTypeCartUpdated, correlationID, &CartUpdatedEvent{
		CartID:      cart.ID,
		CustomerID:  cart.CustomerID,
		Items:       cart.Items,
		ItemCount:   cart.ItemCount(),
		TotalAmount: cart.TotalAmount,
		Currency:    cart.Currency,
		Action:      action,
	})
}

// NewCartClearedEvent creates a cart cleared event
func NewCartClearedEvent(cart *Cart, correlationID string) *EventEnvelope {
	return NewEventEnvelope(EventTypeCartCleared, correlationID, &CartClearedEvent{
		CartID:     cart.ID,
		CustomerID: cart.CustomerID,
	})
}

// NewCartCheckoutEvent creates a checkout event
func NewCartCheckoutEvent(cart *Cart, shippingAddress ShippingAddress, correlationID string) *EventEnvelope {
	return NewEventEnvelope(EventTypeCartCheckout, correlationID, &CartCheckoutEvent{
		CartID:          cart.ID,
		CustomerID:      cart.CustomerID,
		Items:           cart.Items,
		TotalAmount:     cart.TotalAmount,
		Currency:        cart.Currency,
		ShippingAddress: shippingAddress,
	})
}
