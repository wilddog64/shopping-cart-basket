# Basket Service

A Go microservice for managing shopping basket sessions in the Shopping Cart platform.

## Overview

The Cart Service handles:
- Shopping cart session management
- Item add/update/remove operations
- Cart total calculation
- Checkout orchestration (converts cart to order)
- Publishing cart events to RabbitMQ

## Technology Stack

- **Language**: Go 1.21+
- **Framework**: Gin
- **Storage**: Redis
- **Messaging**: RabbitMQ
- **Auth**: OAuth2/OIDC with Keycloak

## Quick Start

### Prerequisites

- Go 1.21+
- Redis
- RabbitMQ (optional)
- Keycloak (optional, for OAuth2)

### Running Locally

```bash
# Install dependencies
go mod download

# Start port-forward to K8s Redis (in separate terminal)
./bin/port-forward.sh

# Run the service
./bin/run-local.sh

# Or auto-start Redis port-forward
./bin/run-local.sh --start-redis
```

### Running Tests

```bash
# Unit tests (no external dependencies)
make test

# Integration tests (auto-manages Redis port-forward)
make test-integration

# With coverage
make coverage
```

### Integration Tests in CI

For CI environments with external Redis, use `test-integration-ci` with environment variables:

```bash
# Get Redis password from K8s secret
kubectl get secret -n shopping-cart-data redis-cart-secret -o jsonpath='{.data.password}' | base64 -d

# Run with external Redis
REDIS_ADDR=redis.example.com:6379 \
REDIS_PASSWORD=<password> \
make test-integration-ci
```

**Credential locations:**
| Secret | Namespace | Key |
|--------|-----------|-----|
| `redis-cart-secret` | `shopping-cart-data` | `password` |

### Building

```bash
# Build binary
make build

# Build Docker image
make docker-build
```

## API Endpoints

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/api/v1/cart` | Get current user's cart | Required |
| POST | `/api/v1/cart/items` | Add item to cart | Required |
| PUT | `/api/v1/cart/items/{itemId}` | Update item quantity | Required |
| DELETE | `/api/v1/cart/items/{itemId}` | Remove item from cart | Required |
| DELETE | `/api/v1/cart` | Clear cart | Required |
| POST | `/api/v1/cart/checkout` | Convert cart to order | Required |
| GET | `/health` | Health check | Public |
| GET | `/health/live` | Liveness probe | Public |
| GET | `/health/ready` | Readiness probe | Public |
| GET | `/metrics` | Prometheus metrics | Public |

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | 8083 | HTTP server port |
| `REDIS_HOST` | localhost | Redis hostname |
| `REDIS_PORT` | 6379 | Redis port |
| `REDIS_PASSWORD` | - | Redis password |
| `REDIS_DB` | 0 | Redis database |
| `CART_TTL` | 168h | Cart expiration (7 days) |
| `OAUTH2_ENABLED` | false | Enable OAuth2 |
| `OAUTH2_ISSUER_URI` | - | Keycloak issuer URL |
| `RABBITMQ_HOST` | localhost | RabbitMQ host |
| `RABBITMQ_PORT` | 5672 | RabbitMQ port |
| `LOG_LEVEL` | info | Log level |

## API Examples

### Add Item to Cart

```bash
curl -X POST http://localhost:8083/api/v1/cart/items \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "productId": "prod-123",
    "name": "Widget",
    "quantity": 2,
    "unitPrice": 29.99
  }'
```

### Get Cart

```bash
curl http://localhost:8083/api/v1/cart \
  -H "Authorization: Bearer $TOKEN"
```

### Update Item Quantity

```bash
curl -X PUT http://localhost:8083/api/v1/cart/items/item-123 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"quantity": 5}'
```

### Remove Item

```bash
curl -X DELETE http://localhost:8083/api/v1/cart/items/item-123 \
  -H "Authorization: Bearer $TOKEN"
```

### Clear Cart

```bash
curl -X DELETE http://localhost:8083/api/v1/cart \
  -H "Authorization: Bearer $TOKEN"
```

### Checkout

```bash
curl -X POST http://localhost:8083/api/v1/cart/checkout \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "shippingAddress": {
      "street": "123 Main St",
      "city": "Springfield",
      "state": "IL",
      "postalCode": "62701",
      "country": "US"
    }
  }'
```

## Project Structure

```
shopping-cart-basket/
├── cmd/server/          # Application entry point
├── internal/
│   ├── config/          # Configuration
│   ├── handler/         # HTTP handlers
│   ├── service/         # Business logic
│   ├── repository/      # Data access (Redis)
│   ├── model/           # Domain models
│   ├── auth/            # JWT authentication
│   └── event/           # RabbitMQ events
├── pkg/response/        # Shared response types
├── k8s/                 # Kubernetes manifests
└── docs/                # Documentation
```

## Documentation

- [Architecture](docs/architecture/README.md)
- [API Reference](docs/api/README.md)
- [Troubleshooting](docs/troubleshooting/README.md)

## Related Services

- **Order Service**: Receives checkout requests
- **Product Catalog**: Product information
- **Keycloak**: Identity provider

## License

MIT
