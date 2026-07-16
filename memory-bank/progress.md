# Progress: Basket Service

## What's Built

### Guest Cart (branch: feat/guest-cart)
- [x] Added guest cart config: `GUEST_CART_TTL` and `GUEST_TOKEN_SECRET`
- [x] Added signed guest token mint/verify flow in `internal/auth/guest.go` with unit coverage
- [x] Added `GuestOrAuthMiddleware` so cart routes admit JWT users or signed guest tokens
- [x] Added guest checkout guard and `POST /api/v1/cart/merge` handler
- [x] Added guest-aware rolling TTL logic and guest-to-user merge flow in `CartService`
- [x] Wired guest token manager through `cmd/server/main.go`, including ephemeral-secret fallback when config is unset
- [x] Updated cart service tests for guest TTL, rolling expiry, and merge behavior
- [ ] Local verification pending in this environment: `go version` returned `command not found`, so `go build ./...`, `go vet ./...`, and `go test ./...` were not run here
- [x] Backend implementation pushed to `origin/feat/guest-cart` at commit `2ba9114`

### Core Application
- [x] Domain model: `Cart`, `CartItem`, `ShippingAddress` with all business methods
- [x] Request/response DTOs: `AddItemRequest`, `UpdateItemRequest`, `CheckoutRequest`, `CartResponse`
- [x] `CartService` — full business logic: GetCart, AddItem, UpdateItemQuantity, RemoveItem, ClearCart, Checkout
- [x] `RedisCartRepository` — Get, Save, Delete with JSON serialization and TTL
- [x] `EventPublisher` — RabbitMQ publishing with common event envelope
- [x] `CartHandler` — all HTTP endpoints wired to service
- [x] `HealthHandler` — /health, /health/live, /health/ready
- [x] Auth middleware — JWT (Keycloak JWKS) and X-User-ID dev fallback
- [x] Logging middleware — structured zap request logging with correlation ID
- [x] Rate limiting middleware — per-IP token bucket
- [x] Security headers middleware
- [x] Configuration via environment variables (`config.Load()`)
- [x] `pkg/response` — shared success/error response helpers
- [x] Prometheus metrics endpoint at `/metrics`
- [x] Application entry point (`cmd/server/main.go`) with dependency wiring

### Testing
- [x] `internal/model/cart_test.go` — unit tests for Cart domain model
- [x] `internal/service/cart_service_test.go` — unit tests with mocked repository
- [x] `internal/service/cart_service_integration_test.go` — integration tests against real Redis
- [x] `internal/repository/cart_repository_integration_test.go` — repository integration tests

### Infrastructure & Operations
- [x] Dockerfile (multi-stage, golang:1.21-alpine builder, alpine runner, non-root)
- [x] Dockerfile.local (local development variant)
- [x] Makefile with targets: build, run, test, test-unit, test-integration, test-integration-ci, coverage, lint, fmt, clean, deps, mocks, docker-build, docker-run
- [x] k8s/base/deployment.yaml — 2 replicas, rolling update, resource limits, probes, pod security
- [x] k8s/base/service.yaml — ClusterIP
- [x] k8s/base/configmap.yaml — non-secret configuration
- [x] k8s/base/serviceaccount.yaml
- [x] k8s/base/kustomization.yaml
- [x] .github/workflows/go-ci.yml — CI pipeline
- [x] .gitignore
- [x] go.mod / go.sum
- [x] CI multi-arch workflow pin — `.github/workflows/go-ci.yml` now references `build-push-deploy.yml@999f8d7` so ghcr publishes amd64+arm64 images (2026-03-17).

### Documentation
- [x] CLAUDE.md — AI assistant guidance
- [x] README.md — setup and usage guide
- [x] docs/README.md
- [x] docs/api/README.md — API reference
- [x] docs/architecture/README.md — architecture diagrams and component details
- [x] docs/troubleshooting/README.md
- [x] docs/testing/README.md — Go unit + integration test workflow, coverage, linting (added 2026-03-17)
- [x] docs/issues/2026-03-17-readme-standardization.md — documents README/docs drift + resolution

## What's Pending / Known Gaps

- [ ] `bin/port-forward.sh` and `bin/run-local.sh` helper scripts (referenced in README but not present in file listing)
- [ ] No HPA (HorizontalPodAutoscaler) manifest — order and product-catalog services have one, basket does not
- [ ] No namespace manifest — relies on namespace being pre-created by infra team
- [ ] No secret manifest — Redis secret managed externally
- [ ] Load/performance testing
- [ ] E2E test integration (referenced for other services)
- [ ] Vault integration fully tested (config present but untested path)

## Known Issues

- `checksum/redis-secret` annotation in `k8s/base/deployment.yaml` is set to `"change-me-on-rotation"` — must be updated manually after each Redis password rotation to trigger pod restarts
- No RabbitMQ reconnection strategy documented (depends on rabbitmq-client-go library behavior)

## API Endpoints Summary

| Method | Path | Auth | Status |
|--------|------|------|--------|
| GET | `/api/v1/cart` | Required | Implemented |
| POST | `/api/v1/cart/items` | Required | Implemented |
| PUT | `/api/v1/cart/items/{itemId}` | Required | Implemented |
| DELETE | `/api/v1/cart/items/{itemId}` | Required | Implemented |
| DELETE | `/api/v1/cart` | Required | Implemented |
| POST | `/api/v1/cart/checkout` | Required | Implemented |
| GET | `/health` | Public | Implemented |
| GET | `/health/live` | Public | Implemented |
| GET | `/health/ready` | Public | Implemented |
| GET | `/metrics` | Public | Implemented |
