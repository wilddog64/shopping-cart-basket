# Active Context: Basket Service

## Current Status

**In Development** — core service is functional with unit and integration tests in place. The service implements the full cart lifecycle including checkout event publishing.

## Implemented Features

- Full cart CRUD: Get/Add/Update/Remove items, Clear cart, Checkout
- Redis-backed persistence with sliding TTL (7 days default)
- Event publishing to RabbitMQ for all cart operations
- JWT authentication via Keycloak JWKS (optional; X-User-ID fallback for dev)
- Prometheus metrics endpoint
- Structured zap logging with correlation IDs
- Rate limiting (per-IP)
- Security headers middleware
- Health check endpoints (live/ready/general)
- Unit tests for service and model layers
- Integration tests for repository (requires Redis)
- Integration tests for service layer (requires Redis)
- Kubernetes manifests (k8s/base/): Deployment, Service, ConfigMap, ServiceAccount, Kustomization
- Dockerfile (multi-stage, non-root, read-only filesystem)
- GitHub Actions CI workflow

## Active Areas of Work

The CLAUDE.md marks the project as "In Development" — the codebase is in an active build phase. Based on the structure, the following areas may be receiving active attention:

- Integration testing pipeline (CI Redis connectivity)
- Kubernetes deployment validation against the cluster

## Key Files Being Worked On

- `internal/service/cart_service.go` — core business logic
- `internal/repository/cart_repository.go` — Redis data access
- `internal/handler/cart_handler.go` — HTTP endpoints
- `internal/handler/middleware.go` — auth, logging, rate limit middleware
- `k8s/base/` — Kubernetes manifests

## Known Integration Points

- **Redis**: K8s service `redis-cart.shopping-cart-data.svc.cluster.local:6379`; secret `redis-cart-secret` in `shopping-cart-data` namespace
- **RabbitMQ**: Exchange `events`; routing keys `cart.*`
- **Keycloak**: JWKS endpoint at `OAUTH2_ISSUER_URI` + `/protocol/openid-connect/certs`
- **Order Service**: Consumes `cart.checkout` events from RabbitMQ

## Development Notes

- Run `make test` for unit tests at any time (no infrastructure needed)
- Run `make test-integration` to run integration tests with automatic K8s Redis port-forward
- The `Dockerfile.local` variant exists for local Docker builds with different defaults
- `bin/port-forward.sh` and `bin/run-local.sh` scripts assist with local development setup (referenced in README but not listed in file tree — may need creation)
- `checksum/redis-secret` annotation in deployment.yaml must be updated manually after Redis password rotation

## CI Status — 2026-03-14

- ✅ Publish workflow now succeeds after switching the shared reusable workflow to `aquasecurity/trivy-action@0.30.0`.
- 🔴 New golangci-lint gate (branch `feature/p4-linter`, PR #1) fails due to pre-existing issues:
  - `cmd/server/main.go:32` — `logger.Sync()` return value not checked (errcheck)
  - `cmd/server/main.go` and `internal/model/cart.go` — not gofmt-ed with `-s`

Latest GitHub Actions runs for PR #1 all fail with the same golangci-lint findings above. Decide whether to address these issues or relax the configuration before merging.
