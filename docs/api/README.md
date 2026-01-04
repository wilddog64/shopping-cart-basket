# Cart Service API Reference

## Base URL

```
http://localhost:8083
```

## Authentication

All cart endpoints require authentication via JWT Bearer token:

```bash
curl -H "Authorization: Bearer <token>" http://localhost:8083/api/v1/cart
```

For development without OAuth2 (`OAUTH2_ENABLED=false`), use `X-User-ID` header:

```bash
curl -H "X-User-ID: user-123" http://localhost:8083/api/v1/cart
```

## Endpoints

### Get Cart

Retrieves the current user's cart. Creates a new empty cart if none exists.

```
GET /api/v1/cart
```

**Response:**

```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "customerId": "user-123",
    "items": [
      {
        "id": "item-uuid",
        "productId": "prod-001",
        "name": "Premium Widget",
        "quantity": 2,
        "unitPrice": 29.99,
        "subTotal": 59.98
      }
    ],
    "itemCount": 2,
    "totalAmount": 59.98,
    "currency": "USD",
    "createdAt": "2024-01-15T10:30:00Z",
    "updatedAt": "2024-01-15T10:35:00Z",
    "expiresAt": "2024-01-22T10:30:00Z"
  }
}
```

### Add Item

Adds an item to the cart. If the product already exists, updates the quantity.

```
POST /api/v1/cart/items
```

**Request:**

```json
{
  "productId": "prod-001",
  "name": "Premium Widget",
  "quantity": 2,
  "unitPrice": 29.99
}
```

**Response:** `201 Created`

```json
{
  "success": true,
  "data": {
    "id": "cart-uuid",
    "customerId": "user-123",
    "items": [...],
    "itemCount": 2,
    "totalAmount": 59.98,
    "currency": "USD"
  }
}
```

**Errors:**

| Status | Code | Description |
|--------|------|-------------|
| 400 | BAD_REQUEST | Invalid request body |
| 400 | BAD_REQUEST | Maximum cart items exceeded (100) |
| 401 | UNAUTHORIZED | Missing or invalid token |

### Update Item Quantity

Updates the quantity of an existing item.

```
PUT /api/v1/cart/items/{itemId}
```

**Request:**

```json
{
  "quantity": 5
}
```

**Response:** `200 OK`

```json
{
  "success": true,
  "data": {
    "id": "cart-uuid",
    "items": [...],
    "totalAmount": 149.95
  }
}
```

**Errors:**

| Status | Code | Description |
|--------|------|-------------|
| 400 | BAD_REQUEST | Invalid request body |
| 404 | NOT_FOUND | Item not found in cart |
| 404 | NOT_FOUND | Cart not found |

**Note:** Setting quantity to 0 removes the item.

### Remove Item

Removes an item from the cart.

```
DELETE /api/v1/cart/items/{itemId}
```

**Response:** `200 OK`

```json
{
  "success": true,
  "data": {
    "id": "cart-uuid",
    "items": [...],
    "totalAmount": 0.00
  }
}
```

**Errors:**

| Status | Code | Description |
|--------|------|-------------|
| 404 | NOT_FOUND | Item not found in cart |
| 404 | NOT_FOUND | Cart not found |

### Clear Cart

Removes all items from the cart.

```
DELETE /api/v1/cart
```

**Response:** `204 No Content`

### Checkout

Initiates checkout process. Publishes a checkout event and clears the cart.

```
POST /api/v1/cart/checkout
```

**Request:**

```json
{
  "shippingAddress": {
    "street": "123 Main St",
    "city": "Springfield",
    "state": "IL",
    "postalCode": "62701",
    "country": "US"
  }
}
```

**Response:** `200 OK`

```json
{
  "success": true,
  "data": {
    "message": "Checkout successful",
    "cart": {
      "id": "cart-uuid",
      "customerId": "user-123",
      "items": [...],
      "totalAmount": 59.98
    }
  }
}
```

**Errors:**

| Status | Code | Description |
|--------|------|-------------|
| 400 | BAD_REQUEST | Invalid shipping address |
| 400 | BAD_REQUEST | Cart is empty |
| 404 | NOT_FOUND | Cart not found |

## Health Endpoints

### Health Check

```
GET /health
```

**Response:**

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "checks": {
    "redis": "ok"
  }
}
```

### Liveness Probe

```
GET /health/live
```

**Response:**

```json
{
  "status": "alive"
}
```

### Readiness Probe

```
GET /health/ready
```

**Response:**

```json
{
  "status": "ready",
  "checks": {
    "redis": "ready"
  }
}
```

### Metrics

```
GET /metrics
```

Returns Prometheus-formatted metrics.

## Error Response Format

All error responses follow this format:

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message"
  }
}
```

## Common Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| BAD_REQUEST | 400 | Invalid request |
| UNAUTHORIZED | 401 | Authentication required |
| FORBIDDEN | 403 | Insufficient permissions |
| NOT_FOUND | 404 | Resource not found |
| RATE_LIMIT_EXCEEDED | 429 | Too many requests |
| INTERNAL_ERROR | 500 | Server error |
| SERVICE_UNAVAILABLE | 503 | Service unavailable |

## Rate Limiting

- Default: 100 requests per second per IP
- Exceeded: Returns 429 with `Retry-After` header

## Examples

### Complete Cart Flow

```bash
# Get cart
curl http://localhost:8083/api/v1/cart -H "X-User-ID: user-123"

# Add items
curl -X POST http://localhost:8083/api/v1/cart/items \
  -H "X-User-ID: user-123" \
  -H "Content-Type: application/json" \
  -d '{"productId":"prod-001","name":"Widget","quantity":2,"unitPrice":29.99}'

# Update quantity
curl -X PUT http://localhost:8083/api/v1/cart/items/{itemId} \
  -H "X-User-ID: user-123" \
  -H "Content-Type: application/json" \
  -d '{"quantity":5}'

# Remove item
curl -X DELETE http://localhost:8083/api/v1/cart/items/{itemId} \
  -H "X-User-ID: user-123"

# Checkout
curl -X POST http://localhost:8083/api/v1/cart/checkout \
  -H "X-User-ID: user-123" \
  -H "Content-Type: application/json" \
  -d '{"shippingAddress":{"street":"123 Main St","city":"Springfield","state":"IL","postalCode":"62701","country":"US"}}'
```
