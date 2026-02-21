# Tech Context: Basket Service

## Language & Runtime

- **Go 1.21+** — module path: `github.com/user/shopping-cart-basket`
- Binary compiled with `-ldflags "-s -w"` to strip debug info

## Core Dependencies (go.mod)

| Dependency | Version | Purpose |
|---|---|---|
| github.com/gin-gonic/gin | v1.9.1 | HTTP framework and router |
| github.com/redis/go-redis/v9 | v9.3.0 | Redis client |
| github.com/golang-jwt/jwt/v5 | v5.2.0 | JWT parsing and validation |
| github.com/google/uuid | v1.5.0 | UUID generation for IDs |
| github.com/prometheus/client_golang | v1.17.0 | Prometheus metrics |
| go.uber.org/zap | v1.26.0 | Structured JSON logging |
| github.com/stretchr/testify | v1.8.4 | Test assertions and mocking |

## Infrastructure Dependencies

| Service | Required | Default Address | Purpose |
|---|---|---|---|
| Redis | Yes | localhost:6379 | Cart state storage |
| Keycloak | Optional | — | JWT issuer / JWKS endpoint |
| RabbitMQ | Optional | localhost:5672 | Event publishing |
| HashiCorp Vault | Optional | localhost:8200 | Dynamic RabbitMQ credentials |

## Development Environment Setup

### Prerequisites

- Go 1.21 or later
- Access to Redis (local, Docker, or K8s port-forward)
- Optional: RabbitMQ, Keycloak for full integration

### Local Setup

```bash
# Clone and install dependencies
go mod download

# Option 1: Port-forward K8s Redis
kubectl port-forward -n shopping-cart-data svc/redis-cart 6379:6379 &
# Get password:
kubectl get secret -n shopping-cart-data redis-cart-secret -o jsonpath='{.data.password}' | base64 -d

# Option 2: Local Redis via Docker
docker run -d -p 6379:6379 redis:7-alpine

# Run the service
REDIS_HOST=localhost REDIS_PASSWORD=<pw> make run
# Or without auth:
make run
```

### Environment Variables (full list with defaults)

```bash
SERVER_PORT=8083
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
CART_TTL=168h
OAUTH2_ENABLED=false
OAUTH2_ISSUER_URI=
OAUTH2_CLIENT_ID=cart-service
RABBITMQ_HOST=localhost
RABBITMQ_PORT=5672
RABBITMQ_VHOST=/
RABBITMQ_USERNAME=guest
RABBITMQ_PASSWORD=guest
RABBITMQ_USE_TLS=false
VAULT_ENABLED=false
VAULT_ADDR=http://localhost:8200
VAULT_TOKEN=
VAULT_ROLE=cart-service
LOG_LEVEL=info
RATE_LIMIT_RPS=100
RATE_LIMIT_BURST=50
```

## Build Tooling

- **Makefile** — primary build interface; see `make help` for all targets
- **Dockerfile** — multi-stage build: `golang:1.21-alpine` builder → `alpine:3.18` runner
- **Dockerfile.local** — variant for local development
- **go.sum** — checked into version control for reproducible builds

## Testing Infrastructure

- Unit tests: in-process, no external dependencies, mock Redis interface
- Integration tests: connect to real Redis; build tag `integration`; auto-manage K8s port-forward via Makefile
- CI: `make test-integration-ci` with `REDIS_ADDR` and `REDIS_PASSWORD` env vars for external Redis

## Kubernetes Deployment

- Manifests in `k8s/base/` managed with Kustomize
- Resources: `deployment.yaml`, `service.yaml`, `configmap.yaml`, `serviceaccount.yaml`, `kustomization.yaml`
- Namespace: `shopping-cart-apps`
- Redis secret injected from `shopping-cart-data/redis-cart-secret`
- Service type: ClusterIP, port 80 → container 8083

## CI/CD

- GitHub Actions workflow: `.github/workflows/go-ci.yml`
- Pipeline: lint → test → build → docker build
