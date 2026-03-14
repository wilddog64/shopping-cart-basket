package model

import (
	"time"

	"github.com/google/uuid"
)

// Cart represents a shopping cart
type Cart struct {
	ID          string     `json:"id"`
	CustomerID  string     `json:"customerId"`
	Items       []CartItem `json:"items"`
	TotalAmount float64    `json:"totalAmount"`
	Currency    string     `json:"currency"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	ExpiresAt   time.Time  `json:"expiresAt"`
}

// CartItem represents an item in the cart
type CartItem struct {
	ID        string  `json:"id"`
	ProductID string  `json:"productId"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unitPrice"`
	SubTotal  float64 `json:"subTotal"`
}

// NewCart creates a new cart for a customer
func NewCart(customerID string, ttl time.Duration) *Cart {
	now := time.Now()
	return &Cart{
		ID:          uuid.New().String(),
		CustomerID:  customerID,
		Items:       []CartItem{},
		TotalAmount: 0,
		Currency:    "USD",
		CreatedAt:   now,
		UpdatedAt:   now,
		ExpiresAt:   now.Add(ttl),
	}
}

// AddItem adds an item to the cart or updates quantity if exists
func (c *Cart) AddItem(item CartItem) {
	// Check if item already exists
	for i, existing := range c.Items {
		if existing.ProductID == item.ProductID {
			c.Items[i].Quantity += item.Quantity
			c.Items[i].SubTotal = float64(c.Items[i].Quantity) * c.Items[i].UnitPrice
			c.recalculateTotal()
			c.UpdatedAt = time.Now()
			return
		}
	}

	// Add new item
	item.ID = uuid.New().String()
	item.SubTotal = float64(item.Quantity) * item.UnitPrice
	c.Items = append(c.Items, item)
	c.recalculateTotal()
	c.UpdatedAt = time.Now()
}

// UpdateItemQuantity updates the quantity of an item
func (c *Cart) UpdateItemQuantity(itemID string, quantity int) bool {
	for i, item := range c.Items {
		if item.ID == itemID {
			if quantity <= 0 {
				// Remove item if quantity is 0 or negative
				c.Items = append(c.Items[:i], c.Items[i+1:]...)
			} else {
				c.Items[i].Quantity = quantity
				c.Items[i].SubTotal = float64(quantity) * c.Items[i].UnitPrice
			}
			c.recalculateTotal()
			c.UpdatedAt = time.Now()
			return true
		}
	}
	return false
}

// RemoveItem removes an item from the cart
func (c *Cart) RemoveItem(itemID string) bool {
	for i, item := range c.Items {
		if item.ID == itemID {
			c.Items = append(c.Items[:i], c.Items[i+1:]...)
			c.recalculateTotal()
			c.UpdatedAt = time.Now()
			return true
		}
	}
	return false
}

// Clear removes all items from the cart
func (c *Cart) Clear() {
	c.Items = []CartItem{}
	c.TotalAmount = 0
	c.UpdatedAt = time.Now()
}

// IsEmpty returns true if the cart has no items
func (c *Cart) IsEmpty() bool {
	return len(c.Items) == 0
}

// ItemCount returns the total number of items in the cart
func (c *Cart) ItemCount() int {
	count := 0
	for _, item := range c.Items {
		count += item.Quantity
	}
	return count
}

// recalculateTotal recalculates the total amount
func (c *Cart) recalculateTotal() {
	total := 0.0
	for _, item := range c.Items {
		total += item.SubTotal
	}
	c.TotalAmount = total
}

// Request/Response DTOs

// AddItemRequest represents a request to add an item
type AddItemRequest struct {
	ProductID string  `json:"productId" binding:"required"`
	Name      string  `json:"name" binding:"required"`
	Quantity  int     `json:"quantity" binding:"required,min=1"`
	UnitPrice float64 `json:"unitPrice" binding:"required,min=0"`
}

// UpdateItemRequest represents a request to update item quantity
type UpdateItemRequest struct {
	Quantity int `json:"quantity" binding:"required,min=0"`
}

// CheckoutRequest represents a checkout request
type CheckoutRequest struct {
	ShippingAddress ShippingAddress `json:"shippingAddress" binding:"required"`
}

// ShippingAddress represents a shipping address
type ShippingAddress struct {
	Street     string `json:"street" binding:"required"`
	City       string `json:"city" binding:"required"`
	State      string `json:"state" binding:"required"`
	PostalCode string `json:"postalCode" binding:"required"`
	Country    string `json:"country" binding:"required"`
}

// CartResponse represents the cart API response
type CartResponse struct {
	ID          string     `json:"id"`
	CustomerID  string     `json:"customerId"`
	Items       []CartItem `json:"items"`
	ItemCount   int        `json:"itemCount"`
	TotalAmount float64    `json:"totalAmount"`
	Currency    string     `json:"currency"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	ExpiresAt   time.Time  `json:"expiresAt"`
}

// ToResponse converts a Cart to CartResponse
func (c *Cart) ToResponse() *CartResponse {
	return &CartResponse{
		ID:          c.ID,
		CustomerID:  c.CustomerID,
		Items:       c.Items,
		ItemCount:   c.ItemCount(),
		TotalAmount: c.TotalAmount,
		Currency:    c.Currency,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
		ExpiresAt:   c.ExpiresAt,
	}
}
