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
- ✅ golangci-lint gate on `feature/p4-linter` (PR [#1](https://github.com/wilddog64/shopping-cart-basket/pull/1)) is green per run `23094080858`, which validated commit `3508a9e9161e3704620e01258a57cb0af860fa65` via `gh api`. The lint fixes were applied in commit `7b9dd065384dca3d3498859d82ea2a8ff7ba9d52` by wrapping `logger.Sync()` in `cmd/server/main.go` with a deferred function that ignores the error and running `gofmt -s` on `main.go` and `internal/model/cart.go`.

## P4 Linter Task — Assigned to Codex (2026-03-14)

**Branch:** `feature/p4-linter`
**Spec:** `wilddog64/shopping-cart-infra/docs/plans/p4-linter-basket.md`
**PR:** https://github.com/wilddog64/shopping-cart-basket/pull/1
**Verified CI run:** `23094080858` — conclusion `success`
**Verified commit:** `3508a9e9161e3704620e01258a57cb0af860fa65` (via `gh api`)

golangci-lint now passes after addressing the errcheck and gofmt findings noted above.
