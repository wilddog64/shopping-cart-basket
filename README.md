# Basket Service

A Go 1.21 microservice that manages shopping-cart sessions. It exposes REST APIs for adding/updating/removing items, stores carts in Redis, publishes checkout events to RabbitMQ, and enforces OAuth2/OIDC policies via Keycloak.

---

## Quick Start

### Prerequisites
- Go 1.21+
- Redis 7+ (local docker: `docker run -d -p 6379:6379 redis:7-alpine`)
- RabbitMQ 3.12+ (optional for event publishing)
- Keycloak (optional for OAuth2 enforcement)

### Install & Run
```bash
# Install dependencies
go mod download

# Start Redis port-forward from Kubernetes (optional helper)
./bin/port-forward.sh

# Run the service locally (auto-starts Redis forward with --start-redis)
./bin/run-local.sh --start-redis

# Build binary / Docker image
make build
make docker-build
```

### Tests
```bash
# Unit tests
make test-unit

# Integration tests (requires Redis)
make test-integration

# CI variant with external Redis
REDIS_ADDR=redis.example.com:6379 \
REDIS_PASSWORD=... make test-integration-ci

# Coverage report
make coverage
```

---

## Usage

### Features
- Shopping cart CRUD operations (add/update/remove/clear)
- Checkout orchestration that publishes RabbitMQ events
- Redis-backed state with TTL-based expiration
- OAuth2/OIDC enforcement with Keycloak JWT validation + dev fallback header
- Prometheus `/metrics` endpoint and health probes

### Technology Stack
| Category | Technology |
|----------|------------|
| Language | Go 1.21 |
| Framework | Gin |
| Storage | Redis |
| Messaging | RabbitMQ |
| Auth | Keycloak OAuth2/OIDC |
| CI | GitHub Actions + Trivy + golangci-lint |

### Environment Variables
| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | 8083 | HTTP server port |
| `REDIS_HOST` | localhost | Redis hostname |
| `REDIS_PORT` | 6379 | Redis port |
| `REDIS_PASSWORD` | – | Redis password |
| `REDIS_DB` | 0 | Redis database |
| `CART_TTL` | 168h | Cart expiration window |
| `OAUTH2_ENABLED` | false | Toggle OAuth2 enforcement |
| `OAUTH2_ISSUER_URI` | – | Keycloak issuer URL |
| `RABBITMQ_HOST` | localhost | RabbitMQ host |
| `RABBITMQ_PORT` | 5672 | RabbitMQ port |
| `LOG_LEVEL` | info | zap log level |

### API Endpoints
| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/api/v1/cart` | Get current user's cart | Required |
| POST | `/api/v1/cart/items` | Add item to cart | Required |
| PUT | `/api/v1/cart/items/{itemId}` | Update item quantity | Required |
| DELETE | `/api/v1/cart/items/{itemId}` | Remove item from cart | Required |
| DELETE | `/api/v1/cart` | Clear cart | Required |
| POST | `/api/v1/cart/checkout` | Convert cart to order | Required |
| GET | `/health`, `/health/live`, `/health/ready` | Health probes | Public |
| GET | `/metrics` | Prometheus metrics | Public |

### API Examples
```bash
# Add an item
curl -X POST http://localhost:8083/api/v1/cart/items \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "productId": "prod-123",
    "name": "Widget",
    "quantity": 2,
    "unitPrice": 29.99
  }'

# Get cart
curl http://localhost:8083/api/v1/cart -H "Authorization: Bearer $TOKEN"

# Update quantity
curl -X PUT http://localhost:8083/api/v1/cart/items/item-123 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"quantity": 5}'

# Remove item
curl -X DELETE http://localhost:8083/api/v1/cart/items/item-123 \
  -H "Authorization: Bearer $TOKEN"

# Clear cart
curl -X DELETE http://localhost:8083/api/v1/cart -H "Authorization: Bearer $TOKEN"

# Checkout
curl -X POST http://localhost:8083/api/v1/cart/checkout \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"shippingAddress": {"street": "123 Main St"}}'
```

### Integration Tests in CI (External Redis)
For CI environments that connect to the shared Redis instance, export the credentials and use the `test-integration-ci` target:
```bash
# Grab password from Kubernetes secret
kubectl get secret -n shopping-cart-data redis-cart-secret \
  -o jsonpath='{.data.password}' | base64 -d

# Run integration tests pointing at external Redis
REDIS_ADDR=redis.example.com:6379 \
REDIS_PASSWORD=<password> \
make test-integration-ci
```

| Secret | Namespace | Key |
|--------|-----------|-----|
| `redis-cart-secret` | `shopping-cart-data` | `password` |

### Authentication
1. Users authenticate via Keycloak; JWTs are validated via JWKS.
2. Customer ID comes from the `sub` claim; optional `X-User-ID` header is available when `OAUTH2_ENABLED=false`.
3. Protected routes cover every `/api/v1/cart*` endpoint.

### Docker & Kubernetes
```bash
# Build + run container locally
make docker-build
make docker-run

# Kubernetes deployment preview
kubectl kustomize k8s/base
kubectl apply -k k8s/base
```

---

## Architecture
See **[Service Architecture](docs/architecture/README.md)** for component diagrams, event flow, configuration matrix, and security considerations.

---

## Directory Layout
```
shopping-cart-basket/
├── cmd/server/          # Application entry point
├── internal/
│   ├── config/          # Configuration loading
│   ├── handler/         # HTTP handlers
│   ├── service/         # Business logic
│   ├── repository/      # Redis persistence
│   ├── model/           # Domain models
│   ├── auth/            # JWT middleware
│   └── event/           # RabbitMQ publisher
├── pkg/response/        # Shared response helpers
├── k8s/                 # Kubernetes manifests
├── docs/                # Architecture/API/testing/troubleshooting
└── Makefile, Dockerfile, etc.
```

---

## Documentation

### Architecture
- **[Service Architecture](docs/architecture/README.md)** — system design, flows, configuration, security.

### API Reference
- **[API Reference](docs/api/README.md)** — endpoint details, payloads, and error codes.

### Testing
- **[Testing Guide](docs/testing/README.md)** — Go unit/integration tests, coverage, linting.

### Troubleshooting
- **[Troubleshooting Guide](docs/troubleshooting/README.md)** — Redis/auth/data/perf issue playbooks.

### Issue Logs
- **[README/docs structure drift](docs/issues/2026-03-17-readme-standardization.md)** — documentation standardization record.

---

## Releases

| Version | Date | Highlights |
|---------|------|------------|
| v0.1.0 | TBD | Initial release — Go/Gin service, Redis-backed carts, RabbitMQ events |

---

## Related
- [shopping-cart-infra](https://github.com/wilddog64/shopping-cart-infra)
- [shopping-cart-order](https://github.com/wilddog64/shopping-cart-order)
- [shopping-cart-payment](https://github.com/wilddog64/shopping-cart-payment)
- [shopping-cart-product-catalog](https://github.com/wilddog64/shopping-cart-product-catalog)
- [shopping-cart-frontend](https://github.com/wilddog64/shopping-cart-frontend)

---

## License
Apache 2.0
