package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/user/shopping-cart-basket/internal/model"
	"github.com/user/shopping-cart-basket/internal/repository"
	"github.com/user/shopping-cart-basket/internal/service"
	"github.com/user/shopping-cart-basket/pkg/response"
	"go.uber.org/zap"
)

// CartHandler handles cart HTTP requests
type CartHandler struct {
	service *service.CartService
	logger  *zap.Logger
}

// NewCartHandler creates a new cart handler
func NewCartHandler(service *service.CartService, logger *zap.Logger) *CartHandler {
	return &CartHandler{
		service: service,
		logger:  logger,
	}
}

// GetCart handles GET /api/v1/cart
func (h *CartHandler) GetCart(c *gin.Context) {
	customerID := getCustomerID(c)
	if customerID == "" {
		response.Unauthorized(c, "Customer ID not found")
		return
	}

	cart, err := h.service.GetCart(c.Request.Context(), customerID)
	if err != nil {
		h.logger.Error("failed to get cart",
			zap.String("customerId", customerID),
			zap.Error(err),
		)
		response.InternalError(c, "Failed to get cart")
		return
	}

	response.Success(c, cart.ToResponse())
}

// AddItem handles POST /api/v1/cart/items
func (h *CartHandler) AddItem(c *gin.Context) {
	customerID := getCustomerID(c)
	if customerID == "" {
		response.Unauthorized(c, "Customer ID not found")
		return
	}

	var req model.AddItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	cart, err := h.service.AddItem(c.Request.Context(), customerID, &req)
	if err != nil {
		if errors.Is(err, service.ErrMaxItemsExceeded) {
			response.BadRequest(c, "Maximum cart items exceeded")
			return
		}
		h.logger.Error("failed to add item",
			zap.String("customerId", customerID),
			zap.String("productId", req.ProductID),
			zap.Error(err),
		)
		response.InternalError(c, "Failed to add item")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    cart.ToResponse(),
	})
}

// UpdateItem handles PUT /api/v1/cart/items/:itemId
func (h *CartHandler) UpdateItem(c *gin.Context) {
	customerID := getCustomerID(c)
	if customerID == "" {
		response.Unauthorized(c, "Customer ID not found")
		return
	}

	itemID := c.Param("itemId")
	if itemID == "" {
		response.BadRequest(c, "Item ID is required")
		return
	}

	var req model.UpdateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	cart, err := h.service.UpdateItemQuantity(c.Request.Context(), customerID, itemID, req.Quantity)
	if err != nil {
		if errors.Is(err, service.ErrItemNotFound) {
			response.NotFound(c, "Item not found in cart")
			return
		}
		if errors.Is(err, repository.ErrCartNotFound) {
			response.NotFound(c, "Cart not found")
			return
		}
		h.logger.Error("failed to update item",
			zap.String("customerId", customerID),
			zap.String("itemId", itemID),
			zap.Error(err),
		)
		response.InternalError(c, "Failed to update item")
		return
	}

	response.Success(c, cart.ToResponse())
}

// RemoveItem handles DELETE /api/v1/cart/items/:itemId
func (h *CartHandler) RemoveItem(c *gin.Context) {
	customerID := getCustomerID(c)
	if customerID == "" {
		response.Unauthorized(c, "Customer ID not found")
		return
	}

	itemID := c.Param("itemId")
	if itemID == "" {
		response.BadRequest(c, "Item ID is required")
		return
	}

	cart, err := h.service.RemoveItem(c.Request.Context(), customerID, itemID)
	if err != nil {
		if errors.Is(err, service.ErrItemNotFound) {
			response.NotFound(c, "Item not found in cart")
			return
		}
		if errors.Is(err, repository.ErrCartNotFound) {
			response.NotFound(c, "Cart not found")
			return
		}
		h.logger.Error("failed to remove item",
			zap.String("customerId", customerID),
			zap.String("itemId", itemID),
			zap.Error(err),
		)
		response.InternalError(c, "Failed to remove item")
		return
	}

	response.Success(c, cart.ToResponse())
}

// ClearCart handles DELETE /api/v1/cart
func (h *CartHandler) ClearCart(c *gin.Context) {
	customerID := getCustomerID(c)
	if customerID == "" {
		response.Unauthorized(c, "Customer ID not found")
		return
	}

	err := h.service.ClearCart(c.Request.Context(), customerID)
	if err != nil {
		h.logger.Error("failed to clear cart",
			zap.String("customerId", customerID),
			zap.Error(err),
		)
		response.InternalError(c, "Failed to clear cart")
		return
	}

	response.NoContent(c)
}

// Checkout handles POST /api/v1/cart/checkout
func (h *CartHandler) Checkout(c *gin.Context) {
	customerID := getCustomerID(c)
	if customerID == "" {
		response.Unauthorized(c, "Customer ID not found")
		return
	}

	var req model.CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	cart, err := h.service.Checkout(c.Request.Context(), customerID, &req)
	if err != nil {
		if errors.Is(err, service.ErrCartEmpty) {
			response.BadRequest(c, "Cart is empty")
			return
		}
		if errors.Is(err, repository.ErrCartNotFound) {
			response.NotFound(c, "Cart not found")
			return
		}
		h.logger.Error("failed to checkout",
			zap.String("customerId", customerID),
			zap.Error(err),
		)
		response.InternalError(c, "Failed to process checkout")
		return
	}

	response.Success(c, gin.H{
		"message": "Checkout successful",
		"cart":    cart.ToResponse(),
	})
}

// Context key for customer ID
const customerIDKey = "customerID"

// getCustomerID gets the customer ID from the context
func getCustomerID(c *gin.Context) string {
	if id, exists := c.Get(customerIDKey); exists {
		if customerID, ok := id.(string); ok {
			return customerID
		}
	}
	return ""
}

// SetCustomerID sets the customer ID in the context
func SetCustomerID(c *gin.Context, customerID string) {
	c.Set(customerIDKey, customerID)
}
