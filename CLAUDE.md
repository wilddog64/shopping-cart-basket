# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Basket Service** is a Go microservice for managing shopping basket sessions. It uses Redis for state storage, integrates with RabbitMQ for event publishing, and supports OAuth2/OIDC authentication via Keycloak.

**Status**: In Development

**Primary Language**: Go 1.21+

## Repository Structure

```
shopping-cart-basket/
├── cmd/server/              # Application entry point
│   └── main.go              # Main function, router setup
├── internal/
│   ├── config/
│   │   └── config.go        # Configuration from env vars
│   ├── handler/
│   │   ├── cart_handler.go  # Cart HTTP handlers
│   │   ├── health_handler.go # Health check handlers
│   │   └── middleware.go    # Auth, logging middleware
│   ├── service/
│   │   └── cart_service.go  # Business logic
│   ├── repository/
│   │   └── cart_repository.go # Redis data access
│   ├── model/
│   │   ├── cart.go          # Cart domain models
│   │   └── event.go         # Event structures
│   ├── auth/
│   │   └── jwt.go           # JWT validation
│   └── event/
│       └── publisher.go     # RabbitMQ publishing
├── pkg/response/
│   └── response.go          # API response helpers
├── k8s/base/                # Kubernetes manifests
├── docs/                    # Documentation
├── go.mod                   # Go module definition
├── Makefile                 # Build automation
└── Dockerfile               # Container build
```

## Development Commands

```bash
# Run locally
make run

# Run tests
make test

# Build binary
make build

# Format code
make fmt

# Run linter
make lint

# Generate coverage
make coverage
```

## Key Design Patterns

### Layer Architecture
- **Handler**: HTTP request/response, validation
- **Service**: Business logic, orchestration
- **Repository**: Data access (Redis)

### Configuration
All configuration via environment variables with sensible defaults:
```go
cfg := config.Load()
```

### Error Handling
Use custom error types with HTTP status mapping:
```go
type AppError struct {
    Code    string
    Message string
    Status  int
}
```

### Logging
Structured logging with zap:
```go
logger.Info("cart updated",
    zap.String("cartId", cart.ID),
    zap.Int("itemCount", len(cart.Items)),
)
```

## Testing Strategy

### Unit Tests
- Mock Redis with interface
- Test service logic in isolation
- Test handler request/response
- Run with: `make test`

### Integration Tests
- Tests against real Redis (K8s redis-cart service)
- Run with: `make test-integration` (auto-manages port-forward)
- For CI: `REDIS_ADDR=... REDIS_PASSWORD=... make test-integration-ci`

### Retrieving Test Credentials
```bash
# Redis password
kubectl get secret -n shopping-cart-data redis-cart-secret -o jsonpath='{.data.password}' | base64 -d
```

## Key Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | 8083 | HTTP port |
| `REDIS_HOST` | localhost | Redis host |
| `REDIS_PORT` | 6379 | Redis port |
| `REDIS_PASSWORD` | - | Redis password |
| `CART_TTL` | 168h | Cart expiration |
| `OAUTH2_ENABLED` | false | Enable OAuth2 |
| `OAUTH2_ISSUER_URI` | - | Keycloak URL |
| `LOG_LEVEL` | info | Log level |

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/cart` | Get cart |
| POST | `/api/v1/cart/items` | Add item |
| PUT | `/api/v1/cart/items/{id}` | Update item |
| DELETE | `/api/v1/cart/items/{id}` | Remove item |
| DELETE | `/api/v1/cart` | Clear cart |
| POST | `/api/v1/cart/checkout` | Checkout |
| GET | `/health` | Health check |
| GET | `/metrics` | Prometheus metrics |

## Integration Points

### Redis
- Key pattern: `cart:{customerId}`
- TTL: 7 days default
- JSON serialization

### RabbitMQ
- Exchange: `events`
- Routing keys: `cart.created`, `cart.updated`, `cart.cleared`, `cart.checkout`

### Keycloak
- JWT validation via JWKS endpoint
- Role extraction from claims

## Code Style

- Follow standard Go conventions
- Use `go fmt` and `goimports`
- Error wrapping with context
- Interfaces for testability
- Context propagation

## Dependencies

- `gin-gonic/gin` - HTTP framework
- `redis/go-redis` - Redis client
- `golang-jwt/jwt` - JWT parsing
- `prometheus/client_golang` - Metrics
- `uber-go/zap` - Logging
- `stretchr/testify` - Testing

## Related Repositories

- `shopping-cart-order` - Order Service (Java) - manages orders after checkout
- `shopping-cart-product-catalog` - Product Catalog (Python)
- `rabbitmq-client-go` - RabbitMQ client library
- `shopping-cart-infra` - Infrastructure & Identity
