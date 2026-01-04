# Cart Service Documentation

Welcome to the Cart Service documentation. This service manages shopping cart sessions for the Shopping Cart platform.

## Quick Links

| Document | Description |
|----------|-------------|
| [Architecture](architecture/README.md) | System design, components, and data flow |
| [API Reference](api/README.md) | REST API endpoints and examples |
| [Troubleshooting](troubleshooting/README.md) | Common issues and debugging guide |

## Overview

The Cart Service is a Go microservice responsible for:
- Managing shopping cart sessions
- Adding, updating, and removing items
- Cart total calculation
- Checkout orchestration
- Publishing cart events to RabbitMQ
- Integration with Keycloak for OAuth2/OIDC authentication

## Getting Started

### Prerequisites

- Go 1.21+
- Redis 7+
- RabbitMQ 3.12+ (optional)
- Keycloak (optional, for OAuth2)

### Running Locally

```bash
# Start Redis
docker run -d -p 6379:6379 redis:7-alpine

# Run the service
./bin/run-local.sh

# Or with make
make run
```

### Running Tests

```bash
# All tests
make test

# Unit tests only
make test-unit

# With coverage
make coverage
```

## Configuration

Key environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_PORT` | HTTP server port | `8083` |
| `REDIS_HOST` | Redis hostname | `localhost` |
| `REDIS_PORT` | Redis port | `6379` |
| `REDIS_PASSWORD` | Redis password | - |
| `CART_TTL` | Cart expiration | `168h` |
| `OAUTH2_ENABLED` | Enable OAuth2 | `false` |
| `OAUTH2_ISSUER_URI` | Keycloak issuer | - |
| `LOG_LEVEL` | Logging level | `info` |

See [Architecture > Configuration](architecture/README.md#configuration) for full list.

## API Quick Reference

### Cart Operations

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/cart` | Get cart |
| POST | `/api/v1/cart/items` | Add item |
| PUT | `/api/v1/cart/items/{id}` | Update quantity |
| DELETE | `/api/v1/cart/items/{id}` | Remove item |
| DELETE | `/api/v1/cart` | Clear cart |
| POST | `/api/v1/cart/checkout` | Checkout |

### Health & Metrics

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/health/live` | Liveness probe |
| GET | `/health/ready` | Readiness probe |
| GET | `/metrics` | Prometheus metrics |

See [API Reference](api/README.md) for complete documentation.

## Project Structure

```
shopping-cart-cart/
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
├── bin/                 # Helper scripts
└── docs/                # Documentation
```

## Related Services

- **Order Service**: Receives checkout events
- **Product Catalog**: Product information
- **Keycloak**: Identity and access management
- **Redis**: Cart state storage
- **RabbitMQ**: Event messaging

## Support

- [Troubleshooting Guide](troubleshooting/README.md)
- [GitHub Issues](https://github.com/your-org/shopping-cart-cart/issues)
