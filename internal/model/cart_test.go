package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewCart(t *testing.T) {
	cart := NewCart("customer-123", 24*time.Hour)

	assert.NotEmpty(t, cart.ID)
	assert.Equal(t, "customer-123", cart.CustomerID)
	assert.Empty(t, cart.Items)
	assert.Equal(t, 0.0, cart.TotalAmount)
	assert.Equal(t, "USD", cart.Currency)
	assert.False(t, cart.CreatedAt.IsZero())
	assert.False(t, cart.UpdatedAt.IsZero())
	assert.False(t, cart.ExpiresAt.IsZero())
}

func TestCart_AddItem(t *testing.T) {
	cart := NewCart("customer-123", 24*time.Hour)

	item := CartItem{
		ProductID: "prod-1",
		Name:      "Test Product",
		Quantity:  2,
		UnitPrice: 10.00,
	}
	cart.AddItem(item)

	assert.Len(t, cart.Items, 1)
	assert.NotEmpty(t, cart.Items[0].ID)
	assert.Equal(t, "prod-1", cart.Items[0].ProductID)
	assert.Equal(t, 2, cart.Items[0].Quantity)
	assert.Equal(t, 20.00, cart.Items[0].SubTotal)
	assert.Equal(t, 20.00, cart.TotalAmount)
}

func TestCart_AddItem_UpdatesExisting(t *testing.T) {
	cart := NewCart("customer-123", 24*time.Hour)

	// Add first item
	cart.AddItem(CartItem{
		ProductID: "prod-1",
		Name:      "Test Product",
		Quantity:  2,
		UnitPrice: 10.00,
	})

	// Add same product again
	cart.AddItem(CartItem{
		ProductID: "prod-1",
		Name:      "Test Product",
		Quantity:  3,
		UnitPrice: 10.00,
	})

	assert.Len(t, cart.Items, 1)
	assert.Equal(t, 5, cart.Items[0].Quantity)
	assert.Equal(t, 50.00, cart.Items[0].SubTotal)
	assert.Equal(t, 50.00, cart.TotalAmount)
}

func TestCart_UpdateItemQuantity(t *testing.T) {
	cart := NewCart("customer-123", 24*time.Hour)

	cart.AddItem(CartItem{
		ProductID: "prod-1",
		Name:      "Test Product",
		Quantity:  2,
		UnitPrice: 10.00,
	})
	itemID := cart.Items[0].ID

	updated := cart.UpdateItemQuantity(itemID, 5)

	assert.True(t, updated)
	assert.Equal(t, 5, cart.Items[0].Quantity)
	assert.Equal(t, 50.00, cart.Items[0].SubTotal)
	assert.Equal(t, 50.00, cart.TotalAmount)
}

func TestCart_UpdateItemQuantity_RemovesWhenZero(t *testing.T) {
	cart := NewCart("customer-123", 24*time.Hour)

	cart.AddItem(CartItem{
		ProductID: "prod-1",
		Name:      "Test Product",
		Quantity:  2,
		UnitPrice: 10.00,
	})
	itemID := cart.Items[0].ID

	updated := cart.UpdateItemQuantity(itemID, 0)

	assert.True(t, updated)
	assert.Empty(t, cart.Items)
	assert.Equal(t, 0.0, cart.TotalAmount)
}

func TestCart_UpdateItemQuantity_NotFound(t *testing.T) {
	cart := NewCart("customer-123", 24*time.Hour)

	updated := cart.UpdateItemQuantity("non-existent", 5)

	assert.False(t, updated)
}

func TestCart_RemoveItem(t *testing.T) {
	cart := NewCart("customer-123", 24*time.Hour)

	cart.AddItem(CartItem{
		ProductID: "prod-1",
		Name:      "Test Product",
		Quantity:  2,
		UnitPrice: 10.00,
	})
	cart.AddItem(CartItem{
		ProductID: "prod-2",
		Name:      "Another Product",
		Quantity:  1,
		UnitPrice: 25.00,
	})
	itemID := cart.Items[0].ID

	removed := cart.RemoveItem(itemID)

	assert.True(t, removed)
	assert.Len(t, cart.Items, 1)
	assert.Equal(t, "prod-2", cart.Items[0].ProductID)
	assert.Equal(t, 25.00, cart.TotalAmount)
}

func TestCart_RemoveItem_NotFound(t *testing.T) {
	cart := NewCart("customer-123", 24*time.Hour)

	removed := cart.RemoveItem("non-existent")

	assert.False(t, removed)
}

func TestCart_Clear(t *testing.T) {
	cart := NewCart("customer-123", 24*time.Hour)

	cart.AddItem(CartItem{
		ProductID: "prod-1",
		Name:      "Test Product",
		Quantity:  2,
		UnitPrice: 10.00,
	})

	cart.Clear()

	assert.Empty(t, cart.Items)
	assert.Equal(t, 0.0, cart.TotalAmount)
}

func TestCart_IsEmpty(t *testing.T) {
	cart := NewCart("customer-123", 24*time.Hour)

	assert.True(t, cart.IsEmpty())

	cart.AddItem(CartItem{
		ProductID: "prod-1",
		Name:      "Test Product",
		Quantity:  1,
		UnitPrice: 10.00,
	})

	assert.False(t, cart.IsEmpty())
}

func TestCart_ItemCount(t *testing.T) {
	cart := NewCart("customer-123", 24*time.Hour)

	assert.Equal(t, 0, cart.ItemCount())

	cart.AddItem(CartItem{
		ProductID: "prod-1",
		Name:      "Test Product",
		Quantity:  2,
		UnitPrice: 10.00,
	})
	cart.AddItem(CartItem{
		ProductID: "prod-2",
		Name:      "Another Product",
		Quantity:  3,
		UnitPrice: 25.00,
	})

	assert.Equal(t, 5, cart.ItemCount())
}

func TestCart_ToResponse(t *testing.T) {
	cart := NewCart("customer-123", 24*time.Hour)

	cart.AddItem(CartItem{
		ProductID: "prod-1",
		Name:      "Test Product",
		Quantity:  2,
		UnitPrice: 10.00,
	})

	response := cart.ToResponse()

	assert.Equal(t, cart.ID, response.ID)
	assert.Equal(t, cart.CustomerID, response.CustomerID)
	assert.Equal(t, cart.Items, response.Items)
	assert.Equal(t, 2, response.ItemCount)
	assert.Equal(t, cart.TotalAmount, response.TotalAmount)
	assert.Equal(t, cart.Currency, response.Currency)
}
